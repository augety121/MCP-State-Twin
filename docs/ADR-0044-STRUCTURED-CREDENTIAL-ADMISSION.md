# ADR-0044: Structured Credential Admission

- Status: Accepted security correction
- Date: 2026-09-12
- Evidence: new storage refusal test reproduced missed JSON credential fields

Accept [SPEC-0044](SPEC-0044-STRUCTURED-CREDENTIAL-ADMISSION.md).
The existing text-pattern detector recognized `api_key=value` but missed quoted
JSON fields and escaped field names. Add bounded structural JSON scanning and
conservative whole-JSON operational redaction, while preserving legitimate
Task authorization rule arrays and numeric `tokens` usage metadata.

All existing callers retain their fail-closed admission behavior; no secret is
redacted into a supposedly equivalent replay artifact. Sensitive artifact data
is refused before file creation. This is a finite documented policy, not a
claim of arbitrary-secret/PII detection or automatic classification of real data.
