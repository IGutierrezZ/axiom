# Revisión de cobertura de la spec — plantilla-ramas

Revisión informativa: no bloquea fases, compuertas ni archivado, y no cambia ningún digest.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Entradas: `proposal.md` y los tres deltas de `specs/`. Ambos devolvieron un resultado utilizable. El informe es la unión deduplicada.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `change-branch-lifecycle` · prov:BRA:1 | `{slug}`: no se fija cómo se reconoce el prefijo del ID de work item (`us-2525-` incluye el tipo, que no sale del ID), si distingue mayúsculas, ni qué pasa si el slug queda vacío (`us-2525`) o inválido. | A, B |
| 2 | use_case | `change-branch-lifecycle` · prov:BRA:5 | Falta el escenario del segundo clon con `ensure --role`, sin registro y con la rama ya publicada: mismo nombre desde la instantánea y los vínculos versionados, y adopción de la rama publicada (RF-BRA-27). | A, B |
| 3 | edge_case | `change-branch-lifecycle` · RF-BRA-15 | Task revinculada a otro ID tras crear la rama y con el registro borrado: no se fija la prioridad de recuperación (ramas existentes, instantánea, plantilla) ni qué pasa si discrepan. | A, B |
| 4 | edge_case | `change-branch-lifecycle` · prov:BRA:3 | Interacción de `--branch` y `--prefix` con la plantilla: `--branch` sobre una rama del cambio diferida (¿se crea ya, se difiere o se rechaza?), `--branch` con `--role`, `--prefix` sin `{prefix}` en la plantilla, y qué rama recibe `--branch` en un repo con seguimiento y ramas de rol. | A, B |
| 5 | error | `sdd-lane-kickoff` · RF-KOF-06 / prov:BRA:4 | El rechazo de la rama compartida con `{role}`/`{task_ref}` solo se prueba en `ensure`; falta en `seal` (para no sellar una instantánea irresoluble), en ODD y en `workspace validate`. | A, B |
| 6 | error | `change-branch-lifecycle` · prov:BRA:4 | Solo se rechazan colisiones entre ramas del mismo repo; falta el nombre que coincide con la rama de integración o principal, con el nombre reservado de archivado o con la rama de otro cambio activo. | A |
| 7 | edge_case | `change-branch-lifecycle` · RF-BRA-16 / prov:BRA:5 | `status` con `integrations.yaml` ilegible y plugin activo; nombre esperado de ramas aún no resolubles; base, destino de PR y promoción `ask` (RF-BRA-28/29) de una rama diferida, y qué worktree se materializa en un repo sin rama todavía. | A, B |
| 8 | edge_case | `change-branch-lifecycle` · RF-BRA-14 | ODD: no se especifica el origen de `{ref}`, `{slug}` y `{kind}` (no hay kickoff), ni `{ref}` sin work item en ODD. | B |
| 9 | edge_case | `change-branch-lifecycle` · prov:BRA:3 / RF-BRA-19, 21 | Consumidores del archivado (`archive_plan.go`, `planner_archive.go`, `branch done`, cambios pendientes de archivar): qué nombre usan sin instantánea ni registro, y `--prefix` en `ensure --archive` con `{prefix}`. | B |
| 10 | uncovered_requirement | `workspace-topology` · RF-TOP-21 | `PUT /api/settings`: los escenarios solo prueban `git.branch_template`; faltan `repositories[].branch_template` y `roles[].branch_template`, el rechazo de `{role}`/`{task_ref}` en la rama con `openspec/` por la API y el aviso de rol sin efecto en monorepo. | A |
| 11 | edge_case | `workspace-topology` · prov:TOP:1 | Multirepo: falta `repositories[].branch_template` del repo de specs con `{role}`/`{task_ref}`, el repo de specs in situ (RF-BRA-05) y `git:` con `{task_ref}` sin ningún repo de código de un solo rol que la consuma. | A |
| 12 | uncovered_requirement | `workspace-topology` · RF-TOP-09 | El punto 11 de la propuesta (ejemplos comentados en `axiom.example.yaml`, ejecutados por `TestExampleGitBlockIsValidOnceUncommented`, y documentación del bloque `git:`) no tiene requisito ni escenario. | A, B |

Evidencia: A leyó la propuesta y los tres deltas; B además consultó `archive_plan.go`, `verdict.go` y `planner_archive.go`.
