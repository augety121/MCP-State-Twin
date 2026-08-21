# SPEC-0015 — Resource Governance (Accepted Local Profile)

- **Status:** Accepted subset via ADR-0013
- **Verification status:** executable unit, runtime, storage, diff, and report-bound tests
- **Profile:** `statetwin.dev/resource-profile/v1alpha1`, `local-preview-v1`

## 1. Typed profile

Run `statetwin limits` to print the complete profile and its SHA-256 digest.
The digest is part of Scenario `EnvironmentIdentity`; changing a semantic
limit changes the environment identity.

Limits cover:

| Class | Current local bound |
|---|---:|
| TwinSpec bytes | 1 MiB |
| tools / entity types | 256 / 256 |
| entity records per branch | 100,000 |
| state bytes | 16 MiB |
| input / output bytes | 1 MiB / 1 MiB |
| JSON depth / members | 64 / 1,000,000 |
| schema bytes / depth | 256 KiB / 32 |
| expression bytes / CEL cost | 4,096 / 10,000 |
| effects per call | 128 |
| query result items | 10,000 |
| diff entries / encoded bytes | 10,000 / 8 MiB |
| audit payload bytes | 2 MiB |
| scenario steps / report bytes | 256 / 32 MiB |
| fault plans / branches / snapshots | 128 / 1,024 / 1,024 |
| concurrent calls | serialized local SQLite writer |

Bundle, cassette, scheduled-event, and future-task limits are zero because
those features are not implemented in this release profile.

## 2. Fail-closed semantics

A limit is checked before the relevant work whenever practical. If a limit
cannot be checked until after evaluation, the transition is rejected and no
world state is committed. Resource exhaustion uses `RESOURCE_LIMIT`; callers
must not interpret it as a domain `CONFLICT`, `NOT_FOUND`, or successful
result.

JSON byte, depth, and member budgets are applied recursively. Diff generation
tracks both entry count and canonical encoded bytes and returns an error rather
than silently truncating a complete diff.

## 3. Storage and identity

Branch, snapshot, and fault-plan count checks occur inside SQLite transactions.
State and fault plans are branch-local. Scenario reports carry
`limitProfileDigest`, and the `limits` CLI command is the machine-readable
inspection point.

## 4. Deferred requirements

The profile does not yet implement:

- OS memory/CPU quotas or distributed tenant fairness;
- scheduler due-event/cascade budgets;
- cassette or reproducible-bundle budgets;
- remote request rate limiting;
- storage quota/GC/WAL checkpoint governance;
- empirical p50/p95 performance budgets.

Each deferred item requires a feature-specific ADR and executable evidence.
