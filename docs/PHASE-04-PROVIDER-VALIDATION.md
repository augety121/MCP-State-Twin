# Phase 4: Secure Provider and Host Validation

- **Target:** `v0.3.x`
- **Entry:** Phase 3 complete and SPEC-0020 remote-staging profile evidenced
- **Authority:** SPEC-0019, SPEC-0020 and SPEC-0021

## Required scope

- ephemeral TLS-protected synthetic MCP endpoint;
- isolated run identity, secrets, storage and teardown;
- OpenAI Responses exact API profile live smoke;
- Anthropic Messages MCP connector exact API profile live smoke;
- provider cancellation/timeout/negative paths where supported;
- sanitized, content-addressed ProviderSmokeReport;
- exact HostProfile and adapter versions;
- evidence TTL, invalidation and compatibility matrix update;
- bounded request, response, time and monetary cost;
- explicit separation from ChatGPT, Codex, Claude and Claude Code claims.

## Exit evidence

- current live report for each claimed API family;
- at least one successful MCP call and one failure/cancellation path;
- all SPEC-0020 security-negative tests;
- no credentials, raw prompts/transcripts/responses or private IDs in artifacts;
- teardown confirmation;
- compatibility validator accepts the report on the same revision;
- stale evidence automatically loses `verified` state.

Product UI/desktop/CLI hosts require their own later profile and evidence. API
evidence never closes a product-profile gate.
