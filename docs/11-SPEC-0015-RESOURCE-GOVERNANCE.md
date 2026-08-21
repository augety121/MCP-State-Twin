# SPEC-0015 — Resource Governance, Limits & DoS Boundaries

> **Status:** Proposal  
> **Implementation baseline:** source-reported partial CEL limits  
> **Verification status:** unverified

## 1. Principle

Every user-authored, agent-controlled or imported dimension that can consume CPU, memory, disk, network or unbounded output must have a limit before stable release.

## 2. Limit classes

At minimum define:

```text
max_spec_bytes
max_tool_count
max_entity_type_count
max_entities_per_branch
max_state_bytes
max_input_bytes
max_output_bytes
max_json_depth
max_json_members
max_schema_bytes
max_schema_depth
max_schema_refs
max_expression_bytes
max_expression_cost
max_effects_per_call
max_query_result_items
max_diff_entries
max_diff_bytes
max_audit_event_bytes
max_report_bytes
max_scenario_steps
max_scheduled_events
max_fault_rules
max_forks
max_concurrent_calls
max_cassette_bytes
max_bundle_files
max_bundle_compressed_bytes
max_bundle_extracted_bytes
future_max_task_count
```

## 3. Requirements

### ST-LIMIT-R001
Every limit exceedance MUST produce a typed error/result.

### ST-LIMIT-R002
Where practical, reject before unbounded allocation/work.

### ST-LIMIT-R003
A limit that can change semantic outcomes MUST be included in or referenced by environment identity.

### ST-LIMIT-R004
Changing semantic limits can invalidate comparison evidence and therefore requires a profile/version change.

### ST-LIMIT-R005
Infrastructure resource exhaustion MUST NOT be reported as a modeled business-domain failure unless that behavior is explicitly part of the Twin.

### ST-LIMIT-R006
Nested/recursive data structures require depth limits in addition to byte limits.

### ST-LIMIT-R007
Imported archive/bundle limits apply before full extraction where technically possible.

## 4. CEL

Retain bounded source and runtime cost.

Also test:
- large comprehensions;
- pathological nested expressions;
- huge JSON values supplied to expressions;
- function/library expansion cost.

## 5. Schema validation

Bound:
- schema bytes;
- recursion/depth;
- reference graph;
- instance size;
- number of validation errors retained.

Hermetic mode must reject external fetch.

## 6. Diff/report bounds

A huge state change can produce a larger diff than the state itself.

The runtime SHOULD support:
- entry cap;
- byte cap;
- explicit truncation metadata;
- digest of complete result where safely computable;
- fail/stream policy defined per profile.

Never silently truncate while presenting a complete diff.

## 7. Scheduler/fault bounds

Virtual time does not make unbounded event loops safe.

Require:
- max due events per advance;
- max cascade depth/ticks;
- max newly scheduled events per transition;
- loop-detection/budget error.

## 8. Fork bounds

Fork count can consume storage.

Limit and report:
- active branches;
- snapshots;
- total state storage;
- audit retention.

## 9. Performance budgets

Before v1.0 set empirical release budgets for a fixed corpus:

- transition p50/p95;
- snapshot;
- fork;
- diff;
- scheduler;
- state scaling;
- memory peak;
- WAL growth/checkpoint;
- evidence overhead.

These are release budgets, not universal performance guarantees.

## 10. Feasibility

**High.**

The difficult part is selecting defaults that are secure while still allowing realistic stateful worlds. Limits should therefore be profile-driven and versioned.
