# uninstall-alcance

Tipo: bug

## Objetivo

Que `axiom uninstall --scope global --agent <agente>` retire solo lo que es de ese agente: ni la huella de máquina de GGA ni el registro MCP de usuario de Engram, salvo petición explícita (`--components gga|engram`) o `--all`. Y que `axiom doctor` avise cuando el Engram que usaría Claude Code solo está en un `.mcp.json` sin aprobar.

## Problema y porqué

El 2026-11-25, `axiom uninstall --scope global --agent claude-code` (para dejar Claude Code solo con la instalación de workspace) hizo dos cosas no pedidas:

1. Retiró ficheros de GGA (`~/bin/gga.cmd`, `~/bin/gga.ps1`, `~/.local/share/gga/lib`) y dejó GGA a medias.
2. Retiró `mcpServers.engram` de `~/.claude.json`. En el repositorio quedó solo el `engram` de `.mcp.json`, que Claude Code tenía sin aprobar (`.claude/settings.local.json` → `enabledMcpjsonServers: ["context7"]`): las sesiones arrancaban sin Engram.

Además, `axiom doctor` (`engram:reachable`) dio OK leyendo ese `.mcp.json`, aunque Claude Code no lo cargaba.

Causas en el código:

- `partialUninstall` (`internal/components/uninstall/service.go`) sustituye una lista de componentes vacía por `allManagedComponents`, que incluye `ComponentGGA`; `fullAgentRemovalComponents` (lo que exige `stateAgentsToRemove` para sacar al agente de `state.json`) no lo incluye. `reconcileSharedComponents` solo protege GGA si quedan otros agentes en `state.json`, y en el caso real Claude Code era el único.
- `componentOperations`, caso `ComponentEngram`, en el ámbito global llama a `engramOperations`, que para Claude Code reescribe `~/.claude.json` quitando `mcpServers.engram` (y para el resto de agentes quita su entrada `engram` o el fichero que la contiene). No distingue una petición explícita de la lista por defecto.
- `checkEngramReachable` (`internal/cli/doctor.go`) prueba los comandos de `engram.ReadPersistedProjectStdioCommands`, que lee `<raíz>/.mcp.json` sin mirar la aprobación de Claude Code.

## Alcance

- Desinstalación parcial global sin `--components`: GGA fuera de la lista por defecto; registro MCP de Engram conservado.
- Aviso de `engram:reachable` cuando el Engram de Claude Code solo está en `.mcp.json` y sin aprobar.

Fuera de alcance:

