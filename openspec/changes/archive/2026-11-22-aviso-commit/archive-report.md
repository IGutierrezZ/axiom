# Informe de archivado: aviso-commit (C-02-d)

Fecha de archivado: 2026-11-22.
Carril: SDD. Topología: `monorepo-embedded`. Almacén de artefactos: `hybrid` (openspec para los ficheros, con copia en Engram `sdd/aviso-commit/archive-report`). Rol: `fullstack`. Ejecución: `continuous`.

## Resumen

El cambio `aviso-commit` queda archivado. Su código y su documentación ya están fusionados en `main` mediante la PR #289 (merge `a0967497`). En este archivado se integraron sus tres deltas (todos `ADDED`, REQ-33.1 a REQ-33.12) en las especificaciones vivas de `sdd-block-review-gates`, `sdd-orchestrator-assets` y `organic-agent-trigger-rules`, se movió la carpeta del cambio a `openspec/changes/archive/2026-11-22-aviso-commit/` y se regeneró el catálogo `openspec/INDEX.md`. No se hizo ningún commit, push, fetch ni apertura de PR.

Estado al cierre: las 20 tareas de `tasks.md` están marcadas (20/20). El informe de verificación es `limpio` (0 críticos, 0 advertencias, 2 sugerencias) y el judgment-day es `APPROVED`. Los huecos de cobertura abiertos son informativos y se listan más abajo.

## Especificaciones integradas

Los tres deltas se compusieron con `axiom sdd archive-compose` (salida a un fichero temporal y `mv` atómico). Las tres specs vivas ya existían, así que no hubo copia directa. Ningún requisito previo se eliminó ni se reescribió: `git diff --numstat` sobre `openspec/specs` da 0 líneas eliminadas.

| Capacidad | Requisitos antes | Requisitos después | Cambios integrados |
|---|---|---|---|
| `sdd-block-review-gates` | 18 | 25 | ADDED REQ-33.1 a REQ-33.7 |
| `sdd-orchestrator-assets` | 21 | 25 | ADDED REQ-33.8 a REQ-33.11 |
| `organic-agent-trigger-rules` | 20 | 21 | ADDED REQ-33.12 |

Los recuentos "antes" y "después" son los que `openspec/INDEX.md` declara antes y después de la regeneración (el catálogo pasa de 612 a 624 requisitos activos y de 2041 a 2086 escenarios BDD). Los identificadores REQ-33.x aparecen en los tres dominios (`rg -c "REQ-33\."`: 8, 8 y 2 coincidencias por fichero).

## Poda histórica (ODD-4.2)

Ninguno. Incremento puramente aditivo: no hay `MODIFIED`, `REMOVED` ni `RENAMED` en los deltas, y no se usó `--supersede`.

## Artefactos consolidados

- `proposal.md`
- `design.md`
- `tasks.md` (20 de 20 tareas marcadas al cierre)
- `verify-report.md` (estado `limpio` en el momento de la verificación)
- `judgment-day.fullstack.md` (`APPROVED`)
- `coverage-review.spec.md` y `coverage-review.design.md` (ronda 1)
- `exploration.md`
- `kickoff.yaml`
- `specs/` (tres deltas: `sdd-block-review-gates`, `sdd-orchestrator-assets`, `organic-agent-trigger-rules`)

No existe `gates.yaml` en este cambio: el kickoff es `continuous`, sin puertas de bloque.

## Estado de tareas

- `tasks.md`: 20 tareas (1.1 a 6.2), todas marcadas `[x]`. Fuente: `tasks.md` (rango más alto). No se modificó el fichero.
- Nota: el `verify-report.md` menciona "21 tareas (1.1 a 6.2)". El recuento del fichero es 20 y el rango 1.1 a 6.2 corresponde a 20 casillas. Prevalece `tasks.md`.

## Resultado de verificación

Fuente: `verify-report.md`, escrito durante la verificación. Es una instantánea de ese momento y no se reescribe.

- En el momento de la verificación: `**Estado:** limpio`, con 0 críticos, 0 advertencias y 2 sugerencias.
- Sugerencia 1: el hash fijado del orquestador de OpenCode que compone Kilocode cambió (deliberado, según el comentario del propio test). Pendiente de confirmar en la revisión de la PR.
- Sugerencia 2: la ejecución de `internal/cli` salió de la caché de Go. Es válida porque no hubo cambios de código posteriores, pero no se midió el tiempo real.
- Comprobaciones ejecutadas en la verificación: `go vet` sobre los paquetes afectados, `go test` sobre `kickoff`, `sddstatus`, `assets`, `components/sdd`, `components/agentguidance` y `TestGolden`, y `go test ./internal/cli/` con filtro `Gate|CommitReview|Kickoff|SDDStatus`.
- Cambios posteriores a la verificación en la rama de cambio: el judgment-day y la corrección de JD-1 a JD-3 (commit `e406b12e`), la documentación de la corrección (`c6c2bd4a`) y el arreglo `bc9bcada` (`fix(cli): nombrar la salida en los rechazos de --commit-review`). Ninguno de ellos está registrado como hallazgo del `verify-report`.

