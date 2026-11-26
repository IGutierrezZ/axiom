# Agentes compatibles con Axiom

← [Back to README](../README.md)

---

## Agent Matrix

| Agent           | ID               | Skills       | MCP | Delegation                       | Output Styles | Slash Commands | Config Path                         |
| --------------- | ---------------- | ------------ | --- | -------------------------------- | ------------- | -------------- | ----------------------------------- |
| Claude Code     | `claude-code`    | Yes          | Yes | Full (Task tool)                 | Yes           | Yes            | `~/.claude`                         |
| OpenCode        | `opencode`       | Yes          | Yes | Full (multi-mode overlay)        | No            | Yes            | `~/.config/opencode`                |
| Gemini CLI      | `gemini-cli`     | Yes          | Yes | Full (experimental)              | No            | No             | `~/.gemini`                         |
| Cursor          | `cursor`         | Yes          | Yes | Full (native subagents)          | No            | No             | `~/.cursor`                         |
| VS Code Copilot | `vscode-copilot` | Yes          | Yes | Full (runSubagent; agentes `.agent.md` por fase, solo en el proyecto) | No            | No             | `~/.copilot` + VS Code User profile |
| GitHub Copilot CLI | `github-copilot-cli` | Yes      | Yes | Agentes `.agent.md` por fase, en `.github/agents` del proyecto (delegación por `description`) | No            | No             | `~/.copilot` o `$COPILOT_HOME` (solo global) |
| Codex           | `codex`          | Yes          | Yes | Native multi-agent (default; solo fallback) | No            | No             | `~/.codex`                          |
| Antigravity     | `antigravity`    | Yes (native) | Yes | Solo-agent + Mission Control     | No            | No             | `~/.gemini/antigravity`             |
| Qwen Code (no disponible de momento) | `qwen-code`      | Yes          | Yes | Full (native sub-agents)         | No            | Yes            | `~/.qwen`                           |
| Trae            | `trae-ide`       | Yes          | Yes | Solo-agent                       | No            | No             | `~/.trae`                           |
| Pi              | `pi`             | Yes          | Yes | Full (package-managed subagents) | No            | Yes            | `~/.pi`                             |

