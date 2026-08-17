# ADR-0001: MCP Protocol Baseline and Provider Neutrality

- **Status:** Accepted for v0.1
- **Date:** 2026-08-17

## Context

The project must be usable from OpenAI/ChatGPT, Anthropic/Claude, and generic agent harnesses without building model-provider logic into the state engine.

MCP has a current 2026-07-28 specification and official SDKs. The official Go SDK documents support for the 2026-07-28 line and older protocol versions.

ChatGPT Developer Mode supports remote MCP read/write tools. Anthropic's MCP connector supports remote MCP tool calls, while its documented connector feature set is narrower than the full MCP specification.

## Decision

1. MCP is the only provider-facing protocol required by the v0.1 core.
2. Runtime implementation targets MCP 2026-07-28 semantics through the official Go SDK.
3. v0.1 product scope is **tools-first**. Resources/prompts/tasks are not part of the cross-provider minimum contract.
4. Provider-specific smoke tests live outside the deterministic state core.
5. Backward protocol support is accepted only where the official SDK provides a tested compatibility path.
6. We do not implement a second private OpenAI- or Anthropic-specific tool protocol in the core.

## Consequences

Positive:

- one simulated tool surface can be reused by multiple hosts;
- less vendor-specific code;
- protocol behavior inherits official SDK maintenance;
- easier conformance testing.

Negative:

- host/model tool-selection behavior will differ;
- feature support is limited to the common subset in early releases;
- protocol evolution remains an external compatibility dependency.

## Non-claim

Provider-neutral protocol compatibility is not equivalent to provider-neutral model behavior.

## References

- https://modelcontextprotocol.io/specification/2026-07-28
- https://github.com/modelcontextprotocol/go-sdk
- https://developers.openai.com/api/docs/guides/developer-mode
- https://platform.claude.com/docs/en/agents-and-tools/mcp-connector