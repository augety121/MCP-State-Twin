# ADR-0056: Bounded Offline Suite Workflow

- Status: Accepted for the bounded synthetic-only subset
- Date: 2026-09-29
- Basis: B09-offline/B10 and the maintainer request for a larger Spec-first batch
- Contracts: SPEC-0056, SPEC-0057, SPEC-0058

Accept a strict suite plan, read-only all-input preflight, frozen private inputs,
serial reuse of RecordMock, deterministic comparison-plan generation, bounded
artifact IO and final report publication. This completes a useful multi-task
workflow without adding provider transport, retries or parallel workers.

Reuse existing Task, Bundle, RunConfig, Evidence, replay and comparison semantics.
Suite claims/reports are separate derived formats, not new agent-facing tools.
All planned trials remain accountable after failures. A stopped suite and a
failed comparison are distinct, and neither authorizes an upgrade.

The reviewable delivery builds on the SPEC-0055 task summaries merged in PR #7.
No main merge, stable release or live evaluation follows from
this decision. Broader B09/B10 and remote/storage guarantees remain open.
