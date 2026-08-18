# SPEC-0006: Host Compatibility and Model Evaluation

- **Status:** Proposed normative specification
- **Scope:** MCP host interoperability and cross-model evaluation evidence
- **Related:** SPEC-0003, SPEC-0004, SPEC-0005, ADR-0002, ADR-0008

## 1. Purpose

This specification defines what MCP State Twin may claim when it is used by
ChatGPT, an OpenAI API client, Claude, Claude Code, or another MCP host. It also
defines the evidence required to compare model-driven tool trajectories.

The project virtualizes the external tool world. It does not virtualize a
model, provider control plane, conversation service, or host permission system.
Accordingly, three claims MUST remain separate:

1. **runtime correctness:** the twin executes its declared state transitions;
2. **protocol compatibility:** a named host can discover and call the exposed
   MCP tools under a recorded configuration;
3. **model evaluation:** a named model can attempt a bounded scenario and is
   scored from environment state and declared assertions.

Evidence for one claim MUST NOT be used as evidence for another.

## 2. Compatibility profiles

Every compatibility result MUST name exactly one profile.

| Profile | Purpose | Minimum evidence |
|---|---|---|
| `generic-mcp` | Automated protocol baseline independent of a model provider | initialize, negotiated protocol, ping, tools/list, tools/call, structured error |
| `openai-api-mcp` | OpenAI API client using an MCP-capable tool path | provider request ID, model identifier, observed tool calls, terminal state |
| `chatgpt-mcp` | Interactive ChatGPT host integration | host surface/version evidence, tool discovery, one read and one state-changing twin call |
| `anthropic-api-mcp` | Anthropic API client using an MCP-capable tool path | provider request ID, model identifier, observed tool calls, terminal state |
| `claude-code-mcp` | Claude Code configured with the twin MCP server | CLI version, MCP configuration digest, allowed/disallowed tool policy, terminal state |
| `custom-mcp` | Any other host or harness | host name/version plus the `generic-mcp` protocol evidence |

A profile name describes the tested integration path, not all products from a
provider. For example, `openai-api-mcp` evidence does not establish ChatGPT
compatibility, and `claude-code-mcp` evidence does not establish every Claude
API integration path.

## 3. Support levels

Each `(profile, host version, protocol version, transport)` tuple MUST have one
of these levels:

- `unverified`: no current reproducible evidence;
- `experimental`: a dated smoke run passed, but the tuple is not a release
  gate and may depend on preview host behavior;
- `verified`: the required suite passed against an immutable runtime revision
  and the evidence is published;
- `regressed`: previously verified evidence now fails;
- `unsupported`: the project intentionally does not support the tuple.

`verified` MUST include an expiry or revalidation policy. A provider or host
version change creates a new tuple; it MUST NOT inherit the old status without
another run. `latest`, `auto`, or an unversioned model alias is insufficient as
the sole identity in durable evidence.

## 4. Trust and deployment boundary

### 4.1 Local hosts

A local host MAY connect to the loopback Streamable HTTP endpoint. The runtime
MUST preserve branch identity in trusted server routing context and MUST NOT
add branch, snapshot, expected answer, or fault controls to tool arguments.

### 4.2 Provider-hosted clients

A provider-hosted client cannot be treated as local. Connecting it to a twin
requires a separately reviewed remote deployment profile covering, at minimum:

- TLS and server identity;
- data-plane authentication and credential rotation;
- per-run branch authorization;
- request-body and concurrency limits;
- tenant and run isolation;
- ingress allowlisting where the provider publishes stable controls;
- audit retention and deletion;
- denial of every control-plane route.

The v0.1 local preview has no such profile. Therefore it MUST NOT claim live
ChatGPT or provider-hosted Claude support merely because it implements MCP.
Temporary tunnels are not release evidence unless their authentication,
exposure, retention, and operator procedure are documented in the report.

### 4.3 Secrets and provider data

Provider API keys, OAuth tokens, authorization headers, cookies, conversation
contents, and account identifiers MUST NOT enter TwinSpec, fixtures, scenario
files, ordered traces, committed reports, or CI logs. A report MAY store an
irreversible run identifier or redacted provider request identifier.

Live-provider tests MUST use synthetic fixtures and prompts. Test accounts MUST
have no production authority. Provider terms and data-retention settings remain
operator responsibilities and MUST be documented outside the runtime claim.

## 5. Tool-surface admission

Before a host run, the harness MUST record:

1. runtime revision and semantic version;
2. TwinSpec digest and canonical MCP surface digest;
3. initial snapshot digest and branch identifier;
4. configured and negotiated MCP protocol versions;
5. transport and endpoint trust class (`loopback`, `private`, or `public`);
6. host/provider/model identities;
7. the tool names and schemas observed by the host when observable.

The canonical surface remains the authority defined by ADR-0008. If the host
or an intermediary rewrites names, descriptions, schemas, annotations, or
error envelopes, the report MUST record the observed difference. The run MUST
be marked `surface_modified`; it MUST NOT be used as same-surface comparison
evidence unless the modification is reviewed and explicitly allowed.

Host-generated wrapper tools and provider-specific approval prompts are not
part of TwinSpec and MUST NOT change transition semantics.

## 6. Harness isolation

Each trial MUST use a fresh branch forked from the declared immutable snapshot.
Trials MUST NOT share mutable twin state, model conversation state, tool cache,
or provider thread state unless the experiment explicitly studies carry-over
and uses a different profile.

The model-visible MCP registry MUST contain only TwinSpec business tools. The
harness MUST keep snapshot, fork, reset, state inspection, canonical diff,
fault configuration, clock control, expected assertions, and scoring logic
outside the model-visible channel.

