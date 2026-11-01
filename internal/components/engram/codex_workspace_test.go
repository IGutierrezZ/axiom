package engram

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/codex"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func boundCodex(root string) agents.Adapter { return codex.NewAdapter().WithWorkspace(root) }

func readCodexTOML(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if _, err := toml.Decode(string(raw), &parsed); err != nil {
		t.Fatalf("%s no es TOML válido: %v\n%s", path, err, raw)
	}
	return parsed
}

// En un proyecto Engram escribe el MCP y las claves de multiagente en
// .codex/config.toml, el prompt de compactación con ruta relativa a .codex y el
// protocolo de memoria en el fichero de instrucciones de Axiom. No escribe
// model_instructions_file (sustituye las instrucciones base), ni los ficheros de
// perfil (Codex ignora profiles en proyecto), ni nada en una carpeta de usuario.
func TestInjectWorkspaceCodexWritesProjectConfig(t *testing.T) {
	validCodexRuntime(t)
	root := t.TempDir()

	result, err := InjectWorkspaceWithOptions(root, boundCodex(root), InjectOptions{})
	if err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}

	configPath := filepath.Join(root, ".codex", "config.toml")
	config := readCodexTOML(t, configPath)
	servers, _ := config["mcp_servers"].(map[string]any)
	if servers["engram"] == nil {
		t.Errorf("falta mcp_servers.engram: %v", config)
	}
	if features, _ := config["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("falta features.multi_agent = true")
	}
	for _, key := range []string{"model_instructions_file", "profiles", "profile", "model_provider", "model_providers", "notify", "otel"} {
		if _, written := config[key]; written {
			t.Errorf("no debe escribirse %s en la configuración de proyecto", key)
		}
	}
	if config["experimental_compact_prompt_file"] != "engram-compact-prompt.md" {
		t.Errorf("experimental_compact_prompt_file = %v", config["experimental_compact_prompt_file"])
	}
	compact, err := os.ReadFile(filepath.Join(root, ".codex", "engram-compact-prompt.md"))
	if err != nil || len(bytes.TrimSpace(compact)) == 0 {
		t.Errorf("falta el prompt de compactación: %v", err)
	}
	for _, rel := range []string{"engram-instructions.md", "sdd-strong.config.toml", "sdd-mid.config.toml", "sdd-cheap.config.toml"} {
		if _, statErr := os.Stat(filepath.Join(root, ".codex", rel)); statErr == nil {
			t.Errorf("no debe escribirse .codex/%s en un proyecto", rel)
		}
	}
	if _, statErr := os.Stat(filepath.Join(root, ".codex", "AGENTS.md")); statErr == nil {
		t.Errorf("las instrucciones no pueden ir a .codex/AGENTS.md")
	}

	source, err := os.ReadFile(filepath.Join(root, ".codex", "axiom-instructions.md"))
	if err != nil {
		t.Fatalf("falta el fichero de instrucciones de Axiom: %v", err)
	}
	for _, section := range []string{"<!-- axiom:engram-protocol -->", "<!-- axiom:engram-passive-capture -->"} {
		if !strings.Contains(string(source), section) {
			t.Errorf("el fichero de instrucciones no contiene %s", section)
		}
	}
	if !result.Changed {
		t.Errorf("la primera inyección debe reportar cambios")
	}
	for _, want := range []string{configPath, filepath.Join(root, ".codex", "engram-compact-prompt.md"), filepath.Join(root, ".codex", "axiom-instructions.md")} {
		found := false
		for _, file := range result.Files {
			found = found || file == want
		}
		if !found {
			t.Errorf("result.Files no incluye %s: %v", want, result.Files)
		}
	}
}

// Inyectar dos veces deja los mismos bytes.
func TestInjectWorkspaceCodexIsIdempotent(t *testing.T) {
	validCodexRuntime(t)
	root := t.TempDir()
	opts := InjectOptions{CodexOrchestratorAssignment: model.CodexPresetOrchestratorAssignment(string(model.CodexPresetRecommended))}

	if _, err := InjectWorkspaceWithOptions(root, boundCodex(root), opts); err != nil {
		t.Fatal(err)
	}
	snapshot := map[string][]byte{}
	for _, rel := range []string{"config.toml", "engram-compact-prompt.md", "axiom-instructions.md"} {
		raw, _ := os.ReadFile(filepath.Join(root, ".codex", rel))
		snapshot[rel] = raw
	}
	second, err := InjectWorkspaceWithOptions(root, boundCodex(root), opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Errorf("la segunda inyección no debe reportar cambios")
	}
	for rel, want := range snapshot {
		got, _ := os.ReadFile(filepath.Join(root, ".codex", rel))
		if !bytes.Equal(got, want) {
			t.Errorf(".codex/%s cambió en la segunda inyección", rel)
		}
	}
	config := readCodexTOML(t, filepath.Join(root, ".codex", "config.toml"))
	if config["model"] != "gpt-6-sol" || config["model_reasoning_effort"] != "medium" {
		t.Errorf("la asignación del orquestador debe escribirse como valores de raíz: %v", config)
	}
}

