package spec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"gopkg.in/yaml.v3"
)

const (
	APIVersion = "statetwin.dev/v1alpha1"
	Kind       = "Twin"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type TwinSpec struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Kind       string          `json:"kind" yaml:"kind"`
	Metadata   Metadata        `json:"metadata" yaml:"metadata"`
	Clock      ClockSpec       `json:"clock" yaml:"clock"`
	State      StateSpec       `json:"state" yaml:"state"`
	Invariants []InvariantSpec `json:"invariants,omitempty" yaml:"invariants,omitempty"`
	Tools      []ToolSpec      `json:"tools" yaml:"tools"`
}

type Metadata struct {
	Name     string           `json:"name" yaml:"name"`
	Upstream UpstreamMetadata `json:"upstream" yaml:"upstream"`
	Fidelity FidelityMetadata `json:"fidelity" yaml:"fidelity"`
}

type UpstreamMetadata struct {
	Protocol      string `json:"protocol" yaml:"protocol"`
	Status        string `json:"status" yaml:"status"`
	SurfaceDigest string `json:"surfaceDigest,omitempty" yaml:"surfaceDigest,omitempty"`
}

type FidelityMetadata struct {
	Level  string `json:"level" yaml:"level"`
	Status string `json:"status" yaml:"status"`
}

type ClockSpec struct {
	Mode    string `json:"mode" yaml:"mode"`
	Initial string `json:"initial" yaml:"initial"`
}

type StateSpec struct {
	Entities map[string]EntitySpec `json:"entities" yaml:"entities"`
}

type EntitySpec struct {
	Key []string `json:"key" yaml:"key"`
}

type InvariantSpec struct {
	ID     string `json:"id" yaml:"id"`
	Assert string `json:"assert" yaml:"assert"`
}

type ToolSpec struct {
	Name           string         `json:"name" yaml:"name"`
	Description    string         `json:"description" yaml:"description"`
	InputSchema    map[string]any `json:"inputSchema" yaml:"inputSchema"`
	OutputSchema   map[string]any `json:"outputSchema,omitempty" yaml:"outputSchema,omitempty"`
	Modeled        *bool          `json:"modeled,omitempty" yaml:"modeled,omitempty"`
	Preconditions  []Condition    `json:"preconditions,omitempty" yaml:"preconditions,omitempty"`
	Effects        []Effect       `json:"effects,omitempty" yaml:"effects,omitempty"`
	Query          *Query         `json:"query,omitempty" yaml:"query,omitempty"`
	Postconditions []Condition    `json:"postconditions,omitempty" yaml:"postconditions,omitempty"`
	Result         string         `json:"result,omitempty" yaml:"result,omitempty"`
}

