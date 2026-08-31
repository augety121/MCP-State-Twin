# Phase 6: Scenario Families and Multi-Agent Evaluation

- **Target:** post-v0.4 research/preview releases
- **Entry:** deterministic scheduler and evidence identity are stable

## Candidate scope

- repository-maintenance, package-release and issue-lifecycle families;
- synthetic email/calendar domains after separate modeling review;
- deterministic scenario generation and held-out fixtures;
- metamorphic relations and counterfactual forks;
- per-Agent isolated branches by default;
- explicitly declared shared-world contention experiments;
- per-Agent attribution and terminal state scoring;
- curriculum packaging for training/evaluation reuse.

## Integrity requirements

Agents MUST NOT see hidden state, expected answers, fault controls or branch
control operations. Shared-world runs require an explicit scheduler profile;
host OS scheduling cannot determine benchmark meaning. Natural-language judges
remain secondary to state assertions.

## Non-claims and exit evidence

This phase does not establish universal A2A support, a general AGI benchmark or
a complete model of reality. Each scenario family needs schema, invariant,
mutation, isolation, leakage, resource and reproducibility evidence before it
is admitted.
