# ODD: MCP de proyecto en `.mcp.json` y guarda de rama con hooks en VS Code Copilot (vscode-mcp-guardas)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/vscode-mcp-guardas.md`. Espejo Engram: `odd/vscode-mcp-guardas/tasks`.
> Rama: `inc/vscode-mcp-guardas` · worktree `C:\repos\axiom-wt\vscode-mcp-guardas\axiom`.

## Objetivo

Dos huecos del adaptador de VS Code Copilot en el ámbito workspace:

- **A.** Declarar los servidores MCP de proyecto en su destino nativo, `<raíz>/.mcp.json` con `mcpServers`, y retirar las entradas que Axiom gestionaba en `.vscode/mcp.json`.
- **B.** Dar a Copilot la guarda de rama automática que ya tienen Claude Code (hooks) y OpenCode (plugin), con los hooks de agente de VS Code en `.github/hooks/`. Evaluar el preflight SDD.

## Problema y porqué

El equipo usa VS Code Copilot a diario. Hoy Axiom escribe `.vscode/mcp.json` (`servers`), que VS Code asocia al arnés Local, deprecado y que se retira en febrero de 2027. El arnés de Copilot no lee ese fichero. Además, en VS Code ninguna guarda automática protege los documentos de Axiom de escribirse en la rama equivocada.

## Evidencia de la documentación oficial (consultada el 26/11/2026; páginas fechadas el 25/11/2026)

MCP:

- https://code.visualstudio.com/docs/agents/reference/mcp-configuration: «The Local harness is deprecated and will be removed in February 2027». Para configuraciones nuevas, los destinos nativos son `.mcp.json` en la raíz del workspace y `~/.copilot/mcp-config.json`, con `mcpServers`, y no `.vscode/mcp.json`. El arnés no lee directamente `.vscode/mcp.json`: solo lo reenvía el editor, sin migrarlo, y descarta `envFile`, `dev` y las variables `${input:…}`.
- El formato portable usa `${NAME}` en lugar de `${env:NAME}`, y no admite `inputs`, `envFile`, `dev` ni `sandboxEnabled`.
- https://code.visualstudio.com/docs/agents/run/manage-mcp-servers: «Workspace `.vscode/mcp.json` and `.mcp.json` configurations inherit Workspace Trust». En modo restringido no arrancan.
- La documentación no dice qué pasa si el mismo nombre de servidor aparece en los dos ficheros. Por eso se retiran las entradas de Axiom de `.vscode/mcp.json` en lugar de mantenerlas duplicadas.

Hooks:

- https://code.visualstudio.com/docs/agent-customization/hooks y https://docs.github.com/en/copilot/reference/hooks-reference: el arnés de Copilot (VS Code, app y CLI) lee `.github/hooks/*.json` con `{"version":1,"hooks":{"preToolUse":[{"type":"command","command":"…","timeoutSec":10}]}}`. Las entradas admiten `bash`/`powershell`/`command`, `cwd`, `env`, `timeoutSec` y `matcher`. En `preToolUse` y `postToolUse`, `matcher` es una regex anclada sobre `toolName`.
- Carga camelCase de `preToolUse`: `sessionId`, `timestamp`, `cwd`, `toolName`, `toolArgs` (JSON codificado como cadena). `postToolUse` añade `toolResult`. **No incluye el nombre del evento**. La variante PascalCase (`PreToolUse`) manda campos snake_case y nombres de herramienta de Claude.
- Nombres de herramienta del arnés: `create` (→ `Write`), `edit`, `str_replace_editor` y `apply_patch` (→ `Edit`), `task` (→ `Agent`), `bash`, `powershell`, `view`, etc. La documentación **no fija los campos de los argumentos** de esas herramientas.
- Salida: `preToolUse` deniega con `{"permissionDecision":"deny","permissionDecisionReason":"…"}` en `stdout` (`hookSpecificOutput` no existe). `postToolUse` añade contexto para el modelo con `{"additionalContext":"…"}`. Vacío o `{}` no decide nada.
- Códigos de salida: `0` lee `stdout`. `2` deniega en `preToolUse` y solo es un aviso en `postToolUse`. Cualquier otro código distinto de cero **deniega en `preToolUse` (fail-closed)** y continúa en el resto. Un tiempo agotado es fail-open siempre.
- Los hooks de repositorio exigen una carpeta de confianza. `chat.useHooks: false` (ajuste de usuario) los apaga. Es una función en preview.
- El arnés de Copilot **también lee `.claude/settings.json` y `.claude/settings.local.json` del repositorio**. Ejecuta juntos los hooks de todas las fuentes, sin deduplicar.
- `subagentStart` no puede bloquear. Su carga (`transcriptPath`, `agentName`…) no trae el prompt. `preToolUse` y `postToolUse` no traen `transcriptPath`. El agente `general-purpose` no emite `subagentStart`.

