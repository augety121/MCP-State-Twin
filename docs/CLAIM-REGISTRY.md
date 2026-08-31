# Public Claim Registry

**Status:** normative evidence ledger under SPEC-0021
**Candidate reviewed:** 2026-08-31
**Rule:** a candidate worktree is not a published GitHub capability until it is
merged and its required CI evidence passes on that revision.

| ID | Bounded public claim | State | Evidence | Exclusions |
|---|---|---|---|---|
| CLM-CORE-001 | TwinSpec `v1alpha1` is strictly decoded, bounded and schema-validated | verified locally | `go test ./internal/spec ./internal/engine` | no arbitrary scripts or remote `$ref` |
| CLM-CORE-002 | Supported serial transitions are atomic and deterministically replayable | verified locally | engine/store deterministic and rollback tests | not model determinism; no deterministic concurrent scheduler |
| CLM-CORE-003 | Snapshots are immutable and forks are isolated | verified locally | store isolation/concurrency tests | no COW/GC/HA claim |
| CLM-MCP-001 | Agent data plane exposes business tools, not control operations | verified locally | server negative discovery tests | data plane is not production-authenticated |
| CLM-MCP-002 | Modern 2026-07-28 and legacy 2025-11-25 tools-first wire profiles have direct tests | verified locally | `go test ./internal/server` | not every optional MCP feature |
| CLM-STORAGE-001 | World schema v4 accepts documented historical fixtures and refuses foreign/future stores | verified locally | store migration and kill-point tests | no downgrade, backup, replication or disk-full guarantee |
| CLM-STORAGE-002 | Episode Journal schema v2 migrates the published schema-v1 fixture transactionally | experimental | episode migration and integrity tests | candidate worktree until merged/CI green |
| CLM-EPISODE-001 | TwinBundle and scripted local Episode produce bounded canonical Evidence | experimental | bundle/episode tests | unsigned; no publisher identity or provider run |
| CLM-REMOTE-001 | One coordinator supports leased hermetic workers with heartbeat and fencing | experimental | coordinator/remote tests | no HA, multi-tenant or worker attestation |
| CLM-REMOTE-002 | One parent Episode accepts at most one matching terminal Evidence envelope | experimental | duplicate/conflict/fencing tests | not exactly-once inference, delivery, tool calls or external effects |
| CLM-PROVIDER-001 | OpenAI Responses adapter request/poll/cancel/MCP parsing has mock contract coverage | experimental | `go test ./internal/provider` | OpenAI live profile unverified; no ChatGPT/Codex claim |
| CLM-PROVIDER-002 | Anthropic Messages MCP connector request/result parsing has mock contract coverage | experimental | `go test ./internal/provider` | Anthropic live profile unverified; no Claude/Claude Code claim |
| CLM-FIDELITY-001 | Reference twins are L1, unverified and unbound | verified as limitation | TwinSpec metadata and README | no L2/L3 or upstream equivalence |
| CLM-SECURITY-001 | Hermetic test paths contain no upstream production passthrough | verified locally | namespace/loopback and source-absence checks | not a remote-production security audit |
| CLM-SECURITY-002 | Remote-staging security profile is available | unverified | SPEC-0020 only | not implemented as a complete deployment profile |

## Maintenance

Update this file in the same change as any public claim. A release candidate
must replace `verified locally` with evidence from the exact candidate CI run or
leave the claim scoped to local verification. Provider evidence must also obey
SPEC-0019 freshness and security prerequisites.
