package sdd

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

const (
	// sharedOrchestratorSectionsAsset holds the canonical body of every
	// orchestrator subsection that each runtime states identically. #3817: the
	// contract lives in twelve hand-maintained near-duplicates, and a single
	// edit to one of them is how ten runtimes were left carrying a contradicted
	// dispatcher guard. Sections that genuinely differ per runtime stay in the
	// runtime asset; only the ones measured as non-drifted moved here.
	sharedOrchestratorSectionsAsset = "skills/_shared/sdd-orchestrator-sections.md"
	sharedOrchestratorSectionOpen   = "{{GENTLE_AI_SDD_SECTION:"
	// stopExitsMarker stands for the single text of the three-exit stops. The
	// shared gates section carries it and substituteSharedOrchestratorSections
	// resolves it with agentguidance.StopExits (DD-12).
	stopExitsMarker = "{{AXIOM_STOP_EXITS}}"
	// delegationRulesMarker and delegationTriggersMarker stand for the delegation
	// rules table and the conditional delegation triggers, whose single source is
	// agentguidance.DelegationRules / DelegationTriggers (decision P2).
	delegationRulesMarker    = "{{AXIOM_DELEGATION_RULES}}"
	delegationTriggersMarker = "{{AXIOM_DELEGATION_TRIGGERS}}"

	openCodeBackgroundPolicyAsset  = "opencode/background-subagents.md"
	openCodeBackgroundPolicyMarker = "<!-- axiom:opencode-background-subagents -->"
	openCodeBackgroundPolicyEnd    = "<!-- /axiom:opencode-background-subagents -->"
	// Marcadores que escribieron las versiones con el nombre anterior: solo se
	// reconocen para sustituirlos por los actuales, nunca se escriben.
	legacyOpenCodeBackgroundPolicyMarker = "<!-- gentle-ai:opencode-background-subagents -->"
	legacyOpenCodeBackgroundPolicyEnd    = "<!-- /gentle-ai:opencode-background-subagents -->"
)

// OrchestratorRenderOptions carries already-resolved prompt policy selections.
// The renderer does not resolve intent or runtime capability. Callers MUST set
// IncludeOpenCodeBackgroundPolicy only after a later resolution step has done
// that work; the zero value preserves the historical prompt bytes.
type OrchestratorRenderOptions struct {
	IncludeOpenCodeBackgroundPolicy bool

	// SharedPromptDir sustituye a SharedPromptDir(homeDir) como carpeta de los
	// prompts compartidos de las fases (SharedPromptDirFor). Vacío usa la ruta
	// global. En workspace apunta a .opencode/prompts/sdd (o .kilo/prompts/sdd).
	SharedPromptDir string
}

// sharedOrchestratorSection returns the canonical body for one shared section,
// or the empty string when the shared asset does not define it.
func sharedOrchestratorSection(name string) string {
	source := assets.MustRead(sharedOrchestratorSectionsAsset)
	start := strings.Index(source, "<!-- sdd-orchestrator-section:"+name+":start -->")
	end := strings.Index(source, "<!-- sdd-orchestrator-section:"+name+":end -->")
	if start < 0 || end < start {
		return ""
	}
	start += len("<!-- sdd-orchestrator-section:" + name + ":start -->")
	return strings.TrimSpace(source[start:end])
}

// sharedOrchestratorSectionPlaceholder matches one {{GENTLE_AI_SDD_SECTION:<name>}}.
var sharedOrchestratorSectionPlaceholder = regexp.MustCompile(`\{\{GENTLE_AI_SDD_SECTION:([^}]*)\}\}`)

