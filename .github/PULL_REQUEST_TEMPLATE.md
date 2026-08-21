## Summary

<!-- What changed and why? Link the issue or ADR. -->

## Scope and status

- [ ] This change is implementation, not proposal-only prose.
- [ ] Any new capability has an accepted ADR/SPEC or explicitly updates one.
- [ ] `IMPLEMENTATION-STATUS.md` is updated for public behavior changes.
- [ ] README and changelog claims are consistent across language variants.

## Safety and determinism

- [ ] Hermetic mode has no upstream passthrough or production write path.
- [ ] Control-plane tools remain unavailable to agents.
- [ ] Unknown behavior fails explicitly; no invented success.
- [ ] Inputs, outputs, expressions and state remain within resource budgets.
- [ ] No credentials, private traces, or personal data were added.

## Verification

```text
go test ./...
go vet ./...
go test -race ./...  # Linux CI evidence required
```

Additional commands and CI links:

## Release impact

- [ ] No release impact.
- [ ] Patch release candidate.
- [ ] Minor release candidate.
- [ ] RFC-0002 / migration notes updated.
