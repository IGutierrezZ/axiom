package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
)

func writeWorkspaceDoctorFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceChecksSkipProjectsWithoutAxiomYAML(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	for id := range runDoctorChecks(t, doctor.Options{Home: home, Root: root}) {
		if strings.HasPrefix(string(id), "project:") {
			t.Fatalf("un proyecto sin axiom.yaml no debe tener %s", id)
		}
	}
}

func TestWorkspaceChecksMonorepo(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeWorkspaceDoctorFile(t, filepath.Join(root, "axiom.yaml"), "workspace:\n  name: demo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  dev:\n    name: Dev\n    repositories:\n      - path: .\n")

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})
	if r := got[doctor.CheckProjectWorkspace]; r.Status != doctor.StatusPass || r.Scope != doctor.ScopeProject {
		t.Fatalf("workspace = %+v", r)
	}
	if r := got[doctor.CheckProjectTopology]; r.Status != doctor.StatusPass || !strings.Contains(r.Detail, "monorepo-embedded") {
		t.Fatalf("topology = %+v", r)
	}
	if r := got[doctor.CheckProjectRoleRepos]; r.Status != doctor.StatusWarn {
		t.Fatalf("role repos sin git = %+v, want warn", r)
	}
	// Sin .git la raíz no es un repositorio: las specs no están versionadas.
	if r := got[doctor.CheckProjectSpecsRepo]; r.Status != doctor.StatusWarn || r.Remedy == nil {
		t.Fatalf("specs sin git = %+v, want warn con remedio", r)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
		after := runDoctorChecks(t, doctor.Options{Home: home, Root: root})
		if r := after[doctor.CheckProjectSpecsRepo]; r.Status != doctor.StatusPass {
			t.Fatalf("specs con git = %+v, want pass", r)
		}
		if r := after[doctor.CheckProjectRoleRepos]; r.Status != doctor.StatusPass {
			t.Fatalf("role repos con git = %+v, want pass", r)
		}
	}
}

func TestWorkspaceChecksMultirepoRolesAndSpecs(t *testing.T) {
	stubDoctorWorld(t)
	home, master := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeWorkspaceDoctorFile(t, filepath.Join(master, "axiom.yaml"), `workspace:
  name: demo
  topology: multirepo
  specs_repository: specs
roles:
  backend:
    name: Backend
    repositories:
      - path: api
      - path: worker
  frontend:
    name: Frontend
    repositories:
      - path: web
`)
	opts := doctor.Options{Home: home, Root: master}

	// Faltan los repositorios de rol y el de specs existe pero no es git.
	if err := os.MkdirAll(filepath.Join(master, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := runDoctorChecks(t, opts)
	if r := got[doctor.CheckProjectRoleRepos]; r.Status != doctor.StatusFail || !strings.Contains(r.Detail, "api (role backend)") || !strings.Contains(r.Detail, "web (role frontend)") {
		t.Fatalf("roles ausentes = %+v, want fail que nombra los repos", r)
	}
	if r := got[doctor.CheckProjectSpecsRepo]; r.Status != doctor.StatusWarn {
		t.Fatalf("specs sin git = %+v, want warn", r)
	}
	if r := got[doctor.CheckProjectWorkspace]; r.Status != doctor.StatusPass {
		t.Fatalf("workspace = %+v: los repos de rol los juzga project:role_repos, no esta comprobación", r)
	}
	if r := got[doctor.CheckProjectTopology]; !strings.Contains(r.Detail, "multirepo") || !strings.Contains(r.Detail, master) {
		t.Fatalf("topology = %+v, want topología y carpeta maestra", r)
	}

	// Existen todos pero solo api, web y specs son git: worker es un aviso.
	for _, name := range []string{"api", "web", "specs"} {
		if err := os.MkdirAll(filepath.Join(master, name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(master, "worker"), 0o755); err != nil {
		t.Fatal(err)
	}
	got = runDoctorChecks(t, opts)
	if r := got[doctor.CheckProjectRoleRepos]; r.Status != doctor.StatusWarn || !strings.Contains(r.Detail, "worker (role backend)") {
		t.Fatalf("repo sin git = %+v, want warn", r)
	}
	if r := got[doctor.CheckProjectSpecsRepo]; r.Status != doctor.StatusPass {
		t.Fatalf("specs = %+v, want pass", r)
	}
	if r := got[doctor.CheckProjectWorkspace]; r.Status != doctor.StatusPass {
		t.Fatalf("workspace = %+v, want pass", r)
	}

	if err := os.MkdirAll(filepath.Join(master, "worker", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r := runDoctorChecks(t, opts)[doctor.CheckProjectRoleRepos]; r.Status != doctor.StatusPass {
		t.Fatalf("todos git = %+v, want pass", r)
	}
}

func TestWorkspaceChecksReportAnUnloadableConfig(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeWorkspaceDoctorFile(t, filepath.Join(root, "axiom.yaml"), "workspace: [esto no es un mapa")

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: root})
	r := got[doctor.CheckProjectWorkspace]
	if r.Status != doctor.StatusFail || r.Remedy == nil {
		t.Fatalf("config ilegible = %+v, want fail con remedio", r)
	}
	if _, ok := got[doctor.CheckProjectRoleRepos]; ok {
		t.Fatalf("sin configuración cargada no hay comprobación de repos")
	}
}

func TestWorkspaceChecksReportAMissingSpecsRepository(t *testing.T) {
	stubDoctorWorld(t)
	home, master := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeWorkspaceDoctorFile(t, filepath.Join(master, "axiom.yaml"), `workspace:
  name: demo
  topology: multirepo
  specs_repository: specs
roles:
  dev:
    name: Dev
    repositories:
      - path: api
`)

	got := runDoctorChecks(t, doctor.Options{Home: home, Root: master})
	r := got[doctor.CheckProjectWorkspace]
	if r.Status != doctor.StatusFail || !strings.Contains(r.Detail, "specs") || r.Remedy == nil {
		t.Fatalf("specs ausente = %+v, want fail que explica la causa", r)
	}
}
