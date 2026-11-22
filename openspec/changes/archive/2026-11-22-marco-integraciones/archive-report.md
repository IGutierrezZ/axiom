# Informe de archivado: marco-integraciones (C-07-b)

Fecha de archivado: 2026-11-22.
Carril: SDD. Topología: `monorepo-embedded`. Almacén de artefactos: `openspec` (estado nativo), con copia en Engram `sdd/marco-integraciones/archive-report`. Rol: `fullstack`. Ejecución: `continuous`.

## Resumen

El cambio `marco-integraciones` queda archivado. Su código ya está fusionado en `main` mediante la PR #285 (merge `c0de4c0f`). En este archivado se integraron sus deltas en las especificaciones vivas, se creó la capacidad nueva `integration-plugins`, se movió la carpeta del cambio a `openspec/changes/archive/2026-11-22-marco-integraciones/` y se regeneró el catálogo `openspec/INDEX.md`. No se hizo ningún push, fetch, apertura de PR ni cambio en código Go.

Estado al cierre: las 63 tareas de `tasks.md` están marcadas (59 en el momento de la verificación; P.1 a P.4 se cerraron después, en esta rama de archivado, con su evidencia en `tasks.md`). El informe de verificación es `limpio`.

## Especificaciones integradas

Los cinco deltas modificados se compusieron con `axiom sdd archive-compose` (`--canonical` la spec viva, `--delta` la spec del cambio). El resultado se escribió primero en un directorio temporal, se comprobó y después se copió sobre la spec viva. La capacidad nueva se copió tal cual desde su delta.

| Capacidad | Requisitos vivos antes | Requisitos vivos después | Cambios integrados |
|---|---|---|---|
| `workspace-topology` | 15 | 18 | ADDED REQ-43.19, REQ-43.20, REQ-43.21 |
| `sdd-block-review-gates` | 16 | 18 | ADDED REQ-43.22, REQ-43.23 |
| `change-branch-lifecycle` | 32 | 34 | ADDED REQ-43.24, REQ-43.25 |
| `sdd-docs-publication` | 8 | 10 | MODIFIED REQ-27.5; ADDED REQ-43.26, REQ-43.27 |
| `sdd-archive-lifecycle` | 14 | 14 | MODIFIED REQ-21.18 |
| `integration-plugins` (nueva) | 0 | 22 | Capacidad completa: REQ-43.1 a REQ-43.18 y REQ-43.28 a REQ-43.31 |

Comprobaciones posteriores a la integración:
- Cada bloque de requisito del delta aparece literalmente en la spec viva resultante. Las únicas líneas del delta que no aparecen en la viva son el título, la nota introductoria y los encabezados de sección (`## ADDED Requirements`, `## MODIFIED Requirements`), que no forman parte de la especificación vigente.
- No hay requisitos duplicados ni marcadores de delta (`## ADDED`, `## MODIFIED`) en las specs vivas.
- No se usó `--supersede`: ningún requisito previo quedó marcado como `[SUPERSEDED]`. Las dos modificaciones (REQ-27.5 y REQ-21.18) reemplazan en su sitio el texto vigente anterior.

## Poda histórica (ODD-4.2)

Ninguno. Los requisitos previos modificados (REQ-27.5 y REQ-21.18) se sustituyeron por su nueva redacción mediante MODIFIED, sin marcarlos como superados. El resto del incremento es aditivo.

## Artefactos consolidados

- `proposal.md`
- `design.md`
- `tasks.md` (63 de 63 tareas marcadas al cierre)
- `verify-report.md` (informe de verificación, estado `limpio`)
- `kickoff.yaml` y `gates.yaml` (registro de compuertas; la compuerta `integration` consta como `approved` con evidencia `pr_merged` contra `c0de4c0f`)
- `explore.md`
- `specs/` (seis deltas: `integration-plugins`, `workspace-topology`, `sdd-block-review-gates`, `change-branch-lifecycle`, `sdd-docs-publication`, `sdd-archive-lifecycle`)
- `.gates.lock`: estado local con punto, movido con la carpeta pero no versionado

## Commits del cambio

