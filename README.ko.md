<div align="center">

<h1>MCP State Twin</h1>

<p><strong>재현 가능한 AI Agent 평가를 위한 deterministic · forkable · stateful MCP 테스트 월드</strong></p>
<p>동일한 world snapshot에서 시작해 Agent마다 서로 다른 유효한 tool trajectory를 실행하고, production service에 쓰지 않은 채 terminal state를 비교합니다.</p>

<p>
  <a href="README.md">简体中文</a> ·
  <a href="README.en.md">English</a> ·
  <a href="README.ja.md">日本語</a> ·
  <strong>한국어</strong>
</p>

<p>
  <a href="https://github.com/augety121/MCP-State-Twin/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/augety121/MCP-State-Twin/ci.yml?branch=main&style=flat-square&label=CI&logo=githubactions&logoColor=white"></a>
  <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/github/license/augety121/MCP-State-Twin?style=flat-square&label=License"></a>
  <img alt="Go 1.26.x" src="https://img.shields.io/badge/Go-1.26.x-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="MCP" src="https://img.shields.io/badge/MCP-Streamable_HTTP-5B5BD6?style=flat-square">
  <img alt="Development Preview" src="https://img.shields.io/badge/status-development_preview-D97706?style=flat-square">
</p>

<p><strong>Fork the tool world, not production.</strong></p>
<p><code>snapshot</code> → <code>fork</code> → <code>act</code> → <code>assert</code> → <code>diff</code></p>

</div>

> [!IMPORTANT]
> **Development Preview · `0.1.0-dev` · latest prerelease `v0.1.0-alpha.1` · production-ready 아님.**
> 현재 주장 가능한 구현 범위는 [Implementation Status](docs/IMPLEMENTATION-STATUS.md), RFC, 승인된 ADR, 사양 및 실행 가능한 테스트 증거에 의해 정의됩니다.

## 상태

**개발 프리뷰(`0.1.0-dev`)이며 최신 공개 prerelease는 `v0.1.0-alpha.1`입니다.** 동작하는 Go CLI와 런타임이 포함되어 있지만 production-ready가 아니며 RFC의 v0.1 release gate를 모두 통과하지 않았습니다.

### 구현 및 검증된 주요 기능

- 엄격한 TwinSpec `v1alpha1` YAML 디코딩, 구조 검증, hermetic JSON Schema 2020-12 입출력 컴파일
- Spec, MCP tool surface, world state에 대한 canonical SHA-256 digest
- `current` / `drifted` / `unknown` surface 불일치 시 fail closed 하는 upstream binding admission
- 4,096-byte 제한과 cost limit를 적용한 CEL 표현식(precondition / effect / query / postcondition / global invariant)
- SQLite 기반 atomic transition, versioned database identity, tool-call audit, transactional control-operation audit
- 불변 logical snapshot, 격리 fork, reset, canonical state diff
- 공식 Go SDK 기반 stateless Streamable HTTP MCP data plane
- 별도 인증되는 HTTP control plane
- 합성 상태를 사용하는 6-tool issue-tracker reference twin
- unit / deterministic replay / MCP HTTP / authorization / output rollback / migration refusal / 100-fork isolation test
- Linux CI의 고정 버전 공식 MCP conformance check(initialize, ping, tools-list, JSON Schema 2020-12)
- bounded TwinSpec/CEL fuzz target과 secret-policy / loopback-only hermetic CI gate
- initialize → snapshot → fork 2회 → 각 fork 변경 → terminal diff CLI 흐름
- deterministic environment identity, ordered tool trace, JSON Pointer state assertion, canonical state diff를 제공하는 bounded Scenario `v1alpha1` runner

### 아직 구현되지 않았거나 검증되지 않은 항목

- recorder, cassette replay, trace redaction, 자동 upstream surface inspection/refresh
- 나머지 deterministic fault phases, scheduler, deterministic entropy, idempotency, crash/cancellation, eventual consistency는 미구현(private clock과 두 fault phases는 구현됨)
- 실제 ChatGPT / OpenAI API / Claude / Claude Code smoke test
- evidence-derived host compatibility report, provider harness
- differential validation, L2 fidelity promotion workflow
- data-plane authentication, TLS, remote multi-tenancy, security audit

자세한 내용은 [Implementation Status](docs/IMPLEMENTATION-STATUS.md)를 참고하세요. Roadmap 항목을 현재 기능처럼 표현하지 않습니다.

## 왜 필요한가

도구를 사용하는 Agent를 제대로 평가하려면 “그럴듯한 JSON”만으로는 충분하지 않습니다.

