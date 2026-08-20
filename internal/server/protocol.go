package server

const (
	MCPGoSDKModule          = "github.com/modelcontextprotocol/go-sdk"
	MCPGoSDKVersion         = "v1.7.0"
	ModernProtocolVersion   = "2026-07-28"
	LegacyProtocolVersion   = "2025-11-25"
	ProtocolEvidenceFormat  = "statetwin.dev/mcp-protocol-evidence/v1alpha1"
	StreamableHTTPTransport = "streamable-http"
)

// ProtocolProfile is a tested State Twin interoperability profile. It is not
// a promise that every optional feature in the named MCP revision is exposed.
type ProtocolProfile struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Transport  string `json:"transport"`
	Lifecycle  string `json:"lifecycle"`
	ToolsFirst bool   `json:"toolsFirst"`
	Status     string `json:"status"`
}

// ProtocolEvidence describes the protocol assumptions compiled into this
// binary. Executable wire tests, not this structure alone, establish evidence.
type ProtocolEvidence struct {
	Format          string            `json:"format"`
	RuntimeVersion  string            `json:"runtimeVersion"`
	SDKModule       string            `json:"sdkModule"`
	SDKVersion      string            `json:"sdkVersion"`
	TestedProfiles  []ProtocolProfile `json:"testedProfiles"`
	OptionalFeature []string          `json:"optionalFeaturesNotClaimed"`
}

func CurrentProtocolEvidence() ProtocolEvidence {
	return ProtocolEvidence{
		Format:         ProtocolEvidenceFormat,
		RuntimeVersion: Version,
		SDKModule:      MCPGoSDKModule,
		SDKVersion:     MCPGoSDKVersion,
		TestedProfiles: []ProtocolProfile{
			{
				Name:       "modern-tools",
				Version:    ModernProtocolVersion,
				Transport:  StreamableHTTPTransport,
				Lifecycle:  "stateless-per-request",
				ToolsFirst: true,
				Status:     "wire-tested",
			},
			{
				Name:       "legacy-tools",
				Version:    LegacyProtocolVersion,
				Transport:  StreamableHTTPTransport,
				Lifecycle:  "initialize-compatibility",
				ToolsFirst: true,
				Status:     "wire-tested",
			},
		},
		OptionalFeature: []string{
			"multi-round-trip requests",
			"tasks extension",
			"subscriptions/listen",
			"resources",
			"prompts",
		},
	}
}
