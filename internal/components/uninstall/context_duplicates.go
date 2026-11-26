package uninstall

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Tipos de contexto que compara ContextDuplicates.
const (
	DuplicateInstructions = "instrucciones"
	DuplicateSkills       = "skills"
	DuplicateSubAgents    = "subagentes"
	DuplicateCommands     = "comandos"
	DuplicateOutputStyles = "output styles"
)

// ContextDuplicate es un contexto gestionado por Axiom que existe a la vez en la
// carpeta de usuario y en el proyecto: el agente carga las dos copias y gasta
// contexto de más. Agents agrupa los agentes que comparten el mismo par de rutas
// (varios leen .agents/skills del proyecto, por ejemplo).
type ContextDuplicate struct {
	Agents        []model.AgentID
	Kind          string
	UserPath      string
	WorkspacePath string
	// Items son los nombres duplicados (skills, subagentes, comandos y estilos); en
	// las instrucciones va vacío.
	Items []string
	// Diverges indica que el contenido gestionado de las dos copias no coincide
	// (normalmente la de usuario es una versión anterior).
	Diverges bool
}

// ContextDuplicates compara, para cada adaptador, lo que Axiom gestiona en la
// carpeta de usuario con lo que gestiona en el proyecto root: instrucciones con
// bloques gestionados, skills, subagentes, comandos y output styles que son
// activos de Axiom. Solo lee. Con root vacío no comprueba nada. Un fichero del
// usuario sin marcas de Axiom no se informa nunca.
func ContextDuplicates(homeDir, root string) []ContextDuplicate {
	if homeDir == "" || root == "" {
		return nil
	}
	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		return nil
	}

	var found []ContextDuplicate
	add := func(agent model.AgentID, duplicate ContextDuplicate) {
		for i := range found {
			existing := &found[i]
			if existing.Kind == duplicate.Kind && existing.UserPath == duplicate.UserPath && existing.WorkspacePath == duplicate.WorkspacePath {
				if !slices.Contains(existing.Agents, agent) {
					existing.Agents = append(existing.Agents, agent)
				}
				return
			}
		}
		duplicate.Agents = []model.AgentID{agent}
		found = append(found, duplicate)
	}

	for _, agent := range registry.SupportedAgents() {
		user, ok := registry.Get(agent)
		if !ok {
			continue
		}
		// Claude Code no se liga: con la raíz del proyecto sus rutas ya son
		// las del proyecto.
		project := agents.BindWorkspace(user, root)

		if user.SupportsSystemPrompt() {
			userFiles, projectFiles := systemPromptFiles(user, homeDir), systemPromptFiles(project, root)
			for i := 0; i < len(userFiles) && i < len(projectFiles); i++ {
				if userFiles[i] == "" || userFiles[i] == projectFiles[i] {
					continue
				}
				userSig, userOK := managedInstructionsSignature(userFiles[i])
				projectSig, projectOK := managedInstructionsSignature(projectFiles[i])
				if userOK && projectOK {
					add(agent, ContextDuplicate{
						Kind: DuplicateInstructions, UserPath: userFiles[i], WorkspacePath: projectFiles[i],
						Diverges: userSig != projectSig,
					})
				}
			}
		}

		if user.SupportsSkills() {
			if duplicate, ok := duplicateAssets(DuplicateSkills, user.SkillsDir(homeDir), project.SkillsDir(root), embeddedDirNames("skills"), "SKILL.md"); ok {
				add(agent, duplicate)
			}
		}
		if user.SupportsSubAgents() {
			names := embeddedFileNames(user.EmbeddedSubAgentsDir())
			if duplicate, ok := duplicateAssets(DuplicateSubAgents, user.SubAgentsDir(homeDir), project.SubAgentsDir(root), names, ""); ok {
				add(agent, duplicate)
			}
		}
		if user.SupportsSlashCommands() {
			names := embeddedFileNames(assets.SDDCommandsAssetDir(agent))
			if duplicate, ok := duplicateAssets(DuplicateCommands, user.CommandsDir(homeDir), project.CommandsDir(root), names, ""); ok {
				add(agent, duplicate)
			}
		}
		if user.SupportsOutputStyles() {
			var names []string
			for _, style := range persona.ManagedOutputStyles() {
				names = append(names, style.File)
			}
			if duplicate, ok := duplicateAssets(DuplicateOutputStyles, user.OutputStyleDir(homeDir), project.OutputStyleDir(root), names, ""); ok {
				add(agent, duplicate)
			}
		}
	}
	return found
}

// duplicateAssets busca, entre los nombres que Axiom incrusta, los que existen en
// las dos carpetas. Si compareFile no está vacío, cada nombre es una carpeta y se
// compara ese fichero; si lo está, cada nombre es un fichero.
func duplicateAssets(kind, userDir, projectDir string, names []string, compareFile string) (ContextDuplicate, bool) {
	if userDir == "" || projectDir == "" || filepath.Clean(userDir) == filepath.Clean(projectDir) {
		return ContextDuplicate{}, false
	}
	duplicate := ContextDuplicate{Kind: kind, UserPath: userDir, WorkspacePath: projectDir}
	for _, name := range names {
		userItem, projectItem := filepath.Join(userDir, name), filepath.Join(projectDir, name)
		if !existsOnDisk(userItem) || !existsOnDisk(projectItem) {
			continue
		}
		duplicate.Items = append(duplicate.Items, strings.TrimSuffix(name, ".md"))
		userCompare, projectCompare := userItem, projectItem
		if compareFile != "" {
			userCompare, projectCompare = filepath.Join(userItem, compareFile), filepath.Join(projectItem, compareFile)
		}
		if !sameFileContent(userCompare, projectCompare) {
			duplicate.Diverges = true
		}
	}
	if len(duplicate.Items) == 0 {
		return ContextDuplicate{}, false
	}
	sort.Strings(duplicate.Items)
	return duplicate, true
}

var managedSectionPattern = regexp.MustCompile(`(?s)<!-- (?:axiom|gentle-ai):(\S+) -->(.*?)<!-- /(?:axiom|gentle-ai):\S+ -->`)

// managedInstructionsSignature devuelve el contenido gestionado por Axiom de un
// fichero de instrucciones, normalizado para compararlo: sus bloques con marcas o,
// en un fichero de steering que Axiom escribe entero, el cuerpo. ok es false si el
// fichero no existe o no tiene nada de Axiom (un fichero del usuario).
func managedInstructionsSignature(path string) (string, bool) {
	raw, err := readManagedFile(path)
	if err != nil {
		return "", false
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if matches := managedSectionPattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		sections := make([]string, 0, len(matches))
		for _, match := range matches {
			sections = append(sections, match[1]+"\x00"+strings.TrimSpace(match[2]))
		}
		sort.Strings(sections)
		return strings.Join(sections, "\x01"), true
	}
	if !isManagedPromptFileContent(content) {
		return "", false
	}
	return strings.TrimSpace(stripManagedFrontmatter(content)), true
}

func embeddedDirNames(dir string) []string {
	entries, err := fs.ReadDir(assets.FS, dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func embeddedFileNames(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := fs.ReadDir(assets.FS, dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func existsOnDisk(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sameFileContent compara dos ficheros ignorando el final de línea; si alguno no
// se puede leer cuenta como distinto.
func sameFileContent(a, b string) bool {
	left, errA := os.ReadFile(a)
	right, errB := os.ReadFile(b)
	if errA != nil || errB != nil {
		return false
	}
	normalize := func(raw []byte) string { return strings.TrimSpace(strings.ReplaceAll(string(raw), "\r\n", "\n")) }
	return normalize(left) == normalize(right)
}
