# SPEC-0062: Suite Audit CLI, Gates and Readable Reports

Status: accepted by [ADR-0059](ADR-0059-SUITE-EVIDENCE-AUDIT.md).

## Commands

eval suite-inspect --root ROOT --out RELATIVE_DIR --format json|markdown prints
safe directory diagnostics. JSON is default. It exits zero for valid diagnostic
states including missing/incomplete/regressed, nonzero for invalid or read errors.

eval suite-verify uses the same flags and full audit, but exits zero only when
regressionGatePassed is true. It prints available diagnostics before a failed
gate; stdout failures take precedence and cannot be swallowed. Unknown flags,
formats and positional arguments are rejected before inspecting artifacts.

Markdown shows directory state, report verification, planned/complete trial
counts, residue, finite problem codes, ordered per-trial state/replay summaries
and the independently computed comparison. It clearly disclaims automatic
upgrade and provenance. Never render saved free text, unexpected paths or raw
responses. No external renderer/service or HTML required.

## Acceptance and documentation

CLI integration covers successful suite, regression with inspect=0/verify!=0,
missing output, tampered report, residual staging, both formats, argument errors
and failed stdout. Guide includes fresh-run verification, failure diagnosis and
why report.json alone is insufficient. Document that inspection is read-only,
not process liveness, an atomic snapshot or a recovery mechanism. Complete the
repository gofmt/vet/tests and exact-candidate CI, including race.
