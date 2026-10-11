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

## 5. The context delta (`context-impact.md`)

With `technicalContext.maintained` declared, each change carries `<changeRoot>/context-impact.md`: `sdd-design` drafts it, `sdd-apply` keeps it current, and the archive integrates it into the maintained documents. It is plain Markdown; outside code fences it has one of two shapes. With entries:

```markdown
# Delta de contexto: <cambio>

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pagos
- Acción: modify
- Motivo: se añade el reintento con retroceso exponencial (RF-9001)

<optional content to integrate, until the next "## Entrada:" or the end of the file>
```

Without entries:

```markdown
# Delta de contexto: <cambio>

Sin impacto: el cambio solo corrige una errata en un mensaje de error
```

- `## Entrada: <ruta>`: the path is relative to the workspace root and may be wrapped in backticks. It must lie under a `technicalContext.maintained` path, never under a `reference` path or in `baseline.documents`.
- Exactly three fields follow, `- Ancla:`, `- Acción:` (`add`, `modify`, or `remove`), and `- Motivo:`, then a blank line. An unknown or missing field is an error, and so is a repeated (document, anchor) pair.
- The anchor is `Title`, `Parent > Title`, or `Grandparent > Parent > Title`, with an optional ordinal ` #N` on the last element. More than one match is rejected, listing each; no match is valid only with `add`. A document that does not exist is valid only with `add`: archive creates it inside the maintained paths.
- `Sin impacto: <motivo>` needs a non-empty motive and cannot coexist with entries; a file with neither is invalid.
- A change with no context delta is never "Sin impacto": archive stops until the impact, or `Sin impacto: <motivo>`, is recorded, also when the change was sealed as continuous.
- Provisional IDs of the change may appear in the motive and in the optional content: archive writes the definitive IDs in the maintained documents and the change folder keeps the delta unaltered.

## 6. Remediating a gate rejected because of the context delta

The digest of the `design` gate covers only `design.md`, and the digest of a `role-apply` gate only that role's tasks file. Correcting `context-impact.md` alone does not reopen a rejected gate. When a gate was rejected only because of the context delta:

1. The phase that remediates (`sdd-design` for `design`, `sdd-apply` for `role-apply`) corrects `context-impact.md`.
2. It appends one dated line, `YYYY-MM-DD: <rejection reason>; <what changed in the delta>`, to the `## Registro de remediación` section of the artifact the gate judges (`design.md`, or the role's tasks file), creating the section when it is missing. The line is never a task checkbox, so it never alters progress.
3. The digest changes, the gate returns to `pending`, and the orchestrator presents it again with the corrected delta.

Never edit `gates.yaml` and never look for a reopen command: there is none.

## 7. Model for `sdd-archive` with directed integration

A `baseline.documents[]` entry with `integration: directed`, or a declared `technicalContext.maintained`, makes `sdd-archive` edit prose, which needs an implementation-level model. Claude Code launches `sdd-archive` with the model assigned to `sdd-apply` (the orchestrator rule "Archive Directed Integration Model (MANDATORY)"). In runtimes where the model is fixed per agent (OpenCode, Kiro, and others) the orchestrator cannot raise it per call: when the project declares directed integration or a maintained context, advise the user to assign `sdd-archive` an implementation-level model with the existing model selector. Never block archive on this: `axiom sdd integrate verify` protects the result whatever the model.
