package agents

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// BindWorkspace liga OpenCode y Kilo a la raíz del proyecto y deja intactos al
// resto de adaptadores, cuya carpeta de configuración ya es una carpeta de punto
// en la raíz.
func TestBindWorkspaceOnlyChangesOpenCodeFamily(t *testing.T) {
	root := t.TempDir()
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		adapter, err := NewAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		bound := BindWorkspace(adapter, root)
		if _, ok := bound.(WorkspaceInstructionsProvider); !ok {
			t.Errorf("%s ligado debe registrar sus instrucciones", id)
		}
		if _, ok := bound.(WorkspacePromptsDirProvider); !ok {
			t.Errorf("%s ligado debe indicar su carpeta de prompts", id)
		}
		if bound.SkillsDir(root) == adapter.SkillsDir(root) {
			t.Errorf("%s ligado debe usar la distribución de proyecto: %s", id, bound.SkillsDir(root))
		}
	}
	claude, _ := NewAdapter(model.AgentClaudeCode)
	if BindWorkspace(claude, root) != claude {
		t.Errorf("Claude Code no cambia con el ámbito")
	}
	opencode, _ := NewAdapter(model.AgentOpenCode)
	if BindWorkspace(opencode, "") != opencode {
		t.Errorf("sin raíz no se liga nada")
	}
}

// BindWorkspace liga Gemini CLI a la raíz del proyecto (sus instrucciones van a
// .gemini/AXIOM.md, registrado en context.fileName) y deja Qwen Code como está:
// su decisión de producto sigue abierta.
func TestBindWorkspaceBindsGeminiAndNotQwen(t *testing.T) {
	root := t.TempDir()
	gemini, err := NewAdapter(model.AgentGeminiCLI)
	if err != nil {
		t.Fatal(err)
	}
	bound := BindWorkspace(gemini, root)
	if _, ok := bound.(WorkspaceContextFileProvider); !ok {
		t.Errorf("Gemini ligado debe registrar su fichero en context.fileName")
	}
	if _, ok := bound.(LegacyPromptFileProvider); !ok {
		t.Errorf("Gemini ligado debe poder retirar el .gemini/GEMINI.md antiguo")
	}
	if bound.SystemPromptFile(root) == gemini.SystemPromptFile(root) {
		t.Errorf("Gemini ligado debe usar un fichero propio: %s", bound.SystemPromptFile(root))
	}

	qwen, err := NewAdapter(model.AgentQwenCode)
	if err != nil {
		t.Fatal(err)
	}
	if BindWorkspace(qwen, root) != qwen {
		t.Errorf("Qwen Code no cambia con el ámbito")
	}
}

// BindWorkspace liga Codex a la raíz del proyecto: sus instrucciones salen de un
// fichero propio que se copia a developer_instructions y sus skills van a
// .agents/skills, no a ~/.codex.
func TestBindWorkspaceBindsCodex(t *testing.T) {
	root := t.TempDir()
	codex, err := NewAdapter(model.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	bound := BindWorkspace(codex, root)
	if _, ok := bound.(WorkspaceInlineInstructionsProvider); !ok {
		t.Errorf("Codex ligado debe registrar sus instrucciones en developer_instructions")
	}
	if _, ok := bound.(LegacyPromptFileProvider); !ok {
		t.Errorf("Codex ligado debe poder retirar el .codex/AGENTS.md antiguo")
	}
	if bound.SystemPromptFile(root) == codex.SystemPromptFile(root) {
		t.Errorf("Codex ligado debe usar un fichero propio: %s", bound.SystemPromptFile(root))
	}
	if bound.SkillsDir(root) == codex.SkillsDir(root) {
		t.Errorf("Codex ligado debe usar .agents/skills: %s", bound.SkillsDir(root))
	}
	if _, ok := codex.(WorkspaceInlineInstructionsProvider); ok {
		t.Errorf("Codex sin ligar no debe ofrecer el registro de instrucciones")
	}
	if BindWorkspace(codex, "") != codex {
		t.Errorf("sin raíz no se liga nada")
	}
}

func TestFactoryResolvesPiAdapter(t *testing.T) {
	adapter, err := NewAdapter(model.AgentPi)
	if err != nil {
		t.Fatalf("NewAdapter(%q) returned error: %v", model.AgentPi, err)
	}

	if got := adapter.Agent(); got != model.AgentPi {
		t.Fatalf("adapter.Agent() = %q, want %q", got, model.AgentPi)
	}
}

func TestDefaultRegistryIncludesPi(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() returned error: %v", err)
	}

	adapter, ok := registry.Get(model.AgentPi)
	if !ok {
		t.Fatalf("registry missing %s adapter", model.AgentPi)
	}

	if got := adapter.Agent(); got != model.AgentPi {
		t.Fatalf("registry adapter.Agent() = %q, want %q", got, model.AgentPi)
	}
}

