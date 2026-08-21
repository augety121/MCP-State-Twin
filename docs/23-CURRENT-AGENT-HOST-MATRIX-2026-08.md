# Current Agent Host Matrix — Research Baseline 2026-08

> **Status:** Research / Documented Capability Baseline  
> **State Twin verification:** NONE implied  
> **Research cut:** 2026-08-18  
> **Rule:** Every row must be revalidated against first-party documentation and then tested before becoming a State Twin compatibility claim.

## 1. Purpose

This matrix informs adapter design.

It does not say these hosts have been tested with MCP State Twin.

## 2. Matrix

| Host / Product | MCP tools | Other MCP features documented | Transport notes | Important host behavior / limitation | State Twin status |
|---|---|---|---|---|---|
| Codex | documented MCP integration | host/product dependent | current official profile must be rechecked at adapter implementation | approvals/autonomy and tool-heavy workflows are host-profile data | documented only |
| OpenAI Responses/API | remote MCP tools | remote tool integration | remote server | allowed-tool filtering and approval policy are relevant | documented only |
| Claude Code | tools; broader MCP client support | host-specific | HTTP and local stdio documented | project/config permission behavior is part of profile | documented only |
| Anthropic Messages MCP connector | tool calls | connector is a subset, not whole MCP | remote HTTP | connector beta/profile and tool allow/deny matter | documented only |
| Anthropic Managed Agents | MCP toolsets | managed-agent feature set | managed runtime | permission policy and output transforms matter | documented only |
| Gemini CLI | tools and resources documented | server prompts/instructions and rich content are documented | stdio/HTTP/SSE documented | prefixes/sanitizes tool names and sanitizes schemas | documented only |
| GitHub Copilot in IDE/CLI | MCP tools; IDE environments may support prompts/resources | product-specific | product-specific | permission/approval model differs by product | documented only |
| GitHub coding agent / code review | MCP tools | cloud path currently tools-only in official docs | remote/configured environment | autonomous use; no per-call user approval; remote OAuth limitations documented | documented only |
| Cursor | tools | prompts, roots, elicitation documented | stdio/SSE/Streamable HTTP | OAuth and host permissions/profile must be recorded | documented only |
| Windsurf Cascade | tools | prompts/resources documented | stdio/HTTP/SSE | accessible tool-count cap documented; tool budget matters | documented only |
| Cline | tools | host feature set | stdio/Streamable HTTP; legacy SSE | `autoApprove` policy can materially change safety/evidence | documented only |
| Amazon Q Developer | tools | prompts documented | local + remote HTTP | MCP tools may be progressively loaded; permission state matters | documented only |
| JetBrains AI Assistant / Junie | external MCP tools | product-specific | stdio/Streamable HTTP/legacy SSE | IDE/agent topology and product version are part of evidence | documented only |
| Zed agent | tools, prompts | agent/editor integration | native MCP + external-agent forwarding | ACP can forward configured MCP servers to external agents | documented only |
| OpenCode | MCP tools | product-specific | local + remote MCP | many tools consume context; tool-set minimization matters | documented only |
| Custom MCP client | depends on client | depends | depends | requires explicit probe/profile | unknown until tested |

## 3. Cross-host conclusions

### 3.1 Tools are the portability baseline

The broadest practical common denominator is MCP **tools**.

Therefore State Twin's universal baseline should remain tools-first.

### 3.2 Resources/prompts are optional

Some hosts support them, some host products do not expose them in all modes.

They must not become required for baseline evaluation.

### 3.3 Host projection is real

Different hosts can alter:
- tool names;
- schemas;
- output representation;
- available tool set;
- approval behavior.

Therefore server conformance alone cannot prove model-visible contract equality.

### 3.4 Product-level separation

Provider name is insufficient.

Examples of separate profiles:
- Claude Code vs Messages connector vs Managed Agents;
- Copilot IDE/CLI vs cloud coding agent;
- Codex vs OpenAI API remote MCP.

## 4. Required implementation follow-up

Before adding a `verified` row:

1. capture exact official source/date;
2. pin/record host version;
3. generate Host Profile;
4. run compatibility lint;
5. test connection;
6. test projection;
7. run stateful scenario;
8. archive evidence.

## 5. Why this matrix is intentionally conservative

Host products evolve quickly.

The matrix is useful as a **research baseline**, not as a permanent compatibility promise.

The evidence registry must supersede this document once real tests exist.
