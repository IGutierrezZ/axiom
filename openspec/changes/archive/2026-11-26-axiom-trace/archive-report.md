# Reporte de Archivado Formal: axiom-trace

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: Limpio en el momento de la verificación (`verify-report` 49565e0c, 1 WARNING observado, sin bloqueos). Estado final por fuentes de mayor rango: código fusionado en `main` mediante la PR #337 (merge commit 25c4f3b0), con CI completa en verde, incluido Deadcode Ratchet.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-axiom-trace/`
- Capacidad nueva integrada: `requirement-trace` (`openspec/specs/requirement-trace/spec.md`), con 15 requisitos.

## Hechos de Estado Final
Fuentes por rango (la de mayor rango prevalece):
1. Tareas persistidas (`tasks.md`): 27 de 27 marcadas como completadas, 0 pendientes.
2. Hechos explícitos de cierre (lanzamiento del archivado): código fusionado en `main` por la PR #337 (merge commit 25c4f3b0); commits de implementación bfc94c78..8bf5366e (U1..U10); merge de `main` 58f9d667; CI verde completa.
3. Snapshots intermedios (`verify-report` 49565e0c, `apply-progress`): valen como historia de su momento.

Cambios posteriores a los snapshots, según fuente de mayor rango:
- La SUGGESTION del verify sobre el Deadcode Ratchet queda resuelta: la CI de la PR #337 pasa completa, incluido Deadcode Ratchet.
- Por C-08-l, el caso `semantic` se retiró de `cmd/axiom/main.go` en el merge 58f9d667.
- Por C-08-h, se borró `internal/workspace/docs_pin_test.go` en el merge 58f9d667. El pin de «axiom trace <ID>» añadido en U10 desapareció con ese fichero; la decisión fue borrar centinelas de documentación. El manual (`docs`, commit 8bf5366e) conserva la documentación del verbo. No hay regresión verificada por esta vía: queda anotado como cambio de cobertura, no como defecto.
- El WARNING del verify (`odd_unparseable` solo si el documento cita el ID) sigue como observación. No se ha confirmado como defecto ni como corregido.

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| — | — | — | Ninguno. Incremento puramente aditivo. | — |

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| requirement-trace | Verbo `axiom trace <ID>` y Sus Opciones | prov:trace:1 | REQ-48.1 | assigned |
| requirement-trace | Validación del ID Contra el Patrón del Proyecto con Error Tipado | prov:trace:2 | REQ-48.2 | assigned |
| requirement-trace | Resolución Entre Provisional y Definitivo por la Traza | prov:trace:3 | REQ-48.3 | assigned |
| requirement-trace | Punto de Extensión de Alias | prov:trace:4 | REQ-48.4 | assigned |
| requirement-trace | Incrementos SDD Archivados como Fuente | prov:trace:5 | REQ-48.5 | assigned |
| requirement-trace | Clasificación de Bug en Cambios SDD | prov:trace:6 | REQ-48.6 | assigned |
| requirement-trace | Documentos ODD como Fuente | prov:trace:7 | REQ-48.7 | assigned |
| requirement-trace | Commit de Integración en la Línea `--first-parent` | prov:trace:8 | REQ-48.8 | assigned |
| requirement-trace | Contrato JSON `axiom.trace/v1` y Salida de Texto | prov:trace:9 | REQ-48.9 | assigned |
| requirement-trace | Solo Lectura | prov:trace:10 | REQ-48.10 | assigned |
| requirement-trace | ID Válido sin Apariciones | prov:trace:11 | REQ-48.11 | assigned |
| requirement-trace | Rama de Integración Ausente con Remedio | prov:trace:12 | REQ-48.12 | assigned |
| requirement-trace | Orden Determinista | prov:trace:13 | REQ-48.13 | assigned |
| requirement-trace | Topología Multirepo con Rutas Ancladas en su Raíz Git | prov:trace:14 | REQ-48.14 | assigned |
| requirement-trace | Criterio de Terminado de Trazabilidad de Punta a Punta | prov:trace:15 | REQ-48.15 | assigned |

Duplicados históricos reportados por `archive-integrate` (no bloqueantes, fuera del cambio): REQ-1.1 a REQ-1.4, REQ-2.1 a REQ-2.3, REQ-3.1 a REQ-3.3 y REQ-4.1 a REQ-4.2.

## Artefactos Consolidados
- `proposal.md`: presente.
- `specs/requirement-trace/spec.md`: presente (delta, integrado en la spec viva).
- `design.md`: presente.
- `tasks.md`: presente; 27 de 27 tareas completadas, 0 pendientes. Las referencias `Cubre:` y los identificadores se reescribieron a los definitivos.
- `verify-report.md`: presente (snapshot de 49565e0c; su estado no se presenta como estado final).
- `explore.md`, `kickoff.yaml`, `ids.yaml`, `coverage-review.spec.md`, `coverage-review.design.md`: presentes.
- Sin matriz de verificación (cambio no ISO).

## Revisiones de Cobertura Pendientes (informativas)
- `coverage-review.spec.md`: 13 huecos; varios absorbidos en `tasks.md`.
- `coverage-review.design.md`: 11 huecos.
- Pendientes notables, no tratados en este cambio: rama de integración por repositorio en multirepo; un error real de git aborta la consulta; anclaje de código solo por `verify-report.md`.

## Trazabilidad de Integración
- Comprobación previa: `axiom sdd archive-check --cwd C:/repos/axiom-wt/axiom-trace-archive/axiom --change axiom-trace --json` → exit 0, `ok: true`, sin hallazgos, `iso: false`.
- Chequeo de integración: `axiom sdd archive-integrate ... --check --json` → exit 0, sin escrituras.
- Integración real: `axiom sdd archive-integrate ... --json` → exit 0, `outcome: created` para `requirement-trace`, 15 asignaciones `assigned`.
- Catálogo maestro: `axiom archive sync --path ...` → exit 0; 49 especificaciones, 511 requisitos, 2231 escenarios; `openspec/INDEX.md` regenerado.
- Contexto técnico mantenido: no proyectado en el estado; no aplica.

## Observaciones de Cierre
- La spec viva `openspec/specs/requirement-trace/spec.md` la compone el verbo tal cual, con el encabezado de delta (`# Delta para requirement-trace`) y la sección `## ADDED Requirements`, y sin `## Purpose`. No se corrigió a mano porque la skill reserva la escritura de `openspec/specs/` al verbo. Queda pendiente de decisión.
