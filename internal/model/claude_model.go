package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ClaudeModelAlias represents one of the Claude model tiers used for
// per-phase model assignments in the SDD orchestrator.
//
// Only four values are valid: ClaudeModelFable, ClaudeModelOpus,
// ClaudeModelSonnet, ClaudeModelHaiku.
type ClaudeModelAlias string

const (
	// ClaudeModelFable is the highest-reasoning tier, above opus, for the most
	// demanding architectural and review work. Maps to the current
	// claude-fable-* family.
	ClaudeModelFable ClaudeModelAlias = "fable"

	// ClaudeModelOpus is the high-capability tier, best for architectural decisions
	// and orchestration. Maps to the current claude-opus-* family.
	ClaudeModelOpus ClaudeModelAlias = "opus"

	// ClaudeModelSonnet is the balanced tier, suitable for most SDD phases.
	// Maps to the current claude-sonnet-* family.
	ClaudeModelSonnet ClaudeModelAlias = "sonnet"

	// ClaudeModelHaiku is the lightweight tier, ideal for mechanical tasks like
	// archiving or simple copy work. Maps to the current claude-haiku-* family.
	ClaudeModelHaiku ClaudeModelAlias = "haiku"
)

// String returns the string representation of the alias.
func (a ClaudeModelAlias) String() string {
	return string(a)
}

// Valid reports whether the alias is one of the known Claude model tiers.
func (a ClaudeModelAlias) Valid() bool {
	switch a {
	case ClaudeModelFable, ClaudeModelOpus, ClaudeModelSonnet, ClaudeModelHaiku:
		return true
	default:
		return false
	}
}

// ClaudeEffort represents a Claude Code subagent effort frontmatter value.
// The empty value means inherit the session/model default and should not be
// written as frontmatter.
type ClaudeEffort string

const (
	ClaudeEffortDefault ClaudeEffort = ""
	ClaudeEffortLow     ClaudeEffort = "low"
	ClaudeEffortMedium  ClaudeEffort = "medium"
	ClaudeEffortHigh    ClaudeEffort = "high"
	ClaudeEffortXHigh   ClaudeEffort = "xhigh"
	ClaudeEffortMax     ClaudeEffort = "max"
)

// Valid reports whether the effort is one of Claude Code's known effort values.
func (e ClaudeEffort) Valid() bool {
	switch e {
	case ClaudeEffortDefault, ClaudeEffortLow, ClaudeEffortMedium, ClaudeEffortHigh, ClaudeEffortXHigh, ClaudeEffortMax:
		return true
	default:
		return false
	}
}

// ClaudeEffortsForModel returns the official Claude Code effort choices for a
// model alias. The first entry is always the default/empty value.
func ClaudeEffortsForModel(alias ClaudeModelAlias) []ClaudeEffort {
	switch alias {
	case ClaudeModelFable, ClaudeModelOpus:
		return []ClaudeEffort{ClaudeEffortDefault, ClaudeEffortLow, ClaudeEffortMedium, ClaudeEffortHigh, ClaudeEffortXHigh, ClaudeEffortMax}
	case ClaudeModelSonnet:
		return []ClaudeEffort{ClaudeEffortDefault, ClaudeEffortLow, ClaudeEffortMedium, ClaudeEffortHigh, ClaudeEffortXHigh, ClaudeEffortMax}
	case ClaudeModelHaiku:
		return []ClaudeEffort{ClaudeEffortDefault}
	default:
		return []ClaudeEffort{ClaudeEffortDefault}
	}
}

// ClaudeEffortAllowedForModel reports whether effort is valid for alias.
func ClaudeEffortAllowedForModel(alias ClaudeModelAlias, effort ClaudeEffort) bool {
	if !effort.Valid() {
		return false
	}
	for _, allowed := range ClaudeEffortsForModel(alias) {
		if effort == allowed {
			return true
		}
	}
	return false
}

