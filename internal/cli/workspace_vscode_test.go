package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// Las pruebas de este fichero cubren la distribución de proyecto de VS Code Copilot
// con --scope workspace. VS Code lee .github/instructions/*.instructions.md (con
// `applyTo: "**"` se aplican siempre), .agents/skills y .mcp.json (clave raíz
// `mcpServers`, compartido con Claude Code y con Workspace Trust); de .vscode/mcp.json,
// del arnés Local, solo se retiran las entradas de Axiom. No se usan .github/copilot-instructions.md ni el
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
		filepath.Join(".vscode", "settings.json"):        []byte("{\n  // ajustes del equipo\n  \"editor.tabSize\": 2\n}\n"),
		filepath.Join(".github", "hooks", "equipo.json"): []byte("{\"version\": 1, \"hooks\": {\"sessionStart\": [{\"type\": \"command\", \"command\": \"echo equipo\"}]}}\n"),
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
// y el MCP a .mcp.json (clave mcpServers, el destino del arnés de Copilot); el
// .vscode/mcp.json del usuario conserva lo suyo, y los ficheros del equipo y el
// perfil de usuario de VS Code no cambian ni un byte.
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

	project := readJSONObject(t, filepath.Join(workspace, ".mcp.json"))
	projectServers, _ := project["mcpServers"].(map[string]any)
	for _, name := range []string{"engram", "context7"} {
		entry, _ := projectServers[name].(map[string]any)
		if entry == nil {
			t.Errorf(".mcp.json debe declarar el servidor %q: %v", name, projectServers)
			continue
		}
		if entry["type"] != "stdio" || fmt.Sprint(entry["tools"]) != "[*]" {
			t.Errorf(".mcp.json debe declarar %q con type stdio y tools [*]: %v", name, entry)
		}
	}
	legacy := readJSONObject(t, filepath.Join(workspace, ".vscode", "mcp.json"))
	servers, _ := legacy["servers"].(map[string]any)
	if servers["propio"] == nil || servers["engram"] != nil || servers["context7"] != nil {
		t.Errorf(".vscode/mcp.json conserva solo el servidor del usuario: %v", servers)
	}
	if inputs, _ := legacy["inputs"].([]any); len(inputs) != 1 {
		t.Errorf("las inputs del usuario deben conservarse: %v", legacy["inputs"])
	}

	// Un agente `.agent.md` por fase SDD en .github/agents.
	if _, err := os.Stat(filepath.Join(workspace, ".github", "agents", "sdd-apply.agent.md")); err != nil {
		t.Errorf("falta el agente sdd-apply en .github/agents: %v", err)
	}
	// Prompts: Axiom no instala ninguno para VS Code. Hooks: solo el suyo, con la guarda de rama.
	assertNothingUnder(t, workspace, filepath.Join(".github", "prompts"), filepath.Join(".github", "AGENTS.md"))
	hooks, err := os.ReadFile(filepath.Join(workspace, ".github", "hooks", "axiom-branch-guard.json"))
	if err != nil || !bytes.Equal(hooks, sdd.VSCodeBranchGuardHooksJSON(model.AgentVSCodeCopilot)) {
		t.Errorf("falta el hook de la guarda de rama en .github/hooks (err=%v):\n%s", err, hooks)
	}
	// La guía de enrutado va al fichero de proyecto: ninguna distribución del perfil
	// de usuario de VS Code (macOS, Windows o Linux) aparece dentro del proyecto.
	assertNothingUnder(t, workspace, "Library", "AppData", ".config")
	assertTeamFilesIntact(t, workspace, team)
	assertSameTree(t, "home y APPDATA", before, userSnapshot())

	notes := strings.Join(result.Execution.ManualActions, "\n")
	if !strings.Contains(notes, "Workspace Trust") || !strings.Contains(notes, ".mcp.json") || strings.Contains(notes, ".vscode/mcp.json") {
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

// Una instalación anterior dejó Engram y Context7 en .vscode/mcp.json, con
// comentarios y un servidor del usuario. sync los lleva a .mcp.json y retira de
// .vscode/mcp.json solo las entradas de Axiom: los comentarios y el servidor propio se
// quedan, y repetir sync no cambia ni un byte de ninguno de los dos ficheros.
func TestWorkspaceVSCodeMigratesLegacyMCPAndIsIdempotent(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	legacyPath := filepath.Join(workspace, ".vscode", "mcp.json")
	projectPath := filepath.Join(workspace, ".mcp.json")
	mustWriteFile(t, legacyPath, []byte("// Servidores MCP del equipo\n{\n  \"servers\": {\n    \"propio\": {\"command\": \"propio\"},\n    \"engram\": {\"command\": \"engram\", \"args\": [\"mcp\", \"--tools=agent\"]},\n    \"context7\": {\"type\": \"http\", \"url\": \"https://mcp.context7.com/mcp\"}\n  },\n  // entradas de entrada\n  \"inputs\": []\n}\n"))

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync: %v", err)
	}

	raw, _ := os.ReadFile(legacyPath)
	for _, comment := range []string{"// Servidores MCP del equipo", "// entradas de entrada"} {
		if !strings.Contains(string(raw), comment) {
			t.Errorf("se perdió el comentario %q:\n%s", comment, raw)
		}
	}
	servers, _ := readJSONObject(t, legacyPath)["servers"].(map[string]any)
	if servers["propio"] == nil || servers["engram"] != nil || servers["context7"] != nil {
		t.Errorf(".vscode/mcp.json debe conservar solo el servidor del usuario: %v", servers)
	}
	projectServers, _ := readJSONObject(t, projectPath)["mcpServers"].(map[string]any)
	if projectServers["engram"] == nil || projectServers["context7"] == nil || projectServers["propio"] != nil {
		t.Errorf(".mcp.json debe declarar solo los servidores de Axiom: %v", projectServers)
	}

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync 2: %v", err)
	}
	settledLegacy, _ := os.ReadFile(legacyPath)
	settledProject, _ := os.ReadFile(projectPath)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	if after, _ := os.ReadFile(legacyPath); !bytes.Equal(after, settledLegacy) {
		t.Errorf("el tercer sync cambió .vscode/mcp.json:\n%s\n---\n%s", settledLegacy, after)
	}
	if after, _ := os.ReadFile(projectPath); !bytes.Equal(after, settledProject) {
		t.Errorf("el tercer sync cambió .mcp.json:\n%s\n---\n%s", settledProject, after)
	}
}

