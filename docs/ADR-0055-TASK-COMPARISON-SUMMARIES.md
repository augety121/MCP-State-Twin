# ADR-0055: Auditable Task-level Comparison Summaries

- Status: Accepted for the bounded offline extension
- Date: 2026-09-29
- Basis: B09-offline and the maintainer request to continue implementation via PR
- Contract: [SPEC-0055](SPEC-0055-TASK-COMPARISON-SUMMARIES.md)

The existing comparison exposes global lifecycle counts and individual pairs,
but users must manually reconstruct task-level failure and missing-data totals.
Add deterministic count summaries from already-verified rows. Keep expected
abstention separate, retain all unscored trials, and preserve pair decisions.
Grouping by Task ID is descriptive and does not establish consistent Task
definitions across repeats. No percentages, aggregate upgrade recommendation,
new model trials or statistical claim follow from the summary.

Also require original integer repeat tokens so YAML float conversion cannot
silently change the fixed comparison plan. Existing valid integer plans and
decision rules remain compatible; the additive experimental output requires
strict consumers to accept the new fields. No evidence or stored Task changes.
