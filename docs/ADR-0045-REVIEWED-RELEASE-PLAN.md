# ADR-0045: Version-bound release plan admission

Status: Accepted, 2026-09-12. B11/B32 maintenance subset; amends release procedures,
not RFC-0002 support scope or ADR-0021's independent release trains.

The existing tag glob and permissive packaging regex admit malformed versions.
Generated notes alone do not require a reviewed scope, migration statement or
stable-gate declaration. Add a bounded, checked-in JSON plan and reviewed Markdown
notes at `releases/<tag>.json` / `releases/<tag>.md` before building any release.

Accept only the documented v0.1 local-core release profile. Semantic tag syntax,
profile/tag/channel agreement and explicit review declarations are independent
checks. Stable needs an additional explicit stable-gates declaration; it does not
inherit approval merely from a green unit test. New release trains require a new
accepted profile, not a parser fallback.

Declarations in repository files are not signatures or proof of human review.
The workflow still needs exact-candidate CI and manual publication. No approved
real release plan, tag or Release is created by this change. See SPEC-0045.
