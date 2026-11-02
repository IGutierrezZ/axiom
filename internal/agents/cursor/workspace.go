package cursor

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/sharedskills"
)

// Distribución de proyecto de Cursor (documentación oficial,
// cursor.com/docs/context/skills): las skills de proyecto se cargan desde
// .agents/skills y .cursor/skills, y por compatibilidad también desde .claude/skills
// y .codex/skills. El resto de la configuración de Axiom (reglas, agentes, MCP) ya
// vive en .cursor/ y no cambia.

// WorkspaceAdapter es el adaptador de Cursor ligado a la raíz de un proyecto. Para
// esa raíz sitúa las skills en la carpeta compartida .agents/skills; para cualquier
// otra carpeta (la de usuario) da las rutas globales de siempre.
type WorkspaceAdapter struct {
	*Adapter
	root string
}

// WithWorkspace devuelve el adaptador ligado a la raíz de un proyecto.
func (a *Adapter) WithWorkspace(root string) *WorkspaceAdapter {
	return &WorkspaceAdapter{Adapter: a, root: root}
}

// inWorkspace indica si dir es la raíz del proyecto a la que está ligado el
// adaptador.
func (a *WorkspaceAdapter) inWorkspace(dir string) bool {
	return a.root != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.root)
}

// SkillsDir es ~/.cursor/skills en la carpeta de usuario y, en el proyecto, la
// compartida .agents/skills: una sola copia evita que Cursor vea las skills
// duplicadas entre .cursor/skills y .agents/skills.
func (a *WorkspaceAdapter) SkillsDir(dir string) string {
	if a.inWorkspace(dir) {
		return sharedskills.Dir(dir)
	}
	return a.Adapter.SkillsDir(dir)
}

// LegacyWorkspaceSkillsDirs lista la carpeta de skills que usaban las instalaciones
// de proyecto anteriores (.cursor/skills), ahora sustituida por .agents/skills. Solo
// se retira lo que se prueba que es de Axiom. Fuera del proyecto no devuelve nada.
func (a *WorkspaceAdapter) LegacyWorkspaceSkillsDirs(dir string) []string {
	if !a.inWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, ".cursor", "skills")}
}
