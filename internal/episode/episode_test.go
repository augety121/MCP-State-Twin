package episode

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func TestLifecycleAllowsOnlyDeclaredTransitions(t *testing.T) {
	lifecycle := NewLifecycle()
	for _, status := range []Status{StatusProvisioning, StatusReady, StatusRunning, StatusEvaluating, StatusSucceeded} {
		if err := lifecycle.Transition(status); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	if err := lifecycle.Transition(StatusRunning); err == nil {
		t.Fatal("terminal lifecycle accepted another transition")
	}
	if lifecycle.Current != StatusSucceeded || len(lifecycle.Events) != 6 {
		t.Fatalf("unexpected lifecycle: %+v", lifecycle)
	}
}

func TestBundledEpisodeProducesDeterministicEvidence(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	bundlePath := filepath.Join(t.TempDir(), "issue-tracker.stb")
	if _, err := bundle.Build(manifestPath, bundlePath); err != nil {
		t.Fatal(err)
	}
	artifact, err := bundle.Open(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Run(context.Background(), artifact, "episode-001", "", "0.2.0-dev", "0123456789012345678901234567890123456789")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(context.Background(), artifact, "episode-001", "", "0.2.0-dev", "0123456789012345678901234567890123456789")
	if err != nil {
		t.Fatal(err)
	}
	if first.EvidenceDigest != second.EvidenceDigest {
		t.Fatalf("evidence is not deterministic: %s != %s", first.EvidenceDigest, second.EvidenceDigest)
	}
	if first.Evidence.Outcome != "succeeded" || first.Evidence.Lifecycle.Current != StatusSucceeded {
		t.Fatalf("unexpected episode result: %+v", first.Evidence)
	}
	if first.Evidence.Report.AgentIdentity != "scripted-scenario" {
		t.Fatal("local scripted episode was mislabeled as a live agent")
	}
}

func TestEpisodeRejectsInvalidIdentityAndUndeclaredScenario(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	bundlePath := filepath.Join(t.TempDir(), "issue-tracker.stb")
	if _, err := bundle.Build(manifestPath, bundlePath); err != nil {
		t.Fatal(err)
	}
	artifact, err := bundle.Open(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), artifact, "bad/id", "", "dev", "unknown"); err == nil {
		t.Fatal("invalid episode ID was accepted")
	}
	if _, err := Run(context.Background(), artifact, "episode-002", "hidden.yaml", "dev", "unknown"); err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("expected undeclared scenario refusal, got %v", err)
	}
	artifact.Manifest.Scenarios = append(artifact.Manifest.Scenarios, "second.yaml")
	if _, err := Run(context.Background(), artifact, "episode-003", "", "dev", "unknown"); err == nil || !strings.Contains(err.Error(), "--scenario is required") {
		t.Fatalf("expected ambiguous Scenario refusal, got %v", err)
	}
}
