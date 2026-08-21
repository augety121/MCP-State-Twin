# Source Registry — MCP State Twin Lifecycle SPEC Pack

> **Status:** Research registry  
> **Research cut:** 2026-08-18  
> **Purpose:** Record the primary sources used to justify architecture and feasibility choices.  
> **Important:** A source proves what the external protocol/product/documentation says. It does **not** prove MCP State Twin has implemented or passed that behavior.

## Source policy

1. Prefer first-party specifications, official repositories and official vendor documentation.
2. Time-sensitive sources MUST be revalidated before implementation/release.
3. Implementation should pin an exact version/commit whenever executable behavior depends on it.
4. Provider product documentation is evidence of documented capability, not evidence that MCP State Twin is compatible.
5. Project-current facts in this pack are only **source-reported baseline** until the repository is inventoried and tests are re-run.

---

## MCP

### SRC-MCP-2026-SPEC

- **Source:** Model Context Protocol specification, revision `2026-07-28`
- **URL:** https://modelcontextprotocol.io/specification/2026-07-28
- **Schema repository:** https://github.com/modelcontextprotocol/modelcontextprotocol/tree/main/schema/2026-07-28
- **Checked:** 2026-08-18
- **Used for:** protocol-version baseline and normative rebaseline work.

### SRC-MCP-2026-RELEASE

- **Source:** official MCP `2026-07-28` release material
- **URL:** https://blog.modelcontextprotocol.io/posts/2026-07-28/
- **Checked:** 2026-08-18
- **Used for:** lifecycle changes, stateless model, MRTR, routing, caching, extensions/deprecations and authorization direction.

### SRC-MCP-DISCOVER

- **Source:** official `server/discover` specification
- **URL:** https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/server/discover.mdx
- **Checked:** 2026-08-18
- **Important nuance:** a server implementing the 2026 revision must implement discovery, while a client may send another RPC without calling discovery first.

### SRC-MCP-GO-SDK

- **Source:** official MCP Go SDK
- **URLs:**
  - https://github.com/modelcontextprotocol/go-sdk
  - https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/protocol.md
  - https://github.com/modelcontextprotocol/go-sdk/releases
- **Checked:** 2026-08-18
- **Used for:** feasibility of modern/legacy lifecycle support and Streamable HTTP implementation.
- **Caveat:** official SDK use is not itself proof of full conformance. State Twin must pin SDK version and maintain independent tests/conformance evidence.

### SRC-MCP-CONFORMANCE

- **Source:** official MCP conformance project
- **URL:** https://github.com/modelcontextprotocol/conformance
- **Checked:** 2026-08-18
- **Used for:** versioned conformance strategy and `2026-07-28` test profile.
- **Project policy derived from this:** record exact conformance version/commit, protocol revision, suite/scenarios, expected failures and raw output.

### SRC-MCP-AUTH

- **Source:** official MCP authorization specification
- **URL:** https://modelcontextprotocol.io/specification/draft/basic/authorization
- **Checked:** 2026-08-18
- **Used for:** OAuth protected-resource/audience semantics and no-token-passthrough boundary.
- **Freshness:** authorization is security-sensitive; re-check the exact accepted/draft revision at implementation time.

### SRC-MCP-TOOLS

- **Source:** official MCP tools specification
- **URL:** https://modelcontextprotocol.io/specification/draft/server/tools
- **Checked:** 2026-08-18
- **Used for:** tool contracts, annotations trust, schema behavior, human approval guidance and tool-surface considerations.
- **Freshness:** re-check exact dated version before writing protocol-wire tests.

### SRC-MCP-TASKS

- **Source:** official MCP Tasks extension
- **URLs:**
  - https://modelcontextprotocol.io/extensions/tasks/overview
  - https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks
- **Checked:** 2026-08-18
- **Used for:** forward design of long-running operations.
- **Project policy:** Tasks remain optional and must not become a v0.1 core assumption.

---

## OpenAI / Codex

### SRC-OPENAI-DEVELOPERS

- **Source:** OpenAI Developers
- **URL:** https://developers.openai.com/
- **Checked:** 2026-08-18
- **Used for:** current Codex/OpenAI developer product surface and MCP integration direction.

### SRC-OPENAI-MCP-API

