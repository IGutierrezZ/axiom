package model

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// CopilotPhaseAssignment configura el modelo y el esfuerzo de razonamiento del
// agente de una fase SDD para GitHub Copilot (VS Code y Copilot CLI comparten el
// mismo `.agent.md`).
//
// Model es texto libre con el formato de nombre de VS Code, p. ej.
// "Claude Sonnet 4.6 (copilot)": el catálogo de Copilot cambia a menudo y no hay
// lista cerrada que validar. Effort vacío significa que el agente no lleva
// `reasoning-effort` y hereda el de la conversación.
type CopilotPhaseAssignment struct {
	Model  string `json:"model"`
	Effort string `json:"effort,omitempty"`
}

// copilotPhaseNames son las fases con agente propio en Copilot: el coordinador y
// cada fase SDD y de Judgment Day. A diferencia de Claude no hay fase "default":
// Copilot no tiene un agente genérico al que aplicarla.
var copilotPhaseNames = []string{
	"orchestrator",
	"sdd-explore", "sdd-research", "sdd-propose", "sdd-spec", "sdd-design",
	"sdd-tasks", "sdd-apply", "sdd-verify", "sdd-archive", "sdd-onboard",
	"jd-judge-a", "jd-judge-b", "jd-fix-agent",
}

// CopilotPhaseNames devuelve las fases que admiten asignación de modelo en
// Copilot, en el orden del flujo SDD.
func CopilotPhaseNames() []string {
	return append([]string(nil), copilotPhaseNames...)
}

func isCopilotPhase(name string) bool {
	for _, phase := range copilotPhaseNames {
		if phase == name {
			return true
		}
	}
	return false
}

// validate comprueba que el modelo no esté vacío y que ni el modelo ni el
// esfuerzo contengan saltos de línea u otros caracteres de control: ambos acaban
// en el frontmatter del `.agent.md` y un salto de línea inyectaría claves.
func (a CopilotPhaseAssignment) validate() error {
	if strings.TrimSpace(a.Model) == "" {
		return fmt.Errorf("el modelo no puede estar vacío")
	}
	if hasControlRune(a.Model) {
		return fmt.Errorf("el modelo %q contiene caracteres de control", a.Model)
	}
	if hasControlRune(a.Effort) {
		return fmt.Errorf("el esfuerzo %q contiene caracteres de control", a.Effort)
	}
	return nil
}

func hasControlRune(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}

// ValidateCopilotPhaseAssignments rechaza fases desconocidas y asignaciones
// inválidas. Informa de todas las violaciones, en el orden de las fases, no solo
// de la primera.
func ValidateCopilotPhaseAssignments(assignments map[string]CopilotPhaseAssignment) error {
	var problems []string
	for _, phase := range copilotPhaseNames {
		if assignment, ok := assignments[phase]; ok {
			if err := assignment.validate(); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", phase, err))
			}
		}
	}
	var unknown []string
	for phase := range assignments {
		if !isCopilotPhase(phase) {
			unknown = append(unknown, phase)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		problems = append(problems, fmt.Sprintf("fases desconocidas: %s", strings.Join(unknown, ", ")))
	}
	if len(problems) > 0 {
		return fmt.Errorf("asignaciones de Copilot no válidas: %s", strings.Join(problems, "; "))
	}
	return nil
}
