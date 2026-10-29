# Reporte de Archivado Formal: inc-25-ramas-y-worktrees-por-topologia

## Resumen de Cierre
- Fecha de Archivado: 2026-10-29
- Estado de Verificación: PASS con avisos, según `verify-report.md` (instantánea del 2026-10-29: 0 críticos, 3 avisos, 4 sugerencias). Al cierre, el aviso A2 queda resuelto y el S1 queda ejecutado; los demás siguen abiertos (ver «Seguimientos Abiertos»).
- Estado de Implementación: 214/214 tareas marcadas en `tasks.md` del archivo; 0 pendientes. La tarea S12.26 (tarea de archivo) la ejecutó `sdd-archive` según la decisión T10.
- Integración: PR #161 fusionado en `main` con el commit de merge `99ee35a1`. Compuerta `integration` aprobada con evidencia `pr_merged`. Compuerta `role-apply:fullstack` aprobada en `ef165c97`.
- Rama de archivado: `inc/inc-25-ramas-y-worktrees-por-topologia-archive` (decisiones D16 y D18). Sin push ni PR.
- Baseline del proyecto: ninguna declarada. `technical_context.maintained`: no declarado.
- Destino en Archivo: `openspec/changes/archive/2026-10-29-inc-25-ramas-y-worktrees-por-topologia/`

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| workspace-topology | Topología Monorepo Desacoplado (REQ-2.3) | [SUPERSEDED / DEPRECADO] | Topología retirada. `workspace.topology: monorepo-decoupled` se rechaza con un error accionable que indica `multirepo` (REQ-25.25). Sin alias ni migración automática. | `workspace-topology/spec.md`, REQ-25.25 |

Notas:
- Se usó `--supersede` en la composición de `workspace-topology`, porque su sección REMOVED deja obsoleto REQ-2.3. No se usó `--superseded-requirements`.
- REQ-24.3 (planning-home-resolution) no queda superado: se renombra (RENAMED) y se modifica para `multirepo`, manteniendo su identificador.

## Dominios Integrados
Composición nativa (`axiom sdd archive-compose`, salida 0 en los nueve dominios con especificación viva):

| Dominio | Acción | Secciones del delta | Requisitos antes → después |
|---------|--------|---------------------|----------------------------|
| axiom-sdd-cli-integration | Actualizado | ADDED | 4 → 7 |
| local-web-dashboard | Actualizado | MODIFIED, ADDED | 8 → 11 |
| multi-project-hub | Actualizado | MODIFIED, ADDED | 6 → 9 |
| organic-agent-trigger-rules | Actualizado | ADDED | 14 → 16 |
| planning-home-resolution | Actualizado | MODIFIED, RENAMED, ADDED; después, edición manual S12.26 | 12 → 13 |
| sdd-archive-lifecycle | Actualizado | MODIFIED, ADDED | 6 → 7 |
| sdd-lane-kickoff | Actualizado | ADDED | 4 → 8 |
| spec-baseline-declaration | Actualizado | MODIFIED | 15 → 15 |
| workspace-topology | Actualizado | MODIFIED, REMOVED (con `--supersede`), ADDED | 7 → 10 |
| change-branch-lifecycle | Nuevo (copia mecánica con `cp` y `diff -r` vacío) | — | 0 → 25 |

Tras la integración, `axiom archive sync` regeneró `openspec/INDEX.md`: 64 especificaciones consolidadas, 478 requisitos activos y 1176 escenarios BDD.

