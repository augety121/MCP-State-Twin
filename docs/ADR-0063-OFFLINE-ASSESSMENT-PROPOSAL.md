# ADR-0063: Independent Offline Assessment — Proposal

- Status: Proposed; not accepted and not implemented
- Date: 2026-09-29
- Proposed contracts: SPEC-0063, SPEC-0064, SPEC-0065
- Requires: explicit implementation authorization and acceptance before code changes

## Evidence and problem

PRs #8 and #9 implemented bounded suites and replay-backed audits. Current
`InspectSuite` proves agreement with the plan saved inside a suite; it does not
bind that plan to an independently supplied intention. `summarizeTasks` groups by
Task ID but explicitly does not establish uniform definitions across repeats.
`regressionReasons` permits equally failing baseline/candidate outcomes to have
no observed regression. These are documented boundaries, not failed tests.

Propose three complementary layers: an independent expected-plan contract,
replay-verified repeat-definition consistency, and a named absolute candidate
acceptance policy. Keep existing comparison decisions, suite artifact formats,
oracle semantics and CLI defaults unchanged. Put new results in a separate
assessment envelope, so old suite reports remain verifiable.

## Proposed delivery and limits

1. SPEC-0063: admit an external expectation and match plan plus trial budgets.
2. SPEC-0064: assess full normalized definitions across all planned repeats.
3. SPEC-0065: combine the independent checks with explicit outcome requirements
   in a new read-only `eval suite-assess` command.

No automatic plan generation from the audited directory, percentage thresholds,
statistical significance, model ranking, provider calls, resume/repair, hashes
or signatures. Existing business digests remain part of definition identity.
An independent local file is not proof of preregistration or trusted provenance.

This document only proposes a B09/B10-offline increment. The acceptance tests in
the linked Specs are future obligations, not executed verification. No new
command is advertised as available. Business code and runtime remain unchanged
during this design stage.