// ClaudePhaseAssignment configures the Claude Code model and effort for one
// SDD/JD phase subagent. Empty Effort means inherit the session/model default.
type ClaudePhaseAssignment struct {
	Model  ClaudeModelAlias `json:"model"`
	Effort ClaudeEffort     `json:"effort,omitempty"`
}

// Valid reports whether the model is valid and the effort is supported by it.
func (a ClaudePhaseAssignment) Valid() bool {
	return a.Model.Valid() && ClaudeEffortAllowedForModel(a.Model, a.Effort)
}

// ClaudePhaseAssignmentsFromLegacy converts the historical model-only map into
// model+effort assignments with default effort. Invalid aliases are ignored.
func ClaudePhaseAssignmentsFromLegacy(assignments map[string]ClaudeModelAlias) map[string]ClaudePhaseAssignment {
	if assignments == nil {
		return nil
	}
	converted := make(map[string]ClaudePhaseAssignment, len(assignments))
	for key, alias := range assignments {
		if !alias.Valid() {
			continue
		}
		converted[key] = ClaudePhaseAssignment{Model: alias, Effort: ClaudeEffortDefault}
	}
	return converted
}

// ClaudePhaseAssignmentsFromModelPreset converts a model-only preset to the new
// assignment shape with default effort.
func ClaudePhaseAssignmentsFromModelPreset(assignments map[string]ClaudeModelAlias) map[string]ClaudePhaseAssignment {
	return ClaudePhaseAssignmentsFromLegacy(assignments)
}

// ClaudeModelPresetBalanced returns the default model assignment table.
// It balances cost and capability for Claude sub-agents: architecture phases use opus;
// implementation and validation use sonnet; archiving uses haiku.
func ClaudeModelPresetBalanced() map[string]ClaudeModelAlias {
	return map[string]ClaudeModelAlias{
		"orchestrator": ClaudeModelOpus,
		"sdd-explore":  ClaudeModelSonnet,
		"sdd-research": ClaudeModelSonnet,
		"sdd-propose":  ClaudeModelOpus,
		"sdd-spec":     ClaudeModelSonnet,
		"sdd-design":   ClaudeModelOpus,
		"sdd-tasks":    ClaudeModelSonnet,
		"sdd-apply":    ClaudeModelSonnet,
		"sdd-verify":   ClaudeModelSonnet,
		"sdd-archive":  ClaudeModelHaiku,
		"sdd-onboard":  ClaudeModelHaiku,
		"jd-judge-a":   ClaudeModelSonnet,
		"jd-judge-b":   ClaudeModelSonnet,
		"jd-fix-agent": ClaudeModelSonnet,
		"default":      ClaudeModelSonnet,
	}
}

// ClaudeModelPresetPerformance returns a model assignment table optimised for
// output quality. Architecture, planning, and verification phases all use opus.
func ClaudeModelPresetPerformance() map[string]ClaudeModelAlias {
	return map[string]ClaudeModelAlias{
		"orchestrator": ClaudeModelOpus,
		"sdd-explore":  ClaudeModelSonnet,
		"sdd-research": ClaudeModelSonnet,
		"sdd-propose":  ClaudeModelOpus,
		"sdd-spec":     ClaudeModelSonnet,
		"sdd-design":   ClaudeModelOpus,
		"sdd-tasks":    ClaudeModelSonnet,
		"sdd-apply":    ClaudeModelSonnet,
		"sdd-verify":   ClaudeModelOpus,
		"sdd-archive":  ClaudeModelHaiku,
		"sdd-onboard":  ClaudeModelHaiku,
		"jd-judge-a":   ClaudeModelOpus,
		"jd-judge-b":   ClaudeModelOpus,
		"jd-fix-agent": ClaudeModelOpus,
		"default":      ClaudeModelSonnet,
	}
}

