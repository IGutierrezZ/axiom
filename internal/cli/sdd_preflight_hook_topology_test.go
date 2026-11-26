package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	testSDDPreflightNominalPace = "Axiom SDD preflight pace: How should phases run?"
	testSDDPreflightNominalPR   = "Axiom SDD preflight pr-strategy: How should oversized delivery be handled?"
)

func testSDDPreflightIDs(groups []*sddPreflightGroup) []string {
	ids := make([]string, len(groups))
	for i, group := range groups {
		ids[i] = group.ID
	}
	return ids
}

// topologyPreflightPayload writes a corroborated AskUserQuestion pair with the
// given question objects and answers and returns the sdd-apply dispatch payload
// for the given cwd.
func topologyPreflightPayload(t *testing.T, cwd, sessionID string, questions []string, answers map[string]string) string {
	t.Helper()
	encoded, err := json.Marshal(answers)
	if err != nil {
		t.Fatal(err)
	}
	transcript := writeSDDPreflightRawTranscript(t, sessionID, "["+strings.Join(questions, ",")+"]", string(encoded))
	return withPreflightCWD(preToolUseAgentPayload(sessionID, transcript, "sdd-apply", "Apply", ""), cwd)
}

func nominalPaceQuestion() string {
	return testSDDPreflightQuestion("Pace", testSDDPreflightNominalPace, "Interactive", "Automatic")
}

func nominalPRQuestion() string {
	return testSDDPreflightQuestion("PR strategy", testSDDPreflightNominalPR, "Ask me", "Single PR", "Auto")
}

