# Reporte de Archivado Formal: eventos-ado

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: el `verify-report` archivado registra un WARNING, corregido después en `686cc3a45` (ver Hallazgos). Verificación contra Azure DevOps real no ejecutada.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-eventos-ado/`
- Código fusionado: PR #366, merge commit `fa076cc3f`, CI en verde.
- Kickoff: `continuous`, rol `fullstack`. Sin gates de revisión.
- Tracker: el repositorio no tiene tracker activo (`integrations:` ausente en `axiom.yaml`); no hay `.tracker-notes.yaml`, ni `summary push`, ni seguimiento en tracker.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento sin `REMOVED` ni `RENAMED`: las deltas solo contienen `MODIFIED` y `ADDED`.

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| azure-devops-adapter | Creación de una Task Hija con Marca Determinista y sin Duplicados | prov:ADO:1 | RF-ADO-20 | assigned |
| azure-devops-adapter | Validación de la Task Vinculada a Mano | prov:ADO:2 | RF-ADO-21 | assigned |
| azure-devops-adapter | Lectura y Escritura de Horas con `test /rev` | prov:ADO:3 | RF-ADO-22 | assigned |
| azure-devops-adapter | Sustitución Idempotente de `System.Description` | prov:ADO:4 | RF-ADO-23 | assigned |
| azure-devops-adapter | Etiquetas de IDs Funcionales como Conjunto Idempotente | prov:ADO:5 | RF-ADO-24 | assigned |
| azure-devops-adapter | Estado y Asignación de la Task por Evento | prov:ADO:6 | RF-ADO-25 | assigned |
| azure-devops-adapter | Transición del Work Item al Archivar | prov:ADO:7 | RF-ADO-26 | assigned |
| azure-devops-adapter | Las Escrituras Nuevas Son Herméticas y No Filtran el PAT | prov:ADO:8 | RF-ADO-27 | assigned |
| change-branch-lifecycle | `ensure --role` Registra el Inicio del Rol la Primera Vez que Queda Listo | prov:BRA:1 | RF-BRA-41 | assigned |
| integration-plugins | Puerto Opcional de Tasks por Rol | prov:INT:1 | RF-INT-25 | assigned |
| integration-plugins | Verbo `axiom integrations tasks create` | prov:INT:2 | RF-INT-26 | assigned |
| integration-plugins | Verbo `axiom integrations tasks bind` | prov:INT:3 | RF-INT-27 | assigned |
| integration-plugins | Verbo `axiom integrations summary push` | prov:INT:4 | RF-INT-28 | assigned |
| integration-plugins | Renderizador Determinista del Resumen de la Spec | prov:INT:5 | RF-INT-29 | assigned |
| integration-plugins | Estimación por Rol en `design.md` | prov:INT:6 | RF-INT-30 | assigned |
| integration-plugins | Verbo `axiom integrations time propose` | prov:INT:7 | RF-INT-31 | assigned |
| integration-plugins | Verbo `axiom integrations time record` | prov:INT:8 | RF-INT-32 | assigned |
| integration-plugins | Precondiciones por Estilo, Carril ODD y Plugin Activo de los Verbos Nuevos | prov:INT:9 | RF-INT-33 | assigned |
| integration-plugins | Salida JSON Versionada y Sin Secretos de los Verbos Nuevos | prov:INT:10 | RF-INT-34 | assigned |
| sdd-archive-lifecycle | Con Tracker Activo el Archivado Reenvía el Resumen con los IDs Definitivos | prov:ARC:1 | RF-ARC-16 | assigned |
| sdd-archive-lifecycle | Resumen Final de `continuous` con Estimaciones, Horas, Quién y Cuándo | prov:ARC:2 | RF-ARC-17 | assigned |
| sdd-orchestrator-assets | El Orquestador Invoca los Verbos de Tasks y de Resumen con Tracker Activo | prov:ORQ:1 | RF-ORQ-29 | assigned |
| sdd-orchestrator-assets | `ensure --role` Antes del Apply con Tracker Activo | prov:ORQ:2 | RF-ORQ-30 | assigned |
| sdd-orchestrator-assets | `sdd-design` Propone la Estimación por Rol | prov:ORQ:3 | RF-ORQ-31 | assigned |
| sdd-orchestrator-assets | `sdd-apply` Propone e Imputa las Horas al Cerrar el Rol | prov:ORQ:4 | RF-ORQ-32 | assigned |
| sdd-orchestrator-assets | Doctrina de los Verbos de Integración en Paridad entre Runtimes y Goldens | prov:ORQ:5 | RF-ORQ-33 | assigned |

Los 26 identificadores provisionales se asignaron sin duplicados (`duplicates[]` vacío). El verbo reescribió los provisionales en las deltas, en las `Cubre:` de `tasks.md` y en `ids.yaml`.

## Requisitos Modificados
RF-ADO-01, RF-ADO-02, RF-ADO-14 (azure-devops-adapter); RF-BRA-33 (change-branch-lifecycle); RF-INT-09, RF-INT-18 (integration-plugins).

## Trazabilidad de la Integración
- `axiom sdd archive-check --change eventos-ado --json`: `ok: true`, sin hallazgos, antes de integrar.
- `axiom sdd archive-integrate --change eventos-ado --check --json`: exit 0, nada escrito.
- `axiom sdd archive-integrate --change eventos-ado --json`: exit 0. Resultado por capacidad: `composed` en las cinco (azure-devops-adapter, change-branch-lifecycle, integration-plugins, sdd-archive-lifecycle, sdd-orchestrator-assets).
- Escritos (`written[]`): `openspec/specs/` de las cinco capacidades; deltas de las cinco; `tasks.md`; `ids.yaml`.
- `axiom archive sync --path <worktree>`: exit 0 dos veces (antes y después del traslado). 50 dominios, 602 requisitos activos, 2920 escenarios BDD. `openspec/INDEX.md` regenerado.
- `go test -count=1 ./internal/idmigration/`: `ok`, exit 0.
- `axiom sdd archive-check` tras el traslado: exit 1 con `ya esta archivado` (RF-ARC-03). Es la señal de cierre esperada.

## Pendiente: `## Purpose` de azure-devops-adapter
El `## Purpose` de `openspec/specs/azure-devops-adapter/spec.md` sigue diciendo que el adaptador «no emite ningún evento del ciclo de vida hacia Azure DevOps» y «no crea work items». Esto ya no es cierto tras el cambio. La delta lo advierte: una delta no puede modificar ese párrafo. El archivado no lo reescribió porque la skill reserva la escritura de `openspec/specs/` al verbo nativo. Queda pendiente una corrección manual aprobada por la persona, por ejemplo en un incremento aparte.

