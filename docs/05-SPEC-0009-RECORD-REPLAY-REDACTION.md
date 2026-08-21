# SPEC-0009 — Recorder, Redaction & Cassette Replay

> **Status:** Proposal  
> **Implementation baseline:** Not verified/treated as not implemented.

## Goal

Add L0 exact-path replay without creating a hidden production proxy.

## Allowed topology

```text
Authorized Harness
  -> Recorder Adapter
  -> Explicit test/sandbox upstream
```

The evaluated agent MUST NOT transparently fall through to production when the Twin lacks a behavior.

## Cassette envelope

- format version
- source profile
- capture timestamp
- sequence
- protocol/tool-surface identity
- request
- response/error
- redaction manifest
- provenance
- optional legal/data classification

## Secret requirements

### ST-REPLAY-R001
Never persist Authorization, Cookie, API keys, bearer tokens, refresh tokens or provider secrets by default.

### ST-REPLAY-R002
Unknown headers default DROP.

### ST-REPLAY-R003
Redaction happens before durable storage.

### ST-REPLAY-R004
If redaction changes behaviorally relevant match data, cassette cannot claim byte-exact semantics.

### ST-REPLAY-R005
Default recording profile is synthetic/test data.

## Matching

Modes MAY include:
- strict sequence;
- tool + canonical arguments.

A miss returns typed `CASSETTE_MISS`.

No semantic guessing.

## L0 claim

L0 means the captured path can be replayed under the declared matching rules.

It does not prove unrecorded behavior, current upstream equivalence or L2.

## Feasibility

**High technically.** Privacy and secret handling are the release blockers.
