package baseline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/plugin"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type ShardReport struct {
	Format string        `json:"format"`
	PlanID string        `json:"planId"`
	Shard  int           `json:"shard"`
	State  string        `json:"state"`
	Rows   []Observation `json:"rows"`
}
type Executor func(context.Context, string, string, *agenteval.Witness) (*plugin.Report, error)

// RunShard claims exactly one shard and never continues or retries another.
func RunShard(parent context.Context, root, name string, shard int, execute Executor) (*ShardReport, error) {
	return runShard(parent, root, name, shard, execute, nil)
}

type livePermission struct {
	allow      bool
	credential func() string
}

// RunLiveShard checks every selected approval before output claim or credential
// lookup. It uses only the existing fixed-endpoint, no-retry API bridge.
func RunLiveShard(parent context.Context, root, name string, shard int, allow bool, credential func() string) (*ShardReport, error) {
	return runShard(parent, root, name, shard, nil, &livePermission{allow, credential})
}
func runShard(parent context.Context, root, name string, shard int, execute Executor, permission *livePermission) (*ShardReport, error) {
	p, err := Load(parent, root, name)
	if err != nil {
		return nil, err
	}
	if shard < 1 || shard > 20 || (p.Frozen.Plan.Mode == "offline-contract" && (execute == nil || permission != nil)) {
		return nil, errors.New("BASELINE_SHARD_INVALID")
	}
	selected := []Trial{}
	for _, t := range p.Frozen.Trials {
		if t.Shard == shard {
			selected = append(selected, t)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("BASELINE_SHARD_INVALID")
	}
	// Resolve every witness and session definition before claiming any output.
	witnesses := map[string]*agenteval.Witness{}
	plans := map[string]plugin.Plan{}
	livePlans := map[string]*agentapi.Plan{}
	for _, t := range selected {
		e := p.Pack.Find(t.EntryID)
		if p.Frozen.Plan.Mode == "api-bridge-live" {
			if permission == nil || !permission.allow || permission.credential == nil {
				return nil, errors.New("LIVE_NOT_AUTHORIZED")
			}
			live, err := agentapi.DecodePlan(p.Pack.Reader.Inputs[p.Frozen.Plan.LivePlans[t.ID]].Raw)
			if err != nil {
				return nil, err
			}
			if _, err = CheckLive(live, e, time.Now(), permission.allow, 0); err != nil {
				return nil, err
			}
			livePlans[t.ID] = live
			continue
		}
		sp := SessionPlan(p.Frozen, t, e.Task.Budgets.EpisodeSeconds)
		if sp.Host.Name != "mcp-client" {
			return nil, errors.New("BASELINE_ADAPTER_UNSUPPORTED")
		}
		plans[t.ID] = sp
		raw := p.Pack.Reader.Inputs[path.Join(e.Entry.Root, e.Entry.Cases)].Raw
		var cases agenteval.CaseManifest
		if baselinepack.Decode(raw, task.MaxBytes, &cases) != nil {
			return nil, errors.New("BASELINE_QUALITY_INVALID")
		}
		for _, c := range cases.Cases {
			if c.TaskID == e.Task.ID && c.Role == "positive" {
				raw = p.Pack.Reader.Inputs[path.Join(e.Entry.Root, c.Witness)].Raw
				w, er := agenteval.DecodeWitness(raw)
				if er != nil {
					return nil, er
				}
				witnesses[t.ID] = w
				break
			}
		}
		if witnesses[t.ID] == nil {
			return nil, errors.New("BASELINE_QUALITY_INVALID")
		}
	}
	inputRoot := filepath.Join(root, "inputs")
	fs, err := os.OpenRoot(inputRoot)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	// The entire frozen experiment has one execution lease. A crash leaves the
	// lease visible; it is never treated as permission to resume the old plan.
	lock, e := fs.OpenFile("shard.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, errors.New("BASELINE_SHARD_BUSY_OR_INTERRUPTED")
	}
	if e = lock.Close(); e != nil {
		return nil, e
	}
	defer fs.Remove("shard.lock")
	if err = fs.Mkdir("runs", 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	info, err := fs.Lstat("runs")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("BASELINE_OUTPUT_INVALID")
	}
	dir := fmt.Sprintf("runs/shard-%02d", shard)
	if fs.Mkdir(dir, 0700) != nil {
		return nil, errors.New("BASELINE_SHARD_ALREADY_CLAIMED")
	}
	report := &ShardReport{"statetwin.dev/baseline-shard/v1alpha1", p.Frozen.Plan.ID, shard, "running", []Observation{}}
	for _, t := range selected {
		report.Rows = append(report.Rows, Observation{TrialID: t.ID, State: "missing", ObservedModel: "unknown", Usage: Usage{Status: "unavailable"}})
	}
	if err = writeJSON(fs, path.Join(dir, "claim.json"), report); err != nil {
		return report, err
	}
	for _, t := range selected {
		var definition any = plans[t.ID]
		if livePlans[t.ID] != nil {
			definition = livePlans[t.ID]
		}
		if err = writeJSON(fs, path.Join(dir, t.ID+".plan.json"), definition); err != nil {
			return report, err
		}
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(p.Frozen.Plan.ShardSeconds)*time.Second)
	defer cancel()
	var failure error
	key := ""
	if permission != nil {
		for _, directory := range []string{".statetwin", ".statetwin/live"} {
			if e := fs.Mkdir(directory, 0700); e != nil && !os.IsExist(e) {
				return report, e
			}
			info, e := fs.Lstat(directory)
			if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return report, errors.New("BASELINE_OUTPUT_INVALID")
			}
		}
		key = permission.credential()
	}
	for i, t := range selected {
		if ctx.Err() != nil {
			failure = ctx.Err()
			break
		}
		start := time.Now()
		if livePlans[t.ID] != nil {
			_, err = agenteval.RecordLive(ctx, inputRoot, livePlans[t.ID], p.Pack.Find(t.EntryID).Bundle, key, permission.allow)
		} else {
			_, err = execute(ctx, inputRoot, path.Join(dir, t.ID+".plan.json"), witnesses[t.ID])
		}
		elapsed := time.Since(start).Seconds()
		if err != nil {
			failure = errors.New("BASELINE_TRIAL_FAILED")
			break
		}
		o, e := inspectTrial(ctx, inputRoot, p, t)
		if e != nil {
			failure = e
			break
		}
		o.LatencySeconds = &elapsed
		report.Rows[i] = o
	}
	report.State = "complete"
	if failure != nil {
		report.State = "interrupted"
	}
	if err = writeJSON(fs, path.Join(dir, "shard-report.json"), report); err != nil {
		return report, err
	}
	return report, failure
}

