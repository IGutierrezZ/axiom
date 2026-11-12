# CodeGraph: detectar el índice ausente y ofrecer su inicialización

- Rama: `inc/codegraph-indice` (worktree `axiom-wt/codegraph-indice/axiom`).
- TDD: desactivado (sin configuración en `axiom.yaml` ni elección explícita); se ejecutan comprobaciones funcionales ordinarias. Runner: `go test`.
- Contexto técnico mantenido: no declarado; sin requisito de documentación adicional.
- Entrega: `exception-ok` (una PR con `size:exception` si supera 400 líneas).

## Objetivo

Que la pantalla «Código y grafo» distinga «hay un MCP de CodeGraph registrado» de «el workspace tiene índice», y que, cuando falte el índice, el botón ofrezca inicializarlo en lugar de lanzar `codegraph index` y fallar.

## Problema

- `CheckCodeGraphConfigured` (`internal/semantic/detector.go`) solo comprueba si algún agente, del workspace o global, registra el MCP de CodeGraph; el panel lo rotula «Configurado en el workspace».
- `ReindexCodeGraph` (`internal/semantic/service.go`) ejecuta `codegraph index` sin comprobar `.codegraph/`; sin `codegraph init` previo falla con `CodeGraph not initialized`.

## Por qué

El panel afirma un estado que no existe y la única acción disponible falla sin salida útil.

## Alcance autorizado

- `internal/semantic/`: detección del índice (`<workspaceRoot>/.codegraph` como directorio), campo `codegraph_indexed` en el estado y operación `init` o `index` según el disco.
- `internal/dashboard/assets/js/screens/semantic.js` (y `index.html` si hace falta): rótulo honesto de la pastilla de registro, pastilla de índice y botón «Inicializar CodeGraph» cuando falte el índice.
- Tests de `internal/semantic` e `internal/dashboard` afectados.

Fuera de alcance: cambiar `axiom codegraph init` de la CLI, el tiempo límite de 60 s y la configuración de Serena.

## Restricciones

- `codegraph init` construye también el índice inicial (CodeGraph 1.6.2); en el panel debe ejecutarse no interactivo (`--yes`), con el workspace como ruta.
- La inicialización la dispara siempre un clic explícito; el estado nunca la lanza solo.

## Tareas

- [x] T1 — Backend: `CodeGraphIndexed` en el estado y `ReindexCodeGraph` que ejecuta `codegraph init --yes <root>` sin índice y `codegraph index` con índice; el resultado informa la operación. Tests en `internal/semantic`. Ruta: delegada (escritor único; toca 3+ ficheros no triviales).
- [ ] T2 — Panel: pastilla «Registrado en un agente» / «Sin registrar en agentes», pastilla de índice en la tarjeta de CodeGraph y botón con rótulo y textos de progreso según el índice. Tests del panel afectados. Ruta: delegada (mismo escritor que T1).

## Criterios de aceptación

- Sin `.codegraph/`, el estado devuelve `codegraph_indexed: false`, el botón dice «Inicializar CodeGraph» y al pulsarlo se ejecuta `init`.
- Con `.codegraph/`, el botón reindexa con `index` como hoy.
- Ninguna pastilla afirma «workspace» para un registro que puede ser global.

## Comprobaciones

- `go test ./internal/semantic/... ./internal/dashboard/...`
- `go vet ./internal/semantic/... ./internal/dashboard/...`

## Progreso

- Plan stop: approved 2026-11-12; implementation continues
