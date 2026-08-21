# Compatibility CI, Evidence Freshness & Staleness

> **Status:** Proposal / Unverified  
> **Purpose:** Keep a large host compatibility matrix trustworthy over time.

## 1. Compatibility verification layers

### Layer 0 — Static profile/source lint

Checks:
- profile schema;
- source references;
- required fields;
- no unsupported global claim;
- freshness metadata.

### Layer 1 — Portable-surface lint

Checks canonical Twin surface against selected host constraints:
- tool names;
- projected names;
- schema transforms;
- feature dependencies;
- tool-count limits;
- transport/auth requirements;
- output profile.

### Layer 2 — Adapter fixture tests

No real host required.

Checks:
- config generation;
- projection;
- parsing;
- normalization;
- error classification.

### Layer 3 — MCP protocol conformance

Tests server protocol independent of host product.

### Layer 4 — Connection smoke

Real host connects and sees expected required tools.

### Layer 5 — Stateful scenario smoke

Real host completes a bounded stateful task.

### Layer 6 — Repeated evaluation

Multiple trials with evidence.

### Layer 7 — Stress/advanced profile

Faults, concurrency, multi-agent, long-running behavior.

## 2. Compatibility linter concept

Future CLI:

```text
statetwin compat lint \
  --spec examples/.../twin.yaml \
  --targets codex,claude-code,gemini-cli,cursor
```

Potential output:

```text
PASS codex
WARN gemini-cli schema projection strips additionalProperties
FAIL windsurf target profile exposes > configured/host tool budget
UNKNOWN some-host no fresh profile
```

This is a proposed interface, not current functionality.

## 3. Evidence freshness

Evidence becomes stale when a material identity changes.

Examples:
- host version;
- model snapshot/alias resolution;
- MCP protocol version;
- State Twin runtime;
- adapter;
- projected schema;
- tool surface;
- approval policy;
- connector beta;
- host permission model.

Time alone may also trigger periodic revalidation.

## 4. Freshness states

```text
fresh
stale_version
stale_surface
stale_adapter
stale_protocol
stale_policy
stale_time
unknown
```

## 5. Scheduled vs manual real-host tests

Real host tests may:
- cost money;
- require credentials;
- be nondeterministic;
- have rate limits.

Therefore CI SHOULD separate:

### PR CI
- static;
- unit;
- protocol;
- adapter fixtures;
- deterministic scenarios.

### Scheduled/manual compatibility jobs
- Codex;
- Claude;
- API hosts;
- other agent products.

A missing external credential must not turn into a false PASS.

## 6. Secret policy

Real-host CI:
- uses secret store;
- does not print secrets;
- redacts config;
- uses synthetic fixtures;
- uses restricted staging endpoint where remote access is needed.

## 7. Claim registry

Every public claim should be backed by:

```yaml
claim:
  id: claude-code-local-http-...
  status: compatible
  capabilitySet: ...
  evidence: ...
  observedAt: ...
  staleAfterPolicy: ...
```

## 8. CI claim guard

Future CI SHOULD fail if:
- README says verified but claim registry lacks fresh evidence;
- evidence host identity no longer matches profile;
- required artifact missing;
- profile is marked stale/blocked.

## 9. Expected failures

Expected failures must be:
- explicit;
- scoped;
- owned;
- reviewed;
- not a permanent blanket suppression.

Unexpected PASS should trigger review because the upstream/host may have changed.

## 10. Compatibility dashboard

A generated dashboard MAY show:

```text
Host
Profile
Documented
Adapter
Connection
Scenario
Last verified
Staleness
Known limitations
```

It MUST NOT compress everything into a green check if important capabilities are untested.

## 11. Feasibility

**High.**

The main operational cost is maintaining real-host credentials and re-running integrations as products evolve.
