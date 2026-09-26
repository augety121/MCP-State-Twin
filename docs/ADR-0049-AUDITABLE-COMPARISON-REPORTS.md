# ADR-0049: Auditable comparison report projections

- Status: Accepted, experimental offline report extension
- Date: 2026-09-26
- Basis: ADR-0039, B09-offline / AE-008 / AE-021 / AE-022

Accept [SPEC-0049](SPEC-0049-COMPARISON-REPORT-ACCOUNTING.md). Keep the existing
plan schema, report format identifier, original fields and CLI behavior. Add
an explicit decision-policy marker, model labels, separate cohort denominators,
verified grading/risk counts and deterministic pair reasons. These are additive
experimental fields; strict old consumers must be updated before consuming them.

Report identity is a projection of validated input, not evidence of chronological
preregistration or authenticity. No additional file hash, signature, manifest,
timestamp, monetary estimate, automatic upgrade or hidden retry selection is added.