La columna «Config Path» es la ruta del ámbito global (`--scope global`). Con `--scope workspace`, el predeterminado, cada agente usa las rutas de proyecto que se resumen en [Modo workspace por agente](#modo-workspace-por-agente).

### Agentes no disponibles de momento

Qwen Code (`qwen-code`) **no está disponible de momento**: no se puede elegir al instalar. No aparecen en la pantalla de selección de agentes de la TUI, no entran en las selecciones por defecto ni en el descubrimiento por carpetas, y `axiom install --agent qwen-code` falla con un error (también si va junto a otros agentes). El motivo es que hoy no es utilizable con Axiom: en el ámbito workspace escribe en rutas que el agente no lee y su MCP solo es global. Su adaptador se conserva en el código para retomarlo más adelante.

- Una instalación global existente que ya lo tiene en el estado se sigue sincronizando con `axiom sync`.
- Con `--scope workspace` se omite con un aviso, igual que los agentes sin modo workspace.
- `uninstall`, `doctor` y `verify` no cambian.
- La sección de Qwen Code de más abajo describe el adaptador conservado, no una integración instalable hoy.

La mayoría de los agentes recibe la política completa del orquestador SDD y las skills correspondientes en su directorio de configuración. Axiom la incorpora al prompt de sistema; OpenCode la recibe mediante el overlay compatible con OpenCode (`opencode.json`). Pi es la excepción: Axiom aprovisiona paquetes de Pi y `gentle-pi` —nombre del paquete externo— gestiona las skills, los prompts, los agentes SDD y las cadenas durante la ejecución. El agente usa SDD cuando el trabajo lo requiere o cuando se lo pides.

`axiom install --scope=workspace` permite limitar al workspace los ficheros de agente compatibles, no solo los de Claude Code. En ese ámbito, Axiom escribe prompts de sistema, skills, agentes SDD y ficheros de persona en el proyecto actual cuando el agente admite configuración local. Las integraciones globales —por ejemplo, paquetes o ajustes que el agente solo lee desde su configuración de usuario— permanecen globales. Axiom migra una vez los nombres antiguos (`gentle-ai`) que encuentre en disco: rutas de estado, marcadores y entradas gestionadas.

#### Modo workspace por agente

El ámbito workspace es el predeterminado de la CLI y de la TUI. Axiom escribe en la raíz del proyecto (el `axiom.yaml` más cercano o, si no hay, la raíz del repositorio git) y no en tu carpeta de usuario; lo que sigue a nivel de máquina está declarado en [Uso](usage.md#cli-flags-install). Las instrucciones ODD/SDD son del equipo y van siempre en un fichero propio de Axiom, nunca en un fichero raíz compartido (`AGENTS.md`, `GEMINI.md`), y la regla general es que lo del proyecto prevalece sobre lo del usuario y ambos conviven (véanse los matices en «Precedencia»).

| Agente | Instrucciones | Configuración, MCP y hooks | Skills |
| --- | --- | --- | --- |
| Claude Code | `.claude/CLAUDE.md` | `.claude/` (`settings.json`, `agents/`, `commands/`, `output-styles/`); MCP en `.mcp.json` (`mcpServers`) | `.claude/skills` |
| OpenCode | `.opencode/axiom.md`, registrado en `instructions` de `opencode.json` | `opencode.json` (o `opencode.jsonc`) y `.opencode/{commands,plugins,prompts/sdd}` | `.agents/skills` |
| Gemini CLI | `.gemini/AXIOM.md`, registrado en `context.fileName` de `.gemini/settings.json` | `.gemini/settings.json` (`mcpServers`) y `.gemini/` | `.agents/skills` |
| Codex | `developer_instructions` de `.codex/config.toml` (bloque `<!-- axiom:workspace-instructions -->`) y `.codex/axiom-instructions.md` | `.codex/config.toml`, `.codex/hooks.json`; exige proyecto de confianza | `.agents/skills` |
| VS Code Copilot | `.github/instructions/axiom.instructions.md` (`applyTo: "**"`) | MCP en `.mcp.json` (`mcpServers`, compartido con Claude Code), `.github/agents/` (un `.agent.md` por fase SDD) y `.github/hooks/axiom-branch-guard.json` (guarda de rama); sin prompts | `.agents/skills` |
| Cursor | `.cursor/rules/axiom.mdc` | `.cursor/` (`agents/`, `mcp.json`) | `.agents/skills` |

- **Sin modo workspace.** Trae, Antigravity, Pi y GitHub Copilot CLI solo tienen configuración global: con `--scope workspace` se omiten con un aviso en castellano y Axiom nunca cae a escribir en su configuración de usuario. Para instalarlos hay que pedir `--scope global` de forma expresa. Pi se omite porque `pi install -l` solo cubre los paquetes (`.pi/settings.json`), mientras que `pi-engram init`, `~/.pi/npm/package.json` y el prompt de sistema solo tienen ruta de usuario. Antigravity queda aplazado: sus reglas de proyecto limitan el tamaño de cada fichero y el orquestador de Axiom no cabe en una sola regla.
- **No seleccionable.** Qwen Code ya no se puede elegir (véase [Agentes no disponibles de momento](#agentes-no-disponibles-de-momento)).
- **Precedencia.** Lo del proyecto prevalece sobre lo del usuario y ambos conviven, con dos matices. En Claude Code, una skill personal con el mismo nombre que una de proyecto gana a esta. En Gemini CLI, el `context.fileName` del proyecto sustituye al del usuario y no se fusiona: Axiom escribe una lista que conserva los nombres que el proyecto ya declaraba (o `GEMINI.md`) y añade `.gemini/AXIOM.md`, de modo que un nombre que solo figure en `~/.gemini/settings.json` deja de cargarse en ese proyecto.
- **GitHub Copilot CLI.** Tiene adaptador (`github-copilot-cli`), pero solo global: lo que Axiom le escribe vive en `~/.copilot` (o `$COPILOT_HOME`) y no hay una ruta de proyecto documentada para ello, así que se omite en workspace. El CLI sí lee del proyecto `.github/instructions`, `.github/skills` y `.mcp.json` (el que comparten Claude Code y VS Code Copilot), así que se beneficia de lo que Axiom escribe en workspace para VS Code Copilot y Claude Code.
- **Lo que no escribe en `~/`.** `engram setup <agente>` se omite con un aviso, porque escribe en la carpeta de usuario del agente; el MCP de Engram ya queda en el proyecto. Con Codex, el aviso sugiere `engram setup codex` como acción manual opcional: ejecútalo solo si quieres además el registro de usuario, porque escribe en `~/.codex`, fuera del proyecto.

#### Skills en `--scope workspace`: una sola copia compartida

Con `--scope workspace`, Axiom escribe las skills en el menor número de carpetas. Los agentes que leen `<proyecto>/.agents/skills` comparten una única copia, con el mismo contenido para todos. Antes cada agente recibía una copia propia y varios leen además otras carpetas, así que veían cada skill varias veces (por ejemplo, avisos `duplicate skill name` en OpenCode). Claude Code no lee `.agents/skills` y conserva la suya.

| Agente | Carpeta de skills de proyecto | Qué dice su documentación |
| --- | --- | --- |
| Codex | `.agents/skills` | «Codex scans `.agents/skills` in every directory from your current working directory up to the repository root» ([learn.chatgpt.com/docs/build-skills](https://learn.chatgpt.com/docs/build-skills.md)). |
| VS Code Copilot | `.agents/skills` | «Local sessions also read project skills from `.claude/skills/` and `.agents/skills/`» ([code.visualstudio.com/docs/copilot/customization/agent-skills](https://code.visualstudio.com/docs/copilot/customization/agent-skills)). Las sesiones locales son las de la vista de chat. La tabla de esa página da `.github/skills` como carpeta de proyecto del arnés de Copilot, y la documentación de GitHub lista `.agents/skills` entre las ubicaciones de proyecto de Copilot. |
| OpenCode | `.agents/skills` | Busca en `.opencode/skills`, `.claude/skills` y `.agents/skills` ([opencode.ai/docs/skills](https://opencode.ai/docs/skills/)). |
| Gemini CLI | `.agents/skills` | `.agents/skills` es un alias de `.gemini/skills` y gana a esta en el mismo nivel ([docs/cli/skills.md](https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md)). |
| Cursor | `.agents/skills` | Carga `.agents/skills` y `.cursor/skills` como carpetas de proyecto ([cursor.com/docs/context/skills](https://cursor.com/docs/context/skills)). |
| Claude Code | `.claude/skills` | No lee `.agents/skills` ([code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills)). Solo se escribe si Claude Code está entre los agentes. |

- La carpeta compartida se escribe una sola vez por ejecución, aunque varios agentes la usen, y no aparece repetida en la copia de seguridad ni en la verificación. Con solo Claude Code no se crea `.agents/skills`.
- Las rutas de skills que Axiom escribe en los prompts, overlays y orquestadores (`{{AXIOM_SKILLS_DIR}}` y las rutas `_shared`) apuntan a la carpeta efectiva de cada agente, y el índice de skills del bloque `axiom:skills-index` de `AGENTS.md` prefiere `.agents/skills` cuando existe (una skill en `skills/` o en una raíz declarada sigue ganando).
- Los ficheros de soporte `_shared` de la copia compartida llevan el hueco genérico `<runtime>` en lugar de la identidad de un agente concreto, igual que la copia de compatibilidad de `~/.agents/skills`, para que su contenido no dependa de qué agente la escriba.
- **Migración.** Las instalaciones workspace anteriores dejaron copias en `.github/skills`, `.opencode/skills`, `.gemini/skills`, `.cursor/skills` y `.codex/skills`. Al instalar o sincronizar, Axiom retira una carpeta de skill solo si prueba que es suya: tiene el nombre de una skill que Axiom distribuye, contiene exactamente los ficheros que Axiom genera y cada uno es byte a byte igual a lo que Axiom genera hoy, y la copia compartida ya existe. Si difiere o tiene ficheros de más, se conserva y se avisa para que la revises. Las skills ajenas no se tocan, y la carpeta contenedora (`.github/skills`, etc.) solo se borra si queda vacía. Los ficheros que se pueden retirar van en la copia de seguridad previa.
- El ámbito global no cambia: cada agente sigue escribiendo en su carpeta de usuario.

---

## Delegation Models

| Model                 | How It Works                                                                                                                                                                                       | Agents                                                                                                    |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Full (sub-agents)** | Each SDD phase runs in an isolated context window via native sub-agent delegation, package-managed subagents, or an OpenCode-compatible overlay. The orchestrator coordinates; sub-agents execute. | Claude Code, OpenCode, Gemini CLI, Cursor, VS Code Copilot, Pi (Qwen Code, con adaptador conservado pero no seleccionable) |
| **Native multi-agent** | The orchestrator delegates through the agent's native collaboration tools when configured and available, with inline execution as a graceful fallback. | Codex |
| **Solo-agent**        | All SDD phases run inline in the same conversation. The orchestrator IS the executor. Engram™ provides cross-phase persistence.                                                                     | Antigravity, Trae                                                                     |

### Cursor Native Subagents

Cursor usa su sistema integrado `.cursor/agents/`. Axiom escribe ficheros de agentes en `~/.cursor/agents/sdd-{phase}.md`; Cursor delega según el campo `description` de la cabecera YAML de cada fichero.

- `sdd-explore` and `sdd-verify` run with `readonly: false` so they can inspect the codebase and execute verification commands
- Each subagent gets its own context window (fresh context, no pollution)
- The orchestrator resolves skill paths from the skill registry and passes exact `SKILL.md` files in the invocation message

### Antigravity + Mission Control

Antigravity is an agent-first platform with built-in sub-agents (Browser, Terminal) managed by Mission Control. However, custom sub-agent creation is not yet available. SDD phases run inline, with Mission Control handling automatic delegation to built-in sub-agents when specialized tooling is needed (e.g., Browser for research during `sdd-explore`).

---

## SDD Mode Support

| Feature          | Claude Code | OpenCode | Gemini CLI | Cursor | VS Code Copilot | Codex | Antigravity | Qwen Code | Trae |   Pi    |
| ---------------- | :---------: | :------: | :--------: | :----: | :-------------: | :---: | :---------: | :-------: | :--: | :-----: |
| SDD orchestrator |     Yes     |   Yes    |    Yes     |  Yes   |       Yes       |  Yes  |     Yes     |    Yes    | Yes  |   Yes   |
| Single-mode SDD  |     Yes     |   Yes    |    Yes     |  Yes   |       Yes       |  Yes  |     Yes     |    Yes    | Yes  |   Yes   |
| Multi-mode SDD   |      —      |   Yes    |     —      |   —    |        —        |   —   |      —      |     —     |  —   | Yes** |

**Multi-mode** (assigning different AI models to each SDD phase) is supported by **OpenCode** through the OpenCode-compatible multi-mode overlay. All other agents run in **single-mode** — the orchestrator manages everything using whatever model the agent is already running.


> **Pi multi-mode** lo gestionan los paquetes de Pi. `gentle-pi` instala los recursos de agentes SDD y cadenas en `.pi/agents/` y `.pi/chains/`; las asignaciones de modelo viven en esos ficheros o en los pasos de las cadenas.

---

## Agent Notes

### Claude Code

- El adaptador `claude-code` cubre los tres entornos que comparten el motor de Claude Code y leen `.claude/`, `CLAUDE.md` y `.mcp.json`: el CLI, la extensión de VS Code y Claude Desktop. Se detecta si existe cualquiera de estas señales, aunque no haya CLI:
  - CLI: el binario `claude` en el `PATH`.
  - Extensión de VS Code: una carpeta `anthropic.claude-code-*` en `~/.vscode/extensions`.
  - Claude Desktop: en Windows, `%APPDATA%\Claude` o `%LOCALAPPDATA%\AnthropicClaude`; en macOS, `/Applications/Claude.app` o `~/Library/Application Support/Claude`.
- Sub-agents via the native Task tool with isolated context windows
- Slash commands for SDD phases are installed as `/sdd-*` (`/sdd-init`, `/sdd-new`, `/sdd-continue`, etc.) in `~/.claude/commands/`; install and sync remove the retired `gentle-sdd-*` command files left by older installations
- Servidores MCP (Context7 y Engram): con `--scope global` se registran en `~/.claude.json`, el registro de usuario que Claude Code lee; con `--scope workspace` se declaran en `<proyecto>/.mcp.json` (clave `mcpServers`), el fichero del que Claude Code carga los servidores de proyecto. La fusión conserva los servidores y las claves que el usuario ya tuviera. Engram rechaza, sin modificarlo, un `.mcp.json` que no sea JSON válido. Al sincronizar en workspace se migra el fichero `<proyecto>/.claude/mcp/engram.json` que dejaron instalaciones anteriores, que Claude Code nunca leyó: solo se retira si tiene exactamente la forma que genera Axiom
- Output styles in `~/.claude/output-styles/`
- System prompt via markdown sections in `~/.claude/CLAUDE.md`
- Managed hooks in `~/.claude/settings.json`: `UserPromptSubmit` refreshes the skill registry; `SessionStart` and `Stop` maintain the review reminder baseline. The `PreToolUse(Agent)` SDD guard reads the parent-confirmed grouped `AskUserQuestion` preflight from the session transcript the hook runner supplies, prepends the canonical `## SDD Session Preflight` block to every packaged `sdd-*` phase dispatch, and refuses missing, child-session, model-authored, malformed, or unanswered authority. The guard cannot detect a transcript file deliberately edited through a shell, so it defeats prompt shortcuts, not adversarial forgery. Uninstall also removes stale preflight producer hooks from earlier installations.

### OpenCode

- Full multi-agent overlay with 23 named agents in `opencode.json`: `axiom-orchestrator`, the `general` and `explore` agents, the 11 SDD phase agents (`sdd-init` to `sdd-onboard`), the Judgment Day agents (`jd-*`) and the review agents (`review-*`)
- Slash commands for SDD phases (`/sdd-new`, `/sdd-explore`, etc.)
- Native OpenCode `task` subagents; the managed task-result plugin records grouped `question` answers only for root sessions, injects the canonical SDD preflight block into every packaged SDD phase, and refuses missing, forged, child-session, malformed, or expired authority; it also canonicalizes the grouped `question` options before they are shown and accepts picked answers tolerantly, so the preflight never falls back to typed chat answers
- La ejecución en segundo plano se configura con `axiom install` / `axiom sync`, usando `--opencode-background-subagents=auto|on|off` o la variable `AXIOM_OPENCODE_BACKGROUND_SUBAGENTS`.
- CLI precedence is flag, non-empty environment, prior managed state, then `auto`; the interactive OpenCode + SDD installer prompts only when that preference is unresolved
- Los launchers gestionados se encuentran bajo `~/.axiom/bin/` (o `<AXIOM_STATE_DIR>/bin/` si defines esa variable; al activarlos, Axiom retira el lanzador antiguo de `~/.gentle-ai/bin/`) y respetan `OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=false`; reinicia OpenCode después de activarlos.
- `serve`, `attach`, Desktop, and sessions not launched through the managed launcher use the safe foreground fallback
- Background jobs are process-local and non-durable, have no filesystem isolation, and must not be used for dependent phases or parallel writers in one worktree
- The TUI model picker asynchronously discovers the active project's effective providers and models through `opencode models --verbose`, including custom, authenticated, plugin, and dynamic providers
- Only models OpenCode reports with tool-call capability appear as selectable SDD-capable options
- Antes de seleccionar modelos, conecta tus proveedores de IA y vuelve al selector; Axiom no actualiza el catálogo de OpenCode.
- Axiom establece `share: disabled` como valor predeterminado de los agentes SDD de OpenCode por privacidad y conserva los valores gestionados por el usuario, como `manual` o `auto`.
- OpenCode Desktop SDD commands resolve the project with `git rev-parse --show-toplevel || pwd` before acting, avoiding Electron current-working-directory drift.
- Con `--scope workspace`, Axiom escribe en la distribución de proyecto que OpenCode lee, no en `<proyecto>/.config/opencode` (que ningún agente lee): la configuración (overlay SDD, `mcp`, `permission`, `instructions`) va en `<proyecto>/opencode.json` —o en `opencode.jsonc` si ya existe, con sus comentarios—; comandos y plugins, en `<proyecto>/.opencode/{commands,plugins}` (las skills, en la carpeta compartida `.agents/skills`); y los prompts compartidos del modo multi, en `.opencode/prompts/sdd`. Las instrucciones (persona, protocolo de memoria) van en un fichero propio, `.opencode/axiom.md`, que se añade a la lista `instructions` de `opencode.json` sin duplicados y conservando las entradas del usuario; el `AGENTS.md` de la raíz es del equipo y Axiom no lo toca. Un `opencode.json` que no sea JSON válido se rechaza sin modificarlo. Los restos de instalaciones anteriores en `<proyecto>/.config/opencode` no se borran (pueden mezclar ficheros del usuario): Axiom avisa para que se revisen. La instalación global no cambia.

### Gemini CLI

- Sub-agents are experimental: require `experimental.enableAgents: true` in `settings.json`
- Custom sub-agents defined as markdown files in `~/.gemini/agents/`
- Con `--scope workspace`, las instrucciones (persona, orquestador ODD/SDD, protocolo de memoria) van en un fichero propio, `<proyecto>/.gemini/AXIOM.md`, y no en `<proyecto>/.gemini/GEMINI.md`, que Gemini no lee como contexto de proyecto (carga el `GEMINI.md` de la raíz y de sus carpetas padre, más el global). Para que lo cargue, Axiom lo registra en `context.fileName` de `<proyecto>/.gemini/settings.json`, con la ruta relativa `.gemini/AXIOM.md` (Gemini une cada nombre a cada carpeta de búsqueda con `path.join`). Como el `context.fileName` del proyecto sustituye al del usuario y no se fusiona, Axiom escribe una lista que conserva los nombres que el proyecto ya declaraba (una cadena se convierte en lista) o `GEMINI.md` si no declaraba ninguno, y añade `.gemini/AXIOM.md` sin duplicados; así el `GEMINI.md` de la raíz, que es del equipo y Axiom no toca ni crea, sigue cargándose. Manda el `context.fileName` del proyecto: un nombre que solo figure en `~/.gemini/settings.json` deja de cargarse en ese proyecto, y si lo necesitas debes declararlo en `.gemini/settings.json`. Si `context` o `context.fileName` tienen un tipo inesperado, o `settings.json` no es JSON válido, no se modifica y se avisa (en el segundo caso, antes de escribir nada). Un `.gemini/GEMINI.md` de una instalación workspace anterior se retira solo si Axiom prueba que lo escribió (únicamente secciones gestionadas y, como mucho, el texto de persona que genera el instalador); si tiene contenido propio se conserva y se avisa. Los comandos y `mcpServers` siguen en `.gemini/`; las skills van a la carpeta compartida `.agents/skills`, que Gemini lee como alias de `.gemini/skills`. La instalación global no cambia (`~/.gemini/GEMINI.md`). Qwen Code, que comparte este diseño, no cambia todavía.

### Cursor

- Subagentes nativos en `~/.cursor/agents/sdd-{phase}.md` (ficheros instalados por Axiom)
- Skills at `~/.cursor/skills/`
- System prompt in `~/.cursor/rules/axiom.mdc`. `axiom sync` retira el antiguo `gentle-ai.mdc` solo si lo escribió Axiom; si tiene contenido propio lo conserva y avisa
- MCP config in `~/.cursor/mcp.json`

### VS Code Copilot

- Uses the `runSubagent` tool with support for parallel execution
- Skills at `~/.copilot/skills/`
- System prompt at `Code/User/prompts/axiom.instructions.md`. `axiom sync` retira el antiguo `gentle-ai.instructions.md` solo si lo escribió Axiom; si tiene contenido propio lo conserva y avisa
- MCP config at `Code/User/mcp.json`
- Con `--scope workspace`, Axiom escribe en la distribución de proyecto que VS Code lee, y nada en el perfil de usuario de VS Code (`%APPDATA%\Code\User`, `~/.config/Code/User`, `~/Library/Application Support/Code/User`) ni en `~/.copilot`:
  - **Instrucciones**: persona, orquestador ODD/SDD y protocolo de memoria van en `<proyecto>/.github/instructions/axiom.instructions.md`, un fichero propio de Axiom cuya cabecera declara `applyTo: "**"` para que VS Code lo aplique a todas las peticiones (sin `applyTo` solo se adjuntaría a mano). Se suman a las instrucciones de usuario. `.github/copilot-instructions.md` y el `AGENTS.md` de la raíz son del equipo y Axiom no los toca, ni siquiera para retirar restos de instalaciones antiguas. Las rutas del orquestador que apuntaban a `.agent/skills` se resuelven a `.agents/skills`.
  - **Skills**: `<proyecto>/.agents/skills/<nombre>/SKILL.md`, la carpeta compartida (las sesiones locales de VS Code la leen junto a `.github/skills`). Una copia anterior en `.github/skills` se migra como se explica en «Skills en `--scope workspace`».
  - **MCP** (Context7 y Engram): `<proyecto>/.mcp.json`, con la clave raíz `mcpServers`, el destino de proyecto del arnés de Copilot ([referencia de VS Code](https://code.visualstudio.com/docs/agents/reference/mcp-configuration)). Es el mismo fichero que lee Claude Code: cada servidor tiene una sola entrada válida para los dos (Context7 por `npx`, con `type: "stdio"` y `tools: ["*"]`), y si se desinstala uno de los dos agentes, las entradas se quedan para el otro. La orden de Engram que ya figure en el fichero se respeta. Un `.mcp.json` que no sea JSON válido se rechaza sin modificarlo y antes de escribir nada.
  - **`.vscode/mcp.json`**: es el formato `servers` del arnés Local, deprecado y que VS Code retira en febrero de 2027; el arnés de Copilot no lo lee. `axiom sync` y la desinstalación retiran de él solo las entradas de Engram y Context7 que escribió Axiom (con la misma forma y sin claves añadidas); los servidores, las `inputs` y los comentarios del usuario se conservan, y el fichero se borra solo si se queda vacío. Si no es JSON válido, no se toca.
  - **Agentes**: un `.agent.md` por fase SDD y de Judgment Day en `<proyecto>/.github/agents/`, el formato compartido que se describe en [Agentes por fase de Copilot](#agentes-por-fase-de-copilot).
  - **Hooks**: `<proyecto>/.github/hooks/axiom-branch-guard.json`, un fichero propio de Axiom con la guarda de rama (`axiom branch guard --agent vscode-copilot --event pre|post`) en `preToolUse` y `postToolUse` del arnés de Copilot ([hooks de VS Code](https://code.visualstudio.com/docs/agent-customization/hooks)), limitada a las herramientas de escritura (`create`, `edit`, `str_replace_editor`, `apply_patch`). Antes de escribir, bloquea un documento de Axiom en la rama equivocada; después, avisa al modelo del resto de ficheros. La orden termina siempre con código 0 (`|| true` en bash, `; exit 0` en PowerShell), porque en `preToolUse` cualquier otro código deniega la escritura: si falta `axiom` en el `PATH`, la escritura se permite. Los hooks de VS Code están en preview, solo corren en una carpeta de confianza y se apagan con `chat.useHooks: false` en los ajustes de usuario. `axiom sync` reescribe ese fichero y la desinstalación lo borra si no lo editaste; los demás ficheros de `.github/hooks` son del equipo y no se tocan. El arnés de Copilot lee además los hooks de `.claude/settings.json` del repositorio, así que con Claude Code instalado también ejecuta su guarda. Las dos usan el mismo motor y dan el mismo veredicto, así que el resultado es el mismo.
  - **Preflight SDD**: no se instala en VS Code. El de Claude Code toma la autoridad de la transcripción de la sesión, y los hooks de Copilot no la ofrecen donde haría falta: `preToolUse` no trae la transcripción y `subagentStart` no puede bloquear. El preflight sigue en las instrucciones del orquestador.
  - **Prompts**: Axiom no instala ninguno para VS Code Copilot, así que no se crea `.github/prompts`. Tampoco se escriben permisos: VS Code no tiene un `deny` de proyecto y Axiom no impone modos de autoaprobación.
  - **Confianza**: VS Code solo arranca los servidores MCP de `.mcp.json` (y los de `.vscode/mcp.json`) si el espacio de trabajo es de confianza (Workspace Trust). Axiom avisa al terminar y no concede esa confianza.
  - GitHub Copilot CLI, que solo tiene ámbito global en Axiom (véase [su sección](#github-copilot-cli)), lee también `.github/instructions`, `.github/skills` y `.github/agents`. La carpeta `.agents/skills` es la compartida que escribe Axiom en workspace.
  - La instalación global no cambia.

### GitHub Copilot CLI

Agente de terminal de GitHub (binario `copilot`). Es un adaptador propio (`github-copilot-cli`), distinto del de VS Code Copilot: solo comparten la carpeta de skills.

- **Raíz**: `$COPILOT_HOME` si está definida y es una ruta absoluta, o `~/.copilot`. `COPILOT_HOME` solo se respeta con el directorio personal real del usuario: una instalación aislada (otra raíz de instalación) se queda dentro de ella.
- **Instrucciones**: `<raíz>/copilot-instructions.md` (persona, orquestador ODD/SDD y protocolo de memoria). El SDD es el mismo orquestador genérico de VS Code Copilot, con la elección de almacén de artefactos en la preflight de sesión.
- **Skills**: `<raíz>/skills/`. Sin `COPILOT_HOME` es la misma carpeta que usa VS Code Copilot. La copia no depende de qué agente la escribe (lleva el hueco genérico de runtime), y desinstalar uno de los dos no retira las skills mientras el otro siga instalado.
- **MCP** (Context7 y Engram): `<raíz>/mcp-config.json`, con la clave raíz `mcpServers` (no `servers`, la de VS Code) y entradas con `type` (`http` para Context7, `local` para Engram) y `tools: ["*"]`. Se conservan los servidores del usuario. Engram se inyecta directamente, sin `engram setup`. CodeGraph tiene un destino nativo `copilot-cli` que escribe en el mismo fichero.
- **Ajustes y permisos**: `<raíz>/settings.json`. Axiom no escribe permisos ni modos de autoaprobación: lo que decidas queda intacto.
- **Agentes**: ninguno en `<raíz>/agents/`. Los agentes por fase viven solo en `.github/agents` del proyecto, que Copilot CLI también lee (véase [Agentes por fase de Copilot](#agentes-por-fase-de-copilot)).
- **Hooks y comandos**: ninguno.
- **Detección**: el binario `copilot` en el `PATH` o una entrada propia de Copilot CLI en la raíz:
  - su estado: `config.json`, `session-state/`, `session-store.db`, `logs/`, etc.;
  - o lo que solo él lee: `settings.json`, `mcp-config.json` y `copilot-instructions.md`.

  `skills/` y `agents/` las lee también VS Code Copilot, y una entrada desconocida no cuenta, así que una `~/.copilot` sin ninguna de esas entradas no se toma por Copilot CLI; y VS Code Copilot se detecta en el escaneo por su extensión `github.copilot`, no por `~/.copilot`.
- **Instalación** (solo se muestra, Axiom no la ejecuta): `npm install -g --ignore-scripts @github/copilot@latest`, con `sudo` en Linux si el npm es del sistema. `axiom doctor` comprueba el binario `copilot`.
- **Ámbito workspace**: no tiene. Con `--scope workspace` se omite con un aviso. Lo único que Axiom escribe en el proyecto por él son los agentes por fase de `.github/agents`: el sync y la instalación de proyecto los escriben cuando Copilot CLI consta en el estado del proyecto o está configurado para el usuario.

### Agentes por fase de Copilot

VS Code Copilot y Copilot CLI comparten un único fichero `.agent.md` por fase SDD y de Judgment Day, con el mismo contenido byte a byte. **Viven solo en la carpeta del proyecto, `<proyecto>/.github/agents/`**: los dos la leen, y es donde se guarda el modelo elegido por fase (en el estado del proyecto).

Axiom no los escribe en `~/.copilot/agents/` (el ámbito global de Copilot CLI y de VS Code Copilot no genera agentes por fase). El motivo es que Copilot CLI da prioridad al agente personal frente al del proyecto cuando coinciden los identificadores: un agente global sin el modelo elegido taparía el del proyecto y la elección del dashboard no se aplicaría.

| Caso | Quién escribe `.github/agents/` |
| --- | --- |
| Proyecto con VS Code Copilot instalado en el proyecto | el adaptador de VS Code Copilot |
| Proyecto con solo Copilot CLI (consta en el estado del proyecto o está configurado para el usuario) | el sync o la instalación de proyecto, porque Copilot CLI no tiene modo workspace propio |
| Los dos | el adaptador de VS Code Copilot; el fichero se escribe una sola vez y las sincronizaciones siguientes no lo reescriben |

- **Frontmatter**: `name` y `description` siempre. `model` y `reasoning-effort` solo cuando hay una asignación para esa fase; sin ella el agente hereda el modelo de la conversación.
- **Formato del modelo**: `<Nombre> (copilot)`, por ejemplo `Claude Sonnet 4.6 (copilot)`. Se elige desde el dashboard (Ecosistema → Modelos, véase [Uso](usage.md#modelo-por-fase-desde-el-dashboard)) y se guarda en `copilot_phase_assignments` del estado del proyecto.
- **Copilot CLI**: acepta ese formato con sufijo desde la versión 1.0.24. No consta en la documentación que lea `reasoning-effort`, así que ahí puede ignorarse.
- **Sin fichero `orchestrator`**: el modelo de la conversación principal no se puede fijar desde un fichero de agente, y por eso Axiom no genera ninguno para el coordinador.
- **Migración**: un `axiom sync`, `axiom install` o `axiom upgrade` global retira de `~/.copilot/agents/` (y de `$COPILOT_HOME/agents/`) los agentes por fase que una versión anterior de Axiom escribió ahí, identificados por el nombre de las plantillas. Los agentes que escribiste tú con otros nombres se conservan, y sin ficheros que retirar no hace nada.
- **Desinstalación**: `axiom uninstall` retira de `.github/agents/` solo los agentes que Axiom gestiona y deja los del equipo. Mientras el otro adaptador de Copilot siga en el estado del proyecto, los ficheros compartidos se conservan.
- **Aviso con Claude Code**: VS Code también lee `.claude/agents` y `~/.claude/agents`. En un proyecto con Claude Code y VS Code Copilot, los agentes de Claude (con `model: sonnet`) aparecen también en VS Code, junto a los de Copilot.

Fuentes: [referencia del fichero de agente personalizado de VS Code](https://code.visualstudio.com/docs/agents/reference/custom-agent-file-reference) y [agentes personalizados de Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli).

### Codex

- CLI-native agent with TOML config at `~/.codex/config.toml`
- Skills at `~/.codex/skills/`
- System prompt at `~/.codex/AGENTS.md`
- Engram instruction files at `~/.codex/engram-instructions.md`
- MCP servers (Engram and Context7) are upserted as `[mcp_servers.<name>]` blocks in `~/.codex/config.toml`
- SDD model-selection profiles written as separate files at `~/.codex/<name>.config.toml`. GPT-5.6 defaults require Codex >= 0.144.0 (the separate-file mechanism itself is available since 0.134.0). Select a profile at runtime via `codex --profile <name>`:

  Los valores predeterminados de modelo y esfuerzo varían según el preset. Estos niveles de esfuerzo son la política de carga de trabajo de Axiom, no valores predeterminados de Codex. Los carriles distinguen el trabajo de cada fase: `sdd-strong` razona sobre el contexto recibido, `sdd-mid` implementa en un bucle agentic y `sdd-cheap` transcribe estructuras con contexto breve y resultados verificables.

  Every curated preset runs the main orchestrator/session at `medium` effort — it plans, routes and adjudicates rather than doing the delegated work. The orchestrator *model* varies: Low-cost runs it on `gpt-5.6-terra` so a Plus plan can still afford `gpt-5.6-sol` in the strong carril, where the reasoning pays. Custom and legacy state preserve existing top-level settings:

  | Profile | Low-cost | Recommended | Powerful | SDD phases |
  |---------|----------|-------------|----------|------------|
  | Orchestrator | `gpt-5.6-terra` / `medium` | `gpt-5.6-sol` / `medium` | `gpt-5.6-sol` / `medium` | main session |
  | `sdd-strong` | `gpt-5.6-sol` / `medium` | `gpt-5.6-sol` / `medium` | `gpt-5.6-sol` / `xhigh` | explore, propose, design, verify, judge |
  | `sdd-mid` | `gpt-5.6-terra` / `medium` | `gpt-5.6-terra` / `high` | `gpt-5.6-sol` / `high` | apply, fix-agent |
  | `sdd-cheap` | `gpt-5.6-luna` / `high` | `gpt-5.6-luna` / `high` | `gpt-5.6-luna` / `high` | spec, tasks, archive, onboard |

- Explicit saved Codex model assignments are preserved on sync, including older pinned IDs such as `gpt-5.5` or `gpt-5.4-mini`. The narrow exception is the exact former implicit-default tuple (`sdd-strong=gpt-5.5`, `sdd-mid=gpt-5.5`, `sdd-cheap=gpt-5.4-mini`), which sync treats as Recommended and upgrades to the current GPT-5.6 tuple; partial, extended, or otherwise different maps remain custom and unchanged.
- GPT-5.6 `max` reasoning effort and `ultra` mode are intentionally not enabled by this default update. `max` requires confirmed Codex support; `ultra` changes orchestration semantics and needs separate design.
- La delegación SDD multiagente queda habilitada por defecto: Axiom escribe `features.multi_agent = true` y `agents.max_threads = 4` / `agents.max_depth = 2` en `~/.codex/config.toml`. Para desactivarla, configura `multi_agent = false` en `[features]`. La delegación requiere ese ajuste y las herramientas nativas de Codex `spawn_agent`, `wait_agent` y `list_agents`; si falta cualquiera, la orquestación continúa en modo de agente único.
- **Delegation**: Native multi-agent by default, with graceful solo-agent fallback
- Con `--scope workspace`, Axiom escribe en la distribución de proyecto que Codex lee, no en `~/.codex` ni en `<proyecto>/.codex/AGENTS.md` (Codex no lo carga; lee el `AGENTS.md` de la raíz, que es del equipo y Axiom no toca). Todo exige que el proyecto sea de confianza.
  - **Configuración**: `<proyecto>/.codex/config.toml`, que prevalece sobre `~/.codex/config.toml` clave a clave. La fusión conserva todo lo del usuario (claves, comentarios, tablas y servidores MCP propios). Un `config.toml` que no sea TOML válido se rechaza sin modificarlo y antes de escribir nada. Axiom escribe ahí `[mcp_servers.engram]`, `[mcp_servers.context7]`, `features.multi_agent = true`, `agents.max_threads` / `agents.max_depth`, el modelo del orquestador si lo elegiste y `experimental_compact_prompt_file = "engram-compact-prompt.md"` (ruta relativa a `.codex/`; si ya declaras el tuyo se respeta y se avisa). Nunca escribe las claves que Codex ignora en la configuración de proyecto (`openai_base_url`, `chatgpt_base_url`, `apps_mcp_product_sku`, `model_provider`, `model_providers`, `notify`, `profile`, `profiles`, `experimental_realtime_ws_base_url`, `otel`), ni `model_instructions_file`, que sustituye las instrucciones base en lugar de añadirse a ellas.
  - **Instrucciones**: persona, orquestador ODD/SDD, protocolo de memoria (con la captura pasiva) y guía de enrutado van en la clave `developer_instructions` de `.codex/config.toml`, que Codex añade a sus instrucciones base. Axiom las compone en `.codex/axiom-instructions.md` y copia ese contenido a un bloque delimitado por `<!-- axiom:workspace-instructions -->` dentro del valor de `developer_instructions`; el texto que ya tuvieras en esa clave queda fuera del bloque y no se toca, y repetir `sync` no cambia ni un byte. El valor se escribe como cadena multilínea literal (o básica con escapes si el texto contiene tres comillas simples seguidas o caracteres de control). Si `developer_instructions` existe con un tipo que no es cadena, no se modifica y se avisa. Un `.codex/AGENTS.md` de una instalación workspace anterior se retira solo si Axiom prueba que lo escribió; si tiene contenido propio se conserva y se avisa.
  - **Hooks**: `<proyecto>/.codex/hooks.json` (registro de skills al iniciar la sesión y guarda de rama de `apply_patch`). Codex pide aprobar cada hook por su hash.
  - **Skills**: `<proyecto>/.agents/skills`, la única carpeta de skills de proyecto que documenta Codex, y las rutas del orquestador que antes apuntaban a `~/.codex/skills` se resuelven a `.agents/skills`. Esa carpeta la leen también otros agentes (Gemini CLI, OpenCode, Cursor, VS Code Copilot), y Axiom escribe una sola copia para todos: ya no instala las skills además en la carpeta propia de cada uno. Un `.codex/skills` de una instalación anterior se migra como se explica en «Skills en `--scope workspace`».
  - **Sin escritura en HOME**: Axiom no escribe la confianza del proyecto en `~/.codex/config.toml`. Al terminar avisa de que Codex pedirá confiar en el proyecto la primera vez y de que cada hook se aprueba por hash. Tampoco se escriben los ficheros de perfil `<carril>.config.toml` (Codex ignora `profiles` en proyecto), ni se valida el ejecutable de Codex, que solo servía para esos perfiles. Los permisos no se escriben en ningún ámbito: Codex usa los suyos.
  - **`engram setup codex`**: con `--scope workspace` Axiom no lo ejecuta, porque escribe en `~/.codex`, y lo avisa como acción manual. El MCP de Engram ya queda declarado en `.codex/config.toml`, así que el comando solo hace falta si además quieres el registro de usuario; ejecutarlo escribe fuera del proyecto. Con `--scope global` se ejecuta como siempre.
  - La instalación global no cambia.

### Antigravity

- Skills at `~/.gemini/antigravity/skills/` (native Antigravity feature)
- MCP config at `~/.gemini/antigravity/mcp_config.json`
- System prompt appended to `~/.gemini/GEMINI.md` (shared with Gemini CLI — collision check warns if both are installed)
- Mission Control handles built-in sub-agent delegation (Browser, Terminal) automatically
- Settings managed via the IDE's Agent settings UI, not via `settings.json`

### Qwen Code (no disponible de momento)

- **Detección**: Axiom detecta Qwen Code por su raíz de configuración (`~/.qwen`) y comprueba si el ejecutable `qwen` está en `PATH`.
- **Config root**: `~/.qwen/` (cross-platform)
- **System prompt**: `~/.qwen/QWEN.md` (managed via `StrategyFileReplace`)
- **Skills**: `~/.qwen/skills/`
- **MCP config**: `~/.qwen/settings.json` (managed via `StrategyMergeIntoSettings` with `mcpServers` key)
- **Slash commands**: `~/.qwen/commands/*.md` — supports custom namespaced slash commands (e.g., `commands/sdd/init.md` → `/sdd:init`)
- **Permisos**: Axiom no inyecta permisos en Qwen Code ni impone el modo `auto_edit`; los valores del usuario no se tocan.
- **Install**: via npm — `npm install -g @qwen-code/qwen-code@latest`
- **Engram slug**: `"qwen-code"` for `engram setup` integration
- **SDD orchestrator**: `internal/assets/qwen/sdd-orchestrator.md` with Qwen-specific path references

### Trae

- **Detección**: Axiom detecta Trae mediante `~/.trae` (aplicación de escritorio, sin ejecutable en `PATH`).
- **Global config root**: `~/.trae/` (cross-platform)
- **Skills**: `~/.trae/skills/`
- **System prompt / rules**: injected via `StrategyMarkdownSections` into the OS-specific `user_rules.md`
  - macOS: `~/Library/Application Support/Trae/User/user_rules.md`
  - Linux: `~/.config/Trae/User/user_rules.md` (respects `XDG_CONFIG_HOME`)
  - Windows: `%APPDATA%\Trae\User\user_rules.md`
- **MCP config**: same OS-specific dir → `mcp.json` (Cursor-compatible `mcpServers` object format)
- **Install**: desktop app only — manual install required; no `--auto-install` support

### Pi

For the full Pi command and package reference, see [Pi Agent](pi.md).

- **Detección**: Axiom detecta Pi mediante el ejecutable `pi` en `PATH` y la raíz `~/.pi`.
- **Ámbito**: solo con `--scope global`. Con `--scope workspace` se omite con un aviso y Axiom no lanza ningún comando de Pi.
- **Instalación**: Pi debe estar instalado antes. Axiom aprovisiona entonces el conjunto de integración de Pi con:
  - `pi install npm:gentle-pi`
  - `pi install npm:gentle-engram`
  - `pi install npm:pi-mcp-adapter`
  - `npm exec --yes --package gentle-engram@latest -- pi-engram init`
  - `pi install npm:@juicesharp/rpiv-ask-user-question`
  - `pi install npm:pi-web-access`
  - `pi install npm:pi-btw`
- **Paquete `gentle-pi`**: el paquete externo aporta a Pi el flujo SDD/OpenSpec, directrices TDD, valores de seguridad, comandos `/gentle:*`, skills, prompts, agentes SDD y cadenas. En un `session_start` normal, copia recursos del proyecto a `.pi/agents/`, `.pi/chains/` y `.pi/gentle-ai/support/` sin sobrescribir ficheros locales, salvo que el comando de recuperación de Pi use `--force`. Iniciar Pi con `pi -ns` omite la carga inicial de skills y hooks; en ese modo no se produce esa actualización automática.
- **Package metadata**: latest verified `gentle-pi` version is `2.5.0`; npm lists `alan_buscaglia` as maintainer, with source at [Gentleman-Programming/gentle-pi](https://github.com/Gentleman-Programming/gentle-pi) and package docs at [npm: gentle-pi](https://www.npmjs.com/package/gentle-pi).
- **Cambio de persona**: `gentle-pi` gestiona `/gentle:persona`, que alterna entre `gentleman` y `neutral`, guarda `.pi/axiom/persona.json` y puede requerir `/reload` o una nueva sesión para actualizar el prompt activo.
- **Asignación de modelos**: `gentle-pi` gestiona `/gentle:models`. Abre un diálogo nativo de Pi para agentes de proyecto, usuario e integrados, prioriza los agentes SDD, guarda `.pi/gentle-ai/models.json` y aplica los cambios en `.pi/agents/*.md` o `.pi/settings.json`.
- **`gentle-engram` package**: adds persistent Engram memory for Pi. It captures sessions, exposes Engram MCP tools through `pi-mcp-adapter`, and degrades safely when the local `engram` binary is missing.
- **MCP adapter wiring**: ComponentEngram declares `npm:pi-mcp-adapter` in `.pi/agent/settings.json` packages and adds `pi-mcp-adapter` `^2.6.0` to `.pi/npm/package.json` without removing unrelated user entries. `pi-engram init` owns the Pi Engram MCP config schema and is run during installation.
- **Subagentes**: `gentle-pi` proporciona Gentle Agents para descubrir y ejecutar agentes SDD desde `.pi/agents/`, con las herramientas `subagent_*` que antes ofrecía `npm:pi-subagents-j0k3r`. Axiom no instala ese paquete por separado; si ya existe una entrada `npm:pi-subagents-j0k3r`, se elimina de `settings.json` durante la siguiente instalación o actualización y Pi lo desinstala al sincronizar sus paquetes.
- **Subagentes en segundo plano**: se configuran con `axiom install` / `axiom sync`, mediante `--pi-background-subagents=auto|on|off` o la variable `AXIOM_PI_BACKGROUND_SUBAGENTS`. No hay launcher ni activación adicional, porque la primitiva es la extensión `pi-subagents-j0k3r` ya instalada.
- CLI precedence is flag, non-empty environment, prior managed state, then `auto`; `auto` never enables by itself, unresolved non-interactive `auto` stays foreground, and the interactive Pi installer prompts only when that preference is unresolved.
- The resolved on/off policy is projected to `~/.pi/axiom/background-subagents.json` as `{"schema":"gentle-pi.background-subagents/v1","policy":"on"|"off"}` (the base directory honors `GENTLE_PI_CONFIG_HOME`); `off` rewrites the policy instead of deleting files, and a file at that path without the managed schema marker is never overwritten.
- **`@juicesharp/rpiv-ask-user-question` package**: lets Pi child agents ask the active user session for clarification when they need human input.
- **Pi companion packages**: `pi-web-access` and `pi-btw` add web access and companion workflow support. Todo tracking ships inside `gentle-pi` (Gentle Todo); an existing `@juicesharp/rpiv-todo` entry is dropped from `settings.json` on the next install or update, and Pi uninstalls it on its next package sync.
- **Solo Pi**: cuando Pi es el único agente seleccionado, Axiom omite las preguntas de persona, componentes del ecosistema y TDD estricto, porque las gestiona `gentle-pi`.
