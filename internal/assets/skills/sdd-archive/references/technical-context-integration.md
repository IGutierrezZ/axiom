# Technical Context Integration (sdd-archive reference)

`sdd-archive` loads this reference when the status projects `technicalContext.maintained` (see `../../_shared/project-baseline-convention.md`). It adds the integration of the maintained technical context to the default Step 2 blocks of `SKILL.md`, which compose the living specs under `openspec/specs/`. Step 3 (move) and Step 5 (report) of `SKILL.md` stay as written.

- `<workspaceRoot>` is `actionContext.workspaceRoot` from the status (`../../_shared/sdd-phase-common.md`, section G). Every native command below takes `--path "<workspaceRoot>"`.
- Run each native command once and include its exit code and output in the phase result. A nonzero exit always triggers the stop protocol; never retry it with a manual edit.
- The change name is `{change-name}`. Native commands read the context delta from the change folder and write only `<changeRoot>/integration.yaml`.

## Which flow applies

| The status projects | Flow |
|---|---|
| `technicalContext.maintained` | Flow 1: maintained context |
| anything else | Flow 2: no integration plan |

## Stop protocol (every flow)

STOP and report `blocked` when the status marks the archive as blocked (`dependencies.archive`, `blockedReasons`, `notes`), when `axiom archive context prepare` rejects, or when any later native command exits nonzero. In the report:

- Quote the message literally and name the continuation it gives.
- List the context edits already made. They stay in the working tree as uncommitted changes, reported and reversible with git. Do not revert them yourself and do not commit them.
- Leave the change in `openspec/changes/`: never run Step 3 after a stop.
- Never assume "Sin impacto" because the change has no context delta, and never deduce the artifact store.

The causes native commands report, with their continuation:

| Cause | Continuation |
|---|---|
| (a) `technicalContext.maintained` is declared and the change has no `context-impact.md` | Record the expected impact or `Sin impacto: <motivo>` in `context-impact.md` |
| (b) The context delta is invalid | Fix the field, anchor or action the message names |
| (c) A git precondition fails (an affected document is not in a git repository, has uncommitted changes or is untracked) | Commit, stash or discard those changes, or put the repository under git |
| (d) The composition of a delta spec or `axiom archive context verify` fails | Fix what the message names |
| (e) The declaration in `axiom.yaml` is invalid, or it still carries the retired block reported as `baseline_retired` | `axiom workspace validate`; for a retired block, migrate its requirements with `axiom archive adopt scan --path "<workspaceRoot>" --change {change-name}` |
| (f) A maintained context is declared and the artifact store is `engram` or `none` | Declare `artifact_store: openspec` or `hybrid` in `openspec/config.yaml`, or remove the declaration |

A rejection that names `axiom archive context verify` or `axiom archive context show` as its continuation (the context is already integrated) is a resume signal: do not edit again and continue from that step.

## Flow 1: maintained context

1. **Prepare.** Run it once, with `--json` to read the structured report:

   ```bash
   axiom archive context prepare --path "<workspaceRoot>" --change "{change-name}" --json
   ```

   `prepare` validates the declaration, the effective artifact store, the git preconditions, the edit authorization of maintained paths outside the specifications repository, and the context delta with its anchors. It records the `HEAD` of every affected repository and writes only `integration.yaml`. The report carries `contextItems` and `warnings`: relay the warnings in your report.

2. **Compose the living specs.** Run the default Step 2 blocks of `SKILL.md` to compose `openspec/specs/{domain}/spec.md` with the native engine.

3. **Refresh the master catalog.** Run `axiom archive sync --path "<workspaceRoot>"` (the master catalog refresh of `SKILL.md`).

4. **Context edits.** Follow "Context edits" below for every item of `contextItems`.

5. **Verify.**

   ```bash
   axiom archive context verify --path "<workspaceRoot>" --change "{change-name}"
   ```

   Exit `0` marks the context entries as integrated in `integration.yaml`. A nonzero exit lists every violation (check, entry, file): apply the stop protocol.

6. **Move.** Run Step 3 of `SKILL.md` unchanged.

7. **Report and trace.** After Step 5 of `SKILL.md` wrote `archive-report.md` in the archived folder, append the trace to it mechanically, never retyping it:

   ```bash
   cd "<workspaceRoot>" && axiom archive context show --path "<workspaceRoot>" --change "{change-name}" --format markdown >> "openspec/changes/archive/YYYY-MM-DD-{change-name}/archive-report.md"
   ```

   `show` also finds the change once it is archived under its dated folder. A nonzero exit is reported, not hidden.

## Flow 2: no integration plan

Run the default Step 2 blocks, then `axiom archive sync --path "<workspaceRoot>"`, then Step 3 and the report of `SKILL.md`. No `axiom archive context` command runs.

## Context edits

The edit is the only model-driven part of the integration. `prepare` fixed what to edit, where and with which text; `verify` checks presence, anchors, scope and IDs afterwards.

- Use bounded `Edit` operations on the sections the plan names. Never rewrite a whole file, never copy a document through Read and Write, and never create any other file, temporary files included.
- The `text` and `reason` of each item already carry the definitive IDs. Use them exactly: never copy a provisional ID from the delta specs into a document (`verify` rejects it).
- **Maintained context.** For each item of `contextItems`, edit `document` at `anchor` according to `action`: `add` creates the section (and the document, only when `create` is true and only inside the maintained paths), `modify` rewrites it, and `remove` deletes it so the anchor no longer resolves. When the context delta says `Sin impacto` (`context.decision` is `no_impact`), edit nothing under the maintained paths.
- Edit only the documents the plan names. Never touch `technicalContext.reference` paths, undeclared documents, registers such as `REGISTRO_INCREMENTOS.md`, or the files of the change folder (delta specs and `context-impact.md` are archived unaltered).
- Never run `git add` or `git commit`: archive leaves every edit uncommitted.
