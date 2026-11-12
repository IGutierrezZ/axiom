# SDD Status and Instructions Contract

Shared OpenSpec-style contract for SDD commands and phase skills. Use it before acting on a change so orchestration does not guess state, paths, or edit scope.

## Purpose

Commands that select, continue, apply, verify, or archive an SDD change MUST first produce or consume structured status. The status is the handoff between orchestrator and phase executor.

## Change Selection

- If a change name is provided, use that exact change after confirming it exists in the selected artifact store.
- If no change name is provided, infer only when the active change is unambiguous from session state or there is exactly one active change.
- If multiple active changes match or the active change is unclear, ask the user to choose. Do not guess.
- If no active changes exist, report that no SDD change is active and suggest `/sdd-new <change>`.

## Native Engine

Native `gentle-ai.sdd-status/v2` is the sole status contract. A request for v1 or another prior contract fails read-only with one instruction: start a fresh implementation state and rerun `axiom sdd status --contract gentle-ai.sdd-status/v2`. Research is optional and does not add a proposal-admission gate. Ask about real unresolved product decisions; pause only dependent work.

- When the `axiom` binary is available, prefer `axiom sdd status [change] --cwd <repo> --json --instructions` for read-only status and `axiom sdd continue [change] --cwd <repo>` only for explicit authorized continuation. This holds for every artifact store: the dispatcher resolves the declared store itself.
- The native dispatcher resolves the artifact store the workspace DECLARES in `openspec/config.yaml` and reports it in `artifactStore`. A declared store is authoritative in both directions: it selects the resolver, and an empty declared store reports as empty rather than silently serving the other store's artifacts. Never re-resolve artifact status yourself, and never branch on the store: read the locators the dispatcher returned in `artifactPaths`.
- For every store, treat native status JSON as authoritative over prompt inference or manually reconstructed state.
- When `blockedReasons` is non-empty, do not proceed to terminal, archive, or apply work. Return or report `blockedReasons` and stop. Explicit read-only verification may still diagnose accessible work without granting edit authority. When `nextRecommended` is `resolve-blockers`, always report `blockedReasons` and stop. When `nextRecommended` is a planning token (`propose`, `spec`, `design`, or `tasks`), launch the corresponding planning phase — missing planning artifacts are the expected output of those phases, not genuine blockers.
- `notes` is a separate, always-present array of informational diagnostics. A non-empty `notes` NEVER blocks anything: never withhold apply, sync, archive, or a terminal route because a note is present, and never read a note as a blocker. Report notes next to the route you report, so a human sees a future authority need before reaching it. `blockedReasons` stays reserved for genuine blockers, so the gate above reads `blockedReasons` alone.
- `nextRecommended` is a bounded machine token for routing, not human prose. Route only by `nextRecommended` and dependency states. A genuine blocker's human-readable explanation belongs in `blockedReasons`; a non-blocking explanation belongs in `notes`.
- If the binary is unavailable or invalid, report that native status is unresolved. Do not fabricate native-shaped status, recompute readiness, or invoke continue as a fallback.

## Status Schema

Render the native projection as markdown or JSON. These fields document the exact frozen external `StatusV2Projection`, not permission to reconstruct native readiness:

