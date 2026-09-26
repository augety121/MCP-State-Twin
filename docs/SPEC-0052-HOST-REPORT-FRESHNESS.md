# SPEC-0052: Host Report Freshness Assessment

Status: accepted by [ADR-0052](ADR-0052-DETERMINISTIC-HOST-REPORT-FRESHNESS.md).
Independent assessment format `statetwin.dev/host-report-assessment/v1alpha1`;
policy `host-report-time-v1`. Not a replacement for HostProfile v1alpha2.

## 1. Input and deterministic policy

`compatibility assess --report report.yaml --at <RFC3339-UTC-time>` MUST load
and validate one report with SPEC-0051 admission. `--at` is required and ends
in `Z`; there is no implicit wall clock, local timezone or timestamp lookup.
The operator-supplied time is not attested. Same input and time give identical
JSON; source bytes and the declared claim level/expiry MUST remain unchanged.

| Report profile | Maximum window from metadata.createdAt |
|---|---|
| generic-mcp, custom-mcp, openai-api-mcp, anthropic-api-mcp | 30 × 24 hours |
| chatgpt-mcp, claude-code-mcp | 14 × 24 hours |

These are project policies, not observed provider stability. The effective expiry
is `min(createdAt + cap, declared validUntil)` when an expiry exists, otherwise
the cap. Non-verified reports may omit expiry but cannot become time-eligible
verified claims. A shorter explicit expiry always wins. No cap may extend it.

## 2. Results and command exit

Temporal classification uses parsed instants, including fractional seconds:

- `at < createdAt`: `not_yet_valid`;
- `at >= effective expiry`: `expired`;
- otherwise: `within_window` (inclusive start, exclusive end).

`claimTimeEligible` is true only when declared level is `verified` and the
classification is `within_window`. Experimental/regressed reports stay ineligible.
The result includes the input profile, declared level, normalized assessment and
observation times, declared/effective expiry, policy and TTL seconds. It MUST
also always include `scopeStatus: not_checked`, `provenance: not_verified` and
`publicationAllowed: false`. It MUST NOT echo raw host/model names, paths, checks,
request identifiers, prompts or report content. No file hashes are generated.

Plain `assess` exits successfully for an admissible diagnostic result, including
expired results. `--require-fresh` first prints the diagnostic result and then
fails with `HOST_REPORT_NOT_TIME_ELIGIBLE` when claimTimeEligible is false.
It is a time gate only, never a live-compatibility or release gate. Invalid input,
invalid time, unrepresentable expiry or output-write failure fails the command;
invalid inputs MUST NOT produce a successful assessment.

## 3. Scope and acceptance

This is read-only, network-free, credential-free, with no external artifacts
fetched or evidence checked. It cannot detect current runtime/surface/host drift,
adapter changes, revoked evidence, forged dates or missing provider provenance.
It cannot generate a verified compatibility matrix. Full B15 remains incomplete.

Tests MUST cover every report profile, creation/end boundary, nanoseconds,
shorter and overlong declarations, no-expiry experimental reports, future dates,
regressed claims, invalid time/calendar/overflow, deterministic bytes and no
source mutation. CLI tests cover required flags, optional gate behavior,
invalid report and writer failure without provider calls or real credentials.
Changing TTLs or interpretation requires a new policy identifier and ADR.
