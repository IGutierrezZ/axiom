---
name: chained-pr
description: "Trigger: the person asks to chain, stack, or slice pull requests, or delivery_strategy is chained. Plan chained PRs that keep each review focused."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Load this skill only when the person asks to chain, stack, or slice pull requests, or when `delivery_strategy` is `chained`. Without that request a change ships as one pull request (`single-pr`) and this skill does not apply.

## Hard Rules

- Chaining is opt-in: never propose or start it on your own because of the size of a change.
- Slicing constrains how work is **split**, never the code itself. Never delete comments, blank lines, docs, or tests, and never compress or restyle code to make a slice look smaller.
- Slicing is bounded: make **one** honest slicing pass and keep the best cohesive split; do not iterate on the split.
- Keep each PR reviewable in about **≤60 minutes**.
- Use one deliverable work unit per PR; keep tests/docs with the unit they verify.
- State start, end, prior dependencies, follow-up work, and out-of-scope items in every chained PR.
- Every child PR must include a dependency diagram marking the current PR with `📍`.
- In Feature Branch Chain, create a draft/no-merge tracker PR; child PR #1 targets the tracker branch, later children target the immediate parent branch.
- Treat polluted diffs as base bugs: retarget or rebase until only the current work unit appears.
- Do not mix chain strategies after the user chooses one.

## Decision Gates

| Condition | Action |
|---|---|
| The person did not ask to chain | Keep a single PR with one work-unit commit per task. |
| Chaining requested, each slice can land independently | Use Stacked PRs to main. |
| Chaining requested, feature must integrate before main | Use Feature Branch Chain with tracker. |
| Generated/vendor/migration diff cannot split cleanly | Keep that part in one slice and say why. |
| No cohesive split exists after one slicing pass | Stop; deliver the best split and report why it cannot be split further. |
| When `delivery_strategy` is `chained`, SDD provides it | Follow it before apply/PR creation. |

## Execution Steps

1. Identify independent work units.
2. Ask for a chain strategy when none was named or cached.
3. Create branches/PRs using the chosen strategy only.
4. Add Chain Context to each PR without replacing the repo PR template.
5. Verify each PR independently: CI/tests/docs/manual checks, rollback scope, and clean diff.
6. Keep tracker PR draft/no-merge until all child PRs are reviewed and integrated.

## Output Contract

Return the chosen strategy, PR order, current PR boundary, dependency diagram, verification plan, and the reason for any slice that could not be split further.

## References

- [references/chaining-details.md](references/chaining-details.md) — strategy diagrams, PR body section, branch commands, and reviewer guidance.
