# ODD: tabla de seguimiento del plan maestro al día

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/tabla-seguimiento-plan.md`.
> Espejo de recuperación en Engram: topic `odd/tabla-seguimiento-plan/tasks`, proyecto `axiom`.

## Objetivo

Poner al día el apartado 8 («Seguimiento») de `docs/plan/backlog-2026-11/README.md`, congelado con todos los cambios en «pendiente», al terminar la ola 11.

## Por qué

El seguimiento real vive en Engram (`plan-backlog/seguimiento`, #2353) por la regla de no tocar el README para seguimiento. El usuario autorizó una excepción puntual al acabar la ola 11 («Sí, al acabar»).

## Alcance autorizado

- Tabla del apartado 8: estado (`fusionado`, `en curso` o `pendiente`), rama y PR de código y de archive con el commit de fusión corto. Columnas sin cambios.
- Nota bajo la tabla con las PR fusionadas fuera del catálogo.
- Una línea «Revisión 11 (2026-11-26): tabla de seguimiento al día» en el registro de revisión (apartado 9).
- Nada más del plan.

## Restricciones

- Sin datos inventados: un cambio que no aparece ni en Engram ni en GitHub queda `pendiente`.
- Fuentes: Engram #2353 y `gh pr list --state merged --limit 200`; ramas cruzadas con los slugs del catálogo.

## Tareas

- [x] **T1 · Tabla de seguimiento al día** (ruta: inline; unidad de documentación pequeña, sin disparadores de delegación)
  - 44 cambios fusionados, C-07-e en curso (worktree `inc/eventos-ado`, sin PR), C-06-d, C-06-f, C-07-f, C-07-g y C-07-h pendientes.
  - Nota con las 4 PR de revisión del plan y 27 PR sueltas.
  - Commit: el único de la rama `inc/tabla-seguimiento-plan` (`docs(plan): poner al día la tabla de seguimiento del plan maestro`).

## Criterio de aceptación

- Cada fila del catálogo tiene un estado coherente con GitHub; cada PR citada existe y está fusionada con ese commit.
- Las pruebas de documentos del paquete `workspace` siguen en verde.

## TDD

Desactivado: cambio solo de documentación. Comprobación funcional: `go test ./internal/workspace -run Doc`.

## Impacto en la spec

Sin impacto en la spec
Motivo: solo documentación del plan

## Verificación

- `go test ./internal/workspace -run Doc`: ok.
- `go test ./internal/idmigration/...`: ok (sin citas `REQ-n.m` nuevas).
- Ninguna prueba lee el README real del plan; solo fixtures.
