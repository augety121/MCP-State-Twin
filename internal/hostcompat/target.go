package hostcompat

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
)

const (
	TargetKind     = "HostCompatibilityTarget"
	TargetFormat   = "statetwin.dev/host-compatibility-target/v1alpha1"
	MaxTargetBytes = 1 << 20
)

// Target is an operator-supplied declaration, not execution evidence. It has no
// outcome, timestamp, claim level, source endpoint or report-derived defaults.
type Target struct {
	APIVersion      string       `json:"apiVersion" yaml:"apiVersion"`
	Kind            string       `json:"kind" yaml:"kind"`
	Format          string       `json:"format" yaml:"format"`
	Runtime         *Runtime     `json:"runtime" yaml:"runtime"`
	Host            *Host        `json:"host" yaml:"host"`
	MCP             *MCP         `json:"mcp" yaml:"mcp"`
	ProcedureDigest string       `json:"procedureDigest" yaml:"procedureDigest"`
	Trial           *TargetTrial `json:"trial" yaml:"trial"`
	RedactionPolicy string       `json:"redactionPolicy" yaml:"redactionPolicy"`
}

type TargetTrial struct {
	ScenarioDigest   string        `json:"scenarioDigest" yaml:"scenarioDigest"`
	PromptDigest     string        `json:"promptDigest" yaml:"promptDigest"`
	ToolPolicyDigest string        `json:"toolPolicyDigest" yaml:"toolPolicyDigest"`
	Limits           *TargetLimits `json:"limits" yaml:"limits"`
}

// Pointers distinguish an explicit zero from omitted or null budget fields.
type TargetLimits struct {
	ProviderRequests          *int `json:"providerRequests" yaml:"providerRequests"`
	ToolCalls                 *int `json:"toolCalls" yaml:"toolCalls"`
	WallTimeMS                *int `json:"wallTimeMs" yaml:"wallTimeMs"`
	MaxTraceBytes             *int `json:"maxTraceBytes" yaml:"maxTraceBytes"`
	RetriesPerProviderRequest *int `json:"retriesPerProviderRequest" yaml:"retriesPerProviderRequest"`
	RetriesPerToolCall        *int `json:"retriesPerToolCall" yaml:"retriesPerToolCall"`
	RepeatedIdenticalCalls    *int `json:"repeatedIdenticalCalls" yaml:"repeatedIdenticalCalls"`
}

func (l *TargetLimits) values() (Limits, error) {
	if l == nil || l.ProviderRequests == nil || l.ToolCalls == nil || l.WallTimeMS == nil || l.MaxTraceBytes == nil || l.RetriesPerProviderRequest == nil || l.RetriesPerToolCall == nil || l.RepeatedIdenticalCalls == nil {
		return Limits{}, errors.New("HOST_TARGET_LIMITS_REQUIRED")
	}
	return Limits{
		ProviderRequests: *l.ProviderRequests, ToolCalls: *l.ToolCalls,
		WallTimeMS: *l.WallTimeMS, MaxTraceBytes: *l.MaxTraceBytes,
		RetriesPerProviderRequest: *l.RetriesPerProviderRequest,
		RetriesPerToolCall:        *l.RetriesPerToolCall, RepeatedIdenticalCalls: *l.RepeatedIdenticalCalls,
	}, nil
}

func LoadTarget(path string) (*Target, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxTargetBytes {
		return nil, errors.New("HOST_TARGET_FILE_INVALID")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("HOST_TARGET_FILE_INVALID")
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxTargetBytes {
		return nil, errors.New("HOST_TARGET_FILE_INVALID")
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxTargetBytes+1))
	if err != nil {
		return nil, errors.New("HOST_TARGET_FILE_INVALID")
	}
	return DecodeTarget(raw)
}

func DecodeTarget(raw []byte) (*Target, error) {
	if len(raw) > MaxTargetBytes {
		return nil, errors.New("HOST_TARGET_RESOURCE_LIMIT")
	}
	if logging.ContainsSensitive(string(raw)) {
		return nil, errors.New("HOST_TARGET_SENSITIVE_CONTENT")
	}
	var target Target
	if err := strictyaml.DecodeOne(raw, MaxTargetBytes, TargetKind, &target); err != nil {
		return nil, errors.New("HOST_TARGET_DECODE_INVALID")
	}
	if !explicitIntegerFields(raw, false) {
		return nil, errors.New("HOST_TARGET_INTEGER_FIELDS_REQUIRED")
	}
	if err := target.Validate(); err != nil {
		return nil, err
	}
	return &target, nil
}

func (t *Target) Validate() error {
	if t == nil || t.Runtime == nil || t.Host == nil || t.MCP == nil || t.Trial == nil {
		return errors.New("HOST_TARGET_REQUIRED_FIELDS")
	}
	raw, err := json.Marshal(t)
	if err != nil || len(raw) > MaxTargetBytes {
		return errors.New("HOST_TARGET_RESOURCE_LIMIT")
	}
	if logging.ContainsSensitive(string(raw)) {
		return errors.New("HOST_TARGET_SENSITIVE_CONTENT")
	}
	limits, err := t.Trial.Limits.values()
	if err != nil {
		return err
	}
	var problems []string
	requireEqual(&problems, "apiVersion", t.APIVersion, APIVersion)
	requireEqual(&problems, "kind", t.Kind, TargetKind)
	requireEqual(&problems, "format", t.Format, TargetFormat)
	validateRuntime(&problems, *t.Runtime, true)
	// Reuse only the identity validators with their strict-claim rules. This
	// private view is never validated as a report, persisted or returned as one.
	view := &Report{Claim: Claim{Level: "verified"}, Runtime: *t.Runtime, Host: *t.Host, MCP: *t.MCP}
	validateHost(&problems, view)
	validateMCP(&problems, view)
	validateLimits(&problems, t.Host.Profile, limits)
	requireDigest(&problems, "mcp.observedSurfaceDigest", t.MCP.ObservedSurfaceDigest)
	requireDigest(&problems, "procedureDigest", t.ProcedureDigest)
	requireDigest(&problems, "trial.scenarioDigest", t.Trial.ScenarioDigest)
	requireDigest(&problems, "trial.promptDigest", t.Trial.PromptDigest)
	requireDigest(&problems, "trial.toolPolicyDigest", t.Trial.ToolPolicyDigest)
	requireEqual(&problems, "redactionPolicy", t.RedactionPolicy, "synthetic-only-v1")
	if len(problems) != 0 {
		return errors.New("HOST_TARGET_IDENTITY_INVALID")
	}
	return nil
}
