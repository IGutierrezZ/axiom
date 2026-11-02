package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/IGutierrezZ/axiom/v3/internal/components/engram"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// Las pruebas de este fichero cubren la distribución de proyecto de Codex con
// --scope workspace. Codex carga <proyecto>/.codex/config.toml (mcp_servers,
// features, agents y developer_instructions, que se añade a las instrucciones
// base), <proyecto>/.codex/hooks.json y <proyecto>/.agents/skills, solo en
// proyectos de confianza; no carga .codex/AGENTS.md ni .codex/skills. Las
// instrucciones de Axiom van en developer_instructions y nunca en el AGENTS.md
// raíz, que es del equipo.

// codexIgnoredProjectKeys son las claves que Codex ignora en la configuración de
// proyecto (y avisa de ello al arrancar): Axiom nunca las escribe ahí.
var codexIgnoredProjectKeys = []string{
	"openai_base_url", "chatgpt_base_url", "apps_mcp_product_sku", "model_provider",
	"model_providers", "notify", "profile", "profiles", "experimental_realtime_ws_base_url", "otel",
}

func codexSelection() model.Selection {
	return model.Selection{
		Agents:     []model.AgentID{model.AgentCodex},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentPersona, model.ComponentSDD, model.ComponentSkills, model.ComponentContext7},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
}

// readTOMLMap analiza el fichero con el analizador de referencia: si Axiom escribió
// un TOML inválido, la prueba falla aquí.
func readTOMLMap(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	var parsed map[string]any
	if _, err := toml.Decode(string(raw), &parsed); err != nil {
		t.Fatalf("%s no es TOML válido: %v\n%s", path, err, raw)
	}
	return parsed
}

func developerInstructionsOf(t *testing.T, configPath string) string {
	t.Helper()
	value, ok := readTOMLMap(t, configPath)["developer_instructions"].(string)
	if !ok {
		t.Fatalf("%s no declara developer_instructions como cadena", configPath)
	}
	return value
}

func captureCodexEngramNotices(t *testing.T) *bytes.Buffer {
	t.Helper()
	var notices bytes.Buffer
	restore := engram.CodexWorkspaceNoticeWriter
	engram.CodexWorkspaceNoticeWriter = &notices
	t.Cleanup(func() { engram.CodexWorkspaceNoticeWriter = restore })
	return &notices
}

// seedCodexUserHome siembra la configuración de usuario de Codex: la instalación en
// el proyecto no debe tocarla ni escribir en ella la confianza del proyecto.
func seedCodexUserHome(t *testing.T, home string) {
	t.Helper()
	for path, content := range map[string]string{
		filepath.Join(home, ".codex", "config.toml"):           "model = \"del-usuario\"\n\n[projects.'/otro/proyecto']\ntrust_level = \"trusted\"\n",
		filepath.Join(home, ".codex", "AGENTS.md"):             "instrucciones globales del usuario\n",
		filepath.Join(home, ".codex", "prompts", "usuario.md"): "prompt del usuario\n",
	} {
		mustWriteFile(t, path, []byte(content))
	}
}

