# Notas de migración: retirada de GGA y de las lápidas de telemetría

Esta versión retira GGA (Gentleman Guardian Angel, el revisor IA previo al commit heredado de gentle-ai) y las últimas lápidas de la telemetría, que ya no enviaba nada desde su retirada.

> **Esta versión no limpia lo que instalaron versiones anteriores.** `axiom install`, `axiom sync`, `axiom uninstall`, `axiom doctor` y `axiom upgrade` ya no conocen GGA ni la telemetría. Si tienes restos, haz la limpieza de abajo **antes** de instalarla (o, con la versión anterior, ejecuta `axiom uninstall --components gga` antes de actualizar).

## Qué limpiar a mano

Borra solo lo que escribió Axiom; si editaste a mano alguno de estos ficheros, conserva lo tuyo.

1. **Hooks de telemetría.** En `~/.claude/settings.json` y en `~/.codex/hooks.json`, las entradas de `Stop` y `SubagentStop` cuya orden es exactamente `axiom telemetry runtime claude --json`, `axiom telemetry runtime codex --json` o sus equivalentes con `gentle-ai`. Si un evento se queda sin entradas, bórralo.
2. **Plugin de telemetría de OpenCode.** `~/.config/opencode/plugins/telemetry-runtime.ts` y `~/.config/opencode/.gentle-ai-telemetry-runtime.json` (o los de la carpeta que marque `XDG_CONFIG_HOME`).
