package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Las pruebas de este fichero cubren la distribución de proyecto de OpenCode y
// Kilo con --scope workspace: opencode.json y .opencode/ (OpenCode), .kilo/ (Kilo).
// Ningún agente lee <proyecto>/.config/opencode ni <proyecto>/.config/kilo, que es
// donde Axiom escribía antes.

func readJSONObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", path, err)
	}
	object, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		t.Fatalf("%s no es JSON válido: %v", path, err)
	}
	return object
}

func instructionsOf(t *testing.T, path string) []string {
	t.Helper()
	raw, _ := readJSONObject(t, path)["instructions"].([]any)
	entries := make([]string, 0, len(raw))
	for _, entry := range raw {
		entries = append(entries, entry.(string))
	}
	return entries
}

func assertNothingUnder(t *testing.T, root string, relative ...string) {
	t.Helper()
	for _, rel := range relative {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Errorf("no debe existir %s", filepath.Join(root, rel))
		}
	}
}

func TestRunInstallWorkspaceOpenCodeAndKiloUseProjectLayout(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode,kilocode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	for _, rel := range []string{
		"opencode.json",
		filepath.Join(".opencode", "axiom.md"),
		filepath.Join(".opencode", "commands", "sdd-init.md"),
		filepath.Join(".agents", "skills", "sdd-init", "SKILL.md"),
		filepath.Join(".opencode", "plugins", "skill-registry.ts"),
		filepath.Join(".kilo", "kilo.json"),
		filepath.Join(".kilo", "rules", "axiom.md"),
		filepath.Join(".kilo", "commands", "sdd-init.md"),
		filepath.Join(".kilo", "plugins", "skill-registry.ts"),
	} {
		if _, err := os.Stat(filepath.Join(workspace, rel)); err != nil {
			t.Errorf("falta %s en el proyecto: %v", rel, err)
		}
	}
	assertNothingUnder(t, workspace, filepath.Join(".config", "opencode"), filepath.Join(".config", "kilo"), "AGENTS.md")
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	for _, tc := range []struct {
		settings, skillsDir, instruction string
	}{
		{"opencode.json", ".agents/skills", ".opencode/axiom.md"},
		{filepath.Join(".kilo", "kilo.json"), ".agents/skills", ".kilo/rules/axiom.md"},
	} {
		path := filepath.Join(workspace, tc.settings)
		settings := readJSONObject(t, path)
		if got := instructionsOf(t, path); !slices.Equal(got, []string{tc.instruction}) {
			t.Errorf("%s: instructions = %v, quiero [%s]", tc.settings, got, tc.instruction)
		}
		mcp, _ := settings["mcp"].(map[string]any)
		for _, server := range []string{"engram", "context7"} {
			if _, ok := mcp[server]; !ok {
				t.Errorf("%s: falta mcp.%s", tc.settings, server)
			}
		}
		if _, ok := settings["permission"]; !ok {
			t.Errorf("%s: falta la clave permission", tc.settings)
		}
		raw, _ := os.ReadFile(path)
		text := string(raw)
		if strings.Contains(text, "{{AXIOM_SKILLS_DIR") || strings.Contains(text, ".config/opencode") || strings.Contains(text, ".config/kilo") {
			t.Errorf("%s: quedan marcadores sin resolver o rutas globales", tc.settings)
		}
		if !strings.Contains(text, tc.skillsDir+"/sdd-research/SKILL.md") {
			t.Errorf("%s: el overlay SDD debe apuntar a %s", tc.settings, tc.skillsDir)
		}
	}
}

// Un opencode.json del usuario conserva sus claves y su instructions, la entrada
// de Axiom aparece una sola vez tras instalar y sincronizar dos veces, y el
// AGENTS.md de la raíz, que es del equipo, no se toca.
func TestWorkspaceOpenCodeKeepsUserConfigAndIsIdempotent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	agentsMD := []byte("# Reglas del equipo\n\nNo tocar.\n")
	userConfig := `{
  "$schema": "https://opencode.ai/config.json",
  "theme": "tokyonight",
  "instructions": ["CONTRIBUTING.md"],
  "mcp": {"propio": {"type": "local", "command": ["propio"]}}
}
`
	mustWriteFile(t, filepath.Join(workspace, "AGENTS.md"), agentsMD)
	mustWriteFile(t, filepath.Join(workspace, "opencode.json"), []byte(userConfig))
	before := homeSnapshotWithoutState(t, home)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentOpenCode},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentPersona, model.ComponentSDD, model.ComponentSkills},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}

	settingsPath := filepath.Join(workspace, "opencode.json")
	settings := readJSONObject(t, settingsPath)
	if settings["theme"] != "tokyonight" || settings["$schema"] != "https://opencode.ai/config.json" {
		t.Errorf("se perdieron claves del usuario: %v", settings)
	}
	if mcp, _ := settings["mcp"].(map[string]any); mcp["propio"] == nil || mcp["engram"] == nil {
		t.Errorf("mcp debe conservar el servidor del usuario y añadir engram: %v", mcp)
	}
	if got := instructionsOf(t, settingsPath); !slices.Equal(got, []string{"CONTRIBUTING.md", ".opencode/axiom.md"}) {
		t.Errorf("instructions = %v, quiero las del usuario primero y la de Axiom una sola vez", got)
	}
	if after, _ := os.ReadFile(filepath.Join(workspace, "AGENTS.md")); !bytes.Equal(after, agentsMD) {
		t.Errorf("AGENTS.md de la raíz cambió:\n%s", after)
	}
	assertNothingUnder(t, workspace, filepath.Join(".config", "opencode"))
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	// Idempotencia: un tercer sync no cambia ningún byte.
	snapshot := permissionsTreeSnapshot(t, workspace)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
}