- **Source:** official OpenAI API documentation
- **URLs:**
  - https://platform.openai.com/docs/api-reference/
  - https://platform.openai.com/docs/quickstart/make-your-first-api-request
- **Checked:** 2026-08-18
- **Used for:** remote MCP tool configuration, tool filtering, approvals and observable MCP list/call objects/events.
- **Policy:** OpenAI API MCP testing is a separate Host Profile from Codex.

### SRC-OPENAI-MODEL-GUIDANCE

- **Source:** official OpenAI model guidance
- **URL:** https://developers.openai.com/api/docs/guides/latest-model
- **Checked:** 2026-08-18
- **Used for:** forward design assumptions that modern agents may be persistent/proactive, execute multi-step workflows and require explicit autonomy/approval boundaries.
- **Project implication:** never assume one model request/turn maps to exactly one tool call.

---

## Anthropic / Claude

### SRC-ANTHROPIC-MESSAGES-MCP

- **Source:** Anthropic Messages API MCP connector documentation
- **URL:** https://platform.claude.com/docs/en/agents-and-tools/mcp-connector
- **Checked:** 2026-08-18
- **Used for:** remote MCP connector profile, supported subset, tool filtering and OAuth considerations.
- **Policy:** only the exact connector/profile capabilities observed at test time may be claimed.

### SRC-CLAUDE-CODE-MCP

- **Source:** Claude Code MCP documentation
- **URL:** https://code.claude.com/docs/en/mcp
- **Checked:** 2026-08-18
- **Used for:** local/remote MCP transport feasibility, configuration scopes and host-profile design.

### SRC-CLAUDE-CODE-PERMISSIONS

- **Source:** Claude Code permission documentation
- **URL:** https://code.claude.com/docs/en/permissions
- **Checked:** 2026-08-18
- **Used for:** host approval/allow/ask/deny evidence.

### SRC-CLAUDE-MANAGED-AGENTS

- **Source:** Anthropic Managed Agents documentation
- **URLs:**
  - https://platform.claude.com/docs/en/managed-agents/mcp-connector
  - https://platform.claude.com/docs/en/managed-agents/permission-policies
  - https://platform.claude.com/docs/en/managed-agents/tools
- **Checked:** 2026-08-18
- **Used for:** separate managed-agent Host Profile, explicit MCP permission policies and host transformation of large tool output.
- **Policy:** raw server output and host-visible transformed output are separate evidence fields.

---

## Runtime primitives

### SRC-SQLITE-ISOLATION

- **Source:** SQLite isolation documentation
- **URL:** https://www.sqlite.org/isolation.html
- **Used for:** transaction isolation and serialized-write feasibility.

### SRC-SQLITE-WAL

- **Source:** SQLite WAL documentation
- **URL:** https://www.sqlite.org/wal.html
- **Used for:** reader/writer behavior, one-writer limitation, same-host constraint and checkpoint operational considerations.

### SRC-JSON-SCHEMA-2020-12

- **Source:** JSON Schema Draft 2020-12
- **URL:** https://json-schema.org/draft/2020-12
- **Used for:** strict tool input/output contract feasibility.

### SRC-CEL

- **Source:** CEL specification / cel-go
- **URLs:**
  - https://github.com/cel-expr/cel-spec
  - https://github.com/cel-expr/cel-go
- **Checked:** 2026-08-18
- **Used for:** bounded expression feasibility; CEL is designed as non-Turing-complete and mutation-free, while the host controls available data/functions.

### SRC-JCS

- **Source:** RFC 8785 — JSON Canonicalization Scheme
- **URL:** https://www.rfc-editor.org/rfc/rfc8785.html
- **Used for:** canonicalization research only.
- **Important:** this pack does NOT silently replace State Twin's existing alpha canonicalization. Any migration needs a new canonicalization ID and compatibility plan.

### SRC-OTEL-GENAI

- **Source:** OpenTelemetry GenAI semantic conventions
- **URL:** https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/
- **Checked:** 2026-08-18
- **Used for:** optional observability-export design and sensitive-content warning.
- **Project policy:** OTel is not the canonical evidence source.

---

## Software supply chain — later lifecycle

### SRC-SLSA

