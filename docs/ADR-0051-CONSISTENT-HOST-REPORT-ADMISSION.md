# ADR-0051: Consistent Host Report Admission

- Status: Accepted; bounded maintenance of ADR-0017, not a live compatibility claim
- Date: 2026-09-26
- Contract: [SPEC-0051](SPEC-0051-HOST-REPORT-CONSISTENCY.md)

## Evidence and decision

Regression tests reproduced admission of an `exact` surface with unequal declared
digests, a `verified` report with zero successful assertions, contradictory
cancellation declarations, impossible protocol dates, padded placeholder host
versions, unknown runtime versions, an empty custom provider and an experimental
expiry before observation. A YAML-escaped credential-like value also bypassed
the raw-byte scan. Correct these admission gaps before any report can be used
by future claim tooling. Keep the report format and existing business digests;
do not rewrite historical reports or infer missing facts.

This intentionally rejects formerly admitted inconsistent preview artifacts.
Calendar validity is not protocol support; equality of supplied digests is not
upstream observation or provenance. A parsed `verified` label remains a claim
made by its author, not a verification performed by this program.

## Scope

No provider, model, SDK, MCP surface, database, or evidence writer changes.
No credential lookup, external execution, file hashing or release promotion.
Secret scanning remains a finite-pattern policy, not general PII detection.
