# SPEC-0013 — Security, Trust Domains & Network Boundary

> **Status:** Proposal  
> **Implementation baseline:** source-reported partial foundation  
> **Verification status:** unverified  
> **Default security profile:** local / synthetic / loopback-only

## 1. Security objective

MCP State Twin must preserve independent trust boundaries between:

```text
Agent / Host
    -> Agent Data Plane

Harness / Operator
    -> Simulation Control Plane

Optional Authorized Harness
    -> Recorder / Differential Upstream
```

A prompt is never an authorization boundary.

## 2. Trust-domain invariant

### ST-SEC-R001
Control operations such as snapshot, fork, reset, clock advancement, fault configuration and hidden expected state MUST NOT be exposed as model-visible MCP business tools.

### ST-SEC-R002
An authorization grant for the data plane MUST NOT automatically authorize the control plane.

### ST-SEC-R003
Control-plane credentials MUST be independently configurable and revocable.

### ST-SEC-R004
Branch IDs, snapshot IDs and call IDs are identifiers, not bearer secrets.

## 3. Default local profile

Until a remote profile is independently qualified, the recommended profile is:

```yaml
network:
  bind: loopback
  production_egress: false
data:
  classification: synthetic-only
control_plane:
  authentication: required
data_plane:
  tls: not-required-on-loopback-profile
remote_multi_tenancy: unsupported
```

### ST-SEC-R010
The local hermetic CI profile MUST deny unexpected network egress.

### ST-SEC-R011
Synthetic fixtures MUST be the default examples and release-test data.

### ST-SEC-R012
A local control token does not make the server Internet-safe.

## 4. Secret handling

### ST-SEC-R020
Authorization headers, cookies, bearer tokens and provider secrets MUST NOT enter canonical evidence.

### ST-SEC-R021
Operational logs MUST redact secrets.

### ST-SEC-R022
Recorder redaction MUST happen before durable storage.

### ST-SEC-R023
Crash dumps/diagnostics that may contain secrets require an explicit operational policy before remote production usage.

## 5. OAuth/resource-server boundary

Current MCP authorization design uses resource/audience-aware OAuth semantics.

### ST-SEC-R030
If State Twin accepts an OAuth token, the token MUST be valid for State Twin as the protected resource.

### ST-SEC-R031
State Twin MUST NOT pass an inbound host token unchanged to an upstream API.

### ST-SEC-R032
If State Twin later invokes an upstream, that upstream credential is a distinct authorization relationship and credential.

### ST-SEC-R033
Token audience/resource mismatch MUST fail closed.

## 6. Host approval is not authorization

Codex/Claude or another host may ask a human to approve a tool call.

This is useful safety UX, but:

### ST-SEC-R040
Host approval MUST NOT bypass server authorization.

### ST-SEC-R041
Server authorization MUST be enforced even when the host reports the tool call as approved.

### ST-SEC-R042
A denied server authorization MUST be observable as a security/authorization result, not a domain result.

## 7. Remote profile — future

A remotely exposed profile cannot be called production-ready before it has:

- TLS;
- authentication;
- authorization;
- token audience/resource validation;
- credential rotation/revocation;
- request/body limits;
- rate limits;
- SSRF controls;
- Origin/Host validation where applicable;
- DNS rebinding considerations;
- egress policy;
- audit;
- incident runbook;
- security tests;
- dependency/supply-chain process.

## 8. SSRF and upstream access

### ST-SEC-R050
User/TwinSpec-provided arbitrary URLs MUST NOT become unrestricted server-side fetch targets.

### ST-SEC-R051
External JSON Schema `$ref` remains denied in hermetic mode.

### ST-SEC-R052
Recorder/differential upstream endpoints require an explicit allowlist/profile.

### ST-SEC-R053
Production endpoints MUST never be selected implicitly because a test endpoint is missing.

## 9. Path/archive security

Future bundles/recordings introduce filesystem inputs.

### ST-SEC-R060
Archive extraction MUST reject path traversal and absolute paths.

### ST-SEC-R061
Symlink escape MUST be rejected.

### ST-SEC-R062
Bundle extraction MUST be size/file-count bounded to resist archive bombs.

## 10. Prompt/tool injection

Tool descriptions and annotations may be untrusted.

### ST-SEC-R070
State Twin MUST treat business tool metadata as data, not trusted instructions to the control plane.

### ST-SEC-R071
Agent-supplied strings MUST never be interpreted as control commands.

### ST-SEC-R072
Hidden expected state/control policies MUST not be inserted into model-visible tool descriptions merely to influence behavior.

## 11. Resource abuse

Security review must include:
- oversized JSON;
- deep nesting;
- CEL cost abuse;
- schema complexity;
- huge diffs;
- huge reports;
- fork explosion;
- scheduler/fault explosion;
- cassette/bundle expansion;
- concurrent connection/call exhaustion.

Limits are normatively owned by SPEC-0015.

## 12. Threat catalog

At minimum test/review:

```text
prompt injection
malicious tool description
confused deputy
token passthrough
wrong audience
token replay
SSRF
DNS rebinding
Host/Origin abuse
path traversal
symlink escape
external schema fetch
oversized JSON
deep recursion
CEL resource abuse
branch/snapshot enumeration
cross-branch leakage
evidence leakage
recorder secret leakage
malicious cassette
malicious bundle
archive bomb
dependency compromise
```

## 13. Multi-tenant warning

Remote multi-tenancy adds:
- tenant identity;
- RBAC;
- cross-tenant isolation;
- quotas;
- encryption;
- data lifecycle;
- backup;
- incident handling;
- noisy-neighbor controls;
- distributed storage.

It requires a separate RFC and threat model.

## 14. Security verification stages

### v0.1 local
- loopback bind tests;
- control auth;
- hidden-control-surface tests;
- egress deny;
- secret policy;
- fuzz/resource bounds.

### v0.2 recorder/fidelity
- redaction tests;
- malicious cassette/bundle tests;
- upstream allowlist tests.

### remote profile
- auth/TLS/OAuth;
- SSRF;
- rate limits;
- external security review/gate.

## 15. Feasibility

**High** for a hardened local profile.  
**Medium-high** for a properly scoped remote OAuth profile.  
**Remote multi-tenancy is a separate architecture problem, not a small extension.**
