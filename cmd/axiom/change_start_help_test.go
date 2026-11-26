package main

import (
	"bytes"
	"strings"
	"testing"
)

// `axiom change --help` lista `start` con sus tres modos y el subcomando se delega en cli.RunChangeStart.
func TestChangeHelpListsStart(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runChange([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("código %d, error %q", code, stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{"\n  start ", "--from", "--adopt", "--discard"} {
		if !strings.Contains(help, want) {
			t.Fatalf("la ayuda de `axiom change` debe contener %q:\n%s", want, help)
		}
	}

	stdout.Reset()
	if code := runChange([]string{"start", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "axiom.change-start/v1") {
		t.Fatalf("`axiom change start --help`: código %d, salida %q", code, stdout.String())
	}

	stderr.Reset()
	if code := runChange([]string{"nope"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "[create|new|start]") {
		t.Fatalf("subcomando desconocido: código %d, error %q", code, stderr.String())
	}
}

func TestAxiomHelpMentionsChangeStart(t *testing.T) {
	if help := axiomHelpText(); !strings.Contains(help, "\n  change start ") {
		t.Fatalf("la ayuda global debe listar `change start`:\n%s", help)
	}
}
