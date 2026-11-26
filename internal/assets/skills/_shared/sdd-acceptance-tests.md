# Acceptance test battery (`pruebas.md`)

Shared contract for `sdd-tasks` (and the refinement rule for `sdd-apply`). The file is written in `<changeRoot>/pruebas.md` only when ALL of these hold:

- The change goes through the SDD lane. An ODD change NEVER generates `pruebas.md`.
- `axiom sdd status --cwd "<workspaceRoot>" <change> --json` projects `acceptanceTests.active: true`.
- The artifact store is `openspec` or `hybrid`.

With the feature not active (`enabled: false`, no block, or an invalid block) do not write the file and do not mention it. The execution style (`continuous` or `checkpointed`) does not change this rule. The battery never changes `nextRecommended` or the dependencies.

## Format

A single markdown table outside code fences, with these nine columns, named exactly as written (Spanish, case-sensitive), in any order, followed by the separator line:

`ID`, `Cubre`, `Rol`, `Tipo`, `Prioridad`, `Entornos`, `Precondiciones`, `Pasos`, `Resultado esperado`

- `ID`: `P-` plus at least two digits (`P-01`), unique and stable: never renumber on edit.
- `Cubre`: non-empty list separated by `;`. Each entry is `ID` (cites a requirement) or `ID#<exact scenario name>` (cites that scenario). A scenario counts as covered only when cited with `ID#name`. Scenario names containing `;` cannot be cited.
- `Rol`: one sealed role of the change. `Tipo`: `manual` or `automatizable`. `Prioridad`: `alta`, `media` or `baja`.
- `Entornos`: non-empty list separated by `;`, without repeats, each one declared in `acceptance_tests.environments` (`acceptanceTests.environments` in the status).
- `Precondiciones` may be empty. `Pasos` and `Resultado esperado` are not empty.
- A `|` inside a cell is written `\|`. Text outside the table is ignored.

Example:

```markdown
| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |
|----|-------|-----|------|-----------|----------|----------------|-------|--------------------|
| P-01 | RF-EJ-01#Acceso a punto de caducar | fullstack | automatizable | alta | dev; pre | Acceso que caduca en siete días | Llegar a la fecha de aviso | La persona recibe un mensaje |
| P-02 | RF-EJ-02; RF-EJ-02#Renovar desde el aviso | fullstack | manual | media | pre | Aviso recibido | Pulsar renovar | El acceso se amplía un año |
```

## Minimization

Minimize the number of cases by applying these four layers, in this order. Minimization is the model's job: Go only checks and reports.

1. Mandatory traceability: every ADDED or MODIFIED requirement and every one of its scenarios is cited.
2. Equivalence partitions and boundary values.
3. Pairwise, only with three or more independent parameters.
4. Merge cases that share the same preconditions.

## Check and presentation

1. Run `axiom sdd tests check --cwd "<workspaceRoot>" --change <change> --json` and fix the file until there are no findings, or leave a note of the ones you cannot resolve.
2. Return `pruebas.md` in `artifacts` and the result of the check (cases, scenarios and requirements covered, open findings) in the summary, so the `tasks` stop can present it beside the task files. A result with findings is reported as such, never as clean.
3. When the feature is active and you did not write `pruebas.md`, declare its absence in the result with the reason. When the feature is not active, say nothing about it.

## Refinement by `sdd-apply`

`sdd-apply` MUST NOT widen the scope of `pruebas.md`: it MAY only refine paths or URLs inside `Pasos`. It never adds, removes or reassigns cases and never alters `Cubre`.
