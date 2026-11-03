# Reporte de Archivado Formal: paradas-tres-salidas

## Resumen de Cierre
- Fecha de Archivado: 2026-11-03
- Estado de Verificación: `pass_with_warnings` según `verify-report.md` (sin hallazgos CRITICAL). Estado al cierre de los avisos en la sección «Hallazgos y estado final».
- Precondición de archivo: puerta `integration` aprobada con `evidence_kind: pr_merged`, `evidence_ref: 34b567b275ee5cc2956f086d6f45b9d71920afc0`, base `main`, `verified: true`. Registro en el commit `4f1d3d40`.
- Tareas: 24 de 24 completadas en `tasks.md` (observado al cierre); 0 pendientes.
- Rama de archivado: `inc/paradas-tres-salidas-archive`.
- Destino en Archivo: `openspec/changes/archive/2026-11-03-paradas-tres-salidas/`

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| sdd-role-closure-handoff | Generación del `handoff.md` de Integración (REQ-21.14) | [SUPERSEDED] | El usuario decidió (Engram #1811) no generar el relevo como artefacto del flujo SDD. El relevo entre personas lo cubren las tres salidas de las paradas. Se marca con `--supersede`, de modo que el requisito se conserva con su motivo. | `sdd-block-review-gates` REQ-26.1; `sdd-role-closure-handoff` REQ-21.15 |

Requisitos renombrados con el ID conservado (no son superaciones):
- `sdd-preflight-configuration`: REQ-21.5, «Bloqueo de Avance sin Modalidad Sellada».
- `sdd-role-closure-handoff`: REQ-21.15, «Ejecución de la Fase `verify` Global tras Aprobarse Todos los `role-apply`».

## Especificaciones Vivas Modificadas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| sdd-block-review-gates | Actualizada | ADDED REQ-26.1 a REQ-26.8; MODIFIED REQ-21.8 a REQ-21.12. |
| sdd-role-closure-handoff | Actualizada | MODIFIED REQ-21.13 y REQ-21.15; REMOVED REQ-21.14 (marcado SUPERSEDED); RENAMED REQ-21.15; propósito corregido. |
| sdd-preflight-configuration | Actualizada | ADDED REQ-26.9; MODIFIED REQ-21.5; RENAMED REQ-21.5; propósito corregido (se retira la política de relevos). |
| sdd-lane-kickoff | Actualizada | MODIFIED REQ-21.2 y REQ-21.4. |
| organic-agent-trigger-rules | Actualizada | ADDED REQ-26.10; MODIFIED REQ-25.40. |
| change-branch-lifecycle | Actualizada | ADDED REQ-26.11; MODIFIED REQ-25.20 y REQ-25.45. Glosario «Cambio pendiente de archivar», fila REQ-25.20 del índice y supuesto S-16 enmendados; supuestos A-7 y A-8 añadidos a la tabla de supuestos. |

Índice maestro: `openspec/INDEX.md` regenerado con `axiom archive sync` (67 especificaciones, 505 requisitos activos, 1285 escenarios).

Pendiente fuera de este archivado: el nombre del directorio de la capacidad `sdd-role-closure-handoff` sigue reflejando el relevo de integración. La nota no normativa del delta deja ese renombrado fuera de este cambio; el «Propósito» sí queda corregido.

## Artefactos Consolidados
- `proposal.md`: presente.
- `specs/`: seis deltas (`change-branch-lifecycle`, `organic-agent-trigger-rules`, `sdd-block-review-gates`, `sdd-lane-kickoff`, `sdd-preflight-configuration`, `sdd-role-closure-handoff`).
- `design.md`: presente.
- `tasks.md`: presente, 24 de 24 completadas.
- `verify-report.md`: presente.
- `explore.md`, `kickoff.yaml`, `gates.yaml`: presentes.
- `.gates.lock`: fichero de cerrojo vacío (0 bytes), sin seguimiento en git. Se eliminó tras el movimiento porque es residuo de ejecución y no un artefacto. Ningún otro fichero difiere del original.

## Hallazgos y estado final
Estado final según la jerarquía de `sdd-archive`: las fuentes de mayor rango prevalecen sobre las instantáneas de verificación.

- **W-01** (`verify-report.md`, en el momento de la verificación): la fase cedida de un `role-apply` anterior se proyectaba contra el roster actual. **Estado final: corregido** en el commit `3c9936b9` (`fix(kickoff): persistir la fase cedida al registrar la salida 2`), recogido en la sección «Corrección posterior a la verificación» del informe.
- **W-02** (`verify-report.md`, en el momento de la verificación): el ratchet largo de `internal/cli` y `scripts/deadcode-ratchet.sh` no se reobservaron en local por restricción del entorno (EDR). **Estado: abierto como evidencia local.** Según el orquestador, la CI de #218 pasó Unit Tests y Go Format; este archivado no lo ha verificado.
- **S-01**: no aplicado. El propio informe lo registra como equivalente y sin cambio necesario.
- **S-02**: historial archivado en `docs/archive/ROADMAP.md:313`. No se modifica.
- **S-03**: resuelto en este archivado (notas no normativas de las deltas aplicadas, ver «Especificaciones Vivas Modificadas»). La prueba propia de la exclusión de intermedias en el dashboard no existe; la cubre la prueba del motor y la suite `dashboard`, en verde en el momento de la verificación.
- **R-5** (`design.md`, riesgo Medio, aceptado como limitación conocida): quien recoge un cambio multirrol en otra máquina ejecuta `ensure`, que crea la rama de seguimiento desde la base y no desde `origin/<seguimiento>`. No bloquea el archivado. Queda como punto abierto 2 del diseño.

## Verificación al archivar
- `go test ./internal/workspace/ ./internal/assets/ -count=1`: `ok` en ambos paquetes (salida 0), ejecutado en el worktree de archivado.
- `axiom sdd status`: `archive: ready` antes del movimiento; `dependencies.archive` sin bloqueos.
- Readback `diff -r`: sin diferencias salvo la eliminación del `.gates.lock` descrita arriba (salida verbatim en la respuesta de la fase de archivado).

## Trazabilidad
- Almacén declarado por el estado nativo: `openspec`. El lanzamiento indicaba `hybrid`; el informe se persiste en fichero y, además, en Engram (`sdd/paradas-tres-salidas/archive-report`). No se leyó ningún observation ID de Engram como fuente de artefactos: los artefactos se leyeron por ruta.
- Referencia de procedimiento: commit `56eafbcd`.
- Referencia de decisión de producto: Engram #1811 (`product/paradas-tres-salidas`), citada en el delta de REQ-21.14.