```yaml
schemaName: gentle-ai.sdd-status
schemaVersion: 2
changeName: <change-name-or-null>
artifactStore: openspec | engram | hybrid | none
planningHome:
  mode: repo-local
  path: <absolute path to openspec>
changeRoot: <absolute path to openspec/changes/<change> or null>
artifactPaths:
  proposal: [<absolute path>]
  specs: [<absolute paths>]
  design: [<absolute path>]
  tasks: [<absolute path>]
  applyProgress: [<absolute path>]
  verifyReport: [<absolute path>]
contextFiles:
  proposal: [<absolute readable files>]
  specs: [<absolute readable files>]
  design: [<absolute readable files>]
  tasks: [<absolute readable files>]
  applyProgress: [<absolute readable files>]
  verifyReport: [<absolute readable files>]
artifacts:
  proposal: missing | done | partial
  specs: missing | done | partial
  design: missing | done | partial
  tasks: missing | done | partial
  applyProgress: missing | done | partial
  verifyReport: missing | done | partial
taskProgress:
  total: 0
  completed: 0
  pending: 0
  allComplete: false
dependencies:
  proposal: blocked | ready | all_done
  specs: blocked | ready | all_done
  design: blocked | ready | all_done
  tasks: blocked | ready | all_done
  apply: blocked | ready | all_done
  verify: blocked | ready | all_done
  archive: blocked | ready | all_done
applyState: blocked | all_done | ready
actionContext:
  mode: repo-local
  workspaceRoot: <absolute path>
  allowedEditRoots: [<absolute paths>]
relationships:
  dependsOn: []
  supersedes: []
  amends: []
  conflictsWith: []
  sameDomainActiveChanges: []
consent: <optional exact axiom.sdd-integration.consent/v1 envelope>
baseline: # optional; present only with a valid `baseline:` declaration in axiom.yaml
  documents:
    - domain: <domain>
      path: <absolute path>
      integration: compose | directed | register
technicalContext: # optional; present only with a valid `technical_context:` declaration in axiom.yaml
  maintained: # optional mode
    paths: [<absolute paths>]
    entry: <absolute path, optional>
  reference: # optional mode, same shape
    paths: [<absolute paths>]
workflow: # always present, with or without a selected change
  lane: odd | sdd | ask
  source: declared | default | no_workspace | invalid
iso: # optional; omitted when axiom.yaml has no `sdd:` block and the selected change is not ISO
  project: iso | not_iso | invalid
  reason: <only with project invalid>
  change: iso | not_iso # absent without a selected change
  matrix: # only in an ISO change whose apply is finished
    state: valid | invalid | missing | not_required
    successful: <boolean>
    findings: <count>
phaseInstructions:
  apply: [<instruction strings>]
  verify: [<instruction strings>]
  archive: [<instruction strings>]
nextRecommended: propose | spec | design | tasks | apply | verify | archive | sdd-new | select-change | resolve-blockers
blockedReasons: []
notes: []
```

SDD status never reads review mode, offers RDD, or exposes review state. After completed implementation, proceed directly toward archive; verification is optional (except in an ISO change, see `ISO Evidence`) and standalone non-SDD review remains separate.

`phaseInstructions` is optional and appears only when instructions are requested. It carries execution-phase keys (`apply`, `verify`, `archive`); planning-phase instructions (`propose`, `spec`, `design`, `tasks`) are surfaced in dispatcher markdown. `consent` requires an OpenSpec-backed native status reporting `blocked(edit_authority_missing)` with a valid persisted marker; never reconstruct it. Empty path fields MUST be arrays, not null. `blockedReasons` and `notes` are always arrays as well, both `[]` rather than null when empty. `changeName` and `changeRoot` are nullable; all other non-optional sections are required in native output.

## Project Baseline and Technical Context

`baseline` and `technicalContext` are additive, optional fields of the status. They project the structural declaration of `axiom.yaml` (`baseline:` and `technical_context:`) so phase skills never interpret that file themselves.

- Read the locators from the status: `baseline.documents[]` (each with `domain`, an absolute `path`, and `integration`: `compose` for a document with a requirement profile, `directed` or `register` for a prose document) and `technicalContext.maintained` / `technicalContext.reference` (each with absolute `paths` and, when declared, an absolute `entry`). Never read `axiom.yaml` to find them, and never assume a default location for them.
- Both fields are omitted entirely (never `null`, never empty) when `axiom.yaml` declares no valid block. A mode that is not declared does not appear. With no declaration the baseline is the OpenSpec default under `planningHome.path`; an absent `baseline` accompanied by a declaration cause in `notes` is an invalid declaration, not the default.
- An invalid declaration is never projected, not even partially. The status stays operational: planning and `apply` are never blocked by it.
- Only archive is gated by a declaration cause: `dependencies.archive` is `blocked` when `axiom.yaml` is unreadable or a declared block is invalid, when `technicalContext.maintained` is declared and the change has no `context-impact.md`, or when a project baseline or maintained context is declared with an artifact store that has no files (`engram` or `none`). The cause always appears in `notes`; it appears in `blockedReasons` (and `nextRecommended` becomes `resolve-blockers`) only when the route would otherwise be `archive`.
- The context delta (`context-impact.md`) is not projected: `artifacts`, `artifactPaths`, and `contextFiles` are unchanged by it.

