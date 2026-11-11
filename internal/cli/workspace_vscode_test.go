package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// Las pruebas de este fichero cubren la distribución de proyecto de VS Code Copilot
// con --scope workspace. VS Code lee .github/instructions/*.instructions.md (con
// `applyTo: "**"` se aplican siempre), .agents/skills y .vscode/mcp.json (clave raíz
// `servers`, con Workspace Trust). No se usan .github/copilot-instructions.md ni el
// AGENTS.md raíz (del equipo), y el perfil de usuario de VS Code no se toca.

func vscodeSelection() model.Selection {
	return model.Selection{
		Agents:     []model.AgentID{model.AgentVSCodeCopilot},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentPersona, model.ComponentSDD, model.ComponentSkills, model.ComponentContext7},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
}

// vscodeUserEnv aísla HOME y APPDATA, siembra en ambos el perfil de usuario de VS Code
// (en Windows está bajo %APPDATA%\Code\User, y con un HOME de prueba bajo
// HOME\AppData\Roaming\Code\User) y devuelve una instantánea de los dos para compararlos
// después. La instalación de proyecto no debe cambiar ni un byte de ninguno.
func vscodeUserEnv(t *testing.T) (home, workspace, appData string, snapshot func() map[string]string) {
	t.Helper()
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ = footprintEnv(t)
	appData = t.TempDir()
	t.Setenv("APPDATA", appData)

	adapter := vscode.NewAdapter()
	userDirs := []string{filepath.Dir(adapter.SettingsPath(home)), filepath.Join(appData, "Code", "User")}
	for _, userDir := range userDirs {
		mustWriteFile(t, filepath.Join(userDir, "settings.json"), []byte("{\"editor.fontSize\": 14}\n"))
		mustWriteFile(t, filepath.Join(userDir, "mcp.json"), []byte("{\"servers\":{\"del-usuario\":{\"command\":\"usuario\"}}}\n"))
		mustWriteFile(t, filepath.Join(userDir, "prompts", "usuario.instructions.md"), []byte("---\napplyTo: \"**\"\n---\nInstrucciones personales.\n"))
	}
	mustWriteFile(t, filepath.Join(home, ".copilot", "skills", "usuario", "SKILL.md"), []byte("skill personal\n"))
	mustWriteFile(t, filepath.Join(home, ".github", "copilot-instructions.md"), []byte("instrucciones globales antiguas\n"))

	snapshot = func() map[string]string {
		all := map[string]string{}
		// De APPDATA solo se compara el perfil de VS Code: el comando go que lanza la
		// instalación escribe su telemetría en el resto y no es huella de Axiom.
		for prefix, root := range map[string]string{"home": home, "appdata": filepath.Join(appData, "Code")} {
			for path, content := range homeSnapshotWithoutState(t, root) {
				all[prefix+"/"+path] = content
			}
		}
		return all
	}
	return home, workspace, appData, snapshot
}

// seedVSCodeTeamFiles siembra los ficheros del equipo que Axiom no debe tocar y
// devuelve su contenido por ruta relativa.
func seedVSCodeTeamFiles(t *testing.T, workspace string) map[string][]byte {
	t.Helper()
	team := map[string][]byte{
		filepath.Join(".github", "copilot-instructions.md"): []byte("# Instrucciones del equipo\n\n## Personality\nSenior Architect (texto del equipo con las mismas palabras que la persona antigua).\n"),
		"AGENTS.md": []byte("# Reglas del equipo\n\nNo tocar.\n"),
		filepath.Join(".vscode", "settings.json"): []byte("{\n  // ajustes del equipo\n  \"editor.tabSize\": 2\n}\n"),
	}
	for rel, content := range team {
		mustWriteFile(t, filepath.Join(workspace, rel), content)
	}
	return team
}

func assertTeamFilesIntact(t *testing.T, workspace string, team map[string][]byte) {
	t.Helper()
	for rel, want := range team {
		if got, err := os.ReadFile(filepath.Join(workspace, rel)); err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s cambió o desapareció (err=%v):\n%s", rel, err, got)
		}
	}
}