func inspectTrial(ctx context.Context, root string, p *Prepared, t Trial) (Observation, error) {
	if p.Frozen.Plan.Mode == "api-bridge-live" {
		return inspectLiveTrial(ctx, root, p, t)
	}
	o := Observation{TrialID: t.ID, State: "missing", ObservedModel: "unknown", Usage: Usage{Status: "unavailable"}}
	e := p.Pack.Find(t.EntryID)
	expected := SessionPlan(p.Frozen, t, e.Task.Budgets.EpisodeSeconds)
	name := path.Join("runs", fmt.Sprintf("shard-%02d", t.Shard), t.ID+".plan.json")
	raw, err := task.ReadFile(root, name, 64<<10)
	if err != nil {
		return o, err
	}
	var plan plugin.Plan
	if baselinepack.Decode(raw, 64<<10, &plan) != nil || !baselinepack.Equal(plan, expected) {
		return o, errors.New("BASELINE_EVIDENCE_INVALID")
	}
	r, err := plugin.Inspect(ctx, root, name)
	if r != nil {
		o.State = r.Execution
		o.Cleanup = r.Cleanup
	}
	if err != nil {
		return o, err
	}
	raw, err = task.ReadFile(root, path.Join(expected.Output, "terminal.json"), limits.MaxReportBytes)
	if err != nil {
		return o, err
	}
	terminal, err := agenteval.DecodePluginTerminal(raw)
	if err != nil || terminal.Evaluation == nil {
		return o, errors.New("BASELINE_EVIDENCE_INVALID")
	}
	if agenteval.VerifyPluginTerminal(ctx, terminal, e.Task, e.Bundle) != nil {
		return o, errors.New("BASELINE_EVIDENCE_INVALID")
	}
	// Inspect has independently replayed this exact terminal from frozen inputs.
	o.Verified = true
	o.Success = r.Decision == "passed"
	o.IllegalAttempts = terminal.Evaluation.PolicyAttempts
	o.IllegalEffects = terminal.Evaluation.CommittedViolations
	for _, c := range terminal.Evaluation.Checks {
		if c.Category == "policy" && !c.Passed {
			o.PolicyFailures++
		}
	}
	return o, nil
}

func Assess(ctx context.Context, root, name string) (*Report, error) {
	p, err := Load(ctx, root, name)
	if err != nil {
		return nil, err
	}
	observations := []Observation{}
	shards := map[string]string{}
	recorded := map[string]Observation{}
	valid := true
	for shard := 1; shard <= (len(p.Frozen.Trials)+p.Frozen.Plan.TrialsPerShard-1)/p.Frozen.Plan.TrialsPerShard; shard++ {
		key := fmt.Sprintf("shard-%02d", shard)
		shards[key] = "missing"
		raw, e := task.ReadFile(filepath.Join(root, "inputs"), path.Join("runs", key, "shard-report.json"), 1<<20)
		if e != nil {
			valid = false
			continue
		}
		var sr ShardReport
		if baselinepack.Decode(raw, 1<<20, &sr) != nil || sr.Format != "statetwin.dev/baseline-shard/v1alpha1" || sr.PlanID != p.Frozen.Plan.ID || sr.Shard != shard || (sr.State != "complete" && sr.State != "interrupted") {
			shards[key] = "invalid"
			valid = false
			continue
		}
		expected := []Trial{}
		for _, trial := range p.Frozen.Trials {
			if trial.Shard == shard {
				expected = append(expected, trial)
			}
		}
		if len(sr.Rows) != len(expected) {
			shards[key] = "invalid"
			valid = false
			continue
		}
		shards[key] = sr.State
		if sr.State != "complete" {
			valid = false
		}
		for i, row := range sr.Rows {
			if row.TrialID != expected[i].ID {
				valid = false
				shards[key] = "invalid"
				continue
			}
			recorded[row.TrialID] = row
		}
	}
	for _, t := range p.Frozen.Trials {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		o, _ := inspectTrial(ctx, filepath.Join(root, "inputs"), p, t)
		if logged, ok := recorded[t.ID]; ok {
			latency := logged.LatencySeconds
			logged.LatencySeconds = nil
			if baselinepack.Equal(logged, o) && latency != nil && *latency >= 0 {
				o.LatencySeconds = latency
			} else if !baselinepack.Equal(logged, o) {
				valid = false
				shards[fmt.Sprintf("shard-%02d", t.Shard)] = "invalid"
			}
		}
		if o.State != "missing" {
			observations = append(observations, o)
		}
	}
	r, err := Decide(p.Frozen, observations)
	if r != nil {
		r.Shards = shards
		if !valid {
			r.Decision = "invalid"
			r.Reason = "missing-or-invalid-shard-evidence"
		}
	}
	return r, err
}
