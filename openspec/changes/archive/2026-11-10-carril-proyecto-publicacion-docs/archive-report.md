# Reporte de Archivado Formal: carril-proyecto-publicacion-docs

## Resumen de Cierre
- Fecha de Archivado: 2026-11-10
- Estado de Verificación: `PASS con advertencias` según `verify-report.md` (0 CRITICAL, 3 WARNING, 2 SUGGESTION; base `main` en `09e75e66`). Estado final de cada advertencia en «Advertencias al archivar».
- Tareas: 28 de 28 marcadas en `tasks.md` (observado al archivar: 28 `[x]`, 0 `[ ]`). Commits de unidad de trabajo según `verify-report.md`: WU-1 `e92d8c5d`, WU-2 `93796b0b`, WU-4 `87237779`, WU-3 `68a6e659`.
- Integración: PR #235 fusionada en `09e75e66` (comprobado: `09e75e66` es ascendente de la rama de archivado). El cambio usa carril `continuous`, sin puerta `integration` sellada; `dependencies.archive` estaba en `ready`.
- Rama de archivado: `inc/carril-proyecto-publicacion-docs-archive`.
- Destino en Archivo: `openspec/changes/archive/2026-11-10-carril-proyecto-publicacion-docs/`

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| organic-agent-trigger-rules | El Protocolo ODD Incluye una Parada Única con Tres Salidas tras Crear el Documento de Feature (REQ-26.10) | [ELIMINADO] (REMOVED, sin marca `[SUPERSEDED]`; no se usó `--supersede`) | El carril ODD deja de tener paradas; las paradas con tres salidas son exclusivas de SDD en modalidad `checkpointed`. El texto eliminado se conserva en el delta archivado de esta carpeta. | `organic-agent-trigger-rules` REQ-27.17 (doctrina ODD sin paradas) |

Requisitos renombrados con el ID conservado (no son superaciones):
- `sdd-block-review-gates`: REQ-26.4, «La Salida 2 Comprueba que los Artefactos Están Confirmados antes de Registrar» → «La Salida 2 No Exige Artefactos Confirmados y Conserva la Guarda de Rama».
- `sdd-block-review-gates`: REQ-26.5, «La Salida 2 Devuelve el Destino y la Orden de PR sin Ejecutar Git Remoto» → «La Salida 2 No Ejecuta Git en `gate record` y Entrega el Destino de la Publicación».

## Especificaciones Vivas Modificadas
Recuento por operación del delta. «Modificado» sustituye el requisito completo por su nueva versión, sin la línea `(Previously: ...)`. Un requisito renombrado y modificado en el mismo cambio cuenta en ambas columnas.