## Decisiones

- **A. Destino `.mcp.json` compartido con Claude Code.** VS Code escribe en `<raíz>/.mcp.json` con `mcpServers` y la misma fusión que Claude Code (`MergeJSONObjects`). Con los dos clientes instalados, cada servidor debe tener una única forma válida para ambos. Si no, cada `sync` reescribiría la entrada del otro y dejaría de ser idempotente. Detalle en T1.
- **A. Forma única de cada entrada.** VS Code escribe la misma orden que Claude Code (Engram: `engram mcp --tools=agent`; Context7: `npx` con el paquete fijado, no el remoto por http de antes) y añade `type: "stdio"` y `tools: ["*"]`. Las dos claves aparecen en todos los ejemplos del arnés de Copilot, y la migración de VS Code añade `tools: ["*"]`. Que Claude Code las acepta se comprobó con `claude mcp get` sobre un `.mcp.json` de prueba: la entrada sale como servidor stdio. La fusión profunda de Claude (`command`/`args`) conserva esas claves, así que el fichero converge y `sync` es idempotente en cualquier orden.
- **A. Orden de Engram.** Se mantiene `stableEngramCommandForMergedConfig`: si el fichero ya trae una orden de Engram, se conserva. Si VS Code llega primero, escribe la ruta absoluta (no es un agente «estándar», porque una aplicación gráfica puede no ver el `PATH`). Si llega primero Claude Code, escribe `engram`. Las dos órdenes funcionan para los dos clientes en esa máquina. La lectura de la orden de VS Code busca `servers` (perfil de usuario) y, si no lo hay, `mcpServers` (proyecto).
- **A. Migración.** En `sync` se retiran de `.vscode/mcp.json` solo las entradas que escribió Axiom: la misma forma y sin claves añadidas (`env`, `headers`…). El texto, los comentarios, las `inputs` y los servidores del usuario se conservan. El fichero se borra solo si queda sin miembros, y la carpeta `.vscode` solo si queda vacía. Un `.vscode/mcp.json` ilegible no bloquea y no se toca. La desinstalación retira de él las entradas con la forma de Axiom, con el criterio de siempre.
- **A. Desinstalación compartida.** `.mcp.json` es de Claude Code y de VS Code. Si se desinstala uno con el otro instalado en el proyecto, las entradas de Axiom se quedan para el otro (`projectMCPJSONStaysInUse`, el mismo patrón que `.agents/skills`).
- **A. Preflight.** `WorkspaceUserConfigFiles` pasa a ser `.mcp.json`: uno ilegible se rechaza antes de escribir, porque la fusión genérica lo sustituiría por un documento vacío.
- **B. Formato camelCase nativo** (`preToolUse`/`postToolUse`) en `.github/hooks/axiom-branch-guard.json`. Es el formato primario documentado. La carga no trae el evento, así que la orden lo declara: `axiom branch guard --agent vscode --event pre|post`. El `matcher` limita la guarda a las herramientas de escritura (`create|edit|str_replace_editor|apply_patch`), para no lanzar un proceso en cada lectura.
- **B. Protocolo de salida de `vscode`.** Siempre termina con código `0`, porque cualquier otro código distinto de `2` deniega en `preToolUse` y rompería el fail-open. El bloqueo sale como JSON con `permissionDecision: "deny"`, el aviso como `additionalContext` y «nada que hacer» sin salida.
- **B. Rutas.** Como la documentación no fija los argumentos, `toolArgs` se acepta como cadena JSON o como objeto. Se leen `path`, `file_path` y `filePath`, y las cabeceras de parche (`*** Add/Update/Delete File:`) de `apply_patch`. Cualquier otra forma no da rutas y la guarda no hace nada (fail-open). Lo fijan pruebas de carga.
- **B. Convivencia con `.claude/settings.json`.** Con Claude Code instalado, VS Code ejecuta también su guarda de `.claude/settings.json` (alias `PreToolUse`, carga snake_case). Se acepta y se justifica: las dos guardas usan el mismo motor, así que dan el mismo veredicto. Un bloqueo duplicado sigue siendo un bloqueo, y el aviso de Claude (código `2` en `PostToolUse`) solo queda en el registro de VS Code, así que el modelo no lo ve dos veces. Además, la guarda `claude-code` lee `tool_input.file_path`, y si Copilot manda los argumentos de su arnés (`path`) no encuentra ruta y no hace nada. Evitarlo exigiría distinguir en la carga a Claude Code de Copilot, y la documentación no da ninguna señal estable para hacerlo.
- **B. Preflight SDD: no viable hoy**, ver T4.
- **Fuera de alcance**: el ámbito global de VS Code (`~/.copilot/mcp-config.json` y `~/.copilot/hooks/`). El perfil de usuario `mcp.json` también es del arnés Local, así que queda como seguimiento.