- **Source:** SLSA Build Track v1.2
- **URL:** https://slsa.dev/spec/v1.2/build-track-basics
- **Checked:** 2026-08-18
- **Used for:** future build-provenance maturity.

### SRC-SPDX

- **Source:** SPDX specifications
- **URL:** https://spdx.dev/use/specifications/
- **Checked:** 2026-08-18
- **Used for:** future SBOM/release artifact metadata.

### SRC-SIGSTORE

- **Source:** Sigstore/Cosign documentation
- **URLs:**
  - https://docs.sigstore.dev/cosign/signing/signing_with_blobs/
  - https://docs.sigstore.dev/cosign/verifying/verify/
- **Checked:** 2026-08-18
- **Used for:** optional future artifact/bundle signing.
- **Important:** signature provenance does not prove semantic Twin fidelity.

---

## MCP State Twin project-local baseline

### SRC-STATE-TWIN-README

The baseline supplied by the project owner before this lifecycle proposal reports:

- development preview `0.1.0-dev`;
- no tagged stable release;
- TwinSpec `v1alpha1`;
- strict YAML/structural validation;
- hermetic JSON Schema 2020-12 tool contracts;
- canonical SHA-256 digests for spec/surface/state;
- bounded CEL;
- SQLite atomic transitions and audit;
- immutable snapshots/forks/reset/diff;
- Streamable HTTP MCP data plane;
- separate authenticated control plane;
- scenario runner;
- synthetic issue-tracker reference twin, L1/unverified/unbound.

It also reports as not implemented or not verified:
- virtual-clock advancement;
- deterministic fault injection;
- recorder/cassette/redaction;
- automatic upstream inspection/refresh;
- live Codex/OpenAI/Claude/Claude Code tests;
- provider harness;
- differential validation/L2 workflow;
- data-plane auth/TLS;
- remote multi-tenancy;
- formal security audit.

**This pack does not independently re-run those local tests.** Before implementation, replace this source-reported baseline with a current repository evidence inventory.

---

# Source refresh rules

Before any release candidate:

1. Re-open MCP dated specification.
2. Re-open official Go SDK release/protocol docs.
3. Re-run pinned official conformance.
4. Re-open each host/provider's official MCP documentation used by a compatibility profile.
5. Record host/product/model versions.
6. Re-open security-sensitive auth requirements.
7. Update this registry's checked date only for sources actually reviewed.

A stale source must never silently remain the basis of a current compatibility claim.
---

# Universal Agent Compatibility Research — 2026-08-18

> These entries support the Universal Agent Compatibility architecture. They document host behavior only. They do not prove State Twin interoperability.

## SRC-CURSOR-MCP

- **Source:** Cursor official Model Context Protocol documentation
- **URL:** https://docs.cursor.com/context/model-context-protocol
- **Checked:** 2026-08-18
- **Documented:** Cursor supports MCP tools, prompts, roots and elicitation, with stdio, SSE and Streamable HTTP transports; OAuth is documented for network transports.
- **Design implication:** Cursor gets its own Host Profile; richer feature support must not become a universal baseline.

## SRC-GEMINI-CLI-MCP

- **Source:** Gemini CLI official MCP server documentation
- **URL:** https://geminicli.com/docs/tools/mcp-server/
- **Checked:** 2026-08-18
- **Documented:** local/remote MCP discovery over stdio/SSE/Streamable HTTP; tools/resources; tool filtering; confirmation policy; rich content; prompts/instructions.
- **Critical projection behavior:**
  - every discovered MCP tool is given an FQN of `mcp_<serverName>_<toolName>`;
  - names are sanitized/truncated for Gemini compatibility;
  - `$schema` and `additionalProperties` are stripped from tool parameter schemas;
  - certain `anyOf/default` combinations are rewritten.
- **Design implication:** host-visible surface may differ from canonical server surface; projection evidence is mandatory.

## SRC-GITHUB-COPILOT-IDE-MCP

- **Source:** GitHub Docs — extending Copilot Chat with MCP
- **URL:** https://docs.github.com/copilot/customizing-copilot/using-model-context-protocol/extending-copilot-chat-with-mcp
- **Checked:** 2026-08-18
- **Documented:** Copilot IDE integrations can discover/use MCP tools; current IDE documentation also describes prompts/resources in supported environments.
- **Design implication:** IDE Copilot is a separate Host Profile from cloud coding agent.

