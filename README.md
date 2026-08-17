# MCP State Twin

> Fork the tool world, not production.

MCP State Twin is a proposed open-source runtime for building deterministic,
forkable, stateful test worlds behind Model Context Protocol (MCP) tools.

The intended use is agent engineering: evaluate a tool-using agent, compare
models or prompts, reproduce failures, and inject faults without letting the
agent modify a production system of record.

> [!IMPORTANT]
> **This repository is currently a design package, not working software.** It
> contains an implementation RFC, accepted architecture decisions, a failure
> analysis, and a roadmap. It does not yet contain a runtime, CLI, Go module,
> MCP server, tests, releases, packages, or provider integration results.

## Repository status

| Area | Current state |
|---|---|
| Product definition | Drafted in [RFC-0001](docs/RFC-0001.md) |
| Protocol baseline | Accepted for v0.1 in [ADR-0001](docs/ADR-0001-PROTOCOL-BASELINE.md) |
| Data/control-plane isolation | Accepted for v0.1 in [ADR-0002](docs/ADR-0002-CONTROL-PLANE-ISOLATION.md) |
| Failure analysis | 100 design inputs catalogued; not a completed test report |
| Runtime and CLI | Not implemented |
| TwinSpec parser or schema | Not implemented; `v1alpha1` syntax is still proposed |
| MCP conformance | Not run |
| ChatGPT / OpenAI integration | Design target; not tested |
| Claude / Anthropic integration | Design target; not tested |
| Production readiness | No |

The project name is also provisional until repository, package, and trademark
checks are completed.

## The problem

A stateful tool call changes what later calls should observe.

Consider an issue-tracker agent that:

1. reads a repository and an issue;
2. creates a comment;
3. retries after an ambiguous timeout;
4. reads the issue again;
5. closes it only if the expected state is present.

A static mock can return plausible JSON, but it does not necessarily preserve
the relationships between those calls. A cassette can replay a recorded path,
but a different model or prompt may take a new valid path. Running every trial
against a live service creates real side effects, shared-state interference,
rate limits, cost, and irreproducible starting conditions.

MCP State Twin is intended to put an explicit state machine behind the same
agent-facing tool contract:

```text
                           production service
                                   X
                                   |
                         no hidden live writes
                                   |
                                   |
    agent under test -- MCP tools --> data plane
                                      |
                              +-------v--------+
                              | transition     |
                              | engine         |
                              |                |
                              | virtual clock  |
                              | seeded faults  |
                              | state store    |
                              +-------+--------+
                                      |
                              isolated branch

    test harness -----------> separate control plane
                              snapshot / fork / reset /
                              inspect / diff / fault setup
```

The data plane is the world visible to the agent. The control plane owns test
administration and hidden assertions. They are separate trust domains.

## Scope

The proposed runtime owns:

- a versioned, reviewable contract for modeled tool state and transitions;
- deterministic state execution for an ordered sequence of tool calls;
- immutable snapshots and isolated branches;
- reset, import, export, and canonical state diff;
- virtual time and explicitly seeded fault schedules;
- upstream tool-surface capture and drift detection;
- state-based scenario assertions;
- an MCP tool data plane and a separate private simulation control plane.

It does **not** own:

- agent planning, orchestration, memory, RAG, or model inference;
- production transaction management or action governance;
- a universal MCP gateway;
- a claim of behavioral equivalence to an undocumented upstream service;
- automatic discovery of every business rule from traces;
- identical behavior from different models or hosts;
- a replacement for MCP conformance testing or general API test tools.

## Core abstraction: TwinSpec

TwinSpec is the proposed intermediate representation for a modeled tool world.
It is meant to describe more than `function(args) -> JSON`:

```text
tool behavior = reads
              + preconditions
              + state effects
              + postconditions
              + result or error mapping
              + time semantics
              + idempotency semantics
```

The following is an **illustrative design sketch**, not an accepted schema and
not something the current repository can execute:

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: Twin

metadata:
  name: issue-tracker-example
  upstream:
    protocol: mcp
    surfaceDigest: "sha256:<captured-tool-surface-digest>"
  fidelity:
    level: L1
    status: unverified

clock:
  mode: virtual
  initial: "2026-08-01T00:00:00Z"

state:
  entities:
    repository:
      key: [owner, name]
    issue:
      key: [repository, number]

tools:
  - name: create_issue
    reads:
      - "repository(input.owner, input.repository)"
    preconditions:
      - "repository.exists == true"
    effects:
      - "insert issue"
    postconditions:
      - "created_issue.number > 0"
    errors:
      - when: "repository.exists == false"
        code: NOT_FOUND
```

The final format still requires ADRs for the bounded expression language,
canonical artifact format, and storage/snapshot strategy. Arbitrary embedded
scripts are outside the v0.1 design because they would weaken determinism,
static analysis, and supply-chain safety.

## Determinism contract

The environment—not the language model—is intended to be deterministic.

```text
E = (runtime version,
     TwinSpec digest,
     snapshot digest,
     scenario seed,
     ordered tool calls)

