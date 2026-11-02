package kilocode

import (
	"os"
	"path/filepath"
)

// Distribución de proyecto de Kilo (documentación oficial, kilo.ai/docs):
//
//   - .kilo/kilo.jsonc o .kilo/kilo.json (o kilo.jsonc, kilo.json en la raíz): claves
//     mcp, permission, agent, instructions. Si existen ambos, gana el de .kilo/.
//   - .kilo/{skills,commands,agents}: recursos del proyecto.
//   - .kilo/rules/: carpeta de reglas; se cargan las que enumera instructions.
//
// Kilo no lee <proyecto>/.config/kilo, que era donde escribía Axiom cuando
// trataba la raíz del proyecto como si fuera la carpeta de usuario.
const (
	projectConfigDirName        = ".kilo"
	projectInstructionsFileName = "axiom.md"
)

// WithWorkspace devuelve una copia del adaptador ligada a la raíz de un
// proyecto. Las rutas que se piden con esa raíz salen en la distribución de
// proyecto de Kilo; las que se piden con cualquier otra carpeta siguen siendo
// las globales de siempre.
func (a *Adapter) WithWorkspace(root string) *Adapter {
	bound := *a
	bound.workspaceRoot = root
	return &bound
}

func (a *Adapter) inWorkspace(dir string) bool {
	return a.workspaceRoot != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.workspaceRoot)
}

// configDir es la carpeta de recursos: .kilo en el proyecto, la carpeta de
// configuración global en cualquier otro caso.
func (a *Adapter) configDir(dir string) string {
	if a.inWorkspace(dir) {
		return filepath.Join(dir, projectConfigDirName)
	}
	return ConfigPath(dir)
}

// projectSettingsPath elige el fichero de configuración del proyecto: el que ya
// exista, en el orden de precedencia de Kilo (.kilo/ antes que la raíz, jsonc
// antes que json), para no crear un segundo fichero junto al del usuario. Si no
// hay ninguno, .kilo/kilo.json: Kilo lo lee igual que kilo.jsonc y, como JSON
// estricto, las fusiones lo reescriben de forma estable (en un .jsonc nuevo, la
// segunda fusión reindenta lo que escribió la primera y el segundo sync informaría
// de un cambio).
func projectSettingsPath(root string) string {
	candidates := []string{
		filepath.Join(root, projectConfigDirName, "kilo.jsonc"),
		filepath.Join(root, projectConfigDirName, "kilo.json"),
		filepath.Join(root, "kilo.jsonc"),
		filepath.Join(root, "kilo.json"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate
		}
	}
	return candidates[1]
}

// WorkspacePromptsDir devuelve la carpeta de los prompts compartidos de SDD
// cuando dir es la raíz del proyecto ligada. ok es false fuera del proyecto.
func (a *Adapter) WorkspacePromptsDir(dir string) (string, bool) {
	if !a.inWorkspace(dir) {
		return "", false
	}
	return filepath.Join(dir, projectConfigDirName, "prompts", "sdd"), true
}

// WorkspaceInstructions indica dónde se registra el fichero de instrucciones de
// Axiom: el fichero de configuración que lo declara y la entrada de su lista
// instructions, relativa a la raíz del proyecto (es la forma que documenta Kilo:
// ".kilo/rules/*.md"). ok es false fuera del proyecto.
func (a *Adapter) WorkspaceInstructions(dir string) (settingsPath, entry string, ok bool) {
	if !a.inWorkspace(dir) {
		return "", "", false
	}
	return a.SettingsPath(dir), filepath.ToSlash(filepath.Join(projectConfigDirName, "rules", projectInstructionsFileName)), true
}

// LegacyWorkspaceSkillsDirs lista la carpeta de skills que usaban las instalaciones
// de proyecto anteriores (.kilo/skills), ahora sustituida por .agents/skills. Kilo las
// lee todas y vería cada skill dos veces. Solo se retira lo que se prueba que es de
// Axiom. Fuera del proyecto no devuelve nada.
func (a *Adapter) LegacyWorkspaceSkillsDirs(dir string) []string {
	if !a.inWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, projectConfigDirName, "skills")}
}