// Un opencode.jsonc existente se usa tal cual (con sus comentarios) y no se crea
// un opencode.json al lado.
func TestWorkspaceOpenCodePrefersExistingJSONC(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	mustWriteFile(t, filepath.Join(workspace, "opencode.jsonc"), []byte("{\n  // tema del usuario\n  \"theme\": \"nord\"\n}\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertNothingUnder(t, workspace, "opencode.json")
	raw, err := os.ReadFile(filepath.Join(workspace, "opencode.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "// tema del usuario") || !strings.Contains(string(raw), "\"nord\"") {
		t.Errorf("opencode.jsonc perdió el comentario o el tema del usuario:\n%.400s", raw)
	}
	if got := instructionsOf(t, filepath.Join(workspace, "opencode.jsonc")); !slices.Equal(got, []string{".opencode/axiom.md"}) {
		t.Errorf("instructions = %v", got)
	}
}

// Un opencode.json que no se puede leer como JSON se rechaza antes de escribir
// nada: es un fichero del usuario y reescribirlo desde cero lo perdería.
func TestWorkspaceOpenCodeRefusesUnreadableSettings(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	broken := []byte("{ esto no es json")
	mustWriteFile(t, filepath.Join(workspace, "opencode.json"), broken)

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode"}, macOSDetectionResult())
	if err == nil || !strings.Contains(err.Error(), "no es JSON válido") {
		t.Fatalf("RunInstall() debe rechazar el fichero ilegible, error = %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(workspace, "opencode.json")); !bytes.Equal(after, broken) {
		t.Errorf("opencode.json se modificó:\n%s", after)
	}
	assertNothingUnder(t, workspace, ".opencode")
}

// Los restos de una instalación anterior en .config/opencode no se borran, pero se avisa.
func TestWorkspaceNoticesStaleConfigFromPreviousInstalls(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	var notices bytes.Buffer
	restore := workspaceInstructionsNoticeWriter
	workspaceInstructionsNoticeWriter = &notices
	t.Cleanup(func() { workspaceInstructionsNoticeWriter = restore })
	stale := filepath.Join(workspace, ".config", "opencode", "skills", "sdd-init", "SKILL.md")
	mustWriteFile(t, stale, []byte("de una instalación anterior\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if _, err := os.Stat(stale); err != nil {
		t.Errorf("los restos no se borran: %v", err)
	}
	if !strings.Contains(notices.String(), filepath.Join(workspace, ".config", "opencode")) {
		t.Errorf("falta el aviso de restos:\n%s", notices.String())
	}
}

// Con el ámbito global la distribución de OpenCode y Kilo no cambia.
func TestRunInstallGlobalOpenCodeAndKiloKeepGlobalLayout(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "opencode,kilocode", "--component", "persona", "--component", "sdd"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	for _, rel := range []string{
		filepath.Join(".config", "opencode", "opencode.json"),
		filepath.Join(".config", "opencode", "AGENTS.md"),
		filepath.Join(".config", "opencode", "skills", "sdd-init", "SKILL.md"),
		filepath.Join(".config", "kilo", "opencode.json"),
		filepath.Join(".config", "kilo", "AGENTS.md"),
	} {
		if _, err := os.Stat(filepath.Join(home, rel)); err != nil {
			t.Errorf("falta %s en HOME: %v", rel, err)
		}
	}
	assertNothingUnder(t, workspace, "opencode.json", ".opencode", ".kilo")
}

// Un instructions que no es una lista no se toca: es del usuario y Axiom avisa en
// lugar de sustituirlo.
func TestWorkspaceInstructionsNotAListIsLeftUntouched(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	var notices bytes.Buffer
	restore := workspaceInstructionsNoticeWriter
	workspaceInstructionsNoticeWriter = &notices
	t.Cleanup(func() { workspaceInstructionsNoticeWriter = restore })
	mustWriteFile(t, filepath.Join(workspace, "opencode.json"), []byte(`{"instructions":"CONTRIBUTING.md"}`))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if got, _ := readJSONObject(t, filepath.Join(workspace, "opencode.json"))["instructions"].(string); got != "CONTRIBUTING.md" {
		t.Errorf("instructions cambió: %v", got)
	}
	if !strings.Contains(notices.String(), "no es una lista") {
		t.Errorf("falta el aviso:\n%s", notices.String())
	}
}

// Kilo reutiliza el kilo.jsonc que ya existe en la raíz del proyecto y conserva sus
// claves y su instructions; sin AGENTS.md de la raíz tocado y sin segundo fichero.
func TestWorkspaceKiloReusesExistingConfigAndIsIdempotent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	mustWriteFile(t, filepath.Join(workspace, "kilo.jsonc"), []byte("{\n  // modelo del equipo\n  \"model\": \"kilo/auto\",\n  \"instructions\": [\"docs/reglas.md\"]\n}\n"))
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentKilocode},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentPersona, model.ComponentSDD, model.ComponentSkills},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}

	for i := 0; i < 2; i++ {
		if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
	}

	assertNothingUnder(t, workspace, filepath.Join(".kilo", "kilo.json"), filepath.Join(".config", "kilo"), "AGENTS.md")
	path := filepath.Join(workspace, "kilo.jsonc")
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "// modelo del equipo") {
		t.Errorf("kilo.jsonc perdió el comentario del usuario")
	}
	if settings := readJSONObject(t, path); settings["model"] != "kilo/auto" {
		t.Errorf("kilo.jsonc perdió el modelo: %v", settings["model"])
	}
	if got := instructionsOf(t, path); !slices.Equal(got, []string{"docs/reglas.md", ".kilo/rules/axiom.md"}) {
		t.Errorf("instructions = %v", got)
	}
}
