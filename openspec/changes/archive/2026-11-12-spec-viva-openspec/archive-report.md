# Reporte de Archivado Formal: spec-viva-openspec

## Resumen de Cierre
- Fecha de Archivado: 2026-11-12
- Estado de Verificación: `verify-report.md` presente, con veredicto de 0 CRITICAL, 0 WARNING y 3 SUGGESTION en el momento de la verificación. Ese informe cita un fallo de entorno conocido en `internal/components/filemerge` (junctions de Windows, también en `main`); no se re-ejecutó en esta fase.
- Estado de Tareas: 67 de 67 casillas marcadas `[x]` en `tasks.md` al archivar.
- Precondiciones: `axiom sdd archive-check --change spec-viva-openspec` sale con código 0 (el cambio no es ISO y no tiene hallazgos). `axiom sdd status` proyecta `dependencies.archive: ready` y `gates: []`, con kickoff `continuous`.
- Destino en Archivo: `openspec/changes/archive/2026-11-12-spec-viva-openspec/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ningún requisito se marca `[SUPERSEDED / DEPRECADO]`: no se usó `--supersede`. Las retiradas se aplican como bloques `REMOVED` declarados en la delta, con su `(Motivo: ...)` y `(Migración: ...)`.

| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| `spec-baseline-declaration` | 15 requisitos (REQ-24.13 a REQ-24.25, REQ-24.60, REQ-24.62) | REMOVED | Retirada de la baseline ajena (decisiones D5 y R1 de `proposal.md`). Motivo y migración por entrada en la delta. | `workspace-topology` (REQ-30.21), `openspec-adoption`, `project-technical-context` según cada entrada |
| `spec-baseline-integration` | 19 requisitos (REQ-24.38 a REQ-24.54, REQ-24.59, REQ-24.61) | REMOVED | Retirada de la baseline ajena. Motivo y migración por entrada en la delta. | `sdd-archive-lifecycle`, `project-technical-context`, `odd-living-spec-integration`, `openspec-adoption` según cada entrada |

## Correcciones de Formato en la Delta
El compositor rechazó `spec-baseline-declaration` con `missing required "(Reason: ...)" note`, porque sus 15 entradas REMOVED no llevaban la nota. Se añadió una línea `(Reason: ...)` antes de cada `(Motivo: ...)`, con el mismo texto del Motivo de esa entrada, en:
- `openspec/changes/spec-viva-openspec/specs/spec-baseline-declaration/spec.md`: 15 líneas.
- `openspec/changes/spec-viva-openspec/specs/spec-baseline-integration/spec.md`: 19 líneas.

Las líneas `(Motivo: ...)` y `(Migración: ...)` se conservan sin cambios. La nota de Migración no se copió al Reason. Ninguna otra delta del cambio tiene sección `## REMOVED Requirements`.

## Composición y Creación por Dominio
| Dominio | Acción | Resultado |
|---------|--------|-----------|
| `axiom-sdd-cli-integration` | Compuesto | `archive-compose` exit 0 |
| `change-branch-lifecycle` | Compuesto | `archive-compose` exit 0 |
| `living-documentation` | Compuesto | `archive-compose` exit 0 |
| `multi-project-hub` | Compuesto | `archive-compose` exit 0 |
| `organic-agent-trigger-rules` | Compuesto | `archive-compose` exit 0 |
| `planning-home-resolution` | Compuesto | `archive-compose` exit 0 |
| `project-technical-context` | Compuesto | `archive-compose` exit 0 |
| `sdd-archive-lifecycle` | Compuesto | `archive-compose` exit 0 |
| `sdd-docs-publication` | Compuesto | `archive-compose` exit 0 |
| `sdd-orchestrator-assets` | Compuesto | `archive-compose` exit 0 |
| `tui-ui-parity` | Compuesto | `archive-compose` exit 0 |
| `workspace-topology` | Compuesto | `archive-compose` exit 0 |
| `spec-baseline-declaration` | Retirada (REMOVED) | `archive-compose` exit 0; 0 requisitos restantes. El fichero se conserva (ver Decisión pendiente). |
| `spec-baseline-integration` | Retirada (REMOVED) | `archive-compose` exit 0; 19 requisitos retirados y 1 conservado (REQ-29.14, de otra capacidad, que la propia delta declara fuera de alcance). |
| `odd-living-spec-integration` | Creada (copia de la delta) | `cp -R` y `diff -r` vacío |
| `openspec-adoption` | Creada (copia de la delta) | `cp -R` y `diff -r` vacío |

Los 14 dominios compuestos cubren los 16 deltas del cambio junto con las 2 creaciones. No se editó a mano ningún canónico.

## Decisión Pendiente: Carpeta de la Capacidad Retirada
La delta `spec-baseline-declaration` declara: «Al archivar, la carpeta `openspec/specs/spec-baseline-declaration/` se retira». La skill `sdd-archive` no define ninguna regla para borrar una capacidad que queda sin requisitos, así que no se ha borrado. `openspec/specs/spec-baseline-declaration/spec.md` queda con 0 requisitos y `openspec/INDEX.md` lo lista con 0 requisitos. Queda para decisión del autor si se elimina la carpeta en un cambio posterior.

## Artefactos Consolidados
- `proposal.md`: presente.
- `specs/`: 16 deltas presentes.
- `design.md`: presente.
- `tasks.md`: presente, 67 de 67 completadas.
- `verify-report.md`: presente (ver Estado de Verificación).
- `archive-report.md`: este informe.

## Pruebas y Comprobaciones
- `axiom sdd archive-check --cwd <worktree> --change spec-viva-openspec`: exit 0, sin hallazgos.
- `axiom sdd archive-compose` (`--canonical --delta --output` + `mv`): 14 dominios con exit 0 tras la corrección de formato.
- `diff -r` de las 2 capacidades creadas: vacío.
- `axiom archive sync --path <worktree>`: exit 0; 72 especificaciones, 553 requisitos activos, 1644 escenarios; `openspec/INDEX.md` regenerado.
- `go test -count=1 ./internal/workspace/ ./internal/docpaths/ ./internal/sddstatus/ ./internal/kickoff/ ./internal/isoevidence/`: todos los paquetes en `ok`.
- `diff -r` del movimiento de la carpeta: ver la salida verbatim en el resultado de la fase.

## Trazabilidad
- Observaciones de Engram leídas en esta fase: ninguna. Fuentes: los ficheros del cambio y el estado nativo de `axiom sdd status`.
- No se registró evidencia de gate `integration`: el status proyecta `dependencies.archive: ready` y el cambio no tiene gates.
- Este informe se guarda también en Engram como `sdd/spec-viva-openspec/archive-report`.
