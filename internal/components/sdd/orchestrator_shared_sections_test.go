package sdd

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// #3817: the SDD orchestrator contract is maintained as every hand-written
// near-duplicates. Measured across them, 19 of 21 shared subsections have
// drifted -- Delegation Rules alone has eleven variants across eleven runtimes
// -- so reconciling those is a decision per section, not a refactor.
//
// These five are the subsections that had NOT drifted semantically: three are
// byte-identical across every runtime that carries them, and the other two
// differ only cosmetically (a ```text fence, and "the user" for "user"). They
// move to one shared asset so the mechanism exists and so this set cannot drift
// again. Each runtime keeps its own heading line, which is why codex may hold a
// section at ## while the others hold it at ###.

// The fork renamed the binary and flattened the verb, so the shipped assets
// spell the status invocation `axiom sdd status`. Upstream still ships
// `axiom sdd-status`, and absorbing it must not reopen this contract:
// either spelling satisfies it.
func containsStatusContinuationInvocation(text string) bool {
	return strings.Contains(text, "axiom sdd status [change] --cwd <repo> --json --instructions") ||
		strings.Contains(text, "gentle-ai sdd-status [change] --cwd <repo> --json --instructions")
}

// Solo se fija la invocación nativa (comando y flags); la redacción es libre.
func assertStatusContinuationContract(t *testing.T, content string) {
	t.Helper()
	if !containsStatusContinuationInvocation(content) {
		t.Errorf("missing status/continuation contract invocation (neither `axiom sdd status` nor `axiom sdd-status` spelling)")
	}
}

func TestRegisteredAgentsRenderStatusContinuationContract(t *testing.T) {
	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ids := registry.SupportedAgents()
	if len(ids) != 11 {
		t.Fatalf("registered cohort changed: %v", ids)
	}
	templates := map[string]bool{}
	for _, agent := range ids {
		templates[sddOrchestratorAsset(agent)] = true
		t.Run(string(agent), func(t *testing.T) {
			content := renderSDDOrchestratorAsset(agent)
			if agent == model.AgentClaudeCode {
				if !strings.Contains(content, "~/.claude/skills/_shared/sdd-orchestrator-workflow.md") {
					t.Fatal("Claude bootstrap lost its lazy workflow reference")
				}
				lazy, err := renderClaudeSessionPreflight()
				if err != nil {
					t.Fatal(err)
				}
				content += lazy
			}
			assertStatusContinuationContract(t, content)
		})
	}
	if len(templates) != orchestratorTemplateCount {
		t.Fatalf("effective template coverage = %d, want %d", len(templates), orchestratorTemplateCount)
	}
	t.Run("claude-lazy-workflow", func(t *testing.T) {
		content, err := renderClaudeSessionPreflight()
		if err != nil {
			t.Fatal(err)
		}
		assertStatusContinuationContract(t, content)
	})
}

var sharedOrchestratorSectionNames = []string{
	"Native SDD Dispatcher Guard",
	"Language Domain Contract",
	"Dependency Graph",
	"Recovery Rule",
	// #4296: the risk-gated delegated-verification rule. Every runtime names a
	// concrete delegation mechanism in its own Delegation Rules body and carries
	// it under "Delegated Verification Gate (MANDATORY)".
	"Delegated Verification Gate (MANDATORY)",
	// INC-21 (design.md S4.7, H-2): the per-change kickoff questionnaire and
	// the four block-review gates, plus their normative boundary against
	// RDD (design.md S1.3). Registered here, in Phase 22, ahead of Phase
	// 23 inserting the {{AXIOM_SDD_SECTION:...}} marker into the
	// every per-agent orchestrator assets: TestEveryRuntimeRendersTheSharedSections'
	// own "continue" guard keeps this registration a no-op for every
	// runtime until that marker exists.
	"SDD Change Kickoff and Block Gates",
	// sdd-revision-independiente (RF-REV-01 a RF-REV-03, RF-GAT-16, RF-PRE-06): the optional
	// independent review (per-role verify, judgment and coverage review) and its preference.
	"SDD Independent Review",
	// single-pr (C-02-c): the one canonical Delivery Strategy body (delivery is
	// never asked: `single-pr | chained`, aliases read only, one non-blocking size
	// warning). Every runtime references it by placeholder, so no inline copy of
	// the retired four-way strategy can come back.
	"Delivery Strategy",
	// aviso-commit (C-02-d, RF-ORQ-21): the per-commit notice, the three review modes and
	// the fixup rules live once here; every runtime references them by placeholder.
	"Commit Review",
}