| Dominio | Acción | ADDED | MODIFIED | RENAMED | REMOVED |
|---------|--------|-------|----------|---------|---------|
| `workspace-topology` | Actualizada | 1 (REQ-27.1) | 1 (REQ-2.1) | 0 | 0 |
| `axiom-sdd-cli-integration` | Actualizada | 2 (REQ-27.2, REQ-27.2b) | 0 | 0 | 0 |
| `sdd-lane-kickoff` | Actualizada | 2 (REQ-27.3, REQ-27.4) | 4 (REQ-21.1, REQ-21.2, REQ-21.3, REQ-25.35) | 0 | 0 |
| `organic-agent-trigger-rules` | Actualizada | 2 (REQ-27.16, REQ-27.17) | 4 (REQ-25.40 y tres requisitos sin ID: «Canonical provider-owned routing facts», «SDD remains optional», «Organic Driven Development is the orchestrator's mandatory default protocol») | 0 | 1 (REQ-26.10) |
| `sdd-block-review-gates` | Actualizada | 1 (REQ-27.12) | 6 (REQ-26.1, REQ-26.3, REQ-26.4, REQ-26.5, REQ-26.6, REQ-26.8) | 2 (REQ-26.4, REQ-26.5) | 0 |
| `change-branch-lifecycle` | Actualizada | 1 (REQ-27.15) | 4 (REQ-25.1, REQ-25.7, REQ-25.10, REQ-26.11) | 0 | 0 |
| `sdd-archive-lifecycle` | Actualizada | 1 (REQ-27.14) | 0 | 0 | 0 |
| `sdd-docs-publication` | Creada (copia mecánica; no existía spec principal) | 8 (REQ-27.5 a REQ-27.11 y REQ-27.13) | 0 | 0 | 0 |

Totales del delta: 18 requisitos añadidos con ID (REQ-27.1 a REQ-27.17 y REQ-27.2b), 19 modificaciones, 2 renombrados y 1 eliminado.

Verificación tras fusionar (con `rg` sobre `openspec/specs/`):
- Cada REQ añadido (REQ-27.1, 27.2, 27.2b, 27.3 a 27.17) aparece exactamente una vez como encabezado de requisito.
- REQ-26.10 ya no aparece como encabezado de requisito en ninguna spec principal.
- Los encabezados antiguos de REQ-26.4 y REQ-26.5 ya no aparecen; los nuevos aparecen una vez cada uno.

Índice maestro: `openspec/INDEX.md` regenerado con `axiom archive sync` (68 especificaciones, 522 requisitos activos, 1390 escenarios). Salida del comando: `[OK] Sincronización de Especificaciones Vivas completada con éxito.`

## Artefactos Consolidados
- `proposal.md`: presente.
- `specs/`: ocho deltas (`workspace-topology`, `axiom-sdd-cli-integration`, `sdd-lane-kickoff`, `organic-agent-trigger-rules`, `sdd-archive-lifecycle`, `sdd-block-review-gates`, `change-branch-lifecycle`, `sdd-docs-publication`).
- `design.md`: presente.
- `tasks.md`: presente, 28 de 28 completadas.
- `verify-report.md`: presente (sin seguimiento en git antes de este archivado; se incorpora al commit de archivado).
- `exploration.md`, `kickoff.yaml`: presentes.

## Advertencias al archivar
- **Limpieza de `(Previously: ...)` en las MODIFIED.** El compositor `axiom sdd-archive-compose` copia las líneas `(Previously: ...)` del delta tal cual. Para cumplir la orden de archivado, las cinco fusiones con MODIFIED que las contenían (`workspace-topology`, `sdd-lane-kickoff`, `organic-agent-trigger-rules`, `sdd-block-review-gates`, `change-branch-lifecycle`) se hicieron desde una copia temporal del delta sin esas líneas (19 líneas en total). Los deltas archivados conservan íntegras esas líneas. Los archivados anteriores (p. ej. `2026-11-03-paradas-tres-salidas`) sí conservaron esas líneas en las specs vivas: este archivado se aparta de ese precedente por instrucción expresa.
- **Comprobación literal «ningún `(Previously:` en `openspec/specs/`» no cumplida, por diseño.** Quedan 61 apariciones en todo el árbol de specs. Todas están en requisitos que este cambio no modifica (procedentes de archivados anteriores): `workspace-topology` 3, `sdd-lane-kickoff` 2, `sdd-block-review-gates` 5, `change-branch-lifecycle` 16, `sdd-archive-lifecycle` 2, y el resto en otros dominios. Ninguna de las líneas que este cambio elimina permanece.
- **REQ-26.10 eliminado sin `--supersede`.** El requisito desaparece de la spec viva, como pidió la orden, y no queda marcado como `[SUPERSEDED]`. El skill sugiere `--supersede` para superaciones; aquí se trata de una eliminación explícita del delta (`REMOVED` con `Reason`), registrada en la tabla de poda.
- **W-1 (`verify-report`, en el momento de la verificación):** REQ-25.7 matizado durante apply. Estado final: resuelto en la spec delta, según el propio informe.
- **W-2 (`verify-report`, en el momento de la verificación):** fallo preexistente en `main`, ajeno al cambio: `internal/components/filemerge` `TestWriteFileAtomicFollows(Relative)DirectoryJunction` en Windows. Estado final: no reobservado en este archivado; sigue abierto como fallo preexistente.
- **W-3 (`verify-report`, en el momento de la verificación):** `internal/cli` completo, `branch`, `sddstatus`, `agentguidance` y `assets` no se reejecutaron en la verificación. Estado final: no cubierto por este archivado. Sí se ejecutaron al archivar las pruebas acotadas indicadas abajo.
- **S-1 y S-2 (`verify-report`):** sugerencias no bloqueantes, sin cambios.

## Verificación al archivar
- `go test ./internal/livingdoc/ ./internal/specbaseline/ ./internal/workspace/ -count=1`: `ok` en los tres paquetes (salida 0), ejecutado en el worktree de archivado.
- Readback `diff -r` de la carpeta del cambio (snapshot previo al movimiento frente al destino): sin salida, código 0.
- Readback `diff -r` de la spec nueva `openspec/specs/sdd-docs-publication/spec.md` frente a su delta: sin salida, código 0.
- Fusiones con `axiom sdd-archive-compose`: código 0 en las siete specs; el comando escribe a un temporal y solo entonces se mueve. Sin ficheros temporales residuales.
- `axiom sdd status` antes del movimiento: `archive: ready`, `blockedReasons: []`, `nextRecommended: archive`.

## Trazabilidad
- Almacén declarado por el estado nativo: `openspec`. Los artefactos se leyeron por ruta; no se leyó ningún observation ID de Engram como fuente. Por ello no hay observation IDs de entrada que registrar.
- Persistencia de este informe: fichero en la carpeta archivada y Engram `sdd/carril-proyecto-publicacion-docs/archive-report`.
- Sin `baseline` ni `technicalContext` proyectados: no hay trazado de integración de proyecto ni contexto de impacto que añadir.