// Las instrucciones van a developer_instructions de .codex/config.toml, el MCP, los
// hooks y las skills a las rutas de proyecto que Codex lee, el AGENTS.md de la raíz
// (del equipo) queda intacto y HOME no cambia ni un byte.
func TestRunInstallWorkspaceCodexUsesProjectLayout(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	seedCodexUserHome(t, home)
	before := homeSnapshotWithoutState(t, home)
	teamAgents := []byte("# Reglas del equipo\n\nNo tocar.\n")
	mustWriteFile(t, filepath.Join(workspace, "AGENTS.md"), teamAgents)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	configPath := filepath.Join(workspace, ".codex", "config.toml")
	config := readTOMLMap(t, configPath)
	instructions := developerInstructionsOf(t, configPath)
	for _, section := range []string{
		"<!-- axiom:workspace-instructions -->", "<!-- axiom:sdd-orchestrator -->", "<!-- axiom:engram-protocol -->",
		"<!-- axiom:engram-passive-capture -->", "<!-- axiom:agent-routing -->",
	} {
		if !strings.Contains(instructions, section) {
			t.Errorf("developer_instructions no contiene %s", section)
		}
	}
	for _, unwanted := range []string{"{{AXIOM_SKILLS_DIR", "~/.codex", "{{CODEX_"} {
		if strings.Contains(instructions, unwanted) {
			t.Errorf("developer_instructions contiene %q sin resolver para un proyecto", unwanted)
		}
	}
	if !strings.Contains(instructions, ".agents/skills/_shared/skill-resolver.md") {
		t.Errorf("el orquestador debe apuntar a las skills del proyecto (.agents/skills)")
	}

	servers, _ := config["mcp_servers"].(map[string]any)
	if servers["engram"] == nil || servers["context7"] == nil {
		t.Errorf("mcp_servers debe declarar engram y context7: %v", servers)
	}
	if features, _ := config["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("falta features.multi_agent = true: %v", config["features"])
	}
	if agentsTable, _ := config["agents"].(map[string]any); agentsTable["max_threads"] != int64(4) || agentsTable["max_depth"] != int64(2) {
		t.Errorf("faltan agents.max_threads/max_depth: %v", config["agents"])
	}
	if config["experimental_compact_prompt_file"] != "engram-compact-prompt.md" {
		t.Errorf("experimental_compact_prompt_file = %v, quiero el nombre relativo a .codex", config["experimental_compact_prompt_file"])
	}
	for _, ignored := range append([]string{"model_instructions_file"}, codexIgnoredProjectKeys...) {
		if _, written := config[ignored]; written {
			t.Errorf("Axiom no debe escribir %s en la configuración de proyecto de Codex", ignored)
		}
	}

	for _, rel := range []string{
		filepath.Join(".codex", "hooks.json"),
		filepath.Join(".codex", "axiom-instructions.md"),
		filepath.Join(".codex", "engram-compact-prompt.md"),
		filepath.Join(".agents", "skills", "sdd-init", "SKILL.md"),
		filepath.Join(".agents", "skills", "go-testing", "SKILL.md"),
		filepath.Join(".agents", "skills", "_shared", "skill-resolver.md"),
	} {
		if _, err := os.Stat(filepath.Join(workspace, rel)); err != nil {
			t.Errorf("falta %s en el proyecto: %v", rel, err)
		}
	}
	hooks, _ := os.ReadFile(filepath.Join(workspace, ".codex", "hooks.json"))
	if !strings.Contains(string(hooks), "skill-registry refresh") || !strings.Contains(string(hooks), "SessionStart") {
		t.Errorf(".codex/hooks.json no declara el hook de registro de skills:\n%s", hooks)
	}
	assertNothingUnder(t, workspace,
		filepath.Join(".codex", "AGENTS.md"), filepath.Join(".codex", "skills"),
		filepath.Join(".codex", "engram-instructions.md"),
		filepath.Join(".codex", "sdd-strong.config.toml"), filepath.Join(".codex", "sdd-mid.config.toml"), filepath.Join(".codex", "sdd-cheap.config.toml"))
	if after, _ := os.ReadFile(filepath.Join(workspace, "AGENTS.md")); !bytes.Equal(after, teamAgents) {
		t.Errorf("el AGENTS.md de la raíz cambió:\n%s", after)
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	notes := strings.Join(result.Execution.ManualActions, "\n")
	if !strings.Contains(notes, "de confianza") || !strings.Contains(notes, "hook") {
		t.Errorf("falta el aviso de confianza del proyecto y aprobación de hooks:\n%s", notes)
	}
}

// El bloque gestionado reproduce exactamente el fichero de instrucciones de Axiom, y
// el texto propio del usuario en developer_instructions (con comillas, barras y
// delimitadores TOML) se conserva fuera del bloque. El resto de la configuración del
// usuario también, y un tercer sync no cambia ni un byte.
func TestWorkspaceCodexKeepsUserConfigAndIsIdempotent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedCodexUserHome(t, home)
	before := homeSnapshotWithoutState(t, home)
	userText := "Texto propio del equipo.\nCon \"comillas\", \\barras\\, ''' y \"\"\" juntos.\n[features]\nno es una tabla\n"
	userConfig := "# Configuración del equipo\nmodel = \"gpt-del-equipo\"\ndeveloper_instructions = " + filemerge.FormatTOMLString(userText) + "\n\n" +
		"[mcp_servers.propio]\ncommand = \"propio\"\nargs = [\"--x\"]\n\n[profiles.mio]\nmodel = \"x\"\n"
	configPath := filepath.Join(workspace, ".codex", "config.toml")
	mustWriteFile(t, configPath, []byte(userConfig))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, codexSelection()); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}

	raw, _ := os.ReadFile(configPath)
	if !strings.Contains(string(raw), "# Configuración del equipo") {
		t.Errorf("se perdió el comentario del usuario:\n%s", raw)
	}
	config := readTOMLMap(t, configPath)
	if config["model"] != "gpt-del-equipo" {
		t.Errorf("model = %v, la clave del usuario debe conservarse", config["model"])
	}
	servers, _ := config["mcp_servers"].(map[string]any)
	if servers["propio"] == nil || servers["engram"] == nil {
		t.Errorf("mcp_servers debe conservar el servidor del usuario y añadir engram: %v", servers)
	}
	if profiles, _ := config["profiles"].(map[string]any); profiles["mio"] == nil {
		t.Errorf("los profiles del usuario no se tocan: %v", config["profiles"])
	}

	value := developerInstructionsOf(t, configPath)
	if !strings.HasPrefix(value, userText) {
		t.Errorf("el texto del usuario debe conservarse exacto antes del bloque:\n%q", value)
	}
	if strings.Count(value, "<!-- axiom:workspace-instructions -->") != 1 {
		t.Errorf("debe haber un único bloque gestionado:\n%s", value)
	}
	source, _ := os.ReadFile(filepath.Join(workspace, ".codex", "axiom-instructions.md"))
	if got := filemerge.ExtractManagedSection(value, filemerge.WorkspaceInstructionsSectionID); got != string(source) {
		t.Errorf("el bloque gestionado no coincide exactamente con .codex/axiom-instructions.md")
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	snapshot := permissionsTreeSnapshot(t, workspace)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, codexSelection()); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
}