Prompt text MUST NOT disclose hidden state, assertion values, expected tool
sequence, or control-plane credentials. The harness MAY disclose task goals and
public domain constraints that a real user would possess.

## 7. Execution budget and termination

Every model-driven trial MUST declare finite limits for:

- provider requests;
- MCP tool calls;
- wall-clock duration;
- provider token or cost budget when available;
- retries per provider request and per tool call;
- maximum repeated identical tool call;
- maximum trace bytes.

The harness MUST distinguish at least:

- `completed`;
- `assertion_failed`;
- `model_stopped`;
- `tool_budget_exceeded`;
- `provider_budget_exceeded`;
- `timeout`;
- `provider_error`;
- `protocol_error`;
- `runtime_error`;
- `cancelled`.

A timeout or ambiguous provider failure MUST NOT be rewritten as success. A
retry policy MUST NOT erase evidence that an earlier call may have committed a
state transition.

## 8. Scoring

State and declared deterministic assertions are the primary oracle. A model's
natural-language answer, provider self-report, or LLM judge MUST NOT override a
failed state assertion or runtime invariant.

For a fixed scenario, all compared models MUST receive:

- the same TwinSpec and canonical tool surface;
- the same initial snapshot digest;
- semantically equivalent task text;
- the same tool-call and wall-clock limits, unless the difference is declared;
- the same fault and virtual-time schedule;
- the same assertion set.

The expected tool sequence MUST NOT be scored. Different valid trajectories may
produce the same successful terminal state.

A single trial MAY establish a smoke result. A comparative model claim MUST use
at least five independent trials per configuration, publish per-trial outcomes,
and report counts rather than only an average. This minimum does not establish
statistical significance; stronger claims require a predeclared analysis.

## 9. Reproducibility identity

The environment identity from SPEC-0005 is deterministic. The model trajectory
is not assumed deterministic. A model-evaluation identity MUST additionally
include:

```text
EvaluationIdentity = (
  environmentDigest,
  scenarioDigest,
  harnessRevision,
  compatibilityProfile,
  hostVersion,
  provider,
  modelIdentifier,
  modelParameters,
  promptDigest,
  toolPolicyDigest,
  trialIndex
)
```

When a host hides a parameter, the report MUST record it as `unknown`; it MUST
NOT invent a default. When a provider resolves a model alias to a dated model
identifier, both values SHOULD be retained if exposed.

## 10. Compatibility report

The canonical report format is
`statetwin.dev/host-compatibility-report/v1alpha1`. A report MUST contain:

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: HostCompatibilityReport
metadata:
  name: example-run
  createdAt: "2026-08-18T00:00:00Z"
runtime:
  version: 0.1.0-dev
  revision: <immutable revision>
  specDigest: sha256:<digest>
  surfaceDigest: sha256:<digest>
  snapshotDigest: sha256:<digest>
host:
  profile: generic-mcp
  name: <host>
  version: <version>
  provider: <provider-or-none>
  model: <model-or-none>
mcp:
  configuredVersion: <version>
  negotiatedVersion: <version>
  transport: streamable-http
  endpointTrust: loopback
  observedSurfaceDigest: sha256:<digest>
  surfaceStatus: exact
trial:
  scenarioDigest: sha256:<digest>
  promptDigest: sha256:<digest>
  toolPolicyDigest: sha256:<digest>
  index: 0
  limits:
    providerRequests: 0
    toolCalls: 32
    wallTimeMs: 60000
  outcome: completed
evidence:
  environmentDigest: sha256:<digest>
  terminalStateDigest: sha256:<digest>
  traceDigest: sha256:<digest>
  assertionSummary:
    passed: 4
    failed: 0
redaction:
  policy: synthetic-only-v1
  secretsDetected: false
```

The concrete schema and serializer are not implemented in the current preview.
Until they are, reports are design examples and MUST NOT be advertised as
runtime output.

## 11. Required test suites

### 11.1 Automated generic MCP suite

This suite is required before `v0.1.0`:

- initialization and protocol negotiation;
- ping;
- exact business-tool discovery;
- successful read-only call;
- successful state-changing call;
- invalid input;
- canonical domain error;
- unknown tool;
- cancellation or documented unsupported status;
- control-tool non-discoverability;
- surface digest equality;
- branch isolation.

### 11.2 Provider profile smoke suite

Each claimed provider profile MUST test at least:

- tool discovery;
- one read-only call;
- one state-changing multi-step scenario;
- one domain error visible to the model;
- terminal state assertions;
- bounded termination;
- artifact redaction.

Manual UI evidence MAY establish `experimental` status, but `verified` requires
a reproducible procedure with immutable runtime and host identifiers. A
screenshot alone is insufficient.

## 12. Claim language

Allowed when supported by evidence:

- "Exposes MCP tools consumable by the tested host configuration."
- "The generic MCP suite passed for the recorded protocol and transport."
- "Model X completed N of M trials for scenario Y under the published budget."

Forbidden without additional evidence:

- "Works with every ChatGPT/Claude model."
- "OpenAI-compatible" or "Claude-compatible" based only on JSON shape.
- "Identical to the upstream service."
- "Deterministic model evaluation" when only the environment is deterministic.
- "Production-ready remote MCP server" under the v0.1 local profile.
- "Safer model" based solely on one scenario or an LLM judge.

## 13. Versioning and change control

Changes to profile names, support levels, evaluation identity, outcome classes,
report fields, or minimum evidence require an ADR and a compatibility note.
Provider-specific implementation details SHOULD live in adapters or harnesses,
not in the deterministic state engine.

The compatibility matrix MUST be regenerated from evidence artifacts. It MUST
NOT be a hand-edited marketing table whose status can diverge from test output.
