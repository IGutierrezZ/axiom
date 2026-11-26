package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

const trackerWritesStart = "**Tracker writes.**"

// trackerRule returns the "Tracker writes" paragraph of the orchestrator rendered for one runtime.
func trackerRule(t *testing.T, prompt string) string {
	t.Helper()
	from := strings.Index(prompt, trackerWritesStart)
	if from < 0 {
		t.Fatalf("orchestrator has no %s paragraph", trackerWritesStart)
	}
	rest := prompt[from:]
	end := strings.Index(rest, "\n\n")
	if end < 0 {
		end = len(rest)
	}
	return rest[:end]
}

// TestTrackerWritesRuleParity pins prov:ORQ:1 to prov:ORQ:4: every runtime receives the same
// "Tracker writes" paragraph, with the moments by style, the failure policy and the bans.
func TestTrackerWritesRuleParity(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()
			rule := trackerRule(t, composeOrchestratorPrompt(agent.ID))
			for _, want := range []string{
				// Condition and bans.
				"Only on the SDD lane", "`not_applicable`", "Never call them on the ODD lane",
				// Moments by style.
				"In `checkpointed`", "`spec` approved, `axiom integrations summary push`",
				"`design` approved, `axiom integrations tasks create`", "`## Estimación por rol`",
				"axiom integrations time propose --role <role>", "axiom integrations time record --role <role> --hours <hours>",
				"In `continuous` there are no gates and no stops", "without asking",
				"axiom branch ensure --change <c> --role <role>", "`task_not_bound`", "also in `continuous`",
				// Zero hours and a failed proposal close without hours.
				"--role <role> --close-only", "0 hours", "anything other than `task_not_bound`", "«sin imputación»",
				// Failures.
				"never undoes an approval", "`.tracker-notes.yaml`", "`rev_conflict`", "`time_pending_ambiguous`", "`time_pending`",
				// Archive.
				"`axiom sdd publish --archive`", "archived folder",
			} {
				if !strings.Contains(rule, want) {
					t.Errorf("%s tracker rule missing %q", agent.ID, want)
				}
			}
		})
	}
}

// TestTrackerDoctrineSkillsNameTheCloseAndTheJournal pins the skill-side doctrine (12.3, 12.6).
func TestTrackerDoctrineSkillsNameTheCloseAndTheJournal(t *testing.T) {
	t.Parallel()

	design := assets.MustRead("skills/sdd-design/SKILL.md")
	for _, want := range []string{"## Estimación por rol", "- <role>: <hours> h", "outside the data lines"} {
		if !strings.Contains(design, want) {
			t.Errorf("sdd-design missing %q", want)
		}
	}

	apply := assets.MustRead("skills/sdd-apply/SKILL.md")
	for _, want := range []string{
		"axiom integrations time propose", "axiom integrations time record --hours <confirmed hours>",
		"axiom integrations time record --hours <proposal>", "`axiom integrations time record --close-only`",
		"proposal is 0 hours", "anything other than `task_not_bound`",
		"`rev_conflict`", "`time_pending_ambiguous`", "`time_pending`",
	} {
		if !strings.Contains(apply, want) {
			t.Errorf("sdd-apply missing %q", want)
		}
	}

	archive := assets.MustRead("skills/sdd-archive/SKILL.md")
	notes := strings.Index(archive, "`.tracker-notes.yaml`")
	move := strings.Index(archive, "### Step 3: Move to Archive")
	if notes < 0 || move < 0 || notes > move {
		t.Errorf("sdd-archive must name .tracker-notes.yaml before the step that moves the folder (notes=%d, move=%d)", notes, move)
	}
	for _, want := range []string{
		"## Seguimiento en el tracker", "«sin estimación»", "`item_id`", "BEFORE the move",
		"axiom integrations summary push --cwd <archive root>", "axiom sdd publish --archive",
	} {
		if !strings.Contains(archive, want) {
			t.Errorf("sdd-archive missing %q", want)
		}
	}
}

// trackerCanaryPAT is the recognisable fake PAT of the Azure DevOps canary (internal/cli).
const trackerCanaryPAT = "ado-canario-PAT-9f8e7d6c5b4a"

// TestTrackerDoctrineNamesNoSecretsOrTrackerRoutes pins hole 16 of the design (prov:ORQ:5): the
// projected orchestrator of every runtime and the three skills never name a tracker route, a
// credential shape, the canary value or a fixture identity.
func TestTrackerDoctrineNamesNoSecretsOrTrackerRoutes(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		"_apis/", "dev.azure.com", "workitems/", "Authorization", "Basic ", "pat=",
		trackerCanaryPAT, "ana.unique.fixture", "ana.gil.privado", "berto.privado",
	}
	check := func(name, text string) {
		for _, word := range forbidden {
			if strings.Contains(text, word) {
				t.Errorf("%s names %q", name, word)
			}
		}
	}
	for _, agent := range catalog.AllAgents() {
		check("orchestrator "+string(agent.ID), composeOrchestratorPrompt(agent.ID))
	}
	for _, skill := range []string{"sdd-design", "sdd-apply", "sdd-archive"} {
		check(skill, assets.MustRead("skills/"+skill+"/SKILL.md"))
	}
}
