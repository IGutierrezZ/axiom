# ODD · modelos-por-fase (P5 · adaptadores y modelo por fase)

> Paso P5 de la hoja de ruta `docs/audits/2026-11-03-vision-producto.md`, objetivo O5 y huecos G5.1–G5.4. Carril ODD, elegido por el usuario el 2026-11-11.

## Objetivo

Que cada entorno que usa el equipo reciba agentes por fase SDD con su propio modelo y esfuerzo, y que esa elección se haga desde el dashboard para cada adaptador que lo soporte. Entornos: OpenCode, Claude Code (CLI, extensión de VS Code y Desktop), VS Code Copilot y Copilot CLI.

## Problema y porqué

Estado real comprobado el 2026-11-11 (la tabla O5 de la auditoría está desfasada en G5.1):

- **G5.1:** la PR #241 creó el adaptador de Copilot CLI (`internal/agents/copilotcli`). Genera instrucciones, skills y MCP, pero `SubAgentsDir` devuelve `""` (`adapter.go:125-131`). No hay agentes por fase ni modelo.
- **G5.2:** el adaptador de VS Code tampoco genera agentes (`vscode/adapter.go:168-178`, `vscode/workspace.go:20-23`). `FileSubAgents` no está activado para VS Code en el manifiesto (`capabilitymanifest/manifest.go:353-355`).
- **G5.3:** `claude.Adapter.Detect` solo mira el binario `claude` y `~/.claude` (`claude/adapter.go:40-65`). Claude Desktop y la extensión `anthropic.claude-code` sin el CLI no se detectan. El adaptador de VS Code ya detecta extensiones en `~/.vscode/extensions` (`vscode/adapter.go:47-58`).
- **G5.4:** el modelo por fase se elige solo en la TUI (`internal/tui/screens/*_model_picker.go`). En la CLI, `--profile` y `--profile-phase` solo valen para OpenCode y no llevan esfuerzo (`internal/cli/sync.go:143-144,241-303`). El dashboard solo lee `opencode.json` (`GetModelAssignments`, `dashboard/service.go:1596-1634`; `GET /api/ecosystem/models` de solo lectura).

Cómo se guarda hoy: `model.Selection` y el estado del proyecto `.axiom/state.json` (`claude_phase_assignments`, `ModelAssignments` de OpenCode, etc., `internal/state/state.go`). `sdd.Inject` (`internal/components/sdd/inject.go`) escribe el modelo en las plantillas por fase: `{{CLAUDE_MODEL}}` y `{{CLAUDE_EFFORT_FRONTMATTER}}` en Claude, y provider/model/variant en OpenCode.

## Decisiones

- **D3 (usuario, 2026-11-11):** un único adaptador `claude-code`. La detección reconoce también Claude Desktop y la extensión de VS Code sin el CLI. Los tres usan el mismo motor y leen `.claude/`, `CLAUDE.md` y `.mcp.json`.
- **Agentes de Copilot (usuario, 2026-11-11):** un único `.agent.md` por fase, compartido por VS Code y Copilot CLI. Lleva `model` en formato «Nombre (copilot)» y `reasoning-effort`.
  - Fuentes:
    - https://code.visualstudio.com/docs/agents/reference/custom-agent-file-reference
    - https://code.visualstudio.com/docs/agents/subagents
    - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli
    - https://github.com/github/copilot-cli/issues/2133
  - Datos verificados en esas fuentes:
    - VS Code y Copilot CLI leen `.github/agents/` del proyecto y `~/.copilot/agents/` del usuario.
    - En VS Code el subagente respeta su propio `model`. El coordinador necesita la herramienta `agent` y la lista `agents:`.
    - Copilot CLI delega por `description` y acepta nombres con sufijo de vendor desde la versión 1.0.24. No consta que lea `reasoning-effort`.
- **Adaptadores que entran en G5.4:** OpenCode, Claude Code y Copilot, que son los que usa el equipo. Kiro y Codex conservan su selección en la TUI y quedan como seguimiento.

## Alcance autorizado

- Detección de Claude sin CLI (G5.3).
- Agentes por fase de Copilot en un fichero compartido, con modelo y esfuerzo, para VS Code y Copilot CLI (G5.1 y G5.2):
  - en workspace, en `.github/agents/`;
  - en global, en `~/.copilot/agents/`.
- Asignaciones por fase de Copilot en `model.Selection` y en el estado, restauradas por `sync` e inyectadas por `sdd.Inject`.
- API del dashboard de lectura y escritura del modelo por fase para OpenCode, Claude y Copilot.
  - La escritura guarda en el estado del proyecto abierto y regenera los ficheros del adaptador sin `os.Chdir`.
  - Solo admite peticiones de origen local, como en P4.
- Pantalla del dashboard para elegir proveedor, modelo y esfuerzo por fase.
- Documentación (`docs/agents.md` y `docs/usage.md`).