Rango fusionado en `main` (PR #285, merge `c0de4c0f`):

- `206baaba` docs(sdd): planificar marco-integraciones (C-07-b)
- `1b0cac94` feat(integrations): manifiesto, puerto y registro cerrado con doble en memoria
- `d778ae8f` feat(workspace): bloque estricto integrations en axiom.yaml
- `a7fa39f7` feat(sdd): estado local con punto en la raíz de la carpeta archivada
- `3f1aba8f` feat(integrations): cola local, vínculos y despachador con reintento
- `0ca07551` feat(integrations): almacén de secretos con entorno y keyring
- `cc14fa7c` feat(cli): verbos axiom integrations sync, emit y secret
- `1eba4e99` feat(cli): eventos desde gate record, ensure --role y publish --archive
- `3d815c20` Merge remote-tracking branch 'origin/main' into inc/marco-integraciones (resolución de conflictos con el bloque `delivery:`)

## Pull request

PR #285, `inc/marco-integraciones` hacia `main`, fusionada con el commit `c0de4c0f`.

## Verificación

Fuente: `verify-report.md`, escrito durante la verificación. Es una instantánea de ese momento y no se reescribe.

- Resultado en el momento de la verificación: `**Estado:** limpio`, con 0 críticos, 0 advertencias atribuibles al cambio y 2 sugerencias.
- Sugerencia 1: solo 10 de los 31 REQ-43.x se citan por identificador en código o pruebas. Sigue abierta, no es bloqueante.
- Sugerencia 2: los checkboxes P.1 a P.4 estaban sin marcar en el momento de la verificación. Se cerraron en la rama de archivado, con evidencia en `tasks.md`.
- Comprobaciones ejecutadas en la verificación: `go vet ./...`, `gofmt -l .` (vacío) y pruebas dirigidas de `internal/integrations`, `internal/workspace`, `internal/hub`, `internal/kickoff`, `internal/docpaths`, `internal/branch`, `internal/sddstatus`, `internal/cli` (filtro de integraciones, gates, publish, archive y canarios) y `cmd/axiom` (ayuda de `branch ensure` e `integrations`).
- No se ejecutó `go test ./...` completo ni `./internal/cli` sin filtro. Los fallos ambientales conocidos (timeouts de review en `internal/cli` completo, dos pruebas dependientes del idioma, `internal/components/filemerge` con junctions, `internal/app` de OpenCode y `internal/components/mcp`) quedan sin comprobar en local; no se declaran verdes. La cobertura completa corresponde a la CI de la PR #285.

## Comandos ejecutados y resultados

- `axiom sdd status marco-integraciones --cwd <wt> --json`: `dependencies.archive: ready`, `nextRecommended: archive`, `blockedReasons: []`, `taskProgress` 63/63.
- `axiom sdd archive-check --cwd <wt> --change marco-integraciones`: código 0. El cambio no es ISO y no tiene hallazgos.
- `axiom sdd archive-compose` (cinco deltas sobre sus specs vivas, salida a directorio temporal): código 0 en las cinco capacidades. Comprobación de cada bloque y de duplicados: sin incidencias.
- Copia de `integration-plugins` desde el delta a la spec nueva: `diff -r` vacío.
- Copia de las cinco specs compuestas a las specs vivas: `diff -r` vacío en cada una.
- `axiom archive sync --path <wt>`: código 0. Regenera `openspec/INDEX.md` (73 especificaciones, 612 requisitos activos, 2040 escenarios tras la integración).
- `git mv` de `openspec/changes/marco-integraciones` a `openspec/changes/archive/2026-11-22-marco-integraciones`: código 0. La carpeta origen ya no existe.
- `diff -r` entre la instantánea previa al movimiento y la carpeta archivada: salida vacía, código 0.
- `go test -count=1 ./internal/workspace/ -run "Docs|Spec|OpenSpec"`: `ok` (3,207 s).
- `axiom sdd integrate verify`: no se ejecutó, porque la skill `sdd-archive` no lo indica para este cierre.

## Estado de Git

- Movimiento de la carpeta del cambio hecho con `git mv`.
- Commit único en la rama de archivado `inc/marco-integraciones-archive`. Sin atribución de IA, sin push, sin fetch y sin PR.
- `.gates.lock` queda fuera del commit: es estado local y nunca se versiona.

## Trazabilidad

- Artefactos leídos: el locador nativo de `axiom sdd status` (almacén `openspec`). No se leyó ninguna observación de Engram para este cierre, así que no hay identificadores de observación que registrar.
- Observación de Engram de este informe: `sdd/marco-integraciones/archive-report` (proyecto `axiom`).
