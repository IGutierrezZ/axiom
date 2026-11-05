package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// stubDoctorWorld aísla el doctor de la máquina: herramientas presentes, disco
// de sobra y Engram sin sondear.
func stubDoctorWorld(t *testing.T) {
	t.Helper()
	origLook, origAvail, origHTTP, origPath := lookPathFn, availableBytesFn, httpGetFn, pathDirsFn
	t.Cleanup(func() {
		lookPathFn, availableBytesFn, httpGetFn, pathDirsFn = origLook, origAvail, origHTTP, origPath
	})
	stubDoctorUserPath(t, nil)
	lookPathFn = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	availableBytesFn = func(string) (int64, error) { return 1 << 30, nil }
	httpGetFn = func(string, time.Duration) (int, error) { return 200, nil }
	pathDirsFn = func() []string { return []string{"/usr/local/bin"} }
	t.Setenv(engramHealthEnvVar, "")
	t.Setenv("AXIOM_STATE_DIR", "")
	setStdioProbeForTest(t, nil)
}

func runDoctorChecks(t *testing.T, opts doctor.Options) map[doctor.CheckID]doctor.Result {
	t.Helper()
	report := (doctor.Runner{Checks: BuildDoctorChecks(opts)}).Run(context.Background())
	byID := map[doctor.CheckID]doctor.Result{}
	for _, c := range report.Checks {
		byID[c.Name] = c
	}
	return byID
}

func TestBuildDoctorChecksReadsTheGivenRootNotTheCwd(t *testing.T) {
	stubDoctorWorld(t)
	home := t.TempDir()
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := state.ProjectStore(home, root).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(elsewhere)

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})

	st := got[doctor.CheckStateJSON]
	if st.Scope != doctor.ScopeProject {
		t.Fatalf("state:json scope = %q, want project", st.Scope)
	}
	if st.Status != doctor.StatusPass || !strings.Contains(st.Detail, "claude-code") {
		t.Fatalf("state:json = %+v, want a pass listing the project agents", st)
	}
	if _, ok := got[doctor.ToolCheckID("claude")]; !ok {
		t.Fatalf("la herramienta del agente del proyecto debe comprobarse: %v", got)
	}
	for _, id := range []doctor.CheckID{doctor.ToolCheckID("axiom"), doctor.CheckEngramReachable, doctor.CheckDiskSpace} {
		if got[id].Scope != doctor.ScopeGlobal {
			t.Fatalf("%s scope = %q, want global", id, got[id].Scope)
		}
	}
}

func TestBuildDoctorChecksFallsBackToGlobalStateWithoutProjectState(t *testing.T) {
	stubDoctorWorld(t)
	home := t.TempDir()
	root := t.TempDir() // sin .axiom/state.json
	other := t.TempDir()
	if err := state.ProjectStore(home, other).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(other) // el cwd SÍ tiene estado de proyecto; no debe usarse

	for _, opts := range []doctor.Options{{Home: home, Root: root}, {Home: home}} {
		got := runDoctorChecks(t, opts)
		st := got[doctor.CheckStateJSON]
		if st.Scope != doctor.ScopeGlobal {
			t.Fatalf("opts %+v: state:json scope = %q, want global", opts, st.Scope)
		}
		if strings.Contains(st.Detail, state.ProjectPath(other)) {
			t.Fatalf("opts %+v: leyó el estado del cwd: %q", opts, st.Detail)
		}
		if _, ok := got[doctor.ToolCheckID("claude")]; ok {
			t.Fatalf("opts %+v: no debe exigir agentes del cwd", opts)
		}
	}
}
