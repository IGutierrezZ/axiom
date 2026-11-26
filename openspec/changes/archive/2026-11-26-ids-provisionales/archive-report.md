# Reporte de Archivado Formal: ids-provisionales

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: Aprobado con observaciones. El `verify-report` del cambio es intermedio y está limpio con 1 WARNING (RED de los pins 8.1 no observado) y 1 SUGGESTION (`## Purpose`). Ver «Hechos finales».
- Tareas: 61/61 completadas (`tasks.md`, observado al archivar); 0 pendientes.
- Código: PR #326 fusionada con merge commit `fb2c97cd`. La CI falló primero en `Go Tests (cli)` (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`: dos mensajes de uso de `archive-integrate` sin resolución nombrada); corregido en `2b008245`; CI verde (14 pass, 5 skipping).
- Destino en Archivo: `openspec/changes/archive/2026-11-26-ids-provisionales/`
- Modo de archivado: `hybrid` con `openspec/` como almacén de ficheros. Flujo clásico (`archive-check`, `archive-compose`, `archive sync`); el binario del PATH no incluye `axiom sdd archive-integrate`.

## Hechos finales (prevalecen sobre los artefactos intermedios)
- Fuente de mayor rango: el encargo de archivado (2026-11-26). Las afirmaciones de `verify-report` (WARNING y SUGGESTION) son de su momento de escritura; el WARNING del RED de los pins 8.1 no se ha reverificado en este cierre.
- El `## Purpose` de `openspec/specs/requirement-identifiers/spec.md` se corrigió a mano tras componer, en una sola frase, para reflejar la capacidad de asignación provisional (REQ-47.1–47.6). REQ-44.3 no se modificó.
- `REQ-47` estaba libre en `openspec/specs/` antes de componer (`rg 'REQ-47\.' openspec/specs` sin coincidencias).
- Los identificadores definitivos REQ-47.1 a REQ-47.10 aparecen en las seis deltas; no hay provisionales pendientes.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. No se usó `--supersede` ni `--superseded-requirements`. Las seis deltas contienen operaciones ADDED y MODIFIED (sin REMOVED ni RENAMED); los MODIFIED reemplazan el cuerpo del requisito en su sitio y no lo marcan como `[SUPERSEDED / DEPRECADO]`.

## Especificaciones Sincronizadas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| requirement-identifiers | Compuesta | ADDED (1 sección), más el `## Purpose` corregido a mano |
| odd-living-spec-integration | Compuesta | ADDED (1), MODIFIED (1) |
| sdd-archive-lifecycle | Compuesta | ADDED (1), MODIFIED (1) |
| sdd-iso-evidence | Compuesta | MODIFIED (1) |
| axiom-sdd-cli-integration | Compuesta | ADDED (1) |
| organic-agent-trigger-rules | Compuesta | MODIFIED (1) |

Catálogo maestro: `openspec/INDEX.md` regenerado con `axiom archive sync` (53 especificaciones, 538 requisitos, 2262 escenarios).

## Artefactos Consolidados
- `proposal.md`, `specs/` (6 dominios), `design.md`, `tasks.md`, `verify-report.md`, `explore.md`, `kickoff.yaml`, `coverage-review.spec.md`, `coverage-review.design.md`.
- Fuentes leídas como ficheros (almacén `hybrid`): `openspec/changes/ids-provisionales/` en el worktree de archivado. No se recuperaron observaciones de Engram para `sdd/ids-provisionales/*` en este cierre (el encargo no permitió resolver el proyecto con `mem_current_project`, que no está disponible en esta sesión); la trazabilidad por observación queda pendiente.

## Riesgos y pendientes
- Pendiente: el RED de los pins 8.1 no quedó observado (WARNING del verify).
- Sugerencia sin resolver: `## Purpose` de la especificación del cambio.
- La rama de archivado `inc/ids-provisionales-archive` no está publicada ni integrada; la publicación y la fusión son decisiones del orquestador.
