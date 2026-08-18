# MCP State Twin

[![CI](https://github.com/augety121/MCP-State-Twin/actions/workflows/ci.yml/badge.svg)](https://github.com/augety121/MCP-State-Twin/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> Fork the tool world, not production.

MCP State Twin is an experimental open-source environment layer for agent
evaluation: deterministic, forkable, stateful test worlds behind Model Context
Protocol (MCP) tools.

It lets agent engineers start multiple runs from the same world snapshot,
allow each run to take a different valid tool trajectory, and compare terminal
state without writing to a production service.

It is not an AGI system, agent framework, memory system, RAG service, or model
runtime. The intended AGI-facing primitive is a reproducible external world in
which agents can act, fail, and be compared without production side effects.

```text
                         immutable snapshot S0
                                  |
                   +--------------+--------------+
                   |              |              |
                fork A         fork B         fork C
                   |              |              |
               Agent A        Agent B        Agent C
                   |              |              |
                   +------ MCP State Twin -------+
                                  |
                    isolated state transitions
                                  |
                      canonical terminal diff

                     production writes: none
```

## Status

**Development preview (`0.1.0-dev`), no tagged release.** The repository now
contains a working Go CLI and runtime, but it is not production-ready and has
not completed the RFC's v0.1 release gates.

Implemented and exercised locally:

- strict TwinSpec `v1alpha1` YAML decoding, structural validation, and
  hermetic JSON Schema 2020-12 input/output compilation;
- canonical SHA-256 digests for specs, MCP tool surfaces, and world state;
- upstream binding admission that fails closed for mismatched `current`,
  `drifted`, or `unknown` surfaces;
- 4,096-byte, cost-limited CEL expressions for preconditions, effects, queries,
  postconditions, and global invariants;
- SQLite-backed atomic transitions, versioned database identity, tool-call
  audit, and transactional control-operation audit;
- immutable logical snapshots, isolated forks, reset, and canonical state
  diff;
- stateless Streamable HTTP MCP data plane through the official Go SDK;
- separately authenticated HTTP control plane;
- six-tool issue-tracker reference twin with synthetic state;
- unit, deterministic replay, MCP HTTP, authorization, output-rollback,
  migration-refusal, and 100-fork isolation tests;
- pinned official MCP conformance checks for initialize, ping, tools-list, and
  JSON Schema 2020-12 on Linux CI;
- bounded TwinSpec/CEL fuzz targets plus pinned secret-policy and loopback-only
  hermetic CI gates (all passed in Linux CI run #6);
- a tested CLI loop: initialize → snapshot → fork twice → mutate each fork →
  diff terminal state.
- a bounded Scenario v1alpha1 runner with deterministic environment identity,
  ordered tool traces, JSON Pointer state assertions, and canonical state diff.

Not yet implemented or verified:

- recorder, cassette replay, trace redaction, or automatic upstream surface
  inspection/refresh;
- deterministic fault injection or virtual-clock advancement;
- live ChatGPT, OpenAI API, Claude, or Claude Code smoke tests;
- differential validation or an L2 fidelity promotion workflow;
- data-plane authentication, TLS, remote multi-tenancy, or a security audit.

See [Implementation Status](docs/IMPLEMENTATION-STATUS.md) for the evidence and
the exact partial boundaries. Roadmap items are not presented as current
features.

## Why this exists

Serious tool-using agent tests need more than plausible JSON.

An issue-tracker agent might read an issue, create a comment, retry after an
ambiguous timeout, read the issue again, and close it only if the expected
state exists. Every call changes what later calls should observe.

Common approaches solve related but different problems:

| Approach | Primary strength | Boundary for agent evaluation |
|---|---|---|
| Record/replay | Reproduce a previously captured path | A new model may take a valid path that was never recorded |
| Static MCP mock | Isolate a client and return controlled data | Cross-call state, constraints, and failure semantics may be incomplete |
| Hand-built benchmark sandbox | Evaluate one curated task collection | Reuse for a developer-owned tool surface is not the primary abstraction |
| Live test/production service | Exercise real behavior | Side effects, rate limits, cost, shared-state pollution, and irreproducible starts |
| MCP State Twin | Execute explicit transitions on forkable world state | Fidelity is limited to behavior that has been modeled and validated |

Record/replay is planned as the L0 fidelity mode; it is complementary rather
than something this project needs to displace.

## Quick start

Requirements:

- Go 1.26.x
- Git

Clone the repository and validate the reference TwinSpec:

```bash
go mod download
go run ./cmd/statetwin validate \
  --spec examples/issue-tracker/twin.yaml
```

Create an isolated database and base snapshot:

```bash
go run ./cmd/statetwin init \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --db demo.db \
  --branch main \
  --snapshot base
```

Fork the same world twice:

```bash
go run ./cmd/statetwin fork --db demo.db --snapshot base --branch run-a
go run ./cmd/statetwin fork --db demo.db --snapshot base --branch run-b
```

Take different valid trajectories:

```bash
go run ./cmd/statetwin call \
  --spec examples/issue-tracker/twin.yaml \
  --db demo.db \
  --branch run-a \
  --tool create_issue \
  --input '{"owner":"octo","repository":"demo","title":"Fork A","body":"Created only in A"}'

go run ./cmd/statetwin call \
  --spec examples/issue-tracker/twin.yaml \
  --db demo.db \
  --branch run-b \
  --tool close_issue \
  --input '{"owner":"octo","repository":"demo","number":1}'
```

Compare terminal worlds:

```bash
go run ./cmd/statetwin diff \
  --db demo.db \
  --before run-a \
  --after run-b
```

The diff uses stable JSON Pointer paths. Object keys containing `/` are escaped
according to JSON Pointer rules, so a key such as `octo/demo#1` appears as
`octo~1demo#1`.

## Run a reproducible scenario

Execute the bundled state-scored scenario:

```bash
go run ./cmd/statetwin scenario \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --scenario examples/issue-tracker/scenario-close-issue.yaml
```

The command exits non-zero on an unexpected error class or failed assertion.
Its JSON report includes the environment digest, ordered tool trace, initial and
terminal state digests, assertion evidence, and canonical state diff. The
current runner identifies itself as `scripted-scenario`; this is deliberately
not presented as a live Codex, OpenAI, Claude, or other model evaluation.

Scenario reports contain tool inputs and results. Use synthetic fixtures only;
do not commit reports containing credentials, production traces, or personal
data.

## Run the MCP server

Set a control-plane token. Do not commit it.

```bash
export STATETWIN_CONTROL_TOKEN='replace-with-a-local-secret'
```

PowerShell:

```powershell
$env:STATETWIN_CONTROL_TOKEN = 'replace-with-a-local-secret'
```

Start the runtime:

```bash
go run ./cmd/statetwin serve \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --db demo.db
```

Default endpoints:

| Plane | Endpoint | Visible operations |
|---|---|---|
| Agent data plane | `http://127.0.0.1:8090/mcp/main` | Only modeled business tools |
| Private control plane | `http://127.0.0.1:8091/v1` | Branch state, snapshot, fork, reset, diff |

The branch ID is part of the MCP URL, not an extra model-visible tool argument.
This keeps the tool input schema identical across branches.

> [!WARNING]
> The current data plane has no authentication or TLS. Both servers bind to
> loopback by default and should remain local. The control token is only one
> development safeguard; it does not make this build safe for Internet
> exposure.

## Reference twin

The included issue-tracker world exposes these agent-visible tools:

- `get_repository`
- `list_issues`
- `get_issue`
- `create_issue`
- `add_comment`
- `close_issue`

Snapshot, fork, reset, diff, state inspection, and future fault controls are
not MCP tools and do not appear in `tools/list`. An integration test connects
with the official MCP Go client and verifies this boundary.

The reference twin is synthetic, `L1`, `unverified`, and `unbound` to an
upstream service. It must not be described as a GitHub-equivalent environment.

## TwinSpec

TwinSpec is the versioned, reviewable contract for a modeled tool world. A tool
is more than `function(args) -> JSON`:

```text
tool behavior = input contract
              + reads and preconditions
              + deterministic state effects
              + postconditions and global invariants
              + structured result or typed error
              + time and idempotency semantics
```

Excerpt from the executable reference spec:

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: Twin

metadata:
  name: issue-tracker
  upstream:
    protocol: mcp
    status: unbound
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
  - name: close_issue
    description: Close an existing open issue in the isolated simulated repository.
    preconditions:
      - expr: "state.entities.issue[input.owner + '/' + input.repository + '#' + string(input.number)].state == 'open'"
        code: CONFLICT
        message: issue is already closed
    effects:
      - op: update
        entity: issue
        key: "input.owner + '/' + input.repository + '#' + string(input.number)"
        merge: true
        value: "{'state': 'closed', 'closedAt': clock}"
```

The complete file is
[examples/issue-tracker/twin.yaml](examples/issue-tracker/twin.yaml).

### Expression boundary

TwinSpec expressions use `cel-go`. Source text is limited to 4,096 UTF-8 bytes,
compiled at load time, and evaluated with a cost limit of 10,000. Expressions
receive only JSON-shaped variables:

- `input`
- `state`
- `vars`
- `item`
- `clock`
- `call_index`

No filesystem, process, network, reflection, or arbitrary Go functions are
registered. Native extensions are not supported in `v1alpha1`.

### JSON Schema boundary

Tool inputs and successful outputs are compiled and validated as JSON Schema
Draft 2020-12. Format assertions are enabled. Local `$defs` and fragments are
supported, while references that require an external network or filesystem
resource fail at startup. An invalid declared output rolls back the transition
and returns `INTERNAL_TWIN_ERROR`.

### Current effect operations

| Operation | Semantics |
|---|---|
| `allocate` | Increment a named deterministic sequence and bind the result to `vars` |
| `insert` | Insert one keyed entity; conflict if it exists |
| `update` | Replace or merge one keyed entity; fail if missing |
| `delete` | Delete one keyed entity; fail if missing |

## Determinism contract

The environment—not the language model—is deterministic:

```text
E = (runtime version,
     TwinSpec digest,
     snapshot digest,
     scenario seed,
     ordered tool calls)

execute(E) -> (ordered structured results, final state digest)
```

Current code virtualizes state allocation and time exposed to expressions. A
test replays a 1,000-call corpus on two branches and checks equality after
every transition and at the final state.

The model can still choose different calls. That is expected. Each comparison
run starts from the same immutable snapshot, while success is evaluated from
terminal state and declared invariants rather than exact trajectory equality.

## Error and transaction semantics

Normal tool transitions run inside one SQLite transaction:

```text
load branch head
  -> validate input
  -> evaluate preconditions
  -> apply effects to an isolated working state
  -> evaluate query, postconditions, and global invariants
  -> commit state and audit record atomically
```

Failed domain outcomes keep the prior state digest and still append a tool-call
audit record. The implemented canonical error classes include:

- `INVALID_INPUT`
- `PRECONDITION_FAILED`
- `NOT_FOUND`
- `CONFLICT`
- `INVARIANT_VIOLATION`
- `UNMODELED_BEHAVIOR`
- `INTERNAL_TWIN_ERROR`

Timeout-before-effect, timeout-after-effect, partial-effect, rate-limit, and
eventual-consistency faults remain specified but are not implemented yet.

SQLite files carry the State Twin application ID and an explicit schema
version. Snapshots persist that storage schema version and bind it into their
IDs. Foreign databases and versions newer than the runtime are rejected.
Snapshot, fork, and reset write a separate control-audit row in the same
transaction as the privileged mutation; bearer tokens and HTTP headers are not
recorded.

## Fidelity levels

“Twin” does not mean “perfect copy.” Fidelity must be declared, bounded, and
supported by evidence.

| Level | Meaning | Intended use |
|---|---|---|
| L0 — Cassette replay | Match recorded interactions | Exact-path smoke/regression tests |
| L1 — Stateful template | Explicit entities and reviewed basic transitions | Development and exploratory workflow tests |
| L2 — Contract-backed | Human-reviewed rules, invariants, differential tests, upstream fingerprint | CI/evaluation within declared coverage |
| L3 — Native/reference | Shared or domain-provided reference logic | High-fidelity domain simulation |

Generated or inferred behavior cannot promote itself to L2/L3. The current
reference twin is L1 and unverified.

## MCP and model providers

The core integrates with MCP, not with model-provider SDKs. This is deliberate:
the project can provide one tool surface without claiming different models will
select the same tools or follow the same trajectory.

The automated integration test covers the official Go SDK as both server and
client over stateless Streamable HTTP. Linux CI also runs the official MCP
conformance framework `v0.1.16` for initialize, ping, tools-list, and JSON
Schema 2020-12. That framework currently exercises protocol versions through
`2025-11-25`; it does not prove every feature of the design baseline
`2026-07-28`.

OpenAI documents remote MCP
servers and ChatGPT Developer mode read/write tools; Anthropic documents remote
MCP tool calls and currently limits its Messages API connector to the tool-call
subset. Those external capabilities motivated the tools-first design, but this
repository has **not** run live provider smoke tests yet.

Design-source links:

- [MCP Specification 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28)
- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [OpenAI: ChatGPT Developer mode](https://developers.openai.com/api/docs/guides/developer-mode)
- [OpenAI: MCP servers for plugins and API integrations](https://developers.openai.com/api/docs/mcp)
- [Anthropic: MCP connector](https://platform.claude.com/docs/en/agents-and-tools/mcp-connector)

## CLI

```text
statetwin validate   validate structure, CEL, and print the spec digest
statetwin init       initialize a branch and optional immutable snapshot
statetwin call       execute one tool directly against a branch
statetwin state      inspect canonical branch state
statetwin snapshot   create an immutable logical snapshot
statetwin fork       create an isolated branch from a snapshot
statetwin diff       compare two branch states
statetwin serve      run separate MCP data and HTTP control planes
statetwin version    print the development version
```

CLI output is structured JSON except for server logs and fatal diagnostics.

## Test and build

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/statetwin
```

The latest local verification was performed on Windows amd64 with Go 1.26.5.
GitHub Actions runs formatting, vet, race-enabled tests, coverage collection,
and build on Linux after the repository is published.

## Architecture boundaries

The project intentionally separates two trust domains:

### Agent data plane

- exposes only business tools declared by TwinSpec;
- binds a branch through the URL;
- never advertises hidden expected state or test controls;
- returns typed domain failures as MCP tool errors the model can observe.

### Simulation control plane

- requires an independent bearer token;
- supports branch state, snapshot, fork, reset, and diff;
- binds to a different loopback address/port by default;
- is operated by a test harness or human, not the agent under test.

Prompt instructions are not an authorization boundary.

## Documentation

| Document | Purpose |
|---|---|
| [Implementation Status](docs/IMPLEMENTATION-STATUS.md) | Evidence-backed implemented/partial/missing matrix |
| [RFC-0001](docs/RFC-0001.md) | Product boundary, hard invariants, architecture, semantics, and release gates |
| [RFC-0002](docs/RFC-0002-V0.1-RELEASE-PROFILE.md) | Normative first-release profile, limits, traceability, and gates |
| [SPEC-0001](docs/SPEC-0001-TWINSPEC-CORE.md) | TwinSpec v1alpha1 data model and admission rules |
| [SPEC-0002](docs/SPEC-0002-RUNTIME-SEMANTICS.md) | Determinism, transactions, snapshots, errors, and limits |
| [SPEC-0003](docs/SPEC-0003-MCP-BOUNDARIES-AND-COMPATIBILITY.md) | MCP data/control planes, hermetic mode, and provider neutrality |
| [SPEC-0004](docs/SPEC-0004-EVIDENCE-FIDELITY-AND-RELEASE.md) | Evidence, provenance, fidelity, and release gates |
| [SPEC-0005](docs/SPEC-0005-SCENARIO-AND-REPORT.md) | Bounded scenarios, state assertions, environment identity, and evidence report |
| [ADR-0001](docs/ADR-0001-PROTOCOL-BASELINE.md) | MCP protocol baseline and provider neutrality |
| [ADR-0002](docs/ADR-0002-CONTROL-PLANE-ISOLATION.md) | Data/control-plane isolation |
| [ADR-0003](docs/ADR-0003-EXPRESSION-ENGINE.md) | Bounded CEL expressions |
| [ADR-0004](docs/ADR-0004-STORAGE-AND-SNAPSHOTS.md) | SQLite and logical snapshot strategy |
| [ADR-0005](docs/ADR-0005-CANONICAL-JSON.md) | Alpha canonical digest contract |
| [ADR-0006](docs/ADR-0006-JSON-SCHEMA-VALIDATION.md) | Hermetic JSON Schema 2020-12 validation |
| [ADR-0007](docs/ADR-0007-STORAGE-IDENTITY-AND-CONTROL-AUDIT.md) | SQLite identity/version and privileged-operation audit |
| [ADR-0008](docs/ADR-0008-MCP-TOOL-SURFACE-DIGEST.md) | Canonical model-facing MCP surface and fail-closed binding |
| [ADR-0009](docs/ADR-0009-OPERATIONAL-LOGGING-BOUNDARY.md) | Operational log redaction boundary |
| [ADR-0010](docs/ADR-0010-SCENARIO-ARTIFACTS.md) | Bounded scenario artifacts and scripted evidence reports |
| [Failure Mode Matrix](docs/FAILURE-MODE-MATRIX.md) | Design risks and required responses—not a test-completion report |
| [v0.1 P0 Traceability](docs/V0.1-P0-TRACEABILITY.md) | P0-by-P0 evidence, exclusions, and stable-release blockers |
| [Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md) | Dated prior-art screen and rejected directions |
| [Roadmap](docs/ROADMAP.md) | Ordered phases and exit criteria |

The RFC and accepted ADRs define intended semantics. Implementation Status and
executable tests define what the current development build can honestly claim.

## Roadmap to the first tagged release

The next release-critical work is:

1. virtual-time advancement and deterministic scheduled faults;
2. crash kill-point and tagged-database migration fixtures;
3. upstream surface inspector and automated refresh (canonical local binding is implemented);
4. prove the new egress-deny hermetic integration gate on Linux CI;
5. pinned official MCP conformance scenarios for the tools-first subset;
6. live OpenAI/ChatGPT and Anthropic/Claude smoke-test matrix;
7. recorder redaction tests if recorder enters v0.1 scope; repository secret
   scanning is configured separately;
8. differential validation and an honest L2 coverage report;
9. a second independent stateful reference domain;
10. complete P0/P1 failure-mode traceability.

Cloud hosting, registries, marketplaces, and automatic production mirroring are
not release priorities.

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before changing protocol, TwinSpec,
canonicalization, or trust-boundary semantics.

Read [SECURITY.md](SECURITY.md) before using the runtime with anything other
than synthetic local fixtures. Do not commit credentials, production traces,
personal data, or third-party recordings that cannot legally be redistributed.

## Positioning and research caveat

The project does not claim to be the first mock server, stateful sandbox,
service-virtualization system, or digital twin. The narrower hypothesis is that
agent engineering benefits from a reusable combination of:

- an MCP-compatible agent-facing surface;
- explicit state-transition contracts;
- forkable deterministic world state;
- strict control-plane isolation;
- declared fidelity and differential validation.

The prior-art search is documented in
[Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md). A dated public search
cannot prove that no similar public, private, or unindexed project exists. The
positioning should change if stronger prior art appears.

## License

[MIT](LICENSE)