Fuera de alcance:
- Kiro y Codex en la UI.
- Modo workspace del adaptador de Copilot CLI (ya descartado en `adaptadores-agentes-upstream`).
- Flags nuevos de la CLI.
- Que VS Code lea también `.claude/agents` (lo hace por diseño): se anota como seguimiento.

## Configuración

- **TDD:** off. `axiom.yaml` no lo configura, así que se hacen las comprobaciones funcionales normales.
- **RDD:** off (por defecto).
- **Ejecutor de pruebas:** `go test` en proceso. **Prohibido compilar binarios sueltos** (Cortex).
  - El JS se prueba con Node mediante `runNodeModules`, solo para módulos de `js/core/`.
  - `internal/cli` entero tarda más de una hora aquí, así que se acota con `-run`.
- **Antes de abrir la PR:** pasar en local `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` y revisar que ninguna función exportada nueva quede solo para pruebas (`scripts/deadcode-ratchet.sh` en la CI). Así se evitan los dos fallos que tuvo la CI de P4.
- **Entrega:** `single-pr` con `size:exception` y un commit por tarea (preferencia del usuario). Previsión: entre 2.000 y 2.800 líneas con pruebas.

## Tareas

- [x] **T1 · Detección de Claude sin CLI (G5.3).** Ruta: escritor delegado.
  - `claude.Adapter.Detect` da Claude por instalado si encuentra el CLI, Claude Desktop o la extensión `anthropic.claude-code` de VS Code.
    - Claude Desktop: Windows `%APPDATA%\Claude` o `%LOCALAPPDATA%\AnthropicClaude`; macOS `/Applications/Claude.app` o `~/Library/Application Support/Claude`.
    - Extensión: `~/.vscode/extensions/anthropic.claude-code-*`, reutilizando el patrón de `vscode/adapter.go:47-58`.
  - Pruebas con un HOME temporal para cada señal.
  - `docs/agents.md` cuenta que el adaptador cubre los tres entornos.
  - Seguimiento: `agentbuilder/engine.go` y el aviso final de `run.go` siguen exigiendo el CLI `claude`, así que con solo Desktop o la extensión esos dos puntos no funcionan.
- [x] **T2 · Asignaciones por fase de Copilot en el modelo y el estado.** Ruta: escritor delegado.
  - `model.Selection.CopilotPhaseAssignments` (modelo y esfuerzo por fase) y su persistencia en `.axiom/state.json`.
  - `sync` las restaura, igual que `ClaudePhaseAssignments` (`internal/cli/sync.go:2186-2200`).
  - Sin asignación, el agente se genera sin `model`, así que hereda el modelo de la conversación.
- [x] **T3 · Agentes por fase de Copilot (G5.1 y G5.2).** Ruta: escritor delegado.
  - Plantillas `.agent.md` de la cohorte SDD (una por fase, sin coordinador) con `name`, `description`, `model` y `reasoning-effort`.
  - `sdd.Inject` las rellena con un resolver de modelos de Copilot.
  - VS Code (workspace) escribe en `.github/agents/` y Copilot CLI (global) en `~/.copilot/agents/`.
  - Si los dos adaptadores están instalados, el fichero compartido se escribe una sola vez y de forma idempotente.
  - Se activa `FileSubAgents` en el manifiesto de los dos adaptadores.
  - `uninstall` quita los agentes.
  - Pruebas: frontmatter generado, idempotencia, convivencia de los dos adaptadores y desinstalación.
- [ ] **T4 · API del dashboard para el modelo por fase (G5.4).** Ruta: escritor delegado.
  - `GET /api/ecosystem/models` devuelve las asignaciones por adaptador instalado (OpenCode, Claude, Copilot) y las opciones de cada uno:
    - modelos de Claude (alias de la TUI) y sus esfuerzos válidos;
    - catálogo de OpenCode;
    - modelos de Copilot sugeridos, con texto libre permitido.
  - `POST /api/ecosystem/models` valida, guarda en el estado del proyecto abierto y regenera los ficheros del adaptador sin `os.Chdir`.
    - 400 con violaciones si los datos no son válidos.
    - 403 si la petición no es de origen local.
- [ ] **T5 · Pantalla del dashboard para el modelo por fase (G5.4).** Ruta: escritor delegado.
  - En Ecosistema, una tabla por adaptador con proveedor y modelo, esfuerzo por fase, guardar y errores.
  - La validación en el navegador va en un módulo de `js/core/`, con pruebas en Node.
- [ ] **T6 · Documentación.** Ruta: en línea si es mecánica.
  - `docs/agents.md`: agentes de Copilot, formato compartido y versión mínima del CLI.
  - `docs/usage.md`: modelo por fase desde el dashboard.

## Criterios de aceptación

