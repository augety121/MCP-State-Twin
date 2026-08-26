# ADR-0018: Deterministic TwinBundle and Local Scripted Episode Preview

- **Status:** Accepted
- **Date:** 2026-08-26
- **Accepts:** the bounded subset of RFC-0003 sections 4 through 8
- **Amends:** ADR-0013 resource profile bundle limits
- **Stable v0.1 impact:** none; preview extension only

## Context

The repository could run individual TwinSpec/fixture/Scenario files, but there
was no closed portable artifact or first-class lifecycle evidence. The vNext
proposal pack described bundles and Episodes, yet its templates had no parser,
path-safety rules, deterministic writer or executable evidence.

Implementing a remote host orchestrator now would cross unresolved TLS,
authentication, provider-credential and data-retention boundaries. A bounded
local artifact and scripted lifecycle can be proven without crossing them.

## Decision

Accept:

1. deterministic ZIP TwinBundle artifacts with strict `bundle.yaml` manifests;
2. exact member digests and bounded path-safe admission;
3. one local scripted EvaluationEpisode per command;
4. a closed lifecycle ending in `SUCCEEDED` or `ASSERTION_FAILED` for completed
   Scenario executions;
5. versioned EpisodeEvidence with a canonical digest;
6. release-build source revision injection; and
7. `local-preview-v2` resource limits for Bundle files and bytes.

## Security invariants

- Bundle paths cannot escape the source/archive root.
- Symlinks, devices, directories and executable extensions receive no special
  execution semantics; only regular member bytes are admitted.
- Undeclared members and digest mismatches fail closed.
- Bundle execution makes no upstream/provider request.
- Evidence is labeled `development` and `scripted-scenario`.
- No Bundle control is added to MCP `tools/list`.

## Explicit exclusions

This ADR does not accept durable/remote Episodes, provider harnesses, HostProfile
adapters, signatures, registry trust, export/import, remote authentication,
multi-tenancy, recorder/replay, L2 promotion or production readiness.

## Consequences

- The resource profile digest changes because Bundle limits now affect accepted
  local behavior.
- Existing v0.1 CLI commands and storage schema remain unchanged.
- The Bundle and Episode formats are alpha and require an ADR for incompatible
  semantic changes.
- Provider compatibility remains unverified until real reports exist.