Issue tracker Agent는 issue를 읽고, comment를 만들고, 모호한 timeout 이후 재시도하고, 다시 issue를 읽은 뒤 예상 상태가 존재할 때만 close할 수 있습니다. **각 호출은 이후 호출이 관찰해야 할 상태를 바꿉니다.**

| 접근 방식 | 주요 강점 | Agent 평가에서의 한계 |
|---|---|---|
| Record/replay | 이미 기록된 경로 재현 | 새 모델이 기록되지 않은 유효 경로를 선택할 수 있음 |
| Static MCP mock | 클라이언트 격리 및 제어된 응답 | cross-call state, constraint, failure semantics가 불완전할 수 있음 |
| Hand-built benchmark sandbox | curated task collection 평가 | 개발자 소유 tool surface 재사용이 핵심 추상화가 아님 |
| Live test / production service | 실제 동작 검증 | side effect, rate limit, cost, shared-state pollution, 재현 불가능한 시작 상태 |
| **MCP State Twin** | forkable world state에서 명시적 transition 실행 | fidelity는 모델링 및 검증된 동작 범위로 제한됨 |

Record/replay는 L0 fidelity mode로 계획되어 있으며 MCP State Twin을 대체하는 것이 아니라 보완합니다.

## 빠른 시작

요구 사항: **Go 1.26.x**, **Git**

```bash
go mod download
go run ./cmd/statetwin validate \
  --spec examples/issue-tracker/twin.yaml
```

격리 DB와 base snapshot을 생성합니다.

```bash
go run ./cmd/statetwin init \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --db demo.db \
  --branch main \
  --snapshot base
```

동일한 snapshot에서 두 world를 fork합니다.

```bash
go run ./cmd/statetwin fork --db demo.db --snapshot base --branch run-a
go run ./cmd/statetwin fork --db demo.db --snapshot base --branch run-b
```

서로 다른 유효 trajectory를 실행합니다.

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

최종 상태를 비교합니다.

```bash
go run ./cmd/statetwin diff \
  --db demo.db \
  --before run-a \
  --after run-b
```

Diff는 안정적인 JSON Pointer path를 사용합니다. key에 `/`가 포함되면 JSON Pointer 규칙에 따라 escape되므로 `octo/demo#1`은 `octo~1demo#1`로 표시됩니다.

## 재현 가능한 Scenario 실행

```bash
go run ./cmd/statetwin scenario \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --scenario examples/issue-tracker/scenario-close-issue.yaml
```

예상하지 못한 error class 또는 assertion failure가 발생하면 non-zero로 종료합니다. JSON report에는 environment digest, ordered tool trace, initial/terminal state digest, assertion evidence, canonical state diff가 포함됩니다.

현재 runner identity는 `scripted-scenario`이며 실제 Codex / OpenAI / Claude 등의 model evaluation으로 표현하지 않습니다.

> [!WARNING]
> Scenario report에는 tool input과 result가 포함됩니다. synthetic fixture만 사용하고 credential, production trace, personal data가 포함된 report를 commit하지 마세요.

## MCP Server 실행

```bash
export STATETWIN_CONTROL_TOKEN='replace-with-a-local-secret'
```

PowerShell:

```powershell
$env:STATETWIN_CONTROL_TOKEN = 'replace-with-a-local-secret'
```

```bash
go run ./cmd/statetwin serve \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --db demo.db
```

| Plane | Endpoint | 노출되는 작업 |
|---|---|---|
| Agent data plane | `http://127.0.0.1:8090/mcp/main` | 모델링된 business tool만 |
| Private control plane | `http://127.0.0.1:8091/v1` | branch state, snapshot, fork, reset, diff, forward-only clock advance |

Branch ID는 model-visible 추가 인수가 아니라 MCP URL의 일부입니다. 따라서 branch가 달라도 tool input schema는 동일합니다.

> [!WARNING]
> 현재 data plane에는 authentication과 TLS가 없습니다. 두 server는 기본적으로 loopback에 bind되며 로컬에서만 사용해야 합니다. control token만으로는 현재 build를 Internet에 안전하게 노출할 수 없습니다.

## Reference Twin

Agent-visible tool:

- `get_repository`
- `list_issues`
- `get_issue`
- `create_issue`
- `add_comment`
- `close_issue`

Snapshot, fork, reset, diff, state inspection, 향후 fault control은 MCP tool이 아니며 `tools/list`에 나타나지 않습니다.

Reference twin은 synthetic, `L1`, `unverified`, upstream에 대해 `unbound`입니다. **GitHub-equivalent environment라고 설명해서는 안 됩니다.**

