package uninstall

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// TestPartialUninstallGlobalKeepsTheUserEngramRegistrationUnlessNamed reproduce
// `axiom uninstall --scope global --agent claude-code` sin --components: retiraba
// mcpServers.engram de ~/.claude.json y, con un .mcp.json de proyecto sin aprobar,
// las sesiones arrancaban sin Engram. Engram es huella de máquina: su registro de
// usuario solo se retira si se nombra (`--components engram`) o con `--all`. La
// sección de protocolo del prompt global se retira en ambos casos.
func TestPartialUninstallGlobalKeepsTheUserEngramRegistrationUnlessNamed(t *testing.T) {
	cases := []struct {
		name       string
		components []model.ComponentID
		wantKept   bool
	}{
		{name: "sin --components conserva el registro", components: nil, wantKept: true},
		{name: "--components engram lo retira", components: []model.ComponentID{model.ComponentEngram}, wantKept: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
			registryPath := claude.UserConfigPath(home)
			writeGlobalFile(t, registryPath, `{"mcpServers":{"engram":{"command":"engram","args":["mcp","--tools=agent"]},"codegraph":{"command":"codegraph"}}}`)
			claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
			writeGlobalFile(t, claudeMD, filemerge.InjectMarkdownSection("# Usuario\n", "engram-protocol", "protocolo de Engram"))

			result, err := newGlobalService(t, home, Options{}).PartialUninstall([]model.AgentID{model.AgentClaudeCode}, tc.components)
			if err != nil {
				t.Fatalf("PartialUninstall() error = %v", err)
			}

			servers, _ := readJSONFileForTest(t, registryPath)["mcpServers"].(map[string]any)
			if _, kept := servers["engram"]; kept != tc.wantKept {
				t.Fatalf("mcpServers.engram kept = %v, want %v: %#v", kept, tc.wantKept, servers)
			}
			if _, kept := servers["codegraph"]; !kept {
				t.Fatalf("mcpServers.codegraph del usuario perdido: %#v", servers)
			}
			if prompt := readGlobalFile(t, claudeMD); strings.Contains(prompt, "protocolo de Engram") {
				t.Fatalf("CLAUDE.md conserva la sección engram-protocol:\n%s", prompt)
			}
			noted := slices.ContainsFunc(result.ManualActions, func(action string) bool {
				return strings.Contains(action, "Engram MCP registration") && strings.Contains(action, registryPath)
			})
			if noted != tc.wantKept {
				t.Fatalf("ManualActions = %v, want kept-registration note = %v", result.ManualActions, tc.wantKept)
			}
		})
	}
}

// TestPartialUninstallGlobalCodexKeepsTheEngramServerAndRetiresItsInstructions
// comprueba el registro de Codex, que comparte config.toml con lo demás que
// escribe el componente Engram: sin --components se conserva [mcp_servers.engram]
// y se retiran las claves de instrucciones y sus ficheros.
func TestPartialUninstallGlobalCodexKeepsTheEngramServerAndRetiresItsInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	codexDir := filepath.Join(home, ".codex")
	configPath := filepath.Join(codexDir, "config.toml")
	instructions := filepath.Join(codexDir, "engram-instructions.md")
	compact := filepath.Join(codexDir, "engram-compact-prompt.md")
	writeGlobalFile(t, configPath, strings.Join([]string{
		`model_instructions_file = "` + filepath.ToSlash(instructions) + `"`,
		`experimental_compact_prompt_file = "` + filepath.ToSlash(compact) + `"`,
		`custom_top = "keep"`,
		``,
		`[mcp_servers.engram]`,
		`command = "engram"`,
		`args = ["mcp", "--tools=agent"]`,
		``,
	}, "\n"))
	writeGlobalFile(t, instructions, "instrucciones")
	writeGlobalFile(t, compact, "compactación")

	svc := newGlobalService(t, home, Options{})
	adapter, ok := svc.registry.Get(model.AgentCodex)
	if !ok {
		t.Fatal("Codex adapter not found in registry")
	}
	if got := adapter.MCPConfigPath(home, "engram"); got != configPath {
		t.Fatalf("Codex MCPConfigPath = %q, want %q", got, configPath)
	}

	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentCodex}, nil); err != nil {
		t.Fatalf("PartialUninstall() error = %v", err)
	}

	config := readGlobalFile(t, configPath)
	if !strings.Contains(config, "[mcp_servers.engram]") {
		t.Fatalf("config.toml perdió [mcp_servers.engram]:\n%s", config)
	}
	if strings.Contains(config, "model_instructions_file") || strings.Contains(config, "experimental_compact_prompt_file") {
		t.Fatalf("config.toml conserva las claves de instrucciones de Engram:\n%s", config)
	}
	if !strings.Contains(config, `custom_top = "keep"`) {
		t.Fatalf("config.toml perdió la clave del usuario:\n%s", config)
	}
	for _, path := range []string{instructions, compact} {
		if pathExists(path) {
			t.Fatalf("%s sigue existiendo, want retirado", path)
		}
	}
}