// Las instrucciones van a .github/instructions con applyTo, las skills a .agents/skills
// y el MCP a .vscode/mcp.json (clave servers, fusionado con lo del usuario); los
// ficheros del equipo y el perfil de usuario de VS Code no cambian ni un byte.
func TestRunInstallWorkspaceVSCodeUsesProjectLayout(t *testing.T) {
	home, workspace, _, userSnapshot := vscodeUserEnv(t)
	team := seedVSCodeTeamFiles(t, workspace)
	userMCP := "{\n  \"inputs\": [{\"id\": \"token\", \"type\": \"promptString\", \"description\": \"Token\"}],\n  \"servers\": {\"propio\": {\"command\": \"propio\", \"args\": [\"--x\"]}}\n}\n"
	mustWriteFile(t, filepath.Join(workspace, ".vscode", "mcp.json"), []byte(userMCP))
	before := userSnapshot()

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "vscode-copilot"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	instructionsPath := filepath.Join(workspace, ".github", "instructions", "axiom.instructions.md")
	raw, err := os.ReadFile(instructionsPath)
	if err != nil {
		t.Fatalf("falta %s: %v", instructionsPath, err)
	}
	instructions := string(raw)
	if !strings.HasPrefix(instructions, "---\n") || !strings.Contains(strings.SplitN(instructions, "\n---\n", 2)[0], `applyTo: "**"`) {
		t.Errorf("la cabecera debe declarar applyTo: \"**\":\n%.300s", instructions)
	}
	for _, section := range []string{"<!-- axiom:sdd-orchestrator -->", "<!-- axiom:engram-protocol -->", "<!-- axiom:agent-routing -->"} {
		if !strings.Contains(instructions, section) {
			t.Errorf("las instrucciones no contienen %s", section)
		}
	}
	for _, unwanted := range []string{"{{AXIOM_SKILLS_DIR", ".agent/skills", "~/.copilot", "Code/User"} {
		if strings.Contains(instructions, unwanted) {
			t.Errorf("las instrucciones de proyecto contienen %q", unwanted)
		}
	}
	if !strings.Contains(instructions, "`.agents/skills/_shared/`") {
		t.Errorf("el orquestador debe apuntar a las skills compartidas del proyecto (.agents/skills)")
	}

	for _, rel := range []string{
		filepath.Join(".agents", "skills", "sdd-init", "SKILL.md"),
		filepath.Join(".agents", "skills", "go-testing", "SKILL.md"),
		filepath.Join(".agents", "skills", "_shared", "skill-resolver.md"),
	} {
		if _, err := os.Stat(filepath.Join(workspace, rel)); err != nil {
			t.Errorf("falta %s en el proyecto: %v", rel, err)
		}
	}

	mcp := readJSONObject(t, filepath.Join(workspace, ".vscode", "mcp.json"))
	servers, _ := mcp["servers"].(map[string]any)
	for _, name := range []string{"propio", "engram", "context7"} {
		if servers[name] == nil {
			t.Errorf(".vscode/mcp.json debe declarar el servidor %q: %v", name, servers)
		}
	}
	if _, wrongKey := mcp["mcpServers"]; wrongKey {
		t.Errorf(".vscode/mcp.json no usa mcpServers: %v", mcp)
	}
	if inputs, _ := mcp["inputs"].([]any); len(inputs) != 1 {
		t.Errorf("las inputs del usuario deben conservarse: %v", mcp["inputs"])
	}

	// Un agente `.agent.md` por fase SDD en .github/agents.
	if _, err := os.Stat(filepath.Join(workspace, ".github", "agents", "sdd-apply.agent.md")); err != nil {
		t.Errorf("falta el agente sdd-apply en .github/agents: %v", err)
	}
	// Prompts y hooks: Axiom no instala ninguno para VS Code.
	assertNothingUnder(t, workspace, filepath.Join(".github", "prompts"), filepath.Join(".github", "hooks"), filepath.Join(".github", "AGENTS.md"))
	// La guía de enrutado va al fichero de proyecto: ninguna distribución del perfil
	// de usuario de VS Code (macOS, Windows o Linux) aparece dentro del proyecto.
	assertNothingUnder(t, workspace, "Library", "AppData", ".config")
	assertTeamFilesIntact(t, workspace, team)
	assertSameTree(t, "home y APPDATA", before, userSnapshot())

	notes := strings.Join(result.Execution.ManualActions, "\n")
	if !strings.Contains(notes, "Workspace Trust") || !strings.Contains(notes, ".vscode/mcp.json") {
		t.Errorf("falta el aviso de Workspace Trust:\n%s", notes)
	}
	_ = home
}

