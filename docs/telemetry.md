# Telemetry

Axiom sends no telemetry. It does not collect usage data and does not contact
any telemetry server, including the upstream Gentle AI one.

## What was removed

- the telemetry client and its automatic sends;
- the `axiom telemetry` command;
- the telemetry hooks installed for the agents (Claude Code, Codex);
- the OpenCode `telemetry-runtime.ts` plugin;
- the collector and its deployment kit.

## Cleanup of earlier installs

Axiom no longer removes what earlier versions installed. Delete the leftover
telemetry hooks and the OpenCode `telemetry-runtime.ts` plugin by hand: the
list is in [`releases/notas-retirada-gga-telemetria.md`](releases/notas-retirada-gga-telemetria.md).

## `~/.gentle-ai/telemetry.json`

If this file exists, it is shared with the upstream Gentle AI tool. Axiom
neither reads nor deletes it. Remove it yourself only if you no longer use the
upstream tool.
