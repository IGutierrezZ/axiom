package cli

import (
	"bytes"
	"io"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// Aislamiento dinámico de la operación remota (DD-14, REQ-27.11, REQ-25.1): sobre un proyecto con un
// remoto desnudo, ni `axiom branch` ni `axiom sdd gate record` ni `axiom sdd status` ejecutan `push`,
// `commit` ni `fetch` (sin `--fetch`). La publicación solo existe en `axiom sdd publish`, que es el
// único verbo que alcanza el puerto de escritura de git (la prueba estática está en
// internal/sddpublish/isolation_test.go).
func TestPublishIsolationNoVerbButPublishPushesCommitsOrFetches(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	bare := t.TempDir()
	integrateGit(t, bare, "init", "-q", "--bare", "-b", "main")
	integrateGit(t, ws.Master, "remote", "add", "origin", bare)
	integrateGit(t, ws.Master, "push", "-q", "origin", "main", "feat/x")

	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) { seen = append(seen, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })

	// Primero la puerta abierta y las decisiones, con el repositorio en la rama del cambio.
	var status bytes.Buffer
	if err := RunSDDStatus([]string{"x", "--cwd", ws.Master, "--json"}, &status); err != nil {
		t.Fatalf("RunSDDStatus con la puerta abierta: %v", err)
	}
	for _, args := range [][]string{
		{"--gate", "spec", "--decision", "rejected", "--reason", "falta un escenario"},
		{"--gate", "spec", "--decision", "approved", "--hand-off"},
		{"--gate", "design", "--decision", "approved"},
	} {
		if _, err := gateRecord(t, ws, args...); err != nil {
			t.Fatalf("gate record %v: %v", args, err)
		}
	}

	// Después `axiom branch`, cuyas operaciones pueden cambiar de rama en `in_place`.
	branchVerbs := [][]string{
		{"status", "--change", "x", "--cwd", ws.Master, "--json"},
		{"ensure", "--change", "x", "--mode", "in_place", "--cwd", ws.Master, "--json"},
		{"ensure", "--change", "x", "--archive", "--mode", "in_place", "--cwd", ws.Master, "--json"},
		{"done", "--change", "x", "--cwd", ws.Master, "--json"},
		{"done", "--change", "x", "--archive", "--cwd", ws.Master, "--json"},
	}
	for _, args := range branchVerbs {
		// Una negativa tipada (cambio sin fusionar, por ejemplo) no importa: lo que se observa es lo
		// que ejecutó git.
		RunBranch(args, io.Discard, io.Discard)
	}

	if len(seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range seen {
		if slices.Contains([]string{"push", "commit", "fetch", "pull", "ls-remote", "clone"}, invocation[1]) {
			t.Errorf("ninguno de estos verbos puede ejecutar %s: %v", invocation[1], invocation)
		}
	}
}
