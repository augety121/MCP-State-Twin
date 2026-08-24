# ADR-0017: Host Compatibility Report Admission

- **Status:** Accepted
- **Date:** 2026-08-24
- **Decision owners:** repository maintainers
- **Accepts:** the report-admission subset of SPEC-0006
- **Does not accept:** a provider compatibility claim

## Context

Provider smoke evidence cannot be a screenshot, prose statement, or unchecked
YAML file. A remote OpenAI or Anthropic run also has a different trust boundary
from the local hermetic runtime. The repository had a proposed report shape but
no concrete decoder, validator, size limit, secret admission rule, or CLI.

## Decision

Adopt `statetwin.dev/host-compatibility-report/v1alpha1` as a strict evidence
artifact with these admission rules:

- one YAML document, at most 1 MiB, known fields only, with aliases, anchors,
  and explicit tags rejected;
- immutable runtime revision and lowercase SHA-256 identities;
- an exact compatibility profile instead of a provider-wide claim;
- finite provider, tool-call, retry, repeated-call, wall-time, and trace limits;
- canonical outcome classes and profile-specific required checks;
- an irreversible provider request-ID digest for API profiles, never the raw ID;
- a reviewed deployment-profile digest for every non-loopback endpoint;
- verified claims require an exact observed surface, successful assertions, a
  completed outcome, and an expiry date; and
- credential-like, private-key, or email patterns and `secretsDetected: true`
  fail admission. Operators remain responsible for excluding all other personal
  data from synthetic-only reports.

`statetwin compatibility validate --report <path>` is the supported admission
command. The deterministic runtime does not import an OpenAI or Anthropic SDK.

## Non-claims

This decision does not:

- perform a provider request;
- prove ChatGPT, OpenAI API, Claude, or Claude Code compatibility;
- make a local endpoint reachable from a provider-hosted client;
- approve a remote deployment profile;
- permit raw provider transcripts, credentials, request IDs, or user data in
  the repository; or
- close RFC-0002's provider-smoke gate without real, reproducible reports from
  both required provider families.

## Consequences

- SPEC-0006's report schema is now executable rather than illustrative.
- Live harnesses may be implemented outside the state engine and must emit an
  artifact accepted by this validator.
- The public compatibility matrix must be generated only from admitted reports;
  an empty evidence set means no provider is verified.
