# Feasibility, Dependency & Landing Matrix

> **Status:** Proposal planning artifact  
> **Purpose:** Distinguish technically feasible work from work that is justified, verified or ready to land.

## 1. Reading rules

- **Feasible** does not mean implemented.
- **Implementable now** means prerequisites are understood well enough to begin after current-repo inventory.
- **Blocked** means a prerequisite or product decision is missing.
- **Future-candidate** means technically possible but should not enter current critical path.

## 2. Core matrix

| Capability | Technical feasibility | Product need now | Key dependency | Main correctness risk | Landing recommendation |
|---|---|---|---|---|---|
| MCP 2026-07-28 rebaseline | High | Immediate | exact SDK + conformance inventory | false protocol claims | **Phase 0 first** |
| explicit protocol registry | High | Immediate | current code inventory | SDK default drift | **Phase 0** |
| `server/discover` modern support | High | Immediate for 2026 claim | SDK/wire implementation | treating discovery as mandatory client first call | **Phase 0** |
| legacy <=2025 lifecycle | High | Unknown | user/host need | dual-stack complexity | **decide, don't assume** |
| virtual clock | High | High | runtime state model | host-clock leakage | **v0.1** |
| deterministic scheduler | High | High | virtual time | tie ordering / event loops | **v0.1** |
| deterministic entropy | High | High | environment identity | algorithm drift | **v0.1** |
| deterministic fault subset | High | High | scheduler/time for temporal faults | mixing infrastructure corruption with modeled failure | **v0.1 bounded** |
| crash killpoints | Medium-high | High | storage semantics | overclaiming real hardware crash fidelity | **v0.1 bounded** |
| explicit branch optimistic concurrency | High | High | storage/head version | race semantics | **v0.1** |
| full storage migration policy | High | High before 1.0 | schema inventory | irreversible corruption | **v0.1→1.0** |
| whole-runtime resource limits | High | High | profiling | limits alter semantics | **v0.1** |
| second reference domain | High | High | domain design | overfitting issue tracker | **v0.1 gate** |
| recorder | High | Medium | redaction policy | secret/PII capture | **v0.2** |
| cassette replay | High | Medium | recorder format | replay mistaken for fidelity | **v0.2** |
| upstream surface inspector | High | Medium | MCP client/profile | auth-scoped surface ambiguity | **v0.2** |
| structural drift detector | High | Medium | canonical surface | stale cache/profile | **v0.2** |
| semantic auto-refresh | Low as a trustworthy verifier | No | semantic oracle | hallucinated equivalence | **do not auto-approve** |
| differential harness | Medium-high | High for L2 | safe upstream oracle | upstream nondeterminism | **v0.2 core** |
| bounded L2 promotion | Medium | High for fidelity claims | differential evidence | coverage too broad | **v0.2 evidence-driven** |
| canonical evidence schema | High | High | current report inventory | schema churn/privacy | **v0.2, design early** |
| portable TwinBundle | High | Medium-high | stable identity/versioning | unsafe imports/version drift | **v0.2 alpha** |
| Codex local host test | High | High | deterministic v0.1 + host adapter | host config/version churn | **Phase 3 after runtime** |
| Claude Code local host test | High | High | deterministic v0.1 + host adapter | permissions/version churn | **Phase 3** |
| OpenAI API remote MCP test | High | High | secure staging remote endpoint | security/data handling | **Phase 3 after staging** |
| Anthropic Messages MCP test | High for documented subset | High | secure remote endpoint + current connector profile | connector scope/beta churn | **Phase 3 scoped** |
| Managed Agents test | Medium-high | Optional | current product access/profile | output transforms/permissions | **Phase 3 optional** |
| multi-agent isolated-fork evaluation | High | High | host adapter | statistical interpretation | **Phase 3** |
| shared-branch multi-agent | Medium-high | Future useful | concurrency evidence | nondeterministic interleavings | **post basic host matrix** |
| Tasks extension | Medium-high | Unknown | real async domain | extension/host support variance | **future RFC on demand** |
| MRTR scenarios | Medium-high | Unknown | interactive domain + host support | deadlock/input races | **future RFC on demand** |
| tool-search/deferred discovery evidence | High | Future useful | host support | host surface ambiguity | **Host Profile extension** |
| OpenTelemetry export | High | Optional | evidence schema | sensitive content/schema churn | **optional after canonical evidence** |
| SBOM/provenance | High | High by 1.0 | release workflow | maintenance | **1.0 hardening** |
| signed release artifacts | High | Medium-high | provenance/identity | confusing signature with semantics | **1.0 hardening** |
| signed TwinBundle | High | Optional | stable bundle | trust-policy complexity | **post bundle stability** |
| data-plane remote OAuth | Medium-high | Only if remote needed | threat model | token/audience/SSRF | **Phase 4** |
| Internet-ready remote mode | Medium | Unknown | full security/ops | large attack surface | **separate release profile** |
| remote multi-tenancy | Medium | No proven need | new storage/identity/RBAC/ops | architecture explosion | **separate RFC / defer** |
| distributed storage | Medium | No proven need | consistency model | breaks local determinism assumptions | **separate RFC / defer** |
| arbitrary native plugins | Technically possible | No | sandbox/security | destroys hermeticity | **reject by default** |
| sandboxed extension runtime | Medium | Unknown | extension use cases + security RFC | escape/determinism | **future** |
| GUI/computer-use Twin | Medium, domain-specific | Unknown | observation/action model | fidelity explosion | **separate product/domain RFC** |
| robotics/physical Twin | Domain-dependent | Unknown | physical simulator/reference | impossible general fidelity | **out of current scope** |
| generic "AGI compatibility" | Not meaningfully provable | No | unknown future systems | meaningless overclaim | **never use as compatibility claim** |

