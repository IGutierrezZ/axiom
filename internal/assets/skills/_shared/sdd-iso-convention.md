# SDD ISO Convention

Shared guidance for ISO evidence and for the bug profile. The engine validates every format below; this file only describes them. The Spanish literals are part of the format and must be reproduced exactly, because the ODD lane uses the same section and the same line.

Read the state from `axiom sdd status --cwd <repo> --json`, never from `axiom.yaml` or from this file. Native status is authoritative.

## When It Applies

- **ISO project.** `axiom.yaml` declares `sdd: {iso: true}`. Status projects it in the `iso` field (`iso.project: iso`). An ISO project only uses SDD: all work, including small, understood work, goes through the SDD lane, and no `odd/tasks/<feature>.md` is created.
- **ISO change.** `axiom sdd kickoff seal` copies the project's `sdd.iso` into `kickoff.yaml`. Status projects it as `iso.change: iso`. A change sealed without that snapshot is not ISO, even in an ISO project, and keeps the rules it had before. A discrepancy between snapshot and project appears as a note: the rules of the change follow its snapshot.
- **Bug profile.** It applies to every project, ISO or not, and is described below.
- **Seal creates the change folder.** Run `axiom sdd kickoff seal` first; never create `openspec/changes/<change>/` by hand beforehand.
- **Store.** An ISO change needs files: `artifact_store: openspec` or `hybrid`. With `engram` or `none`, status reports `blocked(iso_store_unsupported)`.
- **Reasons.** Every ISO refusal is a typed reason, `blocked(<token>): <detail>; <continuation>`. The token table is in `sdd-status-contract.md`; follow the continuation of the reason and never reinterpret it.

## Requirement Identifiers and Scenarios (spec phase)

- Each requirement name of a delta carries exactly one `REQ-<n>.<m>`, for example `### Requirement: Cobro con IVA (REQ-30.1)`. A repeated copy of the same identifier counts as one; a malformed token such as `REQ-30` does not count.
- An ADDED identifier is unique among all the ADDED requirements of the change and absent from the baseline. MODIFIED and REMOVED requirements carry the identifier of the requirement they name; for a requirement that has none, use a RENAMED entry first. A RENAMED entry carries it in the new name.
- Every ADDED or MODIFIED requirement has at least one `#### Scenario:` outside code fences, and no scenario name repeats within a requirement. REMOVED and RENAMED requirements need none.

## The `Cubre:` Line (tasks phase)

- A line of its own in `tasks.md` or `tasks.<role>.md`, indented or bulleted if you like, outside code fences: `Cubre: REQ-x[, REQ-y ...]`. Put it under the task that covers the requirement.
- Coverage is joint across every tasks file of the change: each requirement of the delta (ADDED, MODIFIED, REMOVED and the new name of a RENAMED) appears in some `Cubre:` line. A cited identifier that is neither in the delta nor in the baseline is `iso_cubre_unknown`.
- Do not cite a requirement in a bug without impact on the spec: `iso_bug_contradiction`.

## Bug Profile

A bug goes through SDD with the bug profile along one of three routes. The engine reads `proposal.md`, so write these literals exactly.

| Route | When | Delta | `Cubre:` | Section in `proposal.md` |
|---|---|---|---|---|
| Existing requirement | The bug restores a requirement that already exists | none | cites that requirement | `## Informe de bug` |
| No impact on the spec | The bug touches no requirement | none | none | `## Informe de bug` and `## Impacto en la spec` |
| Delta | The requirement was wrongly specified or missing | yes | as a normal change | none needed |

With a delta, the sections are ignored and the change is a normal one.

```markdown
## Informe de bug

Observado: el total omite el IVA reducido.
Esperado: el total aplica el IVA de cada línea.
Referencia: JIRA-1234

## Impacto en la spec

Sin impacto en la spec
Motivo: el defecto estaba en el script de despliegue, no en un requisito.
```

- `## Informe de bug` needs `Observado:` and `Esperado:` with a value on the same line; `Referencia:` is optional.
- `## Impacto en la spec` needs the exact line `Sin impacto en la spec` (case-sensitive, nothing added) and a non-empty `Motivo:` line.
- Without a delta and without `## Impacto en la spec`, the route is the existing requirement and `Cubre:` must cite it; that requirement needs at least one scenario, and otherwise the bug is a normal change with a delta.
- In an ISO change, the `Motivo:` is the evidence of a bug without impact, and an absent or empty matrix is valid. Nothing is integrated into the baseline when it is archived.
- An incomplete declaration keeps `specs` open with `blocked(bug_report_incomplete)` or `blocked(bug_no_impact_incomplete)`.

