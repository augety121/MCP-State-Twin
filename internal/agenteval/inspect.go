package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const InspectionFormat = "statetwin.dev/agent-artifact-inspection/v1alpha1"

var artifactNames = []string{"claim.json", "closure.json", "terminal.pending.json", "terminal.json"}

type ArtifactInspection struct {
	Name            string `json:"name"`
	State           string `json:"state"`
	ExecutionStatus string `json:"executionStatus,omitempty"`
	EvidenceStatus  string `json:"evidenceStatus,omitempty"`
	CleanupStatus   string `json:"cleanupStatus,omitempty"`
	WorldReplay     string `json:"worldReplay"`
}
type Inspection struct {
	Format             string               `json:"format"`
	State              string               `json:"state"`
	Lane               string               `json:"lane,omitempty"`
	TrialID            string               `json:"trialId,omitempty"`
	Files              []ArtifactInspection `json:"files"`
	EvidenceComplete   bool                 `json:"evidenceComplete"`
	StagingResidue     bool                 `json:"stagingResidue"`
	ResumeAllowed      bool                 `json:"resumeAllowed"`
	SnapshotAtomic     bool                 `json:"snapshotAtomic"`
	ProviderProvenance string               `json:"providerProvenance"`
	Problem            string               `json:"problem,omitempty"`
}

// InspectDirectory is read-only. Run against a quiescent, trusted local root.
// It never logs artifact content, invokes a model, promotes staging or resumes.
// Observations across separate files are not an atomic directory snapshot.
func InspectDirectory(ctx context.Context, root, out string) (*Inspection, error) {
	if task.PortablePath(out) != nil {
		return nil, errors.New("EVIDENCE_INSPECT_PATH_INVALID")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, errors.New("EVIDENCE_INSPECT_CANCELED_OR_TIMED_OUT")
	}
	r := &Inspection{Format: InspectionFormat, State: "incomplete_or_running", ProviderProvenance: "not-proven", Files: []ArtifactInspection{}}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("EVIDENCE_INSPECT_ROOT_UNAVAILABLE")
	}
	defer fs.Close()
	parts := strings.Split(out, "/")
	for i := range parts {
		info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if os.IsNotExist(err) {
			r.State = "not_started"
			return r, nil
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, errors.New("EVIDENCE_INSPECT_PATH_INVALID")
		}
	}
	dir, err := fs.Open(out)
	if err != nil {
		return nil, errors.New("EVIDENCE_INSPECT_FAILED")
	}
	entries, readErr := dir.ReadDir(len(artifactNames) + 1)
	closeErr := dir.Close()
	if (readErr != nil && !errors.Is(readErr, io.EOF)) || closeErr != nil {
		return nil, errors.New("EVIDENCE_INSPECT_FAILED")
	}
	present := map[string]bool{}
	for _, entry := range entries {
		known := false
		for _, name := range artifactNames {
			if entry.Name() == name {
				known = true
			}
		}
		if !known || entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			r.State = "invalid"
			r.Problem = "unexpected_or_unsafe_member"
			return r, nil
		}
		present[entry.Name()] = true
	}
	if len(entries) == 0 {
		return r, nil
	}
	var claim any
	if present["claim.json"] {
		raw, err := readArtifact(fs, path.Join(out, "claim.json"), task.MaxBytes+(16<<10))
		if err == nil && !logging.ContainsSensitive(string(raw)) && json.Valid(raw) {
			if c, err := DecodeRun(raw); err == nil {
				claim = c
				r.Lane = "offline"
				r.TrialID = c.TrialID
			} else if p, err := agentapi.DecodePlan(raw); err == nil {
				claim = p
				r.Lane = "local-api"
				r.TrialID = p.ID
			}
		}
	}
	claimState := "valid"
	if claim == nil {
		claimState = "invalid"
		r.State = "invalid"
		r.Problem = "claim_missing_or_invalid"
	}
	r.Files = append(r.Files, ArtifactInspection{Name: "claim.json", State: claimState, WorldReplay: "not_checked"})
	if claim == nil {
		return r, nil
	}
	var previous *inspectedArtifact
	for _, name := range artifactNames[1:] {
		if err := ctx.Err(); err != nil {
			return nil, errors.New("EVIDENCE_INSPECT_CANCELED_OR_TIMED_OUT")
		}
		item := ArtifactInspection{Name: name, State: "absent", WorldReplay: "not_checked"}
		if !present[name] {
			r.Files = append(r.Files, item)
			continue
		}
		if name != "terminal.json" {
			r.StagingResidue = true
		}
		raw, readErr := readArtifact(fs, path.Join(out, name), limits.MaxReportBytes)
		var a *inspectedArtifact
		if readErr == nil && !logging.ContainsSensitive(string(raw)) {
			a = decodeInspected(raw)
		}
		if a == nil || !same(claim, a.claim) {
			item.State = "invalid"
			r.State = "invalid"
			r.Problem = "artifact_invalid_or_claim_mismatch"
			r.Files = append(r.Files, item)
			continue
		}
		if previous != nil && !same(previous.core(), a.core()) {
			r.State = "invalid"
			r.Problem = "staging_conflict"
		}
		// Pending and published bytes must have identical semantic values.
		if previous != nil && previous.name == "terminal.pending.json" && name == "terminal.json" && !same(previous.value, a.value) {
			r.State = "invalid"
			r.Problem = "publication_conflict"
		}
		a.name = name
		previous = a
		terminal := name != "closure.json"
		err := a.verify(ctx, terminal)
		item.State = "invalid"
		if err == nil {
			item.State = "verified"
			item.WorldReplay = "matched"
			if terminal && a.episode.CleanupStatus == "complete" && a.episode.EvidenceStatus == "complete" && a.episode.WorldReplayable && name == "terminal.json" {
				r.EvidenceComplete = true
			}
		} else if err.Error() == "EVIDENCE_PARTIAL" {
			item.State = "partial_unverified"
		} else {
			r.State = "invalid"
			r.Problem = "artifact_verification_failed"
		}
		// Only enumerated statuses can enter a diagnostic report. No arbitrary
		// failure message, answer, model body, file path or user data is echoed.
		if !knownStatuses(a.episode) {
			item.State = "invalid"
			r.State = "invalid"
			r.Problem = "invalid_status"
		} else {
			item.ExecutionStatus = a.episode.ExecutionStatus
			item.EvidenceStatus = a.episode.EvidenceStatus
			item.CleanupStatus = a.episode.CleanupStatus
		}
		r.Files = append(r.Files, item)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.New("EVIDENCE_INSPECT_CANCELED_OR_TIMED_OUT")
	}
	if r.State == "invalid" {
		r.EvidenceComplete = false
		return r, nil
	}
	if r.EvidenceComplete {
		r.State = "published"
		if r.StagingResidue {
			r.State = "published_with_residue"
		}
	}
	return r, nil
}

