# SPEC-0003: MCP Boundaries and Compatibility

- **Status:** Proposed normative specification
- **Scope:** tools-first MCP integration

## 1. Data plane

The data plane exposes only TwinSpec business tools through the supported MCP
transport. `tools/list` MUST NOT contain snapshot, fork, reset, inspect, diff,
fault, clock, or hidden-oracle controls. Branch identity is taken from trusted
server context, not from model-supplied arguments.

The data plane is local/CI oriented in v0.1. It has no production-grade remote
authentication or TLS claim. Deployments outside loopback require an external
network policy and must be documented as untrusted unless a later security
profile is accepted.

## 2. Control plane

The control plane is a separate endpoint and authorization boundary. It owns
snapshot, fork, reset, state inspection, diff, and future fault/clock actions.
It MUST NOT share the agent-facing MCP tool registry. Every mutation MUST append
an atomic control audit record. Tokens MUST NOT enter logs or persisted audit
payloads.

## 3. Hermetic mode

Hermetic mode MUST contain no upstream client or passthrough write path and
MUST be runnable in a network namespace with only loopback. Failure to model a
behavior results in an explicit error; it never triggers network fallback.

## 4. Provider compatibility

Compatibility means protocol/tool usability, not model behavior equivalence.
The supported matrix records host, provider, model, MCP protocol version,
transport, tools/list, tools/call, schemas, errors, and test date.

ChatGPT, OpenAI API, Claude, Claude Code, and other hosts may choose different
tool trajectories. The core MUST remain provider-neutral and MUST NOT import a
provider SDK to execute state transitions.

The v0.1 promise is tools-first MCP interoperability. Resources, prompts,
tasks, A2A, background wakeups, and host-specific automations require separate
compatibility evidence and are not implied by this document.

## 5. Surface binding

The canonical surface envelope and fail-closed `current/drifted/unknown/unbound`
rules are defined in ADR-0008. A local digest does not prove that an upstream
service was inspected; an upstream inspector remains an extension.
