# Reporte de Archivado Formal: ui-integraciones

## Resumen de Cierre
- Fecha de Archivado: 2026-11-27
- Estado de Verificación: Limpio (`verify-report.md`, rol `fullstack`, estado en el momento de la verificación). Pruebas de archivado ejecutadas en esta fase: ver «Pruebas de la fase de archivado».
- Destino en Archivo: `openspec/changes/archive/2026-11-27-ui-integraciones/`
- Modo de almacenamiento: `openspec` (artefactos en ficheros). Proyecto de Engram: `axiom`, topic `sdd/ui-integraciones/archive-report`.
- Cambio de origen: C-07-h, con `kickoff.yaml` sellado. La precondición de la puerta `integration` la reporta el estado nativo como `dependencies.archive: ready` antes de integrar.
- Commit de código del cambio: fusionado en `main` (merge `ba0e5bbd0`, PR #376).

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| — | Ninguno. | — | La delta no contiene secciones `REMOVED` y no se usó `--supersede` ni `--superseded-requirements`. | — |

Requisitos modificados (sección `MODIFIED`, sustituyen la versión vigente por la de la delta; no son deprecaciones):
| Dominio | Requisito | Estado | Referencia |
|---------|-----------|--------|------------|
| local-web-dashboard | RF-WEB-07 (Estructura de Navegación y Vistas en la Web UI) | MODIFIED | `openspec/specs/local-web-dashboard/spec.md` |
| local-web-dashboard | RF-WEB-10 (Claves de `axiom.yaml` solo por `/api/settings`) | MODIFIED | `openspec/specs/local-web-dashboard/spec.md` |
| local-web-dashboard | RF-WEB-18 (Pantalla «Ajustes del Proyecto») | MODIFIED | `openspec/specs/local-web-dashboard/spec.md` |
| local-web-dashboard | RF-WEB-19 (Endpoint `GET|PUT /api/settings`) | MODIFIED | `openspec/specs/local-web-dashboard/spec.md` |
| workspace-topology | RF-TOP-21 (Catálogo único de claves de `axiom.yaml`) | MODIFIED | `openspec/specs/workspace-topology/spec.md` |
| workspace-topology | RF-TOP-22 (Escritor de round-trip de `axiom.yaml`) | MODIFIED | `openspec/specs/workspace-topology/spec.md` |

## Identificadores Asignados
Fuente: `assignments[]` del JSON de `axiom sdd archive-integrate` (exit 0, `written[]` con 5 entradas). Los identificadores se asignaron con la herramienta, no a mano.

| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| local-web-dashboard | Cabeceras de Seguridad Generales con CSP Fija, sin CORS y sin Respuesta Genérica a `OPTIONS` | prov:WEB:1 | RF-WEB-27 | assigned |
| local-web-dashboard | `Host` y `Origin` Locales en las Rutas que Mutan y en Todas las Rutas `/api/integrations*` | prov:WEB:2 | RF-WEB-28 | assigned |
| local-web-dashboard | `GET /api/integrations` Devuelve Manifiesto, Valores, Estado del Secreto e Identidad sin Revelar Nunca un Secreto | prov:WEB:3 | RF-WEB-29 | assigned |
| local-web-dashboard | Guardar y Eliminar el PAT por Endpoints Propios que Responden Solo `source` | prov:WEB:4 | RF-WEB-30 | assigned |
| local-web-dashboard | «Probar Conexión» Resuelve y Persiste la Identidad | prov:WEB:5 | RF-WEB-31 | assigned |
| local-web-dashboard | «Descubrir» Rellena los Desplegables con el Descubrimiento del Plugin | prov:WEB:6 | RF-WEB-32 | assigned |
| local-web-dashboard | Pantalla «Integraciones» Generada desde el Manifiesto del Plugin | prov:WEB:7 | RF-WEB-33 | assigned |
| local-web-dashboard | Pestaña «Integraciones» en el Detalle del Cambio | prov:WEB:8 | RF-WEB-34 | assigned |
| local-web-dashboard | Alta de Incremento con Work Item de Origen | prov:WEB:9 | RF-WEB-35 | assigned |
| local-web-dashboard | Enlace «Abrir en Azure DevOps» con URL Validada | prov:WEB:10 | RF-WEB-36 | assigned |
| local-web-dashboard | Rutas de Integraciones Clasificadas en el Canario de Secretos | prov:WEB:11 | RF-WEB-37 | assigned |

Los requisitos modificados (RF-WEB-07, RF-WEB-10, RF-WEB-18, RF-WEB-19, RF-TOP-21, RF-TOP-22) conservan su identificador definitivo: no hay provisionales en ellos. `duplicates[]` está vacío.

## Integración y Trazabilidad
- `axiom sdd archive-check --cwd <worktree> --change ui-integraciones --json`: exit 0, `ok: true`, sin hallazgos, cambio no ISO.
- `axiom sdd archive-integrate --cwd <worktree> --change ui-integraciones --json`: exit 0. Capacidades: `local-web-dashboard` (`composed`) y `workspace-topology` (`composed`). Escritos: `openspec/specs/local-web-dashboard/spec.md`, `openspec/specs/workspace-topology/spec.md`, la delta `specs/local-web-dashboard/spec.md` (identificadores definitivos), `tasks.md` (líneas `Cubre:` reescritas a los identificadores definitivos) y `ids.yaml` (traza nueva).
- Excepción registrada: `tasks.md` y la delta se reescribieron dentro de la integración, antes del snapshot de movimiento. Los bytes archivados son los que dejó el verbo; el readback de movimiento compara contra ese estado.
- `axiom archive sync --path <worktree>`: exit 0. Catálogo maestro regenerado en `openspec/INDEX.md`: 50 especificaciones consolidadas, 633 requisitos activos, 3185 escenarios.
- Contexto técnico mantenido: no proyectado por el estado nativo; no aplica la rama de `axiom archive context`.
- Avisos de fases anteriores comprobados: no hay secciones `## Registro de remediación` en `openspec/specs/`; no hay cambios en el `## Purpose` de las especificaciones afectadas; los identificadores `REQ-` que aparecen en `openspec/` están solo en carpetas históricas de `openspec/changes/archive/` y no fueron tocados por este archivado.

## Artefactos Consolidados
Contenido de `openspec/changes/archive/2026-11-27-ui-integraciones/`:
- `proposal.md`: presente.
- `specs/local-web-dashboard/spec.md`: presente (delta con identificadores definitivos).
- `specs/workspace-topology/spec.md`: presente (delta con MODIFIED RF-TOP-21 y RF-TOP-22).
- `design.md`: presente.
- `tasks.md`: presente, 10 de 10 tareas completadas, 0 pendientes.
- `verify-report.md`: presente, estado `limpio` en su momento de escritura.
- `apply-progress.md`, `exploration.md`, `coverage-review.spec.md`, `coverage-review.design.md`, `kickoff.yaml`, `ids.yaml`, `resumen-spec.html`, `resumen-design.html`: presentes.
- `archive-report.md`: este documento (añadido tras el movimiento, sin existir en el origen).

## Estado Final del Cambio al Cierre
Fuentes por orden de jerarquía: `tasks.md` (persistido), luego el lanzamiento de archivado (`verify` limpio y los residuales declarados) y, por último, `verify-report.md` y `coverage-review.*.md` como instantáneas de su momento.
- Tareas: 10 completadas, 0 abiertas (`tasks.md`, al cierre).
- Verificación (`verify-report.md`, instantánea del momento de la verificación): `**Estado:** limpio`. Su sección «No verificado» declara, como límite de aquella verificación: comprobación visual de la CSP en navegador real (H-05), porque el arnés no ejecuta JavaScript, y multirepo con git real.
- Pendientes informativos declarados al archivar (no bloquean el cierre):
  - Comprobación visual de la CSP en navegador real (H-05).
  - Multirepo con git real.
  - El título del work item no se muestra en la interfaz.
  - `/api/explainer` no envía `frame-ancestors` por su CSP fija.
- Huecos de cobertura: `coverage-review.spec.md` (ronda 1, 2026-11-26) registra 22 huecos tras deduplicar y `coverage-review.design.md` (ronda 1, 2026-11-26) registra 12. El informe de design declara el hueco 12 obsoleto (la spec ya fue ajustada, la consecuencia de `kickoff.yaml` ausente sigue vigente). Ningún hueco fue cerrado de forma individual en este archivado: se registran como no resueltos de forma informativa, sin efecto sobre el cierre.

## Pruebas de la fase de archivado
- `go test ./internal/idmigration -run TestRepositoryLegacyIDResidues -count=1`: ok (exit 0).
- `go test ./internal/cli -run 'ArchiveIntegrate|ArchiveSync|Index' -count=1`: ok (exit 0).

## Movimiento de la carpeta
- Origen: `openspec/changes/ui-integraciones/`. Destino: `openspec/changes/archive/2026-11-27-ui-integraciones/`.
- Readback: `diff -r` del snapshot previo al movimiento contra el destino, salida vacía (exit 0). El origen ya no existe.
- Este `archive-report.md` es aditivo y se excluye de la comparación.

## Trazabilidad de Observaciones
- Modo `openspec`: los artefactos se leyeron desde ficheros; no se leyó ninguna observación de Engram en esta fase, por lo que no hay IDs de observación que registrar.
