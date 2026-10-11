# Project Baseline Integration (sdd-archive reference)

`sdd-archive` loads this reference when the status projects `baseline` or `technicalContext.maintained` (see `../../_shared/project-baseline-convention.md`). It replaces the default Step 2 blocks for the project's own documents. Step 3 (move) and Step 5 (report) of `SKILL.md` stay as written.

- `<workspaceRoot>` is `actionContext.workspaceRoot` from the status (`../../_shared/sdd-phase-common.md`, section G). Every native command below takes `--cwd "<workspaceRoot>"`.
- Run each native command once and include its exit code and output in the phase result. A nonzero exit always triggers the stop protocol; never retry it with a manual merge.
- The change name is `{change-name}`. Native commands read the plan, deltas and context delta from the change folder and write only `<changeRoot>/integration.yaml`.

## Which flow applies

| The status projects | Flow |
|---|---|
| `baseline` | Flow 1: project baseline |
| `technicalContext.maintained` and no `baseline` | Flow 2: default baseline with a maintained context |
| neither | Flow 3: default baseline, no integration plan |

## Stop protocol (every flow)

STOP and report `blocked` when the status marks the archive as blocked (`dependencies.archive`, `blockedReasons`, `notes`), when `integrate prepare` rejects, or when any later native command exits nonzero. In the report:

- Quote the message literally and name the continuation it gives.
- List the directed edits already made. They stay in the working tree as uncommitted changes, reported and reversible with git. Do not revert them yourself and do not commit them.
- Leave the change in `openspec/changes/`: never run Step 3 after a stop.
- Never assume "Sin impacto" because the change has no context delta, and never deduce the artifact store.

The causes native commands report, with their continuation:

| Cause | Continuation |
|---|---|
| (a) `technicalContext.maintained` is declared and the change has no `context-impact.md` | Record the expected impact or `Sin impacto: <motivo>` in `context-impact.md` |
| (b) The context delta is invalid | Fix the field, anchor or action the message names |
| (c) A git precondition fails (an affected document is not in a git repository, has uncommitted changes or is untracked) | Commit, stash or discard those changes, or put the repository under git |
| (d) A domain of the change is not declared in the baseline | Declare it in `baseline.documents`, or remove its delta |
| (e) The composition or `integrate verify` fails | Fix what the message names |
| (f) The declaration in `axiom.yaml` is invalid | `axiom workspace validate` |
| (g) A project baseline or maintained context is declared and the artifact store is `engram` or `none` | Declare `artifact_store: openspec` or `hybrid` in `openspec/config.yaml`, or remove the declaration |

A rejection that names `axiom sdd integrate verify` or `axiom sdd integrate show` as its continuation (the composition is already integrated) is a resume signal: do not compose again and continue from that step.

## Flow 1: project baseline

1. **Prepare.** Run it once, with `--json` to read the structured report:

   ```bash
   axiom sdd integrate prepare --cwd "<workspaceRoot>" --change "{change-name}" [--supersede] [--superseded-requirements "<req1,req2>"] --json
   ```

   Pass the supersede options only when the change deliberately supersedes requirements instead of removing them. `prepare` validates the declaration, the effective artifact store, the domains, the git preconditions, the edit authorization of maintained paths outside the specifications repository, the delta bodies, the compositions in memory, the ID correspondence, and the context delta with its anchors. It records the `HEAD` of every affected repository and writes only `integration.yaml`. The report carries `composeCommand`, `directed`, `contextItems`, `register`, `idMap` and `warnings`: relay the warnings in your report.

2. **Compose.** Execute `composeCommand` literally. It has the form `axiom sdd archive-compose --cwd "<workspaceRoot>" --change "{change-name}" --document "<dominio>" [--document ...] --in-place`, lists exactly the domains the plan composes, and writes all of their documents or none. An empty `composeCommand` means the change composes no document.

3. **Directed edits.** Follow "Directed edits" below for every item of `directed` and `contextItems`.

4. **Verify.**

   ```bash
   axiom sdd integrate verify --cwd "<workspaceRoot>" --change "{change-name}"
   ```

   Exit `0` marks the directed and context entries as integrated in `integration.yaml`. A nonzero exit lists every violation (check, entry, file): apply the stop protocol.

5. **Move.** Run Step 3 of `SKILL.md` unchanged.

6. **Report and trace.** After Step 5 of `SKILL.md` wrote `archive-report.md` in the archived folder, append the trace to it mechanically, never retyping it:

   ```bash
   cd "<workspaceRoot>" && axiom sdd integrate show --cwd "<workspaceRoot>" --change "{change-name}" --format markdown >> "openspec/changes/archive/YYYY-MM-DD-{change-name}/archive-report.md"
   ```

   `show` also finds the change once it is archived under its dated folder. A nonzero exit is reported, not hidden.

## Flow 2: default baseline with a maintained context

1. Run `integrate prepare` as in Flow 1. There is no composition with a profile, so `composeCommand` is empty.
2. Run the default Step 2 blocks of `SKILL.md` to compose `openspec/specs/{domain}/spec.md` with the native engine.
3. Run `axiom archive sync --path "<workspaceRoot>"` (the master catalog refresh of `SKILL.md`).
4. Make the context edits of `contextItems` (see "Directed edits").
5. Run `integrate verify`, then Step 3, the report and `integrate show`, exactly as in Flow 1.

## Flow 3: default baseline, no integration plan

Run the default Step 2 blocks, then `axiom archive sync --path "<workspaceRoot>"`, then Step 3 and the report of `SKILL.md`. No `integrate` command runs.

## Directed edits

The edit is the only model-driven part of the integration. `prepare` fixed what to edit, where and with which text; `verify` checks presence, anchors, scope and IDs afterwards.

- Use bounded `Edit` operations on the sections the plan names. Never rewrite a whole file, never copy a document through Read and Write, and never create any other file, temporary files included.
- The `text` and `reason` of each item already carry the definitive IDs. Use them exactly: never copy a provisional ID from the delta specs into a document (`verify` rejects it).
- **Prose domains (`directed`).** For each item of `directed`, open `document` and find the section whose heading title starts with the change name (`section`) as a whole token. If one exists, edit it in place; if none exists, create it. Repeating the integration must leave exactly one such section. Reflect each entry (ADDED, MODIFIED, REMOVED or RENAMED) with its `text`.
- **Register-only domains.** Never edit the documents listed in `register`: they stay `pending` in the trace.
- **Maintained context.** For each item of `contextItems`, edit `document` at `anchor` according to `action`: `add` creates the section (and the document, only when `create` is true and only inside the maintained paths), `modify` rewrites it, and `remove` deletes it so the anchor no longer resolves. When the context delta says `Sin impacto` (`context.decision` is `no_impact`), edit nothing under the maintained paths.
- Edit only the documents the plan names. Never touch `technicalContext.reference` paths, undeclared documents, registers such as `REGISTRO_INCREMENTOS.md`, or the files of the change folder (delta specs and `context-impact.md` are archived unaltered).
- Never run `git add` or `git commit`: archive leaves every edit uncommitted.
