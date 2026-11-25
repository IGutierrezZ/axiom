# ODD: orden copiable en la cabecera y roles en el paso Apply (detalle-chip-roles)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/detalle-chip-roles.md`. Espejo Engram: `odd/detalle-chip-roles/tasks`.
> Rama: `inc/detalle-chip-roles` · worktree `C:\repos\axiom-wt\detalle-chip-roles\axiom`.

## Objetivo

Quitar del dashboard la tarjeta «Siguiente paso» del detalle de incremento y el botón «Continuar» de Inicio, con su endpoint. A cambio, la orden de agente que toca se muestra como un chip copiable en la cabecera del detalle y en cada fila de «Necesita atención». En los cambios multirol, el paso Apply del stepper pasa a mostrar el avance de cada rol.

## Problema

- La tarjeta repetía lo que ya dice el stepper («Listo para el diseño» cuando Design es el paso actual).
- El botón «Continuar con Axiom» ejecutaba `axiom sdd continue` y volcaba en una consola el prompt del despachador, que está pensado para un agente y no para una persona. Además podía escribir el marcador de instancia del cambio sin que se viera.
- Lo único útil era la orden copiable (`/sdd-continue <cambio>`, `/sdd-apply <cambio>`…) y, en multirol, saber qué rol bloquea la barrera. Ninguna de las dos cosas necesita una tarjeta.

## Decisiones

- **Quitar todo** (persona usuaria, 2026-11-25): se van la tarjeta del detalle, el botón y la consola de Inicio, `POST /api/increments/continue` y `Service.ContinueIncrement`. El requisito REQ-15.2 de `dashboard-sdd-orchestration` se retira con un delta `REMOVED`.
- **Chip en la cabecera** (persona usuaria): la orden va en la fila de etiquetas del detalle, junto a `Fase: …`. Se elige frente a ponerla bajo el paso actual del stepper, donde una orden larga rompe el layout en móvil.
- **Roles en Apply** (persona usuaria): solo con más de un rol en `barrier_report.roles`. Con un solo rol el paso queda como antes. Los datos ya llegan al detalle (`BarrierDTO.roles[].tasks`, `apply_done`, `verify_done`), así que no se toca el backend.
- **Una sola fuente de la orden**: `nextCommand(summary)` en `increment.js` sustituye a `deriveNextStep` y la usan el detalle e Inicio. Las reglas de prioridad se mantienen: verificación global → planificación → roles esperando → archivar → apply. Un cambio archivado no tiene orden.

## Alcance autorizado

- `internal/dashboard/assets/` (`index.html`, `style.css`, `js/core/increment.js`, `js/core/components.js`, `js/screens/detalle.js`, `js/screens/inicio.js`).
- `internal/dashboard/server.go`, `service.go` y sus pruebas (`dashboard_test.go`, `planning_home_test.go`, `secrets_canary_test.go`, `workflow_assets_test.go`, `home_assets_test.go`, `assets_test.go`).
- `docs/redesign/README.md` (tablas de Inicio y del detalle).
- Delta `REMOVED` de REQ-15.2 en `## Spec viva` e integración con `axiom archive integrate --odd detalle-chip-roles`.

Fuera de alcance: `axiom sdd continue` por CLI (no cambia), los mockups históricos de `docs/redesign/mockups/` y la pestaña «Roles y barrera».

## Modo TDD

Desactivado: `axiom.yaml` no declara TDD y nadie lo ha pedido. Se ejecutan las comprobaciones funcionales habituales: `go test -count=1 ./internal/dashboard/` (sin binarios sueltos, por Cortex XDR) y una comprobación visual en el navegador con el dashboard servido en proceso.

## Tareas

- [x] **T1 · Quitar la tarjeta, el botón de Inicio y el endpoint; añadir el chip** (ruta: en línea; unidad cohesiva de un solo comportamiento, sin paralelismo ni presión de contexto)
  - `nextCommand` y `commandChip`; chip en `#inc-meta` y en las filas de atención en lugar del botón y del recordatorio.
  - Fuera: `renderNext`, `runContinue` (detalle e Inicio), `continueIncrement`, `deriveNextStep`, la tarjeta y su consola en `index.html`, el hueco de consola de Inicio, el CSS que queda huérfano, el endpoint y `ContinueIncrement`.
  - Pruebas de assets y de servidor al día; delta `REMOVED` de REQ-15.2 y tablas de `docs/redesign/README.md`.
  - Comprobación: `go test -count=1 ./internal/dashboard/`; `gofmt -l` y `go vet ./internal/dashboard/`.
- [ ] **T2 · Avance por rol dentro del paso Apply** (ruta: en línea; misma razón)
  - `detailSteps` añade `roles` al paso Apply cuando hay más de un rol, y el estado `waiting` cuando `barrier_report.satisfied` es falso y el paso es el actual.
  - `renderStepper` pinta una fila por rol (nombre, barra y `n/m` o «verificado» o «sin tasks»), el primer bloqueo como texto y un enlace a la pestaña «Roles y barrera».
  - Prueba de assets que fija el marcado; comprobación visual con un cambio multirol de prueba.
  - Comprobación: `go test -count=1 ./internal/dashboard/`.

## Criterio de terminado

- Pruebas del paquete `internal/dashboard` en verde.
- En el navegador: el detalle no muestra la tarjeta, el chip copia la orden y el paso Apply enseña los roles en un cambio multirol.
- Un commit por tarea en `inc/detalle-chip-roles`, con la integración de la spec en el último.

## Entrega

Una sola PR (`single-pr`), con una previsión de unas 350-450 líneas, en su mayoría borradas. Push, PR y fusión los decide la persona usuaria.

## Progreso

- 2026-11-25: worktree creado y avanzado a `origin/main` (`3c55c493`); documento creado.
- 2026-11-25 · T1 hecha. También se retiraron `reminder()` y `consoleBox()` de `components.js`, que se quedaron sin uso; la clase `.reminder-cmd` se mantiene porque la usa el informe de adopción (`projects.js`). Un POST a la ruta retirada cae en el servidor de ficheros y recibe 405, y la prueba acepta 404 o 405. Evidencia: `gofmt -l` vacío, `go vet ./internal/dashboard/` sin avisos, `go test -count=1 ./internal/dashboard/` → `ok` (110 s).

## Spec viva

### Capacidad: dashboard-sdd-orchestration

```openspec
## REMOVED Requirements

### Requirement: Ejecución y Avance de Fase SDD vía API (REQ-15.2)

(Reason: el dashboard ya no ejecuta `axiom sdd continue`. La salida era el prompt del despachador, pensado para un agente, y el botón no hacía avanzar ninguna fase. El detalle de incremento e Inicio muestran en su lugar la orden de agente copiable de la fase que toca. `axiom sdd continue` sigue disponible en la CLI.)
```
