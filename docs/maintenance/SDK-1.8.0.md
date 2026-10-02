# MCP Go SDK 1.8.0 migration

2026-10-02: PR #15 upgraded the module but left `MCPGoSDKVersion` at 1.7.0.
The failed run 36850413447 therefore rejected the declared protocol evidence.
The correction updates the compiled declaration alongside the existing module
pin; it does not disable the pin test or add optional protocol capabilities.

The [upstream release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0)
retains the negotiated protocol revisions and includes transport bounds and
session/cancellation cleanup fixes. Existing modern/legacy wire, lifecycle,
hermetic and conformance tests remain the compatibility gates.

The same failed pin test was present in the Windows job log, although that job
reported success: a subsequent successful build replaced the native command exit
status. Platform testing and building now use separate steps, so test failure
cannot be hidden by build success. The accepted SQLite update from main is retained.

No historical fixture that explicitly describes an older SDK is rewritten.
Candidate CI, rather than the original failed run, establishes migration results.
