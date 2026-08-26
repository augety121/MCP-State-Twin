# Support

MCP State Twin is pre-release open-source software. Community support is
provided through GitHub issues on a best-effort basis; there is no commercial
SLA.

Use a bug report for reproducible runtime failures and include the commit/tag,
operating system, Go version, sanitized TwinSpec/fixture, command and observed
error class. Do not attach credentials, provider transcripts, production traces
or personal data.

Use GitHub private security advisories for suspected boundary escapes, secret
exposure, cross-branch access or control-plane disclosure. See `SECURITY.md`.

Current supported scope is the local, synthetic, single-process profile in
RFC-0002 and accepted ADRs. Public remote deployment, multi-tenancy and live
provider compatibility are not supported unless a current evidence report says
otherwise.
