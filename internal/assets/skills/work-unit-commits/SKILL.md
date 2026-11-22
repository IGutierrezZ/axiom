---
name: work-unit-commits
description: "Plan commits as reviewable work units. Trigger: implementation, commit splitting, chained PRs, or keeping tests and docs with code."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## When to Use

Load this skill when deciding what belongs in each commit or PR.

Use it for:

- Splitting a feature into reviewable work.
- Preparing commits before opening a PR.
- Turning a change into chained or stacked PRs when the person asks for them.
- Keeping reviewer cognitive load healthy.
- Applying SDD tasks as one work-unit commit each inside a single PR.

## Critical Rules

| Rule | Requirement |
|------|-------------|
| Commit by work unit | A commit represents a deliverable behavior, fix, migration, or docs unit. |
| Do not commit by file type | Avoid `models`, then `services`, then `tests` if none works alone. |
| Keep tests with code | Tests belong in the same commit as the behavior they verify. |
| Keep docs with the user-visible change | Docs belong with the feature or workflow they explain. |
| Tell a story | A reviewer should understand why each commit exists from its diff and message. |
| One PR by default | The commits of a change ship together in a single PR; chaining happens only when the person asks for it. |
| Chain-ready | Each commit should be a candidate chained PR if the person later asks to chain. |
| No code-golf | Never shrink a diff by deleting comments, blank lines, docs, or tests, or by compressing code. Commit by work unit; there is no size target to fit. |

## Work Unit Checklist

Before committing, confirm:

- [ ] The commit has one clear purpose.
- [ ] The repo still makes sense after applying only this commit.
- [ ] Tests or docs for this unit are included when relevant.
- [ ] Rollback is reasonable without reverting unrelated work.
- [ ] Focused test command and exact result are recorded.
- [ ] Runtime harness command/scenario and exact result are recorded, or explicit `N/A` explains why no runtime boundary exists.
- [ ] Rollback boundary names the exact files/behavior removable without unrelated work.
- [ ] The commit message explains the outcome, not the file list.

## Split Examples

| Weak split | Better work-unit split |
|------------|------------------------|
| `add models` | `feat(auth): add token validation domain model and tests` |
| `add services` | `feat(auth): wire token validation into login flow` |
| `add tests` | Tests included with each behavior commit |
| `update docs` | Docs included with the user-facing change they explain |

## PR Relationship

Use work-unit commits as the foundation of the pull request:

1. Build the smallest independent work unit.
2. Include verification for that unit.
3. Commit it with a Conventional Commit message.
4. Keep all the commits in one PR. Only when the person asks to chain, stack, or slice, promote commits or groups of commits into chained PRs with the `chained-pr` skill.

## SDD Relationship

When `sdd-tasks` produces a Delivery Forecast:

- With `delivery_strategy: single-pr` (the default), commit by work unit inside one PR.
- With `delivery_strategy: chained`, follow the `chain_strategy` and the PR slice each work unit names; the `chained-pr` skill plans the chain.
- The forecast never stops implementation and never asks for a decision.
- Splitting is bounded: after one honest slicing pass, keep the best cohesive work-unit split. Do not iterate shrinking the code.

Each SDD work unit should map cleanly to a commit or PR with:

- clear start state,
- clear finished state,
- verification in the same unit,
- rollback that does not remove unrelated work.

Its implementation evidence MUST include:

- Focused test command and exact result.
- Runtime harness command/scenario and exact result, or explicit `N/A` with reason.
- Rollback boundary stated independently of commit creation; uncommitted work units still require it.
- When fixing a bounded review ledger, group atomic work units inside the single correction transaction; work-unit count never creates another fix budget.

## Review and Fixups

Each work-unit commit is announced and can be reviewed while the work continues:

- After each work-unit commit, tell the person in one line the short sha, the task ID and `git show <sha>`.
- Asking for the pull request validates every commit of the change: there is no commit-by-commit approval and nothing blocks the pull request.
- Record only the commits the person asked to change, one line each, never as a checkbox, in the `## Commits con cambios pedidos` section of the tasks file of the change.
- Finish the task in progress first, or park it, before a correction on an earlier commit.
- A commit is published when a remote-tracking branch contains it (`git branch -r --contains <sha>` prints anything); never fetch to decide.
- If it is not published, use `git commit --fixup=<sha>` and defer the autosquash until the pull request is requested: commit any staged task first, then run `git -c sequence.editor=true rebase -i --autosquash --autostash <base>`, whose `--autostash` sets other uncommitted edits aside and restores them. On a conflict, abort the rebase, leave the fixups as they are and report it.
- If it is published, add a corrective commit that cites the corrected sha. Never rewrite published history and never force a push.
- Recommend merging with a merge commit, without squash, so the reviewed commits reach the target branch. Axiom never merges.

## Commands

```bash
# Review the story before committing
git diff --stat
git diff --cached --stat

# Check recent commit style
git log --oneline -5
```