## Edición Manual (S12.26)
- `openspec/specs/planning-home-resolution/spec.md`, tabla «Términos», fila «Repositorio de especificaciones»: «`monorepo-decoupled` y `multirepo`» sustituido por «`multirepo`». Hecho.
- Misma especificación, tabla «Requerimientos de un vistazo», fila REQ-24.3: sustituido igual. Hecho. La concordancia del verbo («`multirepo` re-enraízan») queda tal cual, porque la tarea pide la sustitución literal.
- Encabezado del documento: no contiene la cadena «`monorepo-decoupled` y `multirepo`»; no hubo nada que sustituir. El título del REQ-24.3 lo renombra el RENAMED del delta.
- `workspace-topology` y `spec-baseline-declaration`: no requieren edición a mano. La tabla de REQ-24.13 no menciona topologías.
- Comprobación posterior a la fusión: las menciones restantes de `monorepo-decoupled` en `openspec/specs/**` son notas «(Previously: ...)», el requisito REQ-25.25, escenarios de rechazo, el bloque `[SUPERSEDED / DEPRECADO]` de REQ-2.3 y referencias a su retirada. Ninguna la presenta como valor válido.

## Artefactos Consolidados
Presentes en la carpeta archivada (observado con `ls` y con el `diff -r` de la sección de verificación):
- `proposal.md`, `design.md`, `tasks.md`, `verify-report.md`
- `specs/`: 10 deltas (`axiom-sdd-cli-integration`, `change-branch-lifecycle`, `local-web-dashboard`, `multi-project-hub`, `organic-agent-trigger-rules`, `planning-home-resolution`, `sdd-archive-lifecycle`, `sdd-lane-kickoff`, `spec-baseline-declaration`, `workspace-topology`)
- `kickoff.yaml`, `gates.yaml`, `handoff.md`, `exploration.md`, `spike-guard-runtimes.md`
- Este `archive-report.md`, añadido después del `diff -r`

Faltante: `apply-progress` (el status lo reporta como `missing`). No existe `spec.md` de nivel superior: el cambio se expresa como deltas por dominio. `context-impact.md` no aplica, porque no hay `technical_context.maintained`.

## Hechos del Estado Final
Fuentes, por orden de prelación (`sdd-archive`, Final-State Authority):
1. `tasks.md` archivado: 214 casillas `[x]` y 0 `[ ]`. La única modificación de ese fichero en el archivado es la casilla de S12.26 (`- [ ]` a `- [x]`), tras ejecutar la tarea. Ningún otro byte cambió.
2. Hechos del encargo de archivado: PR #161 fusionado (`99ee35a1`); compuertas `role-apply:fullstack` (`ef165c97`) e `integration` (`pr_merged`, `99ee35a1`) aprobadas. Estado del status antes del archivado: `archive: ready`, `blockedReasons: []`, sin notas.
3. `verify-report.md` (instantánea del 2026-10-29 sobre `99ee35a1` más `4f606065`): usado solo como historial. Su cifra de tareas (213/214) y su aviso A2 se citan como historial de ese momento, no como estado actual.

Estado de los hallazgos de `verify-report.md` al cierre:
- A1 (cobertura no medida): abierto.
- A2 (specs vivas con `monorepo-decoupled`): resuelto al archivar, por la composición y por S12.26.
- A3 (suite local parcial): abierto. La evidencia de suite completa sigue siendo la CI de #161 que cita el informe; no se ha reejecutado.
- S1 (S12.26 pendiente): resuelto.
- S2 (criterio «Invariante» de la propuesta): abierto; ver «Contradicciones Registradas».
- S3 (redacción de REQ-25.36): abierto; ver «Contradicciones Registradas».
- S4 (`strict_tdd: false`, sin auditoría de TDD): abierto.

Cifras finales: 214/214 tareas (`tasks.md` archivado); 64 especificaciones, 478 requisitos activos y 1176 escenarios (`axiom archive sync`). No se han copiado cifras de `verify-report.md` salvo como historial.

