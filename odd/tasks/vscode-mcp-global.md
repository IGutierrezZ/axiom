# ODD: MCP de usuario de VS Code Copilot en `~/.copilot/mcp-config.json` (vscode-mcp-global)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/vscode-mcp-global.md`. Espejo Engram: `odd/vscode-mcp-global/tasks`.
> Rama: `inc/vscode-mcp-global` · worktree `C:\repos\axiom-wt\vscode-mcp-global\axiom`.

## Objetivo

Con `--scope global`, declarar los servidores MCP de VS Code Copilot (Engram y Context7) en el destino de usuario nativo del arnés de Copilot, `<COPILOT_HOME|~/.copilot>/mcp-config.json` con `mcpServers`. Además, retirar del `mcp.json` del perfil de usuario de VS Code (`servers`) solo las entradas que escribió Axiom, en `sync` y en la desinstalación.

## Problema y porqué

Hoy, con `--scope global`, `vscode.Adapter.MCPConfigPath` escribe en `<carpeta de usuario de VS Code>/mcp.json` con la clave `servers`. Ese formato es del arnés Local, que está deprecado y que VS Code retira en febrero de 2027. La PR IGutierrezZ/axiom#323 ya migró el ámbito workspace (`.vscode/mcp.json` → `.mcp.json`) y dejó el global como seguimiento.

## Evidencia de la documentación oficial (consultada el 26/11/2026)

- https://code.visualstudio.com/docs/agents/reference/mcp-configuration:
  - El arnés de Copilot lee directamente `$COPILOT_HOME/mcp-config.json`, o `~/.copilot/mcp-config.json` si `COPILOT_HOME` no está definida. Lee también el `.mcp.json` del workspace. Cita: «Both files use a top-level `mcpServers` object, and the harness reads them directly».
  - Para configuraciones nuevas, la página remite a esos destinos nativos. El `mcp.json` del perfil usa el formato `servers` del arnés Local, que está deprecado («will be removed in February 2027»).
  - El arnés no lee ese fichero directamente: solo el editor lo reenvía, si es elegible, y sin migrarlo.
  - Campos portables: `type` (`stdio`, `http` o `sse`), `command`/`args`/`env` (stdio), `url`/`headers` (http) y `tools` (`["*"]` expone todas).
- https://code.visualstudio.com/docs/copilot/customization/mcp-servers: el fichero de usuario de Copilot es «shared across your workspaces and compatible Copilot experiences on the same host, including Copilot CLI». Es el mismo fichero que ya escribe el adaptador de Copilot CLI.
- https://code.visualstudio.com/docs/agents/run/manage-mcp-servers: Settings Sync sincroniza el `mcp.json` del perfil, no el `mcp-config.json` de Copilot. La documentación no dice qué gana si el mismo servidor aparece en los dos ficheros. Por eso se retira la entrada del perfil y no se deja duplicada.
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers: los ejemplos de JSON usan `"type": "local"`, pero `copilot mcp add --transport` documenta `stdio`, `http` o `sse` (por defecto `stdio`), y el formulario de `/mcp add` llama a STDIO «the standard MCP protocol type name». `tools` toma por defecto `*`.

## Decisiones

