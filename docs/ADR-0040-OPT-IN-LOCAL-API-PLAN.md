# ADR-0040: Explicit Approval for a Local API Bridge

- Status: Accepted for the experimental subset below
- Date: 2026-09-08
- Basis: maintainer authorization to continue the reviewed implementation plan
- Release: development preview; no stable or live-compatibility promotion

## Decision

Accept [SPEC-0040](SPEC-0040-LIVE-PLAN-AND-APPROVAL.md): an independent,
closed live plan for one synthetic AgentTask, one verified TwinBundle and one
operator-selected OpenAI Responses model. This is B04/B05/B12 readiness for
B08, not completion of B08's actual model runs.

The plan generator MUST leave all approval flags false. Execution requires the
reviewed plan, an unexpired approval window, explicit `--allow-live`, a valid
credential and a new plan-specific output directory. No default model, endpoint
override, inferred spending permission, automatic retry or automatic recovery.
Request counts and output-token parameters are bounded; currency cost is
explicitly unknown. Maintainer instructions to keep developing are not API
spending approval.

This does not add an upstream route to hermetic tools. The new CLI lane performs
outbound model requests; its simulated business tools remain in-process and
have no production-write path. Native remote MCP and product hosts are separate.

## Alternatives and limits

Reusing old remote-MCP smoke plans would conflate two protocols and authority
surfaces. Silently permitting real model IDs in offline RunConfig would destroy
the mock/live boundary. Both are rejected.

Approval flags are trusted local operator declarations, not signatures or a
multi-user authorization service. Local directory claims prevent accidental
reuse under that root; copying a plan to a different root is not prevented.
No account-wide monetary limit, model availability or adoption claim follows.
