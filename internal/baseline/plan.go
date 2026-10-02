// Package baseline provides bounded, frozen, paired experiments. Provider
// access is confined to explicitly authorized API-bridge shards, never prepare,
// freeze, assessment or offline execution.
package baseline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/plugin"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const PlanFormat = "statetwin.dev/baseline-plan/v1alpha1"
const FrozenFormat = "statetwin.dev/baseline-frozen/v1alpha1"
const ReportFormat = "statetwin.dev/baseline-report/v1alpha1"
const Algorithm = "stratified-family-bootstrap-v1"

type Config struct {
	ID             string      `json:"id"`
	Provider       string      `json:"provider"`
	RequestedModel string      `json:"requestedModel"`
	Host           plugin.Host `json:"host"`
	Adapter        string      `json:"adapter"`
	PromptVersion  string      `json:"promptVersion"`
}
type Policy struct {
	Delta            *float64           `json:"delta"`
	Floors           map[string]float64 `json:"floors"`
	Required         []string           `json:"required"`
	Algorithm        string             `json:"algorithm"`
	Seed             uint64             `json:"seed"`
	BootstrapSamples int                `json:"bootstrapSamples"`
}
type Plan struct {
	Format         string            `json:"format"`
	ID             string            `json:"id"`
	Revision       string            `json:"revision"`
	Purpose        string            `json:"purpose"`
	Mode           string            `json:"mode"`
	Pack           string            `json:"pack"`
	Profile        string            `json:"profile"`
	Entries        []string          `json:"entries"`
	Configs        []Config          `json:"configs"`
	Repeats        int               `json:"repeats"`
	ShardSeconds   int               `json:"shardSeconds"`
	TrialsPerShard int               `json:"trialsPerShard"`
	Policy         Policy            `json:"policy"`
	LivePlans      map[string]string `json:"livePlans,omitempty"`
}
type Trial struct {
	ID       string `json:"id"`
	EntryID  string `json:"entryId"`
	FamilyID string `json:"familyId"`
	Domain   string `json:"domain"`
	ConfigID string `json:"configId"`
	Repeat   int    `json:"repeat"`
	Shard    int    `json:"shard"`
	Order    int    `json:"order"`
	Required bool   `json:"required"`
}
type Frozen struct {
	Format          string               `json:"format"`
	Plan            Plan                 `json:"plan"`
	RuntimeVersion  string               `json:"runtimeVersion"`
	RuntimeRevision string               `json:"runtimeRevision"`
	PackID          string               `json:"packId"`
	PackRevision    string               `json:"packRevision"`
	Profile         baselinepack.Profile `json:"profile"`
	Trials          []Trial              `json:"trials"`
	Comparison      string               `json:"comparison"`
	SourceTrust     string               `json:"sourceTrust"`
}
type Prepared struct {
	Frozen Frozen
	Pack   *baselinepack.Prepared
}

