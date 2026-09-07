# SPEC-0038: AgentTask Offline Admission and Grading

- Status: Accepted bounded preview via ADR-0038
- Format: `statetwin.dev/agent-task/v1alpha1`; kind `AgentTask`
- Scope: B02/B03 structural authoring and offline witness evidence, not live Agents

## Format and admission

The exact field set is defined by `internal/task.Task`, with examples under
`examples/issue-tracker/agent-tasks`. Required fields: apiVersion/kind/format,
id/revision/domain, relative bundle path, objective, blind mode, tools,
authority, budgets, oracle, expectedOutcome. Context is optional.
An optional faultTool installs exactly one modeled after-commit response fault
for that visible tool in an otherwise fresh private world.

Task documents are at most 256 KiB with YAML depth 32. Unknown fields, anchors,
aliases, explicit tags and multiple documents are rejected. Existing formats
retain their original depth bounds. Tool count is 1–32; rules 1–64; assertions
2–64 including at least one goal and policy. IDs use bounded lowercase names.

Authority rules constrain named top-level scalar arguments of visible tools.
Rules are OR alternatives; equalities inside a rule are AND. Missing arguments
do not match; tool names and resource scope cannot be supplied by an endpoint
URL. Numeric integer equality accepts exactly representable JSON integers, not
rounded permission targets. Authority is checked before dispatch, with input
Schema validated by the runtime before local MCP execution.

Artifact paths are relative, portable and bounded. OpenRoot confines resolution;
observed symlink components and non-regular/oversized files are refused. The
source root is trusted: this is not protection against hostile in-root writers.
Bundle parsing reuses existing bounded archive admission. No schema/storage
migration or generated file-hash manifest is introduced by these commands.

`task validate` means structural admission plus declared surface availability,
oracle compilation and data-policy screening. It MUST NOT report solvability
or live compatibility as verified. Authority rules naming unknown input fields
fail admission. Semantic rule satisfaction is additionally exercised by witnesses.

## Read-only oracle

Assertions expose only before/after business entities and sequences, events and
answer. CEL is declarative with no file/network function. Each expression is at
most 4,096 bytes and has cost limit 10,000; interruption is checked during
comprehensions. Inputs are bounded and detached before evaluation, preserving
int64 values and null, so grading cannot mutate caller state.

Goals and policies are independent. False goal => task_failed; false policy or
unauthorized attempt => policy_violation. Evaluation errors/non-bool results =>
not_evaluated, never pass. Expected legal abstention is explicitly configured.
Runtime metadata such as head/call_count/audit is not business-state invariance.

Event sequence is zero-based contiguous. EffectCommitted requires dispatch;
Delivered also permits an application-generated permission refusal. The count
fields policyAttempts/blockedAttempts/committedViolations cover **authority
violations observed in the supplied events only**, not arbitrary business
policies or an inference about every possible Agent action. Predicate checks
carry independent policy outcomes. A forged caller-created view is not signed
or independently observed evidence.

## Witness execution and bounded guarantees

`TaskWitness` contains kind, taskId, syntheticOnly, calls and answer. It is
explicitly a scripted authoring witness. Maximum calls is the Task's bounded
toolAttempts (at most 32); no model is invoked. The runner creates an in-memory
SQLite world, uses real MCP request/result processing over an offline transport,
applies resource rules, gathers private commit/fault observations, reads the
terminal state and grades it. It owns no production connection or public port.

After-commit effects are linked to the store's actual fault event at that call
index, not inferred merely from the spelling of an error code. Client-visible
business error content is distinct from this private observation. A modeled
TIMEOUT_AFTER_EFFECT without a corresponding effect is not upgraded to a commit.

The report is `statetwin.dev/task-witness-report/v1alpha1`, source
`scripted-witness`. This is not AgentEvidence or ProviderSmokeReport. Reports
contain synthetic business state and input/output only after bounded sensitive
pattern checks; known secret/private-key/email patterns are refused, not copied
to error messages. Screening does not guarantee detecting every possible secret.

The offline runner is synchronous with one active call, bounded context and no
detached workers. On infrastructure, context or data-policy failure it returns
an error, not a sealed partial AgentEvidence; durable partial evidence belongs
to B06/B07 and remains unimplemented. Deferred MCP/store cleanup errors fail the
run. No recovery, live continuation, pricing or whole-process hard quota claim.

## Tests and command usage

Tests in task/evaluator/agenteval cover strict input, resource/path limits,
missing tools, private visibility, authorization, large integer preservation,
null, evaluator failures, six witnesses, wrong target, extra effects, missing
confirmation, extra legal reads, sensitive content and cleanup ownership.
They run in the ordinary cross-platform/race CI suite; a test's presence alone
does not prove it passed on a given commit.

From a repository build, using a fresh local output (never overwrite evidence):

```powershell
New-Item -ItemType Directory -Force examples/issue-tracker/.statetwin
statetwin bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
statetwin task validate --root examples/issue-tracker --task agent-tasks/close-issue.json
statetwin task witness --root examples/issue-tracker --task agent-tasks/after-commit-confirm.json --witness agent-witnesses/after-commit-confirm.json
```

The ordinary root CLI error path is retained: zero for a passing witness;
nonzero for failed evaluation or execution. Rich eval compare exit-code
contracts are not introduced. No `eval run`, model key or paid call is hidden
behind this command. Restore from source control to remove the new experimental
entry points; existing stored worlds and historical artifacts need no migration.
