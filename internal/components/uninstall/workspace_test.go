package uninstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func jsonUnmarshalForTest(text string, target any) error {
	return json.Unmarshal([]byte(text), target)
}

// assertJSONEqual compara dos documentos JSON por su contenido, sin depender de la
// sangría.
func assertJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("got no es JSON válido: %v\n%s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want no es JSON válido: %v\n%s", err, want)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON distinto:\ngot  %s\nwant %s", got, want)
	}
}

func TestIsManagedMCPServerRecognizesOnlyTheShapesAxiomWrites(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"engram", `{"command":"engram","args":["mcp","--tools=agent"]}`, true},
		{"engram", `{"command":"C:\\Users\\yo\\AppData\\Local\\engram\\bin\\engram.exe","args":["mcp","--tools=agent"]}`, true},
		{"engram", `{"type":"local","command":["/usr/local/bin/engram","mcp","--tools=agent"]}`, true},
		{"engram", `{"command":"node","args":["engram-propio.js"]}`, false},
		{"engram", `{"command":"engram","args":["serve"]}`, false},
		{"context7", `{"command":"npx","args":["-y","--package=@upstash/context7-mcp@1.2.3","--","context7-mcp"]}`, true},
		{"context7", `{"type":"remote","url":"https://mcp.context7.com/mcp","enabled":true}`, true},
		{"context7", `{"type":"http","url":"https://mcp.context7.com/mcp"}`, true},
		{"context7", `{"serverUrl":"https://mcp.context7.com/mcp"}`, true},
		{"context7", `{"type":"http","url":"https://contexto.interno.example/mcp"}`, false},
		{"context7", `{"command":"npx","args":["mi-paquete"]}`, false},
		{"otro", `{"command":"engram","args":["mcp"]}`, false},
	}
	for _, tc := range cases {
		var value any
		if err := jsonUnmarshalForTest(tc.value, &value); err != nil {
			t.Fatal(err)
		}
		if got := isManagedMCPServer(tc.name, value); got != tc.want {
			t.Errorf("isManagedMCPServer(%q, %s) = %v, want %v", tc.name, tc.value, got, tc.want)
		}
	}
}

