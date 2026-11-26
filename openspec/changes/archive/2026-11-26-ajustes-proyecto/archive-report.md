# Reporte de Archivado Formal: ajustes-proyecto

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación (final): Aprobado. El informe `verify-report.md` termina en `**Estado:** limpio` tras la corrección del CRITICAL en el commit `15514e98`. La primera versión del informe quedó en `no limpio` (snapshot de verificación, commit `901ee7b06`); ese estado queda como historial y no describe el estado final.
- Estado de Implementación: 9/9 unidades de trabajo (commits `71c56d0f` a `2689dc6e`, más planificación `766c9509`). `tasks.md` contiene 39 casillas, todas marcadas; 0 pendientes.
- Integración: PR de código https://github.com/IGutierrezZ/axiom/pull/361 fusionada con merge commit `3c406805`, CI en verde según el encargo de archivado.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-ajustes-proyecto/`

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| local-web-dashboard | RF-WEB-01, RF-WEB-07, RF-WEB-10 | Modificados por ID (no SUPERSEDED) | El delta MODIFIED reemplaza el texto de cada requisito en la spec viva. RF-WEB-10 cambia de título. No hay deprecación ni eliminación. | `local-web-dashboard/spec.md` (RF-WEB-01, RF-WEB-07, RF-WEB-10) |

Ningún requisito quedó marcado como `[SUPERSEDED]`. Los requisitos añadidos son RF-WEB-18 a RF-WEB-21 (`local-web-dashboard`) y RF-TOP-21 a RF-TOP-23 (`workspace-topology`).

## Integración de Especificaciones Vivas
- `openspec/specs/local-web-dashboard/spec.md`: composición nativa con `axiom sdd archive-compose` (sin `--supersede`). Pasa de 501 a 816 líneas.
- `openspec/specs/workspace-topology/spec.md`: composición nativa con `axiom sdd archive-compose` (sin `--supersede`). Pasa de 808 a 985 líneas.
- `openspec/INDEX.md`: regenerado con `axiom archive sync --path`. Resultado: 50 especificaciones, 576 requisitos activos, 2744 escenarios. Contiene los IDs RF-WEB-01/07/10/18/19/20/21 y RF-TOP-21/22/23.

## Comprobaciones
- `axiom sdd archive-check`: correcto (cambio no ISO, sin comprobaciones adicionales).
- `go test ./internal/idmigration -run Residue -count=1`: `ok`.
- Contexto técnico mantenido: no proyectado en el estado; no aplica integración de contexto.
- Puerta de integración: el estado nativo reportó `archive: ready` sin bloqueos. El cambio no tiene `kickoff.yaml` con puerta `integration` (`gates: []`, estilo `continuous`).

## Trazabilidad de Observaciones (Engram, espejos del almacén `openspec`)
Localizados con búsqueda por `sdd/ajustes-proyecto`. Son espejos resumidos; la fuente íntegra son los ficheros de la carpeta archivada.
- `sdd/ajustes-proyecto/proposal`: #2710
- `sdd/ajustes-proyecto/spec`: #2711
- `sdd/ajustes-proyecto/design`: #2713
- `sdd/ajustes-proyecto/tasks`: #2714
- `sdd/ajustes-proyecto/coverage-review/spec`: #2712
- `sdd/ajustes-proyecto/coverage-review/design`: #2715
- `sdd/ajustes-proyecto/apply-progress`: #2719
- `sdd/ajustes-proyecto/verify-report`: #2720 (snapshot del momento de verificación; el CRITICAL que registra fue corregido después, ver Resumen de Cierre)

## Artefactos Consolidados
- `proposal.md`: presente
- `specs/local-web-dashboard/spec.md` y `specs/workspace-topology/spec.md`: presentes, integrados en las specs vivas
- `design.md`: presente
- `tasks.md`: presente, 39/39 casillas marcadas, 0 pendientes
- `verify-report.md`: presente, estado final `limpio`
- `exploration.md`, `kickoff.yaml`: presentes
- `coverage-review.spec.md` y `coverage-review.design.md`: presentes

## Hallazgos Abiertos al Cierre (informativos, no bloquean el archivado)
- Cobertura de la spec: 14 huecos abiertos, ronda 1 (ver `coverage-review.spec.md`).
- Cobertura del diseño: huecos 1, 3, 5, 6, 7, 8, 10 y 11 abiertos (ver `coverage-review.design.md`). Destacan: `writable` no refleja ficheros de solo lectura ni enlaces simbólicos, y `/api/workspace` puede desfasarse en multirepo.

## Desvíos Registrados
- No se selló `--docs-publication auto` (denegado por el clasificador).
- No se selló `explainer_html`: el binario `axiom` instalado es anterior a C-06-b.
- Auto-fix denegado; la CI se vigiló con `gh pr checks --watch`.