type Condition struct {
	Expr    string `json:"expr" yaml:"expr"`
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type Effect struct {
	Op       string `json:"op" yaml:"op"`
	Entity   string `json:"entity,omitempty" yaml:"entity,omitempty"`
	Key      string `json:"key,omitempty" yaml:"key,omitempty"`
	Value    string `json:"value,omitempty" yaml:"value,omitempty"`
	Sequence string `json:"sequence,omitempty" yaml:"sequence,omitempty"`
	As       string `json:"as,omitempty" yaml:"as,omitempty"`
	Merge    bool   `json:"merge,omitempty" yaml:"merge,omitempty"`
}

type Query struct {
	Entity string `json:"entity" yaml:"entity"`
	Key    string `json:"key,omitempty" yaml:"key,omitempty"`
	Where  string `json:"where,omitempty" yaml:"where,omitempty"`
	Many   bool   `json:"many,omitempty" yaml:"many,omitempty"`
	As     string `json:"as" yaml:"as"`
}

func Load(path string) (*TwinSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read TwinSpec: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var result TwinSpec
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode TwinSpec: %w", err)
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TwinSpec) Digest() (string, error) {
	return canonical.Digest(s)
}

func (s *TwinSpec) Validate() error {
	var problems []string
	if s.APIVersion != APIVersion {
		problems = append(problems, fmt.Sprintf("apiVersion must be %q", APIVersion))
	}
	if s.Kind != Kind {
		problems = append(problems, fmt.Sprintf("kind must be %q", Kind))
	}
	if !namePattern.MatchString(s.Metadata.Name) {
		problems = append(problems, "metadata.name must use lowercase letters, digits, '-' or '_'")
	}
	if s.Metadata.Upstream.Protocol != "mcp" {
		problems = append(problems, "metadata.upstream.protocol must be mcp")
	}
	switch s.Metadata.Upstream.Status {
	case "current", "drifted", "unknown":
		if strings.TrimSpace(s.Metadata.Upstream.SurfaceDigest) == "" {
			problems = append(problems, "metadata.upstream.surfaceDigest is required unless status is unbound")
		}
	case "unbound":
		if strings.TrimSpace(s.Metadata.Upstream.SurfaceDigest) != "" {
			problems = append(problems, "metadata.upstream.surfaceDigest must be empty when status is unbound")
		}
	default:
		problems = append(problems, "metadata.upstream.status must be current, drifted, unknown, or unbound")
	}
	if s.Metadata.Fidelity.Level != "L0" && s.Metadata.Fidelity.Level != "L1" && s.Metadata.Fidelity.Level != "L2" && s.Metadata.Fidelity.Level != "L3" {
		problems = append(problems, "metadata.fidelity.level must be L0, L1, L2, or L3")
	}
	if s.Metadata.Fidelity.Status != "unverified" && s.Metadata.Fidelity.Status != "verified" {
		problems = append(problems, "metadata.fidelity.status must be unverified or verified")
	}
	if (s.Metadata.Fidelity.Level == "L0" || s.Metadata.Fidelity.Level == "L1") && s.Metadata.Fidelity.Status == "verified" {
		problems = append(problems, "L0/L1 artifacts cannot claim verified status")
	}
	if s.Clock.Mode != "virtual" {
		problems = append(problems, "clock.mode must be virtual")
	}
	if strings.TrimSpace(s.Clock.Initial) == "" {
		problems = append(problems, "clock.initial is required")
	}
	if len(s.State.Entities) == 0 {
		problems = append(problems, "state.entities must not be empty")
	}

	toolNames := make(map[string]struct{}, len(s.Tools))
	for i, tool := range s.Tools {
		prefix := fmt.Sprintf("tools[%d]", i)
		if !namePattern.MatchString(tool.Name) {
			problems = append(problems, prefix+".name is invalid")
		}
		if _, exists := toolNames[tool.Name]; exists {
			problems = append(problems, prefix+".name is duplicated")
		}
		toolNames[tool.Name] = struct{}{}
		if strings.TrimSpace(tool.Description) == "" {
			problems = append(problems, prefix+".description is required")
		}
		if tool.InputSchema == nil {
			problems = append(problems, prefix+".inputSchema is required")
		}
		for j, effect := range tool.Effects {
			ep := fmt.Sprintf("%s.effects[%d]", prefix, j)
			switch effect.Op {
			case "allocate":
				if effect.Sequence == "" || effect.As == "" {
					problems = append(problems, ep+" allocate requires sequence and as")
				}
			case "insert", "update", "delete":
				if _, ok := s.State.Entities[effect.Entity]; !ok {
					problems = append(problems, ep+" references unknown entity "+effect.Entity)
				}
				if effect.Key == "" {
					problems = append(problems, ep+" requires key expression")
				}
				if effect.Op != "delete" && effect.Value == "" {
					problems = append(problems, ep+" requires value expression")
				}
			default:
				problems = append(problems, ep+" has unsupported op "+effect.Op)
			}
		}
		if tool.Query != nil {
			if _, ok := s.State.Entities[tool.Query.Entity]; !ok {
				problems = append(problems, prefix+".query references unknown entity "+tool.Query.Entity)
			}
			if tool.Query.As == "" {
				problems = append(problems, prefix+".query.as is required")
			}
		}
	}

	invariantIDs := make([]string, 0, len(s.Invariants))
	seenInvariant := make(map[string]struct{}, len(s.Invariants))
	for _, invariant := range s.Invariants {
		if invariant.ID == "" || invariant.Assert == "" {
			problems = append(problems, "every invariant requires id and assert")
		}
		if _, exists := seenInvariant[invariant.ID]; exists {
			problems = append(problems, "duplicate invariant id "+invariant.ID)
		}
		seenInvariant[invariant.ID] = struct{}{}
		invariantIDs = append(invariantIDs, invariant.ID)
	}
	sort.Strings(invariantIDs)

	if len(problems) > 0 {
		return errors.New("invalid TwinSpec:\n- " + strings.Join(problems, "\n- "))
	}
	return nil
}
