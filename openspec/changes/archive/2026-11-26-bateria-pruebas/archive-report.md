# Reporte de Archivado Formal: bateria-pruebas

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: verificación local limpia (`verify-report.md`, Estado `limpio`, en el momento de la verificación: 0 críticos, 1 aviso W1, 2 sugerencias). W1 sigue abierto como hallazgo informativo (ver «Estado final»).
- Modo de artefactos: `openspec` (ficheros en `openspec/`; sin trazas de Engram leídas para este cierre).
- Kickoff: `continuous`, rol único `fullstack`, `handoff_policy: none`, `post_archive_policy: bug_only`, sin `docs_publication`. No es un cambio ISO (`archive-check` con código 0, sin hallazgos).
- Rama de archivado: `inc/bateria-pruebas-archive` (worktree `C:\repos\axiom-wt\bateria-pruebas-archive\axiom`), desde `origin/main` (da433c3a).
- Destino en Archivo: `openspec/changes/archive/2026-11-26-bateria-pruebas/`

## Estado final (a fecha de cierre)
Fuentes, por orden de autoridad: (1) `tasks.md` archivado; (2) instrucciones de cierre del orquestador; (3) `verify-report.md` y `apply-progress.md` (instantáneas).
- Implementación: fusionada en `main` con la PR #356 (merge commit `da433c3a`), con los commits de tarea T1 a T10 (`cebd5005e` a `961bb3344`). Fuente: orquestador, de mayor rango que las instantáneas.
- Tareas: 10/10 completadas en `tasks.md` (fuente de mayor rango). Sin tareas pendientes.
- Correcciones posteriores al verify: commit `c9ece7113` (`fix(idmigration)`), que declara como residuos los identificadores REQ de los fixtures de `acceptancetests` en `internal/idmigration/residues.yaml`. Fuente: orquestador. CI en verde según el orquestador (14 pass, 4 skipping).
- Verify: `verify-report.md` (commit `c32618040`, en el momento de la verificación) con `**Estado:** limpio`, 0 CRITICAL, 1 WARNING y 2 SUGGESTION. Hallazgos no resueltos y registrados como informativos:
  - W1: RF-GAT-32 (la parada `tasks` presenta `pruebas.md` y declara su ausencia) se cumple solo de forma indirecta: el status proyecta `acceptanceTests` y la skill `sdd-tasks` lo documenta, pero el texto del orquestador que presenta la parada no lo indica y no hay test que lo fije. Aplazado por C-05-d (edita la doctrina en paralelo). Anotado en `apply-progress.md` y en `coverage-review.design.md` (hueco 3).
  - S1: ejecutar `scripts/deadcode-ratchet.sh` en CI.
  - S2: documentar en el manual que `enabled` es obligatorio dentro del bloque.
- Revisión de cobertura (informativa, no bloqueante): 10 huecos en la spec (`coverage-review.spec.md`) y 9 en el design (`coverage-review.design.md`).

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo: las specs delta solo declaran ADDED, con identificadores definitivos ya asignados.

## Requisitos Integrados
Identificadores comprobados libres en `openspec/specs/` antes de integrar (sin coincidencias).

| Capacidad | Requisitos añadidos |
|-----------|---------------------|
| local-web-dashboard | RF-WEB-17 |
| sdd-block-review-gates | RF-GAT-31, RF-GAT-32 |
| sdd-lane-kickoff | RF-KOF-17 a RF-KOF-23 |
| workspace-topology | RF-TOP-19, RF-TOP-20 |

Sin provisionales `prov:` ni duplicados (`duplicates[]` vacío; `assignments[]` vacío).

## Integración de Especificaciones
- Comando: `axiom sdd archive-check --cwd C:\repos\axiom-wt\bateria-pruebas-archive\axiom --change bateria-pruebas`, código de salida 0 (cambio no ISO).
- Comando previo: `axiom sdd archive-integrate --cwd <worktree> --change bateria-pruebas --check --json`, código 0, sin hallazgos.
- Comando: `axiom sdd archive-integrate --cwd <worktree> --change bateria-pruebas --json`, código 0 (`nothingToIntegrate: false`). Capacidades compuestas: local-web-dashboard, sdd-block-review-gates, sdd-lane-kickoff, workspace-topology. Diff de las specs vivas: 460 inserciones y 0 eliminaciones.
- Índice maestro: `axiom archive sync --path <worktree>`, código 0. `openspec/INDEX.md` regenerado: 49 especificaciones, 542 requisitos activos, 2504 escenarios. Las eliminaciones del índice son solo la cabecera de totales y las filas de las cuatro capacidades modificadas (recuentos actualizados).

## Verificación de la Movida
- Comando: `git mv openspec/changes/bateria-pruebas openspec/changes/archive/2026-11-26-bateria-pruebas`, código 0, con snapshot previo (`cp -R`).
- `diff -r` (snapshot previo vs. destino): salida vacía, código 0.
- El origen `openspec/changes/bateria-pruebas/` ya no existe.

## Pruebas Localizadas Ejecutadas
- `go test ./internal/idmigration -run 'TestRepositoryLegacyIDResidues|Cubre|Index' -count=1`: ok (código 0).
- `go test ./internal/livingdoc -run 'Index|Catalog|Sync' -count=1`: ok (código 0).

## Artefactos Consolidados
- `proposal.md`
- `specs/` (4 capacidades: local-web-dashboard, sdd-block-review-gates, sdd-lane-kickoff, workspace-topology)
- `design.md`
- `tasks.md` (10/10 completadas)
- `verify-report.md` (instantánea; Estado `limpio`, con W1 abierto)
- `apply-progress.md`, `coverage-review.spec.md`, `coverage-review.design.md`, `exploration.md`, `kickoff.yaml`
