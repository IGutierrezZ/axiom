# Reporte de Archivado Formal: adopcion-ids

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: limpio. Según `verify-report.md` (momento de la verificación): 0 CRITICAL, 0 WARNING, 2 SUGGESTION, `**Estado:** limpio`.
- Tareas: 37 de 37 marcadas como completadas en `tasks.md`; 0 pendientes.
- Integración con main: la PR #335 se fusionó con el merge `11fb90ff`. Hubo conflicto con C-08-l (#332) y C-08-h (#333), resuelto con un merge de `origin/main` que acepta el borrado de los pins de redacción (ver «Integración de main» en `tasks.md`).
- Destino en Archivo: `openspec/changes/archive/2026-11-26-adopcion-ids/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno marcado como `[SUPERSEDED]`. La delta no trae secciones REMOVED y no se usó `--supersede`. Los requisitos afectados son MODIFIED y sustituyen su texto vivo: REQ-30.9, REQ-30.10, REQ-30.11, REQ-30.12, REQ-30.13 y REQ-30.31 en `openspec-adoption` y `sdd-orchestrator-assets`, y REQ-44.4 en `requirement-identifiers`.

## Identificadores Asignados
Ninguno. Sin identificadores provisionales asignados en esta integración (`assignments[]` vacío). Los literales de provisional que aparecían en `tasks.md` y en la delta eran ejemplos de documentación; se neutralizaron antes de integrar (ver «Neutralización de tokens»).

## Integración de Specs (`axiom sdd archive-integrate`)
- `axiom sdd archive-check --change adopcion-ids --json`: exit 0, `ok: true`, sin hallazgos.
- `axiom sdd archive-integrate --change adopcion-ids --check --json`: exit 0, `written: []`, sin bloqueos.
- `axiom sdd archive-integrate --change adopcion-ids --json`: exit 0. `written`: `openspec/specs/openspec-adoption/spec.md`, `openspec/specs/requirement-identifiers/spec.md`, `openspec/specs/sdd-orchestrator-assets/spec.md`. Las tres capacidades quedan en `outcome: composed`.
- Requisitos modificados (7): REQ-30.9, REQ-30.10, REQ-30.11, REQ-30.12, REQ-30.13 (`openspec-adoption`); REQ-44.4 (`requirement-identifiers`); REQ-30.31 (`sdd-orchestrator-assets`). No hubo requisitos creados.
- `duplicates[]`: 13 identificadores históricos repetidos entre capacidades (REQ-1.1 a REQ-4.2), con `blocking: false`. Son datos informativos, no bloquean el archivado y no pertenecen a este cambio.

## Correcciones de Archivado (a mano, tras integrar)
Aplicadas según «Notas para el archive» de `tasks.md`:
1. `openspec/specs/openspec-adoption/spec.md`, sección «No cubre»: la línea de identificadores funcionales se sustituye por el texto que fija la delta.
2. `openspec/specs/openspec-adoption/spec.md`, fila REQ-30.12 de la tabla: resumen sustituido por «Los IDs del cliente se conservan según el patrón declarado; sin IDs, provisionales».
3. `openspec/specs/openspec-adoption/spec.md`, escenario de separadores mixtos: `RF.001` pasa a `RF_001`, porque el escáner solo admite `-` y `_`.
4. `openspec/specs/requirement-identifiers/spec.md`, `## Purpose`: se quita la excepción temporal de `adoptcheck` con `archive adopt` y la mención a C-03-c.

Restos revisados con `rg -n "hasta C-03-c|RF\.001|serie nueva" openspec/specs`: quedan solo usos intencionados. Son la anotación «(Previously: ...)» de un requisito modificado, escenarios negativos que prohíben la excepción, la expresión «serie nueva» como concepto del patrón y el texto de `odd-living-spec-integration`. Ninguno es una afirmación vigente de la excepción.

## Neutralización de Tokens
Antes de integrar, el orquestador sustituyó en `tasks.md` y en `specs/openspec-adoption/spec.md` los ejemplos literales de provisionales por ordinales simbólicos: `prov::1` → `prov::<k>`, `prov:40:1` → `prov:40:<k>`, `prov:40:2` → `prov:40:<k+1>`, `prov:a:1` → `prov:a:<k>`, `prov:30:1` → `prov:30:<k>`. Así el verbo no los trata como provisionales reales. El cambio es solo de notación de ejemplo; no altera el comportamiento descrito ni las pruebas.

## Hallazgo (seguimiento propuesto)
`axiom sdd archive-integrate` escanea todo `tasks.md` y la prosa de las deltas, no solo los encabezados de requisito. Un ejemplo literal con forma `prov:<clave>:<n>`, incluso dentro de backticks, bloquea el archivado con `provisional_malformed`. Propuesta: que el escáner ignore los spans de código o restrinja la búsqueda de provisionales a los títulos de requisito de la delta.

## Seguimiento Propuesto
- Colisión por cadena en `internal/livespec/ids.go:427` (riesgo R3 del design, SUGGESTION de verify): con `RF-<m>`, un `RF-001` frente a un `RF-1` vivo no se detecta al archivar. Proponer para C-03-d o para un cambio de `livespec`.
- Hallazgo del escáner de `archive-integrate` descrito arriba.

## Artefactos Consolidados
- `proposal.md`
- `explore.md`
- `specs/` (delta de `openspec-adoption`, `requirement-identifiers` y `sdd-orchestrator-assets`)
- `design.md`
- `tasks.md`
- `verify-report.md`
- `coverage-review.spec.md`, `coverage-review.design.md`
- `kickoff.yaml`
- `archive-report.md` (este documento; añadido tras el archivado)

## Trazabilidad
- Store: `openspec` (lectura de ficheros; no se leyeron observaciones de Engram en esta fase).
- Movimiento: `git mv openspec/changes/adopcion-ids openspec/changes/archive/2026-11-26-adopcion-ids`. Readback `diff -r` contra el snapshot previo al movimiento: salida vacía, exit 0.
- Índice maestro: `axiom archive sync --path` (exit 0): 48 especificaciones, 496 requisitos activos, 2225 escenarios; `openspec/INDEX.md` regenerado.
