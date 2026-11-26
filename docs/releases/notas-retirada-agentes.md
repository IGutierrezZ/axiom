# Notas de migración: retirada de seis agentes

Esta versión retira seis agentes: Kiro IDE (`kiro-ide`), Windsurf (`windsurf`), Kimi Code (`kimi`), Hermes (`hermes`), OpenClaw (`openclaw`) y Kilo Code (`kilocode`). Se quedan Claude Code, VS Code Copilot, OpenCode, GitHub Copilot CLI, Gemini CLI, Codex, Cursor, Antigravity, Pi, Trae IDE y Qwen Code (este sigue sin poder elegirse al instalar).

> **Esta versión no limpia lo que instalaron versiones anteriores para esos agentes.** `axiom install`, `axiom sync`, `axiom uninstall` y `axiom doctor` ya no conocen sus rutas. Si usabas alguno, haz la limpieza de abajo **antes** de instalarla (o con la versión anterior, ejecuta `axiom uninstall` de esos agentes antes de actualizar).

## Qué pasa con una instalación anterior

- Un estado (`~/.axiom/state.json` o el de un proyecto) que incluya uno de esos agentes sigue cargando: `axiom sync` avisa de que lo ignora y sincroniza el resto. La siguiente escritura del estado ya no lo guarda. Las asignaciones de modelos de Kiro (`kiro_model_assignments`) se ignoran y desaparecen igual.
- El menú «Configurar modelos» ofrece Claude, OpenCode y Codex.
- `--agent` con uno de esos identificadores falla como cualquier agente desconocido.

## Qué limpiar a mano

Borra solo lo que escribió Axiom; si editaste a mano alguno de estos ficheros, conserva lo tuyo.

1. **Kilo Code.** En `~/.config/kilo/opencode.json`, las entradas de `agent` de Axiom (`axiom-orchestrator`, `sdd-*`, `jd-*` y los perfiles `sdd-orchestrator-<perfil>`), `mcp.engram`, `mcp.context7` y `share`; en `~/.config/kilo/AGENTS.md`, las secciones `<!-- axiom:... -->`; los plugins `background-agents.ts`, `model-variants.ts` y `skill-registry.ts` de `~/.config/kilo/plugins/`; y las skills y comandos de Axiom de `~/.config/kilo/skills/` y `~/.config/kilo/commands/`. En los proyectos instalados con `--scope workspace`: `.kilo/kilo.json` (las mismas entradas y la de `instructions` que apunta a `.kilo/rules/axiom.md`), `.kilo/rules/axiom.md`, `.kilo/prompts/sdd/` y los comandos y plugins de Axiom de `.kilo/`.
2. **OpenClaw.** En `~/.openclaw/openclaw.json`, `mcp.servers.engram` y `mcp.servers.context7`; las skills de Axiom de `~/.openclaw/skills/`; y, en la carpeta de trabajo de OpenClaw, las secciones de Axiom de `AGENTS.md` y `SOUL.md`.
3. **Hermes.** En `~/.hermes/config.yaml`, los bloques `engram` y `context7` de `mcp_servers`; las secciones de Axiom de `~/.hermes/SOUL.md`; y las skills de Axiom de `~/.hermes/skills/` (incluida `hermes-ephemeral-delegation`).
4. **Kimi Code.** En `~/.kimi/`: `KIMI.md` (si no lo has editado), los módulos `persona.md`, `output-style.md`, `sdd-orchestrator.md`, `strict-tdd-mode.md`, `engram-protocol.md` y `agent-routing.md`, la carpeta `agents/` de Axiom (`gentleman.yaml`, `sdd-*.yaml` y `sdd-*.md`) y `mcpServers.engram` y `mcpServers.context7` de `mcp.json`. No borres `~/.config/agents/skills/`: la comparten otros agentes.
5. **Windsurf.** Las secciones de Axiom de `~/.codeium/windsurf/memories/global_rules.md`; `mcpServers.engram` y `mcpServers.context7` de `~/.codeium/windsurf/mcp_config.json`; las skills de Axiom de `~/.codeium/windsurf/skills/`; y, en cada proyecto, los workflows `sdd-*.md` de `.windsurf/workflows/`.
6. **Kiro IDE.** `~/.kiro/steering/axiom.md` (y el heredado `gentle-ai.md`); los agentes de Axiom de `~/.kiro/agents/` (`sdd-*.md`, `jd-*.md`); `engram`, `context7` y `codegraph` de `mcpServers` en `~/.kiro/settings/mcp.json`; las skills de Axiom de `~/.kiro/skills/`; y el hook `~/.kiro/hooks/axiom-skill-registry.json`. En los proyectos instalados con `--scope workspace`, lo mismo bajo `.kiro/`.

## Acciones de quien publica

- Pegar este texto en las notas de la primera versión que incluya el cambio.
