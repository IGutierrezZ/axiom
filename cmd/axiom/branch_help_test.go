package main

import (
	"strings"
	"testing"
)

// REQ-25.1 y decisión T3: la ayuda global lista `axiom branch status` y solo los
// verbos de rama implementados; ni `push`, ni `pr`, ni el retirado `check`.
func TestBranchStatusIsListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()

	if !strings.Contains(help, "\n  branch status ") {
		t.Fatalf("la ayuda global debe listar `branch status`:\n%s", help)
	}
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "branch" {
			continue
		}
		if fields[1] != "status" {
			t.Errorf("la ayuda global anuncia `branch %s`, que aún no existe: %q", fields[1], line)
		}
	}
}
