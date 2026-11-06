package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

func doctorCheckByName(t *testing.T, report *DoctorReport, name string) DoctorCheck {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("el informe no contiene %q: %+v", name, report.Checks)
	return DoctorCheck{}
}

func TestGetDoctorDiagnosticsReadsTheOpenProjectNotTheCwd(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := state.ProjectStore(sandboxHome, root).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(elsewhere)

	report, err := NewService(root).GetDoctorDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	if report.Project != root {
		t.Errorf("project = %q, want %q", report.Project, root)
	}
	st := doctorCheckByName(t, report, "state:json")
	if st.Category != "project" {
		t.Errorf("state:json category = %q, want project (el estado del proyecto abierto)", st.Category)
	}
	if st.Details == "" || st.Status == "" {
		t.Errorf("state:json sin contenido real: %+v", st)
	}
	if got := doctorCheckByName(t, report, "disk:space").Category; got != "global" {
		t.Errorf("disk:space category = %q, want global", got)
	}
	// El agente registrado en el proyecto abierto se exige como herramienta.
	doctorCheckByName(t, report, "tool:claude")
}

func TestGetDoctorDiagnosticsHealthyFollowsFailures(t *testing.T) {
	root := t.TempDir()
	statePath := state.ProjectPath(root)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := NewService(root).GetDoctorDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	if st := doctorCheckByName(t, report, "state:json"); st.Status != "error" || st.Recommendation == "" {
		t.Fatalf("state:json = %+v, want error con remedio", st)
	}
	if report.Healthy {
		t.Fatalf("Healthy = true con un chequeo en error")
	}

}

func TestDoctorStatusView(t *testing.T) {
	for in, want := range map[doctor.Status]string{doctor.StatusPass: "ok", doctor.StatusWarn: "warning", doctor.StatusFail: "error"} {
		if got := doctorStatusView(in); got != want {
			t.Errorf("doctorStatusView(%s) = %s, want %s", in, got, want)
		}
	}
}
