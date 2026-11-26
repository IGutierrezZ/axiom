package sdd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestOptionalResearchMigratesInstalledOpenCodePrompt(t *testing.T) {
	const oldGate = "### Research and Pre-Proposal Gate (MANDATORY) — Offer `sdd-research` immediately after `sdd-explore`; selection makes completion mandatory."
	const userText = "Keep my proposal question round notes and custom release instructions."
	for _, mode := range []model.SDDModeID{model.SDDModeSingle, model.SDDModeMulti} {
		for _, variant := range []string{"inline", "managed", "optional-managed"} {
			name := string(mode) + "/" + variant
			t.Run(name, func(t *testing.T) {
				home := t.TempDir()
				mockNoPackageManager(t)
				settings := filepath.Join(home, ".config", "opencode", "opencode.json")
				gate := oldGate
				if variant == "optional-managed" {
					gate = strings.Split(researchLifecycleContract(), "#### Research-specific gatekeeper precedence")[0]
				}
				if variant != "inline" {
					gate = "<!-- gentle-ai:sdd-research-lifecycle -->\n" + gate + "\n<!-- /gentle-ai:sdd-research-lifecycle -->"
				}
				seed := map[string]any{"agent": map[string]any{"gentle-orchestrator": map[string]any{"mode": "primary", "prompt": "# Custom prompt\n" + userText + "\n" + strings.Replace(renderSDDOrchestratorAsset(model.AgentOpenCode), researchLifecycleContract(), gate, 1)}}}
				data, err := json.Marshal(seed)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(settings, data, 0o644); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					if _, err := Inject(home, opencodeAdapter(), mode, InjectOptions{PreserveOpenCodeOrchestratorPrompt: true}); err != nil {
						t.Fatal(err)
					}
					prompt := readGentleOrchestratorPrompt(t, settings)
					if strings.Contains(prompt, "Research and Pre-Proposal Gate (MANDATORY)") {
						t.Fatal("migration kept the retired research admission block")
					}
					if strings.Count(prompt, userText) != 1 {
						t.Fatal("migration changed unrelated user text")
					}
					if strings.Count(prompt, "### Optional Research and Product Discovery") != 1 {
						t.Fatal("migration duplicated managed research instructions")
					}
				}
			})
		}
	}
}
