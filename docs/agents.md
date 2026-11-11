# Agentes compatibles con Axiom

← [Back to README](../README.md)

---

## Agent Matrix

| Agent           | ID               | Skills       | MCP | Delegation                       | Output Styles | Slash Commands | Config Path                         |
| --------------- | ---------------- | ------------ | --- | -------------------------------- | ------------- | -------------- | ----------------------------------- |
| Claude Code     | `claude-code`    | Yes          | Yes | Full (Task tool)                 | Yes           | Yes            | `~/.claude`                         |
| OpenCode        | `opencode`       | Yes          | Yes | Full (multi-mode overlay)        | No            | Yes            | `~/.config/opencode`                |
| Kilo Code       | `kilocode`       | Yes          | Yes | Full (multi-mode overlay)        | No            | Yes            | `~/.config/kilo`                    |
| Gemini CLI      | `gemini-cli`     | Yes          | Yes | Full (experimental)              | No            | No             | `~/.gemini`                         |
| Cursor          | `cursor`         | Yes          | Yes | Full (native subagents)          | No            | No             | `~/.cursor`                         |
| VS Code Copilot | `vscode-copilot` | Yes          | Yes | Full (runSubagent)               | No            | No             | `~/.copilot` + VS Code User profile |
| GitHub Copilot CLI | `github-copilot-cli` | Yes      | Yes | Orquestador SDD genérico (worker nativo) | No            | No             | `~/.copilot` o `$COPILOT_HOME` (solo global) |
| Codex           | `codex`          | Yes          | Yes | Native multi-agent (default; solo fallback) | No            | No             | `~/.codex`                          |
| Windsurf        | `windsurf`       | Yes (native) | Yes | Solo-agent                       | No            | No             | `~/.codeium/windsurf`               |
| Antigravity     | `antigravity`    | Yes (native) | Yes | Solo-agent + Mission Control     | No            | No             | `~/.gemini/antigravity`             |
| Kimi Code (no disponible de momento) | `kimi`           | Yes          | Yes | Full (native custom agents)      | Via KIMI.md include [^kimi-output-style] | No | `~/.kimi`                           |
| Qwen Code (no disponible de momento) | `qwen-code`      | Yes          | Yes | Full (native sub-agents)         | No            | Yes            | `~/.qwen`                           |
| Kiro IDE        | `kiro-ide`       | Yes          | Yes | Full (native subagents)          | No            | No             | `~/.kiro`                           |
| OpenClaw        | `openclaw`       | Yes          | Yes | Solo-agent                       | No            | No             | `~/.openclaw`                       |
| Trae            | `trae-ide`       | Yes          | Yes | Solo-agent                       | No            | No             | `~/.trae`                           |
| Pi              | `pi`             | Yes          | Yes | Full (package-managed subagents) | No            | Yes            | `~/.pi`                             |
| Hermes (no disponible de momento) | `hermes`         | Yes          | Yes | Full (delegate_task ephemeral)   | No            | No             | `~/.hermes`                         |

