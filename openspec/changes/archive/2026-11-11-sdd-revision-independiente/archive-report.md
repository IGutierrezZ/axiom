# Reporte de Archivado Formal: sdd-revision-independiente

## Resumen de Cierre
- Fecha de Archivado: 2026-11-11
- Estado de Verificación: PASS WITH WARNINGS, según `verify-report.md` en el momento de la verificación (sin CRITICAL; 2 WARNING; 1 SUGGESTION). Estado final de los avisos en la sección «Avisos del verify al cierre».
- Integración: PR IGutierrezZ/axiom#249 (https://github.com/IGutierrezZ/axiom/pull/249) mergeada en `main` con merge commit `8c492396ce9a6515a1465b9c410936b65121b599` (mergedAt 2026-11-11T21:21:40Z). Compuerta `integration` aprobada con evidencia `pr_merged`.
- Destino en Archivo: `openspec/changes/archive/2026-11-11-sdd-revision-independiente/`
- Carril y modo: SDD, kickoff `continuous`, rol único `fullstack`, `handoffPolicy: none`, topología `monorepo-embedded`, artefactos en `openspec`.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento aditivo. No se usó `--supersede` ni `--superseded-requirements`. Los requisitos modificados de `sdd-lane-kickoff` se aplicaron por nombre de requisito con `MODIFIED`, sin superar requisitos previos.

## Especificaciones Vivas Tocadas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| `sdd-block-review-gates` | Actualizada | Composición con `axiom sdd archive-compose` |
| `sdd-lane-kickoff` | Actualizada | Composición con `axiom sdd archive-compose` (requisitos modificados por nombre) |
| `sdd-preflight-configuration` | Actualizada | Composición con `axiom sdd archive-compose` |
| `sdd-independent-review` | Creada | Copia mecánica de la delta spec (dominio nuevo) |
| `openspec/INDEX.md` | Regenerado | `axiom archive sync`: 69 especificaciones, 536 requisitos, 1487 escenarios |

## Artefactos Consolidados
- `proposal.md`: presente
- `specs/` (4 dominios): `sdd-block-review-gates`, `sdd-independent-review`, `sdd-lane-kickoff`, `sdd-preflight-configuration`
- `design.md`: presente
- `tasks.md`: presente, 36/36 tareas completadas, 0 pendientes
- `apply-progress.md`: presente
- `verify-report.md`: presente
- `kickoff.yaml`, `gates.yaml`, `integration.yaml`: presentes. `gates.yaml` registra la compuerta `integration` aprobada con evidencia `pr_merged`. `integration.yaml` es el plan de integración preparado; no contiene entradas de documentos.
- `.gates.lock`: fichero de bloqueo vacío presente en la carpeta archivada; no forma parte del contenido versionado (ver «Limitaciones»).

## Avisos del verify al cierre
Fuentes, por orden de rango: (1) `tasks.md` (36/36), (2) instrucciones de cierre del orquestador, (3) `verify-report.md` (instantánea).

- **W1 (vigente).** Los requisitos REQ-28.1, 28.2, 28.3, 28.6, 28.7, 28.8, 28.9, 28.10 y 28.14 son prosa de orquestador y de skills. Su cobertura es estática (pins de texto en `internal/assets/independent_review_boundary_test.go`, registro de secciones y goldens de proyección). No hay prueba de comportamiento en runtime. Sigue abierto tal como lo registró el verify.
- **W2 (mitigado, según el orquestador).** La suite completa (`internal/cli`, `internal/reviewtransaction`, `go test ./...`) no se ejecutó en el verify. Según las instrucciones de cierre, el CI `Unit Tests` pasó en la PR #249. Esta sesión de archivado no ejecutó ni reprodujo esa suite; la evidencia es la del CI indicada por el orquestador.
- **S1 (sin cambios).** La validación de la respuesta de texto de REQ-28.14 es prosa del orquestador sin prueba de ejecución; el hook está cubierto por `TestSDDPreflightHookIgnoresIndependentReviewPreferenceInteraction`.

Ningún dato del verify se presenta como estado actual salvo lo indicado arriba. Las pruebas citadas por el verify corresponden al momento de la verificación.

## Comprobaciones registradas por el verify (instantánea)
- `go vet ./...`: exit 0 (en el verify).
- `go test` acotado a `internal/assets`, `internal/kickoff`, `internal/sddstatus`, `internal/multirole`, `internal/components/sdd`: ok (en el verify).
- `go test ./internal/cli/ -run` con el patrón de la instrucción: ok (en el verify).
- Fallos ajenos, preexistentes o de Windows, citados por el verify y no ejecutados en esa sesión: `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`, `internal/backup` `TestSnapshotterSkipsUnixSockets`, `internal/components/filemerge` `TestWriteFileAtomicFollowsDirectoryJunction*`.

## Trazabilidad de Engram
- Artefactos leídos en esta fase desde `openspec` (store declarado): `proposal.md`, `specs/*`, `design.md`, `tasks.md`, `verify-report.md`. No se consultaron observaciones Engram de los artefactos.
- Observación previa `sdd/sdd-revision-independiente/archive-report` (id 2176): reporte del intento bloqueado por `blocked(wrong_branch)`, en el que el cambio NO estaba archivado. Este reporte la sustituye (upsert por `topic_key`).

## Traza de integración
- `axiom sdd integrate prepare`: plan preparado en `2026-11-11T21:22:47Z`, baseline por defecto, sin documentos de baseline ni contexto técnico mantenido (`context.decision: none`), `documents: []`.
- `axiom sdd integrate verify`: rechazado con el mensaje «el cambio ya está archivado en sdd/sdd-revision-independiente/archive-report», por la existencia de la observación 2176 (ver «Limitaciones»). No se ha obtenido un resultado PASS de `integrate verify`.
- `axiom sdd integrate show`: «Verificada: pendiente. La integración no registra entradas.»

## Limitaciones
- `axiom sdd integrate verify` no pudo ejecutarse con éxito: el comando rechaza un cambio cuyo reporte de archivado ya existe en Engram, y la observación 2176 existía antes de este cierre. La traza de integración queda con «Verificada: pendiente».
- `.gates.lock` (fichero vacío) quedó dentro de la carpeta archivada y no se incluye en el commit.
- Sin push ni PR: los realiza la persona responsable, según la política del repositorio.