// substituteSharedOrchestratorSections resolves every shared-section
// placeholder in one pass. An unresolvable placeholder panics rather than
// shipping a prompt with a literal template token in it, which is the failure
// mode the rendered goldens would otherwise hide.
//
// One pass, deliberately: a canonical body is literal text, so a placeholder
// appearing inside one is not a nested reference to expand. The earlier
// implementation rescanned from the start after each substitution, which would
// not terminate if a body ever contained a placeholder.
//
// The stop-exits marker is the one token a canonical body does carry. It is not
// a nested section reference: it is a distinct token that a second, explicit
// step resolves after the section pass, so the three-exit text has a single
// source (agentguidance.StopExits) shared with the ODD doctrine (REQ-26.8).
func substituteSharedOrchestratorSections(content string) string {
	content = sharedOrchestratorSectionPlaceholder.ReplaceAllStringFunc(content, func(match string) string {
		name := sharedOrchestratorSectionPlaceholder.FindStringSubmatch(match)[1]
		body := sharedOrchestratorSection(name)
		if body == "" {
			panic(fmt.Sprintf("sdd: shared orchestrator section %q has no canonical body", name))
		}
		return body
	})
	content = strings.ReplaceAll(content, stopExitsMarker, agentguidance.StopExits())
	// The delegation doctrine has one source too (agentguidance/delegation.go),
	// shared with the routing block and the migrated OpenCode prompt.
	content = strings.ReplaceAll(content, delegationRulesMarker, agentguidance.DelegationRules())
	return strings.ReplaceAll(content, delegationTriggersMarker, agentguidance.DelegationTriggers())
}

// composeOrchestratorPrompt is the renderer-owned source seam for every SDD
// orchestrator. It composes the selected historical asset before the existing
// managed-asset and runtime-identity substitutions.
func composeOrchestratorPrompt(agent model.AgentID, options ...OrchestratorRenderOptions) string {
	path := sddOrchestratorAsset(agent)
	// Shared sections resolve first: the fallback session preflight matches
	// exact clauses (init lookup, execution-mode producers) that now live in
	// shared bodies, so it has to see the expanded text, not the placeholder.
	content := substituteSharedOrchestratorSections(assets.MustRead(path))
	if usesFallbackSessionPreflight(agent) {
		content = composeFallbackSessionPreflight(content, agent)
	}
	var renderOptions OrchestratorRenderOptions
	if len(options) > 0 {
		renderOptions = options[0]
	}
	if policy := renderOpenCodeBackgroundPolicy(agent, renderOptions); policy != "" {
		content = appendOpenCodeBackgroundPolicy(content, policy)
	}
	content = replacePiClosedSingleSelectRoute(content, agent)
	content = renderManagedAssetBodyFromContent(path, content)
	return bindRuntimeAgentIdentity(content, agent)
}

// Generic is also consumed by Pi, OpenClaw, and Trae. Select by identity,
// not asset path, so this cohort cannot change deferred runtime prompts.
func usesFallbackSessionPreflight(agent model.AgentID) bool {
	switch agent {
	case model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI, model.AgentCursor, model.AgentGeminiCLI,
		model.AgentAntigravity, model.AgentQwenCode, model.AgentHermes, model.AgentKimi, model.AgentKiroIDE, model.AgentCodex, model.AgentWindsurf:
		return true
	default:
		return false
	}
}

