# Open Questions & Decision Gates

> **Status:** Proposal planning artifact  
> **Purpose:** Explicitly record what must NOT be guessed today.

A rigorous lifecycle spec has both decisions and **deliberately undecided questions**. These gates prevent speculative future requirements from silently becoming architecture commitments.

## DG-001 — Legacy MCP support

**Question:** Should State Twin retain protocol support through `2025-11-25`, or become modern-only after rebaseline?

**Decide when:** Phase 0 inventory shows actual host/user need.

**Evidence required:**
- which target hosts still require legacy lifecycle;
- maintenance/test cost;
- migration impact.

**Default before decision:** do not promise legacy support.

---

## DG-002 — Canonicalization migration

**Question:** Keep current alpha canonicalization indefinitely, or migrate toward a standard such as RFC 8785/JCS?

**Decide when:** before stable v1.0 canonical contract.

**Evidence required:**
- corpus diff;
- number edge cases;
- Unicode behavior;
- backward evidence compatibility;
- performance;
- ecosystem benefit.

**Default:** preserve existing contract; no silent switch.

---

## DG-003 — Virtual-time API surface

**Question:** Exact control endpoints/names for clock advancement?

**Decide when:** SPEC-0007 implementation design.

**Evidence required:** CLI/control-plane consistency and scenarios.

**Invariant already decided:** never model-visible business tools.

---

## DG-004 — PRNG algorithm

**Question:** Which deterministic PRNG algorithm becomes the first stable entropy profile?

**Decide when:** implementing SPEC-0007.

**Criteria:**
- deterministic cross-platform behavior;
- maintained implementation;
- easy golden tests;
- algorithm ID can be persisted.

**Default:** do not specify an algorithm from memory in the normative contract before implementation review.

---

## DG-005 — Branch conflict error wire mapping

**Question:** How does internal `BRANCH_CONFLICT` map to MCP-visible error/result semantics?

**Decide when:** SPEC-0012 implementation.

**Need:** preserve distinction from modeled domain conflict while remaining compatible with MCP tool-call error handling.

---

## DG-006 — Second reference domain

**Question:** Which independent stateful domain best demonstrates generality?

**Decide before:** v0.1 candidate.

**Selection criteria:**
- stateful;
- meaningfully different from issue tracker;
- synthetic fixtures possible;
- no legal/provider dependency;
- exercises time/fault/idempotency if possible.

Potential candidates are examples only, not decisions:
- order/inventory workflow;
- calendar/scheduling workflow;
- simple ticket/approval system.

---

## DG-007 — Recorder upstream policy

**Question:** Will official recorder support be sandbox/test-only or allow explicitly authorized production capture?

**Decide before:** recorder release.

**Security/legal review required.**

**Safe default:** sandbox/test upstream only.

---

## DG-008 — L2 promotion threshold

**Question:** What minimum differential sample/partition coverage qualifies a coverage profile as L2?

**Decide during:** SPEC-0011 implementation.

There is no universal mathematically correct threshold. It must be risk/domain based and explicitly documented.

---

## DG-009 — Evidence event granularity

**Question:** Store full args/results, redacted args/results, digests only, or profile-dependent?

**Decide before:** evidence schema beta.

Tradeoffs:
- replay/debuggability;
- privacy;
- storage;
- provider terms;
- reproducibility.

**Default:** synthetic data allows richer evidence; secrets always excluded.

---

## DG-010 — Bundle archive format

**Question:** tar/zip/directory/OCI-like artifact?

**Decide after:** manifest semantics stabilize.

First stabilize logical bundle model; container/archive format is secondary.

---

## DG-011 — Codex exact harness implementation

**Question:** Which Codex CLI/app/API path is the canonical CI smoke harness?

**Decide when:** host implementation begins.

**Required:** re-read current official OpenAI Codex docs at that time. Do not freeze 2026-08-18 UI/CLI details prematurely.

---

## DG-012 — Claude exact harness implementation

**Question:** Claude Code CLI vs other automation path for deterministic CI smoke?

**Decide when:** host implementation begins.

Need current first-party configuration/permission behavior and available noninteractive test mode.

---

## DG-013 — Public remote staging

**Question:** Where/how is a remote MCP endpoint hosted for OpenAI API / Anthropic Messages connector tests?

**Decide before:** remote provider smoke.

Requirements:
- no production data;
- ephemeral credentials;
- TLS;
- auth;
- rate limit;
- teardown;
- evidence identity.

This is a deployment decision, not core TwinSpec semantics.

---

## DG-014 — Host transcript retention

**Question:** How much Codex/Claude/provider transcript is retained?

**Decide before:** public host evidence.

The evaluation MUST NOT depend on hidden reasoning. Retain only observable/allowed data necessary for debugging/evidence.

---

## DG-015 — Cost accounting

**Question:** Is model/API cost a canonical score or optional observation?

**Recommended direction:** optional host metric, because provider pricing changes and local hosts differ.

Decide before benchmark publication.

---

## DG-016 — Statistical evaluation contract

**Question:** Recommended `N`, confidence intervals, aggregation and failure handling for live-agent benchmarks?

**Decide before:** publishing model comparisons.

This should be a separate evaluation-methodology SPEC/RFC after real host data exists.

Do not invent a universal sample size now.

---

## DG-017 — Tasks extension

**Question:** Is there a real domain requiring durable async tasks?

**Decision trigger:** concrete scenario blocked without Tasks.

Until then: future candidate.

---

## DG-018 — MRTR

**Question:** Is server-initiated user/client input necessary for a real evaluation domain?

**Decision trigger:** concrete host-supported scenario.

Until then: capability-gated future work.

---

## DG-019 — Multi-agent shared-world semantics

