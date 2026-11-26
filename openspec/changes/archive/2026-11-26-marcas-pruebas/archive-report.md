# Reporte de Archivado Formal: marcas-pruebas

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: Verify limpio (`**Estado:** limpio`) en el informe de verificación de la época de verify: 0 CRITICAL, 0 WARNING, 1 SUGGESTION (suite completa no ejecutada en local, cubierta por CI). Instantánea intermedia; no se rehace.
- Cierre de implementación: 35 de 35 tareas marcadas en `tasks.md` (0 pendientes). Rol único `fullstack`, modo `continuous`.
- Integración de código: PR #369, fusionada en `main` (merge commit `f7da2e591`). Commits del PR (`edb77ecaa..b04b92b48`): `docs(sdd): planificar marcas-pruebas (C-06-d)`, `feat(marks): núcleo puro de marcas de pruebas y ruta exacta permitida`, `feat(status): proyectar el resumen de marcas de pruebas`, `feat(sddpublish): publicación acotada y reproducción única`, `feat(dashboard): rutas de marca y publicación de marcas`, `feat(cli): publicador de marcas inyectado en el dashboard`, `feat(dashboard): matriz de marcas y publicación en la pestaña Pruebas`, `docs(sdd): resumen de marcas en archive e integration`, `docs(sdd): verificar marcas-pruebas`.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-marcas-pruebas/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo y sin requisitos `REMOVED`. Los cinco requisitos `MODIFIED` (RF-WEB-17, RF-PUB-01, RF-PUB-05, RF-PUB-07, RF-KOF-22) reemplazan su texto en la capacidad viva sin marcar supersedencia.

## Identificadores Asignados
Fuente: `axiom sdd archive-integrate` (`axiom.sdd-archive-integrate/v1`, exit 0; `nothingToIntegrate: false`; `duplicates: []`). Trazabilidad en `ids.yaml`.

| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| local-web-dashboard | Pantalla de Marcas por Caso y Entorno con Caducidad en la Pestaña «Pruebas» | prov:WEB:1 | RF-WEB-22 | assigned |
| local-web-dashboard | Ruta de Marca con Validación en Servidor y Escritura Local de Solo Añadir | prov:WEB:2 | RF-WEB-23 | assigned |
| local-web-dashboard | Ruta de Publicación de Marcas con Confirmación Explícita y Resultado Tipado | prov:WEB:3 | RF-WEB-24 | assigned |
| local-web-dashboard | Marcas Solo en Multirepo SDD con Estado «No Disponible» Explícito | prov:WEB:4 | RF-WEB-25 | assigned |
| local-web-dashboard | Rutas de Marcas Clasificadas en el Canario de Secretos | prov:WEB:5 | RF-WEB-26 | assigned |
| sdd-archive-lifecycle | El `archive-report` Incluye el Resumen de Marcas y las Marcas Nunca Bloquean el Archivado | prov:ARC:1 | RF-ARC-18 | assigned |
| sdd-block-review-gates | La Presentación de `integration` Muestra el Resumen de Marcas por Entorno como Información | prov:GAT:1 | RF-GAT-33 | assigned |
| sdd-docs-publication | Publicación de Marcas en la Integración del Repositorio de Specs, Acotada a `pruebas-estado.yaml` | prov:PUB:1 | RF-PUB-13 | assigned |
| sdd-docs-publication | Un Único Reintento por Reproducción ante *non-fast-forward* en el Publicador de Marcas | prov:PUB:2 | RF-PUB-14 | assigned |
| sdd-lane-kickoff | Formato de `pruebas-estado.yaml` de Solo Añadir y Caducidad por `plan_digest` | prov:KOF:1 | RF-KOF-25 | assigned |

## Integración de Especificaciones Vivas
Capacidades integradas por `axiom sdd archive-integrate` (5, todas `composed`): `local-web-dashboard`, `sdd-archive-lifecycle`, `sdd-block-review-gates`, `sdd-docs-publication`, `sdd-lane-kickoff`. Las cinco deltas `MODIFIED` del cambio se aplicaron sobre sus requisitos vivos y los `ADDED` recibieron el definitivo indicado arriba.

Catálogo maestro: `axiom archive sync` (exit 0) regenera `openspec/INDEX.md` con 50 especificaciones consolidadas, 612 requisitos activos y 2977 escenarios BDD.

## Marcas de pruebas
No aplica. El cambio no tiene `pruebas.md`; no hay marcas de pruebas que resumir.

## Seguimiento en el tracker
No aplica. El cambio no tiene `integrations.yaml`.

## Artefactos Consolidados
- `proposal.md`
- `specs/` (5 deltas: `local-web-dashboard`, `sdd-archive-lifecycle`, `sdd-block-review-gates`, `sdd-docs-publication`, `sdd-lane-kickoff`)
- `design.md`
- `tasks.md` (35/35 casillas marcadas; los identificadores `prov:` de sus líneas `Cubre:` se reescribieron a definitivos)
- `verify-report.md` (instantánea intermedia, ver Estado de Verificación)
- `coverage-review.spec.md` y `coverage-review.design.md`
- `kickoff.yaml`, `ids.yaml`, `exploration.md`, `resumen-spec.html`, `resumen-design.html`

## Huecos informativos no resueltos
Revisión de cobertura (informativa, no bloqueante, no corregida en archivado):
- `coverage-review.spec.md`: 8 huecos (1 `uncovered_requirement`, 3 `error`, 4 `edge_case`, 1 `use_case`; detalle en el propio informe). Entre ellos, las marcas locales sin publicar al archivar quedan sin aviso (hueco 8).
- `coverage-review.design.md`: 11 huecos (1 `uncovered_requirement`, 2 `error`, 7 `edge_case`, 1 `use_case`; detalle en el propio informe).

Estos huecos se registran tal cual; no se han resuelto ni se ha reabierto ninguna fase.

## Comprobaciones ejecutadas en el archivado
- `axiom sdd archive-check --cwd <raíz> --change marcas-pruebas --json`: exit 0, `ok: true`, `findings: []`, ISO: no.
- `axiom sdd archive-integrate --check`: exit 0, 5 capacidades `composed`, 10 asignaciones `assigned`, `duplicates: []`.
- `axiom sdd archive-integrate` (escritura): exit 0, `written` con 12 rutas (5 specs vivas, 5 deltas reescritas, `tasks.md`, `ids.yaml`).
- `axiom archive sync --path <raíz>`: exit 0.
- `go test ./internal/idmigration -run TestRepositoryLegacyIDResidues -count=1`: ok.
- Movimiento a `archive/2026-11-26-marcas-pruebas/` por `git mv` y readback `diff -r` contra la instantánea previa al movimiento: salida vacía (sin diferencias).

## Notas de trazabilidad
- Observaciones de Engram leídas: no aplica (almacén `openspec`; el informe se guarda también en Engram con `topic_key` `sdd/marcas-pruebas/archive-report`).
- La carpeta del cambio se archiva con `kickoff.yaml` sellado (`executionStyle: continuous`, `explainerHtml: yes`). `dependencies.archive` reportó `ready`.
