package main

import (
	"bytes"
	"strings"
	"testing"
)

// REQ-30.15: la ayuda del grupo `archive` lista `context` junto a los verbos
// históricos, y la ayuda global lo anuncia.
func TestArchiveHelpListsTheContextVerb(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "help"} {
		var stdout, stderr bytes.Buffer
		if code := runArchive([]string{flag}, &stdout, &stderr); code != 0 {
			t.Fatalf("archive %s salió con %d: %s", flag, code, stderr.String())
		}
		help := stdout.String()
		for _, want := range []string{"Uso: axiom archive <subcomando>", "\n  sync ", "\n  list ", "\n  show ", "\n  coldstart ", "\n  context "} {
			if !strings.Contains(help, want) {
				t.Errorf("archive %s: la ayuda no contiene %q:\n%s", flag, want, help)
			}
		}
	}
	if !strings.Contains(axiomHelpText(), "\n  archive context ") {
		t.Errorf("la ayuda global debe listar `archive context`:\n%s", axiomHelpText())
	}
}

func TestArchiveContextIsDispatchedToTheCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"context"}, &stdout, &stderr); code != 1 {
		t.Fatalf("sin verbo debe salir con 1, salió con %d", code)
	}
	if !strings.Contains(stderr.String(), "archive context requiere un subcomando") || !strings.Contains(stderr.String(), "axiom archive context --help") {
		t.Errorf("stderr = %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runArchive([]string{"context", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Uso: axiom archive context") {
		t.Errorf("archive context --help = %d\n%s", code, stdout.String())
	}
}

func TestArchiveUnknownSubcommandNamesTheValidOnes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"inexistente"}, &stdout, &stderr); code != 1 {
		t.Fatalf("código = %d", code)
	}
	if !strings.Contains(stdout.String(), "sync|list|show|coldstart|integrate|adopt|context") {
		t.Errorf("la negativa nombra los subcomandos: %q", stdout.String())
	}
}

// REQ-30.2: `axiom archive integrate` está en la ayuda del grupo y de la ayuda global, y se
// despacha a la CLI.
func TestArchiveHelpListsTheIntegrateVerb(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("archive --help salió con %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\n  integrate ") {
		t.Errorf("la ayuda de archive debe listar `integrate`:\n%s", stdout.String())
	}
	if !strings.Contains(axiomHelpText(), "\n  archive integrate ") {
		t.Errorf("la ayuda global debe listar `archive integrate`:\n%s", axiomHelpText())
	}
}

func TestArchiveIntegrateIsDispatchedToTheCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"integrate"}, &stdout, &stderr); code != 1 {
		t.Fatalf("sin --odd debe salir con 1, salió con %d", code)
	}
	if !strings.Contains(stderr.String(), "archive integrate requiere --odd") || !strings.Contains(stderr.String(), "axiom archive integrate --odd <feature>") {
		t.Errorf("stderr = %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runArchive([]string{"integrate", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Uso: axiom archive integrate --odd <feature>") {
		t.Errorf("archive integrate --help = %d\n%s", code, stdout.String())
	}
}

// REQ-30.9 y REQ-30.15: la ayuda del grupo `archive` lista `integrate`, `adopt` y `context`, la
// ayuda global anuncia `archive adopt` y el verbo se despacha a la CLI.
func TestArchiveHelpListsTheAdoptVerb(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("archive --help salió con %d: %s", code, stderr.String())
	}
	for _, want := range []string{"\n  integrate ", "\n  adopt ", "\n  context "} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("la ayuda de archive debe listar %q:\n%s", want, stdout.String())
		}
	}
	if !strings.Contains(axiomHelpText(), "\n  archive adopt ") {
		t.Errorf("la ayuda global debe listar `archive adopt`:\n%s", axiomHelpText())
	}
}

func TestArchiveAdoptIsDispatchedToTheCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runArchive([]string{"adopt"}, &stdout, &stderr); code != 1 {
		t.Fatalf("sin verbo debe salir con 1, salió con %d", code)
	}
	if !strings.Contains(stderr.String(), "archive adopt requiere un subcomando") || !strings.Contains(stderr.String(), "axiom archive adopt --help") {
		t.Errorf("stderr = %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runArchive([]string{"adopt", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Uso: axiom archive adopt <scan|check>") {
		t.Errorf("archive adopt --help = %d\n%s", code, stdout.String())
	}
}
