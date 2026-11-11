package sdd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/copilotconfig"
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

// InjectCopilotProjectAgents escribe en la carpeta de agentes del proyecto
// (.github/agents) los agentes por fase de Copilot cuando el proyecto no tiene
// VS Code Copilot instalado y solo está presente Copilot CLI, que no tiene modo
// workspace propio. adapter es el de VS Code Copilot ligado a workspaceDir, que solo
// aporta la carpeta y las plantillas: el contenido es byte a byte el que escribiría
// Inject para VS Code, de modo que quien escribe primero no cambia el resultado y
// el fichero se escribe una sola vez.
func InjectCopilotProjectAgents(workspaceDir string, adapter agents.Adapter, opts InjectOptions) (InjectionResult, error) {
	opts.WorkspaceDir = workspaceDir
	return writeSubAgents(workspaceDir, adapter, opts)
}

// retireLegacyCopilotUserAgents quita de la carpeta de agentes de usuario de Copilot
// (~/.copilot/agents y <COPILOT_HOME>/agents) los agentes por fase que una versión
// anterior de Axiom escribió ahí. Copilot CLI da prioridad al agente personal frente
// al del proyecto cuando coinciden los identificadores, así que un agente global sin
// modelo tapaba el elegido en el proyecto. Solo se retiran los ficheros con el nombre
// de una plantilla de Axiom (el mismo criterio que usa uninstall); los agentes que
// escribió el usuario con otros nombres se conservan. Sin ficheros no hace nada.
func retireLegacyCopilotUserAgents(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	var result InjectionResult
	switch adapter.Agent() {
	case model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI:
	default:
		return result, nil
	}
	// Con una carpeta de agentes (VS Code Copilot ligado a un proyecto) el adaptador
	// escribe ahí y no hay nada de usuario que retirar.
	if adapter.SubAgentsDir(homeDir) != "" {
		return result, nil
	}
	entries, err := assets.FS.ReadDir(copilotAgentsAssetDir)
	if err != nil {
		return result, fmt.Errorf("read embedded copilot agents dir: %w", err)
	}
	dirs := []string{filepath.Join(homeDir, ".copilot", "agents")}
	if root := filepath.Join(copilotconfig.Root(homeDir), "agents"); root != dirs[0] {
		dirs = append(dirs, root)
	}
	for _, dir := range dirs {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, statErr := os.Lstat(path)
			if statErr != nil || !info.Mode().IsRegular() {
				continue
			}
			if err := os.Remove(path); err != nil {
				return result, fmt.Errorf("retire legacy copilot agent %s: %w", path, err)
			}
			result.Changed = true
			result.Files = append(result.Files, path)
		}
		// La carpeta solo se quita si ha quedado vacía: puede tener agentes del usuario.
		if remaining, readErr := os.ReadDir(dir); readErr == nil && len(remaining) == 0 {
			_ = os.Remove(dir)
		}
	}
	return result, nil
}

// copilotAgentsAssetDir es la carpeta embebida con las plantillas de los agentes de Copilot.
const copilotAgentsAssetDir = "copilot/agents"
