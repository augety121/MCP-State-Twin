# ADR-0054: Scoped Host Report Assessment

- Status: Accepted; read-only B15 declaration/time gate
- Date: 2026-09-26
- Contract: [SPEC-0054](SPEC-0054-HOST-REPORT-SCOPE-ASSESSMENT.md)

Extend `compatibility assess` with an optional explicit target and a separate
`--require-current` gate. Keep `--require-fresh` time-only for backward
compatibility. Matching scope must not hide expiry, regression, experimental
status, missing evidence or unsupported provider provenance.

Reports expose ordered field identifiers, not raw expected/observed values.
The comparison policy is versioned separately from TTLs. Successful matching
never grants publication authority; outputs still say provenance not verified.
The same report and target can be fabricated, so this is not an attestation,
independent review, environment inspection or provider compatibility certificate.

Existing time-only output is unchanged. Target-mode output adds an explicitly
scoped assessment with no automatic data migration, network requests or writes.
