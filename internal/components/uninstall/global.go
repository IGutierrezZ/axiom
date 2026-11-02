package uninstall

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/engram"
	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// Seams de la parte de máquina de la desinstalación global. Las pruebas los
// sustituyen para no tocar el PATH ni la plataforma reales.
var (
	userPathEntries    = system.UserPathEntries
	removeFromUserPath = system.RemoveFromUserPath
	pathGOOS           = runtime.GOOS
)

// hubRegistryFile es el registro de proyectos del hub dentro de la raíz de estado.
const hubRegistryFile = "workspaces.json"

// managedPathDirs son las carpetas que Axiom añade al PATH de usuario y que una
// desinstalación global completa retira. No incluye ninguna carpeta compartida:
//   - el directorio del binario de Engram, solo donde Axiom lo registra en el PATH
//     (Windows; véase engram.ManagedBinaryDir);
//   - `~/bin`, donde GGA deja sus shims, solo si GGA estaba gestionado por Axiom
//     (ggaManaged, evaluado antes de retirar su configuración);
//   - `<raíz de estado>/bin`, donde viven los lanzadores gestionados de OpenCode.
func (s *Service) managedPathDirs(ggaManaged bool) []string {
	var dirs []string
	if dir := engram.ManagedBinaryDir(pathGOOS); dir != "" {
		dirs = append(dirs, dir)
	}
	if ggaManaged {
		dirs = append(dirs, gga.FootprintPathEntries(s.homeDir, pathGOOS)...)
	}
	dirs = append(dirs, system.BinDir(s.homeDir))
	return uniquePaths(dirs...)
}