func TestFactoryResolvesAntigravityAdapter(t *testing.T) {
	adapter, err := NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(%q) returned error: %v", model.AgentAntigravity, err)
	}

	if got := adapter.Agent(); got != model.AgentAntigravity {
		t.Fatalf("adapter.Agent() = %q, want %q", got, model.AgentAntigravity)
	}
}

func TestDefaultRegistryIncludesAntigravity(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() returned error: %v", err)
	}

	adapter, ok := registry.Get(model.AgentAntigravity)
	if !ok {
		t.Fatalf("registry missing %s adapter", model.AgentAntigravity)
	}

	if got := adapter.Agent(); got != model.AgentAntigravity {
		t.Fatalf("registry adapter.Agent() = %q, want %q", got, model.AgentAntigravity)
	}
}

func TestDefaultRegistrySupportedAgentsMatchesFactoryAgents(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() returned error: %v", err)
	}

	want := []model.AgentID{
		model.AgentAntigravity,
		model.AgentClaudeCode,
		model.AgentCodex,
		model.AgentCursor,
		model.AgentGeminiCLI,
		model.AgentGitHubCopilotCLI,
		model.AgentHermes,
		model.AgentKilocode,
		model.AgentKimi,
		model.AgentKiroIDE,
		model.AgentOpenClaw,
		model.AgentOpenCode,
		model.AgentPi,
		model.AgentQwenCode,
		model.AgentTrae,
		model.AgentVSCodeCopilot,
		model.AgentWindsurf,
	}

	if got := registry.SupportedAgents(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedAgents() = %v, want %v", got, want)
	}
}

func TestFactoryResolvesHermesAdapter(t *testing.T) {
	adapter, err := NewAdapter(model.AgentHermes)
	if err != nil {
		t.Fatalf("NewAdapter(%q) returned error: %v", model.AgentHermes, err)
	}

	if got := adapter.Agent(); got != model.AgentHermes {
		t.Fatalf("adapter.Agent() = %q, want %q", got, model.AgentHermes)
	}
}

func TestDefaultRegistryIncludesHermes(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() returned error: %v", err)
	}

	adapter, ok := registry.Get(model.AgentHermes)
	if !ok {
		t.Fatalf("registry missing %s adapter", model.AgentHermes)
	}

	if got := adapter.Agent(); got != model.AgentHermes {
		t.Fatalf("registry adapter.Agent() = %q, want %q", got, model.AgentHermes)
	}
}

func TestFactoryRejectsUnsupportedOpenClawLookalike(t *testing.T) {
	_, err := NewAdapter(model.AgentID("openclaw-beta"))
	if err == nil {
		t.Fatalf("NewAdapter() expected unsupported agent error")
	}

	if !errors.Is(err, ErrAgentNotSupported) {
		t.Fatalf("NewAdapter() error = %v, want ErrAgentNotSupported", err)
	}
}

// BindWorkspace liga VS Code Copilot a la raíz del proyecto: instrucciones y skills
// van a .github/ y el MCP a .vscode/mcp.json, no al perfil de usuario de VS Code.
func TestBindWorkspaceBindsVSCode(t *testing.T) {
	root := t.TempDir()
	vscode, err := NewAdapter(model.AgentVSCodeCopilot)
	if err != nil {
		t.Fatal(err)
	}
	bound := BindWorkspace(vscode, root)
	if _, ok := bound.(WorkspaceUserConfigFilesProvider); !ok {
		t.Errorf("VS Code ligado debe indicar .vscode/mcp.json como fichero de usuario")
	}
	if scoped, ok := bound.(WorkspaceScopedProvider); !ok || !scoped.InWorkspace(root) {
		t.Errorf("VS Code ligado debe saber que está ligado a la raíz")
	}
	for name, got := range map[string]string{
		"SystemPromptFile": bound.SystemPromptFile(root),
		"SkillsDir":        bound.SkillsDir(root),
		"MCPConfigPath":    bound.MCPConfigPath(root, "engram"),
	} {
		if got == "" || strings.HasPrefix(got, filepath.Join(root, "AppData")) || strings.Contains(got, ".copilot") {
			t.Errorf("%s ligado debe usar la distribución de proyecto: %s", name, got)
		}
	}
	if _, ok := vscode.(WorkspaceUserConfigFilesProvider); ok {
		t.Errorf("VS Code sin ligar no debe ofrecer ficheros de proyecto")
	}
	if BindWorkspace(vscode, "") != vscode {
		t.Errorf("sin raíz no se liga nada")
	}
}
