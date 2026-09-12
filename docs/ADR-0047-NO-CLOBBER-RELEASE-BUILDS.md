# ADR-0047: Refuse existing release output

Status: Accepted, 2026-09-12.

Remove unconditional `rm -rf dist` from the packaging script. Require reviewed
release policy, clean source, repository-root execution and a tag that peels to
HEAD. Claim exactly one new `dist` directory with restrictive umask. Any existing
file, directory or symlink at that path is an error, not a cleanup target.

Keep the five existing cross-compile targets and existing release checksum
mechanism; do not run real release packaging/checksum generation as part of this
maintenance turn. Build serially with a default one-slot Go scheduler. Failures
preserve the new partial directory and propagate; no recursive cleanup, retry or
overwrite fallback. See SPEC-0047.
