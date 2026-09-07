# Compatibility Matrix

**Status:** current evidence summary; not a universal compatibility claim
**Last reviewed:** 2026-08-31

| Profile | Protocol/API surface | State | Current evidence | Missing evidence |
|---|---|---|---|---|
| `responses-functions-offline-v1` | synthetic non-streaming Responses function codec + local SDK MCP | experimental offline only | agenthost/agenteval/CLI tests; SPEC-0039 | actual API transport, account/model availability, approved live run; not a product-host profile |
| `generic-mcp-modern-2026-07-28` | stateless tools-first Streamable HTTP | experimental | raw wire tests for discovery/list/call/result/header rules | complete optional-feature conformance |
| `generic-mcp-legacy-2025-11-25` | initialize + tools-first Streamable HTTP | experimental | pinned SDK/conformance subset and direct handshake tests | broader client matrix |
| `openai-responses-remote-mcp-v1alpha1` | Responses create/retrieve/cancel + remote MCP | unverified | mock contract tests and opt-in harness | SPEC-0019 live report through SPEC-0020 staging |
| `anthropic-messages-mcp-v1alpha1` | Messages API MCP connector beta | unverified | mock contract tests and opt-in harness | SPEC-0019 live report through SPEC-0020 staging |
| `chatgpt-mcp` | ChatGPT product integration | unverified | official-document research only | exact product profile and live evidence |
| `codex-mcp` | Codex product/agent integration | unverified | generic MCP architecture only | exact product profile and live evidence |
| `claude-product-mcp` | Claude web/desktop product | unverified | official-document research only | exact product profile and live evidence |
| `claude-code-mcp` | Claude Code | unverified | generic MCP architecture only | exact versioned host run and evidence |

An API-family result MUST NOT update a product-profile row. Evidence expiration
and state transitions follow SPEC-0019 and SPEC-0021.
