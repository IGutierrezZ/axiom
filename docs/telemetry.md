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

`axiom sync` and `axiom install` remove the Axiom telemetry hooks and the
Axiom-owned OpenCode `telemetry-runtime.ts` plugin left by earlier versions. A
copy you edited yourself is kept, and Axiom prints a warning so you can delete
it by hand.

## `~/.gentle-ai/telemetry.json`

If this file exists, it is shared with the upstream Gentle AI tool. Axiom
neither reads nor deletes it. Remove it yourself only if you no longer use the
upstream tool.