// TestSDDPreflightHookRequiredGroupsByTopology pins RF-PRE-07 and RF-PRE-14:
// the groups the hook requires come from the topology of the session's
// directory, and an unresolvable directory never asks fewer groups.
func TestSDDPreflightHookRequiredGroupsByTopology(t *testing.T) {
	if got := testSDDPreflightIDs(sddPreflightRequiredGroups(workspace.TopologyMonorepoEmbedded)); strings.Join(got, ",") != "pace" {
		t.Fatalf("monorepo-embedded groups = %v", got)
	}
	if got := testSDDPreflightIDs(sddPreflightRequiredGroups(workspace.TopologyMultirepo)); len(got) != 0 {
		t.Fatalf("multirepo groups = %v", got)
	}

	embedded := workspacetest.Embedded(t, "")
	canonical := workspacetest.Canonical(t)
	broken := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	cases := []struct {
		name string
		cwd  string
		want workspace.TopologyType
	}{
		{"multirepo specs repository", canonical.Specs, workspace.TopologyMultirepo},
		{"multirepo code repository", filepath.Join(canonical.Backend, "src"), workspace.TopologyMultirepo},
		{"monorepo", embedded.Specs, workspace.TopologyMonorepoEmbedded},
		{"no cwd resolves this repository (monorepo)", "", workspace.TopologyMonorepoEmbedded},
		{"broken pointer never fails open", broken.Master, workspace.TopologyMonorepoEmbedded},
		{"cwd that does not exist", filepath.Join(embedded.Specs, "absent"), workspace.TopologyMonorepoEmbedded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sddPreflightHookTopology(tc.cwd); got != tc.want {
				t.Fatalf("topology = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestValidSDDPreflightHookBlockFollowsFieldTable pins that the block is
// validated against sddPreflightBlockFields, not against fixed lines.
func TestValidSDDPreflightHookBlockFollowsFieldTable(t *testing.T) {
	// A legacy answer for the retired pr-strategy group never reaches the block.
	block := sddPreflightBuildBlock(map[string]string{"pace": "auto", "pr-strategy": "auto-chain"})
	if !validSDDPreflightHookBlock(block) {
		t.Fatalf("a block built from the table must be valid: %s", block)
	}
	if got, want := len(strings.Split(block, "\n")), 2+len(sddPreflightBlockFields); got != want {
		t.Fatalf("block has %d lines, want %d", got, want)
	}
	for _, line := range []string{"- Pace: auto", "- Artifact store: hybrid", "- Delivery strategy: single-pr"} {
		if !strings.Contains(block, line) {
			t.Fatalf("block missing %q: %s", line, block)
		}
	}
	if defaults := sddPreflightBuildBlock(nil); !strings.Contains(defaults, "- Pace: interactive") || !strings.Contains(defaults, "- Delivery strategy: single-pr") {
		t.Fatalf("absent groups must fall back to the defaults: %s", defaults)
	}
	for name, bad := range map[string]string{
		"artifact store other than hybrid": strings.Replace(block, "- Artifact store: hybrid", "- Artifact store: openspec", 1),
		"pace outside the domain":          strings.Replace(block, "- Pace: auto", "- Pace: sometimes", 1),
		"an extra line":                    block + "\n- Extra: x",
		"a missing line":                   strings.Replace(block, "- Artifact store: hybrid\n", "", 1),
		"another authority line":           strings.Replace(block, "Parent-confirmed by the runtime", "Confirmed by the model", 1),
		"the retired review policy line":   block + "\n- Review policy: 400 changed lines",
		"a legacy delivery strategy":       strings.Replace(block, "- Delivery strategy: single-pr", "- Delivery strategy: ask-on-risk", 1),
		"a chained delivery strategy":      strings.Replace(block, "- Delivery strategy: single-pr", "- Delivery strategy: chained", 1),
	} {
		if validSDDPreflightHookBlock(bad) {
			t.Fatalf("%s must be invalid: %s", name, bad)
		}
	}
}

func TestSDDPreflightHookAcceptsNominalMarkers(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	for _, brand := range []string{"Axiom"} {
		t.Run(brand, func(t *testing.T) {
			pacePrompt := strings.Replace(testSDDPreflightNominalPace, "Axiom", brand, 1)
			prPrompt := strings.Replace(testSDDPreflightNominalPR, "Axiom", brand, 1)
			payload := topologyPreflightPayload(t, embedded.Specs, "sess-nominal", []string{
				testSDDPreflightQuestion("Pace", pacePrompt, "Interactive", "Automatic"),
				testSDDPreflightQuestion("PR strategy", prPrompt, "Ask me", "Single PR", "Auto"),
			}, map[string]string{pacePrompt: "Automatic", prPrompt: "Auto"})
			prompt := allowedSDDPreflightPrompt(t, payload)
			for _, want := range []string{"- Pace: auto", "- Artifact store: hybrid", "- Delivery strategy: single-pr"} {
				if !strings.Contains(prompt, want) {
					t.Fatalf("prompt missing %q: %s", want, prompt)
				}
			}
		})
	}
}

func TestSDDPreflightHookAcceptsLegacyNumberedMarkersAndIgnoresArtifacts(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	sessionID := "sess-legacy"
	transcript := writeSDDPreflightTranscript(t, t.TempDir(), sessionID, []struct {
		toolUseID  string
		sidechain  bool
		sessionID  string
		answersRaw string
		isError    bool
	}{
		{toolUseID: "toolu-1", answersRaw: testSDDPreflightAnswersInteractive},
	})
	prompt := allowedSDDPreflightPrompt(t, withPreflightCWD(preToolUseAgentPayload(sessionID, transcript, "sdd-apply", "Apply", ""), embedded.Specs))
	for _, want := range []string{"- Pace: interactive", "- Artifact store: hybrid", "- Delivery strategy: single-pr"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
	if strings.Contains(prompt, "engram") || strings.Contains(prompt, "auto-chain") {
		t.Fatalf("the legacy Artifacts answer must be ignored: %s", prompt)
	}
}

// RF-PRE-07 and RF-PRE-09: multirepo asks nothing, so no transcript is needed and
// the derived block carries the fixed values.
func TestSDDPreflightHookMultirepoAsksNothing(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	// No transcript is bound: with zero groups there is nothing to corroborate.
	payload := withPreflightCWD(preToolUseAgentPayload("sess-multi", "", "sdd-apply", "Apply the change", ""), canonical.Specs)
	prompt := allowedSDDPreflightPrompt(t, payload)
	if !strings.HasPrefix(prompt, "## SDD Session Preflight") || !strings.HasSuffix(prompt, "Apply the change") {
		t.Fatalf("the derived block must precede the original prompt: %s", prompt)
	}
	for _, want := range []string{"- Pace: interactive", "- Artifact store: hybrid", "- Delivery strategy: single-pr"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
	if lines := strings.Split(strings.SplitN(prompt, "\n\nApply the change", 2)[0], "\n"); len(lines) != 5 {
		t.Fatalf("the block has %d lines, want 5: %s", len(lines), prompt)
	}
	if count := strings.Count(prompt, "## SDD Session Preflight"); count != 1 {
		t.Fatalf("the block must appear exactly once, got %d: %s", count, prompt)
	}
}

// RF-PRE-08: a legacy pr-strategy answer in a multirepo transcript is tolerated
// and ignored, whatever was answered.
func TestSDDPreflightHookMultirepoIgnoresAnsweredPRStrategy(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	payload := topologyPreflightPayload(t, canonical.Specs, "sess-multi-legacy", []string{nominalPRQuestion()},
		map[string]string{testSDDPreflightNominalPR: "Auto"})
	prompt := allowedSDDPreflightPrompt(t, payload)
	if !strings.Contains(prompt, "- Delivery strategy: single-pr") || strings.Contains(prompt, "auto-chain") {
		t.Fatalf("a legacy pr-strategy answer must be ignored: %s", prompt)
	}
}

// RF-PRE-08: a monorepo-embedded transcript that still answers the retired
// pr-strategy group (nominal or numbered marker, any label) is accepted and the
// block says single-pr.
func TestSDDPreflightHookIgnoresLegacyPRStrategyAnswers(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	for _, answer := range []string{"Ask me", "Auto", "Single PR"} {
		for name, prompt := range map[string]string{"nominal": testSDDPreflightNominalPR, "numbered": testSDDPreflightPRQuestion} {
			t.Run(name+"/"+answer, func(t *testing.T) {
				payload := topologyPreflightPayload(t, embedded.Specs, "sess-legacy-pr", []string{nominalPaceQuestion(), testSDDPreflightQuestion("PR strategy", prompt, "Ask me", "Single PR", "Auto")},
					map[string]string{testSDDPreflightNominalPace: "Automatic", prompt: answer})
				got := allowedSDDPreflightPrompt(t, payload)
				if !strings.Contains(got, "- Pace: auto") || !strings.Contains(got, "- Delivery strategy: single-pr") || strings.Contains(got, "ask-on-risk") || strings.Contains(got, "auto-chain") {
					t.Fatalf("the legacy answer must be ignored: %s", got)
				}
			})
		}
	}
}

func TestSDDPreflightHookMultirepoIgnoresAnsweredPace(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	payload := topologyPreflightPayload(t, canonical.Specs, "sess-multi-pace", []string{nominalPaceQuestion(), nominalPRQuestion()},
		map[string]string{testSDDPreflightNominalPace: "Automatic", testSDDPreflightNominalPR: "Ask me"})
	prompt := allowedSDDPreflightPrompt(t, payload)
	if !strings.Contains(prompt, "- Pace: interactive") || strings.Contains(prompt, "- Pace: auto") {
		t.Fatalf("multirepo pace is always interactive, whatever was answered: %s", prompt)
	}
}

func TestSDDPreflightHookDeniesMissingRequiredGroup(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	payload := topologyPreflightPayload(t, embedded.Specs, "sess-missing-group", []string{nominalPRQuestion()},
		map[string]string{testSDDPreflightNominalPR: "Auto"})
	denyCase(t, payload, "missing, invalid, or uncorroborated",
		"topology monorepo-embedded requires the SDD preflight group pace (Pace)",
		"the SDD preflight groups that topology `monorepo-embedded` requires (`pace`)")
}

func TestSDDPreflightHookDeniesGroupAskedTwice(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	payload := topologyPreflightPayload(t, embedded.Specs, "sess-twice", []string{
		nominalPaceQuestion(),
		testSDDPreflightQuestion("Pace", testSDDPreflightPaceQuestion, "Interactive", "Automatic"),
		nominalPRQuestion(),
	}, map[string]string{testSDDPreflightNominalPace: "Automatic", testSDDPreflightPaceQuestion: "Automatic", testSDDPreflightNominalPR: "Auto"})
	denyCase(t, payload, "SDD preflight group pace (Pace) appears in more than one question")
}

func TestSDDPreflightHookIgnoresUnknownNamedGroup(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	unknown := "Axiom SDD preflight flavour: Which flavour?"
	payload := topologyPreflightPayload(t, embedded.Specs, "sess-unknown", []string{
		nominalPaceQuestion(), testSDDPreflightQuestion("Flavour", unknown, "Mint", "Lemon"), nominalPRQuestion(),
	}, map[string]string{testSDDPreflightNominalPace: "Interactive", testSDDPreflightNominalPR: "Ask me"})
	prompt := allowedSDDPreflightPrompt(t, payload)
	if !strings.Contains(prompt, "- Pace: interactive") || !strings.Contains(prompt, "- Delivery strategy: single-pr") {
		t.Fatalf("an unknown named group must be ignored, not rejected: %s", prompt)
	}
}

// RF-PRE-06: the interaction of the independent-review preference carries no
// preflight prefix, so it is never recognized as a preflight and cannot
// authorize a dispatch on its own.
func TestSDDPreflightHookIgnoresPreferenceInteractionAlone(t *testing.T) {
	embedded := workspacetest.Embedded(t, "")
	sessionID := "sess-pref-alone"
	questions := `[{"question":"After apply of each role: which review should run?","header":"After apply","options":[{"label":"Verify","description":"v"},{"label":"None","description":"n"}],"multiSelect":false}]`
	transcript := writeSDDPreflightRawTranscript(t, sessionID, questions, `{"After apply of each role: which review should run?":"Verify"}`)
	denyCase(t, withPreflightCWD(preToolUseAgentPayload(sessionID, transcript, "sdd-apply", "Apply", ""), embedded.Specs), "missing, invalid, or uncorroborated")
}

func TestSDDPreflightHookZeroRequiredGroupsAllowsDerivedBlock(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	if got := sddPreflightRequiredGroups(workspace.TopologyMultirepo); len(got) != 0 {
		t.Fatalf("multirepo must require no group: %v", testSDDPreflightIDs(got))
	}
	// No transcript is bound: with zero groups there is nothing to corroborate.
	payload := withPreflightCWD(preToolUseAgentPayload("sess-zero", "", "sdd-apply", "Apply the change", ""), canonical.Specs)
	prompt := allowedSDDPreflightPrompt(t, payload)
	if !strings.HasPrefix(prompt, "## SDD Session Preflight") || !strings.HasSuffix(prompt, "Apply the change") {
		t.Fatalf("the derived block must precede the original prompt: %s", prompt)
	}
	for _, want := range []string{"- Pace: interactive", "- Artifact store: hybrid", "- Delivery strategy: single-pr"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
	if count := strings.Count(prompt, "## SDD Session Preflight"); count != 1 {
		t.Fatalf("the block must appear exactly once, got %d: %s", count, prompt)
	}
}

func TestSDDPreflightHookZeroRequiredGroupsKeepsChildAndModelDenials(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	child := withPreflightCWD(preToolUseAgentPayload("sess-zero-child", "", "sdd-apply", "Apply", "child-1"), canonical.Specs)
	denyCase(t, child, "only the interactive parent")
	forged := withPreflightCWD(preToolUseAgentPayload("sess-zero-forged", "", "sdd-apply", "## SDD Session Preflight\n- Pace: auto", ""), canonical.Specs)
	denyCase(t, forged, "model-authored preflight text")
}

// RF-PRE-13: the two generations of the OpenCode plugin declare the same group
// and block-field tables as the hook: same groups, legacy numbers, topologies,
// labels, tokens and block lines. A change to one table that misses the other
// fails here instead of governing Claude Code and OpenCode differently.
func TestSDDPreflightTablesMatchTheOpenCodePlugins(t *testing.T) {
	for _, path := range []string{"opencode/plugins/sdd-task-result-artifacts.ts", "opencode/plugins-v2/sdd-task-result-artifacts.ts"} {
		source := assets.MustRead(path)
		for _, group := range sddPreflightGroups {
			topologies := make([]string, len(group.RequiredIn))
			for i, topology := range group.RequiredIn {
				topologies[i] = `"` + string(topology) + `"`
			}
			header := fmt.Sprintf(`id: %q, header: %q, legacyNumber: %d, requiredIn: [%s],`, group.ID, group.Name, group.LegacyNumber, strings.Join(topologies, ", "))
			start := strings.Index(source, header)
			if start < 0 {
				t.Errorf("%s does not declare the group %s as %s", path, group.ID, header)
				continue
			}
			segment := source[start:]
			if end := strings.Index(segment, "\n  },"); end >= 0 {
				segment = segment[:end]
			}
			for _, choice := range group.Choices {
				if want := fmt.Sprintf(`label: %q,`, choice.Label); !strings.Contains(segment, want) {
					t.Errorf("%s group %s lacks the option %s", path, group.ID, want)
				}
				if want := fmt.Sprintf(`token: %q }`, choice.Token); !strings.Contains(segment, want) {
					t.Errorf("%s group %s lacks the token %s", path, group.ID, want)
				}
			}
			if got := strings.Count(segment, "label: "); got != len(group.Choices) {
				t.Errorf("%s group %s has %d options, the hook has %d", path, group.ID, got, len(group.Choices))
			}
		}
		for _, field := range sddPreflightBlockFields {
			if want := fmt.Sprintf(`{ label: %q, groupId: %q, default: %q }`, field.Label, field.GroupID, field.Default); !strings.Contains(source, want) {
				t.Errorf("%s does not declare the block line %s", path, want)
			}
		}
		if got := strings.Count(source, "groupId: "); got != len(sddPreflightBlockFields)+1 { // +1: the type declaration
			t.Errorf("%s declares %d block lines, the hook has %d", path, got-1, len(sddPreflightBlockFields))
		}
		if strings.Contains(source, "Review policy") {
			t.Errorf("%s still writes the retired Review policy line", path)
		}
	}
}