## Judgment-day

Fuente: `judgment-day.fullstack.md`.

- Veredicto: `APPROVED`. Ronda 1, jueces ciegos A y B. Hallazgos CRITICAL confirmados: 0. No hubo ronda de correcciones ni re-juicio.
- JD-1, JD-2 y JD-3 (WARNING, informativos): corregidos fuera del juicio en `e406b12e` (`fix(sdd): commit seguro, fixups del modo 3 y autostash en la doctrina Commit Review`), según el asunto de ese commit y el seguimiento del propio informe.
- JD-4, JD-5 y JD-6 (SUGGESTION): abiertos e informativos. JD-4: `git branch -r --contains` sin fetch puede usar refs remotas desactualizadas. JD-5: la orden de sustitución del rechazo omite `--reason` o `--evidence-kind`. JD-6: la frase «the only way to approve a gate is one of the exact invocations above» choca con la composición `--commit-review <token>`.

## Huecos de cobertura abiertos (informativos)

Fuente: `coverage-review.spec.md` y `coverage-review.design.md`, ronda 1. No bloquean el archivado.

- `coverage-review.spec.md`: 14 huecos; 10 abiertos (4 a 11, 13 y 14).
- `coverage-review.design.md`: 12 huecos; 10 abiertos (2, 3, 4, 5, 7, 8, 9, 10, 11 y 12).

## Commits del cambio

Commits en `main` (PR #289, merge `a0967497`), en orden cronológico:

- `92a11196` docs(openspec): planificar aviso-commit (C-02-d)
- `96657086` feat(sdd): registrar el modo de revisión por commit en la puerta tasks
- `20492a89` feat(sdd): proyectar commitReview en el estado de la puerta tasks
- `3f602ede` docs(sdd): sección Commit Review de fuente única en los orquestadores
- `0a9db236` docs(sdd): contrato commit_review de sdd-apply y fixups en work-unit-commits
- `c82307fa` feat(odd): avisar por commit y reglas comunes de commits en la doctrina ODD
- `986b8d2d` test(golden): regenerar goldens por la doctrina de revisión por commit
- `f1e11bab` docs(openspec): verificación y judgment-day de aviso-commit
- `e406b12e` fix(sdd): commit seguro, fixups del modo 3 y autostash en la doctrina Commit Review
- `c6c2bd4a` docs(openspec): registrar la corrección de JD-1 a JD-3 en aviso-commit
- `bc9bcada` fix(cli): nombrar la salida en los rechazos de --commit-review

Los hashes de `verify-report.md` (`d5389cb5..cdac5184`) son los del rango de la rama antes de la fusión; no se han reconciliado con los de `main`.

## Pull request

PR #289, `inc/aviso-commit` hacia `main`, fusionada con el commit `a0967497`.

## Comandos ejecutados y resultados

- `axiom sdd archive-check --cwd <wt> --change aviso-commit`: código 0. El cambio no es ISO y no tiene hallazgos.
- `axiom sdd archive-compose` (tres deltas sobre sus specs vivas, salida a temporal y `mv`): código 0 en las tres capacidades.
- `axiom archive sync --path <wt>`: código 0. Catálogo regenerado: 73 especificaciones, 624 requisitos activos, 2086 escenarios BDD.
- `git mv openspec/changes/aviso-commit openspec/changes/archive/2026-11-22-aviso-commit`: código 0. La carpeta origen ya no existe.
- `diff -r` entre la instantánea previa al movimiento y la carpeta archivada: salida vacía, código 0.
- `git -C <wt> status --short`: los renombres están en el índice; `openspec/INDEX.md` y las tres specs vivas aparecen como modificadas sin preparar. Nada se ha confirmado.
- `rg -c "REQ-33\." <wt>/openspec/specs`: `organic-agent-trigger-rules` 2, `sdd-orchestrator-assets` 8, `sdd-block-review-gates` 8.

## Estado de Git

- Movimiento de la carpeta del cambio hecho con `git mv`.
- Cambios sin confirmar en la rama de archivado `inc/aviso-commit-archive`: el `git mv`, las tres specs vivas y `openspec/INDEX.md`. Confirmar y publicar corresponde a la persona, según la política del repositorio.
- Sin push, sin fetch y sin PR.

## Trazabilidad

- Artefactos leídos: los locadores de ruta de `axiom sdd status` (almacén `openspec`). No se leyó ninguna observación de Engram de este cambio, así que no hay identificadores de observación que registrar.
- Observación de Engram de este informe: `sdd/aviso-commit/archive-report` (proyecto `axiom`).

## Anomalías observadas

- Tras el archivado, `axiom sdd status aviso-commit` resuelve el cambio por su nombre contra Engram (`artifactStore: engram`, 0 tareas leídas) y devuelve `blocked`. Es un efecto de la búsqueda posterior al movimiento, no un estado real del cambio. `axiom sdd status --cwd <wt>` ya no lista `aviso-commit` como cambio activo y propone `e2e-cumulative`.
- `openspec/INDEX.md` se regenera completo: además de las filas de los tres dominios, cambian la fecha de sincronización y los totales.
