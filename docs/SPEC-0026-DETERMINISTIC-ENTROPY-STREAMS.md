# SPEC-0026: Deterministic Modeled-World Entropy Streams

- **Status:** Accepted via ADR-0026
- **Implementation status:** implemented bounded profile
- **Verification status:** positive, fork/locality, persistence, validation and rollback tests
- **Profile:** `sha256-ctr-v1`

## 1. Purpose and boundary

This contract provides reproducible pseudo-random-looking bytes to a modeled
TwinSpec transition. It makes a simulation input explicit; it does not make an
LLM, Agent scheduler, host process or network deterministic.

The output MUST NOT be used as a password, bearer token, signing key, nonce for
real cryptography or source of production security. Seeds committed to a
TwinSpec are public synthetic fixture data.

## 2. TwinSpec contract

An opted-in TwinSpec declares:

```yaml
entropy:
  algorithm: sha256-ctr-v1
  seed: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

The seed is exactly `sha256:` followed by 64 lowercase hexadecimal digits. The
only accepted algorithm identifier is `sha256-ctr-v1`. A missing `entropy`
block disables the effect; an entropy effect in that TwinSpec MUST fail
admission.

One tool effect is:

```yaml
- op: entropy
  stream: request_id
  bytes: 16
  as: request_id
```

`stream` follows the TwinSpec lowercase identifier grammar, `bytes` is 1..32,
and `as` follows the same identifier grammar. Fields belonging to other effect
operations are rejected rather than ignored. The result is lowercase
hexadecimal and therefore has exactly `bytes * 2` characters.

## 3. Normative derivation

For counter `C`, the emitted digest is:

```text
SHA-256(
  UTF8("statetwin.dev/entropy/sha256-ctr-v1") || 0x00 ||
  seed_bytes || 0x00 ||
  UTF8(stream) || 0x00 ||
  uint64_be(C)
)
```

The runtime emits the first `bytes` digest bytes and hex-encodes them. A new
stream starts at counter `0`. After a successful draw its persisted counter is
`C + 1`. Counter encoding, domain separator, NUL delimiters and big-endian byte
order are semantic compatibility requirements.

## 4. Transaction and branch semantics

Entropy counters live under canonical branch state. Therefore:

1. all effects, postconditions, global invariants and output-schema validation
   remain in the existing atomic tool transaction;
2. any failed outcome discards the increment;
3. snapshot captures every stream counter;
4. fork copies captured counters and isolates later increments;
5. reset restores captured counters;
6. state digest changes when a committed stream counter changes.

Entropy state is hidden from the CEL `state` view. A transition can use only
the value explicitly bound through its own `as` variable. This avoids turning
internal counters into an accidental Agent-facing data surface.

## 5. Limits and identity

`local-preview-v5` binds:

| Limit | Value |
|---|---:|
| streams per branch | 64 |
| bytes per draw | 32 |
| draws per effect | 1 |
| effects per call | inherited 128 total effects |

The TwinSpec digest binds algorithm and seed. Scenario EnvironmentIdentity
additionally records a canonical entropy-profile digest or `none`. Changing the
profile or these limits changes deterministic environment identity.

## 6. Failure semantics

- unknown algorithm, malformed seed, missing profile, invalid stream or byte
  count: TwinSpec admission failure;
- new stream above the bound: `INTERNAL_TWIN_ERROR` in the current engine
  contract and no committed transition;
- any later failed effect/invariant/output validation: entire transition,
  including the counter, rolls back;
- unsupported behavior never falls back to host randomness.

## 7. Executable acceptance evidence

Tests MUST establish:

- equal first draw for independent branches with equal state;
- a fixed algorithm golden vector (`token`, counter 0, 16 bytes);
- different consecutive draws on one stream;
- persisted counter advancement;
- rollback when a later effect fails;
- strict algorithm, seed, stream and size admission;
- resource-profile identity includes entropy limits.

## 8. Explicit non-claims

This profile does not implement cryptographic randomness, model seed control,
probabilistic fault selection, cross-TwinSpec stream compatibility, secret
generation, distributed counter allocation or entropy larger than 32 bytes.
