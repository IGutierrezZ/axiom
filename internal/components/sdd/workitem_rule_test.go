package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestOrchestratorWorkItemRuleParity pins prov:74:24 and prov:74:26: every runtime
// receives the same work-item rule of the SDD kickoff, with the eleven points of
// design D-21, and the stop of `tasks` presents `pruebas.md` informatively
// (RF-GAT-32).
func TestOrchestratorWorkItemRuleParity(t *testing.T) {
	t.Parallel()

	const start = "**Work item of origin.**"
	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			prompt := composeOrchestratorPrompt(agent.ID)
			from := strings.Index(prompt, start)
			if from < 0 {
				t.Fatalf("%s orchestrator has no work-item rule", agent.ID)
			}
			rest := prompt[from:]
			rule := rest[:strings.Index(rest, "\n\n")]

			for _, want := range []string{
				// 1 query, not_applicable leaves the kickoff intact.
				"axiom change start --cwd <repo> --json", "`not_applicable`", "with no other call",
				// 2 ask for the WI, creating it is not an option.
				"ask for it and do not advance", "creating the work item is not an option",
				// 3 phase 1 before ensure and seal, derived name.
				"axiom change start --from <id|url>", "before `axiom branch ensure` and before `axiom sdd kickoff seal`",
				"`ensure --change`, `seal --change`",
				// 4 phase 1 refusal.
				"no branch, worktree, change folder or `kickoff.yaml`",
				// 5 failed ensure/seal keeps the staging, no phase 2, retry or discard.
				"keeps the staging and never runs phase 2", "decision_required", "without repeating phase 1",
				"axiom change start --discard --change <change>",
				// 6 phase 2 after seal and before proposal.md and explore.
				"After `seal` and before `proposal.md` and before `sdd-explore`", "axiom change start --adopt --change <change>",
				// 7 phase 2 refusal stops and no --discard on its own.
				"`change_not_prepared`", "`staging_not_found`", "`staging_conflict`", "`change_start_busy`", "`write_failed`",
				"stops before `proposal.md` and before `sdd-explore`", "in particular `--discard`",
				// 8 warning does not stop.
				"warning with exit code 0", "`staging_not_retired`", "does not stop the flow",
				// 9 abandon offers --discard only when chosen.
				"only runs when the person chooses it",
				// 10 ISO and bug profile.
				"ISO project and with the `bug` profile",
				// 11 explore gets untrusted data.
				"untrusted third-party data", "section H",
				// RF-RUT-18 spirit: no stop added.
				"adds no stop",
			} {
				if !strings.Contains(rule, want) {
					t.Errorf("%s work-item rule missing %q", agent.ID, want)
				}
			}
		})
	}
}

// TestOrchestratorTasksStopPresentsAcceptanceTests pins RF-GAT-32 for every runtime.
func TestOrchestratorTasksStopPresentsAcceptanceTests(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		prompt := composeOrchestratorPrompt(agent.ID)
		for _, want := range []string{
			"`acceptanceTests.active: true`",
			"`pruebas.md` exists",
			"axiom sdd tests check --cwd <repo> --change <change> --json",
			"never as clean",
			"declares its absence",
			"the stop does not mention it",
			"never conditions an approval, is never recorded in `gates.yaml` and does not change the three exits",
		} {
			if !strings.Contains(prompt, want) {
				t.Errorf("%s orchestrator tasks stop missing %q", agent.ID, want)
			}
		}
	}
}
