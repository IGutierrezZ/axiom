package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newRDDManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	return mgr
}

func TestProjectRDDPersistsAndClears(t *testing.T) {
	mgr := newRDDManager(t)
	project := t.TempDir()
	rec, err := mgr.Register(project, "p", "")
	if err != nil {
		t.Fatal(err)
	}

	if mode, found, err := mgr.LookupProjectRDD(project); err != nil || mode != "" || found == nil {
		t.Fatalf("sin preferencia: %q %v %v", mode, found, err)
	}
	if _, err := mgr.SetProjectRDD(rec.ID, "on"); err != nil {
		t.Fatal(err)
	}
	if mode, _, err := mgr.LookupProjectRDD(project); err != nil || mode != "on" {
		t.Fatalf("on: %q %v", mode, err)
	}
	// Re-registrar el proyecto conserva la preferencia.
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if mode, _, _ := mgr.LookupProjectRDD(project); mode != "on" {
		t.Errorf("re-registrar no debe perder la preferencia: %q", mode)
	}
	if _, err := mgr.SetProjectRDD(project, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(mgr.GetConfigPath())
	if strings.Contains(string(raw), "rdd_mode") {
		t.Errorf("inherit retira el campo del JSON:\n%s", raw)
	}
}

func TestProjectRDDRejectsInvalidAndUnregistered(t *testing.T) {
	mgr := newRDDManager(t)
	project := t.TempDir()
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.SetProjectRDD("p", "quizás"); err == nil || !strings.Contains(err.Error(), "no es válido") {
		t.Errorf("valor inválido: %v", err)
	}
	if _, err := mgr.SetProjectRDD("no-existe", "on"); err == nil || !strings.Contains(err.Error(), "no está registrado") {
		t.Errorf("proyecto sin registrar: %v", err)
	}
	if mode, found, err := mgr.LookupProjectRDD(t.TempDir()); err != nil || mode != "" || found != nil {
		t.Errorf("repo fuera de todo proyecto: %q %v %v", mode, found, err)
	}
}

// En multirepo los repositorios de los roles y el de especificaciones cuelgan de la
// carpeta maestra, que es la ruta del proyecto registrado: todos resuelven a su entrada.
// Con proyectos anidados gana el más profundo.
func TestProjectForRepoResolvesRoleAndSpecsReposToTheMasterRecord(t *testing.T) {
	mgr := newRDDManager(t)
	master := t.TempDir()
	if _, err := NewInitializer(mgr, nil).Init(InitOptions{
		Path: master, Name: "Multi", Topology: "multirepo", SpecsRepository: "repo-specs",
		Roles: []RoleInput{{Key: "api", Name: "API", Repositories: []string{"repo-api"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.SetProjectRDD(master, "on"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(master, "repo-api", "src", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, repo := range []string{
		master,
		filepath.Join(master, "repo-specs"),
		filepath.Join(master, "repo-api"),
		filepath.Join(master, "repo-api", "src", "pkg"),
	} {
		if mode, record, err := mgr.LookupProjectRDD(repo); err != nil || mode != "on" || record == nil || record.Name != "Multi" {
			t.Errorf("%s: mode=%q record=%v err=%v", repo, mode, record, err)
		}
	}

	nested := filepath.Join(master, "repo-api")
	if _, err := mgr.Register(nested, "Anidado", ""); err != nil {
		t.Fatal(err)
	}
	if mode, record, _ := mgr.LookupProjectRDD(filepath.Join(nested, "src")); record == nil || record.Name != "Anidado" || mode != "" {
		t.Errorf("gana el proyecto más profundo: mode=%q record=%v", mode, record)
	}
}

func TestProjectRDDUnreadableValueNamesTheFile(t *testing.T) {
	mgr := newRDDManager(t)
	project := t.TempDir()
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(mgr.GetConfigPath())
	broken := strings.Replace(string(raw), `"is_configured"`, `"rdd_mode": "quizás", "is_configured"`, 1)
	if err := os.WriteFile(mgr.GetConfigPath(), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.LookupProjectRDD(project); err == nil || !strings.Contains(err.Error(), mgr.GetConfigPath()) {
		t.Errorf("debe nombrar el fichero: %v", err)
	}
}
