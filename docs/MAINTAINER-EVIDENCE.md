# Maintainer Evidence Ledger

This file is a template for the maintainer's Codex for Open Source
application. It deliberately contains no invented stars, downloads, users, or
provider compatibility claims. Replace every placeholder only with a
reproducible GitHub or package-registry value before submitting an application.

The program's current application form and criteria are documented by OpenAI at
<https://openai.com/form/codex-for-oss/>. This repository does not claim
eligibility; the ledger is only a reproducible way to collect evidence.

## Current project identity

- Repository: https://github.com/augety121/MCP-State-Twin
- Maintainer role: <primary maintainer or core maintainer>
- Evidence snapshot date (UTC): <YYYY-MM-DD>
- Current release/commit: <tag or full commit>
- License: MIT
- Scope: hermetic, deterministic, tools-first MCP state simulation for agent
  evaluation and regression testing

## Usage and ecosystem evidence

| Signal | Value | How to reproduce |
|---|---|---|
| Public stars | <value> | GitHub repository page on snapshot date |
| Forks | <value> | GitHub repository page |
| Releases | <value> | GitHub Releases page; link the latest release |
| Contributors | <value> | GitHub contributors page |
| Dependents / downstreams | <value or none found> | GitHub dependency graph or named downstream links |
| Package downloads | <value or N/A> | Package registry page; do not estimate |
| CI runs | <value> | GitHub Actions run history |
| Open issues / PRs triaged | <value> | Link issue/PR search queries or monthly log |

If a metric is unavailable, write “N/A — not published by the registry” rather
than inferring a number. A small project must not claim broad adoption before
there is evidence.

## Active maintenance evidence

Use the repository's [Release Management](RELEASE-MANAGEMENT.md) policy and
the tag-driven [release workflow](../.github/workflows/release.yml) as the
process baseline. The workflow is process evidence only; each published tag
still needs a public CI run and release URL.

Record links to real activity:

- PR review: <PR URL(s)>
- issue triage: <issue URL(s)>
- release management: <release/tag URL(s)>
- security response: <security advisory or policy evidence>
- regression/evaluation work: [scheduled-action implementation CI run
  33456196081](https://github.com/augety121/MCP-State-Twin/actions/runs/33456196081)
  for commit `e3f0355`; add benchmark/report links only when they exist
- contributor support: <discussion/issue URL(s)>

The repository's contribution contract is CONTRIBUTING.md and its safety
boundary is SECURITY.md. These documents define process; they are not proof
that the process has already happened.

## 2026-09-02 maintenance incident record

- [PR #5 CI failure](https://github.com/augety121/MCP-State-Twin/actions/runs/33498653329):
  parser fuzz exited with `context deadline exceeded`; no saved crashing input
  appears in the log. Other jobs passed. The timeout's underlying cause is
  not proven by this log.
- [Dependabot failure](https://github.com/augety121/MCP-State-Twin/actions/runs/33498581757):
  CEL `go_module_path_mismatch`; upstream v0.32.0 requires `cel.dev/cel-go`.
- Corrective contracts: SPEC-0035/0036, with the independently reproduced CEL
  null conversion defect tracked by SPEC-0037. Exact fresh CI evidence belongs
  in `IMPLEMENTATION-STATUS.md`; old run results are not overwritten.
- Integration route: direct main update under the maintainer's explicit
  request to keep this work on main; no unrelated dependency PR is merged.
- Outcome: implementation `61aac2d` passed the complete
  [CI run 33579877698](https://github.com/augety121/MCP-State-Twin/actions/runs/33579877698),
  including Linux race, two fuzz targets, Windows/macOS, secret scanning,
  hermetic networking and MCP conformance. A manual fresh
  [Dependabot check](https://github.com/augety121/MCP-State-Twin/actions/runs/33579893053)
  succeeded on the new module manifest. These links are observed outcomes,
  not a claim that old failure records changed or that the original fuzz
  timeout's underlying cause was proven.
- PR follow-up: [refresh request on #5](https://github.com/augety121/MCP-State-Twin/pull/5#issuecomment-5503016536)
  asks Dependabot to rebase and validate, not merge the dependency update.

## Codex/API use plan

The official Codex for Open Source form asks how API credits would be used. A
truthful project-specific answer should describe work observable in this
repository, for example:

> API credits would support maintainer workflows for this MCP State Twin:
> reviewing pull requests against executable determinism and hermeticity
> invariants, triaging reproducible failure reports, generating and checking
> synthetic evaluation scenarios, and preparing release evidence. Model output
> would remain advisory; maintainers and CI would make final merge and release
> decisions.

Do not claim Codex or an OpenAI API has already been used unless a linked
repository artifact or CI record proves it.

## Claim audit for automated repository analyses

Automated application helpers may infer security surfaces from repository
labels instead of executable code. Review those statements before submission.
For the current accepted v0.1 profile:

- **Do not claim that the runtime makes external network requests.** Hermetic
  mode has no upstream connector or passthrough path; CI dependency downloads
  are a build-system concern, not a runtime feature.
- **Do not claim a third-party native extension surface.** TwinSpec expressions
  are bounded and declarative, and arbitrary scripts/native adapters are out of
  scope.
- **Do not describe model-controlled tool input as prompt execution.** Inputs
  remain untrusted data validated against schemas and transition rules. Tool
  descriptions may influence an external model's choices, but the runtime does
  not execute them as instructions.
- Actual current review surfaces include bounded YAML/JSON/CEL admission,
  unauthenticated loopback data-plane exposure, authenticated control-plane
  routing, SQLite file/migration handling, HTTP resource limits, dependency
  supply chain, and GitHub Actions/release permissions.

If remote hosting, provider harnesses, recorder support, or native adapters are
added later, re-run this audit instead of inheriting today's statement.

## Application readiness gate

Before applying, confirm:

- the repository and maintainer profile are public;
- at least one released, reproducible artifact exists;
- CI is green on the default branch;
- recent maintenance activity is visible in PRs, issues, and releases;
- README points to implementation status and limitations;
- no credentials, production traces, or personal data are in fixtures; and
- every usage/adoption number in the form has a source URL and snapshot date.

The program page says it considers meaningful usage, broad adoption or clear
ecosystem importance, and active maintenance such as PR review, issue triage,
and release management. Those are eligibility signals, not a guarantee of
selection.
