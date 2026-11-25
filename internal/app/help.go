package app

import (
	"fmt"
	"io"
)

func printHelp(w io.Writer, version string) {
	fmt.Fprintf(w, `axiom — Axiom: Autonomous Agent Engineering System (%s)

USAGE
  axiom                     Launch interactive TUI
  axiom <command> [flags]

COMMANDS
  install      Configure AI coding agents on this machine
  setup        Provision AI coding agents isolated to this workspace (--scope workspace)
  uninstall    Remove Axiom managed files (--scope workspace|global; --all, --purge-state)
  sync         Sync agent configs and skills to current version (--scope global|workspace)
  odd          Organic Driven Development task lifecycle (create, status, promote)
  sdd          Spec-Driven Development orchestration (kickoff, gate, status, continue, archive-compose)
  skill-registry refresh
               Refresh .atl/skill-registry.md with cache-hit fast path
  sdd-status [change]
               Print native SDD phase status for orchestrators
  sdd-continue [change]
               Print native SDD dispatcher routing output
  sdd-attempt grant --cwd <repo> --change <change>
               Record explicitly authorized edit roots for one change instance
  update       Check for available updates
  upgrade      Apply updates to managed tools
  restore      Restore a config backup
  doctor       Run ecosystem health diagnostics
  version      Print version

FLAGS
  --help, -h    Show global help

Run 'axiom help' for this message.
Documentation: https://github.com/IGutierrezZ/axiom
`, version)
}