// Si .vscode/mcp.json solo tenía lo que escribió Axiom, la migración lo borra junto
// con la carpeta .vscode vacía.
func TestWorkspaceVSCodeMigrationRemovesLegacyFileLeftEmpty(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	legacyPath := filepath.Join(workspace, ".vscode", "mcp.json")
	mustWriteFile(t, legacyPath, []byte("{\n  \"servers\": {\n    \"context7\": {\n      \"type\": \"http\",\n      \"url\": \"https://mcp.context7.com/mcp\"\n    },\n    \"engram\": {\n      \"command\": \"C:\\\\tools\\\\engram.exe\",\n      \"args\": [\"mcp\", \"--tools=agent\"]\n    }\n  }\n}\n"))

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	assertNothingUnder(t, workspace, ".vscode")
	if servers, _ := readJSONObject(t, filepath.Join(workspace, ".mcp.json"))["mcpServers"].(map[string]any); servers["engram"] == nil || servers["context7"] == nil {
		t.Errorf(".mcp.json debe declarar Engram y Context7: %v", servers)
	}
}

// Claude Code y VS Code Copilot comparten .mcp.json: con los dos instalados, cada
// servidor tiene una sola entrada válida para ambos y repetir sync no cambia ni un
// byte.
func TestWorkspaceClaudeAndVSCodeShareProjectMCPJSON(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	selection := vscodeSelection()
	selection.Agents = []model.AgentID{model.AgentClaudeCode, model.AgentVSCodeCopilot}
	projectPath := filepath.Join(workspace, ".mcp.json")

	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}
	settled, _ := os.ReadFile(projectPath)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	if after, _ := os.ReadFile(projectPath); !bytes.Equal(after, settled) {
		t.Errorf("el tercer sync cambió .mcp.json:\n%s\n---\n%s", settled, after)
	}

	servers, _ := readJSONObject(t, projectPath)["mcpServers"].(map[string]any)
	context7, _ := servers["context7"].(map[string]any)
	if context7["command"] != "npx" || context7["url"] != nil || context7["type"] != "stdio" {
		t.Errorf("Context7 debe tener una sola forma (npx por stdio) para los dos clientes: %v", context7)
	}
	engram, _ := servers["engram"].(map[string]any)
	if command, _ := engram["command"].(string); command == "" || engram["type"] != "stdio" || fmt.Sprint(engram["args"]) != "[mcp --tools=agent]" {
		t.Errorf("Engram debe tener una sola forma para los dos clientes: %v", engram)
	}
}

