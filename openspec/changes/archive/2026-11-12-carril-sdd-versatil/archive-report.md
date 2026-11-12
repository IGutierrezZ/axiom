# Reporte de Archivado Formal: carril-sdd-versatil

## Resumen de Cierre
- Fecha de Archivado: 2026-11-12
- Estado de Verificación: Cumple, con verificación acotada. El `verify-report.md` archivado (en el momento de la verificación) registra 0 CRITICAL, 2 WARNING (ambos ajenos o fuera de alcance) y 3 SUGGESTION. No se ejecutó la suite completa de `internal/cli`.
- Estado de implementación: 46/46 tareas marcadas en `tasks.md` (leído del fichero archivado tras el movimiento: 46 casillas `[x]`, 0 `[ ]`). Compuerta `role-apply:fullstack` aprobada.
- Compuertas: `spec`, `design`, `tasks`, `role-apply:fullstack` e `integration` aprobadas. `integration` con evidencia `pr_merged` de la PR IGutierrezZ/axiom#256 (commit `b16ff42e`, ya en `main`).
- Destino en Archivo: `openspec/changes/archive/2026-11-12-carril-sdd-versatil/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Los deltas no contienen secciones `REMOVED` ni `RENAMED`, así que no se empleó `--supersede` ni `--superseded-requirements`. Los bloques `MODIFIED` (REQ-21.13, REQ-21.15, REQ-24.38, REQ-28.3, REQ-28.4 y REQ-28.6, según el verify-report) reemplazan el texto vigente sin deprecar requisitos. Incremento aditivo con modificaciones.

## Especificaciones Principales Actualizadas
Composición con `axiom sdd-archive-compose --canonical ... --delta ... --output ...compose-tmp && mv`. Cada comando salió con código 0.

| Dominio | Acción | Detalle |
|---------|--------|---------|
| `axiom-sdd-cli-integration` | Compuesto | ADDED |
| `organic-agent-trigger-rules` | Compuesto | ADDED |
| `sdd-archive-lifecycle` | Compuesto | ADDED |
| `sdd-independent-review` | Compuesto | MODIFIED |
| `sdd-lane-kickoff` | Compuesto | ADDED |
| `sdd-orchestrator-assets` | Compuesto | ADDED |
| `sdd-role-closure-handoff` | Compuesto | MODIFIED |
| `spec-baseline-integration` | Compuesto | ADDED y MODIFIED |
| `sdd-iso-evidence` | Creado (capacidad nueva) | Copia mecánica de un spec completo, sin secciones delta |

Tras las composiciones se regeneró el catálogo con `axiom archive sync`: 70 especificaciones vivas, 555 requisitos activos y 1684 escenarios BDD. `openspec/INDEX.md` queda modificado.

## Artefactos Consolidados
- `proposal.md`: presente.
- `specs/` (9 deltas): presentes.
- `design.md`: presente.
- `tasks.md`: presente, 46/46 casillas marcadas.
- `verify-report.md`: presente. Es un informe intermedio, no el estado final; sus afirmaciones son válidas solo en el momento en que se escribió.
- `kickoff.yaml`, `gates.yaml`: presentes. `gates.yaml` estaba modificado sin commit al empezar el archivado y se movió tal cual.
- `.gates.lock`: fichero vacío creado por `axiom sdd gate record`. Se movió junto con la carpeta porque el movimiento es de directorio completo (ver Pendientes).

## Pendientes y Hallazgos Abiertos
Estos puntos se conservan tal como constan en las fuentes, sin darlos por resueltos:
1. `.codex/AGENTS.md` está desfasada. Es una copia generada versionada, anterior al cambio. Queda fuera de alcance por decisión del usuario (tarea 24.2) y como seguimiento.
2. `internal/components/filemerge` falla en Windows (uniones de directorio), también en `main`. No se atribuye al cambio.
3. Sugerencias del verify, sin resolver en este archivado:
   - El design y la tarea 23.1 nombran 14 tokens; el código y la prueba fijan 13 (`isoevidence.AllCodes()`). Queda constancia de la discrepancia para el auditor.
   - El patrón `-run` con `TestOrchestratorNoDrift` de la tarea 21.2 no coincide con ninguna prueba.
   - `gates.yaml` aparecía modificado y `.gates.lock` sin seguimiento en el momento del verify. Ambos se movieron con la carpeta y siguen sin commit; no deben entrar en ningún commit de archivado sin revisión.
4. Suite de `internal/cli` completa no ejecutada (solo el subconjunto acotado del verify).

## Trazabilidad de Observaciones
- Almacén de los artefactos según el status previo al archivado: `openspec`. Se leyeron los ficheros del cambio en su locator de ruta; no se leyó ninguna observación de Engram, por lo que no hay IDs de observación que registrar como fuente.
- Tras el movimiento, `axiom sdd status` resuelve el cambio en Engram (`engram:sdd/carril-sdd-versatil`) y no lo proyecta como archivado; ver el informe de fase para el resultado observado.
