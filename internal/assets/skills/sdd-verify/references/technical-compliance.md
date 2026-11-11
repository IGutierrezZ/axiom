# Per-role verification and technical compliance

Load this reference only when the orchestrator supplies a role. It is informational: nothing here blocks a phase, a gate, or archive, and the verify never edits code, tasks, or planning artifacts. Its only write is its own report.

## 1. Resolve the role's diff (read-only)

The repository and the base come only from native status, never from a path you guess.

1. Run `axiom branch status --cwd "<workspaceRoot>" --change <change> --json`.
2. For each repository, take the branch of the role (`kind: role`, that role) or, for the single role `fullstack`, the branch of the change (`kind: change`). Take `worktree` (absolute) and `comparedTo` from that JSON.
3. Compute the base: `git -C "<worktree>" merge-base <comparedTo> HEAD`.
4. Read the work: `git -C "<worktree>" diff <base>` (committed, staged, and unstaged changes of tracked files against the working tree) plus every untracked file listed by `git -C "<worktree>" ls-files --others --exclude-standard`, read in full.
5. Exclude the change folder (`<planningHome>/changes/<change>/`): it is planning, not the role's work. Never inspect files changed only by another role.
6. When no branch resolves (no git snapshot), fall back to the files that `apply-progress` declares for the role and state the limitation in the report. That report is `no limpio`.
7. An empty diff produces no compliance judgment: write `**Estado:** no limpio — diff del rol vacío` and say that no judgment is offered.

Always pass `git -C "<worktree>"`. Never use the current directory, a relative path, or a path not returned by native status.

## 2. Compliance sources

Contrast the diff with the rules of these sources, and list each one consulted in the section:

- (a) skills that the role's apply declares loaded in `skill_resolution`;
- (b) registry skills whose triggers match the files the role touched, even when apply did not load them;
- (c) the `entry` file of each `technicalContext` mode, when native status projects it;
- (d) the rules and context of `openspec/config.yaml` that apply to the `apply` phase and to verification.

A missing source (for example, no `technicalContext`) is declared `no aplicable`. It never produces a finding or an error. Never create, modify, or delete files under `technicalContext.reference`; report a finding there and leave the file alone.

## 3. Finding shape

One line per finding: `severity · source (skill <name> | technicalContext | openspec/config.yaml) · rule · path:line[-end] · evidence`.

- Severity is `CRITICAL`, `WARNING`, or `SUGGESTION`.
- A finding without a citable rule is not a violation. Report it, at most, as an `Observación` with no associated rule.
- When there are no findings, the section still exists, says so, and lists the sources consulted.
- A declared check that could not run (tool missing, permission denied) is listed as not executed with its reason. Never present it as passed.

## 4. State line

End the summary with exactly one line:

- `**Estado:** limpio` when there is no `CRITICAL` finding (functional or technical) and every mandatory check ran and passed. Mandatory checks are the commands of `rules.verify` in `openspec/config.yaml` and, when present, `testing.runner.command` and `testing.quality.type_checker.command`.
- `**Estado:** no limpio — <reason>` otherwise, including a check that was unavailable or not executed.

`WARNING` and `SUGGESTION` never make a report `no limpio`. The orchestrator decides the next review by reading this line from the current report, never by inferring it from the phase summary.

## 5. Where the report goes

- File: the `verify_file` that the kickoff assigns to the role (`verify-report.md` for `fullstack`, `verify-report.<role>.md` for any other role).
- Engram: `sdd/{change}/verify-report` for `fullstack`, `sdd/{change}/verify-report/{role}` for any other role.
- With the single role `fullstack`, the role's verify and the change's verify are the same report; do not run it twice without a reason.