// Replace only owned session-choice producers before composing shared sections.
// Historical assets remain intact for runtimes outside the migrated cohort.
func composeFallbackSessionPreflight(content string, agent model.AgentID) string {
	// Validate cardinality before removing any range: a duplicate nested in an
	// earlier range must not disappear before its own validation runs.
	policyHeading := "### Artifact Store Policy"
	if agent == model.AgentCodex {
		policyHeading = "### Artifact store (engram default)"
	}
	for _, heading := range []string{policyHeading, "### Commands", "### Execution Mode", "### Artifact Store Mode", "### Delivery Strategy", "### Chain Strategy"} {
		if _, err := sddSessionPreflightAnchorIndex(content, heading); err != nil {
			panic(fmt.Sprintf("sdd: fallback source %s: %v", heading, err))
		}
	}
	for _, section := range []struct{ start, end, body string }{
		{"### Artifact Store Policy", "### Commands", "The session artifact store is always `hybrid` and is never asked. Pass the `artifactStore` that `axiom sdd status` reports to every phase; never detect or default a separate choice."},
		{"### Artifact Store Mode", "### Delivery Strategy", "Pass the `artifactStore` that `axiom sdd status` reports as `artifact_store.mode` to every sub-agent launch; the session `hybrid` never overrides it."},
	} {
		if agent == model.AgentCodex && section.start == "### Artifact Store Policy" {
			const tableHeading = "### Artifact store (engram default)\n"
			content = strings.Replace(content, tableHeading, "### Artifact store (status-reported backend)\n\nUse the following topic keys and retrieval calls only in `engram` or `hybrid` mode. In `openspec` mode, use OpenSpec artifacts instead; do not call Engram for this table.\n", 1)
			continue
		}
		start := strings.Index(content, section.start+"\n")
		end := strings.Index(content, section.end+"\n")
		if start < 0 || end <= start {
			panic("sdd: missing fallback session policy section: " + section.start)
		}
		content = content[:start] + section.start + "\n\n" + section.body + "\n\n" + content[end:]
	}
	// Keep runtime-specific execution semantics and phase approval rules.
	executionEnd := "In **Interactive** mode, between phases:"
	if agent == model.AgentKimi || agent == model.AgentKiroIDE {
		executionEnd = "Interactive approval is phase-scoped."
	}
	for _, bounds := range [][2]string{
		{"When the user invokes `/sdd-new`", "- **Automatic**"},
		{"No mode is assumed by default: `multirepo` always runs **Interactive** and never asks; `monorepo-embedded` asks it in the session preflight.", executionEnd},
	} {
		start := strings.Index(content, bounds[0])
		end := strings.Index(content, bounds[1])
		if strings.Count(content, bounds[0]) != 1 || strings.Count(content, bounds[1]) != 1 || end <= start {
			panic("sdd: missing or ambiguous fallback execution choice producer")
		}
		content = content[:start] + "Use the execution mode cached by SDD Session Preflight.\n\n" + content[end:]
	}
	for _, replacement := range []struct {
		agents   []model.AgentID
		old, new string
	}{
		{[]model.AgentID{model.AgentCursor}, "**Interactive** is the default behavior", "**Interactive** uses the preflight choice"},
		// Copilot CLI usa la misma plantilla genérica que VS Code Copilot, así que
		// lleva la misma cláusula de origen.
		{[]model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI}, "Artifact store: default `engram` when available.", "Artifact store: use the `artifactStore` that `axiom sdd status` reports."},
	} {
		want := 0
		if slices.Contains(replacement.agents, agent) {
			want = 1
		}
		if strings.Count(content, replacement.old) != want {
			panic("sdd: missing or ambiguous fallback source clause: " + replacement.old)
		}
		content = strings.Replace(content, replacement.old, replacement.new, want)
	}
	// Match each runtime's complete source clause, never a permissive substring.
	legacyInitAction := "run `sdd-init` FIRST (delegate to sdd-init sub-agent)"
	initAction := "delegate to `sdd-init`"
	switch agent {
	case model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI, model.AgentCursor, model.AgentGeminiCLI, model.AgentQwenCode, model.AgentHermes, model.AgentCodex:
	case model.AgentWindsurf:
		legacyInitAction = "run the `sdd-init` phase inline FIRST"
		initAction = "run the `sdd-init` phase inline"
	case model.AgentAntigravity:
		legacyInitAction = "invoke the `sdd-init` phase subagent FIRST"
		initAction = "invoke the `sdd-init` phase subagent"
	case model.AgentKimi:
		legacyInitAction = "run `sdd-init` FIRST by launching the `sdd-init` custom agent"
		initAction = "launch the `sdd-init` custom agent"
	case model.AgentKiroIDE:
		legacyInitAction = "run `sdd-init` FIRST (load the sdd-init skill and execute it)"
		initAction = "load the `sdd-init` skill and execute it in the native Kiro phase context"
	default:
		panic("sdd: unsupported fallback runtime: " + string(agent))
	}
	legacyInitLookup := "1. Search Engram: `mem_search(query: \"sdd-init/{project}\", project: \"{project}\")`\n2. If found → init was done, proceed normally\n3. If NOT found → " + legacyInitAction + ", THEN proceed with the requested command"
	if strings.Count(content, legacyInitLookup) != 1 {
		panic("sdd: missing or ambiguous fallback init lookup")
	}
	content = strings.Replace(content, legacyInitLookup, "1. Use the `artifactStore` that `axiom sdd status` reports. In `openspec` mode, check project context and testing capabilities in `openspec/config.yaml` without calling Engram. In `engram` mode, search `sdd-init/{project}` in Engram. In `hybrid` mode, check both stores.\n2. If the selected store contains initialized project context and testing capabilities, proceed normally; a directory alone is not initialization evidence.\n3. If initialization is missing, "+initAction+" with the cached `artifact_store.mode`, then proceed. If a selected backend is unavailable, STOP and report it; never silently change the reported store.", 1)
	projected, err := projectSDDSessionPreflightWithTool(content, "### Native SDD Dispatcher Guard", "")
	if err != nil {
		panic(err)
	}
	return projected
}