## Project Workflow Lane

`workflow` is an additive field of the status, always present (with or without a selected change). It projects the effective project lane declared in the `workflow:` block of `axiom.yaml`, so orchestrators never read that file themselves.

- `lane` is `odd`, `sdd` or `ask`. `source` says where it comes from: `declared` (the block sets `lane`), `default` (a valid `axiom.yaml` without the block or without `workflow.lane`; the lane is `sdd`), `no_workspace` (there is no `axiom.yaml`) or `invalid` (`axiom.yaml` or its `workflow:` block cannot be read or is invalid, even when the block itself is valid but the rest of the file is not).
- `no_workspace` and `invalid` always project `ask`: the rule fails closed toward asking the lane question. If the status does not answer, the orchestrator acts as `ask`.
- The field is informational: it adds no note and no blocking reason, and the recommended route does not depend on it.

## ISO Evidence

`iso` is an additive, optional field of the status. It projects the ISO declaration of the project (`sdd: {iso: true}` in `axiom.yaml`) and of the selected change, so orchestrators and phase skills never read `axiom.yaml` or `kickoff.yaml` themselves. The formats the rules refer to are in `sdd-iso-convention.md`.

- `project` is `iso` (the project declares `iso: true`), `not_iso` (`iso: false`, or an `sdd:` block without the key) or `invalid` (the block is invalid, contradicts `workflow.lane: odd`, or `axiom.yaml` cannot be read). `invalid` fails closed: it is never read as `not_iso`, and `reason` carries the error and its correction, with or without a selected change.
- `change` is `iso` or `not_iso`, read from the snapshot that `axiom sdd kickoff seal` copied into `kickoff.yaml`. Only a seal made in an ISO project carries that snapshot, so a change without it (a project that is not ISO, a seal made before the declaration existed, or a retro-seal with `--infer`) is not ISO, even if the project later declares ISO, and no ISO rule applies to it. When the snapshot differs from the current declaration, a note says so and the rules of the change follow its snapshot.
- `matrix` appears only in an ISO change whose apply is finished: `state` is `valid`, `invalid`, `missing`, or `not_required` (a bug without impact on the spec and without matrix), `successful` is true when the matrix is valid (or not required) and no row fails, and `findings` counts the findings; their detail is in `blockedReasons`. The orchestrator reads it to decide whether a verify is clean, without reading the report.
- An ISO project only uses SDD: every project that projects `project: iso` sends all work, including small, understood work, through the SDD lane.

### Typed reasons

An ISO refusal is a typed reason in `blockedReasons` (or `notes`), in the form `blocked(<token>): <detail>; <continuation>`. The detail names the domain, section, requirement, row or file; the continuation says what to do. One finding produces one reason, and `axiom sdd continue` reflects them unchanged. Follow the continuation: never reinterpret a token. The closed list:

| Token | Applies to | Stage | Continuation |
|---|---|---|---|
| `bug_report_incomplete` | every project | specs | complete `Observado:` or `Esperado:`, or add `## Informe de bug` to `proposal.md` |
| `bug_no_impact_incomplete` | every project | specs | add the line `Sin impacto en la spec` or a non-empty `Motivo:` to `## Impacto en la spec` |
| `iso_declaration_invalid` | selected change with an invalid or contradictory `sdd:` | archive | `axiom workspace validate`; for the contradiction, retire `iso` or the `odd` lane |
| `iso_requirement_id` | ISO | specs | add or fix `REQ-<n>.<m>` in the requirement name; a RENAMED entry when the requirement exists without one |
| `iso_requirement_scenarios` | ISO | specs | add a `#### Scenario:` to the requirement, or rename a repeated scenario |
| `iso_coverage` | ISO | tasks | add `Cubre: REQ-x` to some task |
| `iso_cubre_unknown` | ISO | tasks | fix the cited identifier or remove it |
| `iso_bug_requirement` | ISO | tasks | cite the existing requirement with `Cubre:`, or handle it as a normal change with a delta |
| `iso_bug_contradiction` | ISO | tasks | remove the `Cubre:` line, or remove the no-impact mark and cite an existing requirement |
| `iso_matrix_missing` | ISO | archive | run `sdd-verify` to write `## Matriz de verificación` in `verify-report.md` |
| `iso_matrix_invalid` | ISO | archive | run `sdd-verify` to fix the row the reason names |
| `iso_matrix_failed` | ISO | archive | fix the failing behaviour, then run `sdd-verify` again |
| `iso_store_unsupported` | ISO change with `engram` or `none` store | archive | declare `artifact_store: openspec` or `hybrid` in `openspec/config.yaml` |

- The reasons of the `specs` and `tasks` stages are blocking reasons: the route stays on the phase that must fix them. The reasons of the `archive` stage always appear in `notes`, and in `blockedReasons` only when, without them, the route would be `archive`; a change with pending tasks keeps routing to `apply`.
- Outside an ISO change none of the `iso_*` tokens appears. `bug_report_incomplete` and `bug_no_impact_incomplete` also apply outside ISO, because the bug profile is available to every project; `iso_declaration_invalid` applies to any selected change when the project declaration is invalid.
- `blocked(wrong_branch)` is still projected first.

### Verify route in an ISO change

In an ISO change `verify` is mandatory before archive. When apply is finished and the matrix is missing or invalid (`iso_matrix_missing`, `iso_matrix_invalid`), `nextRecommended` is `verify` with the reason in `blockedReasons`, and a `continuous` orchestrator chains it like any other recommended phase. With a valid matrix that has failing rows (`iso_matrix_failed`) the route is `resolve-blockers`. The `verify` route replaces `await-gate` only when the open gate is `integration`; the gates `spec`, `design`, `tasks` and `role-apply:<role>` keep their precedence. Every other change keeps verification optional.

## Governance (sealed kickoff)

`governance` is an additive, optional field. It is omitted entirely (never `null`) for a change with no sealed `kickoff.yaml`; every sealed change carries it:

```yaml
governance:
  kickoff: { schema, flowMode, executionStyle: continuous | checkpointed, docsPublication?, sealedAt, sealedBy } # handoffPolicy? appears only when an older kickoff.yaml carries it; it is deprecated and changes nothing; docsPublication? is `auto` only when the person authorized it at sealing
  roster: { source: kickoff, roles: [<role>, ...] } # in the order the kickoff names them
  gates: [{ key, status, blocks, reason?, reopened?, handedOff?, handedOffTo? }]
```