## TwinSpec

TwinSpec은 modelled tool world를 위한 versioned / reviewable contract입니다.

```text
tool behavior = input contract
              + reads and preconditions
              + deterministic state effects
              + postconditions and global invariants
              + structured result or typed error
              + time and idempotency semantics
```

전체 실행 가능한 spec은 [examples/issue-tracker/twin.yaml](examples/issue-tracker/twin.yaml)을 참고하세요.

### Expression boundary

TwinSpec expression은 `cel-go`를 사용하며 source는 최대 4,096 UTF-8 byte, load 시 compile되고 cost limit 10,000으로 평가됩니다. 사용할 수 있는 JSON-shaped variable은 `input`, `state`, `vars`, `item`, `clock`, `call_index`뿐입니다.

filesystem, process, network, reflection, 임의 Go function은 등록되지 않습니다. `v1alpha1`은 native extension을 지원하지 않습니다.

### JSON Schema boundary

Tool input과 successful output은 JSON Schema Draft 2020-12로 compile/validate됩니다. format assertion이 활성화되어 있습니다. local `$defs`와 fragment는 지원하지만 외부 network/filesystem resource가 필요한 reference는 startup에서 실패합니다. 선언된 output이 유효하지 않으면 transition은 rollback되고 `INTERNAL_TWIN_ERROR`를 반환합니다.

### Effect operation

| Operation | Semantics |
|---|---|
| `allocate` | deterministic sequence 증가 후 `vars`에 bind |
| `insert` | keyed entity insert, 이미 존재하면 conflict |
| `update` | keyed entity replace 또는 merge, 없으면 fail |
| `delete` | keyed entity delete, 없으면 fail |

## Determinism contract

**결정적인 것은 environment이며 language model이 아닙니다.**

```text
E = (runtime version,
     TwinSpec digest,
     snapshot digest,
     scenario seed,
     ordered tool calls)

execute(E) -> (ordered structured results, final state digest)
```

각 comparison run은 동일한 immutable snapshot에서 시작하며 성공 여부는 exact trajectory equality가 아니라 terminal state와 declared invariant로 평가합니다.

## Error / transaction semantics

```text
load branch head
  -> validate input
  -> evaluate preconditions
  -> apply effects to an isolated working state
  -> evaluate query, postconditions, and global invariants
  -> commit state and audit record atomically
```

Canonical error class:

- `INVALID_INPUT`
- `PRECONDITION_FAILED`
- `NOT_FOUND`
- `CONFLICT`
- `INVARIANT_VIOLATION`
- `UNMODELED_BEHAVIOR`
- `INTERNAL_TWIN_ERROR`

Timeout-before-effect, timeout-after-effect, partial-effect, rate-limit, eventual-consistency fault는 사양에 정의되어 있지만 아직 구현되지 않았습니다.

## Fidelity level

| Level | 의미 | 용도 |
|---|---|---|
| L0 — Cassette replay | 기록된 interaction 일치 | exact-path smoke/regression test |
| L1 — Stateful template | explicit entity + reviewed basic transition | development / exploratory workflow test |
| L2 — Contract-backed | reviewed rule / invariant / differential test / upstream fingerprint | 선언된 범위의 CI / evaluation |
| L3 — Native/reference | shared 또는 domain-provided reference logic | high-fidelity domain simulation |

생성되거나 추론된 behavior가 스스로 L2/L3로 승격될 수는 없습니다. 현재 reference twin은 **L1 / unverified**입니다.

## MCP와 model provider

Core는 특정 model-provider SDK가 아니라 MCP와 통합됩니다. 하나의 tool surface를 제공할 수는 있지만 서로 다른 model이 동일한 tool이나 trajectory를 선택한다고 주장하지 않습니다.

공식 Go SDK를 server/client 양쪽에서 사용하는 stateless Streamable HTTP integration test가 있고, Linux CI는 MCP conformance framework `v0.1.16`을 고정해 initialize, ping, tools-list, JSON Schema 2020-12를 검사합니다.

Repository는 아직 실제 ChatGPT / OpenAI / Claude smoke test를 완료하지 않았습니다.

Design sources:

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
statetwin scenario   execute a bounded scripted scenario and assertions
statetwin serve      run separate MCP data and HTTP control planes
statetwin version    print the development version
statetwin protocols   print pinned MCP wire-evidence profiles
```

## Test / Build

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/statetwin
```

## Trust boundary

### Agent data plane

