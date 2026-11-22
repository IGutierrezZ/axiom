package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Criterio de terminado del backlog (C-07-b): con el doble configurado en axiom.yaml, los verbos
// reales entregan sus eventos con su clave, los vínculos llegan a integrations.yaml, un fallo se
// recupera con sync y, sin bloque, no hay ningún coste.

func e2eGate(t *testing.T, worktree string, args ...string) {
	t.Helper()
	var out bytes.Buffer
	full := append([]string{"record", "--cwd", worktree, "--change", "x", "--decision", "approved", "--reason", "ok", "--actor", "Ana <ana@example.test>"}, args...)
	if err := RunSDDGate(full, &out); err != nil {
		t.Fatalf("gate record %v: %v", args, err)
	}
}

func TestIntegrationsE2EDoubleReceivesEveryEventWithItsKey(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")
	fake.ReturnLinks(integrations.Link{Kind: "work_item", ID: "42", URL: "https://tracker.example/42"})
	warnings := captureIntegrationWarnings(t)

	// role.start desde ensure --role.
	if code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--role", "backend", "--cwd", repo); code != 0 {
		t.Fatalf("ensure --role: %d %q", code, stderr)
	}
	worktree := plannedBranchWorktree(t, repo, "x")
	changeRoot := filepath.Join(worktree, "openspec", "changes", "x")
	if err := os.WriteFile(filepath.Join(changeRoot, "design.md"), []byte("diseño\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// gate.spec.approved, gate.design.approved y role.apply.approved desde gate record.
	e2eGate(t, worktree, "--gate", "spec")
	e2eGate(t, worktree, "--gate", "design")
	e2eGate(t, worktree, "--gate", "role-apply:backend")

	keys := fake.DeliveredKeys()
	wantPrefixes := []string{"role.start/x/backend", "gate.spec.approved/x/", "gate.design.approved/x/", "role.apply.approved/x/backend/"}
	if len(keys) != len(wantPrefixes) {
		t.Fatalf("claves entregadas = %v", keys)
	}
	for i, prefix := range wantPrefixes {
		if !strings.HasPrefix(keys[i], prefix) {
			t.Errorf("clave %d = %q, se esperaba el prefijo %q", i, keys[i], prefix)
		}
	}

	// emit a mano de un evento: llega al doble con su clave.
	calls := fake.Calls()
	code, out, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", worktree, "--change", "x", "--event", "role.start", "--role", "backend", "--json")
	if code != 0 || !strings.Contains(out, "role.start/x/backend") {
		t.Fatalf("emit: %d %q %q", code, out, errOut)
	}
	if fake.Calls() != calls+1 {
		t.Fatal("emit entrega el evento al doble")
	}

	// integrations.yaml contiene los vínculos devueltos, sin duplicar.
	links, err := os.ReadFile(filepath.Join(changeRoot, "integrations.yaml"))
	if err != nil {
		t.Fatalf("integrations.yaml: %v", err)
	}
	if strings.Count(string(links), "id: \"42\"") != 1 || !strings.Contains(string(links), "https://tracker.example/42") {
		t.Fatalf("vínculos:\n%s", links)
	}
	if fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("entregado todo, no hay cola")
	}
	if warnings.Len() != 0 {
		t.Fatalf("sin fallos no hay avisos: %q", warnings)
	}
}

func TestIntegrationsE2ERetryAfterFailureWithSyncAndExitCodes(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")
	captureIntegrationWarnings(t)
	if code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo); code != 0 {
		t.Fatalf("ensure: %d %q", code, stderr)
	}
	worktree := plannedBranchWorktree(t, repo, "x")
	changeRoot := filepath.Join(worktree, "openspec", "changes", "x")
	queue := filepath.Join(changeRoot, ".integrations-outbox.yaml")

	fake.FailAlways(errors.New("tracker caído"))
	e2eGate(t, worktree, "--gate", "spec")
	if !fileExistsCLI(queue) {
		t.Fatal("la operación queda pendiente")
	}
	if code, _, _ := runIntegrationsCLI(t, "", "sync", "--cwd", worktree); code != 1 {
		t.Fatalf("con algo pendiente sync sale con 1: %d", code)
	}

	fake.Recover()
	if code, _, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", worktree); code != 0 {
		t.Fatalf("recuperado el tracker, sync entrega y sale con 0: %d %q", code, errOut)
	}
	if fileExistsCLI(queue) {
		t.Fatal("la cola se borra al vaciarse")
	}
	if keys := fake.DeliveredKeys(); len(keys) != 1 || !strings.HasPrefix(keys[0], "gate.spec.approved/x/") {
		t.Fatalf("entregas = %v", keys)
	}

	// --retry-abandoned y --discard: diez intentos abandonan la operación.
	fake.FailAlways(errors.New("sigue caído"))
	for i := 0; i < integrations.MaxAttempts; i++ {
		runIntegrationsCLI(t, "", "emit", "--cwd", worktree, "--change", "x", "--event", "role.start", "--role", "backend")
	}
	code, out, _ := runIntegrationsCLI(t, "", "sync", "--cwd", worktree, "--json")
	if code != 1 || !strings.Contains(out, "abandoned") || !strings.Contains(out, "role.start/x/backend") {
		t.Fatalf("abandonada: %d %s", code, out)
	}
	fake.Recover()
	if code, _, _ := runIntegrationsCLI(t, "", "sync", "--cwd", worktree, "--discard", "role.start/x/backend"); code != 0 {
		t.Fatalf("--discard retira la operación: %d", code)
	}
	if fileExistsCLI(queue) {
		t.Fatal("tras --discard no queda cola")
	}
}

func TestIntegrationsE2EZeroCostWithoutTheBlock(t *testing.T) {
	repo, fake := roleRepository(t, "", "backend")
	warnings := captureIntegrationWarnings(t)

	if code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--role", "backend", "--cwd", repo); code != 0 {
		t.Fatalf("ensure --role: %d %q", code, stderr)
	}
	worktree := plannedBranchWorktree(t, repo, "x")
	e2eGate(t, worktree, "--gate", "spec")
	e2eGate(t, worktree, "--gate", "role-apply:backend")

	archiveRepo, _, archiveFake := archiveProject(t, "", false)
	publishArchive(t, archiveRepo)

	for _, root := range []string{filepath.Join(worktree, "openspec", "changes", "x")} {
		for _, name := range []string{".integrations-outbox.yaml", "integrations.yaml", ".integrations.lock"} {
			if fileExistsCLI(filepath.Join(root, name)) {
				t.Errorf("sin bloque no debe existir %s", name)
			}
		}
	}
	for _, git := range []string{repo, archiveRepo} {
		if fileExistsCLI(filepath.Join(git, ".git", "axiom", "integrations")) {
			t.Errorf("sin bloque no debe existir el directorio axiom/integrations de %s", git)
		}
	}
	if fake.Calls() != 0 || archiveFake.Calls() != 0 || warnings.Len() != 0 {
		t.Fatalf("sin bloque ni entregas ni avisos: %d %d %q", fake.Calls(), archiveFake.Calls(), warnings)
	}
}
