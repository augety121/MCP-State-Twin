# ADR-0063: Independent Offline Assessment

- Status: Accepted for the bounded offline assessment subset
- Date: 2026-09-29
- Contracts: SPEC-0063, SPEC-0064, SPEC-0065
- Authorization: maintainer explicitly requested implementation of all three Specs

## Evidence and problem

PRs #8 and #9 implemented bounded suites and replay-backed audits. Current
`InspectSuite` proves agreement with the plan saved inside a suite; it does not
bind that plan to an independently supplied intention. `summarizeTasks` groups by
Task ID but explicitly does not establish uniform definitions across repeats.
`regressionReasons` permits equally failing baseline/candidate outcomes to have
no observed regression. These are documented boundaries, not failed tests.

Accept three complementary layers: an independent expected-plan contract,
replay-verified repeat-definition consistency, and a named absolute candidate
acceptance policy. Keep existing comparison decisions, suite artifact formats,
oracle semantics and CLI defaults unchanged. Put new results in a separate
assessment envelope, so old suite reports remain verifiable.

## Delivery and limits

1. SPEC-0063: admit an external expectation and match plan plus trial budgets.
2. SPEC-0064: assess full normalized definitions across all planned repeats.
3. SPEC-0065: combine the independent checks with explicit outcome requirements
   in a new read-only `eval suite-assess` command.

No automatic plan generation from the audited directory, percentage thresholds,
statistical significance, model ranking, provider calls, resume/repair, hashes
or signatures. Existing business digests remain part of definition identity.
An independent local file is not proof of preregistration or trusted provenance.

This accepts only the B09/B10-offline increment described above. Implementation
evidence is recorded separately in IMPLEMENTATION-STATUS.md. It does not accept
live evaluation, external-user qualification or automatic model upgrades.