- **VS Code global escribe `~/.copilot/mcp-config.json`, con la misma entrada que Copilot CLI.** El arnés de Copilot en VS Code lee ese fichero directamente y lo comparte con Copilot CLI en el mismo equipo. Dejarlo solo a Copilot CLI obligaría a instalar la CLI para tener los MCP en VS Code. Con una sola forma por servidor, `sync` es idempotente con los dos agentes instalados, en cualquier orden.
- **Engram con `type: "stdio"` en los dos agentes.** Hasta ahora Copilot CLI escribía `type: "local"`. Ese valor solo aparece en los ejemplos de GitHub, y la referencia de VS Code no lo admite. `stdio` es el único valor documentado por las dos fuentes. Además es el que ya usa `.mcp.json` desde la IGutierrezZ/axiom#323, y Copilot CLI también lee ese fichero. `sync` reescribe el valor en las instalaciones anteriores.
- **Retirada del perfil en `sync`.** `sync` retira del `mcp.json` del perfil solo las entradas de Engram y Context7 que escribió Axiom, con `mcp.RetireLegacyVSCodeServer`: la misma forma, sin claves añadidas por el usuario. Los servidores, las `inputs` y los comentarios del usuario se conservan. El fichero se borra solo si se queda vacío. `RetireLegacyVSCodeServer` borra la carpeta padre vacía solo si es `.vscode`: la carpeta de usuario de VS Code nunca se borra.
- **Un solo proveedor del fichero antiguo.** `LegacyWorkspaceMCPConfigProvider` pasa a `LegacyMCPConfigProvider` (`LegacyMCPConfigPath`). `vscode.Adapter` devuelve el `mcp.json` del perfil, y `WorkspaceAdapter` devuelve `.vscode/mcp.json` en el proyecto y delega fuera de él. Así la inyección, la copia de seguridad (`legacyMCPConfigPaths`) y la desinstalación usan el mismo mecanismo en los dos ámbitos.
- **La forma de Context7 la decide el fichero.** En `<proyecto>/.mcp.json` (compartido con Claude Code), Context7 va por `npx`. En `mcp-config.json` (compartido con Copilot CLI), va por `http` (`CopilotCLIContext7OverlayJSON`). Engram tiene una sola overlay para los dos ficheros (`copilotEngramOverlayJSON`). Desaparecen las overlays con `servers` (`vsCodeEngramOverlayJSON`, `VSCodeContext7OverlayJSON`) y la lectura de `servers` en `existingMergedEngramCommand` y en la sonda de salud.
- **Orden de Engram.** Se mantiene `stableEngramCommandForMergedConfig`: la primera orden escrita se conserva. Si VS Code llega antes, escribe la ruta absoluta (una aplicación gráfica puede no ver el `PATH`). Si llega antes Copilot CLI, escribe `engram`. En los dos órdenes, `sync` deja el fichero byte a byte igual. Es el mismo criterio que la IGutierrezZ/axiom#323 aceptó para `.mcp.json`.
- **Detección de Copilot CLI.** `copilotconfig.HasCLIState` dejaba de servir si `mcp-config.json` seguía contando como señal propia del CLI: un usuario que solo tuviera VS Code se detectaría como usuario de Copilot CLI. Pasa a la lista de entradas compartidas, junto a `skills/` y `agents/`.
- **Desinstalación global compartida.** En el ámbito global, las estrategias `StrategyMCPConfigFile` pasan por `globalMCPConfigFileOperations`. Esa función retira la entrada del fichero antiguo (`servers`) y la de `mcp-config.json` (`mcpServers`). La segunda solo se retira si ningún otro agente instalado lee el mismo fichero (`mcpConfigStaysInUse`, el patrón de `skillsDirStaysInUse`). El criterio de borrado es el global de siempre (`rewriteMCPServerEntry`). También protege a VS Code cuando se desinstala Copilot CLI, cosa que antes no ocurría.

## Alcance autorizado

Adaptadores `internal/agents/vscode/` y `internal/agents/copilotcli/` (solo lo que comparten), `internal/agents/interface.go`, `internal/copilotconfig`, inyección MCP (`internal/components/mcp`, `internal/components/engram`), desinstalación global (`internal/components/uninstall`), `sync` y copia de seguridad (`internal/cli`), `docs/agents.md`, `docs/platforms.md` y las pruebas de esos paquetes.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y el usuario no ha elegido otro modo. Comprobaciones: `go test` de los paquetes tocados con `-run`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea, con la etiqueta `type:feature`. La rama reutiliza `RetireLegacyVSCodeServer`, de la IGutierrezZ/axiom#323, ya en `main`.

## Tareas