const genericFallbackOnlyNativeRoute = "- Native route: This variant has no classified native question UI for this contract; always use the plain chat or terminal fallback below. When the closed domain of a single-select envelope is unrepresentable here, fall through to the Fallback clause below."

// piClosedSingleSelectNativeRoute is the Pi route for a strictly closed single-select envelope.
const piClosedSingleSelectNativeRoute = "- Native route: For every strictly closed single-select envelope, use ask_user_choice only when the interactive Pi TUI can represent its complete one-question 2-4 ordered-option domain. Pass each user-facing label and description with the envelope-owned canonical option token as value. The selector returns exactly one value; map it to the exact envelope-owned choice once, then select any envelope-owned continuation or invocation once where present. It has no custom/free-text or multi-select path. If the native TUI is unavailable or the envelope is not exactly representable, use the complete chat fallback. ask_user_question is the external open/free-text questionnaire and must not be used for a closed domain; open/free-text questionnaires may use ask_user_question when exactly representable."

func replacePiClosedSingleSelectRoute(content string, agent model.AgentID) string {
	if agent != model.AgentPi {
		return content
	}
	if count := strings.Count(content, genericFallbackOnlyNativeRoute); count != 1 {
		panic(fmt.Sprintf("sdd: Pi native route source clause count = %d, want 1", count))
	}
	return strings.Replace(content, genericFallbackOnlyNativeRoute, piClosedSingleSelectNativeRoute, 1)
}

func composeOpenCodeOrchestratorPrompt(agent model.AgentID, options ...OrchestratorRenderOptions) (string, error) {
	if agent != model.AgentOpenCode && agent != model.AgentKilocode {
		return composeOrchestratorPrompt(agent, options...), nil
	}
	return projectSDDSessionPreflight(composeOrchestratorPrompt(agent, options...), "### SDD Entry Routing (MANDATORY)")
}

func renderOpenCodeBackgroundPolicy(agent model.AgentID, options ...OrchestratorRenderOptions) string {
	var renderOptions OrchestratorRenderOptions
	if len(options) > 0 {
		renderOptions = options[0]
	}
	if agent != model.AgentOpenCode || !renderOptions.IncludeOpenCodeBackgroundPolicy {
		return ""
	}
	return mustReadOpenCodeBackgroundPolicy()
}

func mustReadOpenCodeBackgroundPolicy() string {
	content := assets.MustRead(openCodeBackgroundPolicyAsset)
	if err := validateOpenCodeBackgroundPolicy(content, true); err != nil {
		panic(err.Error())
	}
	return content
}

