package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// captureWorkspaceAgentsNotice sustituye el escritor del aviso de agentes omitidos
// y devuelve el búfer donde se acumula.
func captureWorkspaceAgentsNotice(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	restore := workspaceAgentsNoticeWriter
	workspaceAgentsNoticeWriter = &buffer
	t.Cleanup(func() { workspaceAgentsNoticeWriter = restore })
	return &buffer
}

// workspaceAgentsFixture monta el proyecto con git y el HOME aislados de projectStateFixture y,
// además, aísla APPDATA y XDG_CONFIG_HOME: la detección de GGA lee de ahí y un GGA real de
// la máquina haría que el sync lo escribiera en el HOME de la prueba.
func workspaceAgentsFixture(t *testing.T) (rootA, rootB, home string) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	return projectStateFixture(t)
}

// seedCodexHome siembra la configuración global de Codex en el HOME de la prueba,
// con lo que el descubrimiento por carpetas lo daría por instalado.
func seedCodexHome(t *testing.T, home string) {
	t.Helper()
	for path, content := range map[string]string{
		filepath.Join(home, ".codex", "config.toml"):           "model = \"del-usuario\"\n",
		filepath.Join(home, ".codex", "prompts", "usuario.md"): "prompt del usuario\n",
		filepath.Join(home, ".codex", "AGENTS.md"):             "instrucciones del usuario\n",
	} {
		mustWriteFile(t, path, []byte(content))
	}
}

// seedWindsurfHome siembra la configuración global de Windsurf, un agente sin modo
// workspace, en el HOME de la prueba: el descubrimiento por carpetas lo daría por
// instalado.
func seedWindsurfHome(t *testing.T, home string) {
	t.Helper()
	for path, content := range map[string]string{
		filepath.Join(home, ".codeium", "windsurf", "memories", "global_rules.md"):   "instrucciones del usuario\n",
		filepath.Join(home, ".codeium", "windsurf", "skills", "usuario", "SKILL.md"): "skill del usuario\n",
	} {
		mustWriteFile(t, path, []byte(content))
	}
}

// seedClaudeHome siembra la configuración global de Claude Code, para que el
// descubrimiento por carpetas encuentre también un agente con modo workspace.
func seedClaudeHome(t *testing.T, home string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(home, ".claude", "settings.json"), []byte(`{"permissions":{"deny":["WebFetch"]}}`))
}

// persistedAgents es el estado de proyecto que dejaría una instalación anterior con esos agentes.
func persistedAgents(agentIDs ...string) state.InstallState {
	return state.InstallState{
		InstalledAgents:     agentIDs,
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentEngram},
		Persona:             "neutral",
	}
}

// assertOmittedNotices comprueba que el aviso sale una vez por agente omitido y
// no por componente.
func assertOmittedNotices(t *testing.T, got string, omitted ...model.AgentID) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != len(omitted) {
		t.Fatalf("se esperaba un aviso por agente omitido (%d), hay %d:\n%s", len(omitted), len(lines), got)
	}
	for _, id := range omitted {
		want := string(id) + " omitido: no admite el ámbito workspace"
		if !strings.Contains(got, want) || !strings.Contains(got, "--scope global") {
			t.Errorf("falta el aviso de %s (%q):\n%s", id, want, got)
		}
	}
}

func TestSplitAgentsForScope(t *testing.T) {
	selected := []model.AgentID{model.AgentClaudeCode, model.AgentCodex, model.AgentWindsurf, model.AgentOpenCode, model.AgentVSCodeCopilot, model.AgentAntigravity}

	kept, omitted := splitAgentsForScope(ScopeWorkspace, selected)
	if want := []model.AgentID{model.AgentClaudeCode, model.AgentCodex, model.AgentOpenCode, model.AgentVSCodeCopilot}; !slices.Equal(kept, want) {
		t.Errorf("workspace conserva %v, want %v (Codex y VS Code ya tienen modo workspace)", kept, want)
	}
	if want := []model.AgentID{model.AgentWindsurf, model.AgentAntigravity}; !slices.Equal(omitted, want) {
		t.Errorf("workspace omite %v, want %v", omitted, want)
	}

	kept, omitted = splitAgentsForScope(ScopeGlobal, selected)
	if !slices.Equal(kept, selected) || omitted != nil {
		t.Errorf("global no filtra: kept=%v omitted=%v", kept, omitted)
	}

	if kept, omitted = splitAgentsForScope(ScopeWorkspace, nil); kept != nil || omitted != nil {
		t.Errorf("sin agentes no hay nada que filtrar: kept=%v omitted=%v", kept, omitted)
	}
}

