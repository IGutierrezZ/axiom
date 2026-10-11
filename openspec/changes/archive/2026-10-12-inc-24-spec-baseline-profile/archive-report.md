# Reporte de Archivado Formal: inc-24-spec-baseline-profile

## Resumen de Cierre
- Fecha de Archivado: 2026-10-12
- Estado de Verificación: Aprobado con observaciones. La verificación global (`verify-report.md`, instantánea del 2026-10-11, commit 0b4e97ca) registra 0 CRITICAL, 4 WARNING y 4 SUGGESTION. De los 79 requisitos, 77 se cumplen y 2 (REQ-24.12 y REQ-21.16) son parciales solo en prueba. Ver «Seguimientos abiertos».
- Estado de Implementación: 186/186 tareas marcadas en `tasks.md` (contadas en el fichero antes del movimiento). 0 tareas pendientes.
- Integración: PR #133 fusionado en `main` con el commit de merge `0bb668893ec00360071de6609c546a79387e0289` (2026-10-11). La compuerta `integration` está aprobada con evidencia `pr_merged`, verificada contra el grafo de commits con `--base-ref origin/main`.
- Baseline del proyecto: ninguna declarada (`baseline: None`). Se aplicó la rama por defecto de `sdd-archive`: composición nativa de `openspec/specs/<dominio>/spec.md` y `axiom archive sync`.
- Destino en Archivo: `openspec/changes/archive/2026-10-12-inc-24-spec-baseline-profile/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. El cambio no contiene secciones REMOVED ni RENAMED y no se usó `--supersede`. Los 17 requisitos MODIFIED reemplazan su bloque en el mismo documento; ningún requisito queda marcado como [SUPERSEDED / DEPRECADO].

## Dominios Integrados
Composición nativa (`axiom sdd archive-compose --canonical ... --delta ...`, salida 0 en los siete dominios):

| Dominio | Acción | Detalle |
|---------|--------|---------|
| workspace-topology | Actualizado | 4 MODIFIED; 7 requisitos antes y después |
| sdd-block-review-gates | Actualizado | 2 MODIFIED; 6 requisitos antes y después |
| living-documentation | Actualizado | 7 MODIFIED; 11 requisitos antes y después |
| multi-project-hub | Actualizado | 1 MODIFIED; 6 requisitos antes y después |
| sdd-archive-lifecycle | Actualizado | 2 MODIFIED y 3 ADDED; 3 requisitos antes, 6 después |
| organic-agent-trigger-rules | Actualizado | 1 ADDED; 13 requisitos antes, 14 después |
| tui-ui-parity | Actualizado | 1 MODIFIED; 10 requisitos antes y después |

Dominios nuevos (no existía especificación viva; copia mecánica con `cp`, `diff -r` vacío y `mv`):

| Dominio | Requisitos |
|---------|------------|
| planning-home-resolution | 12 |
| project-technical-context | 12 |
| spec-baseline-declaration | 15 |
| spec-baseline-integration | 19 |

Tras la integración, `axiom archive sync --path .` regeneró `openspec/INDEX.md` (63 especificaciones consolidadas, 433 requisitos activos, 885 escenarios BDD). El fichero `openspec/INDEX.md` queda modificado (+418/-12).

## Artefactos Consolidados
Archivados en la carpeta del cambio (movimiento con `git mv`):
- `proposal.md`, `design.md`, `tasks.md`, `verify-report.md`
- `specs/` con las 11 delta specs
- `kickoff.yaml`, `gates.yaml` (incluye la compuerta `integration`), `handoff.md`, `reanudacion.md`, `exploration.md`, `spike.md`, `spike-cwd-ejecutores.md`, `pr-description.md`
- Este `archive-report.md`, añadido después del readback de `diff -r`

No faltó ningún artefacto declarado en el status. El fichero `.gates.lock` era un bloqueo advisorio vacío (0 bytes, sin seguimiento en git) que la CLI crea durante las compuertas; se eliminó antes del movimiento y no entra en el commit.

## Hechos del Estado Final
Fuentes, por orden de prelación (`sdd-archive`, Final-State Authority):
1. `tasks.md` (fichero): 186 casillas `[x]`, 0 `[ ]`.
2. Hechos del orquestador en el encargo de archivado: PR #133 fusionado con el commit `0bb66889`; compuerta `integration` aprobada con `pr_merged`; `main` local sin cambios.
3. `verify-report.md` (instantánea de verificación): usada solo para el historial de hallazgos.

Aplicación a las instantáneas:
- `verify-report.md` §7 indica `dependencies.archive: blocked` y `nextRecommended: resolve-blockers`. Ese estado es anterior a la fusión del PR #133 y al registro de la compuerta `integration`. En el estado previo al archivado, `axiom sdd status` reportó `dependencies.archive: ready`, `nextRecommended: archive` y `blockedReasons: []`.
- W2 de `verify-report.md` (la rama estaba 4 commits por detrás de `main`, con un conflicto probable en `testdata/golden/sdd-opencode-multi-settings.golden` y en la constante SHA-256 de Kilocode): la fusión del PR #133 cierra la divergencia. Este archivado no repitió la verificación del golden ni de la constante, y el informe no registra cómo se resolvió el conflicto.
- W4 de `verify-report.md` (`.gates.lock` sin seguimiento): resuelto en el cierre, el fichero se eliminó antes del movimiento.

Cifras finales: las de `tasks.md` (186/186) y las de `axiom archive sync` (63 especificaciones, 433 requisitos, 885 escenarios). No se copiaron cifras de `verify-report.md`, salvo las de hallazgos citadas como historial.

## Contradicciones Registradas
- Gate `spec` (`gates.yaml`, decisión rechazada del 2026-10-04 14:31): pedía que archive sustituyera los IDs provisionales del cambio por los definitivos (REQ-24.59) y añadiera la opción `scenarios: include|omit`. La decisión aprobada posterior del mismo día (2026-10-04 14:50, y re-aprobación 21:43) recoge esas enmiendas. La sustitución de IDs de REQ-24.59 aplica al archivado con baseline declarada. Este archivado usó la rama por defecto (`baseline: None`), por lo que no hubo sustitución de IDs provisionales. Los requisitos ADDED se integraron por nombre, con el motor nativo.
- No hay otras contradicciones entre fuentes de igual o distinta prelación.

## Trazabilidad
- Observaciones de Engram leídas en esta fase: ninguna. Los artefactos se leyeron desde los ficheros indicados por el status (store `openspec`).
- Observación citada por `verify-report.md` y no leída en esta fase: `sdd/inc-24-spec-baseline-profile/apply-progress` (observación 562), instantánea de apply.

## Seguimientos Abiertos
No corregidos en este archivado; registrados como estaban en el informe de verificación o en el encargo de cierre.
- W1: escenario `pr_merged` del espacio canónico sin prueba con la fixture canónica (REQ-24.12 y REQ-21.16). Cumplido por construcción en `internal/cli/sdd_gate.go:63`.
- W3: límites de verificación en Windows. Las pruebas de enlaces simbólicos se omitieron; las 860 pruebas `review*_test.go` de `internal/cli` y el paquete `internal/reviewtransaction` completo no terminaron en esta máquina; `bench/` tiene 19 fallos de entorno idénticos a la base; `go test -race` no estaba disponible; la cobertura no se midió en porcentaje.
- S1: `pr-description.md` está desfasada (cita la cabeza `9f4bd977` y «61 commits»). El PR ya está fusionado.
- S2: REQ-4.2 y la parte de `app.js` de REQ-24.25 se verificaron por marcadores de texto, sin ejecutar JavaScript.
- S3: el tamaño real de las costuras de `internal/cli/sdd_archive_compose.go` (+397/-19) y de `cmd/axiom/main.go` (+161/-57) consta solo en la tarea 8.2.
- S4: REQ-24.16 no tiene etiqueta `REQ-` en el código y REQ-24.44 solo aparece en `compose.go`.
- Tamaño de la entrega: unas 29.400 líneas de autoría sin la fase 0, frente a 8.660–12.900 previstas, con `size:exception`.
- Observación posterior al archivado: `axiom sdd status inc-24-spec-baseline-profile --cwd .` ya no resuelve el cambio archivado (`changeRoot: engram:sdd/inc-24-spec-baseline-profile`, `blockedReasons: ["tasks.md has no markdown task checkboxes."]`), y con el nombre con prefijo de fecha devuelve `changeRoot: null`. Causa no diagnosticada en este archivado. El cierre no depende de ese comando.

## Verificación del Archivado
- `go test ./internal/sddstatus -run 'Compose' -count=1`: ok
- `go test ./internal/workspace -count=1`: ok
- `axiom workspace validate --path .`: Resultado: COMPLIANT
- Binario usado: compilado desde el worktree (`go build ./cmd/axiom`). El binario `axiom` instalado (versión `8cf608bd`) no tiene los modos `--cwd`, `--document` e `--in-place` de `archive-compose`.