// TestSharedOrchestratorSectionsHaveOneSource pins that each shared section
// body lives in the shared asset and NOT in the per-runtime orchestrators.
//
// It opens the runtime assets. An earlier version only checked the shared
// asset against a substring extracted from that same shared asset, which is a
// tautology: it could not fail, while its name promised that no duplicated
// body survives in the eleven per-runtime files.
func TestSharedOrchestratorSectionsHaveOneSource(t *testing.T) {
	runtimeAssets := map[string]string{}
	err := fs.WalkDir(assets.FS, ".", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && path.Base(assetPath) == "sdd-orchestrator.md" {
			runtimeAssets[assetPath] = assets.MustRead(assetPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk orchestrator assets: %v", err)
	}
	if len(runtimeAssets) == 0 {
		t.Fatal("no runtime orchestrator assets were found")
	}

	for _, name := range sharedOrchestratorSectionNames {
		body := sharedOrchestratorSection(name)
		if strings.TrimSpace(body) == "" {
			t.Fatalf("shared asset carries no body for %q", name)
		}
		for assetPath, content := range runtimeAssets {
			if strings.Contains(content, body) {
				t.Errorf("%s still carries the %q body inline; it must carry the placeholder instead", assetPath, name)
			}
		}
	}
}

// TestEveryRuntimeRendersTheSharedSections pins that the substitution actually
// reaches the rendered prompt: deleting duplication must not delete content.
func TestEveryRuntimeRendersTheSharedSections(t *testing.T) {
	for _, agent := range []model.AgentID{
		model.AgentOpenCode, model.AgentCursor, model.AgentGeminiCLI, model.AgentQwenCode,
		model.AgentCodex,
		model.AgentClaudeCode, model.AgentAntigravity, model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI,
	} {
		rendered := renderSDDOrchestratorAsset(agent)
		if agent == model.AgentClaudeCode {
			// Claude keeps part of the shared sections in the lazy workflow, not in the bootstrap.
			lazy, err := renderClaudeSessionPreflight()
			if err != nil {
				t.Fatal(err)
			}
			rendered += lazy
		}
		for _, name := range sharedOrchestratorSectionNames {
			if !strings.Contains(rendered, name) {
				continue // a runtime that never carried this section keeps not carrying it
			}
			body := sharedOrchestratorSection(name)
			first := strings.SplitN(strings.TrimSpace(body), "\n", 2)[0]
			if !strings.Contains(rendered, first) {
				t.Errorf("%s rendered %q without its shared body", agent, name)
			}
		}
	}
}

// TestNoRawSharedSectionPlaceholderSurvivesRendering pins that no placeholder
// reaches a rendered prompt.
func TestNoRawSharedSectionPlaceholderSurvivesRendering(t *testing.T) {
	for _, agent := range []model.AgentID{
		model.AgentOpenCode, model.AgentCursor, model.AgentGeminiCLI, model.AgentQwenCode,
		model.AgentCodex,
		model.AgentAntigravity, model.AgentClaudeCode, model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI,
	} {
		if rendered := renderSDDOrchestratorAsset(agent); strings.Contains(rendered, "{{AXIOM_SDD_SECTION:") {
			t.Errorf("%s kept a raw shared-section placeholder", agent)
		}
	}
}
