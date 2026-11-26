# Notas de migración: arranque mínimo (sin TUI, persona heredada, presets, temas, logo ni agent builder)

Esta versión deja el arranque de Axiom en lo esencial: `axiom` sin argumentos abre la UI (igual que `axiom ui`), y la instalación es detectar agentes, elegir componentes e instalar. Se retiran la TUI, las personas `gentleman` y `neutral` (solo queda la de Axiom), los presets, los temas, el logo de OpenCode y el agent builder, todos heredados de gentle-ai.

> **Esta versión no limpia lo que instalaron versiones anteriores.** `axiom install`, `axiom sync`, `axiom uninstall` y `axiom upgrade` ya no conocen nada de lo retirado. Si tienes restos, haz la limpieza de abajo **antes** de instalarla.

## Qué limpiar a mano

Borra solo lo que escribió Axiom; si editaste a mano alguno de estos ficheros, conserva lo tuyo. Las rutas `~/.claude` y `~/.config/opencode` valen también para su copia dentro de un proyecto (ámbito workspace).

1. **Personas `gentleman` y `neutral`** (solo si alguna vez las elegiste):
   - los output styles `~/.claude/output-styles/gentleman.md` y `~/.claude/output-styles/neutral.md`;
   - en `~/.claude/settings.json`, la clave `"outputStyle"` si vale `"Gentleman"` o `"Neutral"`. Con el componente `persona` seleccionado, `axiom sync` la deja en `"Axiom"`; sin él, bórrala;
   - en Pi, `.pi/gentle-ai/persona.json` si su `mode` es `gentleman` o `neutral`. Con el componente `persona`, `axiom sync` lo reescribe con `axiom`.
2. **Temas.** `~/.claude/themes/` y `~/.config/opencode/themes/`: los ficheros `gentleman.json`, `gentleman-cute.json`, `axiom.json` y `axiom-dark.json`. Axiom nunca cambió la clave `theme` de tus ajustes; si apunta a uno de esos temas, elige otro.
3. **Logo de OpenCode.** `~/.config/opencode/tui-plugins/gentle-logo.tsx` y su entrada en el array de plugins de `~/.config/opencode/tui.json`.
4. **Agent builder.** El registro `~/.config/gentle-ai/custom-agents.json` y, si ya no los quieres, los agentes que creaste con él: cada uno es una carpeta de skill con su nombre en `~/.claude/skills/`, `~/.config/opencode/skills/`, `~/.gemini/skills/` o `~/.codex/skills/`. Si lo integraste con SDD, quita también la referencia que añadió a `CLAUDE.md`, `AGENTS.md` o `GEMINI.md`.

## Qué pasa con una instalación anterior

- Un estado (`~/.axiom/state.json` o el de un proyecto) con `persona` o `preset` guardados sigue cargando: `axiom sync` avisa de que los ignora cuando la persona no es `axiom` o hay preset. Si hay componentes `theme`, `claude-theme` u `opencode-gentle-logo`, también avisa de que los ignora y sincroniza el resto. La siguiente escritura del estado ya no los guarda.
- `axiom` sin argumentos abre la UI; sin terminal interactivo, devuelve el mismo error que antes. `axiom tui` ya no existe.
- `axiom install` ya no admite `--persona` ni `--preset`: sin `--component`, instala el conjunto por defecto (Engram, SDD, skills, Context7, permisos y persona), y con `--component` eliges cada componente. `axiom sync` ya no admite `--include-theme`.
- Un estado con `preset: custom` y sin skills elegidas sincroniza ahora las skills por defecto (antes, ninguna). Si no las quieres, desmarca el componente `skills` o elige las skills con `--skill`.
- Con el componente `persona`, `axiom sync` sigue quitando de `opencode.json` el agente `gentleman` que creaba la persona heredada, como ya hacía con la persona de Axiom.

## Acciones de quien publica

- Pegar este texto en las notas de la primera versión que incluya el cambio.