- `roster` is present for every sealed change, in both execution styles, so a multi-role change (for example `backend` and `frontend`) is visible even when it runs `continuous`. Task progress per role is not a separate field: `taskProgress` is the aggregate over every role's `tasks.<role>.md`, and `artifactPaths.tasks` lists those files in roster order.
- `kickoff.docsPublication` is `auto` only in a `continuous` change whose kickoff was sealed with `--docs-publication auto`: the orchestrator then runs `axiom sdd publish` without asking when `tasks` closes and after archive. It is absent otherwise, and a kickoff sealed before this field never publishes automatically.
- `gates` is populated only when `kickoff.executionStyle` is `checkpointed`. Under `continuous` it is always `[]`: no gate is evaluated or presented, and the orchestrator chains phases without stopping.
- `handedOff` and `handedOffTo` appear on a gate approved with exit 2 of its stop (approve, publish and stop): `handedOffTo` names the phase handed to another person (`design`, `tasks`, `apply`, `apply:<role>,<role>` for the roles still pending, or `verify` after the last role). Both are absent on exits 1 and 3. They are information only: the route (`nextRecommended`) is the one an ordinary approval yields, and nothing is blocked by them.
- `gateQuestion` (the typed gate-decision envelope) appears only while `nextRecommended` is `await-gate`, which only a `checkpointed` change with an open gate can reach. Never expect it under `continuous`. For `spec`, `design`, `tasks` and `role-apply:<role>` it offers three choices in the order `rejected`, `hand_off`, `approved`, every one resolved with `axiom sdd gate record`; with the `none` artifact store it offers two (`rejected`, `approved`) plus `handOffUnavailable`, which names the store that makes exit 2 unavailable. The `integration` gate keeps its two choices. On a planning gate whose store keeps files, the question also carries `publication: { remote, branch, prTarget?, invocation }` (the remote is always `origin`, `branch` is the branch that is pushed, `prTarget` is informative and absent in the in-place integration of a `multirepo` specs repository, and `invocation` is an `axiom sdd publish` bound to that branch with `--branch`), or `publicationUnavailable: "<code>"` (for example `wrong_branch`) when the target cannot be resolved; the choices are then still the `gate record` ones. Approving with exit 2 or 3 is the authorization of that push of documents and of nothing else. `axiom sdd gate record --json` returns the same `publish` (or `publishUnavailable`) on every approval except `integration`: it only calculates the invocation, and the orchestrator runs it once afterwards.
- Gate-only preconditions never apply to `continuous`: archive is not blocked by a missing `integration` gate.
- In a `checkpointed` multi-role change, `verify` is blocked until every active role has its `role-apply` gate approved, counting only the roles whose gate policy is `blocking` (the reason names the roles still pending). A `deferred` or `optional` role that is still pending never blocks `verify`: a note names it with its policy and state. The last-active-role notice of `axiom sdd gate record` is emitted when the approved gate closes the last `blocking` role. `handoff.md` plays no part in it: a `handoff.md` that exists is ignored.

## Apply State

- `blocked`: Required apply artifacts are missing, task selection is ambiguous, or action context makes edits unsafe.
- `all_done`: Tasks artifact exists and every implementation task is checked `[x]`.
- `ready`: Tasks artifact exists, at least one implementation task remains unchecked, and edit scope is safe.

## Dependency States

- `proposal`, `specs`, `design`, and `tasks` report whether prerequisite artifacts are blocked, ready, or all done.
- `apply` is `ready` only when specs, design, and tasks are available and task progress is not all done.
- `verify` is an optional diagnostic and may inspect partial implementation. Its availability never certifies a report or task completion; missing artifacts limit conclusions. In an ISO change it is mandatory before archive (see `ISO Evidence`).
- Completed implementation normally recommends `archive`; pending work normally recommends `apply`. Explicit archive can record incomplete work, but actual edit permissions and mechanical archive/spec-composition safety checks remain mandatory.
- Report presence is a locator/status fact, not a verdict. Missing, stale, malformed, or failed reports do not block archive or create remediation state, except the verification matrix of an ISO change, which is validated and must be successful. Preserve historical bytes and report actual unfinished tasks and findings without inventing PASS.
- No SDD phase offers or launches RDD, ordinary 4R, or Judgment Day. Delivery follows ordinary repository policy; optional diagnostics create no review or delivery authority.

## Action Context Guard

The orchestrator MUST carry `actionContext` into any phase launch.

- If manually reconstructed context cannot prove edit ownership or allowed edit roots, stop before editing.
- If `allowedEditRoots` is present, only edit files within those roots.
- If a command cannot prove a file is inside the authoritative workspace or allowed edit roots, stop and ask for clarification.

## Branch Guard (`blocked(wrong_branch)`)

`blocked(wrong_branch)` is the second typed reason of `blockedReasons`, next to `blocked(edit_authority_missing)`. It appears when the repository the change writes to is not on the branch the change expects.

