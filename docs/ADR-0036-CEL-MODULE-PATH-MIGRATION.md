# ADR-0036: Migrate to the Declared CEL Module Path

- **Status:** Accepted
- **Date:** 2026-09-02
- **Related:** ADR-0003, SPEC-0036, ADR-0037

## Decision

Use `cel.dev/cel-go v0.32.0` with matching source imports and checksums.
Maintain the current bounded environment without registering new optional
extensions. Gate migration with explicit semantic/boundary vectors and normal
cross-platform, race, hermetic and protocol tests.

## Consequences

The update resolves the manifest-path cause reported by Dependabot without
an ignore rule or replacement shim. Successful source migration does not
itself prove a fresh Dependabot run passed. Library cost/error behavior may
change; reproducibility remains bound to the exact runtime revision.

The independently reproduced null conversion defect is corrected under
ADR-0037 and is not attributed to the new CEL dependency.
