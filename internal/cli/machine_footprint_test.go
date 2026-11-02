package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kimi"
	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/installcmd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// footprintEnv aísla un HOME y un workspace temporales y fija los seams de
// instalación para que ningún comando real se ejecute. Devuelve ambas rutas y el
// registro de comandos lanzados.
func footprintEnv(t *testing.T) (home, workspace string, commands *commandRecorder) {
	t.Helper()
	home = t.TempDir()
	workspace = t.TempDir()
	// APPDATA fuera de HOME: el comando `go` que lanza la instalación escribe su
	// telemetría ahí y no es huella de Axiom.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", t.TempDir())

	restoreHome := osUserHomeDir
	restoreCommand := runCommand
	restoreLookPath := cmdLookPath
	restoreBackupHome := backup.UserHomeDirFn
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("no se pudo leer el directorio actual: %v", err)
	}
	t.Cleanup(func() {
		osUserHomeDir = restoreHome
		runCommand = restoreCommand
		cmdLookPath = restoreLookPath
		backup.UserHomeDirFn = restoreBackupHome
		if err := os.Chdir(originalCwd); err != nil {
			t.Errorf("no se pudo restaurar el directorio de trabajo: %v", err)
		}
	})
	commands = &commandRecorder{}
	osUserHomeDir = func() (string, error) { return home, nil }
	backup.UserHomeDirFn = func() (string, error) { return home, nil }
	runCommand = commands.record
	cmdLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("no se pudo cambiar al workspace temporal: %v", err)
	}
	return home, workspace, commands
}

// homeSnapshotWithoutState captura el HOME sin `~/.axiom`: las copias de
// seguridad y el estado de Axiom tienen su propia tarea (raíz única de estado) y
// no forman parte de lo que aquí se comprueba.
func homeSnapshotWithoutState(t *testing.T, home string) map[string]string {
	t.Helper()
	snapshot := permissionsTreeSnapshot(t, home)
	for path := range snapshot {
		if path == ".axiom" || strings.HasPrefix(path, ".axiom/") {
			delete(snapshot, path)
		}
	}
	return snapshot
}

