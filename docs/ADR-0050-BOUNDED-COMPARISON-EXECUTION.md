# ADR-0050: Bounded, cancellation-preserving comparison

- Status: Accepted for trusted local offline roots
- Date: 2026-09-26
- Basis: RFC-0001 I-10/I-12; ADR-0039; B09/B12 bounded verification

Accept [SPEC-0050](SPEC-0050-COMPARISON-IO-AND-CANCELLATION.md). Keep one rooted
filesystem handle per comparison, inspect every parent without following observed
symlinks, bound aggregate terminal bytes before parsing, and propagate caller
cancellation even when it occurs inside the final replay. No result may be
returned as a finished comparison after a detected operation-wide interruption.

Read-only and trusted/quiescent-root limitations remain. This is not a filesystem
snapshot, hostile-writer sandbox, hard wall-time/memory quota or resume feature.
No new external IO, evidence mutation, background worker or deletion is added.
