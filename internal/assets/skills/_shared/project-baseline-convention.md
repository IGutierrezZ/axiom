# Technical Context Convention (shared across SDD skills)

A project may declare in its `axiom.yaml` a technical context: documentation that lives outside `openspec/specs/` and that the SDD phases read (`reference`) or keep current (`maintained`). The living specification is only `openspec/specs/`. Phase skills never read `axiom.yaml` to find out: the status exposes the declaration, and this convention says how to read it and what is forbidden. Without a declaration nothing changes and every phase behaves as before. The file keeps its historical name so that existing references stay valid.

## 1. Locators come from the status

- Read the `technicalContext` locators from the status (`axiom sdd status <change> --json`); never read or interpret `axiom.yaml`.
- `technicalContext.maintained` and `technicalContext.reference` are two independent modes. Each one has absolute `paths` and an optional `entry` file.
- A field that is absent from the status means there is no valid declaration of it. Never guess paths. If `notes` report a declaration cause, the declaration is invalid, not absent: report that cause and `axiom workspace validate`. A cause coded `baseline_retired` means `axiom.yaml` still carries a retired block: report it and its continuation, `axiom archive adopt scan`; archive stays blocked until the block is gone.
- Paths in the status are absolute; resolve any other relative artifact path against `actionContext.workspaceRoot`, as `sdd-phase-common.md` section G states.

## 2. Reading

- The living specification is `openspec/specs/{domain}/spec.md`: it is the only source of the current behavior of a capability.
- Read the `entry` file of each declared `technicalContext` mode and consult the remaining `paths` only on demand, without loading them in advance.
- A mode declared without `entry` has nothing to preload: open its files only when the work needs them.
- Only `sdd-explore`, `sdd-design`, `sdd-apply` and `sdd-verify` are required to read the technical context; the other phases are not.

## 3. Prohibitions

- Never create, modify, or delete files under `technicalContext.reference` paths.
- Files under `technicalContext.maintained` paths change only through the archive integration (`axiom archive context`), never by editing them directly from a phase.
- A phase never writes `openspec/specs/` or `openspec/INDEX.md`: only `sdd-archive` promotes the delta specs of the change.

## 4. Checking the context delta from the spec and apply phases

`axiom archive context check --path "<workspaceRoot>" --change <cambio>` validates the context delta of the change (format, maintained and reference paths, anchors) and writes nothing. `sdd-design` drafts the delta, `sdd-apply` keeps it current and may run the check; the archive runs the writing verbs (`prepare`, `verify`) and `show` for the trace.

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

- `## Entrada: <ruta>`: the path is relative to the workspace root and may be wrapped in backticks. It must lie under a `technicalContext.maintained` path, never under a `reference` path.
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

## 7. Model for `sdd-archive` with a maintained context

A declared `technicalContext.maintained` makes `sdd-archive` edit prose, which needs an implementation-level model. Claude Code launches `sdd-archive` with the model assigned to `sdd-apply` (the orchestrator rule "Archive Directed Integration Model (MANDATORY)"). In runtimes where the model is fixed per agent (OpenCode, Kiro, and others) the orchestrator cannot raise it per call: when the project declares a maintained context, advise the user to assign `sdd-archive` an implementation-level model with the existing model selector. Never block archive on this: `axiom archive context verify` protects the result whatever the model.
