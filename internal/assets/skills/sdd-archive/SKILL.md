---
name: sdd-archive
description: "Archive an SDD change honestly by syncing delta specs and preserving its artifacts. Trigger: orchestrator launches archive."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: gentleman-programming
  version: "2.0"
  delegate_only: true
---

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-archive` sub-agent unless you loaded this skill directly through the `skill()` tool.

- If you are the `sdd-archive` sub-agent, continue with the phase work below. Do not delegate. Do not call the Skill tool.
- If you loaded this skill through the `skill()` tool, you are the orchestrator. Stop here and delegate to the dedicated `sdd-archive` sub-agent using your platform's delegation primitive (for example, `task(...)` or a sub-agent invocation).


## Language Domain Contract

Generated technical artifacts default to English. Do not inherit the user's conversational language or the active persona's regional voice for SDD artifacts unless the user explicitly requests that artifact language or the project convention requires it.

If technical artifacts are explicitly requested in another language, use a neutral/professional register unless the user explicitly requests a different tone or regional variant.

Public/contextual comments follow the target context language by default. Explicit user language or tone overrides win; otherwise use a neutral/professional register unless the target context clearly calls for another tone or regional variant.

## Purpose

You are a sub-agent responsible for ARCHIVING. You merge delta specs into the main specs (source of truth), then move the change folder to the archive. You complete the SDD cycle.

## What You Receive

From the orchestrator:
- Change name
- Artifact store mode (`engram | openspec | hybrid | none`)
- Structured status from `skills/_shared/sdd-status-contract.md`, including artifact paths, task progress, dependency states, and actionContext
- Explicit final-state facts for work completed after intermediate artifacts were persisted (verify warnings fixed in later commits, blockers resolved, updated test counts), when the orchestrator has them

## Final-State Authority

The archive report is the terminal record of the cycle. It describes the state of the change AT CLOSE, not the state at earlier points during the cycle. A future reader consults the archive to learn what actually shipped; a stale claim sends them to redo finished work — or to trust that something is pending when it already closed.

`apply-progress` and `verify-report` are intermediate snapshots. Each describes the state of the work at the time it was written, and work routinely continues after they are persisted: verify warnings get fixed in later commits, blocked tasks get completed, test counts change. A snapshot's "done" stays true — work does not un-complete — but its "pending", "blocked", or "open gap" claims are only valid for the moment the snapshot was written. Never present an intermediate snapshot's statement as the current state of the change.

When sources disagree about a fact, rank them — most authoritative first:

1. **The persisted tasks artifact** — observed completion visibility; unchecked tasks remain unchecked.
2. **Explicit final-state facts in the orchestrator's launch prompt** — e.g. "these verify warnings were fixed in later commits", "this blocker was resolved and verification passed". The launch prompt is the most recent account of the change and outranks intermediate snapshots.
3. **`verify-report` and `apply-progress`** — intermediate snapshots. Lowest rank: valid history of what was true at their time, never evidence of final state.

Reporting rules that follow:

- When a higher-ranked source says done/fixed/resolved and a lower-ranked snapshot says pending/blocked/open, report the final state and cite where the fix landed (commit, later evidence). Do NOT echo the stale claim.
- When a contradiction cannot be ranked — e.g. the launch prompt asserts a fact that no higher-ranked source or repository evidence corroborates — record the contradiction in the archive report explicitly: both statements, their sources, and when each was written. Never resolve it silently in either direction.
- Attribute snapshot-derived claims to their source and time ("per `verify-report` {observation-id}, at verification time ..."). Do not restate them in bare present tense as current facts.
- Carry final numbers (test counts, warnings, open issues) from the highest-ranked source that covers them; do not copy numbers from `verify-report` or `apply-progress` when later work changed them.
- Never merge distinct defects or failures into a single causal story. A cause is recorded as confirmed only with evidence; otherwise record the failure as undiagnosed.

This hierarchy governs reporting, not admission. Missing, stale, malformed, or failed optional reports and unfinished tasks do not block archive. Record unresolved findings and incomplete work without claiming they passed or were completed.

## Execution and Persistence Contract

> Follow **Section B** (retrieval) and **Section C** (persistence) from `skills/_shared/sdd-phase-common.md`.

- **engram**: Read `sdd/{change-name}/proposal`, `sdd/{change-name}/spec`, `sdd/{change-name}/design`, `sdd/{change-name}/tasks`, and `sdd/{change-name}/verify-report` when available; verification is optional. Record all observation IDs actually read in the archive report for traceability. Save as `sdd/{change-name}/archive-report`.
- **openspec**: Read and follow `skills/_shared/openspec-convention.md`. Perform merge and archive folder moves.
- **hybrid**: Follow BOTH conventions — persist archive report to Engram (with observation IDs) AND perform filesystem merge + archive folder moves.
- **none**: Return closure summary only. Do not perform archive file operations.

### Archive Readiness

Before any spec sync or archive move, require structured status. Use refreshed native SDD status and preserve actual edit permissions. Completed implementation normally recommends archive; an explicit archive request may close unfinished work without a verification certificate. Review mode is not archive state.

For a change with a sealed `kickoff.yaml`, archive additionally requires the "integration" block-review gate approved: evidence that the change actually merged or deployed toward a preproduction or production environment, never a substitute for or a shortcut past a favorable local `verify-report.md` (RF-ARC-01). Native status reports this as `dependencies.archive`; when blocked, it names the missing gate and the exact command that satisfies it. Record that evidence with `axiom sdd gate record --cwd <repo> --change <name> --gate integration --decision approved --evidence-kind pr_merged|deployment|attestation [--commit <sha>] [--base-ref <ref>] [--evidence "<text>"]` before retrying archive: `pr_merged` is verified against the local commit graph, while `deployment` and `attestation` are honestly recorded as declared, not checked. Archiving with this precondition unmet is refused explicitly, and the change stays out of `openspec/changes/archive/` (RF-ARC-02 introduces no new archive mechanism — only this precondition ahead of the existing one).

**Archive check (mandatory, before composing or moving).** Run `axiom sdd archive-check --cwd <workspaceRoot> --change <name>` before `axiom archive context prepare`, before `axiom sdd archive-integrate` and before any move. On a non-zero exit, STOP and return `blocked`, quoting its typed reasons: integrate nothing and move nothing. It exits `0` for a change that is not ISO and for an ISO change without findings, and it writes nothing. With `--json`, `domains[]` reports `alreadyComposed` per domain, `assignments` the provisional identifiers already assigned by an interrupted archive and `pendingProvisionals` those still to assign; when an earlier archive stopped after integrating, run `axiom sdd archive-integrate` again and then finish the move, because the verb skips what is already integrated. The rules are in `../_shared/sdd-iso-convention.md`.

Archive records actual progress and findings; ordinary repository policy decides delivery.

### Honest Partial Archive

Read the available tasks and reports before closure. Record completed and unfinished tasks, missing artifacts, unresolved findings, and unrun checks. Do not repair checkboxes or rewrite historical reports during archive. Archive may close incomplete work; it must never describe that work as fully implemented or verified. No additional waiver or verification verdict is required.

### Action Context Guard

- If structured status reports `actionContext.mode: workspace-planning`, STOP. Do not move workspace changes into repo-local archives or edit linked repos.
- If `allowedEditRoots` is present, archive operations must stay inside those roots.

### Archive Branch Flow

In a `monorepo-embedded` workspace the archive is a documentation-only change: it runs on its own archive branch and travels in its own pull request. It never runs on the change branch, on a role branch, or on the integration branch (`git.integration_branch`); status reports `blocked(wrong_branch)` there, with the exact continuation.

1. Once the pull request of the change is merged, run `axiom branch ensure --cwd "<workspaceRoot>" --change <c> --archive --json` and use the absolute path of its JSON as `--cwd`, as the repository of every `git -C` command, and as the root of every command below (`cd "<path>" && ...`, section G). If it refuses with `change_not_integrated`, the local integration reference does not hold the merged change yet: suggest `--fetch` only with the explicit authorization of the user, and never pass it on your own.
2. On the archive branch, record the evidence of the merged pull request with `axiom sdd gate record --cwd <path> --change <c> --gate integration --decision approved --evidence-kind pr_merged` (add `--commit <sha>` with the merge commit when you have it). That evidence is the precondition of RF-ARC-01.
3. Run the archive check, then, when the status projects `technicalContext.maintained`, run `axiom archive context prepare`; integrate the delta specs (Step 2, with `axiom sdd archive-integrate`), make the context edits the plan names, and run `axiom archive context verify`; then do the move to `openspec/changes/archive/` of Step 3, in that order. After every rebase of the archive branch, repeat the integration with `--check` as Step 2 describes.
4. Commit the result on the archive branch. The person publishes it and opens a separate documentation pull request toward the integration branch (`git.integration_branch`); you never push and never open it.
5. After that pull request is merged, the person removes the archive worktree and branch with `axiom branch done --cwd "<workspaceRoot>" --change <c>`; in promotion mode `branch`, `done` keeps the change branch (`merge_not_confirmed`) until it is merged into `git.main_branch` and removes only the archive branch.
6. If the documentation pull request is rejected, discard the archive branch (with the user's confirmation, `axiom branch done --cwd "<workspaceRoot>" --change <c> --force`; the result reports the tip so the branch can be recovered): the change stays pending archive, and `axiom branch status` lists it again with the command that restarts this flow.

In a `multirepo` there is no archive branch: the archive runs in place in the specs repository, on its integration branch, and `ensure --archive` refuses with `archive_branch_not_applicable`. The precondition is the evidence of RF-ARC-01: `axiom sdd gate record ... --gate integration` with `--evidence-kind pr_merged` (verified against the specs repository), `deployment` or `attestation`.

## Mechanical Copy Contract (MANDATORY)

Archival is a mechanical filesystem operation. File content MUST NEVER pass through the model's Read/Write path to be copied — a model that summarizes, truncates, or alters even one byte while reporting success corrupts the audit trail silently. The only acceptable copy mechanism is a native shell command (`cp -R`, `mv`, or `git mv`), verified by a structural readback. The living specs under `openspec/specs/` are the one exception: only `axiom sdd archive-integrate` writes them (Step 2), atomically and with its own `written[]` list as evidence, never the model.

- Copy artifacts with the shell only: `cp -R`, `mv`, or `git mv`. NEVER use Read → Write to reproduce artifact content into the archive or main specs — that routes bytes through model generation, where truncation is silent and undetectable without an independent diff.
- After every copy or move, run `diff -r` (source vs. destination) as a MANDATORY readback. The archive-report file is additive-only and excluded from the source/destination comparison (it did not exist in the source change folder).
- The verbatim `diff -r` output MUST appear in the phase result. An empty `diff -r` (no differences) is the only passing evidence; any difference is a truncation or alteration and FAILS the phase. A skipped or missing `diff -r` also FAILS the phase — agent self-report is never sufficient.
- If your platform's tool allowlist does not grant shell access, STOP and report `blocked` with the reason `shell access required for mechanical archive copy is unavailable` — do NOT fall back to Read/Write copying.
- This contract governs copies and moves. The context edits of maintained documents (see `references/technical-context-integration.md`) use bounded `Edit` operations on the sections the plan names, never a whole-file rewrite, and `axiom archive context verify` is their evidence instead of `diff -r`.

## What to Do

### Step 1: Load Skills
Follow **Section A** from `skills/_shared/sdd-phase-common.md`.

### Step 2: Sync Delta Specs to Main Specs

Inspect available delta specs; preserve edit permissions and the mechanical safety checks below.

#### Technical Context Branch (decide first)

Read `technicalContext` from the status (see `../_shared/project-baseline-convention.md`). Never read `axiom.yaml` and never deduce the artifact store yourself.

**IF the status projects `technicalContext.maintained`:** follow `references/technical-context-integration.md` for the integration of the maintained context, for the moment Step 3 runs, and for the integration trace in the archive report. This branch takes precedence over the `engram` and `none` lines below, which would otherwise skip the integration silently. The default blocks below still compose the delta specs: `openspec/specs/` is the only living specification.

The maintained context is integrated only through the native `axiom archive context` verbs. Execute each command literally, from the workspace root, and never rebuild its paths or flags by hand. Its general form is:

```bash
axiom archive context prepare --path "<workspaceRoot>" --change "{change-name}" --json
```

STOP and report `blocked` when the status marks the archive as blocked (`dependencies.archive`, `blockedReasons`, `notes`) or when `axiom archive context prepare` rejects. That includes a maintained context declared with an artifact store that keeps no files (`engram` or `none`, case (f)) and a retired block reported as `baseline_retired`, whose continuation is `axiom archive adopt scan`. Quote the cause literally and name the continuation the message gives. Do not deduce or change the artifact store, do not fall back to a manual edit, and do not move the change to the archive.

**IF `technicalContext.maintained` is not projected:** the default flow below applies unchanged, followed by the master catalog refresh that precedes Step 3.

**IF mode is `engram`:** Skip filesystem sync — artifacts live in Engram only. The archive report (Step 5) records all observation IDs for traceability.

**IF mode is `none`:** Skip — no artifacts to sync.

**IF mode is `openspec` or `hybrid`:** integrate every delta spec of `openspec/changes/{change-name}/specs/` into `openspec/specs/` with the one native invocation below.

#### Integrate the Change Specs (one native invocation)

**Mandatory Native Integration (#4119):** never Read a main spec and apply
ADDED/MODIFIED/REMOVED/RENAMED sections yourself, never copy a delta by hand and never
compose capability by capability. A model-driven Read/Edit merge is exactly how archive
previously dropped unrelated requirements or left a delta unapplied while still reporting
success. The native verb integrates the whole change at once: it validates everything before
writing, assigns the definitive number of every provisional identifier, composes each
capability with the native engine (by identifier when the delta name carries a valid one for
the project, `sdd.id_pattern` of `axiom.yaml`, and by exact title otherwise; every unrelated
requirement stays byte-for-byte), copies a capability that has no main spec yet, rewrites the
provisional identifiers in the deltas, the `Cubre:` lines of the tasks files and the
`Requisito` column of the verification matrix, writes the trace `ids.yaml`, and writes all of
it as one atomic set:

```bash
axiom sdd archive-integrate --cwd "<workspaceRoot>" --change "{change-name}" --json
```

- Run it once, from the archive branch, after `axiom sdd archive-check` and, with a
  maintained technical context, between `axiom archive context prepare` and the context
  edits. It replaces the former per-domain `archive-compose` loop and the full-spec copy
  block: do not run either.
- A nonzero exit means the verb refused: it wrote nothing and its JSON `error` names the
  `code`, the capability, section, requirement or token involved, and the `continuation`.
  Treat this as a blocking failure: STOP the phase and report `blocked` with that exact
  message. Do NOT retry with a manual Read/Edit merge or with `archive-compose`, and do NOT
  move the change into the archive.
- Only a zero exit is integration evidence. Include the invocation and its JSON in the phase
  result. `capabilities[].outcome` says what happened per capability (`created`, `composed`,
  `resumed`, `already_integrated`); an earlier interrupted archive is resumed by running the
  same command again, never by hand.
- `assignments[]` lists each provisional with its definitive number and its state
  (`assigned`, `kept`, `reassigned`); record it in the archive report. `duplicates[]` with
  `blocking: false` are historical duplicates outside the change and are only reported.
- `--check` runs the whole validation and writes nothing, so it is safe at any time. Run
  `axiom sdd archive-integrate --cwd "<path>" --change "{change-name}" --check --json` after
  every rebase of the archive branch: on an archived change it answers `duplicate_assigned_id`
  when the base took a number that the trace assigned. In that case do not edit anything:
  with the user's confirmation discard the archive branch (`axiom branch done --force`, see
  Archive Branch Flow) and archive again from the integration branch, where the change folder
  still holds the provisionals.

##### Poda Progresiva y Deprecación (ODD-4.1)

Cuando un incremento reemplaza o vuelve obsoletos requisitos de la especificación viva canonical, se debe usar la bandera `--supersede` del verbo en lugar de purgarlos en silencio. Esto preserva el requisito marcándolo con `[SUPERSEDED / DEPRECADO]` y su `(Reason: ...)`, evitando la doble fuente de la verdad sin romper la auditoría histórica; vale para los REMOVED de todas las capacidades del cambio:

```bash
axiom sdd archive-integrate --cwd "<workspaceRoot>" --change "{change-name}" --supersede --json
```

Si además existen requisitos de especificaciones monolíticas previas que quedan superados fuera de la sección `REMOVED`, especifícalos con `--superseded-requirements "Nombre Req 1 (REQ-x.y), <capacidad>/Nombre Req 2 (REQ-x.z)"`. Cada nombre casa por su nombre completo, incluido el identificador entre paréntesis, y admite el prefijo `<capacidad>/`; sin prefijo debe casar en una sola capacidad (`superseded_requirement_ambiguous` pide el prefijo y `superseded_requirement_unknown` rechaza un nombre inexistente).

#### Refresh the Master Catalog

After every delta spec is in place (integrated above) and before Step 3, regenerate `openspec/INDEX.md` with the native producer (RF-ARC-02 (4)). Run it once for the whole change:

```bash
axiom archive sync --path "<workspaceRoot>"
```

- A nonzero exit is a blocking failure: STOP and report `blocked` with the exact message. Do not write `openspec/INDEX.md` by hand and do not move the change.
- Archive updates no roadmap: `axiom.yaml` and `openspec/config.yaml` cannot declare any, so there is nothing else to refresh.

### Step 2b: Tracker Follow-up Section (`continuous` with an active tracker, BEFORE the move)

Applies only when the sealed style is `continuous`, a work-item plugin with Tasks support is active, and the change folder holds `integrations.yaml`. The diary `.tracker-notes.yaml` lives in the change folder and is never written to an archived folder, so read it now, before Step 3 moves the folder. Compose the section offline (no tracker call), from `integrations.yaml` and `.tracker-notes.yaml`, and keep it for the archive report of Step 5 as `## Seguimiento en el tracker`:

