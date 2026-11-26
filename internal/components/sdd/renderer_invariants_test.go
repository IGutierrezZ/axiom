package sdd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

type orchestratorContractSection struct {
	name   string
	marker string
	// A sentinel holding "|" accepts any of its alternatives. The fork renamed
	// the binary and flattened the verb, so a shipped invocation can read
	// either `axiom sdd attempt acquire` or `axiom sdd-attempt acquire`;
	// absorbing upstream must not reopen this contract. No sentinel carries a
	// literal "|", so the separator is unambiguous.
	sentinels []string
}

func sectionBodyHasSentinel(sectionBody, sentinel string) bool {
	for _, alternative := range strings.Split(sentinel, "|") {
		if strings.Contains(sectionBody, alternative) {
			return true
		}
	}
	return false
}

// Solo se fijan los encabezados que el renderizador usa como anclas y su orden,
// más el identificador del contrato de consentimiento que parsea el código. La
// redacción de cada sección es libre.
var currentOpenCodeOrchestratorSections = []orchestratorContractSection{
	{name: "blocking prompts", marker: "### Lossless Blocking Prompts (MANDATORY)"},
	{
		name:      "edit-authority consent relay",
		marker:    "#### SDD Edit-Authority Consent Relay (MANDATORY)",
		sentinels: []string{"axiom.sdd-integration.consent/v1"},
	},
	{name: "dispatcher guard", marker: "### Native SDD Dispatcher Guard"},
	{name: "session preflight", marker: "### SDD Session Preflight (HARD GATE)"},
	{name: "lifecycle routing", marker: "### SDD Entry Routing (MANDATORY)"},
	{name: "initialization gate", marker: "### SDD Init Guard (MANDATORY)"},
	{name: "automatic gatekeeper", marker: "### Automatic Mode Gatekeeper (MANDATORY)"},
	{name: "delivery strategy", marker: "### Delivery Strategy"},
	{name: "skill resolution", marker: "### Sub-Agent Launch Pattern"},
	{name: "skill resolution feedback", marker: "### Skill Resolution Feedback"},
	{name: "sub-agent context protocol", marker: "### Sub-Agent Context Protocol"},
}

func assertCurrentOpenCodeOrchestratorContract(t *testing.T, label string, content string, agent model.AgentID, profileName string) {
	t.Helper()

	if strings.Contains(content, runtimeAgentIDPlaceholder) {
		t.Fatalf("%s retains runtime identity placeholder", label)
	}
	lastMarker := -1
	for _, section := range currentOpenCodeOrchestratorSections {
		markerIndex := strings.Index(content, section.marker)
		if markerIndex < 0 {
			t.Errorf("%s missing %s section marker %q", label, section.name, section.marker)
			continue
		}
		if markerIndex <= lastMarker {
			t.Errorf("%s moved %s before a preceding contract section", label, section.name)
		}
		lastMarker = markerIndex
		sectionBody := content[markerIndex+len(section.marker):]
		nearestFollowing := len(sectionBody)
		for _, following := range currentOpenCodeOrchestratorSections {
			if following.marker == section.marker {
				continue
			}
			if followingIndex := strings.Index(sectionBody, following.marker); followingIndex >= 0 && followingIndex < nearestFollowing {
				nearestFollowing = followingIndex
			}
		}
		sectionBody = sectionBody[:nearestFollowing]
		for _, sentinel := range section.sentinels {
			if profileName != "" {
				for _, phase := range ProfilePhaseOrder() {
					sentinel = strings.ReplaceAll(sentinel, phase, phase+"-"+profileName)
				}
			}
			if !sectionBodyHasSentinel(sectionBody, sentinel) {
				t.Errorf("%s %s section lost sentinel %q", label, section.name, sentinel)
			}
		}
	}

	if profileName == "" {
		assertTextContainsClauses(t, label+" model assignment contract", content, []string{
			"<!-- axiom:sdd-model-assignments -->",
		})
	}
}