func readArtifact(fs *os.Root, name string, max int) ([]byte, error) {
	info, err := fs.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > int64(max) {
		return nil, errors.New("EVIDENCE_MEMBER_INVALID")
	}
	f, err := fs.Open(name)
	if err != nil {
		return nil, errors.New("EVIDENCE_MEMBER_INVALID")
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("EVIDENCE_MEMBER_INVALID")
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(max)+1))
	if err != nil || len(raw) > max {
		return nil, errors.New("EVIDENCE_MEMBER_INVALID")
	}
	return raw, nil
}

type inspectedArtifact struct {
	name    string
	claim   any
	episode *AgentEpisode
	value   any
	bundle  string
	verify  func(context.Context, bool) error
}

func decodeInspected(raw []byte) *inspectedArtifact {
	if e, err := DecodeEvidence(raw); err == nil {
		return &inspectedArtifact{claim: &e.Episode.Definition.Config, episode: e.Episode, value: e, bundle: e.Bundle, verify: func(ctx context.Context, terminal bool) error { return replay(ctx, e, terminal) }}
	}
	if e, err := DecodeLiveEvidence(raw); err == nil {
		return &inspectedArtifact{claim: e.Plan, episode: e.Episode, value: e, bundle: e.Bundle, verify: func(ctx context.Context, terminal bool) error { return verifyLive(ctx, e, terminal) }}
	}
	return nil
}
func (a *inspectedArtifact) core() any {
	r := *a.episode
	r.CleanupStatus = ""
	r.EvidenceStatus = ""
	r.WorldReplayable = false
	r.CleanupFailureCode = ""
	if r.FailureCode == "CLEANUP_FAILED" {
		r.FailureCode = ""
	}
	if e, ok := a.value.(*LiveEvidence); ok {
		copy := *e
		copy.Episode = &r
		return copy
	}
	return &AgentEvidence{Format: EvidenceFormat, Bundle: a.bundle, Episode: &r}
}
func knownStatuses(r *AgentEpisode) bool {
	execution := map[string]bool{"running": true, "completed": true, "canceled": true, "timed_out": true, "budget_exhausted": true, "host_error": true}
	return execution[r.ExecutionStatus] && (r.EvidenceStatus == "partial" || r.EvidenceStatus == "complete") && (r.CleanupStatus == "pending" || r.CleanupStatus == "complete" || r.CleanupStatus == "failed")
}
