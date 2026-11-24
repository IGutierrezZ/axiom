package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// REQ-25.1: la ayuda global lista los cinco verbos, `axiom branch ensure`, `status`, `scope`, `done` y
// `guard`, y ninguno más; ni `push`, ni `pr`, ni el retirado `check`.
func TestBranchVerbsAreListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()
	implemented := map[string]bool{"ensure": true, "status": true, "scope": true, "done": true, "guard": true}

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

// REQ-43.24: la ayuda de `axiom branch ensure` documenta `--role`.
func TestBranchEnsureHelpDocumentsTheRoleFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.RunBranch([]string{"ensure", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("ensure --help sale con %d", code)
	}
	help := stdout.String()
	for _, want := range []string{"[--role <rol>]", "  --role <rol>", "role.start", "role_not_in_roster", "kickoff_not_sealed", "role_not_applicable"} {
		if !strings.Contains(help, want) {
			t.Errorf("la ayuda de ensure debe contener %q", want)
		}
	}
}