## 3. Critical dependency DAG

```text
Current repo inventory
        |
        v
MCP 2026 rebaseline
        |
        +------------------------+
        |                        |
        v                        v
Governance/evidence rules   Protocol correctness
        |                        |
        +-----------+------------+
                    |
                    v
        Deterministic Runtime Core
       /       |        |        \
 virtual   scheduler  storage   limits
 time       entropy    recovery
       \       |        /
        +------v-------+
             faults
               |
               v
          v0.1 candidate
               |
      +--------+---------+
      |                  |
      v                  v
 recorder/replay    surface/drift
      |                  |
      +--------+---------+
               v
      differential fidelity
               |
               v
       canonical evidence
               |
               v
          bundle alpha
               |
               v
       real host evaluation
      /       |        |      \
  Codex  Claude Code OpenAI  Anthropic
               |
               v
     optional remote security
               |
               v
              1.0
```

## 4. Why live agents come after deterministic runtime

Connecting Codex/Claude too early is technically possible, but produces weak evidence if:
- protocol profile is unclear;
- snapshots do not include scheduler/entropy;
- failures are not classified;
- branch conflicts are ambiguous;
- evidence schema is unstable.

Therefore the host harness is **not** the first milestone even though it is a major product goal.

## 5. Why future AGI does not change the core order

A more capable agent makes:
- state correctness;
- side-effect boundaries;
- concurrency;
- evidence;
- authorization

more important, not less.

The project should therefore harden the world substrate before optimizing for a specific model generation.

## 6. Feasibility confidence levels

### High
Mature primitives exist and architecture fits current project.

### Medium-high
Implementable, but external host/upstream behavior introduces uncertainty.

### Medium
Requires a significant product/security/consistency choice.

### Domain-dependent
Cannot be generalized honestly without a concrete domain/reference system.

## 7. Re-evaluation triggers

Revisit this matrix when:
- MCP publishes a new dated core revision;
- Go SDK lifecycle materially changes;
- Codex/Claude host MCP behavior changes;
- a second reference domain exposes missing core abstraction;
- remote deployment becomes a real user requirement;
- a real async/MRTR/Tasks scenario appears;
- v1.0 planning begins.


## 8. Universal compatibility extension

| Capability | Feasibility | Main risk | Landing |
|---|---|---|---|
| Host Surface Projection | High | hidden host transforms | design before real-host matrix |
| Portable MCP Tools Profile | High | too narrow/too broad baseline | before adapters |
| Compatibility lint | High | profiles become stale | adapter framework |
| Host Adapter SPI | High | vendor config churn | Phase 3 foundation |
| Host Isolation profiles | High local / medium cloud | built-in tools and secret exposure | before strict benchmark |
| Episode Orchestrator | High | mixing host failure with agent failure | Phase 3 foundation |
| Scenario families | High | generator bugs/answer leakage | after core determinism |
| Metamorphic tests | High | invalid property assumptions | after accepted semantics |
| Curriculum/coaching | High | contamination of blind eval | after episode evidence |
| Gemini CLI adapter | High | name/schema transforms | initial host matrix |
| GitHub Copilot IDE adapter | High | environment variants | initial host matrix |
| GitHub cloud agent profile | Medium-high | autonomous tool use/cloud restrictions | separate profile |
| Cursor adapter | High | product changes | initial host matrix |
| Windsurf adapter | High | tool-count budget | initial host matrix |
| Cline adapter | High | broad built-ins/auto-approve | isolation required |
| Amazon Q adapter | High | permissions/built-ins | initial host matrix |
| JetBrains/Junie adapter | High | IDE/version topology | initial host matrix |
| Zed native adapter | High | dynamic tools | initial host matrix |
| Zed ACP external-agent path | Medium-high | two config/auth layers | later host path |
| OpenCode adapter | High | context/tool modes | initial host matrix |
| ACP integration | High as host path | ACP evolving | optional profile |
| A2A delegation | Medium-high | agent identity/task semantics | future multi-agent |
| Literal "all agents compatible" | Not provable | open-ended ecosystem | never claim |
