package sdd

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
)

// workspaceConventionsClause reconoce, en los orquestadores genéricos y de
// Cursor, Kiro y otros (el de Gemini usa el marcador {{AXIOM_SKILLS_DIR}}), la frase que sitúa los ficheros de convención en la carpeta
// de skills global del agente o, en el proyecto, en `.agent/skills/_shared/`: una
// carpeta que ningún agente usa. Admite las dos formas que traen los orquestadores:
// una ruta entre comillas (`~/.gemini/skills/_shared/`) o «the [agent's] global skills
// directory».
var workspaceConventionsClause = regexp.MustCompile(
	"(?:`[^`\n]+`|the (?:agent's )?global skills directory)" +
		` \(global\) or ` + "`" + `\.agent/skills/_shared/` + "`" + ` \(workspace\)`)

// adaptConventionsClauseForWorkspace reescribe esa frase para una instalación de
// proyecto: los ficheros de convención están en la carpeta de skills efectiva del
// agente (adapter.SkillsDir de la raíz: .agents/skills compartida, .claude/skills,
// .kiro/skills...), que es donde los escribe el componente de skills. Con la
// instalación global, o si el texto no trae la frase, no cambia nada.
func adaptConventionsClauseForWorkspace(content string, adapter agents.Adapter, workspaceRoot string) string {
	if strings.TrimSpace(workspaceRoot) == "" || !workspaceConventionsClause.MatchString(content) {
		return content
	}
	rel, err := filepath.Rel(workspaceRoot, adapter.SkillsDir(workspaceRoot))
	if err != nil || rel == "" || rel == "." {
		return content
	}
	return workspaceConventionsClause.ReplaceAllString(content, "`"+filepath.ToSlash(rel)+"/_shared/`")
}
