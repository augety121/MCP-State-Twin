# Phase 0: MCP 2026-07-28 Evidence Matrix

- **Status:** partial; generated from executable tests and pinned dependency
- **Last verified:** 2026-08-20
- **Runtime:** `0.1.0-dev`
- **MCP Go SDK:** `github.com/modelcontextprotocol/go-sdk v1.7.0`

| Requirement | Evidence | Status | Boundary |
|---|---|---|---|
| Explicit modern profile | `internal/server/protocol.go` and `protocols` CLI | PASS | tools-first only |
| `server/discover` | `TestMCP20260728DiscoverWireContract` | PASS | SDK-computed discovery payload |
| Direct first modern RPC | `TestMCP20260728DirectFirstToolsList` | PASS | `tools/list` only |
| Stateless modern response | raw tests assert no `Mcp-Session-Id` | PASS | Streamable HTTP |
| Header/body version consistency | `TestMCP20260728RejectsHeaderBodyVersionMismatch` | PASS | deterministic HTTP 400 |
| Modern result discriminator | raw tests assert `resultType=complete` | PASS | discover/tools-list |
| Legacy compatibility | `TestMCP20251125LegacyInitializeCompatibility` | PASS | initialize compatibility only |
| Agent control isolation | existing server tests | PASS | controls remain private |
| Official modern conformance suite | CI conformance job | OPEN | pinned suite still covers legacy-era cases |
| Tasks/MRTR/resources/prompts | no implementation | UNSUPPORTED | not a v0.1 minimum |
| Provider-specific live smoke | no credentials/tests | UNSUPPORTED | no ChatGPT/Claude compatibility claim |

The matrix distinguishes a wire-level pass from a complete MCP release claim.
The project does not claim every optional 2026 feature or every host behavior.
The `protocols` command emits the machine-readable evidence profile used by
the README and release review.