## Artefactos Consolidados
- `proposal.md`, `design.md`, `exploration.md`, `kickoff.yaml`, `coverage-review.spec.md`, `coverage-review.design.md`, `verify-report.md`: presentes, archivados sin cambios.
- `specs/` (5 deltas): presentes; los provisionales están reescritos por el verbo.
- `tasks.md`: presente; 97 de 97 casillas completadas, 0 pendientes. Las `Cubre:` están reescritas con los identificadores definitivos.
- `ids.yaml`: presente, generado por el verbo.

## Estado Final y Hallazgos
- Tareas: 97 de 97 completadas (conteo sobre el `tasks.md` archivado).
- Verify: el WARNING del verify (trinquete `TestEveryNamedReviewContinuationIsStructurallyReal` citado en tasks, inexistente) quedó corregido en `686cc3a45` con una nota en `tasks.md`. Dos SUGGESTION siguen abiertas como informativas. Fuente: encargo de archivado y commit `686cc3a45`.
- Cobertura: 22 huecos de la spec y 16 del diseño, todos resueltos en spec o diseño. Detalle en `coverage-review.spec.md` y `coverage-review.design.md`.
- Sin verificar contra Azure DevOps real: HTML que conserva `System.Description`, relación padre en el mismo POST, `CONTAINS` de WIQL sobre `System.Tags`, nombres `Microsoft.VSTS.Scheduling.*`, estado `Tech Review`, asignación por nombre visible. Las pruebas usan el doble `httptest`/`adotest`.

## Trazabilidad Engram
No se leyó ningún observation de Engram: el modo de almacenamiento es `openspec`, y los artefactos se leyeron desde ficheros.
