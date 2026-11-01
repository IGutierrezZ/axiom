package uninstall

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// fakeUserPath sustituye las operaciones del PATH de usuario por una lista en
// memoria y devuelve un puntero a ella. Restaura los seams al terminar la prueba.
func fakeUserPath(t *testing.T, goos string, entries ...string) *[]string {
	t.Helper()
	current := slices.Clone(entries)
	restoreEntries, restoreRemove, restoreGOOS := userPathEntries, removeFromUserPath, pathGOOS
	t.Cleanup(func() { userPathEntries, removeFromUserPath, pathGOOS = restoreEntries, restoreRemove, restoreGOOS })
	pathGOOS = goos
	userPathEntries = func(string) ([]string, error) { return slices.Clone(current), nil }
	removeFromUserPath = func(dir string) error {
		for index, entry := range current {
			if strings.EqualFold(filepath.Clean(entry), filepath.Clean(dir)) {
				current = append(current[:index], current[index+1:]...)
				return nil
			}
		}
		return nil
	}
	return &current
}

func newGlobalService(t *testing.T, home string, opts Options) *Service {
	t.Helper()
	svc, err := NewScopedService(home, t.TempDir(), "dev", opts)
	if err != nil {
		t.Fatalf("NewScopedService() error = %v", err)
	}
	return svc
}

// El PATH solo pierde las entradas gestionadas; las ajenas se quedan y en su orden.
func TestCompleteUninstallRemovesOnlyTheManagedPathEntries(t *testing.T) {
	home := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "estado")
	localAppData := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	t.Setenv("LOCALAPPDATA", localAppData)

	engramDir := filepath.Join(localAppData, "engram", "bin")
	launchersDir := system.BinDir(home)
	ggaBin := filepath.Join(home, "bin")
	foreign := []string{filepath.Join(t.TempDir(), "herramientas"), filepath.Join(t.TempDir(), "otras")}
	// GGA lo gestiona Axiom: sus shims están en HOME.
	if err := os.MkdirAll(ggaBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ggaBin, "gga.cmd"), []byte("@echo off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := fakeUserPath(t, "windows", foreign[0], engramDir, ggaBin, launchersDir, foreign[1])

	result, err := newGlobalService(t, home, Options{}).CompleteUninstall()
	if err != nil {
		t.Fatalf("CompleteUninstall() error = %v", err)
	}

	if !slices.Equal(*entries, foreign) {
		t.Fatalf("PATH = %v, want solo las entradas ajenas %v", *entries, foreign)
	}
	wantRemoved := []string{engramDir, ggaBin, launchersDir}
	if !slices.Equal(result.RemovedPathEntries, wantRemoved) {
		t.Fatalf("RemovedPathEntries = %v, want %v", result.RemovedPathEntries, wantRemoved)
	}
}

// Sin GGA de Axiom en HOME, ~/bin no es de Axiom y su entrada del PATH se queda; en
// plataformas que no persisten PATH, la carpeta de Engram (compartida) tampoco se toca.
func TestCompleteUninstallKeepsPathEntriesAxiomDidNotAdd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	t.Setenv("LOCALAPPDATA", t.TempDir())
	userBin := filepath.Join(home, "bin")
	usrLocal := filepath.Join(t.TempDir(), "usr-local-bin")

	t.Run("sin GGA de Axiom", func(t *testing.T) {
		entries := fakeUserPath(t, "windows", userBin, usrLocal)
		result, err := newGlobalService(t, home, Options{}).CompleteUninstall()
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(*entries, []string{userBin, usrLocal}) || len(result.RemovedPathEntries) != 0 {
			t.Fatalf("PATH = %v, removed = %v: no debe tocar ~/bin si GGA no era de Axiom", *entries, result.RemovedPathEntries)
		}
	})

	t.Run("en Linux no se quita ninguna carpeta de Engram", func(t *testing.T) {
		entries := fakeUserPath(t, "linux", usrLocal, filepath.Join(home, ".local", "bin"))
		if _, err := newGlobalService(t, home, Options{}).CompleteUninstall(); err != nil {
			t.Fatal(err)
		}
		if len(*entries) != 2 {
			t.Fatalf("PATH = %v: /usr/local/bin y ~/.local/bin son compartidas", *entries)
		}
	})
}

