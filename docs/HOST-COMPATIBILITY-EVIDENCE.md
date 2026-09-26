# Host Compatibility Evidence Procedure

**Status:** executable artifact admission implemented; live provider evidence
is still absent.

This procedure applies to `generic-mcp`, OpenAI-family, Anthropic-family, and
other named host profiles. It does not turn protocol support into a provider
compatibility claim.

## 1. Admission command

```text
statetwin compatibility validate --report report.yaml
```

The command accepts one strict
`statetwin.dev/host-compatibility-report/v1alpha1` document and prints its
canonical report digest. Unknown fields, oversized documents, YAML extensions,
mutable revisions, incomplete evidence checks, unbounded trials, raw
credential-like data, and invalid remote trust profiles fail closed.

SPEC-0051 also rejects inconsistent `exact` surface declarations, zero passed
assertions in verified claims, conflicting cancellation checks and decoded YAML
credential patterns. Output explicitly says `validationScope: structure-only`
and `publicationAllowed: false`. This remains structural admission, not proof
that the declared run occurred.

### Time-policy assessment (read-only)

```text
statetwin compatibility assess --report report.yaml --at 2026-09-26T00:00:00Z
statetwin compatibility assess --report report.yaml --at 2026-09-26T00:00:00Z --require-fresh
```

Replace the sample timestamp with the intended audit time. There is no implicit
clock. The earlier of explicit expiry and the profile cap wins: 14 days for
ChatGPT / Claude Code report profiles, 30 days for the other admitted profiles.
Expiry is exclusive. An observation after the audit time is `not_yet_valid`.
Plain assessment returns diagnostics even when expired; `--require-fresh` then
returns nonzero unless the declaration is verified and within its window.

This does **not** check bound versions against current configuration, authenticate
the provider, fetch referenced evidence, or authorize publication. Every assessment
keeps `scopeStatus: not_checked`, `provenance: not_verified` and
`publicationAllowed: false`. A fabricated in-window report can satisfy this
time-only check; it is not a complete CI compatibility gate. Source files are
never rewritten, and this command generates no file hashes. Full contract:
[SPEC-0052](SPEC-0052-HOST-REPORT-FRESHNESS.md).

## 2. Live-run prerequisites

Before a provider-hosted run:

1. freeze the runtime commit, TwinSpec, snapshot, scenario, prompt, and tool
   policy digests;
2. use only synthetic fixtures and an account with no production authority;
3. accept a separate remote deployment profile covering TLS, authentication,
   branch authorization, request limits, tenant isolation, audit retention, and
   complete denial of control-plane routes;
4. use finite budgets for provider requests, tool calls, retries, repeated
   calls, wall time, and trace bytes;
5. hash the provider request ID irreversibly and discard raw credentials and
   transcripts from committed artifacts; and
6. run the profile-specific checks required by SPEC-0006.

Provider-hosted MCP clients require a reachable remote HTTP endpoint. OpenAI's
Responses API documents MCP tools among its supported tool categories:
<https://developers.openai.com/api/reference/cli/resources/responses/methods/create>.
Anthropic's Messages MCP connector currently documents remote HTTP tool calls,
requires a public endpoint, and supports only the MCP tool-call subset:
<https://platform.claude.com/docs/en/agents-and-tools/mcp-connector>.

These links establish documented product capability only. They are not evidence
that either product has successfully used this repository.

## 3. Publication rules

- Store admitted reports under `evidence/host-compatibility/<profile>/` only
  after a real run.
- Never commit API keys, authorization headers, account identifiers, raw request
  IDs, provider transcripts, or production data.
- An `experimental` result records a dated smoke run. A `verified` result also
  needs reproducibility, an exact observed surface, and an expiry date.
- Re-run when the host version, model resolution, MCP profile, transport,
  runtime revision, surface, or remote deployment profile changes.
- A failed or ambiguous run remains failed; do not convert it into success by
  editing the report.

## 4. Current state

The report validator and CLI are implemented. No live OpenAI-family or
Anthropic-family report is committed, so RFC-0002's provider-smoke gate remains
open.