## SRC-GITHUB-COPILOT-CLOUD-MCP

- **Source:** GitHub Docs — repository MCP configuration for Copilot cloud agent/code review
- **URL:** https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/configure-mcp-servers
- **Checked:** 2026-08-18
- **Documented at this research cut:**
  - cloud agent/code review support MCP tools, but not MCP resources/prompts;
  - configured tools may be used autonomously without per-call user approval;
  - remote MCP OAuth is not currently supported in that cloud path;
  - local/stdio/http/sse configuration types are accepted.
- **Design implication:** Copilot cloud agent is not compatibility-equivalent to Copilot IDE/CLI.

## SRC-WINDSURF-MCP

- **Source:** Windsurf official Cascade MCP documentation
- **URL:** https://docs.windsurf.com/windsurf/cascade/mcp
- **Checked:** 2026-08-18
- **Documented:** stdio, Streamable HTTP and SSE; OAuth; MCP tool enable/disable.
- **Important host limit:** official localized documentation states Cascade has a total limit of 100 tools accessible at a time.
- **Design implication:** compatibility lint must include host tool-count budgets; large canonical surfaces should be scenario-filtered rather than globally weakened.

## SRC-CLINE-MCP

- **Sources:** Cline official documentation
- **URLs:**
  - https://docs.cline.bot/mcp/mcp-overview
  - https://docs.cline.bot/features/auto-approve
  - https://docs.cline.bot/usage/cli-overview
- **Checked:** 2026-08-18
- **Documented:** MCP capabilities integrated with Cline; auto-approval/Yolo policies can allow MCP and built-in tools to execute without per-call confirmation; CLI supports automation/headless usage.
- **Design implication:** approval/isolation policy is mandatory evidence.

## SRC-AMAZON-Q-MCP

- **Source:** Amazon Q Developer official MCP documentation
- **URLs:**
  - https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/qdev-mcp.html
  - https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/mcp-ide.html
- **Checked:** 2026-08-18
- **Documented:** MCP tools and prompts; local/remote configuration; per-tool permission levels including Ask/Always allow/Deny in IDE; CLI documentation exposes MCP tools alongside built-ins.
- **Design implication:** host built-ins and permission policy belong in evaluation identity.

## SRC-JETBRAINS-MCP

- **Source:** JetBrains AI Assistant official MCP documentation
- **URL:** https://www.jetbrains.com/help/ai-assistant/mcp.html
- **Checked:** 2026-08-18
- **Documented:** STDIO, Streamable HTTP and legacy SSE support.
- **Design implication:** JetBrains AI Assistant is a distinct Host Profile.

## SRC-JUNIE-MCP

- **Source:** JetBrains Junie official documentation
- **URL:** https://www.jetbrains.com/help/ai-assistant/junie-agent.html
- **Checked:** 2026-08-18
- **Documented:** Junie can use configured external MCP tools; persistent project instructions may come from `AGENTS.md`; debugger MCP tooling exists in supported JetBrains environments.
- **Design implication:** instruction/workspace identity and IDE-provided MCP tools must be captured.

## SRC-ZED-MCP

- **Source:** Zed official MCP documentation
- **URL:** https://zed.dev/docs/ai/mcp
- **Checked:** 2026-08-18
- **Documented:** Zed supports MCP Tools and Prompts and reacts to tool-list change notifications; external agents can receive Zed-configured MCP servers over ACP or use native MCP configuration.
- **Design implication:** surface readiness/dynamic-list behavior and ACP forwarding path belong in evidence.

## SRC-ZED-EXTERNAL-AGENTS

- **Source:** Zed official external agent documentation
- **URL:** https://zed.dev/docs/ai/external-agents
- **Checked:** 2026-08-18
- **Documented:** external agents include Claude, Codex, Gemini CLI, OpenCode, Copilot and Cursor via ACP integrations; agent auth/model/tools/native configuration remain separate from Zed's own model settings.
- **Design implication:** editor-agent path and agent-native MCP path must remain separate identity dimensions.

## SRC-OPENCODE-MCP