- Con solo Claude Desktop o solo la extensión de VS Code, `axiom install` y el alta de P4 ofrecen y detectan `claude-code`.
- Tras `axiom sync` con VS Code Copilot o Copilot CLI instalados, hay un `.agent.md` por fase en la carpeta que toca. Cada uno lleva el `model` y el `reasoning-effort` elegidos, o ninguno si no hay asignación.
- Desde el dashboard se cambia el modelo y el esfuerzo de una fase de OpenCode, Claude y Copilot. El cambio queda en `.axiom/state.json` y en los ficheros del adaptador del proyecto abierto.
- Las pruebas de los paquetes tocados, `go vet`, `gofmt`, el ratchet de rechazos y el de código sin uso pasan.

## Progreso y evidencias

- 2026-11-11: exploración delegada y consulta de fuentes primarias. D3 y el formato compartido los decidió el usuario. Worktree `inc/modelos-por-fase` desde `0546f993`.
- 2026-11-11: T1 hecha (ruta: escritor delegado, commit 65008239). `claude.Adapter.Detect` da `claude-code` por instalado con el CLI, la extensión `anthropic.claude-code-*` o Claude Desktop (Windows `%APPDATA%\Claude` / `%LOCALAPPDATA%\AnthropicClaude`, macOS `/Applications/Claude.app` / `~/Library/Application Support/Claude`). Las variables de entorno solo se respetan con el HOME real, como en el adaptador de VS Code. `binaryPath` queda vacío sin CLI y ningún llamador lo ejecuta. Observado: `go test ./internal/agents/...`: ok; `go vet`: limpio; `gofmt -l`: vacío. Desviación: el commit de T1 no incluyó la actualización de este documento; va en el commit de T2.
- 2026-11-11: T2 hecha (ruta: escritor delegado, commit 0f73a76d). `model.CopilotPhaseAssignment` (modelo en texto libre y esfuerzo opcional) en `Selection` y `SyncOverrides`, con validación (fases desconocidas, modelo vacío, caracteres de control que inyectarían claves en el frontmatter). Persistencia en `copilot_phase_assignments` (`state.go`, `MergeAgents`, `run.go`, `app.go`). `axiom sync` las restaura y rechaza las inválidas con un error que nombra `axiom sync`; sin asignación no cambia nada. Una instalación sin asignaciones de Copilot conserva las del estado (desviación respecto a Claude, que las sobrescribe). Observado: `go test ./internal/agents/... ./internal/model/... ./internal/state/...`: ok; `go test ./internal/cli/ -run 'Sync|Install|Detect|Claude|Copilot|TestEveryProductionRefusal...'`: ok; `go test ./internal/dashboard/`: ok; `go test ./internal/app/`: falla solo `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` (versión de OpenCode no disponible en esta máquina; entorno); `go vet` y `gofmt -l`: limpios. Desviaciones de T2: se rechazan los caracteres de control; una instalación sin asignaciones de Copilot conserva las del estado; un estado inválido hace fallar `sync` con un error que nombra `axiom sync`. El fallo de `internal/app` se confirmó también en `main`.
- 2026-11-11: T3 hecha (ruta: escritor delegado, commit pendiente de registrar). 14 plantillas `internal/assets/copilot/agents/*.agent.md` (las fases de Claude sin `review-*`, que Copilot no usa), con frontmatter `name`, `description`, `model` y `reasoning-effort` (sin `tools`, `agents` ni `handoffs`; sin asignación no hay líneas de modelo). `sdd.Inject` las rellena desde `CopilotPhaseAssignments` con el marcador `{{COPILOT_MODEL_FRONTMATTER}}` y valores YAML con comillas solo si hacen falta. VS Code escribe en `.github/agents` (proyecto) o `~/.copilot/agents` (global) y Copilot CLI en `<raíz Copilot>/agents`; mismo fichero byte a byte, así que el segundo adaptador no reescribe. `FileSubAgents` activado en el manifiesto de ambos (digests actualizados). `uninstall` retira solo los nombres que Axiom gestiona y deja los agentes del usuario y los compartidos mientras el otro adaptador siga instalado. `orchestrator` sale de las fases de Copilot (no hay fichero para la conversación principal) y entra `sdd-init`. Observado: `go test ./internal/agents/... ./internal/components/... ./internal/model/... ./internal/state/... ./internal/copilotconfig/...`: ok salvo `filemerge` (junctions, falla igual en `main`); `go test ./internal/cli/ -run 'Copilot|VSCode|Vscode|SubAgent|Workspace.*(Install|Sync|Uninstall)|RefusalNames...|RunSync_'`: ok (el filtro amplio `Sync|Install|Uninstall` agota 40 min en esta máquina); `go test ./internal/dashboard/`: ok; `go vet` y `gofmt -l`: limpios.

## Seguimientos fuera de alcance

- VS Code también lee `.claude/agents` y `~/.claude/agents`. En un proyecto con Claude Code y VS Code Copilot, los agentes de Claude (`model: sonnet`) aparecen en VS Code junto a los de Copilot.
- Kiro y Codex en la pantalla de modelos del dashboard.
- Actualizar la tabla O5 de la auditoría (G5.1 ya no es «no existe»). Se hará al marcar P5.

Plan stop: approved 2026-11-11; implementation continues