// Un .mcp.json que no es JSON válido se rechaza antes de escribir nada.
func TestWorkspaceVSCodeRefusesInvalidMCPJSON(t *testing.T) {
	_, workspace, _, userSnapshot := vscodeUserEnv(t)
	broken := []byte("{\"mcpServers\": \n[esto no es json")
	mcpPath := filepath.Join(workspace, ".mcp.json")
	mustWriteFile(t, mcpPath, broken)
	before := userSnapshot()

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "vscode-copilot"}, macOSDetectionResult())
	if err == nil || !strings.Contains(err.Error(), "no es JSON válido") {
		t.Fatalf("RunInstall() debe rechazar el fichero ilegible, error = %v", err)
	}
	if after, _ := os.ReadFile(mcpPath); !bytes.Equal(after, broken) {
		t.Errorf(".mcp.json se modificó:\n%s", after)
	}
	assertNothingUnder(t, workspace, ".github")
	assertSameTree(t, "home y APPDATA", before, userSnapshot())
}

// Un .vscode/mcp.json ilegible no bloquea: Axiom ya no escribe en él y lo deja como
// está.
func TestWorkspaceVSCodeLeavesUnreadableLegacyMCPJSON(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	broken := []byte("{\"servers\": \n[esto no es json")
	legacyPath := filepath.Join(workspace, ".vscode", "mcp.json")
	mustWriteFile(t, legacyPath, broken)

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, vscodeSelection()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if after, _ := os.ReadFile(legacyPath); !bytes.Equal(after, broken) {
		t.Errorf(".vscode/mcp.json se modificó:\n%s", after)
	}
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
	if _, statErr := os.Stat(filepath.Join(root, ".mcp.json")); statErr != nil {
		t.Errorf("falta .mcp.json en el proyecto: %v", statErr)
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// Backup y verify cubren las rutas de proyecto de VS Code, y la verificación posterior
// a un sync no da ningún fallo.
func TestWorkspaceVSCodeBackupAndVerifyCoverProjectFiles(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	selection := vscodeSelection()
	wanted := []string{
		filepath.Join(workspace, ".mcp.json"),
		filepath.Join(workspace, ".github", "instructions", "axiom.instructions.md"),
		filepath.Join(workspace, ".agents", "skills", "sdd-init", "SKILL.md"),
	}

	targets, err := syncBackupTargetsScoped(home, workspace, ScopeWorkspace, selection, resolveAdaptersScoped(ScopeWorkspace, workspace, selection.Agents))
	if err != nil {
		t.Fatalf("syncBackupTargetsScoped() error = %v", err)
	}
	// La migración reescribe o borra .vscode/mcp.json: va en la copia para poder
	// restaurarlo, aunque no se verifique.
	if legacy := filepath.Join(workspace, ".vscode", "mcp.json"); !containsPath(targets, legacy) {
		t.Errorf("la copia de seguridad no incluye %s: %v", legacy, targets)
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

// Con el ámbito global VS Code escribe las instrucciones en su perfil de usuario, las
// skills en ~/.copilot/skills y el MCP en ~/.copilot/mcp-config.json (`mcpServers`, el
// destino de usuario del arnés de Copilot). Del mcp.json del perfil (`servers`, arnés
// Local) retira las entradas de Axiom y conserva las del usuario. Nada en el proyecto.
func TestRunInstallGlobalVSCodeKeepsGlobalLayout(t *testing.T) {
	home, workspace, _, _ := vscodeUserEnv(t)
	adapter := vscode.NewAdapter()
	profile := adapter.LegacyMCPConfigPath(home)
	mustWriteFile(t, profile, []byte("{\"servers\":{\"del-usuario\":{\"command\":\"usuario\"},\"engram\":{\"command\":\"engram\",\"args\":[\"mcp\",\"--tools=agent\"]},\"context7\":{\"type\":\"http\",\"url\":\"https://mcp.context7.com/mcp\"}}}\n"))

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
	config := adapter.MCPConfigPath(home, "engram")
	if config != filepath.Join(home, ".copilot", "mcp-config.json") {
		t.Fatalf("MCPConfigPath = %q, quiero ~/.copilot/mcp-config.json", config)
	}
	mcpServers, _ := readJSONObject(t, config)["mcpServers"].(map[string]any)
	for _, name := range []string{"engram", "context7"} {
		entry, _ := mcpServers[name].(map[string]any)
		if entry == nil || entry["tools"] == nil {
			t.Errorf("mcp-config.json debe declarar %s con tools: %v", name, mcpServers)
		}
	}
	servers, _ := readJSONObject(t, profile)["servers"].(map[string]any)
	if servers["engram"] != nil || servers["context7"] != nil || servers["del-usuario"] == nil {
		t.Errorf("el mcp.json del perfil debe perder las entradas de Axiom y conservar las del usuario: %v", servers)
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
