# ADR-0043: Read-only Evidence Directory Inspection

- Status: Accepted experimental diagnostic contract
- Date: 2026-09-12
- Depends on: ADR-0039, ADR-0041, ADR-0042

Accept [SPEC-0043](SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md) and `eval inspect`.
Inspect the four fixed claim/staging/terminal leaves under a trusted local root,
validate claim/artifact relationships and replay eligible local world evidence.

The command MUST NOT write, delete, repair, promote staging, load a credential,
invoke a model or resume an attempt. Recognized incomplete/possibly running
directories remain distinct from invalid directories and published evidence.
Directory observations are not atomic and require quiescent input for reliable
interpretation. Provider provenance remains not proven.

This is not a replacement for standalone artifact verification, a distributed
recovery agent, a live-process detector, a retention/GC policy or task-success
certification. The old mock comparator's artifact-level semantics stay unchanged.
