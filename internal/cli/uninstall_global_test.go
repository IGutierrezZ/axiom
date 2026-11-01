package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

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

// globalLeftoverNames son los rótulos con que el doctor informa de los restos de Kilo,
// Kiro y Codex en HOME.
var globalLeftoverNames = []string{"Kilo", "Kiro", "Codex"}

// footprintLeftovers devuelve las entradas «resto global» del doctor para HOME.
func footprintLeftovers(home string) []footprintEntry {
	var leftovers []footprintEntry
	for _, entry := range collectMachineFootprint(home, state.InstallState{}) {
		if entry.Kind == footprintGlobalLeftover {
			leftovers = append(leftovers, entry)
		}
	}
	return leftovers
}

// install global de Kilo, Kiro, Codex y OpenCode + uninstall global: no queda ninguno de
// sus restos en HOME (opencode.json y plugins de Kilo, steering de Kiro, context7,
// multi_agent y agents.* de Codex), el contenido que el usuario tenía en esos ficheros
// sigue intacto y el doctor informa de cada resto con su agente antes y de ninguno
// después.
func TestUninstallGlobalRetiresKiloKiroAndCodexLeftoversKeepingTheUsersContent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	home, _, _ := footprintEnv(t)

	kiloConfig := filepath.Join(home, ".config", "kilo", "opencode.json")
	kiloUserPlugin := filepath.Join(home, ".config", "kilo", "plugins", "mi-plugin.ts")
	kiroSteering := filepath.Join(home, ".kiro", "steering", "axiom.md")
	kiroUserSteering := filepath.Join(home, ".kiro", "steering", "mi-guia.md")
	kiroMCP := filepath.Join(home, ".kiro", "settings", "mcp.json")
	codexConfig := filepath.Join(home, ".codex", "config.toml")
	openCodeConfig := filepath.Join(home, ".config", "opencode", "opencode.json")

	const userKilo = `{"theme":"mi-tema","agent":{"mi-agente":{"mode":"subagent","description":"del usuario"}},"mcp":{"mio":{"type":"local","command":["node","s.js"]}}}`
	const userKiroMCP = `{"mcpServers":{"mio":{"command":"node","args":["s.js"]}}}`
	const userCodex = "model = \"modelo-del-usuario\"\n\n[mcp_servers.mio]\ncommand = \"node\"\nargs = [\"s.js\"]\n"
	const userOpenCode = `{"theme":"mi-tema-opencode","permission":{"bash":{"npm publish":"deny"}}}`
	mustWriteFile(t, kiloConfig, []byte(userKilo))
	mustWriteFile(t, kiloUserPlugin, []byte("// plugin del usuario\n"))
	mustWriteFile(t, kiroUserSteering, []byte("Guía propia del usuario.\n"))
	mustWriteFile(t, kiroMCP, []byte(userKiroMCP))
	mustWriteFile(t, codexConfig, []byte(userCodex))
	mustWriteFile(t, openCodeConfig, []byte(userOpenCode))

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "kilocode,kiro-ide,codex,opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	// Antes de desinstalar: cada agente figura con su nombre y los de OpenCode no se
	// mezclan con los de Kilo.
	before := footprintLeftovers(home)
	for _, agent := range globalLeftoverNames {
		found := false
		for _, entry := range before {
			found = found || strings.Contains(entry.Name, agent)
		}
		if !found {
			t.Errorf("el doctor debe informar de un resto de %s tras instalar: %+v", agent, before)
		}
	}
	for _, entry := range before {
		kiloPath := strings.Contains(entry.Detail, filepath.Join(".config", "kilo"))
		openCodePath := strings.Contains(entry.Detail, filepath.Join(".config", "opencode"))
		if strings.Contains(entry.Name, "OpenCode") && kiloPath {
			t.Errorf("%q no debe listar rutas de Kilo: %s", entry.Name, entry.Detail)
		}
		if strings.Contains(entry.Name, "Kilo") && openCodePath {
			t.Errorf("%q no debe listar rutas de OpenCode: %s", entry.Name, entry.Detail)
		}
	}
	if !strings.Contains(strings.Join(entryNames(before), "|"), "plugins de OpenCode en HOME") {
		t.Errorf("los plugins de OpenCode deben seguir informándose como de OpenCode: %+v", before)
	}

	runWorkspaceUninstall(t, "--all", "--scope", "global")

	// Restos retirados.
	for _, gone := range []string{
		kiroSteering,
		filepath.Join(home, ".config", "kilo", "plugins", "skill-registry.ts"),
		filepath.Join(home, ".config", "kilo", "plugins", "model-variants.ts"),
	} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s debe retirarse en la desinstalación global (stat = %v)", gone, err)
		}
	}

	// El contenido del usuario queda como estaba.
	for path, want := range map[string]string{
		kiloConfig:     userKilo,
		kiroMCP:        userKiroMCP,
		openCodeConfig: userOpenCode,
	} {
		var wantValue any
		if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
			t.Fatal(err)
		}
		if got := parsedWorkspaceFile(t, path); !reflect.DeepEqual(got, wantValue) {
			t.Errorf("%s no quedó como el del usuario:\nahora %v\nantes %s", path, got, want)
		}
	}
	var wantCodex map[string]any
	if _, err := toml.Decode(userCodex, &wantCodex); err != nil {
		t.Fatal(err)
	}
	if got := parsedWorkspaceFile(t, codexConfig); !reflect.DeepEqual(got, any(wantCodex)) {
		t.Errorf("config.toml de Codex no quedó como el del usuario:\nahora %v\nantes %v", got, wantCodex)
	}
	if got, err := os.ReadFile(kiloUserPlugin); err != nil || string(got) != "// plugin del usuario\n" {
		t.Errorf("el plugin del usuario de Kilo debe seguir intacto: %v %q", err, got)
	}
	if got, err := os.ReadFile(kiroUserSteering); err != nil || string(got) != "Guía propia del usuario.\n" {
		t.Errorf("el steering del usuario de Kiro debe seguir intacto: %v %q", err, got)
	}

	// El doctor no informa de ningún resto.
	if leftovers := footprintLeftovers(home); len(leftovers) != 0 {
		t.Errorf("tras desinstalar el doctor no debe informar de restos: %+v", leftovers)
		t.Logf("HOME tras la desinstalación: %v", homeSnapshotWithoutState(t, home))
	}
}

func entryNames(entries []footprintEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name
	}
	return names
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
