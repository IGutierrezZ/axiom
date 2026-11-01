package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// install global + uninstall global: HOME queda limpio de lo de Axiom, el
// settings.json del usuario conserva su deny y su modo, las entradas de PATH
// gestionadas desaparecen y las ajenas se quedan, y el apartado «Huella de máquina»
// del doctor no lista restos.
func TestUninstallGlobalLeavesHomeCleanAndKeepsTheUsersConfig(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "estado")
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	home, _, _ := footprintEnv(t)

	userSettings := `{"permissions":{"defaultMode":"acceptEdits","deny":["WebFetch","Bash(curl *)"]},"theme":"dark"}`
	mustWriteFile(t, filepath.Join(home, ".claude", "settings.json"), []byte(userSettings))
	mustWriteFile(t, filepath.Join(home, ".agents", "skills", "mi-skill", "SKILL.md"), []byte("Skill propia.\n"))
	mustWriteFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"), []byte(`{"theme":"mi-tema","permission":{"bash":{"npm publish":"deny"}}}`))

	foreign := filepath.Join(t.TempDir(), "herramientas")
	t.Setenv("PATH", foreign+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "claude-code,opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	launchers := system.BinDir(home)
	if !strings.Contains(os.Getenv("PATH"), launchers) {
		// El lanzador de OpenCode solo se añade al PATH si se activa en segundo plano:
		// la prueba necesita que la entrada exista.
		t.Setenv("PATH", launchers+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	result := runWorkspaceUninstall(t, "--all", "--scope", "global")

	// settings.json de Claude: solo lo del usuario.
	var gotSettings, wantSettings any
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("el settings.json del usuario debe seguir existiendo: %v", err)
	}
	if err := json.Unmarshal(raw, &gotSettings); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(userSettings), &wantSettings); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotSettings, wantSettings) {
		t.Errorf("settings.json no quedó como el del usuario:\nahora %s\nantes %s", raw, userSettings)
	}
	if got, err := os.ReadFile(filepath.Join(home, ".agents", "skills", "mi-skill", "SKILL.md")); err != nil || string(got) != "Skill propia.\n" {
		t.Errorf("la skill del usuario debe seguir intacta: %v %q", err, got)
	}
	var gotOpenCode, wantOpenCode any
	raw, err = os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.json"))
	if err != nil {
		t.Fatalf("el opencode.json del usuario debe seguir existiendo: %v", err)
	}
	if err := json.Unmarshal(raw, &gotOpenCode); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"theme":"mi-tema","permission":{"bash":{"npm publish":"deny"}}}`), &wantOpenCode); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotOpenCode, wantOpenCode) {
		t.Errorf("opencode.json no quedó como el del usuario:\n%s", raw)
	}

	// PATH: las entradas gestionadas se van; las ajenas se quedan.
	pathEntries := filepath.SplitList(os.Getenv("PATH"))
	if !slices.Contains(pathEntries, foreign) {
		t.Errorf("la entrada ajena %s debe seguir en el PATH", foreign)
	}
	if slices.Contains(pathEntries, launchers) {
		t.Errorf("la carpeta de lanzadores %s debe salir del PATH", launchers)
	}
	if !slices.Contains(result.RemovedPathEntries, launchers) {
		t.Errorf("RemovedPathEntries = %v, want %s", result.RemovedPathEntries, launchers)
	}

	// La raíz de estado de máquina no se purga sin --purge-state.
	if _, err := os.Stat(stateDir); err != nil {
		t.Errorf("la raíz de estado debe seguir existiendo sin --purge-state: %v", err)
	}

	// El doctor no debe encontrar restos de Axiom en HOME.
	leftovers := 0
	for _, entry := range collectMachineFootprint(home, state.InstallState{}) {
		if entry.Kind == footprintGlobalLeftover {
			leftovers++
			t.Errorf("resto global tras desinstalar: %s %s", entry.Name, entry.Detail)
		}
	}
	if leftovers > 0 {
		t.Logf("HOME tras la desinstalación: %v", homeSnapshotWithoutState(t, home))
	}
}

// --purge-state borra la raíz de estado; sin proyectos registrados en el hub, entera.
func TestUninstallGlobalPurgeStateRemovesTheStateRoot(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "estado")
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	home, _, _ := footprintEnv(t)
	if _, err := RunInstall([]string{"--scope", "global", "--agent", "claude-code"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if _, err := os.Stat(state.Path(home)); err != nil {
		t.Fatalf("la instalación global debe escribir el estado global: %v", err)
	}

	result := runWorkspaceUninstall(t, "--all", "--scope", "global", "--purge-state")

	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Errorf("--purge-state debe borrar la raíz de estado; stat = %v (purgado: %v)", err, result.PurgedPaths)
	}
}

func TestParseUninstallFlagsScopeAndPurge(t *testing.T) {
	t.Run("ámbito", func(t *testing.T) {
		flags, err := ParseUninstallFlags([]string{"--all", "--scope", "global"})
		if err != nil || flags.Scope != "global" {
			t.Fatalf("flags=%+v err=%v", flags, err)
		}
		if _, err := ParseUninstallFlags([]string{"--all", "--scope", "usuario"}); err == nil {
			t.Fatal("un ámbito desconocido debe rechazarse")
		}
	})
	t.Run("purge solo con --all", func(t *testing.T) {
		if _, err := ParseUninstallFlags([]string{"--agent", "claude-code", "--scope", "global", "--purge-state"}); err == nil {
			t.Fatal("--purge-state sin --all debe rechazarse")
		}
		flags, err := ParseUninstallFlags([]string{"--all", "--scope", "global", "--purge-state", "--purge-hub"})
		if err != nil || !flags.PurgeState || !flags.PurgeHub {
			t.Fatalf("flags=%+v err=%v", flags, err)
		}
		if _, err := ParseUninstallFlags([]string{"--all", "--purge-hub"}); err == nil {
			t.Fatal("--purge-hub sin --purge-state debe rechazarse")
		}
	})
}

// Con el ámbito workspace las purgas no se aplican: el estado de máquina nunca se toca
// desde un proyecto.
func TestUninstallOptionsIgnorePurgeInWorkspaceScope(t *testing.T) {
	opts := uninstallOptions(ScopeWorkspace, true, true)
	if opts.PurgeState || opts.PurgeHub {
		t.Fatalf("opts = %+v: el ámbito workspace no purga el estado de máquina", opts)
	}
	global := uninstallOptions(ScopeGlobal, true, false)
	if !global.PurgeState || global.PurgeHub {
		t.Fatalf("opts = %+v", global)
	}
}
