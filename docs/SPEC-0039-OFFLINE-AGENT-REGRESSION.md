# SPEC-0039: Offline Agent Regression Contract

Status: implemented experimental subset accepted by
[ADR-0039](ADR-0039-OFFLINE-AGENT-REGRESSION-LOOP.md), 2026-09-08.
This specification governs the new offline lane, not old scripted Episode,
Journal, provider smoke or remote coordinator formats.

## 1. Purpose and non-claims

Provide an executable path from independent Task to synthetic model turns,
official-SDK MCP business calls, deterministic grading, retained world evidence
and a fixed-plan regression report. It tests the **host and evaluation plumbing**.
The supplied synthetic responses do not demonstrate autonomous model ability.

The codec follows the documented [Responses function-call item contract](https://developers.openai.com/api/docs/guides/function-calling)
and [stateless continuation contract](https://developers.openai.com/api/docs/guides/reasoning),
checked on 2026-09-08. These sources establish request shape, not live access or
successful integration. No actual model ID, endpoint, pricing or account support
is inferred. No new optional SDK dependency or network transport is installed.

## 2. Public entry points

| Command | Contract | Exit behavior |
|---|---|---|
| `eval preflight` | Task/Bundle/offline config/tool projection static admission; no model or world execution | nonzero on invalid/unsupported input |
| `eval mock` | one synthetic Responses script, fresh world/session, new evidence directory | nonzero for execution/cleanup/evidence/task failure; a failed task can still have valid evidence |
| `eval verify` | strict admission, Bundle verification, full world replay and grading comparison | nonzero for partial, incompatible, corrupt or mismatched evidence |
| `eval compare` | fixed plan, JSON or Markdown, no new trials | nonzero for regression, incomparable or inconclusive |

All paths except the operator-selected trusted `--root` are relative, portable
paths confined to that root. `--out` is a **new directory**, with pre-existing
trusted parents. Observed symlink parents are rejected. The root is not a
sandbox against a malicious local actor replacing files during execution.
No command accepts `--endpoint`, credentials, a live switch or a host shell.

## 3. RunConfig and RunDefinition

RunConfig format: `statetwin.dev/agent-run-offline/v1alpha1`. Closed field set:

| Field | Admission |
|---|---|
| `format` | exact current format |
| `trialId` | lowercase letter followed by up to 63 lowercase letters/digits/hyphens |
| `profile` | exactly `responses-functions-offline-v1` |
| `model` | same identifier grammar, required `mock-` prefix; synthetic configuration label |
| `maxOutputTokens` | integer 1–8192; transport request field, not measured token usage |
| `syntheticOnly` | required true |

Config: 16 KiB, bounded strict JSON/YAML document; unknown fields, aliases, tags
and multiple documents fail. Actual model availability is **not checked**.
Task budgets remain [SPEC-0038](SPEC-0038-AGENT-TASK-OFFLINE-ADMISSION.md).

RunDefinition freezes config and Task and records the verified existing Bundle
digest, runtime version/revision, isolation, projection and mock snapshot status.
`unknown` development build revision remains unknown; it is not source attestation.
Use a pinned build for externally meaningful reproducibility. No provider
snapshot is fabricated: the offline value is `not-applicable-mock`.

## 4. Codec and private state

The trusted single-owner Session exposes only request encoding, response
admission, ordered tool admission, result delivery and permanent stop. There is
no HTTP client, filesystem or credential access in `internal/agenthost`.

Each request carries explicit `store:false`, `stream:false`,
`parallel_tool_calls:false`, the configured output-token bound and the fixed
instruction revision compiled into the codec. Only Task objective, context and
resource authorization enter the user message. Oracle, world state, fault plan,
artifact paths, effect markers and control-plane details are private.

Function projection preserves canonical names, descriptions and input schemas
without mutation. Names outside `[A-Za-z0-9_-]{1,64}` or missing/duplicate tools
are refused. `strict:false` avoids silently rewriting optional fields into
required/null fields. Output schemas and MCP annotations are not provider
function fields; this is a declared projection, **not generic lossless MCP**.
Local runtime input/output validation remains authoritative.

Only completed non-streaming responses are admitted. The supported output items
are function calls, assistant text/refusal messages and private reasoning
continuation. Unsupported item types/extensions fail explicitly. Reasoning items
are retained byte-for-byte for the current session's next request, not analyzed
or persisted. Stopping clears history and pending references. A new session has
no prior trial's results or continuation.

Response admission rejects invalid JSON, duplicate object keys, nesting above
32, non-object arguments, unsupported numeric values, duplicate call IDs within
or across turns, malformed IDs and unsupported function-call extensions. The
**entire batch** passes shape/ID admission before exposing any dispatchable call.
Protocol errors permanently stop the session and return no partial batch.

Assistant `phase:commentary` is retained only as continuation, never as a final
answer. A completed response without calls or a final/legacy-unphased answer
is a protocol error; unknown phases are unsupported.

Then each call consumes one attempt and passes host authority/schema checks
before local MCP dispatch. Unknown tools and wrong resources are refused and
counted; they do not become control operations. Tool order matches the response
array. Independent committed tool transactions are never rolled back by a later
batch/protocol failure. No ID is treated as a business idempotency key.

## 5. Budgets, stop and delivery

The mutex-protected governor atomically admits model requests, tool attempts and
content bytes. It rejects negative sizes, exhausted counts/bytes, cancellation,
elapsed Episode deadline and admissions after stop. Rejected authorization and
input attempts count; automatic retries are absent.

The existing Task limits cap 16 model requests, 32 attempts, 1 MiB per response,
8 MiB accumulated transport response/result content, and reserve 64 KiB from
that content budget. A constructed private request must also fit the trace-byte
limit. This byte counter is **not the size of the complete retained world
artifact**, which has an independent 32 MiB encoded bound. Output token count
is requested but not measured by a mock. No monetary governor is claimed.

One Episode and one tool run at a time. The local official MCP handler is invoked
through a fixed in-process HTTP transport, without a listener/DNS/fallback.
All execution calls are joined synchronously; no unbounded background provider
worker exists. Go CPU/memory settings are soft process controls, not hard OS
quotas. Existing CLI quiet mode and local test `GOMAXPROCS=1`, `-p 1` remain.

STOP prevents further admission, not already committed effects. Execution
deadline/cancellation is separate from the bounded terminal inspection/cleanup
context. This profile has no detached remote execution and does not implement
general remote drain/quarantine or OS-process cancellation.

`delivered` in new AgentEpisode means the result was included in a subsequent
synthetic model request. Merely receiving a tool result at the MCP client is
insufficient. `requestFrontiers` records each admitted request's prior-event
count. A tool committed before the next request budget was exhausted remains
committed and undelivered. The old witness delivery meaning is unchanged.

## 6. Evidence, staging and replay

New artifact format: `statetwin.dev/agent-evidence-offline/v1alpha1`, independent
of old EpisodeEvidence and Journal. It contains format, base64 original verified
Bundle and the bounded new AgentEpisode. It is synthetic, local and unsigned.

The whitelist includes run definition, business event inputs/results/error
classes, authority/dispatch/commit/delivery facts, request frontiers, usage
counters, initial/final world states and evaluator result. Provider raw response
bodies, request bodies, private call IDs, headers, URLs, tokens and opaque
continuation are not retained. Known sensitive patterns reject content **before
writing**; all retained Bundle members, including unused scenarios, are checked.
This is finite pattern detection plus synthetic-source policy, not universal
secret discovery or proof that user-provided data is truly synthetic.

Lifecycle for retained evidence:

1. Validate local inputs and claim a new directory exclusively.
2. Write `claim.json`; an existing directory always refuses reuse.
3. Run the sequential loop, stop admission and inspect the terminal world.
4. Grade the bounded read-only view. For completed runs, independently replay
   the closed world and check events, state and grading.
5. Write and sync `closure.json` before disposing the original world.
6. Close owned resources; retain actual cleanup status.
7. Write/sync `terminal.pending.json`, atomically hard-link it as a new
   `terminal.json`, then remove only the owned staging files.

No rename-overwrite fallback is allowed. Unsupported hard links or failed
publication return an error with staging retained. Interrupted/malformed
directories are inspect-only; no automatic model resume or best-attempt retry.
This is not a claim of power-loss durability on every filesystem; directory
entry durability and OS-specific failure injection remain explicit limitations.

Only completed execution with checked replay closure and confirmed cleanup may
set `evidenceStatus:complete` and `worldReplayable:true`. `task_failed` or
`policy_violation` can have complete evidence; valid evidence does not mean a
successful task. Canceled/budget/protocol/inspection/cleanup failures are partial,
even if an observed goal predicate was true. Missing unsafe content is not
silently replaced to create an equivalent replay.

Verification rejects unknown fields, invalid formats/profiles, runtime mismatch,
corrupt Bundle, missing events, wrong sequence/authority/commit/error/result,
incorrect initial/final state, invalid request frontiers/counts, missing delivery
for a completed run, inconsistent scoring and partial status. Replay uses the
same official MCP world path, with zero model requests. It verifies self-consistent
world evidence, not provider provenance, human review, runtime-source authenticity
or a model's reasoning. Fully fabricated internally consistent unsigned artifacts
are outside the authenticity claim. Reported content-byte usage is bounded, not
recomputed from omitted private transport bodies.

## 7. Fixed-plan comparison

Plan format: `statetwin.dev/agent-compare-offline/v1alpha1`, at most 64 KiB and
32 pairs. Each pair fixes Task ID, repeat 1–16, baseline/candidate trial ID and
relative `terminal.json` paths. Duplicate trial IDs, paths (case folded for
cross-platform safety) and Task/repeat pairs are rejected. No retry selector.
The only accepted `allowedDifferences` is `["model"]`, with distinct `mock-`
labels fixed before invoking the two runs.

After verifying both artifacts, comparison removes only per-trial ID and the
declared model-label variable from the complete definitions and compares their
canonical values. Oracle, Bundle, policy, budgets, projection, output-token
configuration or runtime differences are incomparable, not model regressions.
This is a bounded ComparisonKey equivalent, not a general experimental-variable
language or a new hash-manifest format.

Counts remain auditable:

- `planned = notStarted + started`;
- `started = incomplete + terminal`;
- `validlyEvaluated` is an independent verified subset.

Missing directories are not-started. Existing directories without readable valid
terminal records are incomplete/invalid. Recognized terminal failures remain
terminal but unverified/partial. Invalid or missing evidence never disappears
from the denominator. Model/Task/trial identity substitution is incomparable.

A paired task-success loss or increase in verified unauthorized attempts is a
regression. Otherwise the valid, comparable pair is `no_regression_observed`.
Missing/partial/corrupt evidence is inconclusive; definition mismatch is
incomparable. Summary priority is regression, then incomparable, then inconclusive,
then no-regression-observed; every pair remains visible. `upgradeAllowed` is
always false for this synthetic lane. No significance, live pricing or automatic
upgrade claim is derived from a small mock sample.

## 8. Executable validation and remaining gates

Tests are in `internal/agenthost`, `internal/agenteval` and
`cmd/statetwin/agent_eval_test.go`. They cover the six mock tasks, request shape,
private continuation/session isolation, strict batches and IDs, count/byte/time
admission, concurrent governor stop/count bounds, committed prefixes, undelivered
results, privacy refusal, no-clobber publication, tamper/replay, complete failed
tasks, duplicate/missing plans and regression/incomparable decisions.

The [offline guide](guides/OFFLINE-AGENT-REGRESSION.md) is exercised in a clean
temporary root by the CLI integration test. No real credentials or API calls are
needed in ordinary CI. Exact-commit CI status belongs in the implementation ledger.

Still open: real API transport/approved live matrix and cost policy; independent
Task/oracle review and external user trial; broader stage-by-stage storage/disk
failure injection and cleanup recovery; signed provenance, remote reconciliation,
native/product HostProfiles and general configuration comparison. These must
not be marked implemented merely because this offline slice passes.
