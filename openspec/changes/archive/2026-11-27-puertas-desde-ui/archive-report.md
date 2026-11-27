# Reporte de Archivado Formal: puertas-desde-ui

## Resumen de Cierre
- Fecha de Archivado: 2026-11-27
- Estado de Verificación: `verify-report.md` con `**Estado:** limpio` (0 CRITICAL, 0 WARNING, 2 SUGGESTION), según el informe en el momento de su escritura. Fuente de mayor rango: el estado final del cambio, con 22 de 22 tareas completadas y las 8 unidades de trabajo hechas.
- Destino en Archivo: `openspec/changes/archive/2026-11-27-puertas-desde-ui/`
- Trazabilidad de Engram: el almacén nativo del cambio es `openspec`; no se leyeron observaciones de Engram como fuente de los artefactos.
- Fusión: `openspec/specs/local-web-dashboard/spec.md` y `openspec/specs/sdd-block-review-gates/spec.md`, mediante `axiom sdd archive-integrate`.
- Commits del cambio (tras el autosquash): fea57c1ec, 889ec81fc, 7ff1cf4e5, 64763929a, df4e38185, 1c09aab79, 935627a58, 791746edd, 629abcf15, 514d4d075. PR #381 fusionada con el merge commit 9c86b86e5.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo. El delta no contiene secciones REMOVED ni MODIFIED; `--supersede` no fue necesario.

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| local-web-dashboard | Consulta de la Parada Actual del Cambio desde el Detalle del Incremento | prov:WEB:1 | RF-WEB-38 | assigned |
| local-web-dashboard | Ruta POST de Registro de Puerta con las Protecciones de las Rutas Mutantes | prov:WEB:2 | RF-WEB-39 | assigned |
| local-web-dashboard | Ruta POST de Publicación de Documentos del Cambio con Aviso Remoto Confirmado | prov:WEB:3 | RF-WEB-40 | assigned |
| local-web-dashboard | Presentación de las Salidas de la Parada: Tres Salidas y Dos en `integration` | prov:WEB:4 | RF-WEB-41 | assigned |
| local-web-dashboard | Modo de Revisión por Commit al Aprobar `tasks` | prov:WEB:5 | RF-WEB-42 | assigned |
| local-web-dashboard | Un Único Clic de Confirmación con Aviso de Operación Remota Encadena Registrar y Publicar | prov:WEB:6 | RF-WEB-43 | assigned |
| local-web-dashboard | Fallo de Publicación como «Aprobada sin Publicar» con Reintento | prov:WEB:7 | RF-WEB-44 | assigned |
| local-web-dashboard | Rutas de Puertas Clasificadas en el Canario de Secretos y sin Estilos ni Scripts en Línea | prov:WEB:8 | RF-WEB-45 | assigned |
| sdd-block-review-gates | El Dashboard Local es un Canal Humano Equivalente a `axiom sdd gate record` | prov:GAT:1 | RF-GAT-34 | assigned |

Sin duplicados (`duplicates[]` vacío). Ningún provisional fue reasignado.

## Artefactos Consolidados
- `proposal.md`: presente
- `specs/local-web-dashboard/spec.md` y `specs/sdd-block-review-gates/spec.md`: presentes, integrados
- `design.md`: presente
- `tasks.md`: presente, 22 de 22 completadas, 0 pendientes
- `verify-report.md`: presente, `limpio` en el momento de su escritura
- `coverage-review.spec.md` y `coverage-review.design.md`: presentes (informativos; 16 huecos en spec y 14 en design, seis notas vinculantes aplicadas)
- `kickoff.yaml`, `exploration.md`, `resumen-spec.html`, `resumen-design.html`: presentes
- `apply-progress`: ausente (`applyProgress: missing` en el status); no bloquea el archivado
- `ids.yaml`: traza de la integración, generada por `axiom sdd archive-integrate`

## Integración nativa (trazabilidad)
1. `axiom sdd archive-check --cwd <worktree> --change puertas-desde-ui --json`: exit 0, `ok: true`, sin hallazgos, `iso: false`.
2. `axiom sdd archive-integrate ... --check --json`: exit 0, dos capacidades `composed`, nueve asignaciones `assigned`, sin escritura.
3. `axiom sdd archive-integrate ... --json`: exit 0, escribió `openspec/specs/local-web-dashboard/spec.md`, `openspec/specs/sdd-block-review-gates/spec.md`, los deltas, `tasks.md` (líneas `Cubre:`) e `ids.yaml`.
4. `axiom archive sync --path <worktree>`: exit 0. 50 especificaciones, 643 requisitos activos (antes 634), 3220 escenarios BDD (antes 3188). `openspec/INDEX.md` regenerado con 9 requisitos nuevos.

## Gobernanza
- Kickoff sellado en `continuous` con `docsPublication` no declarado, `explainerHtml: yes`, rol único `fullstack`.
- `gates.yaml` no existe y no hay gates de bloque; el status proyecta `dependencies.archive: ready`, por lo que no se exigió un gate de integración.
- `acceptanceTests`, `integrations.yaml` y `technicalContext.maintained` no se proyectan: no hay secciones de marcas de pruebas, de seguimiento en el tracker ni de contexto mantenido.

## Pendientes informativos (no bloquean el archivado)
- Prueba en navegador real de la UI: no ejecutada (sin ejecutor de JS).
- `role-apply` multirepo y `RecordRoot` con worktree de archivado registrado: sin prueba con git real.
- Los verbos del tracker no se lanzan desde la UI: fuera de alcance del cambio.
- Auto-fix denegado por el clasificador; la CI se vigiló con `gh pr checks --watch` y pasó a la primera.
- `--docs-publication auto` denegado; el kickoff quedó sellado sin publicación automática.
- Trinquete de código muerto: no se ejecutó en local; pasó en la CI.

## Nota sobre bytes archivados
`archive-integrate` reescribió los provisionales en `tasks.md` (líneas `Cubre:`) y en los deltas antes del movimiento, según su contrato. Los bytes de `tasks.md` del archivo no son, por tanto, los originales del último commit; el cambio está en `git diff` del worktree de archivo y es la reescritura de identificadores, no una alteración de contenido. El `diff -r` de la sección siguiente compara el archivo con el snapshot tomado tras la integración, por lo que verifica el movimiento, no la integración.

## Verificación del movimiento
```
diff -r "$snapshot_root/source" "openspec/changes/archive/2026-11-27-puertas-desde-ui"
(salida vacía, DIFF_EXIT=0)
```
La carpeta de origen `openspec/changes/puertas-desde-ui` ya no existe.
