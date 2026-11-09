package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

func stubManagedDigest(t *testing.T, digest string) {
	t.Helper()
	orig := doctorManagedDigestFn
	t.Cleanup(func() { doctorManagedDigestFn = orig })
	doctorManagedDigestFn = func() (string, error) { return digest, nil }
}

func writeProjectState(t *testing.T, home, root string, s state.InstallState) {
	t.Helper()
	if err := state.ProjectStore(home, root).Write(s); err != nil {
		t.Fatal(err)
	}
}

func TestProjectAssetsCheckComparesTheRecordedDigest(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())

	// Sin estado de proyecto no hay comprobaciones de proyecto (la salida de la CLI no cambia).
	if _, ok := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAssets]; ok {
		t.Fatalf("sin .axiom/state.json no debe haber comprobaciones de assets")
	}

	stubManagedDigest(t, "bbbbbbbbbbbbbbbbbbbb")
	writeProjectState(t, home, root, state.InstallState{ManagedAssetDigest: "aaaaaaaaaaaaaaaaaaaa"})
	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAssets]
	if got.Status != doctor.StatusWarn || got.Scope != doctor.ScopeProject {
		t.Fatalf("digest distinto = %+v, want warn de proyecto", got)
	}
	if got.Remedy == nil || !strings.Contains(got.Remedy.Description, "axiom sync --scope workspace") {
		t.Fatalf("remedio = %+v, want 'axiom sync --scope workspace'", got.Remedy)
	}

	writeProjectState(t, home, root, state.InstallState{ManagedAssetDigest: "bbbbbbbbbbbbbbbbbbbb"})
	if got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAssets]; got.Status != doctor.StatusPass {
		t.Fatalf("digest igual = %+v, want pass", got)
	}

	writeProjectState(t, home, root, state.InstallState{})
	if got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAssets]; got.Status != doctor.StatusPass {
		t.Fatalf("sin digest registrado = %+v, want pass (omitida)", got)
	}
}

func TestProjectAdaptersCheckLooksAtTheProjectRoot(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeProjectState(t, home, root, state.InstallState{InstalledAgents: []string{"claude-code", "agente-fantasma"}})

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAdapters]
	if got.Status != doctor.StatusWarn || !strings.Contains(got.Detail, "claude-code") || !strings.Contains(got.Detail, "agente-fantasma") {
		t.Fatalf("agentes ausentes = %+v, want warn que nombra ambos", got)
	}
	if got.Remedy == nil || !strings.Contains(got.Remedy.Description, "--scope workspace") {
		t.Fatalf("remedio = %+v", got.Remedy)
	}

	// La configuración del HOME no cuenta: se mira la raíz del proyecto.
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeProjectState(t, home, root, state.InstallState{InstalledAgents: []string{"claude-code"}})
	if got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAdapters]; got.Status != doctor.StatusWarn {
		t.Fatalf("config solo en HOME = %+v, want warn", got)
	}

	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectAdapters]; got.Status != doctor.StatusPass {
		t.Fatalf("config en la raíz = %+v, want pass", got)
	}
}