// Instalación limpia, dos sync y un tercero: el tercero no cambia ni un byte del
// proyecto ni de HOME.
func TestWorkspaceCodexThirdSyncChangesNothing(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedCodexUserHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, codexSelection()); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}
	snapshot := permissionsTreeSnapshot(t, workspace)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, codexSelection()); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// sync --scope workspace por la CLI, con un estado persistido que lista Codex, lo
// instala en el proyecto: ya no se omite.
func TestRunSyncWorkspaceInstallsPersistedCodexInProject(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	notices := captureWorkspaceAgentsNotice(t)
	before := homeSnapshotWithoutState(t, home)
	if err := state.ProjectStore(home, root).Write(persistedAgents("codex")); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	if want := []model.AgentID{model.AgentCodex}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
	if notices.Len() != 0 || len(result.OmittedAgents) != 0 {
		t.Errorf("Codex admite workspace y no debe omitirse: aviso=%q omitidos=%v", notices, result.OmittedAgents)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".codex", "config.toml")); statErr != nil {
		t.Errorf("falta .codex/config.toml en el proyecto: %v", statErr)
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// Un developer_instructions que no es una cadena es del usuario: no se toca y se
// avisa. El resto de la instalación sigue.
func TestWorkspaceCodexNonStringDeveloperInstructionsIsLeftUntouched(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	notices := captureWorkspaceNotices(t)
	configPath := filepath.Join(workspace, ".codex", "config.toml")
	mustWriteFile(t, configPath, []byte("developer_instructions = [\"a\", \"b\"]\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex", "--component", "persona", "--component", "sdd", "--component", "engram"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	config := readTOMLMap(t, configPath)
	if got, _ := config["developer_instructions"].([]any); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("developer_instructions cambió: %v", config["developer_instructions"])
	}
	if !strings.Contains(notices.String(), "developer_instructions") {
		t.Errorf("falta el aviso:\n%s", notices.String())
	}
	if servers, _ := config["mcp_servers"].(map[string]any); servers["engram"] == nil {
		t.Errorf("el MCP de Engram debe instalarse igualmente: %v", config["mcp_servers"])
	}
}

// Un config.toml que no es TOML válido se rechaza antes de escribir nada.
func TestWorkspaceCodexRefusesInvalidConfigToml(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	broken := []byte("model = \n[esto no es toml")
	configPath := filepath.Join(workspace, ".codex", "config.toml")
	mustWriteFile(t, configPath, broken)

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult())
	if err == nil || !strings.Contains(err.Error(), "no es TOML válido") {
		t.Fatalf("RunInstall() debe rechazar el fichero ilegible, error = %v", err)
	}
	if after, _ := os.ReadFile(configPath); !bytes.Equal(after, broken) {
		t.Errorf("config.toml se modificó:\n%s", after)
	}
	assertNothingUnder(t, workspace, filepath.Join(".codex", "axiom-instructions.md"), filepath.Join(".agents", "skills"), filepath.Join(".codex", "hooks.json"))
}

// Si el usuario ya declara su propio prompt de compactación, Axiom lo respeta y avisa.
func TestWorkspaceCodexKeepsUserCompactPrompt(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	notices := captureCodexEngramNotices(t)
	configPath := filepath.Join(workspace, ".codex", "config.toml")
	mustWriteFile(t, configPath, []byte("experimental_compact_prompt_file = \"mi-compactacion.md\"\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex", "--component", "engram"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if got := readTOMLMap(t, configPath)["experimental_compact_prompt_file"]; got != "mi-compactacion.md" {
		t.Errorf("experimental_compact_prompt_file = %v, el del usuario debe conservarse", got)
	}
	if !strings.Contains(notices.String(), "experimental_compact_prompt_file") {
		t.Errorf("falta el aviso:\n%s", notices.String())
	}
}

// Migración: el .codex/AGENTS.md de una instalación workspace anterior (que Codex
// nunca leyó) se retira solo cuando es provablemente de Axiom.
func TestWorkspaceCodexRetiresProvablyAxiomLegacyAgentsMD(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	legacy := filepath.Join(workspace, ".codex", "AGENTS.md")
	mustWriteFile(t, legacy, []byte("<!-- axiom:persona -->\nPersona antigua.\n<!-- /axiom:persona -->\n\n<!-- axiom:sdd-orchestrator -->\nOrquestador antiguo.\n<!-- /axiom:sdd-orchestrator -->\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertNothingUnder(t, workspace, filepath.Join(".codex", "AGENTS.md"))
	if _, err := os.Stat(filepath.Join(workspace, ".codex", "axiom-instructions.md")); err != nil {
		t.Errorf("falta .codex/axiom-instructions.md: %v", err)
	}
}

// Un .codex/AGENTS.md con contenido que Axiom no escribió se deja y se avisa.
func TestWorkspaceCodexKeepsLegacyAgentsMDWithUserContent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	legacy := filepath.Join(workspace, ".codex", "AGENTS.md")
	content := []byte("# Mis notas\n\n<!-- axiom:sdd-orchestrator -->\nOrquestador antiguo.\n<!-- /axiom:sdd-orchestrator -->\n")
	mustWriteFile(t, legacy, content)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if after, _ := os.ReadFile(legacy); !bytes.Equal(after, content) {
		t.Errorf(".codex/AGENTS.md cambió:\n%s", after)
	}
	if !strings.Contains(strings.Join(result.Execution.ManualActions, "\n"), legacy) {
		t.Errorf("falta el aviso del fichero conservado: %v", result.Execution.ManualActions)
	}
}

// Con el ámbito global Codex sigue escribiendo en ~/.codex como siempre: AGENTS.md,
// model_instructions_file, skills en ~/.codex/skills y nada en el proyecto.
func TestRunInstallGlobalCodexKeepsGlobalLayout(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "codex", "--component", "persona", "--component", "sdd", "--component", "engram", "--component", "skills"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	agentsMD, err := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if err != nil || !strings.Contains(string(agentsMD), "<!-- axiom:sdd-orchestrator -->") {
		t.Errorf("falta ~/.codex/AGENTS.md con el orquestador: %v", err)
	}
	if !strings.Contains(string(agentsMD), "~/.codex/skills/_shared/skill-resolver.md") {
		t.Errorf("el orquestador global debe seguir apuntando a ~/.codex/skills")
	}
	config := readTOMLMap(t, filepath.Join(home, ".codex", "config.toml"))
	if config["model_instructions_file"] == nil || config["experimental_compact_prompt_file"] == nil {
		t.Errorf("la instalación global debe seguir escribiendo model_instructions_file y el prompt de compactación: %v", config)
	}
	if _, declared := config["developer_instructions"]; declared {
		t.Errorf("la instalación global no debe declarar developer_instructions")
	}
	for _, rel := range []string{filepath.Join(".codex", "skills", "sdd-init", "SKILL.md"), filepath.Join(".codex", "engram-instructions.md"), filepath.Join(".codex", "hooks.json")} {
		if _, err := os.Stat(filepath.Join(home, rel)); err != nil {
			t.Errorf("falta ~/%s: %v", rel, err)
		}
	}
	assertNothingUnder(t, home, filepath.Join(".codex", "axiom-instructions.md"))
	assertNothingUnder(t, workspace, ".codex", ".agents")
}

// Backup y verify cubren las rutas de proyecto de Codex, y la verificación posterior
// a un sync no da ningún fallo.
func TestWorkspaceCodexBackupAndVerifyCoverProjectFiles(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	selection := codexSelection()
	wanted := []string{
		filepath.Join(workspace, ".codex", "config.toml"),
		filepath.Join(workspace, ".codex", "axiom-instructions.md"),
		filepath.Join(workspace, ".codex", "hooks.json"),
		filepath.Join(workspace, ".codex", "engram-compact-prompt.md"),
		filepath.Join(workspace, ".codex", "AGENTS.md"),
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
	for _, want := range wanted[:4] {
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

// El aviso de confianza solo sale con Codex y con el ámbito workspace.
func TestWorkspaceTrustNotes(t *testing.T) {
	if notes := workspaceTrustNotes(ScopeWorkspace, []model.AgentID{model.AgentClaudeCode, model.AgentCodex}); len(notes) != 1 || !strings.Contains(notes[0], "de confianza") {
		t.Errorf("con Codex en workspace se espera un aviso de confianza: %v", notes)
	}
	if notes := workspaceTrustNotes(ScopeGlobal, []model.AgentID{model.AgentCodex}); len(notes) != 0 {
		t.Errorf("con ámbito global no hay aviso: %v", notes)
	}
	if notes := workspaceTrustNotes(ScopeWorkspace, []model.AgentID{model.AgentClaudeCode}); len(notes) != 0 {
		t.Errorf("sin Codex no hay aviso: %v", notes)
	}
}

// Codex no escribe permisos ni en global ni en el proyecto: no hay destino válido
// para el componente, así que no se omite nada ni se escribe nada.
func TestWorkspaceCodexHasNoPermissionsTarget(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedCodexUserHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex", "--component", "permissions", "--component", "engram"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	config := readTOMLMap(t, filepath.Join(workspace, ".codex", "config.toml"))
	for _, key := range []string{"permissions", "default_permissions", "approval_policy", "sandbox_mode"} {
		if _, written := config[key]; written {
			t.Errorf("Axiom no escribe %s en la configuración de Codex", key)
		}
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}
