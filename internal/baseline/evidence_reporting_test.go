package baseline

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func TestBaselineInvalidTrialsSuppressInference(t *testing.T) {
	_, p := fixture(t)
	f := p.Frozen
	for _, condition := range []string{"missing", "unverified", "interrupted", "cleanup"} {
		t.Run(condition, func(t *testing.T) {
			rows := outcomes(f, true)
			switch condition {
			case "missing":
				rows = rows[1:]
			case "unverified":
				rows[0].Verified = false
			case "interrupted":
				rows[0].State = "timeout"
			case "cleanup":
				rows[0].Cleanup = "failed"
			}
			r, err := Decide(f, rows)
			if err != nil || r.Decision != "invalid" {
				t.Fatal(r, err)
			}
			if r.Interval.Status != "invalid" || r.Interval.Lower != nil || r.Interval.Upper != nil || r.Interval.Samples != 0 {
				t.Fatalf("invalid cohort produced inference: %+v", r.Interval)
			}
			if len(r.Trials) != 144 || r.Metrics[f.Trials[0].ConfigID].Planned != 72 || r.Metrics[f.Trials[1].ConfigID].Planned != 72 {
				t.Fatal("invalid evidence changed the frozen denominator")
			}
			if r.Interval.Families != 24 || r.Interval.Algorithm != Algorithm || r.Interval.Seed != f.Plan.Policy.Seed {
				t.Fatalf("lost inference metadata: %+v", r.Interval)
			}
		})
	}
}

func TestBaselineLiveReportedModelFromVerifiedReceipts(t *testing.T) {
	root, p := fixture(t)
	ctx := context.Background()
	trial := p.Frozen.Trials[0]
	entry := p.Pack.Find(trial.EntryID)
	b, err := bundle.OpenBytes(entry.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "issue-tracker", "agent-mocks", "read-issue.json"))
	if err != nil {
		t.Fatal(err)
	}
	script, err := agenthost.DecodeMock(raw)
	if err != nil {
		t.Fatal(err)
	}
	config := &agenteval.RunConfig{Format: agenteval.ConfigFormat, TrialID: trial.ID, Profile: agenthost.Profile, Model: "mock-statetwin", MaxOutputTokens: 128, SyntheticOnly: true}
	episode, err := agenteval.RecordMock(ctx, root, "synthetic-source", entry.Task, entry.Bundle, config, script)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	live := &agentapi.Plan{Format: agentapi.PlanFormat, ID: trial.ID, Provider: "openai", Profile: agentapi.Profile, Model: "requested-alias", Task: entry.Task, BundleDigest: b.Digest, IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), MaxRequests: entry.Task.Budgets.ModelRequests, MaxOutputTokens: 128, CostPolicy: agentapi.CostPolicy, Approved: true, SyntheticDataApproved: true, UnknownCostApproved: true}
	// Construct a synthetic unsigned envelope to exercise the real assessment
	// boundary. No provider client, credentials, or network request is used.
	episode.Format = agenteval.LiveEpisodeFormat
	episode.Source = "provider-live"
	episode.Definition.Config = agenteval.RunConfig{Format: agenteval.LiveConfigFormat, TrialID: live.ID, Profile: agentapi.Profile, Model: live.Model, MaxOutputTokens: live.MaxOutputTokens, SyntheticOnly: true}
	episode.Definition.Isolation = "in-process-tools-fixed-provider-egress"
	episode.Definition.Projection = agenteval.LiveProfile
	episode.Definition.ModelSnapshot = "unknown"
	evidence := &agenteval.LiveEvidence{Format: agenteval.LiveEvidenceFormat, Plan: live, StartedAt: now.Format(time.RFC3339Nano), Bundle: base64.StdEncoding.EncodeToString(entry.Bundle), Episode: episode}
	for i := 0; i < episode.Usage.ModelRequests; i++ {
		evidence.Receipts = append(evidence.Receipts, agentapi.Receipt{Sequence: i + 1, AdmittedAt: now.Format(time.RFC3339Nano), Outcome: "response_received", HTTPStatus: 200, ReportedModel: "reported-model-v1", Cost: "unknown"})
	}
	if len(evidence.Receipts) < 2 {
		t.Fatal("fixture must exercise multiple response receipts")
	}
	write := func(name string, value any) {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(full, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	liveName := "synthetic-live-plan.json"
	write(liveName, live)
	if _, err = p.Pack.Reader.Read(liveName, 64<<10); err != nil {
		t.Fatal(err)
	}
	p.Frozen.Plan.LivePlans = map[string]string{trial.ID: liveName}
	write(path.Join("runs", "shard-01", trial.ID+".plan.json"), live)
	for _, condition := range []string{"single", "mixed", "invalid"} {
		t.Run(condition, func(t *testing.T) {
			evidence.Receipts[1].ReportedModel = "reported-model-v1"
			want := "reported-model-v1"
			if condition == "mixed" {
				evidence.Receipts[1].ReportedModel = "reported-model-v2"
				want = "mixed"
			}
			if condition == "invalid" {
				evidence.Receipts[1].ReportedModel = ""
				want = "unknown"
			}
			write(path.Join(live.OutputDirectory(), "terminal.json"), evidence)
			o, err := inspectLiveTrial(ctx, root, p, trial)
			if condition == "invalid" {
				if err == nil || o.Verified {
					t.Fatal("invalid receipt promoted to verified observation")
				}
			} else if err != nil || !o.Verified || !o.Success {
				t.Fatal(o, err)
			}
			if o.ObservedModel != want || o.Usage.Status != "unavailable" {
				t.Fatalf("receipt model lost when token usage is unavailable: %+v", o)
			}
		})
	}
}
