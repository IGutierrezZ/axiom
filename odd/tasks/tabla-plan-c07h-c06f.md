# ODD: cerrar la tabla de seguimiento del plan maestro

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/tabla-plan-c07h-c06f.md`.
> Espejo de recuperación en Engram: topic `odd/tabla-plan-c07h-c06f/tasks`, proyecto `axiom`.

Tipo: docs

## Objetivo

Marcar como fusionados los últimos cambios del plan en la tabla de seguimiento (apartado 8) de `docs/plan/backlog-2026-11/README.md` y dejar el plan cerrado.

## Por qué

Las olas 14 (C-07-h) y 15 (C-06-f) están fusionadas y la tabla las mostraba «pendiente». El usuario autorizó estos ODD de documentación para tener el README al día («Sí, al acabar»), como #374 y #375.

## Alcance autorizado

- Filas C-07-h y C-06-f de la tabla de seguimiento: estado, rama y PR de código y de archive con el commit de fusión corto.
- Fila C-07-e: figuraba «en curso» pero está fusionada (#366 y #367); se corrige con la misma evidencia para que la tabla quede cerrada.
- Frase de estado sobre la tabla: fecha 2026-11-27 y plan completo.
- La tabla de olas (apartado 5) no tiene columna de estado: no se toca.

## Tareas

- [x] **T1 · Cerrar la tabla de seguimiento** (ruta: inline; unidad de documentación pequeña, sin disparadores de delegación)
  - C-07-h: `inc/ui-integraciones`, #376 (`ba0e5bbd`); archive #377 (`5bfede0c`).
  - C-06-f: `inc/puertas-desde-ui`, #381 (`9c86b86e`); archive #383 (`39b9ad94`).
  - C-07-e: `inc/eventos-ado`, #366 (`fa076cc3`); archive #367 (`63eaaab0`).
  - Evidencia: `gh pr view <n> --json headRefName,state,mergeCommit` para cada PR, todas `MERGED`.
  - Commit: `docs(plan): cerrar la tabla de seguimiento del plan maestro`.

## Criterio de aceptación

- Ninguna fila de la tabla queda en «pendiente» ni «en curso»; cada PR citada existe y está fusionada con ese commit.

## TDD

Desactivado: cambio solo de documentación. Comprobación funcional: `go test ./internal/workspace -run Doc`.

## Impacto en la spec

Sin impacto en la spec
Motivo: solo documentación del plan

## Verificación

- `go test ./internal/workspace -run Doc`: ok.