// Instalación limpia, dos sync y un tercero: el tercero no cambia ni un byte del
// proyecto, del HOME ni de APPDATA.
func TestWorkspaceVSCodeThirdSyncChangesNothing(t *testing.T) {
	home, workspace, _, userSnapshot := vscodeUserEnv(t)
	team := seedVSCodeTeamFiles(t, workspace)
	before := userSnapshot()

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "vscode-copilot"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}
	snapshot := permissionsTreeSnapshot(t, workspace)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
	assertSameTree(t, "home y APPDATA", before, userSnapshot())
	assertTeamFilesIntact(t, workspace, team)
}

// Un .vscode/mcp.json con comentarios (VS Code lo admite) los conserva, con los
// servidores del usuario, y repetir sync no cambia ni un byte.
func TestWorkspaceVSCodeMCPKeepsCommentsAndIsIdempotent(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	mcpPath := filepath.Join(workspace, ".vscode", "mcp.json")
	mustWriteFile(t, mcpPath, []byte("// Servidores MCP del equipo\n{\n  \"servers\": {\n    \"propio\": {\"command\": \"propio\"}\n  },\n  // entradas de entrada\n  \"inputs\": []\n}\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "vscode-copilot"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	raw, _ := os.ReadFile(mcpPath)
	for _, comment := range []string{"// Servidores MCP del equipo", "// entradas de entrada"} {
		if !strings.Contains(string(raw), comment) {
			t.Errorf("se perdió el comentario %q:\n%s", comment, raw)
		}
	}
	servers, _ := readJSONObject(t, mcpPath)["servers"].(map[string]any)
	if servers["propio"] == nil || servers["engram"] == nil || servers["context7"] == nil {
		t.Errorf("faltan servidores tras la fusión: %v", servers)
	}

	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}
	settled, _ := os.ReadFile(mcpPath)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	if after, _ := os.ReadFile(mcpPath); !bytes.Equal(after, settled) {
		t.Errorf("el tercer sync cambió .vscode/mcp.json:\n%s\n---\n%s", settled, after)
	}
}

// Un .vscode/mcp.json que no es JSON válido se rechaza antes de escribir nada.
func TestWorkspaceVSCodeRefusesInvalidMCPJSON(t *testing.T) {
	_, workspace, _, userSnapshot := vscodeUserEnv(t)
	broken := []byte("{\"servers\": \n[esto no es json")
	mcpPath := filepath.Join(workspace, ".vscode", "mcp.json")
	mustWriteFile(t, mcpPath, broken)
	before := userSnapshot()

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "vscode-copilot"}, macOSDetectionResult())
	if err == nil || !strings.Contains(err.Error(), "no es JSON válido") {
		t.Fatalf("RunInstall() debe rechazar el fichero ilegible, error = %v", err)
	}
	if after, _ := os.ReadFile(mcpPath); !bytes.Equal(after, broken) {
		t.Errorf(".vscode/mcp.json se modificó:\n%s", after)
	}
	assertNothingUnder(t, workspace, ".github")
	assertSameTree(t, "home y APPDATA", before, userSnapshot())
}

