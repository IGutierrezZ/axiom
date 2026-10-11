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

## 4. Deltas for a baseline with a requirement profile

This applies to a `baseline.documents[]` entry with `integration: compose`: a document with a requirement profile (the project's own heading pattern, level, and identity). `sdd-spec` writes the deltas of those domains with these rules; prose domains (`directed` or `register`) use the plain delta format with no ID rules.

- A requirement name is the project's heading without the leading `#` signs, so the level's `#` signs, one space, and the name form a heading the profile recognizes (for example `RF-120: <title>`). A name that does not match is rejected, naming the requirement and the pattern.
- MODIFIED, REMOVED, and RENAMED requirements use the IDs that exist in the baseline document. Read them from the document; never guess or renumber.
- An ADDED requirement carries a provisional ID: it matches the profile's heading pattern, equals no ID of the document, is unique across the change, and sits outside the existing range. `axiom sdd integrate check --cwd "<workspaceRoot>" --change <cambio>` prints a suggested provisional start for each ID family. A family with no ID in the document cannot receive an ADDED requirement.
- An ADDED requirement also carries a note line `(Placement: after <ID>)` or `(Placement: section "<title>")`. The note is delta metadata and is never integrated.
- A requirement body contains no heading at or above the level of the profile's requirement headings outside code fences; `check` names the offending line.
- Archive replaces each provisional ID with the next free definitive ID of its family (with the zero padding of the family's highest ID) and writes the definitive IDs in every integrated text, references between new requirements included. The correspondence is recorded in the integration trace.
- Before the spec phase is finished, run `axiom sdd integrate check --cwd "<workspaceRoot>" --change <cambio>` and correct every violation it reports. `check` writes nothing.
