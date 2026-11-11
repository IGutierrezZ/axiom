package sdd

import (
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// copilotModelFrontmatterPlaceholder es el marcador de las plantillas
// `.agent.md` de Copilot (assets/copilot/agents). Se sustituye por las líneas
// `model:` y `reasoning-effort:` de la fase, o desaparece con su salto de línea
// cuando la fase no tiene asignación y el agente hereda el modelo de la
// conversación.
const copilotModelFrontmatterPlaceholder = "{{COPILOT_MODEL_FRONTMATTER}}"

// copilotAgentFileSuffix es la extensión de los agentes personalizados de VS Code
// y de Copilot CLI; ambos leen el mismo fichero.
const copilotAgentFileSuffix = ".agent.md"

// copilotPlainScalar admite en YAML sin comillas los valores de modelo y esfuerzo
// con la forma habitual («Claude Sonnet 4.6 (copilot)», «high»). Cualquier otro
// valor va entre comillas dobles, de modo que los analizadores de frontmatter más
// simples del CLI y VS Code lean lo mismo.
var copilotPlainScalar = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()/+-]*$`)

// copilotAgentPhase devuelve la fase SDD de un fichero de agente de Copilot
// (`sdd-apply.agent.md` → `sdd-apply`).
func copilotAgentPhase(fileName string) string {
	return strings.TrimSuffix(fileName, copilotAgentFileSuffix)
}

// injectCopilotModelFrontmatter rellena el marcador de una plantilla de agente de
// Copilot con la asignación de su fase. Sin modelo no escribe ninguna de las dos
// claves: un `reasoning-effort` sin `model` tampoco tendría a qué aplicarse.
func injectCopilotModelFrontmatter(content string, assignment model.CopilotPhaseAssignment) string {
	line := renderCopilotModelFrontmatter(assignment)
	if line == "" {
		content = strings.ReplaceAll(content, copilotModelFrontmatterPlaceholder+"\r\n", "")
		content = strings.ReplaceAll(content, copilotModelFrontmatterPlaceholder+"\n", "")
		return strings.ReplaceAll(content, copilotModelFrontmatterPlaceholder, "")
	}
	return strings.ReplaceAll(content, copilotModelFrontmatterPlaceholder, line)
}

func renderCopilotModelFrontmatter(assignment model.CopilotPhaseAssignment) string {
	modelName := strings.TrimSpace(assignment.Model)
	if modelName == "" {
		return ""
	}
	lines := []string{"model: " + copilotYAMLScalar(modelName)}
	if effort := strings.TrimSpace(assignment.Effort); effort != "" {
		lines = append(lines, "reasoning-effort: "+copilotYAMLScalar(effort))
	}
	return strings.Join(lines, "\n")
}

func copilotYAMLScalar(value string) string {
	if copilotPlainScalar.MatchString(value) {
		return value
	}
	// Los caracteres de control los rechaza model.ValidateCopilotPhaseAssignments
	// antes de llegar aquí; con las comillas dobles basta escapar la barra y la
	// comilla.
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