func seedFootprintHome(t *testing.T, home string) {
	t.Helper()
	for path, content := range map[string]string{
		filepath.Join(home, ".agents", "skills", "go-testing", "SKILL.md"):         "copia del usuario\n",
		filepath.Join(home, ".config", "opencode", "plugins", "skill-registry.ts"): "// plugin antiguo\n",
		filepath.Join(home, ".config", "opencode", "plugins", "model-variants.ts"): "// plugin antiguo\n",
		filepath.Join(home, ".claude", "settings.json"):                            `{"permissions":{"deny":["WebFetch"]}}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// La instalación con --scope workspace y el preset por defecto no escribe nada en
// HOME ni lanza `engram setup`, y avisa del comando manual.
func TestRunInstallWorkspaceDefaultPresetLeavesHomeUntouched(t *testing.T) {
	home, workspace, commands := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code,opencode"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	for _, command := range commands.get() {
		if strings.Contains(command, "engram setup") {
			t.Errorf("con --scope workspace no debe lanzarse engram setup, se lanzó %q", command)
		}
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	if _, statErr := os.Stat(filepath.Join(workspace, ".claude")); statErr != nil {
		t.Errorf("la instalación debe escribir en el workspace: %v", statErr)
	}

	joined := strings.Join(result.Execution.ManualActions, "\n")
	for _, want := range []string{"engram setup", "claude-code", "opencode", "ya está configurado en el proyecto"} {
		if !strings.Contains(joined, want) {
			t.Errorf("el aviso de engram setup omitido no contiene %q:\n%s", want, joined)
		}
	}
	for _, unwanted := range []string{"GGA", "CodeGraph"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("el preset por defecto no elige %s, pero hay aviso:\n%s", unwanted, joined)
		}
	}
}

// Con el ámbito global todo sigue igual: se lanza engram setup y se refresca la
// copia de skills de compatibilidad que ya existe en ~/.agents/skills.
func TestRunInstallGlobalStillRunsEngramSetupAndRefreshesCompatibilitySkills(t *testing.T) {
	home, _, commands := footprintEnv(t)
	seedFootprintHome(t, home)
	compat := filepath.Join(home, ".agents")
	before := permissionsTreeSnapshot(t, compat)

	result, err := RunInstall([]string{"--scope", "global", "--agent", "claude-code", "--component", "engram", "--component", "sdd"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	found := false
	for _, command := range commands.get() {
		if strings.Contains(command, "engram setup claude-code") {
			found = true
		}
	}
	if !found {
		t.Errorf("con --scope global debe lanzarse engram setup claude-code, comandos: %v", commands.get())
	}
	if after := permissionsTreeSnapshot(t, compat); len(after) <= len(before) {
		t.Errorf("con --scope global debe refrescarse ~/.agents/skills: antes %d ficheros, después %d", len(before), len(after))
	}
	if joined := strings.Join(result.Execution.ManualActions, "\n"); strings.Contains(joined, "engram setup") {
		t.Errorf("con --scope global no hay aviso de engram setup omitido:\n%s", joined)
	}
}

// Sincronizar con --scope workspace no toca la copia de compatibilidad, no refresca
// los plugins de OpenCode de HOME y no limpia skills retiradas en HOME.
func TestRunSyncWorkspaceLeavesHomeUntouched(t *testing.T) {
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	var cleanedRoots []string
	restoreRetire := retireInstalledSkills
	retireInstalledSkills = func(root string) skills.RetiredCopies {
		cleanedRoots = append(cleanedRoots, root)
		return restoreRetire(root)
	}
	t.Cleanup(func() { retireInstalledSkills = restoreRetire })

	// Sin SDD, con OpenCode: es el caso en que sync refresca los plugins ya
	// instalados (incidencia 1440).
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode, model.AgentOpenCode},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentSkills},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("RunSyncWithSelectionScoped() error = %v", err)
	}

	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	for _, root := range cleanedRoots {
		if strings.HasPrefix(filepath.Clean(root), filepath.Clean(home)+string(filepath.Separator)) {
			t.Errorf("sync workspace no debe limpiar skills retiradas en HOME: %s", root)
		}
	}
	if len(cleanedRoots) == 0 {
		t.Errorf("sync workspace sí debe limpiar las skills retiradas del proyecto")
	}
}

// Con el ámbito global la limpieza de skills retiradas sigue incluyendo HOME.
func TestRetiredSkillRootsScope(t *testing.T) {
	home, workspace := t.TempDir(), t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	homeRoot := filepath.Clean(adapter.SkillsDir(home))
	workspaceRoot := filepath.Clean(adapter.SkillsDir(workspace))

	global := retiredSkillRoots(home, workspace, ScopeGlobal, []agents.Adapter{adapter})
	if len(global) != 1 || global[0] != homeRoot {
		t.Errorf("global debe limpiar solo HOME, got %v", global)
	}
	scoped := retiredSkillRoots(home, workspace, ScopeWorkspace, []agents.Adapter{adapter})
	if len(scoped) != 1 || scoped[0] != workspaceRoot {
		t.Errorf("workspace debe limpiar solo el proyecto, got %v", scoped)
	}
}

// El hub de prompts de Kimi se crea en el proyecto con el ámbito workspace.
func TestRunInstallWorkspaceKimiHubStaysInWorkspace(t *testing.T) {
	home, workspace, _ := footprintEnv(t)
	cmdLookPath = missingBinaryLookPath
	restoreInstallcmdLookPath := installcmd.OverrideLookPath(func(name string) (string, error) {
		if name == "uv" {
			return "/usr/bin/uv", nil
		}
		return "", exec.ErrNotFound
	})
	t.Cleanup(restoreInstallcmdLookPath)
	// Kimi se da por instalado: la prueba va del hub de prompts, no de la detección.
	restoreKimiLookPath := kimi.LookPathOverride
	kimi.LookPathOverride = func(string) (string, error) { return "/usr/local/bin/kimi", nil }
	t.Cleanup(func() { kimi.LookPathOverride = restoreKimiLookPath })
	allowUnselectableAgentsForTest(t)
	before := homeSnapshotWithoutState(t, home)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "kimi", "--component", "permissions"}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	if _, err := os.Stat(filepath.Join(workspace, ".kimi", "KIMI.md")); err != nil {
		t.Errorf("el hub de Kimi debe crearse en el workspace: %v", err)
	}
}

func TestEngramSetupSkippedNote(t *testing.T) {
	agentsList := []model.AgentID{model.AgentClaudeCode, model.AgentOpenCode, model.AgentCursor}
	withEngram := []model.ComponentID{model.ComponentEngram}

	got := engramSetupSkippedNote(ScopeWorkspace, withEngram, agentsList)
	if len(got) != 1 {
		t.Fatalf("workspace con Engram debe dar un aviso, got %v", got)
	}
	for _, want := range []string{"engram setup claude-code", "engram setup opencode", "carpeta de usuario"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("el aviso no contiene %q: %s", want, got[0])
		}
	}
	if strings.Contains(got[0], "cursor") {
		t.Errorf("Cursor no usa engram setup y no debe aparecer: %s", got[0])
	}

	if got := engramSetupSkippedNote(ScopeGlobal, withEngram, agentsList); got != nil {
		t.Errorf("con ámbito global no hay aviso, got %v", got)
	}
	if got := engramSetupSkippedNote(ScopeWorkspace, []model.ComponentID{model.ComponentSDD}, agentsList); got != nil {
		t.Errorf("sin Engram no hay aviso, got %v", got)
	}
	if got := engramSetupSkippedNote(ScopeWorkspace, withEngram, []model.AgentID{model.AgentCursor}); got != nil {
		t.Errorf("sin agentes que usen engram setup no hay aviso, got %v", got)
	}
	t.Setenv("AXIOM_ENGRAM_SETUP_MODE", "off")
	if got := engramSetupSkippedNote(ScopeWorkspace, withEngram, agentsList); got != nil {
		t.Errorf("con el modo de setup apagado no hay nada que avisar, got %v", got)
	}
}

func TestOptInFootprintNotes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	// GGA elegido de forma explícita: rutas de HOME y, en Windows, el PATH.
	notes := optInFootprintNotes(ScopeWorkspace, home, []model.ComponentID{model.ComponentGGA}, nil, model.Selection{})
	if len(notes) != 1 {
		t.Fatalf("GGA explícito debe dar un aviso, got %v", notes)
	}
	for _, path := range gga.FootprintPaths(home, runtime.GOOS) {
		if !strings.Contains(notes[0], path) {
			t.Errorf("el aviso de GGA no lista %s:\n%s", path, notes[0])
		}
	}
	if hasPathChange := strings.Contains(notes[0], "PATH de usuario"); hasPathChange != (runtime.GOOS == "windows") {
		t.Errorf("la modificación del PATH solo se declara en Windows (GOOS=%s):\n%s", runtime.GOOS, notes[0])
	}

	// CodeGraph no admite ubicación de proyecto en Axiom: se declara igual.
	notes = optInFootprintNotes(ScopeWorkspace, home, nil, nil, model.Selection{CommunityTools: []model.CommunityToolID{model.CommunityToolCodeGraph}})
	if len(notes) != 1 || !strings.Contains(notes[0], "CodeGraph") || !strings.Contains(notes[0], "paquete global") {
		t.Errorf("CodeGraph explícito debe declarar su huella global, got %v", notes)
	}

	// Plugins de OpenCode elegidos: rutas de la configuración de usuario, y solo
	// si OpenCode está entre los agentes.
	selection := model.Selection{OpenCodePlugins: []model.OpenCodeCommunityPluginID{model.OpenCodePluginGentleLogo}}
	if got := optInFootprintNotes(ScopeWorkspace, home, nil, []model.AgentID{model.AgentClaudeCode}, selection); len(got) != 0 {
		t.Errorf("sin OpenCode no se instalan plugins, got %v", got)
	}
	notes = optInFootprintNotes(ScopeWorkspace, home, nil, []model.AgentID{model.AgentOpenCode}, selection)
	if len(notes) != 1 || !strings.Contains(notes[0], filepath.Join(home, ".config", "opencode")) {
		t.Errorf("los plugins de OpenCode deben declarar rutas de HOME, got %v", notes)
	}

	// Sin elecciones explícitas, o con ámbito global, no hay nada que declarar.
	if got := optInFootprintNotes(ScopeWorkspace, home, []model.ComponentID{model.ComponentSDD}, []model.AgentID{model.AgentOpenCode}, model.Selection{}); len(got) != 0 {
		t.Errorf("sin opt-in no hay avisos, got %v", got)
	}
	if got := optInFootprintNotes(ScopeGlobal, home, []model.ComponentID{model.ComponentGGA}, nil, model.Selection{}); len(got) != 0 {
		t.Errorf("con ámbito global no hay avisos, got %v", got)
	}
}

// Elegir GGA de forma explícita con --scope workspace sigue escribiendo en HOME
// (consentimiento expreso), pero la instalación lo declara con un aviso.
func TestRunInstallWorkspaceExplicitGGAEmitsFootprintNote(t *testing.T) {
	home, _, _ := footprintEnv(t)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode", "--component", "gga"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	joined := strings.Join(result.Execution.ManualActions, "\n")
	if !strings.Contains(joined, "GGA se ha elegido de forma explícita") || !strings.Contains(joined, filepath.Join(home, ".local", "share", "gga")) {
		t.Errorf("falta el aviso de huella de GGA con las rutas de HOME:\n%s", joined)
	}
}

func TestRunSyncWorkspaceExplicitGGAEmitsFootprintNote(t *testing.T) {
	home, workspace, _ := footprintEnv(t)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentOpenCode},
		Components: []model.ComponentID{model.ComponentGGA},
		Persona:    model.PersonaNeutral,
	}
	result, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("RunSyncWithSelectionScoped() error = %v", err)
	}
	if joined := strings.Join(result.Execution.ManualActions, "\n"); !strings.Contains(joined, "GGA se ha elegido de forma explícita") {
		t.Errorf("sync con GGA explícito debe declarar su huella:\n%s", joined)
	}
}