// El resto de la configuración del usuario, incluido su texto en developer_instructions,
// se conserva, y una línea de ese texto que parezca una cabecera no se confunde con
// una tabla.
func TestInjectWorkspaceCodexKeepsUserConfig(t *testing.T) {
	validCodexRuntime(t)
	root := t.TempDir()
	configPath := filepath.Join(root, ".codex", "config.toml")
	userText := "Notas.\n[features]\nmulti_agent = false\n"
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	user := "# del usuario\nmodel = \"x\"\ndeveloper_instructions = '''\n" + userText + "'''\n\n[mcp_servers.propio]\ncommand = \"propio\"\n"
	if err := os.WriteFile(configPath, []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if _, err := InjectWorkspaceWithOptions(root, boundCodex(root), InjectOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	raw, _ := os.ReadFile(configPath)
	if !strings.Contains(string(raw), "# del usuario") {
		t.Errorf("se perdió el comentario del usuario:\n%s", raw)
	}
	config := readCodexTOML(t, configPath)
	if config["developer_instructions"] != userText {
		t.Errorf("developer_instructions = %q, quiero el texto del usuario intacto", config["developer_instructions"])
	}
	if config["model"] != "x" {
		t.Errorf("model = %v", config["model"])
	}
	servers, _ := config["mcp_servers"].(map[string]any)
	if servers["propio"] == nil || servers["engram"] == nil {
		t.Errorf("mcp_servers = %v", servers)
	}
	if features, _ := config["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("features.multi_agent = %v", features["multi_agent"])
	}
}

// Un experimental_compact_prompt_file del usuario se respeta y se avisa.
func TestInjectWorkspaceCodexKeepsUserCompactPrompt(t *testing.T) {
	validCodexRuntime(t)
	root := t.TempDir()
	var notices bytes.Buffer
	restore := CodexWorkspaceNoticeWriter
	CodexWorkspaceNoticeWriter = &notices
	t.Cleanup(func() { CodexWorkspaceNoticeWriter = restore })
	configPath := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("experimental_compact_prompt_file = \"mio.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InjectWorkspaceWithOptions(root, boundCodex(root), InjectOptions{}); err != nil {
		t.Fatal(err)
	}

	if got := readCodexTOML(t, configPath)["experimental_compact_prompt_file"]; got != "mio.md" {
		t.Errorf("experimental_compact_prompt_file = %v, quiero el del usuario", got)
	}
	if !strings.Contains(notices.String(), "experimental_compact_prompt_file") {
		t.Errorf("falta el aviso:\n%s", notices.String())
	}
}

// Un config.toml que no es TOML válido se rechaza sin tocar nada.
func TestInjectWorkspaceCodexRefusesInvalidToml(t *testing.T) {
	validCodexRuntime(t)
	root := t.TempDir()
	configPath := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	broken := []byte("model = \n[no es toml")
	if err := os.WriteFile(configPath, broken, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := InjectWorkspaceWithOptions(root, boundCodex(root), InjectOptions{})
	if err == nil || !strings.Contains(err.Error(), "no es TOML válido") {
		t.Fatalf("error = %v, se esperaba el rechazo del TOML inválido", err)
	}
	if after, _ := os.ReadFile(configPath); !bytes.Equal(after, broken) {
		t.Errorf("config.toml se modificó:\n%s", after)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".codex", "engram-compact-prompt.md")); statErr == nil {
		t.Errorf("no debe escribirse nada si el TOML es inválido")
	}
}

// El ejecutable de Codex no interviene en un proyecto: un CLI incompatible no
// bloquea la instalación, porque no hay perfiles que escribir.
func TestInjectWorkspaceCodexIgnoresCodexRuntime(t *testing.T) {
	root := t.TempDir()

	if _, err := InjectWorkspaceWithOptions(root, boundCodex(root), InjectOptions{}); err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); err != nil {
		t.Errorf("falta .codex/config.toml: %v", err)
	}
}

// La instalación global no cambia: sigue con model_instructions_file, ficheros de
// perfil y sin fichero propio de instrucciones.
func TestInjectGlobalCodexKeepsGlobalLayout(t *testing.T) {
	validCodexRuntime(t)
	home := t.TempDir()

	if _, err := InjectWithOptions(home, codexAdapter(), InjectOptions{}); err != nil {
		t.Fatal(err)
	}

	config := readCodexTOML(t, filepath.Join(home, ".codex", "config.toml"))
	if config["model_instructions_file"] == nil {
		t.Errorf("la instalación global debe escribir model_instructions_file")
	}
	if _, declared := config["developer_instructions"]; declared {
		t.Errorf("la instalación global no debe escribir developer_instructions")
	}
	for _, rel := range []string{"engram-instructions.md", "sdd-strong.config.toml"} {
		if _, err := os.Stat(filepath.Join(home, ".codex", rel)); err != nil {
			t.Errorf("falta ~/.codex/%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "axiom-instructions.md")); err == nil {
		t.Errorf("la instalación global no usa el fichero de instrucciones de proyecto")
	}
	agentsMD, _ := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if strings.Contains(string(agentsMD), "engram-passive-capture") {
		t.Errorf("la captura pasiva global va en engram-instructions.md, no en AGENTS.md")
	}
}
