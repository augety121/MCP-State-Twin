package episode

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/scenario"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/world"
)

const (
	APIVersion     = "statetwin.dev/v1alpha1"
	Kind           = "EpisodeEvidence"
	EvidenceFormat = "statetwin.dev/episode-evidence/v1alpha1"
)

type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusProvisioning    Status = "PROVISIONING"
	StatusReady           Status = "READY"
	StatusRunning         Status = "RUNNING"
	StatusEvaluating      Status = "EVALUATING"
	StatusSucceeded       Status = "SUCCEEDED"
	StatusAssertionFailed Status = "ASSERTION_FAILED"
	StatusRuntimeError    Status = "RUNTIME_ERROR"
	StatusCancelled       Status = "CANCELLED"
)

var episodeIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type Lifecycle struct {
	Current Status  `json:"current"`
	Events  []Event `json:"events"`
}

type Event struct {
	Sequence int    `json:"sequence"`
	From     Status `json:"from,omitempty"`
	To       Status `json:"to"`
}

type RuntimeIdentity struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

type Evidence struct {
	APIVersion   string           `json:"apiVersion"`
	Kind         string           `json:"kind"`
	Format       string           `json:"format"`
	EpisodeID    string           `json:"episodeId"`
	ClaimLevel   string           `json:"claimLevel"`
	BundleDigest string           `json:"bundleDigest"`
	ScenarioPath string           `json:"scenarioPath"`
	Runtime      RuntimeIdentity  `json:"runtime"`
	Lifecycle    Lifecycle        `json:"lifecycle"`
	Outcome      string           `json:"outcome"`
	Report       *scenario.Report `json:"report"`
}

type Envelope struct {
	APIVersion     string    `json:"apiVersion"`
	Kind           string    `json:"kind"`
	Evidence       *Evidence `json:"evidence"`
	EvidenceDigest string    `json:"evidenceDigest"`
}

type transitionObserver func(Event) error

func NewLifecycle() Lifecycle {
	return Lifecycle{Current: StatusCreated, Events: []Event{{Sequence: 0, To: StatusCreated}}}
}

func (l *Lifecycle) Transition(next Status) error {
	if l == nil {
		return errors.New("episode lifecycle is required")
	}
	if !allowedTransition(l.Current, next) {
		return fmt.Errorf("invalid episode transition %s -> %s", l.Current, next)
	}
	l.Events = append(l.Events, Event{Sequence: len(l.Events), From: l.Current, To: next})
	l.Current = next
	return nil
}

func (e *Evidence) Digest() (string, error) {
	return canonical.Digest(e)
}

func Run(ctx context.Context, artifact *bundle.Artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision string) (*Envelope, error) {
	return run(ctx, artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision, nil)
}

func run(ctx context.Context, artifact *bundle.Artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision string, observer transitionObserver) (*Envelope, error) {
	if artifact == nil {
		return nil, errors.New("TwinBundle artifact is required")
	}
	if !episodeIDPattern.MatchString(episodeID) {
		return nil, errors.New("episode ID must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	}
	if runtimeVersion == "" {
		return nil, errors.New("runtime version is required")
	}
	if runtimeRevision == "" {
		runtimeRevision = "unknown"
	}
	selected, err := selectScenario(artifact.Manifest, scenarioPath)
	if err != nil {
		return nil, err
	}
	lifecycle := NewLifecycle()
	if err := transition(&lifecycle, StatusProvisioning, observer); err != nil {
		return nil, err
	}
	twin, err := spec.Decode(artifact.Files[artifact.Manifest.Spec])
	if err != nil {
		return nil, fmt.Errorf("decode bundled TwinSpec: %w", err)
	}
	initial, err := decodeFixture(artifact.Files[artifact.Manifest.Fixture])
	if err != nil {
		return nil, err
	}
	testScenario, err := scenario.Decode(artifact.Files[selected])
	if err != nil {
		return nil, fmt.Errorf("decode bundled Scenario: %w", err)
	}
	if err := transition(&lifecycle, StatusReady, observer); err != nil {
		return nil, err
	}
	if err := transition(&lifecycle, StatusRunning, observer); err != nil {
		return nil, err
	}
	report, err := scenario.Run(ctx, twin, initial, testScenario, runtimeVersion)
	if err != nil {
		_ = transition(&lifecycle, StatusRuntimeError, observer)
		return nil, fmt.Errorf("run bundled Scenario: %w", err)
	}
	if err := transition(&lifecycle, StatusEvaluating, observer); err != nil {
		return nil, err
	}
	outcome := "succeeded"
	terminal := StatusSucceeded
	if !report.Passed {
		outcome = "assertion_failed"
		terminal = StatusAssertionFailed
	}
	if err := transition(&lifecycle, terminal, observer); err != nil {
		return nil, err
	}
	evidence := &Evidence{
		APIVersion: APIVersion, Kind: Kind, Format: EvidenceFormat,
		EpisodeID: episodeID, ClaimLevel: "development", BundleDigest: artifact.Digest,
		ScenarioPath: selected, Runtime: RuntimeIdentity{Version: runtimeVersion, Revision: runtimeRevision},
		Lifecycle: lifecycle, Outcome: outcome, Report: report,
	}
	if err := limits.ValidateJSON(evidence, limits.MaxReportBytes); err != nil {
		return nil, fmt.Errorf("RESOURCE_LIMIT: episode evidence: %w", err)
	}
	digest, err := evidence.Digest()
	if err != nil {
		return nil, err
	}
	return &Envelope{APIVersion: APIVersion, Kind: "EpisodeEvidenceEnvelope", Evidence: evidence, EvidenceDigest: digest}, nil
}

func transition(lifecycle *Lifecycle, next Status, observer transitionObserver) error {
	if err := lifecycle.Transition(next); err != nil {
		return err
	}
	if observer == nil {
		return nil
	}
	event := lifecycle.Events[len(lifecycle.Events)-1]
	if err := observer(event); err != nil {
		return fmt.Errorf("persist episode transition %s -> %s: %w", event.From, event.To, err)
	}
	return nil
}

func IsTerminal(status Status) bool {
	switch status {
	case StatusSucceeded, StatusAssertionFailed, StatusRuntimeError, StatusCancelled:
		return true
	default:
		return false
	}
}

func allowedTransition(current, next Status) bool {
	switch current {
	case StatusCreated:
		return next == StatusProvisioning || next == StatusCancelled
	case StatusProvisioning:
		return next == StatusReady || next == StatusRuntimeError || next == StatusCancelled
	case StatusReady:
		return next == StatusRunning || next == StatusCancelled
	case StatusRunning:
		return next == StatusEvaluating || next == StatusRuntimeError || next == StatusCancelled
	case StatusEvaluating:
		return next == StatusSucceeded || next == StatusAssertionFailed || next == StatusRuntimeError
	default:
		return false
	}
}

func selectScenario(manifest bundle.Manifest, requested string) (string, error) {
	if requested == "" {
		if len(manifest.Scenarios) != 1 {
			return "", errors.New("--scenario is required when a TwinBundle contains multiple scenarios")
		}
		return manifest.Scenarios[0], nil
	}
	for _, candidate := range manifest.Scenarios {
		if candidate == requested {
			return requested, nil
		}
	}
	return "", fmt.Errorf("scenario %q is not declared by the TwinBundle", requested)
}

func decodeFixture(data []byte) (*world.State, error) {
	state, err := world.DecodeStrict(data)
	if err != nil {
		return nil, fmt.Errorf("decode bundled fixture: %w", err)
	}
	return state, nil
}
