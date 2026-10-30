package sdd

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// codexWorkspaceReplacements son las frases del orquestador de Codex que describen
// la instalación global (rutas en ~/.codex y los ficheros de perfil de modelo) y que
// en un proyecto dejan de ser ciertas. Van de la más larga a la más corta para que
// una frase completa se sustituya antes que la ruta suelta que contiene. Con la
// instalación global el texto no se toca.
//
// En un proyecto no se escriben los ficheros <carril>.config.toml: Codex ignora la
// clave profiles de la configuración de proyecto. Tampoco hay ~/.codex: la
// configuración es .codex/config.toml y las skills están en el directorio de skills
// del proyecto.
var codexWorkspaceReplacements = []struct{ old, new string }{
	{
		"Note: the `~/.codex/<tier>.config.toml` profile files apply to whole CLI sessions launched with `codex --profile <name>`. They do NOT apply to spawned sub-agents — for those, pass `reasoning_effort` and `model` directly as shown above.",
		"Note: a workspace install does not write model-selection profile files (Codex ignores `profiles` in project configuration). For spawned sub-agents, pass `reasoning_effort` and `model` directly as shown above.",
	},
	{
		"When the Codex CLI is installed and supports GPT-5.6 profiles, axiom writes three SDD model-selection profile files into `~/.codex/` during installation. Without the CLI, axiom still configures shared files but leaves existing profile files unchanged. Each profile pins both a `model` and a `model_reasoning_effort` so Codex picks the right tier for each carril.",
		"A workspace install does not write model-selection profile files: Codex ignores `profiles` in project configuration. The carriles below only name the tiers; each one pins a `model` and a `reasoning_effort` to pass to `spawn_agent`.",
	},
	{
		"These profile files apply to whole CLI sessions: `codex --profile <name> \"<prompt>\"`. They do NOT apply to spawned sub-agents. When delegating a phase via `spawn_agent`, pass the tier's effort directly as `reasoning_effort` (with `fork_turns: \"none\"`), using the same tier values below.",
		"When delegating a phase via `spawn_agent`, pass the tier's effort directly as `reasoning_effort` (with `fork_turns: \"none\"`), using the tier values below.",
	},
	{
		"`~/.codex/skills/_shared/` (global) or `.agent/skills/_shared/` (workspace)",
		"`~/.codex/skills/_shared/`",
	},
	{"~/.codex/config.toml", ".codex/config.toml"},
}

// adaptCodexOrchestratorForWorkspace reescribe el orquestador de Codex para una
// instalación de proyecto: la configuración es .codex/config.toml, las skills están
// en skillsDir (relativo a la raíz del proyecto, con barras normales) y no hay
// ficheros de perfil de modelo. Con otro agente, o con la instalación global, no
// cambia nada.
func adaptCodexOrchestratorForWorkspace(content string, adapter agents.Adapter, workspaceRoot string) string {
	if adapter.Agent() != model.AgentCodex || strings.TrimSpace(workspaceRoot) == "" {
		return content
	}
	for _, replacement := range codexWorkspaceReplacements {
		content = strings.ReplaceAll(content, replacement.old, replacement.new)
	}
	rel, err := filepath.Rel(workspaceRoot, adapter.SkillsDir(workspaceRoot))
	if err != nil || rel == "" {
		return content
	}
	return strings.ReplaceAll(content, "~/.codex/skills", filepath.ToSlash(rel))
}