// upgradeLegacyOpenCodeBackgroundPolicy reescribe los marcadores del nombre
// anterior de un prompt preservado para que el resto del flujo vea solo los
// actuales (sin duplicar el bloque).
func upgradeLegacyOpenCodeBackgroundPolicy(content string) string {
	content = strings.ReplaceAll(content, legacyOpenCodeBackgroundPolicyMarker, openCodeBackgroundPolicyMarker)
	return strings.ReplaceAll(content, legacyOpenCodeBackgroundPolicyEnd, openCodeBackgroundPolicyEnd)
}

func appendOpenCodeBackgroundPolicy(content, policy string) string {
	markerCount := strings.Count(content, openCodeBackgroundPolicyMarker)
	endCount := strings.Count(content, openCodeBackgroundPolicyEnd)
	if markerCount != 0 || endCount != 0 {
		if err := validateOpenCodeBackgroundPolicy(content, false); err != nil {
			panic(err.Error())
		}
		return content
	}

	if err := validateOpenCodeBackgroundPolicy(policy, true); err != nil {
		panic(err.Error())
	}
	return strings.TrimRight(content, "\n") + "\n\n" + policy + "\n"
}

func validateOpenCodeBackgroundPolicy(content string, standalone bool) error {
	trimmed := strings.TrimSpace(content)
	start := strings.Index(trimmed, openCodeBackgroundPolicyMarker)
	end := strings.Index(trimmed, openCodeBackgroundPolicyEnd)
	if strings.Count(trimmed, openCodeBackgroundPolicyMarker) != 1 ||
		strings.Count(trimmed, openCodeBackgroundPolicyEnd) != 1 || start < 0 || end <= start ||
		(start+len(openCodeBackgroundPolicyMarker) < len(trimmed) && trimmed[start+len(openCodeBackgroundPolicyMarker)] != '\n') ||
		(end > 0 && trimmed[end-1] != '\n') ||
		(start > 0 && trimmed[start-1] != '\n') ||
		(end+len(openCodeBackgroundPolicyEnd) < len(trimmed) && trimmed[end+len(openCodeBackgroundPolicyEnd)] != '\n') {
		return fmt.Errorf("sdd: inconsistently marked OpenCode background policy")
	}
	if standalone && (start != 0 || end+len(openCodeBackgroundPolicyEnd) != len(trimmed)) {
		return fmt.Errorf("assets: OpenCode background policy must contain only its marked section")
	}
	if strings.TrimSpace(trimmed[start+len(openCodeBackgroundPolicyMarker):end]) == "" {
		return fmt.Errorf("sdd: empty OpenCode background policy")
	}
	return nil
}

// sddOrchestratorAsset returns the embedded asset path for the SDD orchestrator
// content based on the agent. Agent-specific assets take priority; generic is fallback.
func sddOrchestratorAsset(agent model.AgentID) string {
	switch agent {
	case model.AgentClaudeCode:
		return "claude/sdd-orchestrator.md"
	case model.AgentGeminiCLI:
		return "gemini/sdd-orchestrator.md"
	case model.AgentCodex:
		return "codex/sdd-orchestrator.md"
	case model.AgentAntigravity:
		return "antigravity/sdd-orchestrator.md"
	case model.AgentWindsurf:
		return "windsurf/sdd-orchestrator.md"
	case model.AgentCursor:
		return "cursor/sdd-orchestrator.md"
	case model.AgentKimi:
		return "kimi/sdd-orchestrator.md"
	case model.AgentQwenCode:
		return "qwen/sdd-orchestrator.md"
	case model.AgentKiroIDE:
		return "kiro/sdd-orchestrator.md"
	case model.AgentHermes:
		return "hermes/sdd-orchestrator.md"
	case model.AgentOpenCode, model.AgentKilocode:
		return "opencode/sdd-orchestrator.md"
	default:
		return "generic/sdd-orchestrator.md"
	}
}
