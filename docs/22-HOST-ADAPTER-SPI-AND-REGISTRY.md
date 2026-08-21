# Host Adapter SPI & Compatibility Registry

> **Status:** Proposal / Unverified  
> **Purpose:** Make host integration systematic, testable and replaceable as agent products evolve.

## 1. Architecture

```text
Episode Orchestrator
        |
        v
Host Adapter SPI
        |
        +--> Codex Adapter
        +--> Claude Code Adapter
        +--> Gemini CLI Adapter
        +--> Copilot Adapter
        +--> Cursor Adapter
        +--> Windsurf Adapter
        +--> Cline Adapter
        +--> Amazon Q Adapter
        +--> JetBrains/Junie Adapter
        +--> Zed/ACP Adapter
        +--> OpenCode Adapter
        +--> Custom MCP Adapter
```

Adapters do not own Twin world semantics.

## 2. Conceptual SPI

An adapter SHOULD conceptually implement:

```text
Describe()
ProbeCapabilities()
ResolveProfile()
ProjectSurface()
RenderConfiguration()
PrepareRun()
StartOrAttach()
WaitUntilSurfaceReady()
InvokeObjective()
CollectHostObservation()
NormalizeObservation()
Cleanup()
```

Names are conceptual, not a mandated Go API yet.

## 3. Adapter constraints

### ST-ADAPT-R001
Adapter MUST NOT mutate TwinSpec business behavior.

### ST-ADAPT-R002
Adapter MAY configure transport, authentication, host permissions, tool filters and host-specific surface projection.

### ST-ADAPT-R003
Adapter MUST preserve raw observations before normalization.

### ST-ADAPT-R004
Normalization MUST be versioned and auditable.

### ST-ADAPT-R005
Unknown capability remains unknown.

### ST-ADAPT-R006
A documentation-derived capability is not equivalent to a runtime-probed capability.

## 4. Capability probe

Probe output:

```yaml
probe:
  hostVersion: ...
  transport:
    streamableHttp: supported
    stdio: supported
  mcp:
    tools: supported
    prompts: unknown
    resources: unknown
  approval:
    mode: ...
  projection:
    toolNameTransform: ...
    schemaTransform: ...
  limits:
    maxToolsVisible: ...
```

Every field records provenance:

```text
documented
probed
inferred
unknown
```

`inferred` cannot support a verified compatibility claim.

## 5. Registry layout

Recommended repository structure:

```text
compat/
  hosts/
    codex/
      profile.yaml
      sources.md
      known-limitations.md
      projection-fixtures/
      runner/
    claude-code/
    gemini-cli/
    github-copilot-vscode/
    github-copilot-cloud-agent/
    cursor/
    windsurf/
    cline/
    amazon-q/
    jetbrains-junie/
    zed/
    opencode/
    custom-mcp/
```

## 6. Registry entry

Each host entry MUST separate:

```text
documented capabilities
runtime probe evidence
adapter code
known limitations
real scenario evidence
freshness
```

## 7. Host identity

Host identity can include:

```yaml
provider: google
product: gemini-cli
version: ...
channel: stable
platform: ...
configurationSchemaVersion: ...
observedAt: ...
```

Do not assume product name alone defines behavior.

## 8. Workspace identity

Coding-agent evaluation requires workspace identity:

```yaml
workspace:
  repoCommit: ...
  worktreeDigest: ...
  instructionFiles:
    - path: AGENTS.md
      digest: ...
  skillsPluginsHooks:
    - ...
  builtInToolsEnabled:
    - shell
    - filesystem
  networkPolicy: ...
```

This information may materially affect agent behavior.

## 9. Instruction provenance

Host-specific instruction files may include:
- AGENTS.md;
- CLAUDE.md;
- GEMINI.md;
- project rules;
- system prompts/config;
- skills/plugins/hooks.

Where observable and permitted, record their content digest or normalized configuration identity.

Do not include hidden provider system instructions that cannot be observed.

## 10. Memory state

Host profile includes:

```yaml
memory:
  freshProcess: true|false|unknown
  persistentMemoryResetGuaranteed: true|false|unknown
  previousConversationCleared: true|false|unknown
```

A fresh State Twin fork does not imply a fresh agent.

## 11. Built-in tools

Record built-in non-MCP tools:

```yaml
builtIns:
  shell: enabled
  filesystem: enabled
  web: disabled
  editor: enabled
```

Strict cross-host evaluation may require disabling or constraining built-ins when possible.

## 12. Approval/retry policy

Adapters record:
- ask/allow/deny;
- auto-approval;
- per-tool policy;
- automatic retries;
- reconnect behavior.

These can change world call traces.

## 13. Adapter fixtures

Each adapter SHOULD have fixture tests for:
- config generation;
- tool-name projection;
- schema projection;
- capability parsing;
- output transforms;
- permission state;
- error classification.

Fixture tests do not replace real-host smoke.

## 14. Staleness

An adapter/profile becomes stale when a material identity dimension changes:
- host major behavior/version;
- protocol support;
- config syntax;
- schema projection;
- permission model;
- output transform.

Age alone is not the only criterion.

## 15. Feasibility

**High.**

This layer is the practical mechanism that lets the project support many agent products without contaminating the core runtime.
