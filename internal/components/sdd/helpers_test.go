package sdd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// agentForAssetPath maps an embedded asset to the runtime identity that
// installs it, taken from its family directory. Rendering is per-runtime, so
// no test may render an asset without naming the runtime it is rendering for.
func agentForAssetPath(t *testing.T, path string) model.AgentID {
	t.Helper()

	family, _, ok := strings.Cut(path, "/")
	if !ok {
		t.Fatalf("asset path %q has no runtime family directory", path)
	}
	// copilot/agents lo comparten VS Code Copilot y Copilot CLI, que lo renderizan
	// igual (TestInjectCopilotAgentsAreSharedByVSCodeAndCopilotCLI).
	agent, known := map[string]model.AgentID{
		"antigravity": model.AgentAntigravity,
		"claude":      model.AgentClaudeCode,
		"codex":       model.AgentCodex,
		"copilot":     model.AgentVSCodeCopilot,
		"cursor":      model.AgentCursor,
		"gemini":      model.AgentGeminiCLI,
		"opencode":    model.AgentOpenCode,
		"qwen":        model.AgentQwenCode,
	}[family]
	if !known {
		t.Fatalf("asset family %q in %q has no mapped runtime identity", family, path)
	}
	return agent
}

func markdownFrontmatter(t *testing.T, path string) string {
	t.Helper()
	parts := strings.SplitN(assets.MustRead(path), "---", 3)
	if len(parts) != 3 {
		t.Fatalf("%s missing frontmatter", path)
	}
	return parts[1]
}

func assertTextContainsClauses(t *testing.T, label, content string, clauses []string) {
	t.Helper()
	for _, clause := range clauses {
		if !strings.Contains(content, clause) {
			t.Errorf("%s missing required clause %q", label, clause)
		}
	}
}

func readGentleOrchestratorPrompt(t *testing.T, settingsPath string) string {
	t.Helper()
	payload, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatal(err)
	}
	agentsMap := root["agent"].(map[string]any)
	orchestrator, ok := agentsMap["axiom-orchestrator"].(map[string]any)
	if !ok {
		orchestrator, ok = agentsMap["gentle-orchestrator"].(map[string]any)
	}
	if !ok {
		t.Fatal("missing orchestrator in agents map")
	}
	return orchestrator["prompt"].(string)
}
