# Reporte de Archivado Formal: identidad-git-actor

## Resumen de Cierre
- Fecha de Archivado: 2026-11-18
- Estado de Verificación: PASS (0 CRITICAL, 0 WARNING, 1 SUGGESTION) según `verify-report.md` en el momento de la verificación. Es una instantánea: se conserva como historial y no sustituye al estado final descrito aquí.
- Integración: compuerta `integration` aprobada con evidencia `pr_merged`, referencia `bddd7ee9` (PR #276 hacia `main`), verificada contra el grafo local de commits.
- Estado de tareas al cierre: 18/18 completadas (`tasks.md`: 18 casillas marcadas, 0 pendientes). Unidades U1 (1.1-1.6), U2 (2.1-2.3), U3 (3.1-3.7) y U4 (4.1-4.2).
- Commits de implementación observados en el repositorio: U1 `6720cf2d`, U2 `dc4f8984`, U3 `9c133770`, cierre de apply `697e9612`; fusión en `main` `bddd7ee9`.
- Destino en Archivo: `openspec/changes/archive/2026-11-18-identidad-git-actor/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo: la capacidad `change-actor-identity` es nueva y no reemplaza ni deprecia ningún requisito previo. Las capacidades `sdd-lane-kickoff` y `sdd-preflight-configuration` no cambian; `sealed_by: inferred` se conserva (REQ-42.4).

## Integración de Deltas en las Specs Vivas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| `change-actor-identity` | Creada | `openspec/specs/change-actor-identity/spec.md`: 9 requisitos (REQ-42.1 a REQ-42.9), todos nuevos. |

Procedimiento y trazabilidad:
- El delta `specs/change-actor-identity/spec.md` no tenía spec viva previa, así que no hubo composición con `axiom sdd-archive-compose`. Se copió mecánicamente con `cp` a `openspec/specs/change-actor-identity/spec.md` y la lectura de `diff -r` del momento de la copia fue vacía.
- Desviación deliberada respecto del contrato de copia: se retiraron de la cabecera del delta el título `# Delta para ...`, la marca `## ADDED Requirements` y la frase «Todo lo que sigue es `ADDED`», y se añadió el encabezado de capacidad `## 42. Capacidad: \`change-actor-identity\``, según el formato de las specs vivas. El cuerpo de los requisitos no se ha tocado. La diferencia entre el delta y la spec viva se limita a esas líneas (salida verbatim en la sección de verificación).
- `axiom archive sync --path <workspaceRoot>` regeneró `openspec/INDEX.md` (salida: 72 especificaciones consolidadas, 572 requisitos activos, 1759 escenarios BDD). El diff de `INDEX.md` añade la fila y el dominio de `change-actor-identity` y actualiza los totales de la cabecera.
- `axiom sdd integrate` está retirado (REQ-30.25) y no se ejecutó.

## Artefactos Consolidados
- `proposal.md`: presente (rastreado en git).
- `specs/change-actor-identity/spec.md`: presente (rastreado en git).
- `design.md`: presente (rastreado en git).
- `tasks.md`: presente (rastreado en git), 18/18 completadas.
- `exploration.md`, `kickoff.yaml`, `apply-progress.md`: presentes (rastreados en git).
- `verify-report.md`: presente. Es untracked en el worktree de archivado y se movió con la carpeta.
- `gates.yaml` (registro de compuertas, untracked) y `.gates.lock` (fichero de bloqueo vacío, untracked): presentes y movidos con la carpeta, para conservar la evidencia de la compuerta `integration`.

## Verificación Mecánica (diff -r)
Copia de la spec nueva desde el delta (`cp`), antes de la edición de cabecera:
```
(salida vacía, exit 0)
```

Movimiento de la carpeta (`cp -R` a snapshot previo, luego `git mv openspec/changes/identidad-git-actor openspec/changes/archive/2026-11-18-identidad-git-actor`, exit 0). Comprobación de que el origen ya no existe: `No such file or directory`. Readback del destino contra el snapshot previo al movimiento:
```
(salida vacía, exit 0)
```

Readback de la spec viva contra el delta (diferencias esperadas solo en la cabecera, según la desviación documentada arriba):
```
1c1
< # Delta para change-actor-identity
---
> ## 42. Capacidad: `change-actor-identity`
3c3
< > Capacidad nueva: no existe spec viva previa. Todo lo que sigue es `ADDED`. Ninguna otra capacidad cambia: ...
---
> > Capacidad nueva integrada en las specs vivas al archivar `identidad-git-actor` (2026-11-18). Ninguna otra capacidad cambia: ...
5d4
< ## ADDED Requirements
```
El cuerpo de los requisitos (REQ-42.1 a REQ-42.9) es idéntico byte a byte.

## Comprobaciones Ejecutadas (heredadas de la verificación)
| Comando | Resultado (según `verify-report.md`) |
|---|---|
| `go test ./internal/gitidentity/... ./internal/kickoff/... ./internal/gitcommon/...` | ok |
| `go test ./internal/cli/ -run "ActorIdentity\|SDDGate\|KickoffSeal\|Seal\|Gate\|PlanningHome\|WriterGuard"` | ok |
| `go test ./cmd/axiom/ -run "Handoff\|Guard\|Seal\|Gate\|Kickoff"` | ok |
| `go vet` sobre los paquetes tocados; `gofmt -l` | exit 0; salida vacía |
| `go test ./internal/cli/` completo | No ejecutado en Windows (prueba `review` ajena que no termina en esta máquina). Cubierto por la CI de la PR #276 (9 checks en verde, según el encargo). |
| `go test ./...` y `go vet ./...` globales | No ejecutados (fuera del alcance acordado). |

## Hallazgos y Advertencias No Resueltos
- SUGGESTION (sin acción bloqueante): REQ-42.9 define un escenario para la forma JSON de `kickoff seal --json`, pero la matriz de `tasks.md` solo asigna la prueba `TestGateRecordJSONShapeUnchanged`. No hay prueba nominal equivalente para `seal`. El código no añade campos a la salida de `seal`, así que es una laguna de prueba, no de comportamiento. Seguimiento recomendado para un cambio futuro.
- CRITICAL: ninguno. WARNING: ninguno.
- Trazabilidad de observaciones de Engram: las fases de este cambio operan en modo `hybrid` y sus localizadores son rutas de fichero; los artefactos se leyeron desde esos ficheros, no desde observaciones de Engram. No hay IDs de observación leídos que registrar.

## Trazabilidad
- Compuerta `integration`: `gates.yaml`, `evidence_kind: pr_merged`, `evidence_ref: bddd7ee9`, `evidence_base_ref: main`, `verified: true`.
- Implementación: `internal/gitidentity/gitidentity.go`, `internal/cli/sdd_actor_identity.go`, `internal/cli/sdd_gate.go`, `internal/cli/sdd_kickoff.go`, `internal/kickoff/types.go`, `internal/kickoff/args.go`.
- Especificación viva: `openspec/specs/change-actor-identity/spec.md`; catálogo: `openspec/INDEX.md`.
- Commits de archivado: pendientes; los realiza el orquestador en la rama `inc/identidad-git-actor-archive`.
