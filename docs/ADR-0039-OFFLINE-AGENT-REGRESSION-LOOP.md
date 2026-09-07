# ADR-0039: Bounded Offline Agent Regression Loop

- Status: Accepted for the explicit offline subset below
- Date: 2026-09-08
- Basis: maintainer request to push and continue implementing the reviewed plan
- Release status: experimental; no change to stable v0.1 support or gates

## Decision

Accept the offline portions of B04/B05/B06/B07/B09/B10/B12 in the
[reviewed phase plan](planning/agent-evaluation/PHASE-SPECS.md), with the exact
scope in [SPEC-0039](SPEC-0039-OFFLINE-AGENT-REGRESSION.md). Retain the existing
kernel, official SDK MCP path, independent Task and read-only evaluator.

The new host is a synthetic Responses function-call codec and sequential mock
loop, not a network API client. `eval mock` consumes explicitly synthetic
response items and cannot read credentials or configure an endpoint. Each run
owns a fresh private session and in-memory world. Existing hermetic commands
gain no external path. There is no shell/computer capability or MCP control tool.

Requests project only visible objective/context/authorization and declared
business function schemas. Explicit `strict:false` preserves their optional
field semantics; unsupported names are refused, not renamed. Private reasoning
continuation remains in current-session memory and is never an evidence field.

The separate evidence kind retains a verified TwinBundle, Task, bounded business
events, request-delivery frontiers, terminal state and grading. A full replay
closure must be checked and synced before the world is disposed. Publication
uses a same-directory no-clobber hard link; unsupported filesystems fail closed.
Partial results cannot become complete, even if a goal predicate is true.

Comparison uses canonical equality of complete bounded definitions after
excluding only declared model labels and per-trial identity. No new file hash
manifest is introduced. Existing Bundle identity checks remain. Plan entries
are retained in the denominator; comparison never selects a best retry.

## Not accepted or claimed

- No paid/live provider transport, provider availability or real-model score.
- No ChatGPT, Codex or Claude product-host compatibility claim.
- No native remote MCP deployment, OS sandbox or general remote drain protocol.
- No automatic resume/retry after interruption, signed provenance or
  exactly-once business side effects.
- No arbitrary configuration comparison, monetary governor or statistical
  significance claim; the accepted comparison variable is a synthetic model label.
- No assurance of hardware power-loss durability of directory entries on all
  filesystems; staging is inspect-only, not an automatic recovery journal.
- No blanket completion of B04–B12, AE-001–024, all 109 missing original
  requirement clauses, or the proposed full lifecycle.

This is an offline regression product slice. Broader lifecycle fault-injection,
remote cancellation/reconciliation, real live profiles and independent user
trials remain separate work packages and evidence gates.
