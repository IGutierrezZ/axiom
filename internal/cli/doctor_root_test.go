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

// TestBuildDoctorChecksUsesTheAgentsSyncManagesWhenTheProjectHasNotRecordedThem fija
// que un proyecto con estado propio pero sin lista de agentes (`installed_agents` nulo,
// como lo deja `axiom sync --scope workspace` sin instalación previa) se comprueba con
// los agentes que sync descubre por ficheros y gestiona, no como un proyecto sin
// agentes: antes, engram:reachable no sondeaba el MCP del proyecto y mandaba a
// `axiom sync`, que no registra agentes, así que el aviso no se iba nunca.
func TestBuildDoctorChecksUsesTheAgentsSyncManagesWhenTheProjectHasNotRecordedThem(t *testing.T) {
	stubDoctorWorld(t)
	home := t.TempDir()
	root := t.TempDir()
	if err := state.ProjectStore(home, root).Write(state.InstallState{InstalledAgents: nil}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(root, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{"engram":{"command":"engram","args":["mcp","--tools=agent"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeApprovalFixture(t, filepath.Join(root, ".claude", "settings.local.json"), `{"enabledMcpjsonServers":["engram"]}`)
	t.Chdir(t.TempDir())

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})

	st := got[doctor.CheckStateJSON]
	if st.Status != doctor.StatusPass || !strings.Contains(st.Detail, "has not recorded") || !strings.Contains(st.Detail, "claude-code") {
		t.Fatalf("state:json = %+v, want un pass que diga que el proyecto no ha registrado agentes y nombre los que gestiona sync", st)
	}
	if _, ok := got[doctor.ToolCheckID("claude")]; !ok {
		t.Fatalf("la herramienta del agente que gestiona sync debe comprobarse: %v", got)
	}
	engramResult := got[doctor.CheckEngramReachable]
	if engramResult.Status != doctor.StatusPass || !strings.Contains(engramResult.Detail, mcpPath) {
		t.Fatalf("engram:reachable = %+v, want un pass que sondee el MCP del proyecto %s", engramResult, mcpPath)
	}
}

// TestBuildDoctorChecksRespectsAnEmptyRecordedAgentList fija la otra mitad: una lista
// registrada y vacía es una selección (ningún agente), no una lista sin registrar, así
// que el doctor no descubre agentes por ficheros y manda a instalar.
func TestBuildDoctorChecksRespectsAnEmptyRecordedAgentList(t *testing.T) {
	stubDoctorWorld(t)
	home := t.TempDir()
	root := t.TempDir()
	if err := state.ProjectStore(home, root).Write(state.InstallState{InstalledAgents: []string{}}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})

	st := got[doctor.CheckStateJSON]
	if st.Status != doctor.StatusWarn || !strings.Contains(st.Detail, "no installed agents") {
		t.Fatalf("state:json = %+v, want el aviso de proyecto sin agentes", st)
	}
	if _, ok := got[doctor.ToolCheckID("claude")]; ok {
		t.Fatalf("una lista vacía registrada no debe exigir la herramienta de un agente descubierto: %v", got)
	}
	engramResult := got[doctor.CheckEngramReachable]
	if engramResult.Remedy == nil || engramResult.Remedy.ID != doctor.RemedyInstall {
		t.Fatalf("engram:reachable = %+v, want el remedio %q y no 'axiom sync'", engramResult, doctor.RemedyInstall)
	}
}
