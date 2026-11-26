package sdd

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

const (
	runtimeAgentIDPlaceholder    = "{{AXIOM_RUNTIME_AGENT_ID}}"
	researchLifecyclePlaceholder = "{{AXIOM_RESEARCH_LIFECYCLE}}"
)

// nativeReviewerResultSchema is the shape a Judgment Day judge returns.
const nativeReviewerResultSchema = `{"findings":[{"location":"path:line or path:start-end","severity":"CRITICAL","claim":"observable incorrect behavior","evidence_class":"deterministic","causal_disposition":"introduced","proof_refs":["concrete proof"]}],"evidence":["what was inspected"]}`

func renderSDDOrchestratorAsset(agent model.AgentID, options ...OrchestratorRenderOptions) string {
	return composeOrchestratorPrompt(agent, options...)
}

func researchLifecycleContract() string {
	return strings.TrimSpace(assets.MustRead("skills/_shared/research-lifecycle.md"))
}

// renderManagedAsset resolves one embedded asset into the exact bytes a single
// runtime installs. The agent is required, not optional: only the renderer
// knows which runtime is about to receive these bytes, and baking a constant
// into shared prose would hand every runtime the same runtime identity.
func renderManagedAsset(agent model.AgentID, path string) string {
	return bindRuntimeAgentIdentity(renderManagedAssetBody(path), agent)
}

// bindRuntimeAgentIdentity is the single substitution point every rendered
// asset passes through, so no branch added to renderManagedAssetBody can leak
// an unbound placeholder or an unspecialized identity.
func bindRuntimeAgentIdentity(content string, agent model.AgentID) string {
	return strings.ReplaceAll(content, runtimeAgentIDPlaceholder, string(agent))
}

func renderManagedAssetBody(path string) string {
	return renderManagedAssetBodyFromContent(path, assets.MustRead(path))
}

func renderManagedAssetBodyFromContent(path, content string) string {
	content = strings.ReplaceAll(content, researchLifecyclePlaceholder, researchLifecycleContract())
	if strings.Contains(path, "/agents/jd-judge-") {
		return replaceManagedSection(content, "## Review ledger contract", "", judgmentDayReviewerContract())
	}
	if strings.Contains(path, "/agents/jd-fix-agent.") {
		return content
	}
	if strings.Contains(content, "## Review ledger contract") {
		return replaceManagedSection(content, "## Review ledger contract", "", "")
	}
	return content
}

// replaceManagedSection replaces the section that starts at heading and ends
// before nextHeading (or at the end of content) with heading plus body. An
// empty body removes the section altogether, heading included.
func replaceManagedSection(content, heading, nextHeading, body string) string {
	start := strings.Index(content, heading)
	if start < 0 {
		return content
	}
	end := len(content)
	if nextHeading != "" {
		remainder := content[start+len(heading):]
		for _, candidate := range []string{"\n#### " + nextHeading, "\n### " + nextHeading, "\n## " + nextHeading} {
			if relative := strings.Index(remainder, candidate); relative >= 0 {
				end = start + len(heading) + relative + 1
				break
			}
		}
	}
	replacement := ""
	if body != "" {
		replacement = heading + "\n\n" + body + "\n\n"
	}
	return strings.TrimRight(content[:start], "\n") + "\n\n" + replacement + strings.TrimLeft(content[end:], "\n")
}

func judgmentDayReviewerContract() string {
	return fmt.Sprintf(`You are a read-only adversarial reviewer. Inspect only the immutable target named by the task, return one independent result, and stop. Do not edit, delegate, or inspect unrelated scope.

Report only real, user-impacting defects. Every severe finding must state whether the candidate introduced, behavior-activated, or worsened the behavior and cite changed-hunk, differential-test, candidate-created-path, or before/after proof. Mark unchanged defects pre-existing or base-only; use unknown when causality cannot be proved.

Use BLOCKER | CRITICAL | WARNING | SUGGESTION. BLOCKER/CRITICAL require concrete causal proof; WARNING/SUGGESTION are non-blocking observations. Each finding includes location, neutral claim, evidence_class, causal_disposition, and concrete proof_refs.

Return one JSON object and no prose. Use exactly this native result shape:

%s

This is a judgment-day judge result. Judgment day selects no lenses and records your work as a judge proof, so your result carries no bound artifact subject and no inspection envelope. The only allowed top-level fields are findings and evidence, and the only allowed finding fields are location, severity, claim, evidence_class, causal_disposition, and proof_refs. Never emit summary, skill_resolution, or any other unknown field. Keep orchestration metadata outside the native result JSON; evidence contains only genuine inspection evidence.

Return {"findings":[],"evidence":["what was inspected"]} when clean.`, nativeReviewerResultSchema)
}