// Ningún adaptador sin modo workspace puede resolver una ruta en HOME con el
// ámbito workspace: la salvaguarda queda confinada al proyecto.
func TestComponentInjectionDirScopedNeverResolvesHomeInWorkspace(t *testing.T) {
	home, workspace := t.TempDir(), t.TempDir()
	reg, err := agents.NewDefaultRegistry()
	if err != nil {
		t.Fatalf("registry error: %v", err)
	}
	for _, id := range []model.AgentID{model.AgentTrae, model.AgentWindsurf, model.AgentAntigravity} {
		adapter, ok := reg.Get(id)
		if !ok {
			t.Fatalf("falta el adaptador %s", id)
		}
		if got := componentInjectionDirScoped(home, workspace, ScopeWorkspace, adapter); got != workspace {
			t.Errorf("%s con ámbito workspace: got %q, want la raíz del proyecto %q", id, got, workspace)
		}
		if got := componentInjectionDirScoped(home, workspace, ScopeGlobal, adapter); got != home {
			t.Errorf("%s con ámbito global: got %q, want HOME %q", id, got, home)
		}
	}
}

// install --scope workspace omite los agentes sin modo workspace, también los
// elegidos con --agent, avisa una vez por agente y no escribe nada en HOME.
func TestRunInstallWorkspaceOmitsAgentsWithoutWorkspaceMode(t *testing.T) {
	home, workspace, _ := footprintEnv(t)
	t.Setenv("AXIOM_STATE_DIR", "")
	seedFootprintHome(t, home)
	seedWindsurfHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := homeSnapshotWithoutState(t, home)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code,antigravity,windsurf"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentAntigravity, model.AgentWindsurf)
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(result.Resolved.Agents, want) {
		t.Errorf("agentes del plan = %v, want %v", result.Resolved.Agents, want)
	}
	if _, statErr := os.Stat(filepath.Join(workspace, ".claude")); statErr != nil {
		t.Errorf("Claude Code sí admite workspace y debe instalarse en el proyecto: %v", statErr)
	}
	for _, unwanted := range []string{".windsurf", ".github", ".vscode", ".codeium"} {
		if _, statErr := os.Stat(filepath.Join(workspace, unwanted)); !os.IsNotExist(statErr) {
			t.Errorf("un agente omitido no debe crear %s en el proyecto; stat = %v", unwanted, statErr)
		}
	}
	persisted, readErr := state.ProjectStore(home, workspace).Read()
	if readErr != nil {
		t.Fatalf("leer el estado del proyecto: %v", readErr)
	}
	if want := []string{"claude-code"}; !slices.Equal(persisted.InstalledAgents, want) {
		t.Errorf("el proyecto registra %v, want solo %v", persisted.InstalledAgents, want)
	}
}

// Si ningún agente elegido admite el ámbito workspace, install termina con un
// error claro en lugar de instalar solo los componentes de máquina.
func TestRunInstallWorkspaceWithOnlyAgentsWithoutWorkspaceModeFails(t *testing.T) {
	home, workspace, commands := footprintEnv(t)
	t.Setenv("AXIOM_STATE_DIR", "")
	seedWindsurfHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "windsurf"}, macOSDetectionResult())
	if err == nil {
		t.Fatal("RunInstall() debe fallar cuando ningún agente admite workspace")
	}
	for _, want := range []string{"ningún agente", "workspace", "windsurf", "--scope global"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	if len(commands.get()) != 0 {
		t.Errorf("no debe lanzarse ningún comando, se lanzaron %v", commands.get())
	}
	if entries, _ := os.ReadDir(workspace); len(entries) != 0 {
		t.Errorf("el proyecto debe quedar vacío, contiene %v", entries)
	}
}

