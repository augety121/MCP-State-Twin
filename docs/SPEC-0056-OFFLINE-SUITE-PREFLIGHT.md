# SPEC-0056: Offline Suite Plan and Frozen Preflight

Status: accepted bounded subset by [ADR-0056](ADR-0056-BOUNDED-OFFLINE-SUITES.md).
Implements B09-offline/B10 orchestration of existing synthetic trials; no live lane.

## 1. Problem and delivered behavior

Today an operator manually writes a RunConfig and invokes mock for each trial,
then maintains a separate comparison plan. A suite must admit all inputs before
the first output directory or world is created, freeze what will run, and derive
one exact comparison plan. Preflight is not proof of model/task success.

`eval suite-preflight --root ROOT --suite RELATIVE_PLAN` is read-only. `eval suite`
uses the same admission before execution. No endpoint, credential, provider,
shell, parallelism, retry, resume or runtime-control parameter is accepted.

## 2. Closed schema

Format: `statetwin.dev/agent-suite-offline/v1alpha1`. One strict YAML/JSON document,
64 KiB, depth 16, known fields, no aliases/anchors/explicit tags/duplicate keys.

| Field | Requirement |
|---|---|
| format | exact format above |
| baselineModel / candidateModel | distinct existing mock-label grammar |
| maxOutputTokens | explicit original integer, 1–8192, shared by both sides |
| pairs | ordered 1–16 pairs |
| pair.taskId | existing comparison Task ID grammar; exact loaded Task ID |
| pair.task | root-relative existing AgentTask file |
| pair.repeat | explicit original integer 1–16; unique Task ID/repeat |
| pair.baselineResponses / candidateResponses | root-relative MockResponses files |

Generated trial IDs are `baseline-01`, `candidate-01`, etc. Pair position, not
repeat number, determines the suffix. They are unique within one suite directory.
Comparison paths are `<trialId>/terminal.json`, relative to that directory.
Only `model` is an allowed comparison difference. No inferred best attempt.

All referenced paths follow existing portable-path and rooted regular-file
checks. Observed symlink components fail. The chosen root and its writers must
be trusted/quiescent; this is not hostile-filesystem or atomic-input isolation.

## 3. Preflight and input freezing

Process every pair in order: bounded Task decode, exact ID check, referenced
TwinBundle admission, both generated RunConfig admissions and tool projections,
MockResponses envelope decoding, and existing finite synthetic/secret checks.
Scan every Bundle member, including unused scenarios, before any output writes.
Refuse known sensitive patterns in scripts and typed Task values. Private frames
are used only in memory, never added to suite reports. Runtime-only response
protocol, tool authorization and task grading remain execution checks.

Retain private copies of admitted Task, Bundle bytes and scripts. Later source
file edits must not change that prepared run. The public summary exposes only
format, profile, model labels, planned pairs/trials, limits and the generated
comparison plan, never Task text, scripts or original input paths. Preparation
checks cancellation between bounded operations and propagates context failure.

The input resource profile `offline-suite-v1` admits at most 64 MiB cumulative
encoded member reads and 64 MiB cumulative extracted Bundle content, counted
per reference (including repeats), under existing per-file limits. These bounds
are not hard RSS quotas. Prepared data is not a chronological preregistration,
signature, model capability or source-attestation claim.

## 4. Errors and acceptance

Finite errors: `SUITE_PLAN_INVALID`, `SUITE_INPUT_INVALID`,
`SUITE_RESOURCE_LIMIT`, `DATA_POLICY_REJECTED`; context cancellation/deadline
remain recognizable. No source value/path/parser diagnostic is echoed.

Tests must cover last-pair failure causing zero writes, float/null/missing
numbers, duplicate Task/repeat, bounds, unsupported fields/models, Task-ID
substitution, unsafe paths, secret sentinels, deterministic generated plans,
all six existing tasks and frozen inputs after source mutation. No external calls.