// El estado de máquina se conserva por defecto y solo --purge-state lo borra; el hub se
// conserva si lista proyectos salvo --purge-hub.
func TestCompleteUninstallPurgeStateAndHub(t *testing.T) {
	seed := func(t *testing.T, hubProjects int) (home, stateDir string) {
		t.Helper()
		home = t.TempDir()
		stateDir = filepath.Join(t.TempDir(), "estado")
		t.Setenv("AXIOM_STATE_DIR", stateDir)
		t.Setenv("LOCALAPPDATA", t.TempDir())
		fakeUserPath(t, "linux")
		hub := `{"version":"1.0","workspaces":[]}`
		if hubProjects > 0 {
			hub = `{"version":"1.0","workspaces":[{"id":"a","path":"/p/a"},{"id":"b","path":"/p/b"}]}`
		}
		for name, content := range map[string]string{
			"workspaces.json":                      hub,
			"workspaces.json.lock":                 "",
			"state.json":                           `{"installed_agents":[]}`,
			filepath.Join("cache", "x.json"):       "{}",
			filepath.Join("bin", "launcher.cmd"):   "x",
			filepath.Join("backups", "viejo", "m"): "x",
		} {
			path := filepath.Join(stateDir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return home, stateDir
	}

	t.Run("sin el flag no se toca", func(t *testing.T) {
		home, stateDir := seed(t, 2)
		result, err := newGlobalService(t, home, Options{}).CompleteUninstall()
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"workspaces.json", "state.json", filepath.Join("cache", "x.json"), filepath.Join("backups", "viejo", "m")} {
			if _, err := os.Stat(filepath.Join(stateDir, name)); err != nil {
				t.Errorf("%s debe seguir existiendo sin --purge-state: %v", name, err)
			}
		}
		if len(result.PurgedPaths) != 0 {
			t.Errorf("PurgedPaths = %v, want vacío", result.PurgedPaths)
		}
	})

	t.Run("purge-state conserva el hub con proyectos", func(t *testing.T) {
		home, stateDir := seed(t, 2)
		result, err := newGlobalService(t, home, Options{PurgeState: true}).CompleteUninstall()
		if err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(stateDir)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		if !slices.Equal(names, []string{"workspaces.json", "workspaces.json.lock"}) {
			t.Fatalf("la raíz de estado debe quedarse solo con el hub, tiene %v", names)
		}
		if !strings.Contains(strings.Join(result.ManualActions, "\n"), "--purge-hub") {
			t.Errorf("falta el aviso de que se conserva el hub: %v", result.ManualActions)
		}
	})

	t.Run("purge-state con purge-hub borra todo", func(t *testing.T) {
		home, stateDir := seed(t, 2)
		if _, err := newGlobalService(t, home, Options{PurgeState: true, PurgeHub: true}).CompleteUninstall(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
			t.Fatalf("la raíz de estado debe desaparecer entera; stat = %v", err)
		}
	})

	t.Run("purge-state sin proyectos en el hub borra todo", func(t *testing.T) {
		home, stateDir := seed(t, 0)
		if _, err := newGlobalService(t, home, Options{PurgeState: true}).CompleteUninstall(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
			t.Fatalf("sin proyectos registrados el hub se purga con el resto; stat = %v", err)
		}
	})
}

// Una AXIOM_STATE_DIR que contiene HOME o es la raíz no se purga nunca.
func TestPurgeStateRefusesUnsafeStateDirs(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(home, "documento-del-usuario.txt")
	if err := os.WriteFile(marker, []byte("importante"), 0o644); err != nil {
		t.Fatal(err)
	}
	// El servicio se crea con una raíz de estado segura; la peligrosa solo se fija
	// al llamar a la purga, que es la que lee system.AxiomDir. Nunca se ejecuta una
	// desinstalación entera con ellas: crearía copias de seguridad en esas carpetas.
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	svc, err := NewScopedService(home, t.TempDir(), "dev", Options{PurgeState: true})
	if err != nil {
		t.Fatal(err)
	}

	for name, stateDir := range map[string]string{
		"HOME":             home,
		"padre de HOME":    filepath.Dir(home),
		"raíz del sistema": filepath.VolumeName(home) + string(filepath.Separator),
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("AXIOM_STATE_DIR", stateDir)
			var result Result
			svc.purgeStateDir(&result)
			if _, statErr := os.Stat(marker); statErr != nil {
				t.Fatalf("el fichero del usuario debe seguir existiendo: %v", statErr)
			}
			if len(result.PurgedPaths) != 0 {
				t.Fatalf("PurgedPaths = %v, no se debe purgar nada", result.PurgedPaths)
			}
			if !strings.Contains(strings.Join(result.ManualActions, "\n"), "No se purga") {
				t.Fatalf("falta el aviso de que no se purga: %v", result.ManualActions)
			}
		})
	}
}

// Las tres funciones del Claude global: el deny y el modo del usuario sobreviven a la
// desinstalación global, y la configuración propia de Axiom desaparece.
func TestGlobalUninstallKeepsTheUsersClaudeSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	t.Setenv("LOCALAPPDATA", t.TempDir())
	fakeUserPath(t, "linux")
	settings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	deny := ""
	for _, rule := range axiomClaudeDenyRules(t) {
		deny += `,"` + rule + `"`
	}
	seed := `{"permissions":{"defaultMode":"acceptEdits","deny":["WebFetch"` + deny + `]},"outputStyle":"Axiom"}`
	if err := os.WriteFile(settings, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := newGlobalService(t, home, Options{}).PartialUninstall([]model.AgentID{model.AgentClaudeCode}, nil); err != nil {
		t.Fatalf("PartialUninstall() error = %v", err)
	}

	got := readJSONFileForTest(t, settings)
	if _, exists := got["outputStyle"]; exists {
		t.Errorf("outputStyle de Axiom debe retirarse: %#v", got)
	}
	permissionsObject := got["permissions"].(map[string]any)
	if permissionsObject["defaultMode"] != "acceptEdits" {
		t.Errorf("defaultMode = %v, want el del usuario", permissionsObject["defaultMode"])
	}
	if gotDeny := stringsAt(t, got, "permissions", "deny"); !slices.Equal(gotDeny, []string{"WebFetch"}) {
		t.Errorf("deny = %v, want solo la entrada del usuario", gotDeny)
	}
}
