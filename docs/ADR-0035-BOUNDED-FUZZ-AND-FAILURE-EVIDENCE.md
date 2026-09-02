# ADR-0035: Count-bounded Fuzz Smoke with Preserved Failures

- **Status:** Accepted
- **Date:** 2026-09-02
- **Related:** I-12, I-16, ADR-0022, SPEC-0035

## Decision

Replace short wall-time fuzz search budgets with explicit iteration budgets,
one worker, independent watchdogs and an allowlisted synthetic-counterexample
artifact. Run the second target after a first-target failure while preserving
the failed job. Test wrapper failure propagation with a synthetic command.

## Rationale and alternatives

The observed deadline contains no crashing input. Ignoring that error,
disabling fuzz, downgrading SQLite without causal evidence, or repeated reruns
until green would discard evidence. Merely increasing a duration keeps the
same cancellation boundary. Count-bounded smoke removes that boundary from
normal termination; watchdogs still detect noncompletion.

This is a test-policy correction, not proof that the original timeout is
harmless or that every future fuzz run will finish successfully.