// Con --scope global Codex se instala en HOME como siempre y no hay aviso.
func TestRunInstallGlobalStillInstallsCodexInHome(t *testing.T) {
	home, _, _ := footprintEnv(t)
	t.Setenv("AXIOM_STATE_DIR", "")
	notices := captureWorkspaceAgentsNotice(t)

	result, err := RunInstall([]string{"--scope", "global", "--agent", "codex", "--component", "sdd"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if want := []model.AgentID{model.AgentCodex}; !slices.Equal(result.Resolved.Agents, want) {
		t.Errorf("agentes del plan = %v, want %v", result.Resolved.Agents, want)
	}
	if notices.Len() != 0 {
		t.Errorf("con --scope global no hay aviso de agentes omitidos:\n%s", notices)
	}
	if entries, err := os.ReadDir(filepath.Join(home, ".codex")); err != nil || len(entries) == 0 {
		t.Errorf("con --scope global Codex debe escribir en ~/.codex: entradas=%v err=%v", entries, err)
	}
}

// Sin estado de proyecto, sync --scope workspace descubre los agentes por las
// carpetas de HOME: Windsurf (sin modo workspace) se omite y HOME queda intacto.
func TestRunSyncWorkspaceWithoutProjectStateLeavesHomeUntouchedForDiscoveredWindsurf(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	if len(result.Agents) != 0 || !result.NoOp {
		t.Errorf("sin agentes con modo workspace el sync no hace nada: agentes=%v noop=%v", result.Agents, result.NoOp)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".windsurf")); !os.IsNotExist(statErr) {
		t.Errorf("Windsurf omitido no debe crear .windsurf en el proyecto; stat = %v", statErr)
	}
	if _, statErr := os.Stat(state.ProjectPath(root)); !os.IsNotExist(statErr) {
		t.Errorf("un sync que no instala nada no debe persistir Windsurf en el proyecto; stat = %v", statErr)
	}
}

// Con Windsurf y Claude Code descubiertos, solo Claude Code se sincroniza, en el proyecto.
func TestRunSyncWorkspaceDiscoveredWindsurfIsOmittedAndClaudeStillSyncs(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	seedClaudeHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := homeSnapshotWithoutState(t, home)

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude")); statErr != nil {
		t.Errorf("Claude Code debe sincronizarse en el proyecto: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".windsurf")); !os.IsNotExist(statErr) {
		t.Errorf("Windsurf omitido no debe crear .windsurf en el proyecto; stat = %v", statErr)
	}
}

// Un proyecto que ya tiene Windsurf persistido (de un sync anterior a esta corrección)
// no vuelve a escribir en HOME: el agente se omite también desde el estado.
func TestRunSyncWorkspaceWithPersistedWindsurfLeavesHomeUntouched(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	if err := state.ProjectStore(home, root).Write(persistedAgents("windsurf")); err != nil {
		t.Fatal(err)
	}
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	if len(result.Agents) != 0 || !result.NoOp {
		t.Errorf("sin agentes con modo workspace el sync no hace nada: agentes=%v noop=%v", result.Agents, result.NoOp)
	}
}

// El mismo estado persistido con un agente que sí admite workspace sincroniza solo
// ese agente.
func TestRunSyncWorkspaceWithPersistedWindsurfAndClaudeSyncsOnlyClaude(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	if err := state.ProjectStore(home, root).Write(persistedAgents("windsurf", "claude-code")); err != nil {
		t.Fatal(err)
	}
	notices := captureWorkspaceAgentsNotice(t)
	before := homeSnapshotWithoutState(t, home)

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
}

// Un agente elegido con --agent también se omite, igual que en install.
func TestRunSyncWorkspaceExplicitAgentWithoutWorkspaceModeIsOmitted(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	result, err := RunSync([]string{"--scope", "workspace", "--agents", "windsurf"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	assertOmittedNotices(t, notices.String(), model.AgentWindsurf)
	if len(result.Agents) != 0 || !result.NoOp {
		t.Errorf("agentes=%v noop=%v, se esperaba un sync vacío", result.Agents, result.NoOp)
	}
}

// La TUI y las llamadas programáticas no escriben por stderr: el agente omitido
// viaja como aviso del resultado y HOME queda intacto.
func TestRunSyncWithSelectionScopedWorkspaceOmitsWindsurfSilently(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedWindsurfHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentWindsurf},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentSDD, model.ComponentSkills},
		Persona:    model.PersonaNeutral,
	}
	result, err := RunSyncWithSelectionScoped(home, root, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("RunSyncWithSelectionScoped() error = %v", err)
	}

	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	if notices.Len() != 0 {
		t.Errorf("la ruta programática no escribe por stderr:\n%s", notices)
	}
	if want := []model.AgentID{model.AgentWindsurf}; !slices.Equal(result.OmittedAgents, want) {
		t.Errorf("OmittedAgents = %v, want %v", result.OmittedAgents, want)
	}
	if warnings := strings.Join(result.Warnings(), "\n"); !strings.Contains(warnings, "windsurf omitido") {
		t.Errorf("los avisos del resultado deben nombrar a Windsurf:\n%s", warnings)
	}
}

// Con --scope global, sync sigue escribiendo la configuración de Codex en HOME.
func TestRunSyncGlobalStillWritesCodexInHome(t *testing.T) {
	_, _, home := workspaceAgentsFixture(t)
	seedCodexHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	result, err := RunSync([]string{"--scope", "global", "--agents", "codex"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	if want := []model.AgentID{model.AgentCodex}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
	if notices.Len() != 0 || len(result.OmittedAgents) != 0 {
		t.Errorf("con --scope global no se omite ningún agente: aviso=%q omitidos=%v", notices, result.OmittedAgents)
	}
	after := permissionsTreeSnapshot(t, home)
	changed := false
	for path, content := range after {
		if strings.HasPrefix(path, ".codex/") && before[path] != content {
			changed = true
		}
	}
	if !changed {
		t.Errorf("con --scope global sync debe escribir en ~/.codex; antes %v, después %v", keysOf(before), keysOf(after))
	}
}

// Instalación sin agentes (TUI incluida) comparte el mismo mensaje de error.
func TestErrNoWorkspaceAgentsNamesEveryOmittedAgent(t *testing.T) {
	err := errNoWorkspaceAgents([]model.AgentID{model.AgentAntigravity, model.AgentWindsurf})
	for _, want := range []string{"antigravity", "windsurf", "--scope global"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
}