execute(E) -> (ordered structured results, final state digest)
```

For a supported runtime/platform combination, equal `E` should produce equal
structured results and final state. Time, generated identifiers, pagination
cursors, random sampling, and injected faults must therefore come from
deterministic providers.

This contract does not say that two LLM runs will emit the same tool calls.
Each trial instead starts from the same immutable snapshot, and correctness is
primarily evaluated from terminal state and declared invariants.

## Required invariants

These are implementation requirements from the RFC, **not claims that the
current repository has already verified them**:

1. Hermetic mode has no hidden live writes.
2. Snapshot, reset, fault, and hidden-state controls are absent from the
   default agent-visible `tools/list`.
3. Modeled transitions obey the determinism contract above.
4. Unknown behavior fails explicitly; it is never replaced with plausible
   invented data.
5. A twin is bound to a canonical upstream tool-surface fingerprint, including
   model-facing descriptions where available.
6. A normal single-tool transition is atomic unless an explicit, reproducible
   partial-effect fault is being simulated.
7. Every scenario run owns an isolated branch namespace.
8. A model is never the sole correctness oracle.
9. Secrets and raw credentials stay out of specs, traces, logs, and exports.
10. Errors retain their canonical evidence; the runtime does not silently turn
    unknowns, validation failures, or timeouts into success.

See [RFC-0001 §5](docs/RFC-0001.md#5-hard-invariants) and the
[failure-mode matrix](docs/FAILURE-MODE-MATRIX.md) for the normative detail.

## Fidelity is explicit

“Twin” does not mean “perfect copy.” Every artifact is expected to declare what
evidence supports it and which behavior is not modeled.

| Level | Evidence | Appropriate use |
|---|---|---|
| L0 — Cassette replay | Recorded request/response matching | Exact-path smoke and regression tests |
| L1 — Stateful template | Explicit entities and reviewed basic transitions | Development and exploratory workflow tests |
| L2 — Contract-backed | Human-reviewed rules, invariants, differential tests, upstream fingerprint | CI/evaluation within declared coverage |
| L3 — Native/reference | Shared or domain-provided reference logic with a stated correspondence boundary | High-fidelity domain simulation |

An inferred or generated TwinSpec starts unverified. It cannot be promoted to
L2 or L3 merely because an LLM produced convincing rules. Promotion requires
reviewable evidence and tests, and even an L2 twin must publish its uncovered
behavior.

## How this differs from adjacent tools

These categories overlap and can be complementary. The distinction below is
about the primary abstraction, not a claim that every project in a category
lacks every other feature.

| Approach | Primary artifact | Best at | Boundary relevant here |
|---|---|---|---|
| Record/replay | Interaction cassette | Reproducing previously recorded calls | A new trajectory may have no matching recording |
| Static MCP mock | Tool definitions and generated responses | Client development and isolated happy paths | Cross-call state and business invariants may be incomplete |
| Hand-built benchmark sandbox | Benchmark-specific environment | Evaluating a fixed task collection | Reuse for a developer-owned upstream surface is not the main abstraction |
| Domain digital twin | Domain-specific reference model | High-fidelity simulation in one domain | Not a generic MCP tool-world contract |
| MCP State Twin | Versioned state-transition contract plus forkable world state | New multi-step trajectories from identical snapshots | Fidelity is limited to explicitly modeled and validated behavior |

Record/replay is intentionally retained as the proposed L0 fidelity level. It
is not treated as a competing technology that must be replaced.

The full prior-art screen and the directions deliberately rejected by this
project are documented in [Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md).
The document is a dated research snapshot, not proof that no similar public or
private project exists.

## MCP and provider compatibility

MCP is the proposed provider-facing boundary. Model-specific SDKs do not belong
inside the deterministic state core.

The v0.1 design baseline is MCP `2026-07-28` with a tools-first surface through
the official Go SDK. The tools-first decision reflects a practical common
denominator: OpenAI documents remote MCP use and ChatGPT Developer mode access
to read/write tools, while Anthropic documents remote MCP tool calls and notes
that its Messages API connector currently supports only the tool-call subset of
MCP.

| Host path | External capability documented by the provider | Project status |
|---|---|---|
| ChatGPT Developer mode | Remote MCP tools, including read/write tools | Planned smoke test; no result yet |
| OpenAI Responses API | Remote MCP server tools | Planned integration test; no result yet |
| Anthropic Messages API | Remote MCP tool calls over HTTP | Planned smoke test; no result yet |
| Generic MCP client | Compatible negotiated tool subset | Planned conformance and smoke tests |

Protocol compatibility means that a host can list and call the modeled tools.
It does not mean that ChatGPT, Claude, Codex, or another agent will select the
same tools, follow the same trajectory, or reach the same result.

Because there is no server implementation in this repository yet, **none of
these integrations can currently be installed or exercised from this repo**.

## Security and evaluation integrity

The proposed security boundary is structural:

- the agent-facing MCP endpoint exposes only simulated application tools;
- snapshot, fork, reset, state inspection, virtual-time control, fault setup,
  and hidden assertions live on a separately authenticated control plane;
- prompt instructions are not treated as access control;
- hermetic CI is expected to enforce network egress denial in addition to
  application-level safeguards;
- recorder artifacts, if recording is implemented, are treated as sensitive
  and untrusted input;
- synthetic fixtures are preferred over production recordings.

The project does not claim that structured business data can always be
irreversibly anonymized. Users must also verify that recording a third-party
service is permitted by its terms and their privacy obligations.

See [ADR-0002](docs/ADR-0002-CONTROL-PLANE-ISOLATION.md) and the
[P0/P1 failure modes](docs/FAILURE-MODE-MATRIX.md).

## Proposed implementation direction

The RFC currently recommends, but the repository has not yet implemented:

- Go and a single-binary `statetwin` CLI;
- the official Model Context Protocol Go SDK;
- SQLite as the first transactional state backend;
- YAML authoring with canonical JSON hashing for portable artifacts;
- a bounded declarative expression language;
- optional OpenTelemetry integration;
- a GitHub-like issue workflow as the first reference twin;
- a second independent stateful domain before v0.1 release.

No package version, CLI command, storage layout, or TwinSpec syntax should be
treated as stable until code, tests, and the remaining ADRs land.

## Roadmap and release gates

Implementation is deliberately ordered around the smallest credible loop:

1. freeze the open ADRs and map every hard invariant to a test design;
2. build and fuzz the deterministic kernel;
3. implement immutable snapshots, isolated forks, import/export, and diff;
4. expose the modeled tools through the official MCP SDK;
5. implement and test the separate control plane;
6. add safe upstream inspection, redaction, and drift detection;
7. ship at least one useful reference twin with state-based scenarios;
8. run conformance and cross-provider smoke tests;
9. add trace-assisted bootstrap only after the deterministic core is trusted;
10. publish measured fidelity and differential-validation reports.

The v0.1 label is blocked until the release gates in
[RFC-0001 §27](docs/RFC-0001.md#27-launch-gates-for-v01) and the
[engineering roadmap](docs/ROADMAP.md#v01-release-gate) are satisfied. In
particular, performance targets in the RFC are targets—not benchmark results.

## Installation and quick start

There is no installation or quick-start command yet. Any command shown in the
RFC is a proposed interface and will fail against the current repository.

The first honest quick start can be added only after a reproducible build can:

1. validate a versioned TwinSpec;
2. start an MCP data plane against a synthetic fixture;
3. create two isolated branches from the same snapshot;
4. execute different tool-call sequences;
5. produce canonical final-state digests and a stable diff;
6. prove that no production write path was reachable in hermetic mode.

## Contributing at the current stage

The most useful contributions now are design review and falsification:

- identify a violated or missing hard invariant;
- map a P0/P1 failure mode to a reproducible test design;
- challenge TwinSpec semantics with a concrete stateful API workflow;
- provide stronger prior art that changes the project positioning;
- propose an ADR for the expression engine, snapshot strategy, or canonical
  artifact format;
- define observable fields for a safe differential test against a disposable
  fixture service.

Claims such as “perfect simulation,” “one-click twin for any SaaS,” or “the
first stateful MCP sandbox” are intentionally out of scope. A useful review
should instead state the behavior, evidence, fidelity boundary, and failure
mode precisely.

## Documentation map

| Document | Role | Status |
|---|---|---|
| [RFC-0001](docs/RFC-0001.md) | Product boundary, invariants, architecture, semantics, testing, release gates | Draft for implementation |
| [ADR-0001](docs/ADR-0001-PROTOCOL-BASELINE.md) | MCP protocol baseline and provider neutrality | Accepted for v0.1 |
| [ADR-0002](docs/ADR-0002-CONTROL-PLANE-ISOLATION.md) | Data-plane/control-plane isolation | Accepted for v0.1 |
| [Failure Mode Matrix](docs/FAILURE-MODE-MATRIX.md) | 100 risks and required responses | Design input, not test evidence |
| [Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md) | Prior art and rejected project directions | Research snapshot dated 2026-08-17 |
| [Roadmap](docs/ROADMAP.md) | Ordered implementation phases and exit criteria | Pre-implementation plan |

For design semantics, the RFC and accepted ADRs take precedence over this
README. The roadmap is a plan, not a delivery claim.

## Source baseline

The protocol/provider statements above were checked against these primary
sources on **2026-08-17**:

- [MCP Specification 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28)
- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [OpenAI: ChatGPT Developer mode](https://developers.openai.com/api/docs/guides/developer-mode)
- [OpenAI: MCP servers for plugins and API integrations](https://developers.openai.com/api/docs/mcp)
- [Anthropic: MCP connector](https://platform.claude.com/docs/en/agents-and-tools/mcp-connector)

The broader research basis and adjacent OSS are listed in
[Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md). Those references
support the problem framing; they do not prove this proposed implementation or
its future adoption.

## License

[MIT](LICENSE)