func TestOpenCodeBaseOrchestratorPreservesCurrentContract(t *testing.T) {
	content, err := composeOpenCodeOrchestratorPrompt(model.AgentOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrentOpenCodeOrchestratorContract(t, "OpenCode base orchestrator", content, model.AgentOpenCode, "")

	visibilityMarker := "<!-- axiom:" + openCodeDelegationVisibilitySectionID + " -->"
	for _, required := range []string{
		visibilityMarker,
	} {
		if !strings.Contains(content, required) {
			t.Errorf("OpenCode base orchestrator lost current Desktop delegation section sentinel %q", required)
		}
	}
}

func TestOpenCodeBaseInjectionBindsAssignmentsAndPreservesContract(t *testing.T) {
	mockNoPackageManager(t)
	home := t.TempDir()
	assignments := map[string]model.ModelAssignment{
		"gentle-orchestrator": {ProviderID: "openai", ModelID: "gpt-5.1", Effort: "high"},
		"sdd-apply":           {ProviderID: "anthropic", ModelID: "claude-sonnet-4-5"},
	}
	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{OpenCodeModelAssignments: assignments}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	agentsMap := readOpenCodeAgents(t, filepath.Join(home, ".config", "opencode", "opencode.json"))
	prompt := agentPrompt(t, agentsMap, "gentle-orchestrator")
	assertCurrentOpenCodeOrchestratorContract(t, "OpenCode base injection", prompt, model.AgentOpenCode, "")

	for agentName, wantModel := range map[string]string{
		"axiom-orchestrator": "openai/gpt-5.1",
		"sdd-apply":          "anthropic/claude-sonnet-4-5",
	} {
		entry, ok := agentsMap[agentName].(map[string]any)
		if !ok && agentName == "axiom-orchestrator" {
			entry, ok = agentsMap["gentle-orchestrator"].(map[string]any)
		}
		if !ok || entry["model"] != wantModel {
			t.Fatalf("%s model = %#v, want %q", agentName, entry["model"], wantModel)
		}
	}
	orchEntry, _ := agentsMap["axiom-orchestrator"].(map[string]any)
	if orchEntry == nil {
		orchEntry = agentsMap["gentle-orchestrator"].(map[string]any)
	}
	if variant := orchEntry["variant"]; variant != "high" {
		t.Fatalf("orchestrator variant = %#v, want high", variant)
	}
}

func TestOpenCodeNamedProfileOrchestratorPreservesCurrentContract(t *testing.T) {
	const profileName = "rapid"
	profile := model.Profile{
		Name:              profileName,
		OrchestratorModel: model.ModelAssignment{ProviderID: "openai", ModelID: "gpt-5.1"},
		PhaseAssignments: map[string]model.ModelAssignment{
			"sdd-apply": {ProviderID: "anthropic", ModelID: "claude-sonnet-4-5"},
		},
	}
	home := t.TempDir()
	overlay, err := GenerateProfileOverlay(profile, home, filepath.Join(home, "opencode.json"), nil, "")
	if err != nil {
		t.Fatalf("GenerateProfileOverlay() error = %v", err)
	}

	var root struct {
		Agent map[string]any `json:"agent"`
	}
	if err := json.Unmarshal(overlay, &root); err != nil {
		t.Fatalf("unmarshal profile overlay: %v", err)
	}
	orchestratorName := "sdd-orchestrator-" + profileName
	content := agentPrompt(t, root.Agent, orchestratorName)
	assertCurrentOpenCodeOrchestratorContract(t, "OpenCode named profile", content, model.AgentOpenCode, profileName)

	assertTextContainsClauses(t, "OpenCode named profile model assignments", content, []string{
		"<!-- axiom:sdd-model-assignments -->",
		"<!-- /axiom:sdd-model-assignments -->",
		"| orchestrator | openai/gpt-5.1 |",
		"| sdd-apply-" + profileName + " | anthropic/claude-sonnet-4-5 |",
	})
	for _, phase := range ProfilePhaseOrder() {
		suffixed := phase + "-" + profileName
		if !strings.Contains(content, suffixed) {
			t.Errorf("OpenCode named profile missing suffixed reference %q", suffixed)
		}
		if strings.Contains(content, suffixed+"-"+profileName) {
			t.Errorf("OpenCode named profile double-suffixed reference %q", suffixed+"-"+profileName)
		}
		for offset := 0; ; {
			relative := strings.Index(content[offset:], phase)
			if relative < 0 {
				break
			}
			start := offset + relative
			end := start + len(phase)
			if !strings.HasPrefix(content[end:], "-"+profileName) {
				t.Errorf("OpenCode named profile retained unsuffixed reference %q", content[start:end])
			}
			offset = end
		}
	}

	profileAgent, ok := root.Agent["sdd-apply-"+profileName].(map[string]any)
	if !ok || profileAgent["model"] != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("named sdd-apply agent model = %#v, want anthropic/claude-sonnet-4-5", profileAgent["model"])
	}
	if orchestrator, ok := root.Agent[orchestratorName].(map[string]any); !ok || orchestrator["model"] != "openai/gpt-5.1" {
		t.Fatalf("named orchestrator model = %#v, want openai/gpt-5.1", orchestrator["model"])
	}

	// This is an existing named-profile exclusion, not a future inheritance rule:
	// profile rendering deliberately removes the default Desktop narration block.
	visibilityMarker := "<!-- axiom:" + openCodeDelegationVisibilitySectionID + " -->"
	if strings.Contains(content, visibilityMarker) {
		t.Fatalf("named profile unexpectedly inherited the default Desktop delegation section")
	}
}
