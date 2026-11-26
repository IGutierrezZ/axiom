package main

import (
	"bytes"
	"strings"
	"testing"
)

// C-03-d (prov:trace:1): `axiom trace` está registrado en el despachador y en la ayuda global.
// Los nombres empiezan por TestTraceVerb para entrar en el mismo patrón de `-run`.

func TestTraceVerbIsListedInTheGlobalHelp(t *testing.T) {
	if help := axiomHelpText(); !strings.Contains(help, "\n  trace ") {
		t.Fatalf("la ayuda global debe listar `trace`:\n%s", help)
	}
}

func TestTraceVerbWithoutIDFailsWithUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runTrace(nil, &stdout, &stderr); code != 1 {
		t.Fatalf("runTrace(nil) = %d, quería 1", code)
	}
	if !strings.Contains(stderr.String(), "requiere un ID") || stdout.Len() != 0 {
		t.Fatalf("stderr = %q stdout = %q", stderr.String(), stdout.String())
	}
}

func TestTraceVerbHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runTrace([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("runTrace(--help) = %d, quería 0 (%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Uso: axiom trace") || stderr.Len() != 0 {
		t.Fatalf("stdout = %q stderr = %q", stdout.String(), stderr.String())
	}
}