// ggaManagedByAxiom dice si hay en HOME algo de GGA que Axiom escribió: su
// configuración o sus shims.
func (s *Service) ggaManagedByAxiom() bool {
	if gga.Managed(s.homeDir) {
		return true
	}
	for _, path := range []string{gga.RuntimeCMDPath(s.homeDir), gga.RuntimePS1Path(s.homeDir)} {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// pathEntryPresent dice si dir figura entre las entradas del PATH de usuario. La
// comparación recorta espacios y comillas e ignora mayúsculas, como el resto de
// operaciones de PATH.
func pathEntryPresent(entries []string, dir string) bool {
	want := filepath.Clean(strings.Trim(strings.TrimSpace(dir), `"`))
	for _, entry := range entries {
		if strings.EqualFold(filepath.Clean(strings.Trim(strings.TrimSpace(entry), `"`)), want) {
			return true
		}
	}
	return false
}

// removeManagedPathEntries retira del PATH de usuario las carpetas gestionadas que
// siguen en él y deja el resto de entradas como estaban. Un fallo no interrumpe la
// desinstalación: se anota como acción manual con la carpeta que hay que quitar.
func (s *Service) removeManagedPathEntries(ggaManaged bool, result *Result) {
	entries, err := userPathEntries(pathGOOS)
	if err != nil {
		result.ManualActions = append(result.ManualActions,
			fmt.Sprintf("No se pudo leer el PATH de usuario (%v): quita a mano las carpetas que Axiom añadió, si las hay: %s", err, strings.Join(s.managedPathDirs(ggaManaged), ", ")))
		return
	}
	for _, dir := range s.managedPathDirs(ggaManaged) {
		if !pathEntryPresent(entries, dir) {
			continue
		}
		if err := removeFromUserPath(dir); err != nil {
			result.ManualActions = append(result.ManualActions,
				fmt.Sprintf("No se pudo retirar %s del PATH de usuario (%v): quítalo a mano.", dir, err))
			continue
		}
		result.RemovedPathEntries = append(result.RemovedPathEntries, dir)
	}
}

// hubRegisteredProjects cuenta los proyectos del registro del hub de dir. Un
// registro ausente o ilegible cuenta como ninguno.
func hubRegisteredProjects(dir string) int {
	data, err := os.ReadFile(filepath.Join(dir, hubRegistryFile))
	if err != nil {
		return 0
	}
	var registry struct {
		Workspaces []json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return 0
	}
	return len(registry.Workspaces)
}

// stateDirIsSafeToPurge rechaza raíces de estado que no pueden borrarse sin riesgo:
// vacía, la raíz del sistema de ficheros, HOME o una carpeta que contiene HOME o el
// proyecto (AXIOM_STATE_DIR mal configurada).
func (s *Service) stateDirIsSafeToPurge(dir string) error {
	clean := filepath.Clean(dir)
	if strings.TrimSpace(dir) == "" || clean == "." || clean == filepath.VolumeName(clean)+string(filepath.Separator) {
		return fmt.Errorf("la raíz de estado %q no es una carpeta que se pueda borrar", dir)
	}
	for _, protected := range []string{s.homeDir, s.workspaceDir} {
		if strings.TrimSpace(protected) == "" {
			continue
		}
		rel, err := filepath.Rel(clean, filepath.Clean(protected))
		if err == nil && !strings.HasPrefix(rel, "..") {
			return fmt.Errorf("la raíz de estado %q contiene %q: no se purga", dir, protected)
		}
	}
	return nil
}

// purgeStateDir borra la raíz de estado de Axiom (system.AxiomDir) a petición
// expresa (--purge-state). Si el registro del hub lista proyectos y no se pidió
// también --purge-hub, ese registro se conserva y se borra lo demás: los proyectos
// registrados siguen resolviéndose. La copia de seguridad de esta desinstalación vive
// en esa raíz y se borra con ella.
func (s *Service) purgeStateDir(result *Result) {
	dir := system.AxiomDir(s.homeDir)
	if err := s.stateDirIsSafeToPurge(dir); err != nil {
		result.ManualActions = append(result.ManualActions, "No se purga el estado de Axiom: "+err.Error())
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			result.ManualActions = append(result.ManualActions, fmt.Sprintf("No se pudo leer %s para purgarlo: %v", dir, err))
		}
		return
	}

	keepHub := hubRegisteredProjects(dir) > 0 && !s.options.PurgeHub
	var failed []string
	for _, entry := range entries {
		name := entry.Name()
		if keepHub && strings.HasPrefix(name, hubRegistryFile) {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.RemoveAll(path); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", path, err))
			continue
		}
		result.PurgedPaths = append(result.PurgedPaths, path)
	}
	if len(failed) > 0 {
		result.ManualActions = append(result.ManualActions, "No se pudo purgar: "+strings.Join(failed, ", ")+". Bórralo a mano.")
	}
	if keepHub {
		result.ManualActions = append(result.ManualActions,
			fmt.Sprintf("Conservado el registro del hub (%s): lista %d proyecto(s) registrado(s). Usa --purge-hub junto a --purge-state para borrarlo también.",
				filepath.Join(dir, hubRegistryFile), hubRegisteredProjects(dir)))
		return
	}
	if err := os.Remove(dir); err == nil {
		result.PurgedPaths = append(result.PurgedPaths, dir)
	}
	result.ManualActions = append(result.ManualActions,
		"Se purgó el estado de Axiom, copias de seguridad incluidas: la de esta desinstalación ya no existe.")
}

// finishGlobalUninstall completa una desinstalación global completa con lo que no
// es configuración de un agente: las entradas de PATH gestionadas, la carpeta de
// lanzadores si queda vacía y, si se pidió, la purga del estado.
func (s *Service) finishGlobalUninstall(ggaManaged bool, result *Result) {
	s.removeManagedPathEntries(ggaManaged, result)

	binDir := system.BinDir(s.homeDir)
	if removed, err := removeDirIfEmptyRecursive(binDir); err == nil && removed {
		result.RemovedDirectories = append(result.RemovedDirectories, binDir)
	}

	if s.options.PurgeState {
		s.purgeStateDir(result)
	}
	if legacy := filepath.Join(s.homeDir, ".gentle-ai"); isExistingDir(legacy) {
		result.ManualActions = append(result.ManualActions,
			fmt.Sprintf("Queda %s, el estado de versiones anteriores de Axiom. Axiom no lo borra porque la herramienta original comparte ese nombre: bórralo a mano si no la usas.", legacy))
	}
	result.ManualActions = dedupeSortedStrings(result.ManualActions)
}

// compatibilitySkillsOperations retira de la copia de skills de compatibilidad
// (`~/.agents/skills`) las skills que Axiom instala y las de la carpeta compartida
// _shared. Install y sync refrescan esa copia, que es de todos los agentes y no de
// uno, así que solo la retira una desinstalación global completa. Las skills del
// usuario no se tocan y la carpeta solo se quita si queda vacía.
func (s *Service) compatibilitySkillsOperations() (targets []string, ops []operation, err error) {
	skillDir := filepath.Join(s.homeDir, ".agents", "skills")
	retiredTargets, retiredOps := retireInstalledSkillOps(skillDir)
	targets = append(targets, retiredTargets...)
	ops = append(ops, retiredOps...)

	embedded, err := fs.ReadDir(assets.FS, "skills")
	if err != nil {
		return nil, nil, fmt.Errorf("read embedded skills: %w", err)
	}
	for _, entry := range embedded {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(skillDir, entry.Name())
		targets = append(targets, dirPath)
		ops = append(ops, removeTree(dirPath))
	}
	skillsEmpty, agentsEmpty := removeDirIfEmpty(skillDir), removeDirIfEmpty(filepath.Dir(skillDir))
	skillsEmpty.quiet, agentsEmpty.quiet = true, true
	return targets, append(ops, skillsEmpty, agentsEmpty), nil
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
