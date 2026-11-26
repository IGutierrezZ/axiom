package main

import (
	"strings"
	"testing"
)

// RF-HUB-13 y RF-BRA-01: la ayuda global lista los cuatro subcomandos de `axiom autofetch`, un verbo
// de primer nivel, y los cinco verbos de `axiom branch` siguen siendo los mismos.
func TestAutofetchVerbsAreListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()
	for _, verb := range []string{"enable", "disable", "status", "run"} {
		if !strings.Contains(help, "\n  autofetch "+verb+" ") {
			t.Errorf("la ayuda global debe listar `autofetch %s`:\n%s", verb, help)
		}
	}
	for _, line := range strings.Split(help, "\n") {
		if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == "branch" && fields[1] == "autofetch" {
			t.Errorf("autofetch no es un subcomando de branch: %q", line)
		}
	}
}
