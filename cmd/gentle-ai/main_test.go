package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGentleAICompatWrapper_RetiredNotice(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--version"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("run() debía devolver error de comando retirado, obtuvo nil")
	}

	errStr := stderr.String()
	if !strings.Contains(errStr, retiredNotice) {
		t.Fatalf("stderr no contiene el aviso de retiro esperado:\n%s", errStr)
	}
}

func TestGentleAICompatWrapper_HelpFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--help"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("run() con --help debía devolver error, obtuvo nil")
	}

	if !strings.Contains(stderr.String(), retiredNotice) {
		t.Fatalf("stderr no contiene el aviso de retiro con --help")
	}
}
