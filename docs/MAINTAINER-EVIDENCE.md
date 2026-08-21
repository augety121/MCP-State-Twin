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
- regression/evaluation work: <CI run, benchmark, or report URL(s)>
- contributor support: <discussion/issue URL(s)>

The repository's contribution contract is CONTRIBUTING.md and its safety
boundary is SECURITY.md. These documents define process; they are not proof
that the process has already happened.

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
