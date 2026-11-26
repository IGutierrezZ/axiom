package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

var integrationsVerbs = []string{"sync", "emit", "secret", "connect", "discover", "tasks", "summary"}

// RF-INT-18: la ayuda global lista los verbos de integraciones, con el texto del diseño y sus
// descripciones alineadas en la misma columna.
func TestIntegrationsVerbsAreListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()
	for _, line := range []string{
		"\n  integrations sync      Reintenta las entregas pendientes a los trackers configurados en axiom.yaml (--change, --json)",
		"\n  integrations emit      Emite a mano un evento del ciclo de vida de un cambio a los plugins configurados (--change, --event)",
		"\n  integrations secret    Guarda (set, solo por stdin), consulta (status) o borra (delete) un secreto de integración",
		"\n  integrations connect   Prueba la conexión con el tracker y guarda tu identidad en ~/.axiom (--plugin, --user)",
		"\n  integrations discover  Imprime el proceso del tracker: tipos, estados, campos, iteraciones y miembros (--plugin, --json)",
	} {
		if !strings.Contains(help, line) {
			t.Errorf("la ayuda global debe contener %q:\n%s", line, help)
		}
	}
	listed := 0
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "integrations" {
			continue
		}
		listed++
		known := false
		for _, verb := range integrationsVerbs {
			known = known || fields[1] == verb
		}
		if !known {
			t.Errorf("la ayuda anuncia `integrations %s`, que no existe: %q", fields[1], line)
		}
	}
	if listed != len(integrationsVerbs) {
		t.Errorf("la ayuda global lista %d verbos de integrations, quería %d", listed, len(integrationsVerbs))
	}
}

// `axiom integrations --help`, cada subcomando y un subcomando desconocido, a través de la misma
// función que despacha cmd/axiom.
func TestIntegrationsHelpAndUnknownSubcommand(t *testing.T) {
	run := func(args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		code := cli.RunIntegrations(args, strings.NewReader(""), &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}
	code, out, _ := run("--help")
	if code != 0 {
		t.Errorf("--help: %d", code)
	}
	for _, verb := range integrationsVerbs {
		if !strings.Contains(out, "\n  "+verb) {
			t.Errorf("la ayuda de la familia debe listar %s:\n%s", verb, out)
		}
	}
	for _, sub := range integrationsVerbs {
		if code, out, _ := run(sub, "--help"); code != 0 || !strings.Contains(out, "axiom integrations "+sub) {
			t.Errorf("%s --help: %d %q", sub, code, out)
		}
	}
	code, out, errOut := run("inspect")
	if code == 0 || !strings.Contains(errOut, "inspect") {
		t.Errorf("subcomando desconocido: %d %q", code, errOut)
	}
	for _, verb := range integrationsVerbs {
		if !strings.Contains(out, "\n  "+verb) {
			t.Errorf("un subcomando desconocido muestra la ayuda de la familia con %s:\n%s", verb, out)
		}
	}
}
