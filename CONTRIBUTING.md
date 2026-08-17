# Contributing

MCP State Twin is currently an implementation-preview project. Contributions
should preserve the hard invariants in [RFC-0001](docs/RFC-0001.md) and should
not expand marketing claims beyond measured behavior.

## Development setup

Requirements:

- Go 1.26.x
- Git

```bash
go mod download
go test ./...
go build ./cmd/statetwin
go run ./cmd/statetwin validate --spec examples/issue-tracker/twin.yaml
```

Before submitting a change:

```bash
gofmt -w .
go vet ./...
go test -race ./...
```

## Design changes

Open an ADR or RFC change before implementing anything that changes:

- a hard invariant;
- TwinSpec semantics or canonical digests;
- snapshot/fork isolation;
- data-plane/control-plane visibility;
- failure taxonomy;
- fidelity-level meaning;
- supported MCP protocol behavior.

## Pull requests

A pull request should include:

1. the exact behavior changed;
2. automated tests, including a negative/failure test where relevant;
3. documentation updates;
4. compatibility and migration impact;
5. evidence for any new performance or fidelity claim.

Do not include production recordings, access tokens, cookies, personal data,
or third-party fixtures that cannot legally be redistributed.