- El TUI (`internal/tui`): su pantalla de componentes preselecciona todo `catalog.MVPComponents()`, GGA y Engram incluidos, y la persona ve y puede desmarcar cada uno; esa selección cuenta como explícita. Queda como posible seguimiento.
- La desinstalación con ámbito workspace: retira el `engram` del `.mcp.json` del proyecto, que es de esa instalación.
- Aprobar servidores en nombre del usuario: el doctor solo lee.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run`, `go install` ni `go test ./...`.
- No se lanzan pruebas de `internal/app`, `internal/dashboard`, `internal/update`, `internal/opencode`, `internal/components/gga`, `bench` ni `e2e`. `internal/cli` se lanza troceado con `-run`.
- Push, PR y merge los decide el usuario.

## Decisiones

- **GGA (T1).** La lista por defecto de `partialUninstall` pasa a ser `fullAgentRemovalComponents`, que ya es la definición de «retirar un agente entero» que usa `stateAgentsToRemove`. GGA solo entra si se nombra (`--components gga`) o con `--all` (`CompleteUninstall` sigue usando `allManagedComponents`). `reconcileSharedComponents` no cambia: sigue protegiendo un `--components gga` explícito mientras otro agente lo use.
- **Engram (T2).** Evidencia: el único sitio que retira el registro es `componentOperations` → `engramOperations` / `managedMCPOperations` en el caso `ComponentEngram`. La huella del doctor (`doctor_footprint.go`) ya trata Engram como elemento «declarado» de máquina, y Engram separa la memoria por proyecto, así que su registro de usuario no es de un agente concreto. Decisión: en una desinstalación parcial global en la que la persona no nombra `engram`, se conserva la entrada MCP `engram` de cada agente; se retira con `--components engram` o con `--all` (`--purge-state` exige `--all`). El resto de lo que escribe el componente Engram se sigue retirando: las secciones `engram-protocol` del prompt de sistema, el fichero heredado `~/.claude/mcp/engram.json` y, en Codex, las claves de instrucciones y multiagente de `config.toml` y sus ficheros `engram-instructions.md` / `engram-compact-prompt.md`. Así se cumple la spec viva (`engram-protocol-injection`: no dejar activos de protocolo huérfanos) y el agente conserva el servidor MCP. `Result.ManualActions` lo explica con una nota «No action needed».
  - Alternativa descartada: quitar `ComponentEngram` de la lista por defecto, como GGA. Rompe `stateAgentsToRemove` (el agente no saldría de `state.json`) y deja las secciones de protocolo en el prompt.
- **Doctor (T3).** Si Claude Code está instalado, `~/.claude.json` no declara `mcpServers.engram` y el proyecto sí lo declara en `.mcp.json`, se comprueba la aprobación: aprobado si algún `settings` (proyecto `.claude/settings.local.json` y `.claude/settings.json`, usuario `~/.claude/settings.json`) tiene `enableAllProjectMcpServers: true` o `engram` en `enabledMcpjsonServers`, salvo que `engram` esté en `disabledMcpjsonServers`. Sin aprobación, `engram:reachable` pasa de `pass` a `warn` y lo dice; un fallo de la sonda sigue siendo `fail`. Solo lectura: el remedio indica cómo aprobarlo, nunca escribe.
  - Limitación conocida: no se leen aprobaciones antiguas guardadas en `~/.claude.json` (`projects[<ruta>]`); en ese caso el aviso puede ser un falso positivo, pero no un fallo.

## TDD

Modo: desactivado (sin configuración en `axiom.yaml` ni petición). Aun así, cada tarea escribe primero la prueba que reproduce el caso y se observa en rojo antes del arreglo. Ejecutor: `go test` por paquete, `-count=1`.

## Tareas

- [x] T1. GGA fuera de la desinstalación parcial por defecto. Ruta: inline (unidad pequeña, causa localizada).
  - Prueba: `TestPartialUninstallWithoutComponentsKeepsGGAOnLastAgent` (Claude Code único agente, sin `--components`): GGA se conserva y el agente sale de `state.json`. RED observado: borraba los cuatro ficheros de GGA.
  - Arreglo: la lista por defecto de `partialUninstall` es `fullAgentRemovalComponents`.
  - **Aislamiento de pruebas (hallazgo).** En Windows `gga.ConfigPath` resuelve `%APPDATA%\gga\config` y no cuelga de `homeDir`: las pruebas de GGA del paquete que no fijan `APPDATA` escribían y borraban la configuración real de GGA de quien las ejecuta. La primera ejecución en rojo de esta prueba lo hizo (creó y borró `%APPDATA%\gga\config`); la copia de la desinstalación real de hoy (`~/.axiom/backups/20261125100156…`) registra ese fichero como `"existed": false`, así que no había nada que perder. Nuevo `TestMain` del paquete que fija un `APPDATA` temporal, y la prueba nueva fija el suyo con `t.Setenv`.
  - Evidencia: `go test ./internal/components/uninstall/ -count=1` → `ok` (14,8 s); `%APPDATA%\gga` sigue sin existir tras la ejecución.
- [x] T2. Conservar el registro MCP de usuario de Engram en la desinstalación parcial global. Ruta: inline.
  - Pruebas (`engram_registration_test.go`): `TestPartialUninstallGlobalKeepsTheUserEngramRegistrationUnlessNamed` (Claude Code: sin `--components` conserva `mcpServers.engram` y avisa; con `--components engram` lo retira; en ambos casos retira la sección `engram-protocol` de `CLAUDE.md` y respeta otros servidores) y `TestPartialUninstallGlobalCodexKeepsTheEngramServerAndRetiresItsInstructions` (Codex: conserva `[mcp_servers.engram]`, retira las claves y ficheros de instrucciones). RED observado en el caso por defecto de ambas.
  - Arreglo: campo `keepEngramRegistration` (parcial, global, sin `engram` nombrado); `engramOperationsKeepingRegistration` sustituye a `engramOperations` / `managedMCPOperations` en ese caso; `cleanCodexTOMLKeepingEngramServer`; nota `keptEngramRegistrationNote` en `ManualActions`, que lee lo que queda en disco y nombra el fichero. `--all` (y por tanto `--purge-state`) no cambia.
  - Evidencia: `go test ./internal/components/uninstall/ -count=1` → `ok`; `go test ./internal/cli -count=1 -run 'Uninstall'` → `ok`; `gofmt -l` y `go vet` limpios.
- [ ] T3. Aviso de `engram:reachable` por `.mcp.json` sin aprobar. Ruta: inline.

## Criterios de aceptación

- Prueba que reproduce cada caso (rojo antes, verde después).
- `go test -count=1` en verde para `./internal/components/uninstall/`, `./internal/components/engram/` (si cambia) y `./internal/cli/` troceado con `-run` sobre doctor y uninstall.
- Un commit por tarea.

## Impacto en la spec

Sin impacto en la spec
Motivo: ninguna capacidad de `openspec/specs/` describe el alcance de `uninstall --agent` ni `engram:reachable`; `engram-protocol-injection` (retirar activos de protocolo) se sigue cumpliendo.

## Progreso

- Exploración hecha; documento creado. Siguiente: T1.
