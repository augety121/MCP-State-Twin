# Repository instructions

- Treat `docs/RFC-0001.md` and accepted ADRs as design authority.
- Never describe roadmap items as implemented without executable tests.
- Do not add upstream passthrough or production-write paths to hermetic mode.
- Do not expose snapshot, fork, reset, state inspection, or fault controls as
  agent-facing MCP tools.
- Keep TwinSpec expressions declarative and resource-bounded; arbitrary script
  execution is out of scope.
- Unknown behavior must fail explicitly rather than return invented success.
- Run `gofmt`, `go vet ./...`, and `go test -race ./...` for code changes.
- Never commit real credentials, production traces, or personal data.
