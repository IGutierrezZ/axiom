package cli

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
)

// workspaceSkillsDirs lleva la cuenta de las carpetas de skills que una ejecución
// ya ha escrito. Con --scope workspace varios agentes comparten .agents/skills
// (Codex, VS Code, OpenCode, Kilo, Gemini y Cursor): se escribe una sola vez, con el
// mismo contenido para todos, y el resto de agentes de esa carpeta no la repiten.
//
// Con el ámbito global no deduplica nada: cada agente escribe en su carpeta de
// usuario, como siempre, y la salida no cambia.
type workspaceSkillsDirs struct {
	scope   InstallScope
	written map[string]struct{}
}

func newWorkspaceSkillsDirs(scope InstallScope) *workspaceSkillsDirs {
	return &workspaceSkillsDirs{scope: scope, written: map[string]struct{}{}}
}

// claim devuelve true si skillsDir ya lo escribió otro agente de esta ejecución. Si
// no, lo anota para los siguientes y devuelve false. Una carpeta vacía (agente sin
// skills) no se anota nunca.
func (w *workspaceSkillsDirs) claim(skillsDir string) (already bool) {
	if w == nil || w.scope != ScopeWorkspace || skillsDir == "" {
		return false
	}
	key := filepath.Clean(skillsDir)
	if _, ok := w.written[key]; ok {
		return true
	}
	w.written[key] = struct{}{}
	return false
}

// claimSkills anota la carpeta de skills del adaptador para el componente de
// skills y devuelve true si ya la escribió otro agente.
func (w *workspaceSkillsDirs) claimSkills(adapter agents.Adapter, targetDir string) bool {
	if !adapter.SupportsSkills() {
		return false
	}
	return w.claim(adapter.SkillsDir(targetDir))
}

// claimSDD es claimSkills para el componente SDD: sdd.Inject solo escribe las
// skills si el adaptador admite instrucciones de sistema, así que un agente que no
// las admite no reclama la carpeta (la escribiría el siguiente).
func (w *workspaceSkillsDirs) claimSDD(adapter agents.Adapter, targetDir string) bool {
	if !adapter.SupportsSystemPrompt() {
		return false
	}
	return w.claimSkills(adapter, targetDir)
}

// legacySkillsAdapters devuelve los adaptadores ligados al proyecto que declaran
// carpetas de skills anteriores que migrar. Solo con --scope workspace: en global no
// hay nada que migrar.
func legacySkillsAdapters(scope InstallScope, adapters []agents.Adapter) []agents.Adapter {
	if scope != ScopeWorkspace {
		return nil
	}
	var selected []agents.Adapter
	for _, adapter := range adapters {
		if _, ok := adapter.(agents.LegacySkillsDirProvider); ok {
			selected = append(selected, adapter)
		}
	}
	return selected
}

// legacySkillsRetirementStep migra las instalaciones workspace anteriores, que
// dejaron una copia de las skills en la carpeta propia de cada agente
// (.opencode/skills, .kilo/skills, .gemini/skills, .cursor/skills, .github/skills,
// .codex/skills). Ahora hay una sola copia compartida en .agents/skills y esas
// carpetas harían ver cada skill dos veces. Solo retira una carpeta de skill cuando
// prueba que es de Axiom (nombre de skill gestionada y contenido byte a byte igual a lo
// que Axiom genera hoy); lo demás se conserva y se avisa. No falla la ejecución ni
// tiene vuelta atrás propia: los ficheros que puede borrar van en la copia de
// seguridad previa.
type legacySkillsRetirementStep struct {
	state        *runtimeState
	id           string
	workspaceDir string
	adapters     []agents.Adapter
}

func (s legacySkillsRetirementStep) ID() string { return s.id }

func (s legacySkillsRetirementStep) Run() error {
	for _, adapter := range s.adapters {
		provider := adapter.(agents.LegacySkillsDirProvider)
		shared := adapter.SkillsDir(s.workspaceDir)
		for _, legacyDir := range provider.LegacyWorkspaceSkillsDirs(s.workspaceDir) {
			result := sdd.RetireLegacyWorkspaceSkills(legacyDir, shared, adapter.Agent())
			if s.state == nil {
				continue
			}
			s.state.retiredFiles = append(s.state.retiredFiles, result.Removed...)
			s.state.retirementNotes = append(s.state.retirementNotes, result.Notes...)
		}
	}
	return nil
}

// legacySkillsBackupPaths lista los ficheros que la migración puede borrar, para que
// la copia de seguridad previa los guarde y una vuelta atrás los restaure.
func legacySkillsBackupPaths(workspaceDir string, scope InstallScope, adapters []agents.Adapter) []string {
	var paths []string
	for _, adapter := range legacySkillsAdapters(scope, adapters) {
		provider := adapter.(agents.LegacySkillsDirProvider)
		shared := adapter.SkillsDir(workspaceDir)
		for _, legacyDir := range provider.LegacyWorkspaceSkillsDirs(workspaceDir) {
			paths = append(paths, sdd.LegacyWorkspaceSkillFiles(legacyDir, shared, adapter.Agent())...)
		}
	}
	return paths
}