func TestWorkspaceMCPRemovalKeepsAUserServerWithTheSameNameAndWarns(t *testing.T) {
	workspace := t.TempDir()
	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(workspace, ".mcp.json")
	seed := `{"mcpServers":{"context7":{"type":"http","url":"https://contexto.interno.example/mcp"},"engram":{"command":"engram","args":["mcp","--tools=agent"]},"mio":{"command":"node"}}}`
	if err := os.WriteFile(mcpPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter, _ := svc.adapter(model.AgentClaudeCode)

	for _, name := range []string{"context7", "engram"} {
		_, ops := svc.workspaceMCPOperations(adapter, name)
		for _, op := range ops {
			if _, _, err := op.apply(op.path); err != nil {
				t.Fatalf("operation on %q error = %v", op.path, err)
			}
		}
	}

	servers := readJSONFileForTest(t, mcpPath)["mcpServers"].(map[string]any)
	if _, kept := servers["context7"]; !kept {
		t.Errorf("el context7 del usuario debe conservarse: %#v", servers)
	}
	if _, gone := servers["engram"]; gone {
		t.Errorf("el engram de Axiom debe retirarse: %#v", servers)
	}
	if _, kept := servers["mio"]; !kept {
		t.Errorf("el servidor del usuario debe conservarse: %#v", servers)
	}
	if len(svc.notices) != 1 || !strings.Contains(svc.notices[0], "context7") {
		t.Errorf("notices = %v, want un aviso sobre context7", svc.notices)
	}
}

func TestCleanCodexWorkspaceTOMLRetiresOnlyWhatAxiomWrites(t *testing.T) {
	content := strings.Join([]string{
		`model = "modelo-del-equipo"`,
		`experimental_compact_prompt_file = "engram-compact-prompt.md"`,
		``,
		`[features]`,
		`multi_agent = true`,
		``,
		`[agents]`,
		`max_threads = 8`,
		`max_depth = 2`,
		``,
		`[mcp_servers.engram]`,
		`command = "engram"`,
		`args = ["mcp", "--tools=agent"]`,
		``,
		`[mcp_servers.context7]`,
		`url = "https://mcp.context7.com/mcp"`,
		``,
		`[mcp_servers.mio]`,
		`command = "node"`,
		``,
	}, "\n")

	engram, changed := cleanCodexWorkspaceTOML(content, "engram")
	if !changed {
		t.Fatal("el componente engram debe cambiar el fichero")
	}
	for _, gone := range []string{"experimental_compact_prompt_file", "[features]", "multi_agent", "[mcp_servers.engram]", "max_depth"} {
		if strings.Contains(engram, gone) {
			t.Errorf("engram debe retirar %q:\n%s", gone, engram)
		}
	}
	// max_threads vale 8: es del usuario, así que la tabla [agents] y esa clave se quedan.
	for _, kept := range []string{`model = "modelo-del-equipo"`, "[agents]", "max_threads = 8", "[mcp_servers.context7]", "[mcp_servers.mio]"} {
		if !strings.Contains(engram, kept) {
			t.Errorf("engram debe conservar %q:\n%s", kept, engram)
		}
	}

	both, _ := cleanCodexWorkspaceTOML(engram, "context7")
	if strings.Contains(both, "context7") || !strings.Contains(both, "[mcp_servers.mio]") {
		t.Errorf("context7 debe retirar solo su tabla:\n%s", both)
	}
}

func TestCleanCodexWorkspaceTOMLKeepsAUserCompactPromptAndForeignServers(t *testing.T) {
	content := "experimental_compact_prompt_file = \"mi-prompt.md\"\n\n[mcp_servers.engram]\ncommand = \"node\"\nargs = [\"mi-engram.js\"]\n"
	got, changed := cleanCodexWorkspaceTOML(content, "engram")
	if changed || got != content {
		t.Fatalf("no debe tocar lo que no tiene la forma de Axiom (changed=%v):\n%s", changed, got)
	}
}

func TestCleanCodexWorkspaceTOMLRemovesTheKeyEvenWithBracketLinesInAMultilineValue(t *testing.T) {
	content := "developer_instructions = '''\nTexto\n[no es una tabla]\n'''\nexperimental_compact_prompt_file = \"engram-compact-prompt.md\"\n\n[mcp_servers.engram]\ncommand = \"engram\"\nargs = [\"mcp\", \"--tools=agent\"]\n"
	got, _ := cleanCodexWorkspaceTOML(content, "engram")
	if strings.Contains(got, "experimental_compact_prompt_file") {
		t.Fatalf("la clave debe retirarse aunque un valor multilínea tenga líneas entre corchetes:\n%s", got)
	}
	if !strings.Contains(got, "[no es una tabla]") {
		t.Fatalf("el valor multilínea debe conservarse:\n%s", got)
	}
}

func TestRemoveCodexDeveloperInstructionsBlock(t *testing.T) {
	block := "<!-- axiom:workspace-instructions -->\ninstrucciones de Axiom\n<!-- /axiom:workspace-instructions -->"

	t.Run("conserva el texto del usuario", func(t *testing.T) {
		content := "developer_instructions = \"\"\"Texto del equipo\n\n" + block + "\"\"\"\nmodel = \"x\"\n"
		got, changed := removeCodexDeveloperInstructionsBlock(content, "developer_instructions")
		if !changed || strings.Contains(got, "axiom:") || !strings.Contains(got, "Texto del equipo") || !strings.Contains(got, `model = "x"`) {
			t.Fatalf("changed=%v:\n%s", changed, got)
		}
	})

	t.Run("retira la clave si solo había el bloque", func(t *testing.T) {
		content := "model = \"x\"\ndeveloper_instructions = \"\"\"" + block + "\"\"\"\n"
		got, changed := removeCodexDeveloperInstructionsBlock(content, "developer_instructions")
		if !changed || strings.Contains(got, "developer_instructions") || !strings.Contains(got, `model = "x"`) {
			t.Fatalf("changed=%v:\n%s", changed, got)
		}
	})

	t.Run("sin bloque no toca nada", func(t *testing.T) {
		content := "developer_instructions = \"Texto del equipo\"\n"
		got, changed := removeCodexDeveloperInstructionsBlock(content, "developer_instructions")
		if changed || got != content {
			t.Fatalf("changed=%v:\n%s", changed, got)
		}
	})
}

func TestContextFileNameRemoval(t *testing.T) {
	cases := []struct {
		name     string
		seed     string
		wantJSON string
		changed  bool
	}{
		{
			name:     "solo quedan los nombres por defecto: la clave era de Axiom",
			seed:     `{"theme":"x","context":{"fileName":["GEMINI.md",".gemini/AXIOM.md"]}}`,
			wantJSON: `{"theme":"x"}`,
			changed:  true,
		},
		{
			name:     "conserva la lista del usuario",
			seed:     `{"context":{"fileName":["GEMINI.md","CONTRIBUTING.md",".gemini/AXIOM.md"],"loadMemoryFromIncludeDirectories":true}}`,
			wantJSON: `{"context":{"fileName":["GEMINI.md","CONTRIBUTING.md"],"loadMemoryFromIncludeDirectories":true}}`,
			changed:  true,
		},
		{
			name:     "sin la entrada de Axiom no cambia",
			seed:     `{"context":{"fileName":["GEMINI.md","CONTRIBUTING.md"]}}`,
			wantJSON: `{"context":{"fileName":["GEMINI.md","CONTRIBUTING.md"]}}`,
			changed:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := contextFileNameRemoval(".gemini/AXIOM.md", []string{"GEMINI.md"})([]byte(tc.seed))
			if err != nil {
				t.Fatal(err)
			}
			if changed != tc.changed {
				t.Fatalf("changed = %v, want %v", changed, tc.changed)
			}
			assertJSONEqual(t, string(got), tc.wantJSON)
		})
	}
}

