package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// gateRecordRun ejecuta `gate record` en proceso y devuelve su stdout normalizado (la raíz del
// espacio sustituida por <ROOT>, también escapada para JSON) y el error.
func gateRecordRun(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSDDGate(append([]string{"record", "--cwd", root, "--change", "x"}, args...), &out)
	return normalizeRoot(out.String(), root), err
}

func normalizeRoot(text, root string) string {
	escaped := strings.ReplaceAll(root, `\`, `\\`)
	return strings.ReplaceAll(strings.ReplaceAll(text, escaped, "<ROOT>"), root, "<ROOT>")
}

func writeDesign(t *testing.T, changeRoot, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(changeRoot, "design.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGateRecordIntegrationApprovalsEmitEventsWithTheRecordedActor(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	captureIntegrationWarnings(t)
	writeDesign(t, changeRoot, "diseño\n")
	if err := os.MkdirAll(filepath.Join(changeRoot, "specs", "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeRoot, "specs", "d", "spec.md"), []byte("spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, gate := range []string{"spec", "design", "role-apply:fullstack"} {
		if _, err := gateRecordRun(t, root, "--gate", gate, "--decision", "approved", "--reason", "ok", "--actor", "Ana <ana@example.test>"); err != nil {
			t.Fatalf("gate record %s: %v", gate, err)
		}
	}

	specDigest, _ := kickoff.ArtifactDigest([]string{filepath.Join(changeRoot, "specs", "d", "spec.md")})
	designDigest, _ := kickoff.ArtifactDigest([]string{filepath.Join(changeRoot, "design.md")})
	got := fake.Delivered()
	if len(got) != 3 {
		t.Fatalf("eventos entregados = %+v", got)
	}
	wantKeys := []string{
		"gate.spec.approved/x/d" + specDigest[:16],
		"gate.design.approved/x/d" + designDigest[:16],
	}
	if got[0].IdempotencyKey != wantKeys[0] || got[1].IdempotencyKey != wantKeys[1] {
		t.Fatalf("claves = %v %v, se esperaban %v", got[0].IdempotencyKey, got[1].IdempotencyKey, wantKeys)
	}
	if !strings.HasPrefix(got[2].IdempotencyKey, "role.apply.approved/x/fullstack/") || got[2].Role != "fullstack" {
		t.Fatalf("role.apply.approved = %+v", got[2])
	}
	for _, ev := range got {
		if ev.Actor != "Ana <ana@example.test>" {
			t.Errorf("el evento lleva el actor registrado: %+v", ev)
		}
	}
}

func TestGateRecordIntegrationTasksAndRejectedEmitNothingAndCreateNoFiles(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	captureIntegrationWarnings(t)

	for _, args := range [][]string{
		{"--gate", "tasks", "--decision", "approved", "--reason", "ok"},
		{"--gate", "spec", "--decision", "rejected", "--reason", "falta"},
		{"--gate", "design", "--decision", "rejected", "--reason", "falta"},
	} {
		if _, err := gateRecordRun(t, root, args...); err != nil {
			t.Fatalf("gate record %v: %v", args, err)
		}
	}
	if fake.Calls() != 0 {
		t.Fatalf("tasks y rejected no emiten (llamadas %d)", fake.Calls())
	}
	for _, name := range []string{".integrations-outbox.yaml", "integrations.yaml", ".integrations.lock"} {
		if fileExistsCLI(filepath.Join(changeRoot, name)) {
			t.Errorf("no debe crearse %s", name)
		}
	}
}

func TestGateRecordIntegrationFailingPluginKeepsStdoutAndExitsZero(t *testing.T) {
	// Línea base: el mismo recorrido sin bloque integrations.
	baseRoot, baseChange, _ := integrationsWorkspace(t, "")
	sealForGateTest(t, baseRoot, "x")
	writeDesign(t, baseChange, "diseño\n")
	baseText, err := gateRecordRun(t, baseRoot, "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana")
	if err != nil {
		t.Fatal(err)
	}
	baseJSON, err := gateRecordRun(t, baseRoot, "--gate", "spec", "--decision", "approved", "--reason", "ok", "--actor", "Ana", "--json")
	if err != nil {
		t.Fatal(err)
	}

	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	fake.FailAlways(errors.New("tracker caído"))
	sealForGateTest(t, root, "x")
	warnings := captureIntegrationWarnings(t)
	writeDesign(t, changeRoot, "diseño\n")

	text, err := gateRecordRun(t, root, "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana")
	if err != nil {
		t.Fatalf("con el plugin fallando el verbo sale con éxito: %v", err)
	}
	if text != baseText {
		t.Fatalf("stdout idéntico con y sin integración:\n%s\n---\n%s", text, baseText)
	}
	jsonOut, err := gateRecordRun(t, root, "--gate", "spec", "--decision", "approved", "--reason", "ok", "--actor", "Ana", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if jsonOut != baseJSON {
		t.Fatalf("axiom.sdd-gate-record/v1 idéntico byte a byte con y sin integración:\n%s\n---\n%s", jsonOut, baseJSON)
	}
	if !strings.Contains(warnings.String(), "Aviso:") || !strings.Contains(warnings.String(), "axiom integrations sync") {
		t.Fatalf("el fallo avisa por stderr: %q", warnings)
	}
	if !fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("la operación queda pendiente en la cola")
	}
}

func TestGateRecordIntegrationRepeatedLeavesOnePendingOperationAndChangedArtifactAnother(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	fake.FailAlways(errors.New("tracker caído"))
	sealForGateTest(t, root, "x")
	captureIntegrationWarnings(t)
	writeDesign(t, changeRoot, "diseño uno\n")

	pending := func() int {
		data, err := os.ReadFile(filepath.Join(changeRoot, ".integrations-outbox.yaml"))
		if err != nil {
			return 0
		}
		return strings.Count(string(data), "  - key: ")
	}
	record := func() {
		if _, err := gateRecordRun(t, root, "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana"); err != nil {
			t.Fatal(err)
		}
	}
	record()
	record()
	if pending() != 1 {
		t.Fatalf("repetir sobre el mismo artefacto deja una sola operación pendiente, hay %d", pending())
	}
	writeDesign(t, changeRoot, "diseño dos\n")
	record()
	if pending() != 2 {
		t.Fatalf("reaprobar un artefacto cambiado genera otra operación con otra clave, hay %d", pending())
	}
}

func TestGateRecordIntegrationInvalidBlockWarnsAndDoesNotDispatch(t *testing.T) {
	root, changeRoot, _ := integrationsWorkspace(t, "integrations:\n  azure: {}\n")
	sealForGateTest(t, root, "x")
	warnings := captureIntegrationWarnings(t)
	writeDesign(t, changeRoot, "diseño\n")

	if _, err := gateRecordRun(t, root, "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana"); err != nil {
		t.Fatalf("el bloque inválido no hace fallar el verbo: %v", err)
	}
	if !strings.Contains(warnings.String(), "no es válido") || !strings.Contains(warnings.String(), "axiom workspace validate") {
		t.Fatalf("aviso = %q", warnings)
	}
	if fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("no se despacha")
	}
}
