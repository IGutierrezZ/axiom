# tui-uninstall-preseleccion

Tipo: bug

## Objetivo

Que la desinstalación parcial del TUI, con la selección que ofrece por defecto, haga lo mismo que `axiom uninstall --agent <agente>` sin `--components` desde la PR #304:

- No retira GGA.
- En el ámbito global, conserva el registro MCP `engram` de cada agente.

## Problema y porqué

La PR #304 (`odd/tasks/uninstall-alcance.md`) corrigió el alcance en el CLI y dejó el TUI fuera de alcance. En el TUI, `defaultUninstallComponents` (`internal/tui/model.go`) marca todo `catalog.MVPComponents()`, GGA y Engram incluidos, y `startUninstall` pasa esa lista tal cual al servicio. Para el servicio, una lista con `gga` o `engram` es una petición explícita: con la selección por defecto, el TUI retira GGA (si no queda otro agente en `state.json`) y el registro MCP de Engram.

## Alcance

- Selección por defecto de la desinstalación parcial del TUI.
- Lo que el TUI pasa al servicio cuando la selección es la de por defecto.

Fuera de alcance:

- Los modos de desinstalación completa del TUI (Full, Full + binario, Clean Install), que marcan todos los componentes y siguen siendo explícitos.
- El servicio de desinstalación y el CLI.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run`, `go install` ni `go test ./...`.
- No se lanzan pruebas de `internal/app`, `internal/dashboard`, `internal/update`, `internal/opencode`, `internal/components/gga`, `bench` ni `e2e`. `internal/tui` sí se lanza, con `-run`.
- Push, PR y merge los decide el usuario.

## Decisiones

Engram tiene que seguir marcado por defecto. `stateAgentsToRemove` exige `fullAgentRemovalComponents`, que incluye Engram: sin Engram, el agente no sale de `state.json` y quedan las secciones `engram-protocol` del prompt. El servicio solo conserva el registro MCP cuando la lista de componentes no nombra `engram`.

Decisión del usuario (2026-11-25), opción «selección por defecto»:

- `defaultUninstallComponents` marca todo `catalog.MVPComponents()` menos GGA.
- En la desinstalación parcial, si la selección coincide como conjunto con la de por defecto, `startUninstall` pasa una lista vacía de componentes. Es lo mismo que el CLI sin `--components`: el servicio aplica `fullAgentRemovalComponents`, conserva el registro MCP de Engram y lo explica en `ManualActions`, que la pantalla de resultado ya muestra.
- Cualquier otra selección se pasa tal cual y es explícita: con Engram marcado, retira también el registro; con GGA marcado, retira GGA.
- La decisión depende solo de la selección visible, no de si la persona ha tocado las casillas.
- Limitación aceptada: para retirar el registro de Engram de un agente sin cambiar nada más, hay que usar el CLI (`--components engram`) o una desinstalación completa.

Alternativa descartada: una opción «Conservar / Retirar registro MCP» en la subpantalla de Engram. Es más explícita, pero toca la API del servicio, el modelo y más pantallas.

## TDD

Modo: desactivado (no hay configuración en `axiom.yaml` ni petición). Aun así, primero se escribe la prueba que reproduce el caso y se observa en rojo. Ejecutor: `go test ./internal/tui -run <patrón> -count=1`.

## Tareas

- [x] T1. Selección por defecto sin GGA y delegación al comportamiento por defecto del servicio. Ruta: inline (unidad pequeña, causa localizada).
  - Pruebas (`internal/tui/uninstall_default_selection_test.go`):
    - `TestDefaultUninstallComponentsExcludesGGA`: la selección por defecto no incluye GGA y sí Engram, también tras `withResetUninstallState`.
    - `TestStartUninstallPassesComponentsOnlyForNonDefaultSelection`: en modo parcial, la selección por defecto (en cualquier orden, y también por la variante con perfiles) pasa una lista vacía. La selección por defecto más GGA, solo Engram y el modo completo se pasan tal cual.
  - RED observado: la selección por defecto incluía `gga` y llegaba entera al servicio.
  - Arreglo (`internal/tui/model.go`):
    - `defaultUninstallComponents` omite GGA.
    - `uninstallComponentsForService` pasa una lista vacía cuando el modo es parcial y la selección coincide como conjunto con la de por defecto.
  - Evidencia:
    - `go test ./internal/tui -run 'Uninstall' -count=1` → `ok` (12,5 s).
    - `go test ./internal/tui/... -count=1` → `ok` (`internal/tui` 31,4 s; `internal/tui/screens` 5,6 s).
    - `go vet ./internal/tui` y `gofmt -l internal/tui/` limpios.

## Aceptación

- Con la selección por defecto del TUI, el servicio recibe una lista vacía de componentes.
- Con cualquier otra selección, recibe la lista tal cual.
- Las pruebas de desinstalación de `internal/tui` pasan.

## Impacto en la spec

Sin impacto en la spec

Motivo: alinea el TUI con el comportamiento del CLI que corrigió la PR #304, que tampoco cambió la spec viva.

## Progreso

- 2026-11-25: worktree creado, documento escrito.
- 2026-11-25: T1 hecha (el commit va en la sección «Commits»).

## Commits

- T1: `fix(tui): no preseleccionar GGA y delegar la selección por defecto de la desinstalación`.

## Siguiente paso

Abrir la PR cuando el usuario lo pida: `scripts/axiom-worktree.ps1 pr tui-uninstall-preseleccion -Lane odd`.
