# SPEC-0060: Bounded Read-only Suite Directory Inspection

Status: accepted by [ADR-0059](ADR-0059-SUITE-EVIDENCE-AUDIT.md).

## Contract

InspectSuite(ctx, root, out) and eval suite-inspect inspect a trusted, quiescent
root-relative directory. Missing directory => not_started; empty or valid
claim-only directory => incomplete_or_running. Neither implies a running process.
Malformed metadata/members => invalid. Do not create or mutate any file.

Enumerate at most 37 root entries (four metadata names plus at most 32 planned
trial directories plus overflow). After plan admission accept only exact planned
directories. Each trial admits at most four known evidence files plus overflow;
reject unknown names, symlinks, directories in file slots and nonregular files.
Inspect all path components. Never echo unexpected filenames. Before replay,
bound total observed trial artifact sizes to 128 MiB. Existing per-file limits
also apply. This stat admission is not an atomic snapshot; concurrent writers
are outside the trust contract. Existing replay/comparison per-operation bounds
remain active; repeated reads mean the size bound is not a total IO/RSS quota.

Use existing InspectDirectory for each trial in plan order, retaining incomplete,
partial and invalid states and staging residue. Missing trials stay in planned
counts. Use a cooperative 120-second outer deadline; propagate cancellation
rather than return a success-shaped partial result. No live API, resume, repair,
staging promotion or deletion. snapshotAtomic and resumeAllowed remain false.

## Acceptance

Cover missing/empty/claim-only/plan-only/complete/interrupted directories, unknown
members, oversized files, symlinks, invalid child claims, pending residue and
cancellation. Directory contents before/after inspection must be unchanged.