- The reason names the repository, the current branch and the expected branch, and carries an executable continuation: `axiom branch ensure --cwd <absolute path> --change <change>` (plus `--lane` when it applies). Run it, then repeat the status. It has no consent envelope: the continuation is a command the orchestrator runs, not a grant the human answers.
- In a repository with a change branch (`monorepo-embedded`) it applies in every phase. In the archive phase the expected branch is the archive branch `<branch_prefix>/<change>-archive` and the continuation adds `--archive`: standing on the change branch, on the integration branch, or on a role branch blocks. In a `multirepo`, a code repository is checked only when apply evaluates a work unit that touches it, and the specs repository expects the integration branch, also in the archive phase.
- When `wrong_branch` and `edit_authority_missing` both apply, `wrong_branch` is projected first, because the authorized edit roots depend on the right branch; after the branch is corrected, `blocked(edit_authority_missing)` may appear.
- A change whose `kickoff.yaml` has no branch-policy snapshot (sealed before the branch policy, or with `--infer`) is exempt. Without git, or outside a git repository, the guard does not block and adds a note.
- `axiom sdd continue` reports the same block and authorizes no next phase.

## Edit Authority Consent

Status inspection requires no launch, review, delivery, or archive preflight and never runs a phase.
Without a persisted marker, missing-root status omits consent and displays explicit preparation guidance.
Before continue, the host must check current human scope includes the change-local marker; a planning-only
marker grant does not grant source roots. Read-only/excluded-marker scope suppresses the mutating call.
Continue preserves valid marker bytes and reads back the no-replace winner; malformed, inaccessible,
failed publication/readback, or detected directory replacement emits no usable consent. No universal
atomic visibility is claimed for publisher copy fallbacks. Grant and grant replay consume the CURRENT
persisted marker and refuse absent/stale identity without initializing a marker or appending authority.


A change whose tasks.md work units target paths outside `allowedEditRoots` never reports apply ready. Native status reports `applyState: blocked` and `blockedReasons` carries a `blocked(edit_authority_missing)` reason naming each unauthorized edit root and the three exits: edit tasks.md so every work unit stays inside the authorized edit roots, grant this change edit authority for the named edit roots, or mark a read-only input with `(read-only)` on its line.

- Detection is conservative prose inspection: backticked path-like tokens inside markdown checkbox lines that resolve to a path outside the authorized roots. A different repository is named by its Git root; a same-repository target is narrowed to its containing edit root; a directory in no Git repository is named as itself. A backticked path immediately followed by `(read-only)` (case-insensitive) is a read-only input and not an edit target; the marker annotates only the path it follows, so an unmarked path on the same line still counts.
- Read-only status never prepares consent; after explicit authorized `sdd-continue` preparation, an OpenSpec-backed status reporting `blocked(edit_authority_missing)` carries the typed `axiom.sdd-integration.consent/v1` envelope as the optional `consent` block: headline, reason, `value`, the missing roots as evidence, exactly two choices with answer tokens `granted` and `declined` (each with label, effect, and an exact invocation), and an off-path note.
- A later work unit's own unauthorized root does NOT block the current work unit. It is reported as a `note(future_edit_roots)` entry in `notes` naming that future root, so the current unit's `applyState` stays `ready` and `blockedReasons` stays empty until that work unit is reached.
- Answer flow: the orchestrator relays the COMPLETE envelope losslessly as a blocking prompt. Only on the human's explicit `granted` answer does the agent execute the envelope's named grant invocation, verbatim and exactly once, then re-enter through native status. The agent NEVER runs the grant unprompted and NEVER answers on the human's behalf.
- Decline stays blocked: the agent runs the envelope's decline invocation, nothing is persisted, the change stays `blocked(edit_authority_missing)`, and the reason names all three exits.

## Status Output

Every command that acts on a change MUST show status before launching an executor or performing archive work:

- Active change selection and schemaName.
- Artifact statuses and paths/topics used as context.
- Task progress and unchecked task list when tasks exist.
- Next recommended action.
- `blockedReasons` whenever it is non-empty, including any edit-root blockers.
- `notes` whenever it is non-empty, labeled as informational and explicitly not a blocker.

For an already archived change, terminal `all_done` dependencies mean no phase remains; they do not certify implementation or verification. Read the archive report and preserved task/report artifacts for actual outcomes.