func TestInstructionsEntryRemoval(t *testing.T) {
	remove := instructionsEntryRemoval(".opencode/axiom.md")

	got, changed, err := remove([]byte(`{"instructions":["docs/reglas.md",".opencode/axiom.md"],"theme":"x"}`))
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	assertJSONEqual(t, string(got), `{"instructions":["docs/reglas.md"],"theme":"x"}`)

	got, changed, err = remove([]byte(`{"instructions":[".opencode/axiom.md"],"theme":"x"}`))
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	assertJSONEqual(t, string(got), `{"theme":"x"}`)

	if _, changed, _ := remove([]byte(`{"instructions":["docs/reglas.md"]}`)); changed {
		t.Fatal("sin la entrada de Axiom no debe cambiar")
	}
}

func TestRemoveManagedAgentEntriesKeepsTheUsersAgents(t *testing.T) {
	seed := `{"agent":{"axiom-orchestrator":{"__managed_by":"axiom/sdd"},"review-risk":{"__managed_by":"axiom/sdd"},"mio":{"mode":"subagent"}},"share":"disabled"}`
	got, changed, err := removeManagedAgentEntries([]byte(seed))
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	assertJSONEqual(t, string(got), `{"agent":{"mio":{"mode":"subagent"}},"share":"disabled"}`)

	got, _, err = removeDefaultOpenCodeShare(got)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(got), `{"agent":{"mio":{"mode":"subagent"}}}`)

	if _, changed, _ := removeDefaultOpenCodeShare([]byte(`{"share":"manual"}`)); changed {
		t.Fatal("un modo de compartición elegido por el usuario no se toca")
	}
}

func TestPersonaThatAgentsWithFileReplaceReceiveWithoutMarkersIsRecognized(t *testing.T) {
	persona := assets.MustRead("generic/persona-axiom.md")
	content := strings.TrimRight(persona, "\n") + "\n\n<!-- axiom:sdd-orchestrator -->\norquestador\n<!-- /axiom:sdd-orchestrator -->\n"

	updated, sectionsChanged := removeMarkdownSections(content, "persona")
	if sectionsChanged {
		t.Fatalf("no hay sección persona con marcadores")
	}
	updated, personaChanged := removeManagedPersonaPreamble(updated)
	if !personaChanged || strings.Contains(updated, "## Rules") {
		t.Fatalf("la persona sin marcadores de Axiom debe retirarse:\n%s", updated)
	}
	if !strings.Contains(updated, "orquestador") {
		t.Fatalf("las secciones con marcadores no se tocan aquí:\n%s", updated)
	}

	// Un preámbulo del usuario que no es una persona de Axiom se conserva.
	own := "## Rules\n\n- Mi regla.\n\n<!-- axiom:sdd-orchestrator -->\nx\n<!-- /axiom:sdd-orchestrator -->\n"
	if _, changed := removeManagedPersonaPreamble(own); changed {
		t.Fatal("un preámbulo propio del usuario no es una persona de Axiom")
	}
}

