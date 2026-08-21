# SPEC-0014 — Evidence, Audit & Observability

> **Status:** Proposal  
> **Implementation baseline:** source-reported partial audit/scenario foundation  
> **Verification status:** unverified

## 1. Principle

MCP State Twin's canonical evidence must be a **project-owned, versioned schema**.

OpenTelemetry, provider traces and host logs are optional exports. They are not the source of truth for reproducible evaluation.

## 2. Evidence layers

### Layer A — Environment identity

```text
runtime_version
build_id
MCP_protocol_profile
MCP_extension_profile
storage_schema_version
canonicalization_id
TwinSpec_digest
surface_digest
fixture/snapshot_digest
scenario_digest
scheduler_semantics
entropy_profile
fault_plan_digest
resource_limit_profile
```

### Layer B — World trace

Per event/call:

```text
event_id
call_id
agent_id_if_known
parent_agent_id_if_known
tool_name
canonical_input_digest
pre_state_digest
post_state_digest
branch_head_before
branch_head_after
world_commit_sequence
virtual_time
fault_reference
raw_server_result_digest
```

### Layer C — Host observation

```text
provider
product
host_version
model_configured_id
resolved_model_snapshot_if_exposed
host_config_digest
MCP_config_digest
host_visible_surface_digest
allowed_tools
approval_policy
host_timeout_budget
host_visible_result_digest
host_transform_class
```

### Layer D — Evaluation

```text
assertions_evaluated
assertions_skipped
assertions_failed
terminal_invariants
canonical_diff
score_dimensions
completion_reason
accepted_divergences
```

## 3. Evidence schema requirements

### ST-EVID-R001
Evidence schema versions MUST be explicit and independently versioned.

### ST-EVID-R002
A digest field MUST identify hash algorithm and canonicalization contract, directly or by referenced environment profile.

### ST-EVID-R003
Raw server result and host-visible transformed result MUST remain distinguishable.

### ST-EVID-R004
Secrets MUST be removed before durable evidence storage.

### ST-EVID-R005
Host wall-clock timestamps are non-deterministic metadata unless explicitly included in a non-deterministic observation profile.

### ST-EVID-R006
Interrupted runs MUST be representable as `incomplete`.

### ST-EVID-R007
Skipped assertions MUST NOT be counted as passed assertions.

### ST-EVID-R008
Evidence MUST distinguish a task failure from a host/harness/infrastructure failure.

### ST-EVID-R009
Evidence MUST identify applicable source/tool/profile versions needed to interpret the run.

## 4. Audit semantics

Audit serves security/forensics; evidence serves evaluation/reproduction.

They may share records but should not be conflated.

Audit examples:
- privileged control operations;
- auth decisions;
- storage recovery;
- configuration changes.

Canonical evaluation examples:
- world transition;
- terminal state;
- assertion.

## 5. Multi-agent ordering

For concurrent calls:

```text
host_observed_sequence
agent_id
parent_agent_id
request_id
world_commit_sequence
```

The authoritative world ordering is `world_commit_sequence`.

Host/model thought order is not required and may be unavailable.

## 6. Provider/host transforms

A host may:
- filter tools;
- require approval;
- truncate output;
- store large output as a file/attachment;
- retry;
- aggregate tool calls.

Evidence MUST preserve the distinction:

```text
server_truth != host_observation
```

when a transform occurs.

## 7. Redaction

Redaction policy itself MUST be versioned.

Evidence may store:
- redacted value;
- digest of pre-redaction canonical content only if doing so does not create unacceptable secret leakage;
- a classification marker.

Do not hash secrets blindly and assume this is safe; low-entropy secrets can still be attacked offline.

## 8. OpenTelemetry export

An optional OTel exporter MAY emit spans/attributes.

### ST-EVID-R020
OTel export MUST follow an explicit sensitive-data policy.

### ST-EVID-R021
Exporter failure MUST NOT alter world semantics.

### ST-EVID-R022
OTel semantic-convention changes MUST NOT silently change canonical State Twin evidence.

## 9. Evidence integrity

Eventually stable evidence MAY be:
- content-addressed;
- bundled;
- accompanied by build provenance;
- signed.

Signature/provenance is separate from semantic correctness.

## 10. Retention

Profiles SHOULD define:
- retention;
- maximum report size;
- whether tool arguments/results are persisted;
- data classification;
- deletion rules.

## 11. Required tests

- complete scripted scenario evidence;
- incomplete/crashed run;
- skipped assertion;
- host output transform;
- secret redaction;
- concurrent call ordering;
- digest mismatch detection;
- unsupported evidence version;
- exporter failure;
- evidence size limit.

## 12. Feasibility

**High.**

The main challenge is not collecting data; it is keeping evidence schemas stable, minimal, privacy-safe and semantically precise.
