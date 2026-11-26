# Reporte de Archivado Formal: migrar-ids-rf

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Cambio: migrar-ids-rf (C-03-e), migración de las especificaciones propias de Axiom de `REQ-n.m` a identificadores funcionales `RF-<DOM>-<NN>`.
- Código: PR #347 fusionada en `main` (merge `acaa8343`). Rama de cambio `inc/migrar-ids-rf`; rama de archivado `inc/migrar-ids-rf-archive` (worktree `C:\repos\axiom-wt\migrar-ids-rf-archive\axiom`).
- Estado de Verificación: verify limpio en el momento de la verificación (`verify-report.md`, commit `771e53b4`): 0 CRITICAL, 0 WARNING, 2 SUGGESTION. Posteriormente se añadieron el merge de `main` (`f2c0e7f3`, conflictos resueltos con la versión de `main` y migración reaplicada) y la corrección del trinquete de código muerto (`42b7a74f`, `resolveEquivalents` movido a `resolve_helpers_test.go`). CI de #347 en verde.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-migrar-ids-rf/`

## Estado Final de las Tareas (observado en `tasks.md`)
- Casillas marcadas: 28 de 28; pendientes: 0.
- Nota de discrepancia: el encargo de archivado indicaba 38/38. El artefacto persistido contiene 28 casillas, todas marcadas. No se ha podido reconciliar el total de 38; se registra el valor observado en el archivo. Ninguna casilla se ha modificado.

## Integración de Deltas en la Spec Viva
Herramienta: `axiom sdd archive-integrate --cwd <worktree> --change migrar-ids-rf` (precedida de `--check`, sin escritura, con el mismo plan).

| Provisional | Definitivo asignado | Capacidad | Requisito |
|-------------|---------------------|-----------|-----------|
| `prov:TRC:1` | `RF-TRC-16` | `requirement-trace` | Mapa Versionado de Identificadores Heredados como Fuente de Equivalencias de la Traza |
| `prov:TRC:2` | `RF-TRC-17` | `requirement-trace` | Mapa Ausente Degrada sin Alias y Mapa Inválido Falla con Error Tipado |
| `prov:RID:1` | `RF-RID-16` | `requirement-identifiers` | Las Specs Propias de Axiom Usan el Patrón Declarado y no Citan Identificadores Heredados Fuera de los Residuos Declarados |

- Las asignaciones coinciden con las referencias cruzadas que ya citaba el texto de las deltas (RF-TRC-16, RF-TRC-17 y RF-RID-16). Comprobado en `openspec/specs/requirement-trace/spec.md` (líneas 358 y 399) y `openspec/specs/requirement-identifiers/spec.md` (línea 987). No fue necesaria ninguna corrección de referencias cruzadas.
- Las deltas `MODIFIED RF-TRC-02` y `RF-TRC-04` se aplicaron sobre requisitos existentes de la spec viva.
- Traza de identificadores: `openspec/changes/archive/2026-11-26-migrar-ids-rf/ids.yaml`.
- Ajuste manual, una frase: el `## Purpose` de `openspec/specs/requirement-trace/spec.md` ya no dice que el mapa queda fuera; ahora indica que el mapa existe en `openspec/id-map.yaml` y que `axiom trace` lo usa como fuente de equivalencias (RF-TRC-16) a través del punto de extensión (RF-TRC-04).

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. No se usó `--supersede` ni `--superseded-requirements`, y ningún delta contiene `REMOVED`. Los requisitos `RF-TRC-02` y `RF-TRC-04` se modifican en su texto, no se superan.

## Índice Maestro
- `axiom archive sync --path <worktree>`: 49 especificaciones consolidadas, 516 requisitos activos, 2316 escenarios. `openspec/INDEX.md` regenerado.

## Contexto Técnico Mantenido
- No aplicable: el estado del cambio no proyecta `technicalContext.maintained`. No se ejecutaron `axiom archive context prepare` ni `verify`.

## Verificación de Archivado
- Archive-check (`axiom sdd archive-check`): exit 0; el cambio no es ISO.
- Movimiento de la carpeta: `git mv` a `openspec/changes/archive/2026-11-26-migrar-ids-rf`. El origen ya no existe en `openspec/changes/`.
- Readback `diff -r` (snapshot previo al movimiento frente al destino): salida vacía, `diff_status=0`.
- Pruebas localizadas: `go test -count=1 ./internal/idmigration ./internal/openspecdoc ./internal/reqid`: `ok` en los tres paquetes, exit 0.

## Artefactos Consolidados
- `proposal.md`
- `specs/requirement-identifiers/spec.md` y `specs/requirement-trace/spec.md` (deltas, integradas en la spec viva)
- `design.md`
- `tasks.md` (28/28 marcadas)
- `verify-report.md` (estado limpio en su momento, ver resumen)
- `coverage-review.spec.md` (12 huecos) y `coverage-review.design.md` (9 huecos), informativos
- `kickoff.yaml`
- `ids.yaml` (generado por `archive-integrate`)
- `archive-report.md` (este informe, añadido al archivo)

## Trazabilidad de Fuentes
- Fuentes leídas por ruta (locadores de sistema de ficheros, store `hybrid`): deltas y `tasks.md` del worktree de archivado, `verify-report.md`, y el `ids.yaml` generado.
- Observaciones de Engram localizadas con `mem_search` (proyecto `axiom`), solo como referencia de trazabilidad. No se usó su contenido como fuente en esta fase:
  - `sdd/migrar-ids-rf/proposal` (2651), `sdd/migrar-ids-rf/spec` (2652), `sdd/migrar-ids-rf/design` (2654), `sdd/migrar-ids-rf/tasks` (2657), `sdd/migrar-ids-rf/verify-report` (2660), `sdd/migrar-ids-rf/apply-progress` (2658), `sdd/migrar-ids-rf/explore` (2650), `sdd/migrar-ids-rf/coverage-review/spec` (2653), `sdd/migrar-ids-rf/coverage-review/design` (2655), `sdd/migrar-ids-rf/dominios` (2649), `sdd/migrar-ids-rf/estado` (2661).

## Pendientes Informativos y Riesgos Aceptados
- Huecos de cobertura (informativos, absorbidos en gran parte en `tasks.md`): 12 en la spec y 9 en el diseño, según `coverage-review.*.md`.
- Pendientes informativos del verify y de la cobertura, sin resolver en este cierre:
  - La traza por ID nuevo de un requisito sin ID antiguo no ve los incrementos que lo tocaron por título.
  - El aviso `legacy_ambiguous` no tiene escenario en la spec.
  - La huella de inmutabilidad del mapa es global (sha256).
  - El visor del dashboard (`internal/dashboard/assets/js/core/spec.js:25`) solo reconoce IDs `REQ`.
- `scripts/deadcode-ratchet.sh`: no verificado en local (requiere instalar un ejecutable que el EDR bloquea). Queda para CI.
- Provisionales residuales: `coverage-review.*.md` y `verify-report.md` conservan `prov:` en su texto histórico. Son informes de fase y no se reescriben.