func Prepare(ctx context.Context, root, name string) (*Prepared, error) {
	r := baselinepack.NewReader(ctx, root)
	raw, err := r.Read(name, 64<<10)
	if err != nil {
		return nil, err
	}
	var p Plan
	if baselinepack.Decode(raw, 64<<10, &p) != nil {
		return nil, errors.New("BASELINE_PLAN_INVALID")
	}
	if p.Format != PlanFormat || !baselinepack.Label(p.ID) || !baselinepack.Label(p.Revision) || (p.Purpose != "pilot" && p.Purpose != "release-comparison") || (p.Mode != "offline-contract" && p.Mode != "api-bridge-live") || len(p.Configs) != 2 || len(p.Entries) < 1 || len(p.Entries) > 72 || p.Repeats < 1 || p.Repeats > 5 || len(p.Entries)*p.Repeats*2 > 240 || p.TrialsPerShard < 1 || p.TrialsPerShard > 12 || p.ShardSeconds < 1 || p.ShardSeconds > 2700 || p.Policy.Delta == nil || *p.Policy.Delta < 0 || *p.Policy.Delta > 1 || p.Policy.Algorithm != Algorithm || p.Policy.BootstrapSamples != 10000 || len(p.Policy.Floors) != 2 {
		return nil, errors.New("BASELINE_PLAN_INVALID")
	}
	total := len(p.Entries) * p.Repeats * 2
	if (total+p.TrialsPerShard-1)/p.TrialsPerShard > 20 {
		return nil, errors.New("BASELINE_RESOURCE_LIMIT")
	}
	for i, c := range p.Configs {
		if !baselinepack.Label(c.ID) || !baselinepack.Label(c.PromptVersion) || (i == 1 && c.ID == p.Configs[0].ID) {
			return nil, errors.New("BASELINE_CONFIG_INVALID")
		}
		if p.Mode == "api-bridge-live" {
			if c.Provider != "openai" || !agentapi.ValidModel(c.RequestedModel) || c.Adapter != agentapi.Profile || c.Host.Name != "api-bridge" || c.Host.Version != "v1alpha1" || c.Host.Model != c.RequestedModel || !c.Host.FreshSession || !c.Host.ToolsOnly || c.Host.AdditionalTools != 0 || c.PromptVersion != "blind-objective-v1" {
				return nil, errors.New("BASELINE_CONFIG_INVALID")
			}
			continue
		}
		sample := plugin.Plan{Format: plugin.PlanFormat, ID: "admission", Pack: p.Pack, Profile: p.Profile, EntryID: p.Entries[0], Host: c.Host, Output: "admission-output", Mode: p.Mode, DeadlineSeconds: 1}
		if !baselinepack.Label(c.ID) || c.Provider != "mock" || c.RequestedModel != "mock/statetwin" || sample.Validate() != nil || !baselinepack.Label(c.PromptVersion) || (c.Adapter != "go-sdk-1.8.0" && c.Adapter != "inspect-0.3.275") || (i == 1 && c.ID == p.Configs[0].ID) {
			return nil, errors.New("BASELINE_CONFIG_INVALID")
		}
		if (c.Host.Name == "inspect") != (c.Adapter == "inspect-0.3.275") {
			return nil, errors.New("BASELINE_CONFIG_INVALID")
		}
	}
	pack, err := baselinepack.PrepareWith(r, p.Pack, p.Profile)
	if err != nil {
		return nil, err
	}
	f := Frozen{FrozenFormat, p, server.Version, server.Revision, pack.Pack.ID, pack.Pack.Revision, pack.Profile, []Trial{}, "whole-configuration", "local-operator-asserted"}
	seen := map[string]bool{}
	required := map[string]bool{}
	families := map[string]string{}
	for _, id := range p.Policy.Required {
		if required[id] {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
		required[id] = true
	}
	for _, id := range p.Entries {
		e := pack.Find(id)
		if e == nil || seen[id] {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
		seen[id] = true
		if p.Purpose == "pilot" && e.Entry.Split != "dev" || p.Purpose == "release-comparison" && (e.Entry.Split != "evaluation" || e.Entry.Disclosure != "public-evaluation-split") {
			return nil, errors.New("BASELINE_DISCLOSURE_INVALID")
		}
		if old := families[e.Entry.Family]; old != "" && old != e.Task.Domain {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
		families[e.Entry.Family] = e.Task.Domain
		floor, ok := p.Policy.Floors[e.Task.Domain]
		if !ok || floor < 0 || floor > 1 {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
		for repeat := 0; repeat < p.Repeats; repeat++ {
			order := []int{0, 1}
			if (len(f.Trials)/2)%2 == 1 {
				order = []int{1, 0}
			}
			for _, c := range order {
				n := len(f.Trials)
				f.Trials = append(f.Trials, Trial{fmt.Sprintf("trial-%03d", n+1), id, e.Entry.Family, e.Task.Domain, p.Configs[c].ID, repeat + 1, n/p.TrialsPerShard + 1, n + 1, required[id]})
			}
		}
	}
	for id := range required {
		if !seen[id] {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
	}
	if p.Mode == "offline-contract" && len(p.LivePlans) != 0 {
		return nil, errors.New("BASELINE_CONFIG_INVALID")
	}
	if p.Mode == "api-bridge-live" {
		if len(p.LivePlans) != len(f.Trials) {
			return nil, errors.New("BASELINE_LIVE_BINDING_INVALID")
		}
		ids := map[string]bool{}
		for _, t := range f.Trials {
			name := p.LivePlans[t.ID]
			raw, err := r.Read(name, task.MaxBytes+(16<<10))
			if err != nil {
				return nil, err
			}
			live, err := agentapi.DecodePlan(raw)
			if err != nil {
				return nil, err
			}
			e := pack.Find(t.EntryID)
			var config Config
			for _, c := range p.Configs {
				if c.ID == t.ConfigID {
					config = c
				}
			}
			b, err := bundle.OpenBytes(e.Bundle)
			if err != nil || ids[live.ID] || live.Model != config.RequestedModel || !baselinepack.Equal(live.Task, e.Task) || agenteval.PreflightLive(live, b) != nil || r.RejectOutput(live.OutputDirectory()) != nil {
				return nil, errors.New("BASELINE_LIVE_BINDING_INVALID")
			}
			ids[live.ID] = true
		}
		for id := range ids {
			if r.RejectOutput(".statetwin/live/"+id) != nil {
				return nil, errors.New("BASELINE_LIVE_BINDING_INVALID")
			}
		}
	}
	if r.RejectOutput("runs") != nil || r.RejectOutput("shard.lock") != nil {
		return nil, errors.New("BASELINE_OUTPUT_INVALID")
	}
	return &Prepared{f, pack}, nil
}

func writeExclusive(fs *os.Root, name string, v []byte) error {
	if task.PortablePath(name) != nil {
		return errors.New("BASELINE_OUTPUT_INVALID")
	}
	if err := fs.MkdirAll(path.Dir(name), 0700); err != nil {
		return err
	}
	f, err := fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("BASELINE_OUTPUT_EXISTS")
	}
	n, err := f.Write(v)
	if err == nil && n != len(v) {
		err = errors.New("BASELINE_OUTPUT_FAILED")
	}
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	return closed
}
func writeJSON(fs *os.Root, name string, v any) error {
	raw, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return writeExclusive(fs, name, raw)
}

func Freeze(ctx context.Context, root, name, output string) (*Frozen, error) {
	p, err := Prepare(ctx, root, name)
	if err != nil {
		return nil, err
	}
	if p.Pack.Reader.RejectOutput(output) != nil {
		return nil, errors.New("BASELINE_OUTPUT_INVALID")
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	parts := strings.Split(output, "/")
	for i := 1; i < len(parts); i++ {
		info, e := fs.Lstat(strings.Join(parts[:i], "/"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("BASELINE_OUTPUT_INVALID")
		}
	}
	if fs.Mkdir(output, 0700) != nil {
		return nil, errors.New("BASELINE_OUTPUT_EXISTS")
	}
	dest, err := fs.OpenRoot(output)
	if err != nil {
		return nil, err
	}
	defer dest.Close()
	for n, input := range p.Pack.Reader.Inputs {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err = writeExclusive(dest, path.Join("inputs", n), input.Raw); err != nil {
			return nil, err
		}
	}
	if err = writeJSON(dest, "frozen.json", p.Frozen); err != nil {
		return nil, err
	}
	return &p.Frozen, nil
}

// Load recomputes the complete trial inventory from frozen raw definitions. No
// result, mutable model alias or caller-supplied row can redefine a denominator.
func Load(ctx context.Context, root, name string) (*Prepared, error) {
	raw, err := task.ReadFile(root, "frozen.json", 256<<10)
	if err != nil {
		return nil, err
	}
	var frozen Frozen
	if baselinepack.Decode(raw, 256<<10, &frozen) != nil || frozen.Format != FrozenFormat || frozen.RuntimeVersion != server.Version || frozen.RuntimeRevision != server.Revision {
		return nil, errors.New("BASELINE_FROZEN_INVALID")
	}
	p, err := Prepare(ctx, path.Join(root, "inputs"), name)
	if err != nil {
		return nil, err
	}
	if !baselinepack.Equal(p.Frozen, frozen) {
		return nil, errors.New("BASELINE_FROZEN_INVALID")
	}
	return p, nil
}

func SessionPlan(f Frozen, t Trial, deadline int) plugin.Plan {
	var host plugin.Host
	for _, c := range f.Plan.Configs {
		if c.ID == t.ConfigID {
			host = c.Host
		}
	}
	return plugin.Plan{Format: plugin.PlanFormat, ID: t.ID, Pack: f.Plan.Pack, Profile: f.Plan.Profile, EntryID: t.EntryID, Host: host, Output: path.Join("runs", fmt.Sprintf("shard-%02d", t.Shard), t.ID), Mode: f.Plan.Mode, DeadlineSeconds: deadline}
}
