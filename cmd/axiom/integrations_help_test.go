package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// REQ-43.18: la ayuda global lista los tres verbos de integraciones, con el texto del diseño.
func TestIntegrationsVerbsAreListedInTheGlobalHelp(t *testing.T) {
	help := axiomHelpText()
	for _, line := range []string{
		"\n  integrations sync    Reintenta las entregas pendientes a los trackers configurados en axiom.yaml (--change, --json)",
		"\n  integrations emit    Emite a mano un evento del ciclo de vida de un cambio a los plugins configurados (--change, --event)",
		"\n  integrations secret  Guarda (set, solo por stdin), consulta (status) o borra (delete) un secreto de integración",
	} {
		if !strings.Contains(help, line) {
			t.Errorf("la ayuda global debe contener %q:\n%s", line, help)
		}
	}
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "integrations" && fields[1] != "sync" && fields[1] != "emit" && fields[1] != "secret" {
			t.Errorf("la ayuda anuncia `integrations %s`, que no existe: %q", fields[1], line)
		}
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
	if code, out, _ := run("--help"); code != 0 || !strings.Contains(out, "sync") || !strings.Contains(out, "emit") || !strings.Contains(out, "secret") {
		t.Errorf("--help: %d %q", code, out)
	}
	for _, sub := range []string{"sync", "emit", "secret"} {
		if code, out, _ := run(sub, "--help"); code != 0 || !strings.Contains(out, "axiom integrations "+sub) {
			t.Errorf("%s --help: %d %q", sub, code, out)
		}
	}
	if code, _, errOut := run("inspect"); code == 0 || !strings.Contains(errOut, "inspect") {
		t.Errorf("subcomando desconocido: %d %q", code, errOut)
	}
}
