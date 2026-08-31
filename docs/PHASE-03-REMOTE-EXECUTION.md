# Phase 3: Durable Remote Episode Execution

- **Target:** `v0.2.x` experimental profile
- **Entry:** Phase 1 complete; relevant Phase 2 storage/recovery gates complete
- **Authority:** ADR-0020 and SPEC-0018

## Required scope

- independent Episode Journal identity and schema-v2 migration;
- immutable parent request digest;
- bounded attempts, leases and heartbeat extension;
- monotonically increasing fencing tokens;
- cooperative durable cancellation;
- hermetic-only safe retry after proven `NO_EFFECT` or expired lease;
- external ambiguity as terminal `COMMIT_UNKNOWN`;
- atomic idempotent acceptance of one terminal Evidence envelope;
- coordinator authentication and non-loopback TLS admission;
- worker runtime/capability identity and bounded TwinBundle revalidation.

## Non-claims

No multi-coordinator HA, replication, arbitrary workflow DAG, worker code
upload, production multi-tenancy or exactly-once provider/tool/external effect.

## Exit evidence

- concurrent claims select one owner;
- heartbeat and maximum lease bounds;
- stale fence refusal after recovery;
- hermetic expiry requeue and attempt exhaustion;
- external expiry produces `COMMIT_UNKNOWN` without retry;
- queued/leased cancellation races;
- coordinator restart and Journal reopen;
- identical completion is an idempotent lookup;
- conflicting completion is rejected;
- credential and raw error absence;
- Journal v1 fixture migration and kill-point integrity checks.
