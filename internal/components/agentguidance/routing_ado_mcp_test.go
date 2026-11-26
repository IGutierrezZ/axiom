package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestODDRoutingAzureDevOpsMCPReadsOnly pins the ODD side of mcp-ado (C-07-g): the work-item
// section of every runtime lets the agent read through the `azure-devops` MCP server and forbids
// writing to Azure DevOps at a lifecycle event.
func TestODDRoutingAzureDevOpsMCPReadsOnly(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()
			section := workItemRoutingSection(t, string(agent.ID))
			for _, want := range []string{
				"the `azure-devops` MCP server (signed in with Entra) lets you read",
				"Never use it, or any route other than the binary, to write to Azure DevOps at a lifecycle event",
				"Never ask for, write or relay a token for it.",
			} {
				if !strings.Contains(section, want) {
					t.Errorf("%s ODD work-item section lacks %q", agent.ID, want)
				}
			}
		})
	}
}