- [x] T1 · Destino MCP global de VS Code en `<COPILOT_HOME|~/.copilot>/mcp-config.json`. Instalación y `sync` con la entrada de Copilot CLI (Engram con `type: "stdio"` en los dos agentes). Retirada de las entradas de Axiom del `mcp.json` del perfil en `sync` y en la desinstalación. La desinstalación de cualquiera de los dos agentes no retira el MCP mientras el otro siga instalado. Copia de seguridad del fichero antiguo, detección de Copilot CLI, comentarios de los adaptadores, `docs/agents.md`, `docs/platforms.md` y pruebas.

  Pruebas nuevas o reescritas: destino y contención de `COPILOT_HOME` (`TestMCPConfigPathUsesCopilotUserConfig`), fichero antiguo del perfil (`TestLegacyMCPConfigPathUsesVSCodeUserProfile`), Engram y Context7 de VS Code en `mcp-config.json` con retirada del perfil, conservando servidores y comentarios del usuario y la carpeta `Code/User`. También: VS Code y Copilot CLI sobre el mismo fichero en los dos órdenes (idempotente byte a byte), sonda de salud de VS Code, migración de `local` a `stdio`, desinstalación de uno con el otro instalado y luego del último, y `HasCLIState` sin `mcp-config.json`. La prueba de desinstalación compartida falla si se desactiva `mcpConfigStaysInUse` (comprobado).

  Evidencia:
  - `go test ./internal/agents/vscode/ ./internal/agents/copilotcli/ ./internal/copilotconfig/ ./internal/components/mcp/ ./internal/components/engram/ ./internal/components/uninstall/`: ok.
  - `go test ./internal/system/ ./internal/agents/ -run 'Copilot|Scan|Skills'`: ok.
  - `go test ./internal/cli/ -run 'VSCode|WorkspaceTrust|UninstallWorkspace|ComponentPaths|Copilot|Backup|GlobalAgent'`: ok.
  - `go test ./internal/cli/ -run 'Footprint|Leftover|Doctor|EngramHealth|Persisted'`: ok.

Una sola tarea: instalación, `sync` y desinstalación son un mismo comportamiento sobre un fichero compartido. Partirlas dejaría un commit intermedio con la desinstalación global sin rumbo (seguiría buscando `servers` en `mcp-config.json`). Ruta: inline. La exploración (mapa de consumidores, desinstalación global y overlays) y la lectura de la documentación oficial se delegaron en exploradores de solo lectura, porque superaban el presupuesto de evidencia en línea.

## Criterio de terminado

- `axiom install --scope global --agent vscode-copilot` deja Engram y Context7 en `~/.copilot/mcp-config.json` (`mcpServers`) y ninguna entrada de Axiom en el `mcp.json` del perfil. Los servidores del usuario se conservan.
- Con VS Code y Copilot CLI instalados, `sync` repetido deja el mismo fichero byte a byte.
- La desinstalación global de VS Code no deja entradas de Axiom en el perfil y no rompe Copilot CLI.
- `go test` (con `-run`) de los paquetes tocados en verde.

## Progreso

- Worktree creado desde `origin/main` (`b7e91d67`) y apilado sobre `origin/inc/vscode-mcp-guardas`. Tras fusionarse la IGutierrezZ/axiom#323 (y la #324), los commits se reaplicaron sobre `origin/main` (`931c2f4d`) sin el merge de apilado.
- T1 `9abdfeb3`, hecha y comprobada (ver la evidencia de la tarea).

## No verificado

- No hay ninguna prueba contra un VS Code o un Copilot CLI reales. Que los dos acepten `type: "stdio"` en `mcp-config.json` sale de la documentación oficial. El dato de Copilot CLI es indirecto: `copilot mcp add --transport stdio` y «local (stdio) servers».
- La documentación no dice qué gana si el mismo servidor aparece en el `mcp.json` del perfil y en `mcp-config.json`. Por eso se retira la entrada del perfil.
- Con `COPILOT_HOME` definida, las skills de VS Code siguen en `~/.copilot/skills` (como antes) y el MCP va a `$COPILOT_HOME/mcp-config.json`, que es lo que documenta VS Code. Las skills quedan fuera de alcance.

## Siguiente paso

Abrir la PR (`type:feature`), CI en verde, merge commit y `done`.