La columna «Config Path» es la ruta del ámbito global (`--scope global`). Con `--scope workspace`, el predeterminado, cada agente usa las rutas de proyecto que se resumen en [Modo workspace por agente](#modo-workspace-por-agente).

### Agentes no disponibles de momento

Kimi Code (`kimi`), Qwen Code (`qwen-code`) y Hermes (`hermes`) **no están disponibles de momento**: no se pueden elegir al instalar. No aparecen en la pantalla de selección de agentes de la TUI, no entran en las selecciones por defecto ni en el descubrimiento por carpetas, y `axiom install --agent kimi|qwen-code|hermes` falla con un error (también si va junto a otros agentes). El motivo es que hoy no son utilizables con Axiom: en el ámbito workspace escriben en rutas que el agente no lee y su MCP solo es global. Sus adaptadores se conservan en el código para retomarlos más adelante.

- Una instalación global existente que ya los tiene en el estado se sigue sincronizando con `axiom sync`.
- Con `--scope workspace` se omiten con un aviso, igual que los agentes sin modo workspace.
- `uninstall`, `doctor` y `verify` no cambian.
- Las secciones de Kimi Code, Qwen Code y Hermes de más abajo describen el adaptador conservado, no una integración instalable hoy.

La mayoría de los agentes recibe la política completa del orquestador SDD y las skills correspondientes en su directorio de configuración. Axiom la incorpora al prompt de sistema; OpenCode y Kilo Code la reciben mediante el overlay compatible con OpenCode (`opencode.json`). Pi es la excepción: Axiom aprovisiona paquetes de Pi y `gentle-pi` —nombre del paquete externo— gestiona las skills, los prompts, los agentes SDD y las cadenas durante la ejecución. El agente usa SDD cuando el trabajo lo requiere o cuando se lo pides.

`axiom install --scope=workspace` permite limitar al workspace los ficheros de agente compatibles, no solo los de Claude Code. En ese ámbito, Axiom escribe prompts de sistema, skills, agentes SDD y ficheros de persona en el proyecto actual cuando el agente admite configuración local. Las integraciones globales —por ejemplo, paquetes o ajustes que el agente solo lee desde su configuración de usuario— permanecen globales. Algunos nombres de ruta, paquetes, marcadores y variables `gentle-ai` o `GENTLE_AI_*` se conservan como identificadores heredados de compatibilidad; no son el comando actual del producto, que es `axiom`.

#### Modo workspace por agente

El ámbito workspace es el predeterminado de la CLI y de la TUI. Axiom escribe en la raíz del proyecto (el `axiom.yaml` más cercano o, si no hay, la raíz del repositorio git) y no en tu carpeta de usuario; lo que sigue a nivel de máquina está declarado en [Uso](usage.md#cli-flags-install). Las instrucciones ODD/SDD son del equipo y van siempre en un fichero propio de Axiom, nunca en un fichero raíz compartido (`AGENTS.md`, `GEMINI.md`), y la regla general es que lo del proyecto prevalece sobre lo del usuario y ambos conviven (véanse los matices en «Precedencia»).

| Agente | Instrucciones | Configuración, MCP y hooks | Skills |
| --- | --- | --- | --- |
| Claude Code | `.claude/CLAUDE.md` | `.claude/` (`settings.json`, `agents/`, `commands/`, `output-styles/`); MCP en `.mcp.json` (`mcpServers`) | `.claude/skills` |
| OpenCode | `.opencode/axiom.md`, registrado en `instructions` de `opencode.json` | `opencode.json` (o `opencode.jsonc`) y `.opencode/{commands,plugins,prompts/sdd}` | `.agents/skills` |
| Kilo Code | `.kilo/rules/axiom.md`, registrado en `instructions` | `.kilo/kilo.json` (o el `kilo.jsonc`/`kilo.json` existente) y `.kilo/{commands,plugins,prompts/sdd}` | `.agents/skills` |
| Gemini CLI | `.gemini/AXIOM.md`, registrado en `context.fileName` de `.gemini/settings.json` | `.gemini/settings.json` (`mcpServers`) y `.gemini/` | `.agents/skills` |
| Codex | `developer_instructions` de `.codex/config.toml` (bloque `<!-- axiom:workspace-instructions -->`) y `.codex/axiom-instructions.md` | `.codex/config.toml`, `.codex/hooks.json`; exige proyecto de confianza | `.agents/skills` |
| VS Code Copilot | `.github/instructions/axiom.instructions.md` (`applyTo: "**"`) | `.vscode/mcp.json` (clave `servers`); sin agentes, prompts ni hooks | `.agents/skills` |
| Cursor | `.cursor/rules/axiom.mdc` | `.cursor/` (`agents/`, `mcp.json`) | `.agents/skills` |
| Kiro IDE | `.kiro/steering/axiom.md` | `.kiro/` (`agents/`, `settings/mcp.json`, `hooks/`) | `.kiro/skills` |

- **Sin modo workspace.** Windsurf, Trae, Antigravity, Pi, OpenClaw y GitHub Copilot CLI solo tienen configuración global: con `--scope workspace` se omiten con un aviso en castellano y Axiom nunca cae a escribir en su configuración de usuario. Para instalarlos hay que pedir `--scope global` de forma expresa. Pi se omite porque `pi install -l` solo cubre los paquetes (`.pi/settings.json`), mientras que `pi-engram init`, `~/.pi/npm/package.json` y el prompt de sistema solo tienen ruta de usuario. OpenClaw se omite porque su configuración, incluido el MCP de Engram, vive en `~/.openclaw/openclaw.json` y no documenta configuración por proyecto. Antigravity queda aplazado: sus reglas de proyecto limitan el tamaño de cada fichero y el orquestador de Axiom no cabe en una sola regla.
- **No seleccionables.** Kimi Code, Qwen Code y Hermes ya no se pueden elegir (véase [Agentes no disponibles de momento](#agentes-no-disponibles-de-momento)).
- **Precedencia.** Lo del proyecto prevalece sobre lo del usuario y ambos conviven, con dos matices. En Claude Code, una skill personal con el mismo nombre que una de proyecto gana a esta. En Gemini CLI, el `context.fileName` del proyecto sustituye al del usuario y no se fusiona: Axiom escribe una lista que conserva los nombres que el proyecto ya declaraba (o `GEMINI.md`) y añade `.gemini/AXIOM.md`, de modo que un nombre que solo figure en `~/.gemini/settings.json` deja de cargarse en ese proyecto.
- **GitHub Copilot CLI.** Tiene adaptador (`github-copilot-cli`), pero solo global: lo que Axiom le escribe vive en `~/.copilot` (o `$COPILOT_HOME`) y no hay una ruta de proyecto documentada para ello, así que se omite en workspace. El CLI sí lee del proyecto `.github/instructions`, `.github/skills` y el `.mcp.json` de Claude Code, así que se beneficia de lo que Axiom escribe en workspace para VS Code Copilot y Claude Code.
- **Lo que no escribe en `~/`.** `engram setup <agente>` se omite con un aviso, porque escribe en la carpeta de usuario del agente; el MCP de Engram ya queda en el proyecto. Con Codex, el aviso sugiere `engram setup codex` como acción manual opcional: ejecútalo solo si quieres además el registro de usuario, porque escribe en `~/.codex`, fuera del proyecto.

#### Skills en `--scope workspace`: una sola copia compartida

Con `--scope workspace`, Axiom escribe las skills en el menor número de carpetas. Los agentes que leen `<proyecto>/.agents/skills` comparten una única copia, con el mismo contenido para todos. Antes cada agente recibía una copia propia y varios leen además otras carpetas, así que veían cada skill varias veces (por ejemplo, avisos `duplicate skill name` en OpenCode). Claude Code no lee `.agents/skills` y conserva la suya.

| Agente | Carpeta de skills de proyecto | Qué dice su documentación |
| --- | --- | --- |
| Codex | `.agents/skills` | «Codex scans `.agents/skills` in every directory from your current working directory up to the repository root» ([learn.chatgpt.com/docs/build-skills](https://learn.chatgpt.com/docs/build-skills.md)). |
| VS Code Copilot | `.agents/skills` | «Local sessions also read project skills from `.claude/skills/` and `.agents/skills/`» ([code.visualstudio.com/docs/copilot/customization/agent-skills](https://code.visualstudio.com/docs/copilot/customization/agent-skills)). Las sesiones locales son las de la vista de chat. La tabla de esa página da `.github/skills` como carpeta de proyecto del arnés de Copilot, y la documentación de GitHub lista `.agents/skills` entre las ubicaciones de proyecto de Copilot. |
| OpenCode | `.agents/skills` | Busca en `.opencode/skills`, `.claude/skills` y `.agents/skills` ([opencode.ai/docs/skills](https://opencode.ai/docs/skills/)). |
| Kilo | `.agents/skills` | «`~/.agents/skills/` and `.agents/skills/` - Open agent standard, loaded by default» ([kilo.ai/docs/customize/skills](https://kilo.ai/docs/customize/skills)). |
| Gemini CLI | `.agents/skills` | `.agents/skills` es un alias de `.gemini/skills` y gana a esta en el mismo nivel ([docs/cli/skills.md](https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md)). |
| Cursor | `.agents/skills` | Carga `.agents/skills` y `.cursor/skills` como carpetas de proyecto ([cursor.com/docs/context/skills](https://cursor.com/docs/context/skills)). |
| Claude Code | `.claude/skills` | No lee `.agents/skills` ([code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills)). Solo se escribe si Claude Code está entre los agentes. |
| Kiro | `.kiro/skills` | Su documentación solo cita `.kiro/skills`; no confirma que lea `.agents/skills`, así que conserva su carpeta ([kiro.dev/docs/skills](https://kiro.dev/docs/skills/)). |

- La carpeta compartida se escribe una sola vez por ejecución, aunque varios agentes la usen, y no aparece repetida en la copia de seguridad ni en la verificación. Con solo Claude Code no se crea `.agents/skills`.
- Las rutas de skills que Axiom escribe en los prompts, overlays y orquestadores (`{{AXIOM_SKILLS_DIR}}` y las rutas `_shared`) apuntan a la carpeta efectiva de cada agente, y el índice de skills del bloque `axiom:skills-index` de `AGENTS.md` prefiere `.agents/skills` cuando existe (una skill en `skills/` o en una raíz declarada sigue ganando).
- Los ficheros de soporte `_shared` de la copia compartida llevan el hueco genérico `<runtime>` en lugar de la identidad de un agente concreto, igual que la copia de compatibilidad de `~/.agents/skills`, para que su contenido no dependa de qué agente la escriba.
- **Migración.** Las instalaciones workspace anteriores dejaron copias en `.github/skills`, `.opencode/skills`, `.kilo/skills`, `.gemini/skills`, `.cursor/skills` y `.codex/skills`. Al instalar o sincronizar, Axiom retira una carpeta de skill solo si prueba que es suya: tiene el nombre de una skill que Axiom distribuye, contiene exactamente los ficheros que Axiom genera y cada uno es byte a byte igual a lo que Axiom genera hoy, y la copia compartida ya existe. Si difiere o tiene ficheros de más, se conserva y se avisa para que la revises. Las skills ajenas no se tocan, y la carpeta contenedora (`.github/skills`, etc.) solo se borra si queda vacía. Los ficheros que se pueden retirar van en la copia de seguridad previa.
- El ámbito global no cambia: cada agente sigue escribiendo en su carpeta de usuario.

[^kimi-output-style]: Kimi has no `settings.json` `outputStyle` mechanism like Claude Code. Instead, `KIMI.md` unconditionally includes `output-style.md` as a Jinja module — the canonical tone/language/philosophy channel for Kimi's persona (`persona.md` carries only tooling/action directives plus a pointer to this module).

---

## Delegation Models

| Model                 | How It Works                                                                                                                                                                                       | Agents                                                                                                    |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Full (sub-agents)** | Each SDD phase runs in an isolated context window via native sub-agent delegation, package-managed subagents, or an OpenCode-compatible overlay. The orchestrator coordinates; sub-agents execute. | Claude Code, OpenCode, Kilo Code, Gemini CLI, Cursor, VS Code Copilot, Kiro IDE, Pi (Kimi Code y Qwen Code, con adaptador conservado pero no seleccionable) |
| **Full (delegate_task)** | The orchestrator uses Hermes's native `delegate_task` primitive to spawn ephemeral workers in fresh context windows. Workers receive only a self-contained mission; the parent receives only their final summary. Toolsets, MCP, and skills must be passed explicitly (not inherited by default). | Hermes |
| **Native multi-agent** | The orchestrator delegates through the agent's native collaboration tools when configured and available, with inline execution as a graceful fallback. | Codex |
| **Solo-agent**        | All SDD phases run inline in the same conversation. The orchestrator IS the executor. Engram™ provides cross-phase persistence.                                                                     | Windsurf, Antigravity, OpenClaw, Trae                                                                     |

### Cursor Native Subagents

Cursor usa su sistema integrado `.cursor/agents/`. Axiom escribe ficheros de agentes en `~/.cursor/agents/sdd-{phase}.md`; Cursor delega según el campo `description` de la cabecera YAML de cada fichero.

- `sdd-explore` and `sdd-verify` run with `readonly: false` so they can inspect the codebase and execute verification commands
- Each subagent gets its own context window (fresh context, no pollution)
- The orchestrator resolves skill paths from the skill registry and passes exact `SKILL.md` files in the invocation message

### Windsurf Cascade

Windsurf runs as a solo-agent (no custom sub-agents). The orchestrator leverages Windsurf-native features:

- **Plan Mode** — creates persistent plan documents that can be @mentioned across sessions; ideal for spec and design artifacts on large changes
- **Code Mode** — default agentic execution mode
- **Native Workflows** — `sdd-new` is available as a `.windsurf/workflows/sdd-new.md` workflow
- **Size Classification** — the orchestrator routes tasks through Small/Medium/Large decision paths

### Antigravity + Mission Control

Antigravity is an agent-first platform with built-in sub-agents (Browser, Terminal) managed by Mission Control. However, custom sub-agent creation is not yet available. SDD phases run inline, with Mission Control handling automatic delegation to built-in sub-agents when specialized tooling is needed (e.g., Browser for research during `sdd-explore`).

### Kiro Native Subagents

Kiro usa agentes personalizados en `~/.kiro/agents/`. Axiom escribe los agentes de fase (`sdd-init` a `sdd-onboard` y los agentes de Judgment Day) y traduce el campo `model:` de las asignaciones de Kiro (`auto|opus|sonnet|haiku|minimax|glm|deepseek|qwen`) a los identificadores nativos de Kiro.

- Frontmatter includes `includeMcpJson: true` for all phase agents
- Phase-specific tools are preserved (`sdd-explore` and `sdd-verify` use read/shell/context7 as required)
- El orquestador permanece en el fichero de steering `~/.kiro/steering/axiom.md`; su contenido delega en los agentes nativos.

---

## SDD Mode Support

| Feature          | Claude Code | OpenCode | Kilo Code | Gemini CLI | Cursor | VS Code Copilot | Codex | Windsurf | Antigravity | Kiro IDE | Qwen Code | OpenClaw | Trae |   Pi    | Hermes |
| ---------------- | :---------: | :------: | :-------: | :--------: | :----: | :-------------: | :---: | :------: | :---------: | :------: | :-------: | :------: | :--: | :-----: | :----: |
| SDD orchestrator |     Yes     |   Yes    |    Yes    |    Yes     |  Yes   |       Yes       |  Yes  |   Yes    |     Yes     |   Yes    |    Yes    |   Yes    | Yes  |   Yes   |  Yes   |
| Single-mode SDD  |     Yes     |   Yes    |    Yes    |    Yes     |  Yes   |       Yes       |  Yes  |   Yes    |     Yes     |   Yes    |    Yes    |   Yes    | Yes  |   Yes   |  Yes   |
| Multi-mode SDD   |      —      |   Yes    |    Yes    |     —      |   —    |        —        |   —   |    —     |      —      |  Yes\*   |     —     |    —     |  —   | Yes\*\* |   —    |

**Multi-mode** (assigning different AI models to each SDD phase) is supported by **OpenCode** and **Kilo Code** through the OpenCode-compatible multi-mode overlay, and by **Kiro IDE** through native subagent `model:` frontmatter. All other agents run in **single-mode** — the orchestrator manages everything using whatever model the agent is already running.

> \* **Kiro multi-mode** assigns models per phase through `KiroModelAssignments` (configured via _Configure Models → Configure Kiro models_ in the TUI). The selected Kiro alias (`auto|opus|sonnet|haiku|minimax|glm|deepseek|qwen`) is resolved to a Kiro-native model ID and stamped into each `~/.kiro/agents/sdd-{phase}.md` at sync time.

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
- La ejecución en segundo plano se configura con `axiom install` / `axiom sync`, usando `--opencode-background-subagents=auto|on|off` o la variable heredada `GENTLE_AI_OPENCODE_BACKGROUND_SUBAGENTS`.
- CLI precedence is flag, non-empty environment, prior managed state, then `auto`; the interactive OpenCode + SDD installer prompts only when that preference is unresolved
- Los launchers gestionados se encuentran bajo `~/.axiom/bin/` (o `<AXIOM_STATE_DIR>/bin/` si defines esa variable; la ruta heredada `~/.gentle-ai/bin/` solo se limpia) y respetan `OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=false`; reinicia OpenCode después de activarlos.
- `serve`, `attach`, Desktop, and sessions not launched through the managed launcher use the safe foreground fallback
- Background jobs are process-local and non-durable, have no filesystem isolation, and must not be used for dependent phases or parallel writers in one worktree
- The TUI model picker asynchronously discovers the active project's effective providers and models through `opencode models --verbose`, including custom, authenticated, plugin, and dynamic providers
- Only models OpenCode reports with tool-call capability appear as selectable SDD-capable options
- Antes de seleccionar modelos, conecta tus proveedores de IA y vuelve al selector; Axiom no actualiza el catálogo de OpenCode.
- Axiom establece `share: disabled` como valor predeterminado de los agentes SDD de OpenCode por privacidad y conserva los valores gestionados por el usuario, como `manual` o `auto`.
- OpenCode Desktop SDD commands resolve the project with `git rev-parse --show-toplevel || pwd` before acting, avoiding Electron current-working-directory drift.
- Review launch runs from an ordinary already-running OpenCode session: no restart, child process, special user-visible session, or `OPENCODE_DISABLE_PROJECT_CONFIG` / `OPENCODE_DISABLE_EXTERNAL_SKILLS` variable is required (rdd-advisory-transport SKILL.md).
- Con `--scope workspace`, Axiom escribe en la distribución de proyecto que OpenCode lee, no en `<proyecto>/.config/opencode` (que ningún agente lee): la configuración (overlay SDD, `mcp`, `permission`, `instructions`) va en `<proyecto>/opencode.json` —o en `opencode.jsonc` si ya existe, con sus comentarios—; comandos y plugins, en `<proyecto>/.opencode/{commands,plugins}` (las skills, en la carpeta compartida `.agents/skills`); y los prompts compartidos del modo multi, en `.opencode/prompts/sdd`. Las instrucciones (persona, protocolo de memoria) van en un fichero propio, `.opencode/axiom.md`, que se añade a la lista `instructions` de `opencode.json` sin duplicados y conservando las entradas del usuario; el `AGENTS.md` de la raíz es del equipo y Axiom no lo toca. Un `opencode.json` que no sea JSON válido se rechaza sin modificarlo. Los restos de instalaciones anteriores en `<proyecto>/.config/opencode` no se borran (pueden mezclar ficheros del usuario): Axiom avisa para que se revisen. La instalación global no cambia.

### Kilo Code

- **Detección**: Axiom detecta Kilo Code en `~/.config/kilo` y comprueba si el ejecutable `kilo` está en `PATH`.
- Uses the OpenCode-compatible adapter: `AGENTS.md`, `skills/`, `commands/`, and `opencode.json` live under `~/.config/kilo`
- Full SDD delegation is provided by the merged multi-agent overlay in `~/.config/kilo/opencode.json`, not by a separate native sub-agent directory
- Kilo no expone la vía de interceptación de herramientas ejecutables gestionada por Axiom; su preflight sigue siendo un bloqueo a nivel del prompt y no declara aplicar autoridad en tiempo de ejecución.
- MCP servers are merged into `opencode.json`; Engram uses the OpenCode-style local MCP entry with `command` as an array
- Auto-install is supported via npm: `npm install -g @kilocode/cli`
- Con `--scope workspace`, Axiom escribe en la distribución de proyecto de Kilo, no en `<proyecto>/.config/kilo`: la configuración en `<proyecto>/.kilo/kilo.json` (o en el `kilo.jsonc`/`kilo.json` que ya exista, respetando la precedencia de Kilo: así no se crea un segundo fichero junto al del usuario), comandos y plugins en `<proyecto>/.kilo/{commands,plugins}` (las skills, en la carpeta compartida `.agents/skills`), y las instrucciones en `.kilo/rules/axiom.md`, registrado en `instructions` de la configuración del proyecto con la ruta relativa a la raíz que documenta Kilo (`.kilo/rules/axiom.md`). El `AGENTS.md` de la raíz no se toca. La instalación global no cambia.

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
  - **MCP** (Context7 y Engram): `<proyecto>/.vscode/mcp.json`, con la clave raíz `servers` (no `mcpServers`). La fusión conserva los servidores, las `inputs` y los comentarios que el usuario ya tuviera (VS Code admite JSONC en ese fichero; los comentarios que estén dentro del bloque `servers` se pierden al reescribirlo). Un `mcp.json` que no sea JSON válido se rechaza sin modificarlo y antes de escribir nada.
  - **Agentes, prompts y hooks**: Axiom no instala ninguno para VS Code Copilot hoy (solo skills, instrucciones y MCP), así que no se crean `.github/agents`, `.github/prompts` ni `.github/hooks`. Tampoco se escriben permisos: VS Code no tiene un `deny` de proyecto y Axiom no impone modos de autoaprobación.
  - **Confianza**: VS Code solo arranca los servidores MCP de `.vscode/mcp.json` si el espacio de trabajo es de confianza (Workspace Trust). Axiom avisa al terminar y no concede esa confianza.
  - GitHub Copilot CLI, que solo tiene ámbito global en Axiom (véase [su sección](#github-copilot-cli)), lee también `.github/instructions` y `.github/skills`. La carpeta `.agents/skills` es la compartida que escribe Axiom en workspace.
  - La instalación global no cambia.

### GitHub Copilot CLI

Agente de terminal de GitHub (binario `copilot`). Es un adaptador propio (`github-copilot-cli`), distinto del de VS Code Copilot: solo comparten la carpeta de skills.

- **Raíz**: `$COPILOT_HOME` si está definida y es una ruta absoluta, o `~/.copilot`. `COPILOT_HOME` solo se respeta con el directorio personal real del usuario: una instalación aislada (otra raíz de instalación) se queda dentro de ella.
- **Instrucciones**: `<raíz>/copilot-instructions.md` (persona, orquestador ODD/SDD y protocolo de memoria). El SDD es el mismo orquestador genérico de VS Code Copilot, con la elección de almacén de artefactos en la preflight de sesión.
- **Skills**: `<raíz>/skills/`. Sin `COPILOT_HOME` es la misma carpeta que usa VS Code Copilot. La copia no depende de qué agente la escribe (lleva el hueco genérico de runtime), y desinstalar uno de los dos no retira las skills mientras el otro siga instalado. Quien ya tenía VS Code Copilot verá que el primer sync tras instalar Copilot CLI reescribe `_shared/review-ledger-contract.md`: la identidad del agente pasa a `<runtime>`. Es un cambio de texto sin efecto, porque ninguno de los dos tiene revisión nativa.
- **MCP** (Context7 y Engram): `<raíz>/mcp-config.json`, con la clave raíz `mcpServers` (no `servers`, la de VS Code) y entradas con `type` (`http` para Context7, `local` para Engram) y `tools: ["*"]`. Se conservan los servidores del usuario. Engram se inyecta directamente, sin `engram setup`. CodeGraph tiene un destino nativo `copilot-cli` que escribe en el mismo fichero.
- **Ajustes y permisos**: `<raíz>/settings.json`. Axiom no escribe permisos ni modos de autoaprobación: lo que decidas queda intacto.
- **Hooks, comandos y subagentes de fichero**: ninguno.
- **Detección**: el binario `copilot` en el `PATH` o una entrada propia de Copilot CLI en la raíz:
  - su estado: `config.json`, `session-state/`, `session-store.db`, `logs/`, etc.;
  - o lo que solo él lee: `settings.json`, `mcp-config.json` y `copilot-instructions.md`.

  `skills/` y `agents/` las lee también VS Code Copilot, y una entrada desconocida no cuenta, así que una `~/.copilot` sin ninguna de esas entradas no se toma por Copilot CLI; y VS Code Copilot se detecta en el escaneo por su extensión `github.copilot`, no por `~/.copilot`.
- **Instalación** (solo se muestra, Axiom no la ejecuta): `npm install -g --ignore-scripts @github/copilot@latest`, con `sudo` en Linux si el npm es del sistema. `axiom doctor` comprueba el binario `copilot`.
- **Ámbito workspace**: no tiene. Con `--scope workspace` se omite con un aviso.

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
  - **Skills**: `<proyecto>/.agents/skills`, la única carpeta de skills de proyecto que documenta Codex, y las rutas del orquestador que antes apuntaban a `~/.codex/skills` se resuelven a `.agents/skills`. Esa carpeta la leen también otros agentes (Gemini CLI, OpenCode, Kilo, Cursor, VS Code Copilot), y Axiom escribe una sola copia para todos: ya no instala las skills además en la carpeta propia de cada uno. Un `.codex/skills` de una instalación anterior se migra como se explica en «Skills en `--scope workspace`».
  - **Sin escritura en HOME**: Axiom no escribe la confianza del proyecto en `~/.codex/config.toml`. Al terminar avisa de que Codex pedirá confiar en el proyecto la primera vez y de que cada hook se aprueba por hash. Tampoco se escriben los ficheros de perfil `<carril>.config.toml` (Codex ignora `profiles` en proyecto), ni se valida el ejecutable de Codex, que solo servía para esos perfiles. Los permisos no se escriben en ningún ámbito: Codex usa los suyos.
  - **`engram setup codex`**: con `--scope workspace` Axiom no lo ejecuta, porque escribe en `~/.codex`, y lo avisa como acción manual. El MCP de Engram ya queda declarado en `.codex/config.toml`, así que el comando solo hace falta si además quieres el registro de usuario; ejecutarlo escribe fuera del proyecto. Con `--scope global` se ejecuta como siempre.
  - La instalación global no cambia.

### Windsurf

- Skills at `~/.codeium/windsurf/skills/` (native Windsurf feature)
- MCP config at `~/.codeium/windsurf/mcp_config.json`
- Global rules at `~/.codeium/windsurf/memories/global_rules.md`
- Workflows at `.windsurf/workflows/` (workspace-scoped)

### Antigravity

- Skills at `~/.gemini/antigravity/skills/` (native Antigravity feature)
- MCP config at `~/.gemini/antigravity/mcp_config.json`
- System prompt appended to `~/.gemini/GEMINI.md` (shared with Gemini CLI — collision check warns if both are installed)
- Mission Control handles built-in sub-agent delegation (Browser, Terminal) automatically
- Settings managed via the IDE's Agent settings UI, not via `settings.json`

### Kimi Code (no disponible de momento)

- Installation requires the `uv` Python package manager (`uv tool install kimi-cli`).
- Root custom agent at `~/.kimi/agents/gentleman.yaml` with `system_prompt_path: ../KIMI.md`
- `KIMI.md` is a thin Jinja template that includes modular prompt files:
  `persona.md`, `output-style.md`, `engram-protocol.md`, `sdd-orchestrator.md`
- Built-in Kimi variables are preserved in `KIMI.md`: `${KIMI_AGENTS_MD}` and `${KIMI_SKILLS}`

### Kiro IDE

- **Detección**: Axiom detecta Kiro mediante el ejecutable `kiro` en `PATH`; también informa de si existe `~/.kiro`. Tener solo el directorio de configuración no basta para marcar Kiro como instalado.
- **Steering file** (all platforms): `~/.kiro/steering/axiom.md` with frontmatter `inclusion: always`. `axiom sync` retira el antiguo `gentle-ai.md` solo si lo escribió Axiom; si tiene contenido propio lo conserva y avisa
- Native subagents at `~/.kiro/agents/sdd-{phase}.md` (11 phase files, plus the Judgment Day and review agents)
- Skills (all platforms) at `~/.kiro/skills/`
- **MCP config at a separate root** — always `~/.kiro/settings/mcp.json` (macOS/Linux) or `%USERPROFILE%\.kiro\settings\mcp.json` (Windows), regardless of GlobalConfigDir
- Native Kiro specs workflow: `.kiro/specs/<feature>/requirements.md`, `design.md`, `tasks.md` — with approval gates before apply and archive phases
- Manual install only — download from [kiro.dev/downloads](https://kiro.dev/downloads)
- See [docs/kiro.md](kiro.md) for full path reference and SDD behavior details

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

### OpenClaw

- **Detección**: Axiom detecta OpenClaw mediante el ejecutable `openclaw` en `PATH` y la raíz `~/.openclaw`.
- **Ámbito**: solo con `--scope global`. Con `--scope workspace` se omite con un aviso y no se escribe nada (ni `~/.openclaw/openclaw.json` ni `SOUL.md` en el proyecto).
- **Instalación**: manual; instala primero OpenClaw y después ejecuta `axiom install --agent openclaw`.
- **Workspace activo**: Axiom lee `agents.defaults.workspace` de `~/.openclaw/openclaw.json` y escribe allí los ficheros de instrucciones.
- **Instructions**: Engram and SDD protocols are injected into workspace `AGENTS.md`; persona is injected into workspace `SOUL.md`.
- **MCP config**: Engram and Context7 are merged into global `~/.openclaw/openclaw.json` under `mcp.servers`; legacy root `mcpServers` entries are migrated.
- **Skills**: selected portable skills and SDD phase skills are workspace-scoped at `<workspace>/.openclaw/skills/`.

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
- **Cambio de persona**: `gentle-pi` gestiona `/gentle:persona`, que alterna entre `gentleman` y `neutral`, guarda `.pi/gentle-ai/persona.json` y puede requerir `/reload` o una nueva sesión para actualizar el prompt activo.
- **Asignación de modelos**: `gentle-pi` gestiona `/gentle:models`. Abre un diálogo nativo de Pi para agentes de proyecto, usuario e integrados, prioriza los agentes SDD, guarda `.pi/gentle-ai/models.json` y aplica los cambios en `.pi/agents/*.md` o `.pi/settings.json`.
- **`gentle-engram` package**: adds persistent Engram memory for Pi. It captures sessions, exposes Engram MCP tools through `pi-mcp-adapter`, and degrades safely when the local `engram` binary is missing.
- **MCP adapter wiring**: ComponentEngram declares `npm:pi-mcp-adapter` in `.pi/agent/settings.json` packages and adds `pi-mcp-adapter` `^2.6.0` to `.pi/npm/package.json` without removing unrelated user entries. `pi-engram init` owns the Pi Engram MCP config schema and is run during installation.
- **Subagentes**: `gentle-pi` proporciona Gentle Agents para descubrir y ejecutar agentes SDD desde `.pi/agents/`, con las herramientas `subagent_*` que antes ofrecía `npm:pi-subagents-j0k3r`. Axiom no instala ese paquete por separado; si ya existe una entrada `npm:pi-subagents-j0k3r`, se elimina de `settings.json` durante la siguiente instalación o actualización y Pi lo desinstala al sincronizar sus paquetes.
- **Subagentes en segundo plano**: se configuran con `axiom install` / `axiom sync`, mediante `--pi-background-subagents=auto|on|off` o la variable heredada `GENTLE_AI_PI_BACKGROUND_SUBAGENTS`. No hay launcher ni activación adicional, porque la primitiva es la extensión `pi-subagents-j0k3r` ya instalada.
- CLI precedence is flag, non-empty environment, prior managed state, then `auto`; `auto` never enables by itself, unresolved non-interactive `auto` stays foreground, and the interactive Pi installer prompts only when that preference is unresolved.
- The resolved on/off policy is projected to `~/.pi/gentle-ai/background-subagents.json` as `{"schema":"gentle-pi.background-subagents/v1","policy":"on"|"off"}` (the base directory honors `GENTLE_PI_CONFIG_HOME`); `off` rewrites the policy instead of deleting files, and a file at that path without the managed schema marker is never overwritten.
- **`@juicesharp/rpiv-ask-user-question` package**: lets Pi child agents ask the active user session for clarification when they need human input.
- **Pi companion packages**: `pi-web-access` and `pi-btw` add web access and companion workflow support. Todo tracking ships inside `gentle-pi` (Gentle Todo); an existing `@juicesharp/rpiv-todo` entry is dropped from `settings.json` on the next install or update, and Pi uninstalls it on its next package sync.
- **Solo Pi**: cuando Pi es el único agente seleccionado, Axiom omite las preguntas de persona, componentes del ecosistema y TDD estricto, porque las gestiona `gentle-pi`.

### Hermes Ephemeral Delegation

Hermes uses `delegate_task` to spawn ephemeral sub-agents. Each worker starts in a fresh context window and returns only its final summary to the parent orchestrator.

**Delegation rules:**

- Delegate by what the evidence costs, never by how many files it touches: inline evidence is one parallel batch of at most 3 calls and about 10k tokens; when it does not fit, one read-only explorer returns a handoff of at most about 2k tokens with `path:line` evidence. A writer is delegated only for a named reason (independent units in parallel, or protecting the parent context), and a unit of about 100 changed lines or less always runs inline. Test/build execution and fresh adversarial review may use fresh workers per action. Each trigger holds only when its condition truly holds; when in doubt, work inline.
- Each worker mission must be self-contained: include the exact goal, file paths or targets, relevant prior context, constraints, expected evidence, and the toolsets/MCP/skills the worker is allowed to use.
- Toolsets, MCP servers, and skills are NOT automatically inherited from the parent; pass them explicitly in the mission when `inherit_mcp_toolsets` is false (the default).
- Prefer parallel workers only for truly independent workstreams. Dependencies must run sequentially.
- Treat worker output as self-report: verify file writes, test pass/fail, URLs, and external side effects before reporting success.

**Tuning knobs** (configure in `~/.hermes/config.yaml` under the `delegation` key):

| Parameter | Default | Effect |
|-----------|---------|--------|
| `max_spawn_depth` | 2 | Maximum recursive delegation depth |
| `max_concurrent_children` | 4 | Maximum parallel workers |
| `max_iterations` | agent default | Iteration budget per worker |
| `child_timeout_seconds` | agent default | Hard timeout per worker |
| `inherit_mcp_toolsets` | false | When true, workers inherit parent MCP toolsets automatically |
| `subagent_auto_approve` | false | When true, workers auto-approve tool calls |

La tabla completa de decisión de delegación está en `~/.hermes/skills/hermes-ephemeral-delegation/SKILL.md` (instalada por Axiom). El orquestador SDD de `~/.hermes/SOUL.md` remite a esa skill.

### Hermes (no disponible de momento)

- **Detección**: Axiom informa por separado de la presencia del ejecutable `hermes` en `PATH` y de la raíz `~/.hermes`; el directorio de configuración determina si Hermes figura como configurado, aunque falte el ejecutable.
- **Instalación**: solo detección; Axiom no instala Hermes. Instálalo manualmente y después ejecuta `axiom install --agent hermes`.
- **Config path**: `~/.hermes/` (config.yaml, SOUL.md, skills/)
- **MCP config**: Engram and Context7 are injected as YAML blocks under `mcp_servers:` in `~/.hermes/config.yaml` (`StrategyMergeIntoYAML`). Pre-existing top-level keys (e.g. `model:`) are preserved verbatim.
- **Prompt de sistema**: Axiom escribe el orquestador SDD y la persona en `~/.hermes/SOUL.md` mediante los marcadores Markdown heredados `<!-- gentle-ai:sdd-orchestrator -->` y `<!-- gentle-ai:persona -->`.
- **Skills**: Axiom escribe las skills de fases SDD en `~/.hermes/skills/`; el registro de skills también consulta esa ruta.
- **Permisos**: Hermes utiliza un formato de permisos no documentado. Axiom no inyecta permisos en Hermes.
- **Profiles**: Hermes does not support multi-mode SDD (no per-phase model routing). Single-mode only.
- **Memory**: Hermes has a native memory and skill-learning loop. Engram complements it — Engram provides cross-agent, cross-session memory protocol so knowledge is portable across all agents, not just Hermes.
- **Marcadores de persona e identidad**: Los marcadores `<!-- axiom:persona -->` / `<!-- /axiom:persona -->` en `SOUL.md` delimitan la sección que Axiom gestiona y actualiza durante la sincronización; los marcadores heredados `<!-- gentle-ai:persona -->` se siguen reconociendo por compatibilidad. Las tres personas de Hermes (`axiom`, `neutral` y `gentleman`) incluyen literalmente `## Identity` con la instrucción de identificarse como **Axiom running on Hermes Agent** en cualquier idioma. Cualquier sección `## Identity` escrita fuera de los marcadores se conserva y puede contradecir la sección gestionada.