- TwinSpec에 선언된 business tool만 노출
- URL을 통해 branch bind
- hidden expected state 또는 test control을 공개하지 않음
- typed domain failure를 model-observable MCP tool error로 반환

### Simulation control plane

- 독립 bearer token 필요
- state / snapshot / fork / reset / diff 제공
- data plane과 다른 loopback address/port에 기본 bind
- Agent가 아니라 test harness 또는 사람이 조작

**Prompt instruction은 authorization boundary가 아닙니다.**

## 문서

- [Implementation Status](docs/IMPLEMENTATION-STATUS.md)
- [Project Map](docs/PROJECT-MAP.md) — 제품 경계, 아키텍처 및 수명주기
- [Documentation Governance](docs/DOCS-GOVERNANCE.md) — 문서 권위와 증거 규칙
- [Release Management](docs/RELEASE-MANAGEMENT.md) — 버전, CI, 태그 및 릴리스 증거
- [RFC-0001](docs/RFC-0001.md)
- [RFC-0002](docs/RFC-0002-V0.1-RELEASE-PROFILE.md)
- [SPEC-0001](docs/SPEC-0001-TWINSPEC-CORE.md) / [SPEC-0002](docs/SPEC-0002-RUNTIME-SEMANTICS.md) / [SPEC-0003](docs/SPEC-0003-MCP-BOUNDARIES-AND-COMPATIBILITY.md)
- [SPEC-0004](docs/SPEC-0004-EVIDENCE-FIDELITY-AND-RELEASE.md) / [SPEC-0005](docs/SPEC-0005-SCENARIO-AND-REPORT.md) / [SPEC-0006](docs/SPEC-0006-HOST-COMPATIBILITY-AND-MODEL-EVALUATION.md)
- [Phase 0 MCP 2026 Gap Matrix](docs/PHASE-0-MCP-2026-GAP-MATRIX.md) / [vNext Adoption Record](docs/VNEXT-ADOPTION.md)
- [vNext SPEC Pack Traceability Matrix](docs/VNEXT-TRACEABILITY.md)
- [SPEC-0007](docs/SPEC-0007-VIRTUAL-TIME-ENTROPY-SCHEDULER.md) / [SPEC-0008](docs/SPEC-0008-DETERMINISTIC-FAULTS.md) / [SPEC-0012](docs/SPEC-0012-STORAGE-CONCURRENCY-RECOVERY.md) / [ADR-0011](docs/ADR-0011-HEAD-VERSION-AND-VIRTUAL-CLOCK.md) / [ADR-0012](docs/ADR-0012-DETERMINISTIC-FAULT-PREVIEW.md)
- [SPEC-0015](docs/SPEC-0015-RESOURCE-GOVERNANCE.md) / [ADR-0013](docs/ADR-0013-RESOURCE-GOVERNANCE-PROFILE.md)
- [Failure Mode Matrix](docs/FAILURE-MODE-MATRIX.md)
- [v0.1 P0 Traceability](docs/V0.1-P0-TRACEABILITY.md)
- [Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md)
- [Roadmap](docs/ROADMAP.md)

RFC와 accepted ADR은 intended semantics를 정의하고, Implementation Status와 executable test는 현재 build가 실제로 주장할 수 있는 범위를 정의합니다.

## Contributing / Security

Protocol, TwinSpec, canonicalization, trust-boundary semantics를 변경하기 전에 [CONTRIBUTING.md](CONTRIBUTING.md)를 읽어 주세요.

Synthetic local fixture 이외의 데이터를 사용하기 전에 [SECURITY.md](SECURITY.md)를 확인하세요. credential, production trace, personal data, 재배포 권한이 없는 third-party recording을 commit하지 마세요.

## Positioning

이 project는 자신이 최초의 mock server, stateful sandbox, service virtualization system 또는 digital twin이라고 주장하지 않습니다.

더 좁은 가설은 Agent engineering이 MCP-compatible surface, explicit state-transition contract, forkable deterministic world state, strict control-plane isolation, declared fidelity와 differential validation의 재사용 가능한 조합에서 이점을 얻을 수 있다는 것입니다.

Prior-art 조사는 [Competitive Landscape](docs/COMPETITIVE-LANDSCAPE.md)에 기록되어 있습니다. 더 강한 prior art가 발견되면 positioning은 증거에 맞게 변경되어야 합니다.

## License

MCP State Twin은 **MIT License**로 제공됩니다. 전체 라이선스 본문은 [LICENSE](LICENSE)를 확인하세요. README의 설명과 차이가 있는 경우 `LICENSE`의 표준 MIT 본문이 우선합니다.
