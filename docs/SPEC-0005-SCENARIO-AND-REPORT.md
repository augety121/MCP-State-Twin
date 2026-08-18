# SPEC-0005: Scenario and Evidence Report

- **Status:** Proposed normative specification
- **Version:** `statetwin.dev/v1alpha1`
- **Scope:** scripted, deterministic TwinSpec evaluation scenarios

## 1. Purpose

A Scenario is a bounded sequence of calls against one TwinSpec and one initial
synthetic world. It produces a machine-readable evidence report. It is not an
agent implementation, provider harness, workflow engine, or claim that a model
completed the scenario.

## 2. Input contract

Every Scenario MUST be exactly one YAML document:

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: Scenario
metadata:
  name: close-issue
steps: []
finalAssertions: []
```

Unknown fields, multiple documents, explicit YAML tags, anchors, and aliases
MUST be rejected. The document is limited to 256 KiB, 256 steps, 1,024 total
assertions, YAML document depth 128, JSON input/value depth 64, and JSON Pointer
depth 64. Scenario inputs and assertion values MUST remain in the JSON value
domain; YAML-native timestamps and host-specific values are rejected.

Each step declares a stable `id`, a TwinSpec business `tool`, an input object,
and an expected canonical error class. Empty `errorClass` means success. The
runner MUST stop after the first unexpected error class or failed assertion so
later failures are not contaminated by an invalid trajectory.

## 3. Assertions

State assertions use RFC 6901 JSON Pointer and one of:

- `equals`: the resolved value and declared value have identical canonical
  JSON;
- `exists`: the pointer resolves, including when its value is JSON null;
- `absent`: the pointer does not resolve.

Assertions cannot execute CEL, scripts, filesystem access, network access, or
host functions. `/` inside a map key is encoded as `~1`; `~` is encoded as
`~0`. Invalid escapes MUST fail admission.

## 4. Execution semantics

The development profile:

1. creates a fresh in-memory SQLite store;
2. initializes branch `scenario` from the supplied fixture;
3. creates immutable snapshot `scenario-base` and an isolated baseline fork;
4. executes steps serially through the same runtime path as MCP tool calls;
5. evaluates assertions against committed branch state;
6. emits the terminal state digest and canonical diff from the baseline.

Expected domain failures are evidence and MAY pass a scenario. Infrastructure
errors, unknown tools, invalid scenarios, and resource-limit failures MUST fail
the runner explicitly. The accumulated canonical trace is limited to 16 MiB.

## 5. Environment identity

The report records and hashes this envelope:

```text
runtime semantic version
TwinSpec digest
MCP surface digest
initial snapshot digest
Scenario digest
seed = 0
initial virtual clock
scheduler = serial-v0.1
fault profile = none
```

Model/provider identity is deliberately outside this digest. A future provider
harness records agent identity separately and MUST NOT alter environment
determinism.

## 6. Report contract

The report format is `statetwin.dev/scenario-report/v1alpha1` and includes:

- pass/fail and deterministic environment identity;
- initial and terminal state digests;
- ordered tool trace with input, result, error class, call index, and before /
  after state digests;
- canonical state diff;
- assertion results and `declaredInvariantIds` (declarations, not a separate
  per-invariant execution trace);
- configured/fired faults and unsupported-behavior count.

`agentIdentity: scripted-scenario` means repository-defined calls, not an
OpenAI, Claude, Codex, or other live agent run.

## 7. Trust boundary

Scenario reports contain tool inputs and results. The current profile is for
synthetic fixtures only. Reports MUST NOT be committed when they contain
credentials, authorization headers, production traces, or personal data.
Recorder redaction and signed evidence bundles remain separate future work.