## The Verification Matrix (verify phase)

In an ISO change, `verify` is mandatory before archive: status routes to `verify` while the matrix is missing or invalid, and `archive` stays blocked until it is valid and every row passes. With deferred or optional roles open, only the blocking roles hold `verify` back.

- The matrix is an additional section of the consolidated `verify-report.md` (the report of the single role `fullstack`), next to the "Cumplimiento técnico" section. It never contains a line that starts with `verdict:`: every row starts with `|`.
- Exact level-2 heading `## Matriz de verificación`, outside code fences, and only one per report.
- One row per affected scenario, no row for any other scenario, none repeated.

```markdown
## Matriz de verificación

| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |
|---|---|---|---|---|---|---|---|---|---|---|
| REQ-30.1 | Cobro con IVA general | unitario | `internal/pagos/cobro_test.go:TestCobraIVA` | `go test ./internal/pagos -run TestCobraIVA -count=1` | 0 | | | | pass | 1a2b3c4 |
| REQ-30.2 | Recibo impreso | manual | acta-12 | | | Ana Pérez | 2026-11-12 | La impresora fiscal no tiene emulador | pass | |
```

- Columns are recognized by name, in any order. Required: Requisito, Escenario, Tipo, Referencia, Comando, Salida, Responsable, Fecha, Justificación, Resultado. `Commit` is optional and never validated. A cell that does not apply stays empty or holds `-`.
- `Tipo` is `unitario`, `integración`, `e2e` or `manual`. `Resultado` is `pass` or `fail`.
- **Automatic row** (any type except `manual`): a resolvable `Referencia`, a non-empty `Comando`, an integer `Salida`; `pass` requires `Salida` `0`.
- **Manual row**: `Responsable`, `Fecha` as `AAAA-MM-DD` and `Justificación`, all non-empty. The reference is optional and is never resolved.
- **Test reference forms**, for automatic rows: `ruta/fichero:símbolo` (path relative to the project root, with `/`; the symbol appears as a whole identifier) and `TestX` (a top-level Go test function found in the working tree). The engine only reads files; it never runs the `Comando`, so run it yourself and record its real exit code.
- A bug without impact on the spec has no affected scenario: an absent or empty matrix is valid, and any row is invalid.
- Run `sdd-verify` again when the matrix is invalid or a row fails; fix the behaviour first when a row fails.
- **Historical matrix.** When a new verify replaces an earlier report, move the earlier matrix into the *Historical context* section and lower its heading to level 3 with the date: `### Matriz de verificación (histórica, AAAA-MM-DD)`. A report with history never has two level-2 matrix sections.

## Archive Check and Interrupted Archives

- `axiom sdd archive-check --cwd <workspaceRoot> --change <name> [--json]` is read-only. It exits `0` for a non-ISO change or when there are no findings, and `1` with the typed reasons otherwise. It covers the cases that compose nothing, such as a new capability that is copied and a bug without a delta.
- `sdd-archive` runs it before composing and before any move. `archive-compose` also refuses on the same reasons, as defence in depth.
- `--json` (`axiom.sdd-archive-check/v1`) reports `domains[]` with `alreadyComposed`: `true` when the baseline already holds the whole delta of that domain.
- **Resume an interrupted archive.** If an earlier run composed some domains and stopped before moving the folder, compose only the domains with `alreadyComposed: false`, then finish the move. Composing a domain again fails with "a requirement named ... already exists".

## Trace

`axiom sdd trace --cwd <dir> --change <name> [--integration-branch <ref>] [--json]` is read-only. It derives from git history, never from a state file: the commit that brought the change with its `tasks.md` and `verify-report.md`, the archive commit, and the chain requirement of the delta, `Cubre:` line, matrix rows. A divergence from the merged `integration` gate evidence appears as a discrepancy, never as an error. `--json` is `axiom.sdd-trace/v1`.

## Reviews Stay Informational

In an ISO change the matrix is the only verification result that blocks: review findings, a `JUDGMENT: ESCALATED` verdict and the coverage review remain informational. A non-ISO change keeps the rule that missing or failed verification never gates archive.
