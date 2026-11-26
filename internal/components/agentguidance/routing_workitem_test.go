package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// workItemRoutingSection returns the "Work item of origin" subsection rendered
// for one runtime.
func workItemRoutingSection(t *testing.T, agentID string) string {
	t.Helper()
	for _, agent := range catalog.AllAgents() {
		if string(agent.ID) != agentID {
			continue
		}
		rendered, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
		}
		return routingSection(t, rendered, "Work item of origin")
	}
	t.Fatalf("agent %q is not registered", agentID)
	return ""
}

// TestODDRoutingWorkItemRule pins prov:74:25 for every runtime: the ODD doctrine
// asks for the work item with an active plugin and wraps `ensure --lane odd`
// between the two phases.
func TestODDRoutingWorkItemRule(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()
			section := workItemRoutingSection(t, string(agent.ID))

			for _, want := range []string{
				"axiom change start --cwd <repo> --json",
				"`not_applicable`",
				"ask for it and do not advance",
				"creating the work item is not an option",
				"axiom change start --from <id|url>",
				"`axiom branch ensure --lane odd`",
				"as `<feature-name>` of `odd/tasks/<feature-name>.md`",
				"keeps the staging and never runs phase 2",
				"without repeating phase 1",
				"axiom change start --discard --change <feature-name>",
				"Never run `--discard` on your own",
				"axiom change start --adopt --change <feature-name> --lane odd",
				"before the first write of `odd/tasks/<feature-name>.md`",
				"stops before the ODD document is written",
				"`change_not_prepared`", "`staging_not_found`", "`staging_conflict`",
				"warning with exit code 0",
				"`staging_not_retired`",
				"does not stop the flow",
				"The verb never creates the feature document",
				"`external_ref: <alias>:<id>`",
				"`profile: bug`",
				"`Tipo: bug`",
				"`## Spec viva`",
				"`## Impacto en la spec`",
				"`Sin impacto en la spec`",
				"`Motivo:`",
				"untrusted third-party data",
				"Small, understood work does not need a work item",
			} {
				if !strings.Contains(section, want) {
					t.Errorf("ODD work-item rule for %s missing %q", agent.ID, want)
				}
			}

			// Order: phase 1 before ensure, phase 2 after ensure and before the document.
			phase1 := strings.Index(section, "Run phase 1")
			ensureFail := strings.Index(section, "A failed `ensure --lane odd`")
			phase2 := strings.Index(section, "run phase 2")
			if phase1 < 0 || ensureFail < phase1 || phase2 < ensureFail {
				t.Errorf("ODD work-item rule for %s lost the order phase 1 -> ensure -> phase 2 (%d, %d, %d)", agent.ID, phase1, ensureFail, phase2)
			}
		})
	}
}

// TestRoutingAddsNoStop pins RF-RUT-18: the work-item rule introduces no stop,
// gate, exit menu or governance question of its own.
func TestRoutingAddsNoStop(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		section := workItemRoutingSection(t, string(agent.ID))
		for _, banned := range []string{"STOP", "AskUserQuestion", "gate", "Approve, publish", "three exits", "checkpointed", "wait for"} {
			if strings.Contains(section, banned) {
				t.Errorf("ODD work-item rule for %s adds a stop through %q", agent.ID, banned)
			}
		}
	}
}