**Question:** Which conflict policy and fairness assumptions should shared branches expose?

**Decide after:** single-agent branch concurrency is stable and a real multi-agent scenario exists.

No fairness guarantee should be invented without implementation/harness control.

---

## DG-020 — Remote multi-tenancy

**Question:** Should State Twin ever become a hosted multi-tenant service?

**This is a product fork decision.**

Require a dedicated RFC covering:
- tenants;
- identity;
- RBAC;
- isolation;
- storage;
- HA;
- quotas;
- billing/cost;
- encryption;
- backup;
- incident response.

No decision is needed for local 1.0.

---

## DG-021 — Extension runtime

**Question:** Is declarative TwinSpec insufficient for real adopters?

**Decision trigger:** at least two concrete extension needs not responsibly expressible with reviewed built-ins.

Do not create arbitrary native plugin support speculatively.

---

## DG-022 — GUI/computer-use worlds

**Question:** Should this repository support visual/computer-use environments?

**Decision trigger:** a concrete evaluation use case and fidelity design.

Likely requires separate world adapter family, not expansion of MCP tool semantics.

---

## DG-023 — Future protocol adapter

**Question:** Should internal world runtime be exposed through a protocol-neutral adapter interface?

**Decision trigger:** actual non-MCP host/protocol demand.

Architecturally avoid unnecessary MCP coupling now, but do not build unused abstraction layers.

---

## DG-024 — Supply-chain target

**Question:** Which SLSA/provenance/signing maturity level is required for 1.0?

**Decide during:** release-engineering hardening.

Do not confuse provenance level with semantic verification.

---

## DG-025 — Supported platform matrix

**Question:** Which OS/architecture combinations become officially supported at 1.0?

**Decide from:** CI availability + deterministic corpus evidence + maintainer capacity.

README should not imply support beyond tested platforms.

---

# Decision rule

A Decision Gate closes only when:

```text
problem is real
+ options documented
+ primary sources refreshed
+ tradeoffs recorded
+ migration impact understood
+ owner accepts decision
```

If evidence is insufficient, leaving a gate OPEN is more rigorous than guessing.


---

## DG-026 — Initial universal host set

**Question:** Which hosts are release-gating vs best-effort?

**Recommended initial release-gating set after the framework exists:**
- Codex;
- Claude Code;
- Gemini CLI;
- one GitHub Copilot local/IDE path;
- one generic MCP client.

Other hosts can be added incrementally.

**Reason:** trying to gate v0.x on every commercial agent immediately would make release availability depend on external products.

---

## DG-027 — Portable profile exact MCP revision

**Question:** Should the portable tools profile require only `2026-07-28`, or also support a legacy target?

**Decide from:** Phase 0 host evidence.

Do not preserve legacy purely for theoretical compatibility.

---

## DG-028 — Host schema projection policy

**Question:** Which schema rewrites count as `syntactic` versus `semantic-risk`?

**Decide by:** golden projection corpus + accepted input-equivalence tests.

No blanket classification.

---

## DG-029 — Strict benchmark built-in tools

**Question:** For coding-agent comparisons, which non-MCP built-ins may remain enabled?

**Decision should be per benchmark profile**, not universal.

Examples:
- coding benchmark may require filesystem/editor;
- pure tool-world benchmark may prefer MCP-only.

---

## DG-030 — Surface readiness timeout

**Question:** How long/what events define a "settled" tool surface for progressively loading hosts?

**Decide per adapter** based on observable host behavior.

No arbitrary global sleep.

---

## DG-031 — Host projection persistence

**Question:** Should host-projected schemas be stored as full artifacts or digests+transform logs?

**Decide before:** evidence schema beta.

Synthetic public fixtures may store full projections; sensitive/proprietary host internals may require minimized evidence.

---

## DG-032 — Tool-set minimization policy

**Question:** Should every episode expose only required tools, or include standardized distractors?

**Recommended:** support both as explicit scenario profiles.

Do not let hidden filtering make one host's task easier.

---

## DG-033 — Curriculum feedback protocol

**Question:** How is post-run feedback delivered back to agents?

**Decide after:** blind evaluation episodes work.

Keep coaching outside core Twin semantics.

---

## DG-034 — ACP path support

**Question:** Which ACP editor/agent path, if any, becomes a tested State Twin compatibility profile?

**Decision trigger:** real user value from editor-hosted external agents.

ACP remains optional.

---

## DG-035 — A2A evaluation

**Question:** Should State Twin ship an A2A test harness for agent delegation?

**Decision trigger:** a concrete multi-agent scenario where A2A gives value beyond host-local subagents.

State Twin should not become an A2A server by default.

---

## DG-036 — Non-MCP function bridge

**Question:** Is there a high-value agent host that lacks MCP and justifies a native-function bridge?

**Decision trigger:** concrete host demand.

Such bridge is separate from MCP conformance.

---

## DG-037 — Compatibility evidence expiry

**Question:** Which changes trigger mandatory rerun and whether an age threshold also applies?

**Must include identity-driven invalidation.**

Time-based expiry may be added per host ecosystem.

---

## DG-038 — Benchmark private artifacts

**Question:** Where are held-out assertions/seeds stored in public CI?

**Need:** prevent evaluated agents from reading them while still allowing reproducible trusted evaluation.

This may require split repositories/secrets/encrypted artifact infrastructure later.

---

## DG-039 — Multimodal artifact backend

**Question:** local content-addressed directory, DB blob, or external artifact store?

**Decide only when real multimodal use exists.**

---

## DG-040 — Tool projection vs exact fidelity

**Question:** What happens when a host cannot represent a bound upstream schema exactly?

**Default:** mark host semantic-risk/unsupported for strict fidelity. Never weaken canonical upstream contract silently.
