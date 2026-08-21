# Failure Mode & Edge-Case Catalog

> **Status:** Proposal seed catalog  
> **Purpose:** Prevent happy-path-only specifications.  
> **Important:** This catalog is intentionally extensible. No finite document can prove it contains every future failure.

## 1. Determinism

- host wall-clock leakage
- timezone database differences
- DST boundary behavior
- timestamp precision
- rounding policy
- PRNG algorithm change
- seed missing
- UUID randomness leak
- random iteration
- Go map order
- locale-sensitive sorting
- Unicode normalization
- floating-point representation
- `-0`
- NaN / Infinity
- JSON number precision
- JSON key order
- array order accidentally treated as set
- dependency version change
- compiler/runtime/platform variation
- scheduler tie
- retry duplicate
- cancellation race
- concurrent commit order
- hidden cache state

## 2. State model

- missing entity
- duplicate entity
- malformed fixture
- invalid entity key
- key collision
- broken reference
- sequence overflow
- huge entity
- huge state
- invariant failure
- postcondition failure
- invalid successful output after mutation
- rollback failure
- snapshot missing
- snapshot incompatible
- fork collision
- reset during mutation
- snapshot during mutation
- diff incompatible state versions
- scheduler state missing from snapshot
- entropy state missing from snapshot

## 3. Protocol

- unsupported version
- malformed per-request metadata
- missing protocol metadata
- server discovery failure
- direct first RPC without discover
- legacy initialize against modern-only endpoint
- modern RPC against legacy endpoint
- routing header mismatch
- malformed/duplicate JSON-RPC ID
- unknown method
- invalid params
- malformed result
- missing required result discriminator
- pagination cursor invalid/stale
- list cache stale
- tool schema invalid
- tool result invalid
- cancellation/disconnect
- deprecated SSE profile accidentally advertises modern protocol
- extension negotiation mismatch
- Tasks response without opt-in
- MRTR request to unsupported host
- authorization metadata mismatch

## 4. Tool behavior

- schema-valid but domain-invalid input
- precondition race
- unmodeled business case
- unmodeled error
- output schema mismatch
- duplicate call
- idempotency-key collision
- ambiguous timeout
- hidden state dependency
- authorization-filtered tool list
- tool description drift
- annotation drift
- host tool rename
- host aggregator collision
- host hides tool
- host tool search exposes late

## 5. Fault injection

- fail before validation
- fail before effect
- fail after effect
- fail after commit
- response lost
- retry after ambiguous commit
- repeated rate limit
- rate-limit clock boundary
- stale read
- delayed visibility
- duplicate delivery
- out-of-order visibility
- cancellation before commit
- cancellation after commit
- crash before audit
- crash after audit
- fault plan exhausted
- fault selector too broad
- fault selector matches zero
- fault+scheduled event tie
- fault counter fork semantics

## 6. Storage

- two writers same branch
- writers on different branches
- reader during write
- snapshot/write race
- reset/write race
- fork/reset race
- diff/write race
- branch head overflow
- WAL checkpoint starvation
- WAL growth
- DB busy/lock timeout
- disk full
- read-only filesystem
- permission loss
- I/O error
- process kill
- corrupt database
- wrong application ID
- future schema
- interrupted migration
- copied DB without WAL/SHM
- network filesystem misuse
- backup taken incorrectly
- restore digest mismatch

## 7. Recorder / replay

- secret in header
- secret in JSON body
- PII
- credential in error message
- unknown sensitive header
- redaction occurs too late
- redaction changes matching key
- cassette miss
- cassette collision
- stale cassette
- incompatible cassette format
- cassette path traversal
- enormous cassette
- upstream rate limit
- production endpoint accidentally selected
- upstream changed during capture

## 8. Differential fidelity

- upstream nondeterminism
- upstream unavailable
- upstream rate-limited
- upstream auth scope differs
- upstream surface changes mid-run
- sandbox differs from production
- test invalid
- state cannot be observed upstream
- upstream timing noisy
- accepted divergence expires
- coverage profile too broad
- AI-generated rule self-promoted
- legal restriction on recording/evidence
- production write accidentally attempted

## 9. Security

- prompt injection
- malicious tool description
- tool injection
- confused deputy
- token passthrough
- token wrong audience
- token replay
- credential leak
- bearer token in audit
- SSRF
- DNS rebinding
- Origin abuse
- Host header abuse
- path traversal
- symlink escape
- external schema `$ref`
- archive traversal
- zip bomb
- huge JSON
- deep JSON
- schema bomb
- CEL cost abuse
- fork exhaustion
- event/fault explosion
- report exfiltration
- cross-branch leakage
- cross-tenant leakage in future mode
- malicious bundle
- dependency compromise

## 10. Host evaluation

- host silently filters tools
- host tool search/deferred loading
- host permission prompt
- approval denied
- approval timeout
- host automatic retry
- host truncates output
- host stores result as file
- host summarizes result
- multiple calls in one model request
- parallel calls
- subagent call attribution unavailable
- model alias changes underneath
- host version changes
- connector beta changes
- context compression
- persistent host memory
- max-turn exhaustion
- cost/budget exhaustion
- network timeout
- host crash
- local config precedence changes
- remote auth expiration
- host does not call `server/discover`
- host protocol version unknown

## 11. Evidence

- incomplete report
- duplicate event
- missing event
- event order ambiguity
- digest mismatch
- unsupported schema version
- canonicalization mismatch
- raw vs host-visible result confused
- stale host profile
- wall-clock metadata included in deterministic identity
- skipped assertion reported PASS
- expected failure silently ignored
- evidence produced by untrusted build
- signature valid but semantic evidence false
- secret hashed but still recoverable
- redaction changes digest without metadata

## 12. Lifecycle/governance

- accepted spec not implemented
- implementation diverges from spec
- README stale
- roadmap presented as feature
- provider docs stale
- SDK upgraded without conformance rerun
- conformance suite upgraded without baseline review
- deprecation without migration
- migration without recovery plan
- fidelity claim too broad
- compatibility claim transitive across products
- AI-generated candidate self-approved
- requirement ID reused
- evidence artifact deleted
- version field omitted
- unsupported future extension silently accepted

## 13. Forward-agent cases

- subagents share unknown memory
- long-running task resumes after host restart
- task cancellation races completion
- tool surface changes mid-agent run
- host performs programmatic tool calls not directly chosen by model
- one agent holds stale world assumptions while another commits
- host batches calls
- host hides raw result
- agent generates new tool/spec
- multimodal tool output too large
- GUI/computer-use semantics mixed with MCP tool semantics

## 14. Exhaustiveness mechanism

Because new failure modes will appear:

1. every incident or newly discovered edge case adds a catalog entry;
2. each SPEC maps relevant entries to requirements/tests;
3. each release reviews unresolved P0/P1 entries;
4. unknown behavior remains an explicit result;
5. coverage metrics never claim logical exhaustiveness.

The project's rigor comes from the **mechanism for absorbing unknowns**, not from pretending this list is final.
