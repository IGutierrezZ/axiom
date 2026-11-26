# Architecture & Development

← [Back to README](../README.md)

---

## Architecture

```
cmd/axiom/                 CLI entrypoint
internal/
  app/                     Command dispatch + runtime wiring
  model/                   Domain types (agents, components, skills, presets, personas)
  catalog/                 Registry definitions (agents, skills, components)
  system/                  OS/distro detection, dependency checks, platform guards
  cli/                     Install flags, validation, orchestration, dry-run
  planner/                 Dependency graph, resolution, ordering, review payloads
  installcmd/              Profile-aware command resolver (brew/apt/pacman/dnf/winget/go install)
  pipeline/                Staged execution + rollback orchestration
  backup/                  Config snapshot + restore
  assets/                  Embedded skill files + persona templates
  components/              Per-component install/inject logic
    engram/  sdd/  skills/  mcp/  persona/  theme/  permissions/
    communitytool/         Community tool install/guidance/config orchestration
    opencodeplugin/        OpenCode TUI plugin registration/local plugin helpers
    uninstall/             Managed uninstall cleanup service
    filemerge/             Marker-based file merging (inject without clobbering)
  skillregistry/           .atl skill registry refresh/list support
  agents/                  Agent adapters (config strategy per agent)
    claude/  opencode/  gemini/  cursor/  vscode/  codex/  antigravity/
  opencode/                OpenCode model/config parsing utilities
  state/                   Installation state tracking
  update/                  Self-update + upgrade logic
    upgrade/               Tool upgrade execution and reporting
  verify/                  Post-apply health checks + reporting
  tui/                     Bubbletea TUI (Rose Pine theme)
    styles/  screens/
scripts/                   Installer, release, and maintainer scripts (install.sh, release preflights, cross-lane battery)
e2e/                       Docker-based E2E tests (Ubuntu + Arch)
testdata/                  Golden test fixtures
```

---

## Testing

```bash
# Unit tests
go test ./...

# Dry-run smoke test (macOS/Linux)
axiom install --dry-run --agent claude-code --preset minimal

# Dry-run smoke test (Windows PowerShell)
axiom.exe install --dry-run --agent claude-code --preset minimal
```

Test coverage is broad and changes frequently. Keep this section qualitative unless counts are generated automatically:

- Unit tests cover agent adapters, components, system detection, app dispatch, update/upgrade behavior, and TUI flows.
- The Organic Runtime E2E suite (`e2e/organicruntime`) drives real Claude Code and OpenCode binaries through the installed routing guidance; see [Testing agents deterministically](./testing-agents-deterministically.md).
- Golden fixtures snapshot generated component output under `testdata/`.
- Full pipeline paths are tested: detection, planning, execution, backup, restore, and verification.
- Agent adapter tests include cross-platform path validation.

---

## Relationship to Gentleman.Dots

| | Gentleman.Dots | AI Gentle Stack |
|--|---------------|-----------------|
| **Purpose** | Dev environment (editors, shells, terminals) | AI development layer (agents, memory, skills) |
| **Installs** | Neovim, Fish/Zsh, Tmux/Zellij, Ghostty | Configures Claude Code, OpenCode, Gemini CLI, Cursor, VS Code Copilot, Codex, Antigravity |
| **Overlap** | None — complementary | None — different layer |

Install Gentleman.Dots first for your dev environment, then AI Gentle Stack for the AI layer on top.

---

## License

MIT
