# Reporte de Archivado Formal: puerto-reqid

## Resumen de Cierre
- Fecha de Archivado: 2026-11-24
- Estado de Verificación: Limpio según el informe de verificación final (`verify-report.md`), tal como lo declara el lanzamiento del orquestador (fuente de mayor rango que la instantánea del informe). Judgment-day: APPROVED en la ronda 1 con JD-1 corregido. W-5 corregido en `1c2f9b24` sin re-juicio independiente.
- Estado de Tareas: 41 de 41 completadas según `tasks.md` (`taskProgress.allComplete: true`). La tarea 10.2 se cerró en esta rama por la CI: el job «Unit Tests» (`go test ./...` más el trinquete) pasó sobre `2cd8ac77`.
- Destino en Archivo: `openspec/changes/archive/2026-11-24-puerto-reqid/`
- Integración: PR #293 fusionada en `main` el 2026-11-24 (merge `fc3f5ab1`).

## Historial de Commits de la Rama
- Planificación: `b26c9d97`
- Implementación: `5ca498e6` … `70bca4a9`
- Registros: `c458f5e8`, `6eb58bc3`, `f1385887`
- Corrección JD-1: `2a5888f8`
- Informe de judgment-day: `b5c874de`
- Corrección W-5: `1c2f9b24`
- Trinquete de código muerto: `2cd8ac77` (se retiró la envoltura `openspecdoc.IsSuperseded` sin llamadas)

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. No se usó `--supersede` y ninguna delta contiene secciones `REMOVED`. Las deltas `MODIFIED` y `RENAMED` sustituyen el texto del requisito vigente sin marcar ningún requisito previo como `[SUPERSEDED / DEPRECADO]`.

## Sincronización de Especificaciones Principales
| Dominio | Acción | Detalle |
|---------|--------|---------|
| axiom-sdd-cli-integration | Actualizada | 1 MODIFIED (`archive-compose`) |
| odd-living-spec-integration | Actualizada | 1 ADDED, 1 MODIFIED, 1 RENAMED (`archive-compose`) |
| sdd-archive-lifecycle | Actualizada | 1 MODIFIED (`archive-compose`) |
| sdd-iso-evidence | Actualizada | 1 MODIFIED, 1 RENAMED (`archive-compose`) |
| spec-baseline-integration | Actualizada | 1 MODIFIED (`archive-compose`) |
| requirement-identifiers | Creada | Capacidad nueva; copia mecánica byte a byte de la delta |

Tras la composición, se recortó el salto de línea final sobrante de `odd-living-spec-integration`, `sdd-archive-lifecycle` y `requirement-identifiers`. Las deltas archivadas conservan sus bytes originales.

## Catálogo Maestro
- `openspec/INDEX.md` regenerado con `axiom archive sync` (exit 0): 75 especificaciones consolidadas, 650 requisitos activos, 2279 escenarios BDD.
- Líneas de `INDEX.md`: 3414 antes, 3544 después. Referencias a `requirement-identifiers`: 3.

## Artefactos Consolidados
- `proposal.md`
- `specs/` (6 deltas: axiom-sdd-cli-integration, odd-living-spec-integration, requirement-identifiers, sdd-archive-lifecycle, sdd-iso-evidence, spec-baseline-integration)
- `design.md`
- `tasks.md` (con la marca 10.2 `[x]` pendiente de commit en el momento del archivado)
- `verify-report.md` (instantánea intermedia; su estado final se toma de la fuente de mayor rango indicada arriba)
- Artefacto ausente: `apply-progress` (no existe en el cambio).

## Trazabilidad
- Almacén: `openspec`. No se leyeron observaciones de Engram para este cambio; las fuentes son los ficheros indicados.
- Órdenes ejecutadas: `axiom sdd archive-check` (exit 0, cambio no ISO), `axiom sdd archive-compose` (5 veces, exit 0), `axiom archive sync` (exit 0), `git mv` (exit 0), `diff -r` de readback (vacío).

## Pendientes e Informativos (no corregidos en el archivado)
- Sospechas de un solo juez y los hallazgos S-1 y S-2: siguen abiertos como informativos según `verify-report.md`. No se corrigieron durante el archivado.
- Discrepancia del plan: `routing.go` no contiene `REQ-<n>.<m>` y no cambió ningún golden.

## Avisos para el Orquestador
- El commit de archivado debe incluir la marca de `tasks.md` (10.2) que aún no estaba confirmada en el momento del archivado.
- Los cambios de `openspec/specs/` y `openspec/INDEX.md` quedan sin confirmar en esta rama hasta que el orquestador haga el commit.
