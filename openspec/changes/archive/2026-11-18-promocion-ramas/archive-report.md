# Reporte de Archivado Formal: promocion-ramas

## Resumen de Cierre
- Fecha de Archivado: 2026-11-18
- Estado de Implementación: 42 de 42 tareas marcadas en `tasks.md` (fases 1.1-5.7), según el estado nativo de SDD en el momento del archivado.
- Estado de Verificación: el `verify-report.md` fue escrito en la fase de verificación y registra 0 CRITICAL, 2 WARNING y 2 SUGGESTION. Es una instantánea intermedia. Según el orquestador (hechos de estado final), los WARNING W1 y W2 se resolvieron después con el commit `e806d39d`, y los SUGGESTION siguen abiertos. El archivado no volvió a ejecutar la verificación funcional del cambio.
- Integración: fusionado en `main` mediante el PR #277 (merge commit `148f0588`).
- Destino en Archivo: `openspec/changes/archive/2026-11-18-promocion-ramas/`
- Carril y kickoff: `sdd`, `executionStyle: continuous`, `handoffPolicy: none`, rol `fullstack`. No hay gates de bloque registrados, por lo que la precondición de integración no aplica (`dependencies.archive: ready`).

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento sin requisitos eliminados ni renombrados: la delta no contiene secciones `REMOVED` ni `RENAMED`, por lo que no se usó `--supersede`.

Requisitos modificados por sustitución del bloque completo (no son deprecados; el texto anterior queda en la historia de git y en la anotación `(Previously: ...)` de cada bloque):

| Dominio | Requisito | Estado | Nota |
|---------|-----------|--------|------|
| change-branch-lifecycle | REQ-25.3, 25.8, 25.10, 25.14, 25.15, 25.16, 25.17, 25.24, 25.45 | MODIFIED | Bloque completo sustituido por la delta |
| workspace-topology | REQ-25.26, 25.27 | MODIFIED | Bloque completo sustituido por la delta |
| sdd-lane-kickoff | REQ-25.36 | MODIFIED | Bloque completo sustituido por la delta |

## Requisitos Añadidos
- `change-branch-lifecycle`: REQ-25.46 (modo de promoción por repositorio y por cambio), REQ-25.47 (`ensure`: base efectiva, `ask` y `--promotion`), REQ-25.48 (`status`: siguiente PR hacia la rama principal), REQ-25.49 (`done`: destino de la confirmación según el modo), REQ-25.50 (contratos de integración y publicación que no cambian con el modo).

## Especificaciones Vivas Integradas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| change-branch-lifecycle | Actualizada | Composición nativa con `axiom sdd archive-compose` (5 requisitos ADDED, 9 MODIFIED). Después, ediciones acotadas a mano en las tablas «Términos» (filas «Rama principal» y «Modo de promoción») y «Códigos tipados» (`decision_required` y `plan_conflict` ampliados), según la nota de integración de la cabecera de la delta. |
| workspace-topology | Actualizada | Composición nativa (2 requisitos MODIFIED). |
| sdd-lane-kickoff | Actualizada | Composición nativa (1 requisito MODIFIED). |

Catálogo maestro: `axiom archive sync` regeneró `openspec/INDEX.md` (71 especificaciones consolidadas, 568 requisitos activos, 1819 escenarios BDD).

## Artefactos Consolidados
- `proposal.md` (presente)
- `specs/change-branch-lifecycle/spec.md`, `specs/workspace-topology/spec.md`, `specs/sdd-lane-kickoff/spec.md` (presentes; deltas)
- `design.md` (presente)
- `tasks.md` (presente; 42 completadas, 0 pendientes; bytes intactos)
- `verify-report.md` (presente; instantánea intermedia, ver Hallazgos)
- Adicionales presentes en la carpeta: `explore.md`, `kickoff.yaml`
- `apply-progress.md`: ausente (el localizador devuelve `<unresolved>`; no es bloqueante para el archivado)

## Trazabilidad de Observaciones de Engram
- Ninguna observación de Engram fue leída en esta fase. Los artefactos se leyeron desde sus rutas de `openspec/changes/promocion-ramas/`, que es el almacén que informó el estado nativo (`artifactStore: openspec`).
- Discrepancia registrada: la orden de lanzamiento indicaba store `hybrid`; el estado nativo reporta `openspec`. Se siguió el estado nativo (`sdd-phase-common.md`, sección B).
- Copia en Engram de este reporte: `sdd/promocion-ramas/archive-report`.

## Verificación de Archivado (comandos observados)
| Comando | Resultado |
|---------|-----------|
| `axiom sdd status promocion-ramas --cwd <wt> --json --instructions` | exit 0; `archive: ready`, `taskProgress` 42/42, sin `blockedReasons` |
| `axiom sdd archive-check --cwd <wt> --change promocion-ramas --json` | exit 0; `iso: false`, `ok: true`, sin findings |
| `axiom sdd archive-compose` (3 deltas) | exit 0 en cada dominio; escritura por `.compose-tmp` y `mv` |
| `axiom archive sync --path <wt>` | exit 0; `openspec/INDEX.md` regenerado |
| `axiom sdd integrate verify --cwd <wt>` | exit 1: el comando ya no existe («se retiró con la baseline ajena»). No se escribió nada. La skill no define otro verificador para este caso. |
| Contexto técnico mantenido | No proyectado por el estado nativo; `axiom archive context verify` no aplica |
| `go test ./internal/workspace/ -run "Docs|Spec|OpenSpec" -count=1` | ok |
| `go test ./internal/livespec/ ./internal/livingdoc/ -count=1` | ok |

Readback mecánico `diff -r` (copia previa al movimiento frente al destino): salida vacía, exit 0. El `archive-report.md` es aditivo y queda fuera de la comparación.

## Hallazgos y Contradicciones
- SUGGESTION S1 (tests de `publish`, `handoff`, archivado con `main_branch` y `done` multirepo): en el momento de la verificación estaba abierto. El commit `e806d39d` añade tests con ese alcance y el `verify-report.md` lo registra como remediación de W1 y W2. Según el orquestador, S1 sigue como SUGGESTION sin cerrar; no se ha confirmado de forma independiente que esté cubierto por completo.
- SUGGESTION S2 (tiempo de espera ambiental de `reviewtransaction` en `main`): abierto. No guarda relación con el cambio; conviene aislarlo aparte.
- Contradicción entre fuentes, resuelta por jerarquía de Final-State Authority:
  - `verify-report.md` (escrito en la verificación): W1 y W2 abiertos.
  - Hechos de estado final del orquestador y remediación registrada en el propio `verify-report.md`: W1 y W2 resueltos en `e806d39d`.
  - Se reporta el estado final (resueltos), citando `e806d39d`. El `verify-report.md` no se modificó.
- Tareas: 42 de 42 marcadas en `tasks.md`; commits de implementación según el orquestador: `c29db67b`, `2232e291`, `1c4d8886`, `92869ef6`, `81e0a74a`, `001da130`, `e806d39d`.
- Ninguna tarea pendiente ni check no ejecutado que bloquee el archivado.

## Notas de Integración
- Los requisitos `(Previously: ...)` de las deltas quedan en las specs vivas como anotación histórica, según la delta.
- La copia de `diff -r` de la sección de verificación es la evidencia de byte-identidad de la carpeta archivada.
