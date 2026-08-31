# SPEC-0020: Remote Staging Security Profile

- **Status:** Accepted via ADR-0021; not implemented as a complete profile
- **Profile:** `remote-staging-v1alpha1`
- **Target:** Phase 4 / v0.3
- **Depends on:** SPEC-0003, SPEC-0013 proposal requirements, SPEC-0019

## 1. Boundary

This profile permits live-provider validation against an ephemeral synthetic
MCP State Twin endpoint. It is not a production multi-tenant hosting contract.
The local hermetic profile remains the default.

## 2. Mandatory controls

A conforming deployment MUST provide:

- TLS for every non-loopback listener;
- distinct data-plane, control-plane and coordinator credentials;
- token audience, expiration and constant-time verification;
- per-run isolated branch/world identity;
- deny-by-default egress with an explicit provider allowlist;
- SSRF, redirect and DNS-rebinding defenses;
- request/header/body/response/rate/cost/time limits;
- runtime secret injection without Journal, state, log or artifact persistence;
- synthetic-only fixtures, prompts and tool results;
- bounded retention and automatic teardown;
- sanitized operational audit;
- cancellation propagation and terminal ambiguity reporting;
- dependency, secret and artifact scanning on the deployed revision.

Bearer authentication without TLS is allowed only on an explicit loopback IP.
A hostname that merely resolves to loopback is insufficient for the plaintext
exception.

## 3. Credential domains

Credentials MUST be separated:

```text
provider credential
MCP data-plane credential
simulation control credential
Episode coordinator credential
```

Possession of one credential MUST NOT authorize another plane. Credentials
MUST NOT appear in CLI arguments, URLs, Evidence, Bundle manifests or public CI
logs.

## 4. Network admission

Remote URLs MUST reject embedded credentials, fragments and unexpected query
parameters. Redirects MUST be disabled or revalidated against the same policy.
Resolved addresses MUST be checked at connection time. Link-local, loopback,
private, multicast and metadata-service ranges MUST be rejected unless the
profile explicitly declares and isolates them.

## 5. Tenant and run isolation

Phase 4 supports one synthetic evaluation run per isolated environment. It
MUST NOT claim general multi-tenancy. Branch IDs are not authorization
boundaries. World storage, Evidence output and credentials MUST be unique per
run and removed according to the declared retention policy.

## 6. Failure behavior

- authentication failure: fail closed without revealing token validity detail;
- TLS or certificate failure: no downgrade;
- egress-policy failure: no alternate endpoint;
- cleanup failure: mark the run failed and alert the operator;
- uncertain provider/tool effect: `COMMIT_UNKNOWN`, never inferred success;
- secret scanner finding: evidence rejected and release gate failed.

## 7. Required evidence

Before a HostProfile can become `verified`, the deployment must show:

- TLS-negative and plaintext-non-loopback rejection tests;
- credential separation tests;
- redirect/SSRF/DNS policy tests;
- request and cost limit tests;
- synthetic-data attestation;
- secret absence in state, Journal, report and CI artifact;
- teardown verification;
- dated deployment manifest digest.