// ClaudeModelPresetEconomy returns a model assignment table optimised for cost.
// SDD phases use sonnet except archive; JD agents use haiku for maximum savings.
func ClaudeModelPresetEconomy() map[string]ClaudeModelAlias {
	return map[string]ClaudeModelAlias{
		"orchestrator": ClaudeModelSonnet,
		"sdd-explore":  ClaudeModelSonnet,
		"sdd-research": ClaudeModelSonnet,
		"sdd-propose":  ClaudeModelSonnet,
		"sdd-spec":     ClaudeModelSonnet,
		"sdd-design":   ClaudeModelSonnet,
		"sdd-tasks":    ClaudeModelSonnet,
		"sdd-apply":    ClaudeModelSonnet,
		"sdd-verify":   ClaudeModelSonnet,
		"sdd-archive":  ClaudeModelHaiku,
		"sdd-onboard":  ClaudeModelHaiku,
		"jd-judge-a":   ClaudeModelHaiku,
		"jd-judge-b":   ClaudeModelHaiku,
		"jd-fix-agent": ClaudeModelHaiku,
		"default":      ClaudeModelSonnet,
	}
}

// ClaudeModelPresetDiversity returns a model assignment table optimised for
// perspective diversity in judgment-day reviews. Judge A uses opus for deep
// architectural reasoning, Judge B uses haiku for fast pattern matching,
// and the fix agent uses sonnet for balanced implementation.
func ClaudeModelPresetDiversity() map[string]ClaudeModelAlias {
	base := maps.Clone(ClaudeModelPresetBalanced())
	base["jd-judge-a"] = ClaudeModelOpus
	base["jd-judge-b"] = ClaudeModelHaiku
	base["jd-fix-agent"] = ClaudeModelSonnet
	return base
}

// claudePhaseNames son las claves de asignación que admite Claude Code, en el orden
// en que las muestran la TUI y el dashboard: las fases SDD, los agentes de Judgment
// Day, los roles de la revisión nativa y la delegación general («default»). No
// incluye «orchestrator»: Claude Code controla el modelo de la sesión principal.
var claudePhaseNames = []string{
	"sdd-explore", "sdd-research", "sdd-propose", "sdd-spec", "sdd-design",
	"sdd-tasks", "sdd-apply", "sdd-verify", "sdd-archive", "sdd-onboard",
	"jd-judge-a", "jd-judge-b", "jd-fix-agent",
	"risk", "readability", "reliability", "resilience", "refuter", "validator",
	"default",
}

// ClaudePhaseNames devuelve las claves de asignación de Claude Code en orden de
// presentación.
func ClaudePhaseNames() []string {
	return append([]string(nil), claudePhaseNames...)
}

// ClaudePhaseAssignmentProblems devuelve una frase por cada violación: claves
// desconocidas y asignaciones cuyo modelo no es un alias conocido o cuyo esfuerzo no
// admite ese modelo. Las informa todas, en el orden de las fases, no solo la primera.
func ClaudePhaseAssignmentProblems(assignments map[string]ClaudePhaseAssignment) []string {
	var problems []string
	known := make(map[string]bool, len(claudePhaseNames))
	for _, phase := range claudePhaseNames {
		known[phase] = true
		assignment, ok := assignments[phase]
		if !ok {
			continue
		}
		switch {
		case !assignment.Model.Valid():
			problems = append(problems, fmt.Sprintf("%s: el modelo %q no es válido", phase, assignment.Model))
		case !ClaudeEffortAllowedForModel(assignment.Model, assignment.Effort):
			problems = append(problems, fmt.Sprintf("%s: el esfuerzo %q no es válido para %s", phase, assignment.Effort, assignment.Model))
		}
	}
	var unknown []string
	for phase := range assignments {
		if !known[phase] {
			unknown = append(unknown, phase)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		problems = append(problems, fmt.Sprintf("fases desconocidas: %s", strings.Join(unknown, ", ")))
	}
	return problems
}
