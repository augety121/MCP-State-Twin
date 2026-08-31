# SPEC-0019: HostProfile and Live Compatibility Evidence

- **Status:** Accepted via ADR-0021; implementation is not complete
- **Format:** `statetwin.dev/host-profile/v1alpha2`
- **Target:** Phase 4 / v0.3
- **Depends on:** SPEC-0003, SPEC-0004, SPEC-0006, SPEC-0020

## 1. Purpose

This specification prevents protocol support, API adapter tests and product
compatibility from being collapsed into one boolean claim. Compatibility is an
evidence-backed relationship between an exact host profile and a reviewed
runtime revision.

## 2. Required identity

A HostProfile MUST bind:

```yaml
apiVersion: statetwin.dev/host-profile/v1alpha2
kind: HostProfile
metadata:
  id: openai-responses-remote-mcp-v1alpha1
subject:
  family: openai-api
  product: responses-api
  version: exact-observed-version-or-date
transport:
  protocolProfile: mcp-modern-2026-07-28
  mode: remote-mcp
adapter:
  name: openai-responses
  version: exact-build
securityProfile: remote-staging-v1alpha1
capabilities: {}
evidence:
  runtimeRevision: git-sha
  observedAt: RFC3339
  freshnessTTL: 30d
  reportDigest: sha256:...
```

Host, API family, product, transport, MCP profile, adapter and security profile
MUST be explicit. Missing identity fields make the claim `unverified`.

## 3. Claim states

| State | Meaning |
|---|---|
| `unsupported` | the profile is intentionally not supported |
| `unverified` | design or adapter exists without admissible live evidence |
| `experimental` | live evidence exists but the contract is preview/unstable |
| `verified` | all required evidence is current and accepted |
| `regressed` | a previously verified profile now fails |
| `stale` | evidence exceeded its TTL or a bound version changed |

`verified` MUST NOT be inferred from SDK compilation, MCP conformance, mock
servers, another provider family or another product in the same family.

## 4. Capability vocabulary

Each capability is independent and uses `supported`, `unsupported`, `beta` or
`unknown`:

- remote MCP tools;
- tool discovery observed;
- background execution;
- retrieve/poll;
- remote cancel;
- cancellation idempotency;
- provider idempotency key;
- request identifier location;
- streaming;
- tool approval behavior;
- data-retention eligibility.

The existence of a cancel endpoint establishes `remoteCancel: supported`; it
does not establish `cancelIdempotent: documented`. Documentation and evidence
for each field MUST be separate.

## 5. Live evidence admission

A live report MUST include:

1. exact runtime commit and adapter version;
2. exact provider/API profile and requested model identifier;
3. MCP endpoint digest, not a credential-bearing URL;
4. synthetic prompt digest, not the raw prompt;
5. tool-list and tool-call observations;
6. terminal provider status;
7. a bounded, sanitized error class on failure;
8. report and response digests;
9. execution timestamp and freshness TTL;
10. security profile identity;
11. positive tool-call path;
12. at least one negative, timeout or cancellation path.

The report MUST NOT contain provider keys, authorization headers, raw provider
request IDs, raw prompts, transcripts, response bodies or production data.

## 6. Freshness

API profiles default to a 30-day evidence TTL. UI/product profiles default to
14 days because their behavior may change independently of API versions. A
profile becomes stale immediately when any bound host version, adapter,
protocol profile, tool surface or security profile changes.

These TTLs are project claim policy, not a statement about provider stability.

## 7. Profile separation

The following are separate claims and MUST have separate evidence:

- generic MCP client;
- OpenAI Responses API;
- ChatGPT MCP integration;
- Codex;
- Anthropic Messages API MCP connector;
- Claude web/desktop;
- Claude Code;
- any third-party Agent host.

## 8. Current admission

The repository currently has OpenAI Responses and Anthropic Messages mock
contract tests and an opt-in live harness. They establish request/response
shape only. Until dated reports satisfy this SPEC and SPEC-0020, both API
profiles remain `unverified`; product profiles remain `unverified` or
`unsupported` as recorded in `COMPATIBILITY-MATRIX.md`.