## Contradicciones Registradas
1. REQ-25.36, sellado único de `kickoff.yaml` (`sdd-lane-kickoff`, texto normativo y escenario «segundo seal»): la especificación dice que un segundo `seal` «se rechaza como cambio ya sellado». El informe de verificación (S3, instantánea del 2026-10-29) indica que la implementación devuelve al ganador sin reescribir, y que solo la guarda A lo rechaza desde otra rama. Ninguna fuente de rango superior lo resuelve. Se deja la redacción tal cual y no se inventa ningún requisito. Queda como seguimiento.
2. Criterio 11 de `proposal.md` §12 («Invariante», `axiom branch check`) frente a REQ-25.1 (cuatro verbos, sin `check`) y REQ-25.20 (pendientes informativos). Prevalece la especificación reabierta el 2026-10-17, que es el estado vigente. La propuesta no se ha reescrito, porque el archivo no modifica documentos históricos. Queda como seguimiento S2.
3. Almacén de artefactos: el status reporta `artifactStore: openspec`; el preflight del orquestador indica `hybrid`. Se persiste el reporte en el fichero del archivo y en Engram, para cubrir ambas declaraciones.
4. Cifra de tareas: 213/214 en `verify-report.md` frente a 214/214 en `tasks.md`. Resuelta por prelación: `tasks.md` y el encargo de archivado (S12.26 ejecutada).

## Trazabilidad
- Observaciones de Engram leídas en esta fase: ninguna. Los artefactos se leyeron desde los ficheros indicados por el status.
- Observaciones citadas por los artefactos del cambio: ninguna identificada.
- Persistencia de este reporte: topic key `sdd/inc-25-ramas-y-worktrees-por-topologia/archive-report`, proyecto `axiom`, tipo `architecture`.

## Verificación del Archivado
- `axiom sdd status` (ejecutado con el worktree como directorio de proceso, antes de mover nada): `archive: ready`, `blockedReasons: []`, compuertas `spec`, `design`, `tasks` e `integration` aprobadas.
- `axiom sdd archive-compose` sobre los nueve dominios vivos: salida 0 en todos. Solo `workspace-topology` con `--supersede`. Cada composición escribió a `.compose-tmp` y se movió solo tras salida 0.
- Copia mecánica de `change-branch-lifecycle` con `cp` y `diff -r`: sin diferencias.
- `axiom archive sync --path <workspaceRoot>`: salida 0; catálogo regenerado.
- `git mv` de la carpeta del cambio al archivo y `diff -r` contra la instantánea previa al movimiento: sin diferencias. La instantánea se tomó después de la edición de S12.26.
- `axiom workspace validate --path <workspaceRoot>`: `COMPLIANT`, salida 0. Esta versión no acepta `--cwd`; se usó `--path`, como en el archivo anterior.
- `go test -count=1 ./internal/workspace`: ok (34,7 s). `go test -count=1 ./internal/assets`: ok (23,0 s).
- `axiom sdd integrate verify`: no aplica, porque el cambio no tiene baseline del proyecto ni contexto técnico mantenido.

## Seguimientos Abiertos
No corregidos en este archivado. Se registran como estaban en el informe de verificación o en el encargo de cierre.
- A1: cobertura no medida (`go test -cover` sin ejecutar, sin umbral declarado).
- A3: suite local parcial. Pendiente de la CI de #161 citada; `internal/cli` completo y paquetes con fallo de entorno conocido no se han ejecutado localmente.
- S2: criterio «Invariante» de `proposal.md` §12 obsoleto por D17. El texto de la propuesta no se actualiza en el archivo.
- S3: redacción de REQ-25.36 frente al comportamiento implementado. Ver «Contradicciones Registradas».
- S4: `strict_tdd: false` en `openspec/config.yaml`; no se ha auditado evidencia de TDD.
- Redacción de REQ-24.3 en las tablas: la concordancia plural de «re-enraízan» queda sin ajustar, por la sustitución literal de S12.26.
- Entrega: PR documental aparte pendiente. Push, PR y fusión son decisiones del usuario bajo la política ordinaria del repositorio.
- `inc-23-knowledge-profile` queda fuera de este plan y sin archivar.