// sync --scope workspace por la CLI, con un estado persistido que lista VS Code, lo
// instala en el proyecto: ya no se omite.
func TestRunSyncWorkspaceInstallsPersistedVSCodeInProject(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	notices := captureWorkspaceAgentsNotice(t)
	before := homeSnapshotWithoutState(t, home)
	if err := state.ProjectStore(home, root).Write(persistedAgents("vscode-copilot")); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	if want := []model.AgentID{model.AgentVSCodeCopilot}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
	if notices.Len() != 0 || len(result.OmittedAgents) != 0 {
		t.Errorf("VS Code admite workspace y no debe omitirse: aviso=%q omitidos=%v", notices, result.OmittedAgents)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".vscode", "mcp.json")); statErr != nil {
		t.Errorf("falta .vscode/mcp.json en el proyecto: %v", statErr)
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// Backup y verify cubren las rutas de proyecto de VS Code, y la verificación posterior
// a un sync no da ningún fallo.
func TestWorkspaceVSCodeBackupAndVerifyCoverProjectFiles(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	selection := vscodeSelection()
	wanted := []string{
		filepath.Join(workspace, ".vscode", "mcp.json"),
		filepath.Join(workspace, ".github", "instructions", "axiom.instructions.md"),
		filepath.Join(workspace, ".agents", "skills", "sdd-init", "SKILL.md"),
	}

	targets, err := syncBackupTargetsScoped(home, workspace, ScopeWorkspace, selection, resolveAdaptersScoped(ScopeWorkspace, workspace, selection.Agents))
	if err != nil {
		t.Fatalf("syncBackupTargetsScoped() error = %v", err)
	}
	for _, want := range wanted {
		if !containsPath(targets, want) {
			t.Errorf("la copia de seguridad no incluye %s: %v", want, targets)
		}
	}
	for _, target := range targets {
		if strings.HasPrefix(target, home) && !strings.HasPrefix(target, filepath.Join(home, ".axiom")) {
			t.Errorf("la copia de seguridad de workspace no debe incluir rutas de HOME: %s", target)
		}
	}

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync: %v", err)
	}
	report := runPostSyncVerificationScoped(home, workspace, ScopeWorkspace, selection)
	if report.Failed != 0 {
		t.Errorf("la verificación tiene %d fallos: %+v", report.Failed, report.Checks)
	}
	for _, want := range wanted {
		checked := false
		for _, check := range report.Checks {
			if strings.Contains(check.Description, want) || strings.Contains(check.ID, want) {
				checked = true
			}
		}
		if !checked {
			t.Errorf("la verificación no comprueba %s", want)
		}
	}
}

// Con el ámbito global VS Code sigue escribiendo en su perfil de usuario y en
// ~/.copilot/skills, y nada en el proyecto.
func TestRunInstallGlobalVSCodeKeepsGlobalLayout(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	adapter := vscode.NewAdapter()

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "vscode-copilot", "--component", "persona", "--component", "sdd", "--component", "engram", "--component", "skills", "--component", "context7"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	instructions, err := os.ReadFile(adapter.SystemPromptFile(home))
	if err != nil || !strings.Contains(string(instructions), "<!-- axiom:sdd-orchestrator -->") || !strings.Contains(string(instructions), `applyTo: "**"`) {
		t.Errorf("falta %s con el orquestador y applyTo: %v", adapter.SystemPromptFile(home), err)
	}
	if !strings.Contains(string(instructions), ".agent/skills/_shared/") {
		t.Errorf("el orquestador global conserva su texto de siempre")
	}
	servers, _ := readJSONObject(t, adapter.MCPConfigPath(home, "engram"))["servers"].(map[string]any)
	if servers["engram"] == nil || servers["del-usuario"] == nil {
		t.Errorf("el mcp.json del perfil de usuario debe declarar engram y conservar lo del usuario: %v", servers)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "skills", "sdd-init", "SKILL.md")); err != nil {
		t.Errorf("falta ~/.copilot/skills/sdd-init: %v", err)
	}
	assertNothingUnder(t, workspace, ".github", ".vscode")
}

// El aviso de Workspace Trust solo sale con VS Code Copilot y con el ámbito workspace.
func TestWorkspaceTrustNotesVSCode(t *testing.T) {
	notes := workspaceTrustNotes(ScopeWorkspace, []model.AgentID{model.AgentVSCodeCopilot})
	if len(notes) != 1 || !strings.Contains(notes[0], "Workspace Trust") {
		t.Errorf("con VS Code en workspace se espera un aviso de Workspace Trust: %v", notes)
	}
	if notes := workspaceTrustNotes(ScopeGlobal, []model.AgentID{model.AgentVSCodeCopilot}); len(notes) != 0 {
		t.Errorf("con ámbito global no hay aviso: %v", notes)
	}
	if notes := workspaceTrustNotes(ScopeWorkspace, []model.AgentID{model.AgentVSCodeCopilot, model.AgentCodex}); len(notes) != 2 {
		t.Errorf("con Codex y VS Code se esperan dos avisos: %v", notes)
	}
}
