# ADR-0059: Read-only Offline Suite Evidence Audit

- Status: Accepted for the bounded offline subset
- Date: 2026-09-29
- Contracts: SPEC-0059 through SPEC-0062

Accept strict stored metadata admission, bounded whole-suite directory diagnosis,
independent replay comparison and CLI rendering/gates. This extends ADR-0056's
writer without changing its output schema or task grading. Existing successful
suite artifacts remain readable. No repair, resume, deletion, provider access,
new agent-facing tools or authenticity claims are introduced.

Separate structural validity, publication, evidence completeness, report agreement
and regression eligibility. A plausible saved report must never authorize a gate
without rereading and replaying the planned trial evidence. Partial runs remain
diagnostic and cannot claim that their runtime stop cause has been reconstructed.

This accepts a B09/B10-offline usability and evidence-integrity subset only;
live evaluation, external feedback and stable qualification remain open.