## Alcance autorizado

Adaptador `internal/agents/vscode/`, inyección MCP (`internal/components/mcp`, `internal/components/engram`), desinstalación y `sync` de workspace, aviso de Workspace Trust (`internal/cli/workspace_instructions.go`), `internal/branch/engine/guard_hook.go` y su orden de la CLI, instalación del hook de la guarda, manifiesto de capacidades, `docs/agents.md`, pruebas de esos paquetes y la spec viva `change-branch-lifecycle` (REQ-25.22).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go test` de los paquetes tocados con `-run`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`.

## Tareas

- [x] T1 · Bloque A: VS Code declara sus MCP de proyecto en `.mcp.json` (`mcpServers`). `sync` y la desinstalación retiran las entradas de Axiom de `.vscode/mcp.json`. Aviso de Workspace Trust, comentario del adaptador y docs (`agents.md`, `usage.md`, `engram.md`, `platforms.md`, `manual-de-inicio.md`) actualizados. Pruebas nuevas: migración con comentarios e idempotencia, fichero vaciado, Claude y VS Code compartiendo `.mcp.json`, `.mcp.json` ilegible rechazado, `.vscode/mcp.json` ilegible intacto, desinstalación con Claude instalado, `RemoveJSONCObjectMember` y `RetireLegacyVSCodeServer`. Evidencia: `go test ./internal/agents/... ./internal/components/{mcp,engram,uninstall,filemerge}/ -run 'VSCode|Workspace|MCP|Context7|Uninstall|JSONC'` ok; `go test ./internal/cli/ -run 'VSCode|WorkspaceTrust|UninstallWorkspace|ComponentPaths|WorkspaceClaude|EngramClaudeWorkspace'` ok.
- [ ] T2 · Bloque B: `axiom branch guard --agent vscode --event pre|post` con el protocolo de Copilot. Pruebas de carga y de salida.
- [ ] T3 · Bloque B: instalación, `sync` y desinstalación de `.github/hooks/axiom-branch-guard.json` sin tocar los hooks del usuario. Manifiesto de capacidades, comentario del adaptador y `docs/agents.md`.
- [ ] T4 · Preflight SDD: evaluación documentada aquí. `## Spec viva` (REQ-25.22) integrada con `axiom archive integrate --odd vscode-mcp-guardas`.

## Criterio de terminado

- Una instalación de workspace con VS Code deja Engram y Context7 en `.mcp.json` y no deja entradas de Axiom en `.vscode/mcp.json`. Las del usuario se conservan.
- `sync` repetido es byte a byte idéntico, también con Claude Code y VS Code a la vez.
- `axiom branch guard --agent vscode` bloquea un documento de Axiom en la rama equivocada y avisa del resto, con el protocolo de Copilot. Siempre sale con código `0`.
- `.github/hooks/axiom-branch-guard.json` se instala, se sincroniza y se retira sin tocar otros ficheros de `.github/hooks/`.
- `go test` (con `-run`) de los paquetes tocados en verde.

## Progreso

- Worktree creado desde `origin/main` (`b4b5aa10`).

## Siguiente paso

T1.