func TestWorkspaceDirectoriesThatKeepTeamContentAreNotReported(t *testing.T) {
	workspace := t.TempDir()
	teamFile := filepath.Join(workspace, ".claude", "agents", "equipo.md")
	if err := os.MkdirAll(filepath.Dir(teamFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(teamFile, []byte("agente del equipo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.PartialUninstall([]model.AgentID{model.AgentClaudeCode}, nil)
	if err != nil {
		t.Fatalf("PartialUninstall() error = %v", err)
	}
	for _, action := range result.ManualActions {
		if strings.Contains(action, "non-managed files") {
			t.Errorf("una carpeta con contenido del equipo no debe dar aviso: %s", action)
		}
	}
	if _, err := os.Stat(teamFile); err != nil {
		t.Fatalf("el fichero del equipo debe seguir existiendo: %v", err)
	}
}

func TestWorkspaceUninstallRemovesTheGeneratedSkillRegistryAndKeepsTheUsersAtlSkills(t *testing.T) {
	workspace := t.TempDir()
	for rel, content := range map[string]string{
		filepath.Join(".atl", "skill-registry.md"):            "registro\n",
		filepath.Join(".atl", ".skill-registry.cache.json"):   "{}",
		filepath.Join(".atl", "skills", "propia", "SKILL.md"): "skill propia\n",
		"AGENTS.md": "# Equipo\n\n<!-- axiom:skills-index -->\n## Skills\n<!-- /axiom:skills-index -->\n",
	} {
		path := filepath.Join(workspace, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentClaudeCode}, nil); err != nil {
		t.Fatalf("PartialUninstall() error = %v", err)
	}

	for _, gone := range []string{".skill-registry.cache.json", "skill-registry.md"} {
		if _, err := os.Stat(filepath.Join(workspace, ".atl", gone)); !os.IsNotExist(err) {
			t.Errorf(".atl/%s debe retirarse; stat = %v", gone, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace, ".atl", "skills", "propia", "SKILL.md")); err != nil {
		t.Errorf("la skill propia de .atl/skills debe conservarse: %v", err)
	}
	agentsMD, err := os.ReadFile(filepath.Join(workspace, "AGENTS.md"))
	if err != nil || strings.Contains(string(agentsMD), "axiom:") || !strings.Contains(string(agentsMD), "# Equipo") {
		t.Errorf("AGENTS.md debe conservar lo del equipo sin el índice de Axiom: %v\n%s", err, agentsMD)
	}
}

func TestWorkspaceUninstallRefusesToRunWithoutAProjectRoot(t *testing.T) {
	if _, err := NewScopedService(t.TempDir(), "", "dev", Options{Scope: ScopeWorkspace}); err == nil {
		t.Fatal("el ámbito workspace sin raíz debe fallar")
	}
}

func TestSkillsDirStaysInUseWhileAnotherAgentOfTheProjectUsesIt(t *testing.T) {
	workspace := t.TempDir()
	statePath := filepath.Join(workspace, ".axiom", "state.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte(`{"installed_agents":["codex","opencode","claude-code"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	svc.removing = []model.AgentID{model.AgentCodex}

	shared := filepath.Join(workspace, ".agents", "skills")
	if !svc.skillsDirStaysInUse(shared) {
		t.Error(".agents/skills sigue en uso por OpenCode")
	}
	if !svc.skillsDirStaysInUse(filepath.Join(workspace, ".claude", "skills")) {
		t.Error(".claude/skills sigue en uso por Claude, que permanece instalado")
	}

	svc.removing = []model.AgentID{model.AgentCodex, model.AgentOpenCode}
	if svc.skillsDirStaysInUse(shared) {
		t.Error("sin otros agentes que la usen, .agents/skills no está en uso")
	}
	if !slices.Contains(svc.otherAgentsInProject(), model.AgentClaudeCode) {
		t.Errorf("otherAgentsInProject() = %v, want claude-code", svc.otherAgentsInProject())
	}
}
