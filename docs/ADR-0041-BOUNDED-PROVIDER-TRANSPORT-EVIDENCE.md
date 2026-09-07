# ADR-0041: Bounded Responses Transport and Separate Live Evidence

- Status: Accepted for the experimental subset below
- Date: 2026-09-08
- Depends on: ADR-0038, ADR-0039, ADR-0040
- Release: contract-tested readiness; actual provider compatibility unverified

## Decision

Accept [SPEC-0041](SPEC-0041-PROVIDER-TRANSPORT-AND-EVIDENCE.md). Use a single
fixed HTTPS endpoint, no redirect/proxy/custom route, sequential POSTs and no
automatic retry. Reuse the transport-free codec and official in-process MCP
world path, not the old native remote-MCP provider-smoke path.

Maintain separate live RunConfig/Episode/Evidence kinds and a separate verifier.
Successful HTTP delivery, tool commit, task success, cleanup and evidence
completeness are distinct facts. Unknown provider acceptance cannot become a
zero-effect/zero-cost result. Terminal closure replays the local world, never
the model. Public receipts omit raw request/response, private IDs and headers.

Contract tests label their artifacts `contract-test`; production execution
labels its artifacts `provider-live`. Neither unsigned label authenticates
provider origin. Do not feed live artifacts to the existing mock comparator.

## Not accepted

No production tool passthrough, general HTTP/MCP client, shell or computer-use
adapter; no background Responses tasks, polling, remote cancellation, automatic
continuation after crash, exactly-once billing/effects, signed receipts or full
storage crash-recovery guarantee. Real model scoring and product-host support
require independent future evidence.
