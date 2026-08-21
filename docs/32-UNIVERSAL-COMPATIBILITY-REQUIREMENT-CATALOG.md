# Universal Compatibility — Normative Requirement Catalog

> **Status:** Proposal / Unverified  
> **Purpose:** Consolidate the new compatibility framework into stable requirement candidates for future traceability.

## Compatibility semantics

### ST-HCOMP-R001
Compatibility MUST be scoped to a Host Profile and capability set.

### ST-HCOMP-R002
Protocol conformance MUST NOT be used as proof of model-visible surface equivalence across hosts.

### ST-HCOMP-R003
Canonical server surface and host/model-visible surface MUST be separately representable.

### ST-HCOMP-R004
Known host name/schema/output transformations MUST be evidence.

### ST-HCOMP-R005
Unknown host transformation behavior MUST remain `unknown`.

### ST-HCOMP-R006
Compatibility MUST NOT transit across products from the same provider.

### ST-HCOMP-R007
A host profile MUST include observation date and applicable version identifiers.

### ST-HCOMP-R008
A materially changed host/profile MUST invalidate or stale prior evidence according to policy.

## Portable tools profile

### ST-PORT-R001
The baseline live-agent profile MUST depend only on MCP tools and world/evaluation semantics.

### ST-PORT-R002
Prompts/resources/roots/elicitation/Tasks/MRTR MUST be optional capability profiles.

### ST-PORT-R003
Canonical tool identity MUST survive host renaming/prefixing through an explicit mapping.

### ST-PORT-R004
Projected schema MUST have its own digest when observable.

### ST-PORT-R005
Lossy schema projection MUST NOT be classified as lossless compatibility.

### ST-PORT-R006
Tool descriptions MUST NOT leak expected state or evaluator controls.

### ST-PORT-R007
A portable scenario SHOULD expose only a declared task-relevant surface/distractor set.

### ST-PORT-R008
Surface readiness MUST be established before live-agent scoring begins.

## Adapter requirements

### ST-ADAPT-R001
Host adapters MUST NOT mutate Twin business semantics.

### ST-ADAPT-R002
Adapter normalization MUST preserve raw observations.

### ST-ADAPT-R003
Capability fields MUST declare provenance: documented/probed/inferred/unknown.

### ST-ADAPT-R004
Inferred capability alone MUST NOT support a verified compatibility claim.

### ST-ADAPT-R005
Adapter fixture tests MUST cover all nontrivial projection rules.

### ST-ADAPT-R006
Host-specific config syntax MUST remain outside TwinSpec core.

## Isolation

### ST-ISO-R001
Evaluated agents MUST NOT receive the control-plane token.

### ST-ISO-R002
Private assertions/expected state MUST NOT be agent-readable in strict evaluation.

### ST-ISO-R003
The isolation profile MUST state whether built-in non-MCP tools are enabled.

### ST-ISO-R004
Workspace identity SHOULD be captured for coding-agent runs.

### ST-ISO-R005
Host instruction/config artifacts that may affect behavior SHOULD be digested where observable.

### ST-ISO-R006
Agent memory reset guarantee MUST be `true`, `false`, or `unknown`.

### ST-ISO-R007
Unknown memory reset prevents a claim of fully independent repeated trials.

### ST-ISO-R008
A run with evaluator-secret exposure MUST be invalidated.

## Episode

### ST-EPISODE-R001
Every live-agent run MUST have an Episode identity.

### ST-EPISODE-R002
Episode mode MUST distinguish blind evaluation from coaching/curriculum.

### ST-EPISODE-R003
Episode budgets MUST be explicit when used for scoring or termination.

### ST-EPISODE-R004
Terminal state/invariants SHOULD be primary success criteria.

### ST-EPISODE-R005
A host/infrastructure failure MUST not count as an agent task failure.

### ST-EPISODE-R006
Cleanup MUST revoke/delete ephemeral credentials/resources where used.

## Capability uplift

### ST-UPLIFT-R001
Coaching feedback MUST be labeled and excluded from blind-evaluation claims.

### ST-UPLIFT-R002
Capability curriculum levels MUST NOT be presented as a universal intelligence scale.

### ST-UPLIFT-R003
Capability uplift claims MUST distinguish agent-system behavior improvement from base-model weight changes.

### ST-UPLIFT-R004
Unsafe/destructive side effects SHOULD be evaluated separately from task success.

## Scenario family

### ST-FAM-R001
Family generator version + seed MUST deterministically identify an instance.

### ST-FAM-R002
Semantic generator changes require a version change.

### ST-FAM-R003
Expected properties MUST derive from accepted Twin semantics.

### ST-FAM-R004
Metamorphic transforms MUST declare whether they preserve or alter the expected semantic outcome.

### ST-FAM-R005
Held-out seeds/instances MUST not be exposed to the agent under private evaluation.

### ST-FAM-R006
Scenario generators MUST remain bounded by resource governance.

## Cross-protocol

### ST-XPROTO-R001
MCP, ACP and A2A trust/identity boundaries MUST remain distinct.

### ST-XPROTO-R002
Control-plane authorization MUST never be inherited through ACP/A2A.

### ST-XPROTO-R003
A non-MCP bridge MUST NOT produce an MCP conformance claim.

### ST-XPROTO-R004
Cross-protocol delegation evidence SHOULD preserve available origin/agent/call identity without requiring hidden reasoning.

## Compatibility CI

### ST-COMPATCI-R001
Public verified compatibility claims MUST reference fresh evidence.

### ST-COMPATCI-R002
Stale evidence MUST not remain displayed as fresh/verified.

### ST-COMPATCI-R003
Real-host integration failures due to missing credentials MUST be `blocked/skipped`, never PASS.

### ST-COMPATCI-R004
Expected failures MUST be scoped and reviewed.

### ST-COMPATCI-R005
Unexpected PASS of a known expected failure SHOULD trigger compatibility review.

## Multimodal/artifacts

### ST-ART-R001
Large binary artifacts SHOULD be content-addressed and referenced rather than embedded in canonical JSON state.

### ST-ART-R002
Artifact media type, size and digest MUST be explicit.

### ST-ART-R003
Host artifact transforms MUST be evidence.

### ST-ART-R004
Artifact imports/outputs MUST obey path, MIME and resource safety policies.

## Claim rule

No requirement above is `verified` until it appears in the requirement traceability matrix with executable evidence.
