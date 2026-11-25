# Sync and Cloud

[Back to Codebase Guide](../CODEBASE-GUIDE.md)

Axiom sync refreshes managed agent configuration. Engram memory is personal and is not shared through git. Cloud sync is not implemented in this source tree.

## Sync boundaries

| Flow | Command surface | Owner | What changes |
|---|---|---|---|
| Axiom config sync | `axiom sync` | `internal/cli/sync.go`, components, adapters | Agent prompts, skills, MCP configs, SDD profiles, persona assets, and configured community tool guidance. |
| Engram memory sharing | Not used | External Engram runtime | Nothing in the repository: memory stays in each person's local store; only `.engram/config.json` is versioned. |
| Cloud sync | Not present in Axiom source | External or future Engram capability | Do not document implementation here without source. |
| Autosync | Not present in Axiom source | External or future Engram capability | Do not imply background sync exists in this repo. |

## Axiom sync path

```text
axiom sync
  -> parse sync flags
  -> discover installed agents from the scope state (<workspace root>/.axiom/state.json or ~/.axiom/state.json) or explicit flags
  -> build managed selection
  -> run component injectors
  -> verify readiness
  -> report files changed or no-op
```

Important behavior from `internal/cli/sync.go`:

- Default sync scope includes SDD, Engram, Context7, skills, and persona.
- Persona sync resolves the persisted persona from the scope state (`<workspace root>/.axiom/state.json` in workspace scope, `~/.axiom/state.json` in global scope) when the selection does not set one explicitly; the safe fallback is neutral.
- Permissions and theme are user-adjacent and not included by default.
- OpenCode SDD profile flags preserve and update profile model assignments.
- Community tool guidance/config, such as CodeGraph guidance, belongs to the managed config sync path when the tool is configured or legacy guidance needs cleanup.
- Idempotency matters: `FilesChanged == 0` means managed assets were already current.

## Memory is not shared through git

Engram memory is personal (see [Engram Commands](../engram.md)). The team shares each change's documentation in OpenSpec or the ODD feature document, not memory exports. `.engram/chunks/` and `.engram/manifest.json` are ignored; `axiom sync` refreshes agent configuration and never touches memory.

## Remote transport boundary

No remote transport implementation is present in this repository beyond update/download logic for external binaries and releases. Do not describe an Engram cloud transport, cloud server, or cloud store split as Axiom code unless that code is added here.

## Cloud server/cloud store split

This repository does not contain cloud server or cloud store packages. If future Engram cloud docs are added, document them as an external Engram responsibility and keep this page focused on how Axiom discovers, installs, or configures that capability.

## Contributor checklist

- [ ] Use `axiom sync` for managed config, not memory export/import.
- [ ] Do not commit Engram memory exports (`.engram/chunks/`, `.engram/manifest.json`).
- [ ] Keep sync changes idempotent and test `FilesChanged` expectations.
- [ ] Check persona behavior against persisted state and neutral fallback rules.
- [ ] Keep community tool guidance separate from OpenCode plugin registration.
- [ ] Do not touch untracked local `.engram/cloud.json` or `.engram/engram.db` during docs or sync work.

## Navigation

Previous: [Interfaces](interfaces.md) | Next: [Dashboard](dashboard.md)
