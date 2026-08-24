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
