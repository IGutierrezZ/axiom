# Judgment Day Prompts and Formats

## Judge Prompt

```markdown
You are blind Judge {A|B} in explicit Judgment Day mode.

Target: {immutable target identity and exact paths}
Skills to load: {resolved SKILL.md paths}
Criteria: correctness, edge cases, error handling, performance, security, and project conventions.

Run one exhaustive read-only sweep. Do not edit, delegate, refute, or inspect unrelated scope. If scoped re-judging, read ONLY the frozen ledger and immutable fix delta; record any fix-caused defect with proof.

Return one JSON object and no prose, using exactly this native result shape:

{"findings":[{"location":"path:line or path:start-end","severity":"CRITICAL","claim":"observable incorrect behavior","evidence_class":"deterministic","causal_disposition":"introduced","proof_refs":["concrete proof"]}],"evidence":["what was inspected"]}

This is a judgment-day judge result, not a `axiom review capture-result` lens artifact. Judgment day selects no lenses and records your work as a judge proof, so your result carries no bound artifact subject and no inspection envelope. The only allowed top-level fields are `findings` and `evidence`, and the only allowed finding fields are `location`, `severity`, `claim`, `evidence_class`, `causal_disposition`, and `proof_refs`. Never emit `summary`, `skill_resolution`, or any other unknown field. Keep orchestration metadata outside the native result JSON; `evidence` contains only genuine inspection evidence. Return `{"findings":[],"evidence":["what was inspected"]}` when clean, then terminate.
```

> This shape governs judgment-day judges only. An ordinary bounded review lens is a different artifact: it is captured with `axiom review capture-result`, and its result must additionally echo the binding's top-level `subject_hash` and a completed `inspection` envelope, or admission refuses it. `axiom review schema reviewer` emits that schema with a working example. The two shapes are not interchangeable, and neither one is a relaxation of the other.

## Fix Actor Prompt

```markdown
You are the bounded Judgment Day fix actor.

Confirmed severe ledger IDs: {table}
Skills to load: {resolved SKILL.md paths}

Apply only confirmed fixes as atomic work units. For each unit, record focused test result, runtime evidence or justified N/A, and rollback boundary. Never review, add findings, refactor unrelated code, or launch another actor. Mark addressed IDs fixed and return control to the parent orchestrator for scoped re-judgment.

End with: Skill Resolution: {paths-injected|fallback-registry|fallback-path|none} — {details}
```

## Verdict Shape

```yaml
target_identity: <sha256>
round: 1 | 2
confirmed: []
suspect: []
contradictions: []
info: []
fix_work_units: []
scoped_rejudgment: approved | escalated | not_run
terminal_state: approved | escalated
skill_resolution: <value>
```

Warnings and suggestions are informational. After the second scoped re-judgment, remaining severe findings require `escalated`; no third round exists.

## SDD Role Target

Applies when the target is the diff of one role of an SDD change. Everything above still holds: two blind judges in parallel, agreement as corroboration, only confirmed severe findings are fixed, at most two rounds, and the terminal verdicts `APPROVED | ESCALATED`. The judgment is informational and never blocks a phase, a gate, or archive.

- **Activation.** The orchestrator offers it or applies the session preference `verify-judgment`; accepting the offer or holding that preference is the explicit request. Launch it only when the current report of the role (its `verify_file`) ends with `**Estado:** limpio`. An explicit request from the user prevails over a verify that is `no limpio`: launch it and say that the verify was not clean. Before launching, print one cost line (two blind judges and up to two rounds with the fix actor); it never stops `continuous`.
- **Immutable target.** The identity is `{role, repositories, merge-base, HEAD SHA, sha256}`. Resolve each repository only from `axiom branch status --cwd "<workspaceRoot>" --change <change> --json` (the `worktree` and `comparedTo` of the role branch, or of the change branch for `fullstack`), always with `git -C "<worktree>"`. The sha256 covers `git -C "<worktree>" diff <merge-base>` (committed, staged, and unstaged changes against the working tree) plus the list and content of the untracked files from `git -C "<worktree>" ls-files --others --exclude-standard`, excluding the change folder. Judges read the working tree read-only while no writer acts. Before merging results, the parent recomputes the sha256; if it changed, discard the round and report it, with no automatic retry.
- **Fixes.** `jd-fix-agent` edits only inside `actionContext.allowedEditRoots` intersected with the `worktree` of the role, and never under `technicalContext.reference`. A confirmed finding whose fix needs a path outside those roots stays pending for a human decision and prevents `APPROVED`.
- **First-round confirmation.** In `checkpointed`, ask before the first fix round edits anything. In `continuous`, the `verify-judgment` preference already includes that authorization.
- **Registry.** The parent merges and persists the ledger in `review-ledger.md` (role `fullstack`) or `review-ledger.<role>.md`; in Engram, `sdd/{change}/review-ledger` or `sdd/{change}/review-ledger/{role}`.
- **Fix record.** Applied fixes go as work units in the subsection `### Correcciones del juicio — <role>` of the single `apply-progress` of the change, which the parent merges without overwriting earlier content. Never write them in `tasks*.md`: that would alter what the `role-apply` gate digests.
- **Final verification.** When fixes were applied, the parent launches `sdd-verify` for the role again. Its report becomes the current report of the role and keeps the earlier one as historical context. Without fixes the current report does not change and the verdict is reported separately.
