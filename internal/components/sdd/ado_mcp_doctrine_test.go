package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestAzureDevOpsMCPRuleParity pins the agent/binary boundary of mcp-ado (C-07-g): every runtime
// receives the same paragraph, which lets the agent read Azure DevOps through the MCP server and
// keeps every lifecycle write in the binary.
func TestAzureDevOpsMCPRuleParity(t *testing.T) {
	t.Parallel()

	const start = "**Azure DevOps through MCP.**"
	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()
			prompt := composeOrchestratorPrompt(agent.ID)
			from := strings.Index(prompt, start)
			if from < 0 {
				t.Fatalf("%s orchestrator has no %s paragraph", agent.ID, start)
			}
			rule := prompt[from:]
			if end := strings.Index(rule, "\n\n"); end >= 0 {
				rule = rule[:end]
			}
			for _, want := range []string{
				"`azure-devops` plugin is active", "`azure-devops`", "Entra",
				"only to read", "possible duplicates",
				"Never use it, or any route other than the binary, to write to Azure DevOps at a lifecycle event",
				"`axiom change start` and the tracker verbs above", "never blocks the flow",
				"Never ask for, write or relay a token",
			} {
				if !strings.Contains(rule, want) {
					t.Errorf("%s Azure DevOps MCP rule missing %q", agent.ID, want)
				}
			}
		})
	}
}
