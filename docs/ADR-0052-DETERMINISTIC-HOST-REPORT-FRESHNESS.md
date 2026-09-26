# ADR-0052: Deterministic Host Report Freshness

- Status: Accepted; bounded read-only B15 time-policy subset
- Date: 2026-09-26
- Contract: [SPEC-0052](SPEC-0052-HOST-REPORT-FRESHNESS.md)

The existing report records expiry but admission does not assess a date. Keep
structural validation clock-independent and add an explicitly timed assessment.
API profiles use SPEC-0019's 30-day policy; ChatGPT / Claude Code product profiles
use its 14-day policy. This decision also assigns 30 days to generic/custom
reports as a conservative local declaration policy. It does not infer the
unknown custom host's product family.

The effective end is the earlier of declared expiry and this policy cap.
Expiry is exclusive; future observations are not yet valid. The assessor never
extends the source expiry, reads the wall clock, alters source artifacts or
creates a live compatibility claim. The CLI can optionally enforce time
eligibility, but provenance and current scope still require independent evidence.

Full identity matching, artifact-backed claim derivation, automatic matrix
generation, revocation and native/provider live verification remain separate
B15 work. A report with plausible declarations can be fabricated; time policy
cannot fix that. Existing compatibility rows remain unverified/experimental.