- One entry per Task: role, ID, URL, estimate (`estimate_hours`, or «sin estimación» when absent), hours recorded (sum of the `minutes` of that role's `time_entries`), and each entry with its actor, date and `item_id`.
- A role with no `time_entries` is reported with 0 h, «sin imputación».
- Every note of `.tracker-notes.yaml` with no later `done` note for the same verb and role is a pending event: list it with its code and its `continuation` command.
- Without `integrations.yaml`, omit the section. Never retype it from memory and never invent a value.

### Step 3: Move to Archive

**IF mode is `engram`:** Skip — there are no `openspec/` directories to move. The archive report in Engram serves as the audit trail.

**IF mode is `none`:** Skip — no filesystem operations.

**IF mode is `openspec` or `hybrid`:** Move the entire change folder to archive with date prefix, using a mechanical shell move. NEVER Read each artifact and Write it into the archive — that routes file content through the model and can truncate or alter bytes silently:

```bash
# Run this block as one shell transaction so the EXIT trap remains active.
# The snapshot is recursive and must be created before either move attempt.
source="openspec/changes/{change-name}"
destination="openspec/changes/archive/YYYY-MM-DD-{change-name}"
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/sdd-archive.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT
cp -R "$source" "$snapshot_root/source"

# Mechanical move (MANDATORY): git mv when tracked, mv otherwise
mkdir -p openspec/changes/archive
if [ -e "$destination" ] || [ -L "$destination" ]; then
  printf 'archive destination collision: source %s and destination %s remain unchanged. Resolve the destination collision, then rerun this archive step.\n' "$source" "$destination" >&2
  exit 1
fi

if git mv "$source" "$destination"; then
  :
else
  git_mv_status=$?
  if [ -e "$source" ] || [ -L "$source" ]; then
    :
  else
    printf 'git mv failed with status %s and source %s is absent; refusing plain mv fallback.\n' "$git_mv_status" "$source" >&2
    exit "$git_mv_status"
  fi
  if diff -r "$snapshot_root/source" "$source"; then
    fallback_source_diff_status=0
  else
    fallback_source_diff_status=$?
  fi
  if [ "$fallback_source_diff_status" -ne 0 ]; then
    printf 'git mv failed with status %s and source %s changed; refusing plain mv fallback.\n' "$git_mv_status" "$source" >&2
    exit "$git_mv_status"
  fi
  if [ -e "$destination" ] || [ -L "$destination" ]; then
    printf 'archive destination collision: source %s and destination %s remain unchanged. Resolve the destination collision, then rerun this archive step.\n' "$source" "$destination" >&2
    exit 1
  fi
  if mv "$source" "$destination"; then
    :
  else
    move_status=$?
    exit "$move_status"
  fi
fi

# The source must be gone before comparing the archived tree with its snapshot.
if [ -e "$source" ] || [ -L "$source" ]; then
  printf 'archive move left the source directory in place\n' >&2
  exit 1
fi

# MANDATORY readback: only empty diff output passes.
if diff -r "$snapshot_root/source" "$destination"; then
  diff_status=0
else
  diff_status=$?
fi
if [ "$diff_status" -ne 0 ]; then
  exit "$diff_status"
fi
```

Use today's date in ISO format (e.g., `2026-02-16`).

The `snapshot_root` is removed safely by the EXIT trap after the readback, including when the move or comparison fails. Compare the archived folder against that pre-move recursive snapshot; do not substitute a model readback, staged tree, or post-move source. The `archive-report` you write in Step 5 is additive and excluded from the comparison because it did not exist in the source snapshot. Any non-empty `diff -r` output or non-zero status is truncation, alteration, or an operational failure and FAILS the phase; a missing `diff -r` also FAILS the phase.

The portable destination guard rejects a destination that already exists before either move attempt; it does not provide an atomic cross-process no-clobber guarantee. Do not add a suffix, overwrite, merge, delete, or otherwise choose a destination automatically.

### Historical Malformed Nesting Recovery (Manual Only)

This guidance is only for the historical malformed shape `archive/YYYY-MM-DD-{change-name}/{change-name}/`. Run this block manually only after inspecting the paths:

```bash
active_source="openspec/changes/{change-name}"
outer_destination="openspec/changes/archive/YYYY-MM-DD-{change-name}"
nested_source="$outer_destination/{change-name}"

if [ -e "$active_source" ] || [ -L "$active_source" ] ||
   [ ! -d "$outer_destination" ] || [ -L "$outer_destination" ] ||
   [ ! -d "$nested_source" ] || [ -L "$nested_source" ]; then
  printf 'historical archive recovery refused: active source must be absent, and outer destination and nested source must be real directories; all paths remain unchanged. Resolve the ambiguous shape manually.\n' >&2
  exit 1
fi

mv "$nested_source" "$active_source"
```

Never automatically delete, overwrite, or merge the outer archive directory. If the active source exists or is a symlink, the outer destination or nested source is not a real directory, or the shape is otherwise ambiguous, stop and resolve the paths manually. After the active source is restored and the collision is resolved, rerun this archive step.

### Step 4: Verify Archive

**IF mode is `openspec` or `hybrid`:** The Mechanical Copy Contract above is the verification: the verbatim `diff -r` output from Step 3 MUST appear in the phase result, and an empty diff is the only passing evidence; for Step 2 the evidence is the zero exit and the JSON of `axiom sdd archive-integrate`. In addition, confirm:
- [ ] Main specs updated correctly
- [ ] Change folder moved to archive
- [ ] Archive preserves all artifacts that existed; missing artifacts are reported
- [ ] Archived tasks retain their original bytes; completed and unfinished counts are reported honestly
- [ ] Active changes directory no longer has this change
- [ ] With a maintained context: `axiom archive context verify` passed before the move, and the integration trace was appended to the archive report
- [ ] Verbatim `diff -r` readback output is included in the result and is empty (no differences)

A failed or skipped `diff -r` FAILS the phase regardless of the checkboxes above — agent self-report is never sufficient evidence of byte-identity.

**IF mode is `engram`:** Confirm available artifact observation IDs are recorded in the archive report; preserve the tasks observation and report unfinished work and diagnostic findings honestly.

**IF mode is `none`:** Skip verification — no persisted artifacts.

### Step 4b: Push the Summary of the Archived Change (only with an active tracker)

When a work-item plugin with Tasks support is active, run `axiom integrations summary push --cwd <archive root> --change <c> --json` on the archived folder, after the move is verified and before the orchestrator runs `axiom sdd publish --archive` (which closes the work item). On the archived folder it replaces the provisional identifiers with the definitive ones. It answers `not_applicable` (exit code 0) when there is no active plugin. A failure never fails the archive and is not journaled (the diary is not written to an archived folder): it shows only in the output, so record its code and continuation in the archive report under `## Seguimiento en el tracker`.

### Step 5: Persist Archive Report

**This step is MANDATORY — do NOT skip it.**

Follow **Section C** from `skills/_shared/sdd-phase-common.md`.
- artifact: `archive-report`
- topic_key: `sdd/{change-name}/archive-report`
- type: `architecture`

En proyectos gobernados por Axiom, el informe `archive-report.md` (o en Engram) DEBE redactarse en español y contener obligatoriamente la sección de trazabilidad de poda histórica (ODD-4.2):

```markdown
# Reporte de Archivado Formal: {change-name}

## Resumen de Cierre
- Fecha de Archivado: {YYYY-MM-DD}
- Estado de Verificación: {Aprobado (PASS) con evidencias formales}
- Destino en Archivo: `openspec/changes/archive/{YYYY-MM-DD}-{change-name}/`

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| {dominio} | {Requisito previo} | [SUPERSEDED] | {Explicación del reemplazo o deprecación} | `{dominio}/spec.md#{requisito}` |
*(Si no se superó ningún requisito previo, indicar explícitamente: "Ninguno. Incremento puramente aditivo.")*

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| {capacidad} | {nombre del requisito} | {provisional} | {definitivo} | {assigned / kept / reassigned} |
*(Se copia de `assignments[]` del JSON de `axiom sdd archive-integrate`. Si el cambio no usó identificadores provisionales, indicar explícitamente: "Ninguno. Sin identificadores provisionales.")*

## Artefactos Consolidados
- `proposal.md`
- `spec.md`
- `design.md`
- `tasks.md`
- `verify-report.md`

## Seguimiento en el tracker
*(Solo con tracker activo. En `continuous`, la sección compuesta en el paso 2b; en `checkpointed`, el resultado de `summary push`. Si el cambio no usó tracker, omitir la sección.)*
```

When the maintained-context branch ran, append the integration trace mechanically with `axiom archive context show`, as `references/technical-context-integration.md` describes. Never retype the trace.

### Step 6: Return Summary

Return to the orchestrator:

```markdown
## Change Archived

**Change**: {change-name}
**Archived to**: `openspec/changes/archive/{YYYY-MM-DD}-{change-name}/` (openspec/hybrid) | Engram archive report (engram) | inline (none)

### Specs Synced
| Domain | Action | Details |
|--------|--------|---------|
| {domain} | Created/Updated | {N added, M modified, K removed requirements} |

### Archive Contents
- proposal.md: {observed present/missing; do not infer completion}
- specs/: {observed present/missing; do not infer completion}
- design.md: {observed present/missing; do not infer completion}
- tasks.md: {present/missing}, {completed}/{total} tasks complete; {pending} unfinished

### Source of Truth Updated
The following specs now reflect the new behavior:
- `openspec/specs/{domain}/spec.md`

### SDD Cycle Complete
The change is archived. Implementation: {actual state}. Verification: {not run / partial / actual results}.
Unfinished tasks and unresolved findings: {list or none observed}.
```

## Rules

- Archival is a MECHANICAL filesystem operation: copy/move artifacts with `cp -R`/`mv`/`git mv` via the shell only, NEVER via model Read/Write — a model can truncate or alter bytes silently while reporting success, and only an independent `diff -r` catches it
- After every archive copy or move, run `diff -r` (source vs. destination, archive-report additive-only) and include its verbatim output in the phase result; an empty diff is the only passing evidence, and a skipped/missing `diff -r` FAILS the phase
- If shell access is unavailable for mechanical copy, STOP and report `blocked` — do NOT fall back to Read/Write copying
- The archive report reflects FINAL state per the Final-State Authority hierarchy: never echo stale `verify-report`/`apply-progress` claims as current facts, and record unrankable contradictions explicitly instead of resolving them silently
- ALWAYS sync delta specs BEFORE moving to archive
- When merging into existing specs, PRESERVE requirements not mentioned in the delta
- Use ISO date format (YYYY-MM-DD) for archive folder prefix
- If the merge would be destructive (removing large sections), WARN the orchestrator and ask for confirmation
- The archive is an AUDIT TRAIL — never delete or modify archived changes
- If `openspec/changes/archive/` doesn't exist, create it
- Apply any `rules.archive` from `openspec/config.yaml`
- Never create, modify, or delete files under `technicalContext.reference` paths.
- Never edit the files of `technicalContext.maintained` outside the integration plan, and never assume "Sin impacto" when the change has no context delta.
- Return envelope per **Section D** from `skills/_shared/sdd-phase-common.md`.
