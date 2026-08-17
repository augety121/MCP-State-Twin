# Security Policy

## Project status

MCP State Twin is pre-release software. It is intended for synthetic,
isolated test worlds. It has not received an external security audit and must
not be treated as a production security boundary.

The current development build has important limitations:

- the MCP data plane has no built-in authentication;
- TLS termination is not implemented;
- recorder, secret scanning, egress enforcement, and multi-tenant isolation
  are not implemented;
- the control plane uses one bearer token and is intended to remain on
  loopback;
- native TwinSpec extensions are deliberately unsupported.

Do not expose either endpoint to an untrusted network. Use synthetic fixtures,
bind to loopback, and enforce network egress denial outside the process.

## Reporting a vulnerability

Please do not open a public issue for a vulnerability that could expose
credentials, escape the simulated environment, cross branch boundaries, or
make hidden control operations agent-visible.

Use GitHub's private security-advisory workflow for this repository. Include:

- affected commit or version;
- minimal reproduction;
- expected and observed trust boundary;
- whether real external systems or secrets were reached;
- suggested mitigation, if known.

Until a public security contact is published, avoid sending real secrets or
production traces with a report.

## Security invariants

The design-level security requirements are maintained in
[RFC-0001](docs/RFC-0001.md),
[ADR-0002](docs/ADR-0002-CONTROL-PLANE-ISOLATION.md), and the
[Failure Mode Matrix](docs/FAILURE-MODE-MATRIX.md). A documented invariant is
not proof that the development build already satisfies it; consult
[Implementation Status](docs/IMPLEMENTATION-STATUS.md).
