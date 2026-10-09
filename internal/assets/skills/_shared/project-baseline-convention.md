# Project Baseline and Technical Context Convention (shared across SDD skills)

A project may declare in its `axiom.yaml` that its living specification (the baseline) and its technical context belong to the project and live outside `openspec/specs/`. Phase skills never read `axiom.yaml` to find out: the status exposes the declaration, and this convention says how to read it and what is forbidden. Without a declaration nothing changes: `openspec/specs/` stays the baseline and every phase behaves as before.

## 1. Locators come from the status

- Read the `baseline` and `technicalContext` locators from the status (`axiom sdd status <change> --json`); never read or interpret `axiom.yaml`.
- `baseline.documents[]` lists each declared document: `domain`, absolute `path`, and `integration` (`compose`, `directed` or `register`).
- `technicalContext.maintained` and `technicalContext.reference` are two independent modes. Each one has absolute `paths` and an optional `entry` file.
- A field that is absent from the status means there is no valid declaration of it. Never guess paths. If `notes` report a declaration cause, the declaration is invalid, not the default: report that cause and `axiom workspace validate`, and do not treat the project as having the OpenSpec default baseline.
- Paths in the status are absolute; resolve any other relative artifact path against `actionContext.workspaceRoot`, as `sdd-phase-common.md` section G states.

## 2. Reading

- When the status projects `baseline`, read the project's specifications from `baseline.documents` and treat `openspec/specs/` as the baseline only when `baseline` is absent.
- Read the `entry` file of each declared `technicalContext` mode and consult the remaining `paths` only on demand, without loading them in advance.
- A mode declared without `entry` has nothing to preload: open its files only when the work needs them.
- Only `sdd-explore`, `sdd-design`, `sdd-apply` and `sdd-verify` are required to read the technical context; the other phases are not.

## 3. Prohibitions

- Never create, modify, or delete files under `technicalContext.reference` paths.
- Files under `technicalContext.maintained` paths and under `baseline.documents` change only through the archive integration, never by editing them directly from a phase.
- When the status projects `baseline`, never create or write `openspec/specs/` or `openspec/INDEX.md`: a second baseline would contradict the project's own. The same applies when `notes` report an invalid `baseline:` declaration.
- A domain that the baseline does not declare is rejected by archive, not promoted to `openspec/specs/{domain}/`.
