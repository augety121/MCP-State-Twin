# ADR-0048: Fail-closed offline comparison grading

- Status: Accepted for the bounded offline correction below
- Date: 2026-09-26
- Basis: maintainer request to continue implementation; RFC-0001 I-8/I-10/I-16,
  ADR-0039 and AE-021/AE-022

Accept [SPEC-0048](SPEC-0048-COMPARISON-GRADING-ELIGIBILITY.md). Internally
consistent replay is not sufficient to count a trial as validly evaluated:
`not_evaluated`, unknown outcomes and evaluator errors must remain inconclusive.
Comparison must detect new failed policy assertions, not just an increased
count of unauthorized attempts. It must preserve all existing success-loss and
identity/budget mismatch checks.

This is an experimental comparison-policy correction, identified in new reports
as `offline-regression-v2`. It does not mutate historical evidence, change Task
oracles or provider behavior, add live evaluation, or establish statistical
significance. Old reports without that policy marker do not gain these checks.
