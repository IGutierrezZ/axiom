package main

import (
	"strings"
	"testing"
)

// REQ-25.1: la ayuda global lista los cuatro verbos, `axiom branch ensure`, `status`, `done` y
// `guard`, y ninguno más; ni `push`, ni `pr`, ni el retirado `check`.
func TestBranchVerbsAreListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()
	implemented := map[string]bool{"ensure": true, "status": true, "done": true, "guard": true}

	for verb := range implemented {
		if !strings.Contains(help, "\n  branch "+verb+" ") {
			t.Fatalf("la ayuda global debe listar `branch %s`:\n%s", verb, help)
		}
	}
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "branch" {
			continue
		}
		if !implemented[fields[1]] {
			t.Errorf("la ayuda global anuncia `branch %s`, que aún no existe: %q", fields[1], line)
		}
	}
}
