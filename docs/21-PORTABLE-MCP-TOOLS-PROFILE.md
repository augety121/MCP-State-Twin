# Portable MCP Tools Profile

> **Status:** Proposal / Unverified  
> **Purpose:** Define the narrow MCP feature profile most likely to interoperate across heterogeneous current agent hosts.

## 1. Design objective

MCP State Twin should expose a **portable tools-first profile** for live agent evaluation.

This profile is deliberately narrower than the full MCP feature space.

## 2. Required baseline

A portable episode requires:

```text
tools/list-equivalent host visibility
tools/call
JSON Schema input contract
typed/structured success and failure
stable canonical tool identity
terminal state scoring
```

## 3. Optional—not baseline

The portable profile MUST NOT require:

- prompts;
- resources;
- roots;
- elicitation;
- Tasks;
- MRTR;
- MCP Apps/UI;
- provider-specific OAuth extensions;
- host-specific file sandboxes.

They may be negotiated as optional capabilities.

## 4. Tool declaration constraints

### ST-PORT-R001
Canonical tool names MUST obey the selected MCP protocol profile.

### ST-PORT-R002
Canonical tool names SHOULD be conservative enough to survive host prefixing/sanitization where possible.

### ST-PORT-R003
Canonical tool identity MUST not be replaced by the projected host name.

### ST-PORT-R004
Descriptions MUST not contain hidden evaluator answers, expected terminal state, control-plane URLs or credentials.

## 5. Input schema

### ST-PORT-R010
TwinSpec stores the canonical input schema.

### ST-PORT-R011
Host adapters may project/sanitize schema only when required by the host.

### ST-PORT-R012
Projection must preserve an auditable transform list.

### ST-PORT-R013
If a schema feature is unsupported and cannot be losslessly represented, compatibility status is `semantic-risk` or `unsupported`, not silently PASS.

## 6. Output schema and content

### ST-PORT-R020
Canonical tool output MUST be validated against its declared output contract when one exists.

### ST-PORT-R021
For State-Twin-native portable tools, structured JSON output SHOULD be accompanied by a text JSON representation when the selected MCP compatibility profile recommends this for older/heterogeneous clients.

### ST-PORT-R022
Strict upstream-fidelity mode MUST NOT invent extra user-visible result content merely to satisfy a host.

In such a case, a host adapter may normalize observation externally, but server semantics remain upstream-faithful.

## 7. Error model

The portable profile distinguishes:

```text
modeled domain error
protocol/transport error
authorization error
host error
infrastructure error
```

A model-visible tool failure should be intentional and typed.

## 8. Idempotency

Where a modeled upstream has idempotency semantics:
- TwinSpec expresses them explicitly;
- ambiguous-response tests evaluate safe retry;
- host automatic retry is recorded separately.

No universal idempotency behavior is invented.

## 9. Tool-set minimization

A scenario bundle SHOULD expose only tools needed by the task family plus any explicitly declared distractors.

Reasons:
- some hosts impose tool-count limits;
- many tools consume context;
- tool-name collisions become more likely;
- evaluation of selection quality should be deliberate.

Tool minimization is not hidden answer leakage; it must be declared in the episode profile.

## 10. Distractor profile

To evaluate tool selection, a scenario may declare:

```yaml
tools:
  required: [...]
  relevantOptional: [...]
  distractors: [...]
```

Distractor design MUST avoid revealing expected action through naming.

## 11. Surface quiescence

Before a blind live-host run begins, the harness MUST establish a declared surface-ready condition.

Examples:
- target MCP server connected;
- expected required tools visible;
- no unresolved tool-load error;
- authorization profile settled.

If a host progressively loads tools and the required surface is not ready, the run is `HOST_NOT_READY`, not agent failure.

## 12. List-change behavior

If a host supports dynamic tool-list notifications:
- the event is recorded;
- a running episode defines whether surface changes are allowed.

Default blind-eval policy:
- no undeclared mid-episode business-tool surface change.

## 13. Built-in tool collisions

Coding agents may also have shell, filesystem, web or editor tools.

Portable MCP profile alone does not guarantee isolation.

Host Isolation Profile determines whether those built-ins are:
- disabled;
- allowed and recorded;
- uncontrolled.

## 14. Authentication

Authentication details are host/profile-specific.

Portable tool semantics MUST not embed credentials into:
- tool arguments;
- model-visible descriptions;
- evidence payloads.

## 15. Large outputs

If a host truncates or transforms results:
- server raw result digest remains canonical world evidence;
- host-visible representation is recorded separately;
- task scoring should depend on world state unless the scenario explicitly tests result consumption.

## 16. Capability negotiation

The adapter should produce a resolved profile such as:

```yaml
portableTools:
  call: supported
  structuredContent: supported
  outputSchema: projected
  prompts: not_required
  resources: not_required
  tasks: not_required
  mrtr: not_required
```

## 17. Why tools-first

This profile intentionally targets the feature subset shared most broadly across current agent hosts.

It is not a statement that tools are the only useful MCP capability.

It is the project's portability baseline.
