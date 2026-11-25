# Notas de migración: retirada de GGA y de las lápidas de telemetría

Esta versión retira GGA (Gentleman Guardian Angel, el revisor IA previo al commit heredado de gentle-ai) y las últimas lápidas de la telemetría, que ya no enviaba nada desde su retirada.

> **Esta versión no limpia lo que instalaron versiones anteriores.** `axiom install`, `axiom sync`, `axiom uninstall`, `axiom doctor` y `axiom upgrade` ya no conocen GGA ni la telemetría. Si tienes restos, haz la limpieza de abajo **antes** de instalarla (o, con la versión anterior, ejecuta `axiom uninstall --components gga` antes de actualizar).

## Qué limpiar a mano

Borra solo lo que escribió Axiom; si editaste a mano alguno de estos ficheros, conserva lo tuyo.

1. **Hooks de telemetría.** En `~/.claude/settings.json` y en `~/.codex/hooks.json`, las entradas de `Stop` y `SubagentStop` cuya orden es exactamente `axiom telemetry runtime claude --json`, `axiom telemetry runtime codex --json` o sus equivalentes con `gentle-ai`. Si un evento se queda sin entradas, bórralo.
2. **Plugin de telemetría de OpenCode.** `~/.config/opencode/plugins/telemetry-runtime.ts` y `~/.config/opencode/.gentle-ai-telemetry-runtime.json` (o los de la carpeta que marque `XDG_CONFIG_HOME`).
3. **GGA, si lo instalaste con Axiom** (`--component gga` o el preset personalizado). Lo que escribía Axiom:
   - la configuración `~/.config/gga/config` y la plantilla `~/.config/gga/AGENTS.md` (en Windows, `%APPDATA%\gga\`);
   - el script del modo PR `~/.local/share/gga/lib/pr_mode.sh`;
   - en Windows, los shims `~/bin/gga.ps1` y `~/bin/gga.cmd`, y la entrada `~/bin` del PATH de usuario si no la usas para nada más.

   El binario `gga` y su runtime los instala el propio GGA (Homebrew o su `install.sh`): desinstálalos con su herramienta si ya no lo usas. Desactiva también los hooks de `gga install` que hubieras puesto en tus repositorios.

## Qué pasa con una instalación anterior

- Un estado (`~/.axiom/state.json` o el de un proyecto) cuya selección incluya `gga` sigue cargando: `axiom sync` avisa de que ignora el componente y sincroniza el resto. La siguiente escritura del estado ya no lo guarda.
- `--component gga` falla como cualquier componente desconocido.
- `axiom doctor` deja de exigir el binario `gga` y de listar su huella; `axiom upgrade` deja de actualizarlo.

## Acciones de quien publica

- Pegar este texto en las notas de la primera versión que incluya el cambio.
