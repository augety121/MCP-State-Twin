# Release Lifecycle & Quality Gates

> **Status:** Proposal  
> **Implementation:** Not implied  
> **Verification:** Unverified

## 1. Principle

Versions are evidence gates, not marketing deadlines.

A release does not become stable because enough features exist. It becomes stable when the contracts promised by that release have evidence.

---

## Phase 0 — Protocol Truthfulness

### Deliverables

- current repo docs/code/test inventory;
- current implementation evidence inventory;
- MCP `2026-07-28` gap matrix;
- revised SPEC-0003;
- revised protocol ADR;
- exact Go SDK version pin;
- exact MCP conformance version/commit;
- expected-failure baseline;
- README claim correction.

### Exit gate

Can answer mechanically:

```text
Which MCP versions are supported?
Which are verified?
Which transports?
Which SDK version?
Which conformance build?
Which scenarios?
Which expected failures?
```

No unresolved protocol claim contradiction.

---

## Phase 1 — Deterministic Runtime / v0.1 Candidate

### Deliverables

- SPEC-0007 virtual time/entropy/scheduler;
- bounded SPEC-0008 deterministic faults;
- SPEC-0012 concurrency/crash/migration;
- SPEC-0015 critical resource limits;
- second independent stateful reference domain;
- deterministic cross-platform corpus for declared support matrix;
- hermetic local profile.

### Exit gate

For the declared deterministic profile:

```text
same environment identity
+ same world commit schedule
=> same result/state/audit identity
```

Must demonstrate:
- branch isolation;
- branch conflicts;
- snapshot/fork equivalence;
- output rollback;
- crash killpoints;
- virtual time;
- seeded entropy;
- no hidden control-plane tools;
- no unexpected egress.

### v0.1 non-goals

Still may omit:
- recorder;
- L2 differential fidelity;
- live provider matrix;
- remote Internet profile.

---

## Phase 2 — Fidelity & Evidence / v0.2 Candidate

### Deliverables

- SPEC-0009 recorder/replay/redaction;
- SPEC-0010 surface/drift inspector;
- SPEC-0011 differential harness;
- SPEC-0014 evidence schema;
- SPEC-0016 bundle alpha;
- at least one legal/safe L0 cassette workflow;
- at least one bounded L2 coverage profile.

### Exit gate

- redaction verified;
- cassette miss behavior verified;
- drift fail-closed policy verified;
- L2 claim points to exact coverage/evidence;
- evidence schema versioned;
- bundle digest verification works.

---

## Phase 3 — Real Host Evaluation / v0.3 Candidate

### Deliverables

- revised SPEC-0006/Host Evaluation;
- Codex profile/harness;
- Claude Code profile/harness;
- OpenAI API remote MCP profile after secure staging;
- Anthropic Messages MCP profile;
- optional Managed Agents profile;
- repeated-run report;
- host compatibility evidence registry.

### Exit gate

Every public verified compatibility row has:
- exact product/profile/version;
- date;
- model/config;
- environment digest;
- scenario;
- terminal assertions;
- evidence artifact.

No global provider compatibility claims.

---

## Phase 4 — Security-Hardened Remote Mode / v0.4+ only if justified

### Deliverables

- TLS;
- data-plane authentication;
- authorization/RBAC profile;
- OAuth protected-resource/audience semantics if used;
- no token passthrough;
- SSRF controls;
- Origin/Host controls;
- rate/request limits;
- credential lifecycle;
- remote deployment runbook;
- security tests;
- staging environment.

### Exit gate

- threat model reviewed;
- auth tests pass;
- secret rotation/revocation drill;
- recovery drill;
- abuse/resource tests;
- no default production upstream writes;
- external review or explicitly scoped internal security gate.

Remote multi-tenancy is still a separate RFC.

---

## Phase 5 — Stable v1.0

### Deliverables

- stable TwinSpec API;
- stable runtime semantic contract;
- stable evidence schema;
- stable bundle format;
- stable storage migration policy;
- canonicalization compatibility policy;
- protocol support policy;
- host compatibility freshness policy;
- deprecation policy;
- upgrade guide;
- backup/restore guide;
- reproducible release workflow;
- SBOM/build provenance;
- signed artifacts if adopted.

### Exit gate

- all P0 requirements traceable;
- no known correctness blocker;
- supported platform matrix verified;
- migration/recovery evidence;
- release claims evidence-backed;
- security boundary documented;
- historical artifacts not silently reinterpreted.

`1.0` means contract stability, not feature completeness.

---

## Phase 6 — Post-1.0 Evolution

Candidate tracks:
- Tasks/MRTR reference profiles;
- advanced multi-agent evaluation;
- tool-search/deferred discovery evidence;
- sandboxed extension runtime;
- computer-use world adapter;
- distributed/remote architecture only via new RFC.

Every track preserves or versions the v1 deterministic/evidence contract.

---

# Quality Gate per Material Change

A mature PR should map to:

```text
requirement ID
 -> implementation
 -> unit test
 -> integration test
 -> determinism test if semantic
 -> security test if boundary-related
 -> evidence fixture
 -> Implementation Status
 -> documentation/claim update
```

## Required review questions

- Does this change state semantics?
- Does it change canonical bytes/digests?
- Does it change security boundary?
- Does it change host-visible surface?
- Does it change evidence?
- Does it change storage?
- Does it introduce nondeterminism?
- Does it need migration?
- Does it invalidate compatibility/fidelity evidence?

---

# P0 Release Blockers

Examples:

- nondeterministic canonical state digest;
- silent state corruption;
- fork isolation breach;
- control-plane capability exposed to agent;
- hermetic profile can write production unexpectedly;
- secret persisted in evidence;
- protocol claim contradicted by pinned conformance;
- migration can corrupt supported DB without recovery;
- unknown behavior returns fabricated success;
- required audit/state atomicity violated.

---

# P1 Examples

- incomplete failure coverage;
- compatibility evidence stale;
- missing resource limit in a non-exploitable local-only path;
- documentation drift;
- performance regression outside correctness boundary.

P1 classification still requires owner and release decision.

---

# Release evidence bundle

Each release SHOULD archive:

```text
version manifest
source commit
dependency lock/module versions
test matrix
conformance output
deterministic corpus report
security gate report
migration report
host compatibility evidence
known expected failures
SBOM/provenance if in release scope
```

---

# Release claim policy

A release note MUST separate:

- New: implemented + verified
- Experimental: implemented but bounded/partially verified
- Specified: design exists but not implemented
- Known gaps
- Breaking changes
- Evidence links

Roadmap items never belong in the “implemented” list.