- **Source:** OpenCode official MCP documentation
- **URL:** https://opencode.ai/v2/docs/mcp-servers
- **Checked:** 2026-08-18
- **Documented:** local stdio and remote Streamable HTTP MCP; prompts/instructions in current v2 docs; MCP tools may be exposed through Code Mode or directly; MCP tool sets consume context.
- **Design implication:** tool-context budget and projection mode belong in Host Profile.

## SRC-CLAUDE-CODE-MCP-2026

- **Source:** Claude Code official MCP documentation
- **URL:** https://code.claude.com/docs/en/mcp
- **Checked:** 2026-08-18
- **Documented:** remote HTTP is recommended for remote MCP; local stdio is supported; SSE is deprecated; OAuth support exists for HTTP servers.
- **Design implication:** Claude Code HTTP and stdio are separate transport profiles when necessary.

## SRC-ANTHROPIC-MESSAGES-MCP-2026

- **Source:** Anthropic Messages API MCP connector
- **URL:** https://platform.claude.com/docs/en/agents-and-tools/mcp-connector
- **Checked:** 2026-08-18
- **Documented at this research cut:**
  - current beta header `mcp-client-2025-11-20`;
  - tool calls are supported;
  - only the tool-call portion of MCP is currently supported;
  - remote public HTTP is required; Streamable HTTP/SSE documented;
  - local stdio cannot be directly connected;
  - allowlist/denylist/per-tool configuration and OAuth bearer tokens are supported.
- **Design implication:** this is a distinct remote Host Profile and must not inherit Claude Code compatibility.

## SRC-ANTHROPIC-MANAGED-AGENTS-MCP-2026

- **Sources:** Anthropic Managed Agents MCP and permissions docs
- **URLs:**
  - https://platform.claude.com/docs/en/managed-agents/mcp-connector
  - https://platform.claude.com/docs/en/managed-agents/permission-policies
  - https://platform.claude.com/docs/en/managed-agents/tools
- **Checked:** 2026-08-18
- **Documented:** MCP toolsets have explicit permission policies; current MCP toolset default is confirmation-oriented (`always_ask`); very large MCP output can be written to a sandbox file while the model receives a truncated preview.
- **Design implication:** raw server result and host-visible result are separate evidence.

## SRC-MCP-TOOLS-PORTABILITY

- **Source:** official MCP Tools specification
- **URL:** https://modelcontextprotocol.io/specification/draft/server/tools
- **Checked:** 2026-08-18
- **Relevant design fact:** structured tool output has an output schema contract; for backwards compatibility, the official spec recommends also returning serialized JSON in a TextContent block when structured content is returned.
- **Project implication:** State-Twin-native portable tools may use dual structured/text representation, while strict upstream-fidelity mode must preserve the bound upstream behavior.

## SRC-ACP

- **Source:** Agent Client Protocol official documentation
- **URLs:**
  - https://agentclientprotocol.com/get-started/introduction
  - https://agentclientprotocol.com/get-started/architecture
- **Checked:** 2026-08-18
- **Documented:** ACP standardizes editor/client-to-coding-agent communication, is MCP-friendly and reuses MCP representations where practical; editors may forward MCP configuration to agents.
- **Project implication:** ACP is an integration path in front of an agent, not a replacement for State Twin's MCP world interface.

## SRC-ACP-V2

- **Source:** ACP v2 proposal/RFD collection
- **URL:** https://agentclientprotocol.com/rfds/v2/overview
- **Checked:** 2026-08-18
- **Status:** proposal/draft work.
- **Project implication:** do not vendor ACP draft semantics into State Twin core.

## SRC-A2A-1

- **Source:** official Agent2Agent project/specification
- **URLs:**
  - https://a2a-protocol.org/latest/specification
  - https://github.com/a2aproject/A2A
  - https://github.com/a2aproject/A2A/blob/main/docs/topics/a2a-and-mcp.md
- **Checked:** 2026-08-18
- **Documented:** A2A focuses on agent-to-agent collaboration while MCP focuses on agent-to-tool/resource interaction; the protocols are complementary. The published A2A specification line is 1.0 at this research cut.
- **Project implication:** future multi-agent delegation may use A2A around agents while each agent continues to use State Twin through MCP. State Twin does not become an A2A agent by default.
