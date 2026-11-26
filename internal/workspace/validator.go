package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// FS es una interfaz para abstraer las operaciones del sistema de archivos y facilitar pruebas unitarias.
type FS interface {
	Stat(name string) (os.FileInfo, error)
	IsDir(path string) (bool, error)
}

// OSFileSystem es la implementación por defecto que interactúa con el sistema de archivos real del sistema operativo.
type OSFileSystem struct{}

func (OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (OSFileSystem) IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// DefaultFS devuelve la implementación estándar del sistema de archivos.
func DefaultFS() FS {
	return OSFileSystem{}
}

// monorepoRole es el único rol que admite la topología monorepo-embedded.
const monorepoRole = "fullstack"

// Validate comprueba deterministamente que un WorkspaceConfig cumple las reglas de topología de Axiom en baseDir.
func Validate(fs FS, cfg *WorkspaceConfig, baseDir string) (*ValidationReport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("la configuración del workspace no puede ser nula")
	}

	cleanBaseDir := filepath.Clean(baseDir)
	report := &ValidationReport{
		Valid:         true,
		Topology:      cfg.Workspace.Topology,
		WorkspaceRoot: cleanBaseDir,
		CheckedPaths:  make([]string, 0),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// 1. Verificar la carpeta maestra
	isDir, err := fs.IsDir(cleanBaseDir)
	if err != nil || !isDir {
		report.Valid = false
		report.Errors = append(report.Errors, fmt.Sprintf("la carpeta maestra del workspace '%s' no existe o no es un directorio accesible", cleanBaseDir))
		return report, nil
	}
	report.CheckedPaths = append(report.CheckedPaths, cleanBaseDir)

	// 2. Validación de repositorio canónico de especificaciones según la topología
	specsRepo := strings.TrimSpace(cfg.Workspace.SpecsRepository)

	switch cfg.Workspace.Topology {
	case TopologyMonorepoEmbedded:
		// En monorepo embebido, specs puede ser "." o una subcarpeta como "openspec" o "specs"
		if specsRepo != "" && specsRepo != "." {
			targetSpecs := filepath.Join(cleanBaseDir, specsRepo)
			report.CheckedPaths = append(report.CheckedPaths, targetSpecs)
			if ok, err := fs.IsDir(targetSpecs); err != nil || !ok {
				report.Valid = false
				report.Errors = append(report.Errors, fmt.Sprintf("el repositorio de especificaciones '%s' declarado para monorepo embebido no existe", specsRepo))
			}
		}

	case TopologyMultirepo:
		// En multirepo, es REQUISITO OBLIGATORIO tener un repositorio canónico de specs
		if specsRepo == "" || specsRepo == "." {
			report.Valid = false
			report.Errors = append(report.Errors, "los proyectos con topología 'multirepo' requieren obligatoriamente un repositorio canónico de especificaciones dentro de la carpeta maestra común ('specs_repository' no puede ser '.' ni estar vacío)")
		} else {
			targetSpecs := filepath.Join(cleanBaseDir, specsRepo)
			report.CheckedPaths = append(report.CheckedPaths, targetSpecs)
			if ok, err := fs.IsDir(targetSpecs); err != nil || !ok {
				report.Valid = false
				report.Errors = append(report.Errors, fmt.Sprintf("el repositorio canónico de especificaciones '%s' no existe en la carpeta maestra", specsRepo))
			}
		}

	default:
		// La carga ya rechaza estos valores; una configuración construida a mano
		// con la topología retirada o desconocida tampoco se da por válida (RF-TOP-08).
		report.Valid = false
		report.Errors = append(report.Errors, topologyError(cfg.Workspace.Topology).Error())
	}

	// 3. Validación de repositorios asignados a cada rol
	for roleKey, role := range cfg.Roles {
		for _, repo := range role.Repositories {
			cleanRepoPath := filepath.Clean(strings.TrimSpace(repo.Path))
			var fullRepoPath string
			if filepath.IsAbs(cleanRepoPath) {
				fullRepoPath = cleanRepoPath
			} else {
				fullRepoPath = filepath.Join(cleanBaseDir, cleanRepoPath)
			}

			report.CheckedPaths = append(report.CheckedPaths, fullRepoPath)
			if ok, err := fs.IsDir(fullRepoPath); err != nil || !ok {
				report.Valid = false
				report.Errors = append(report.Errors, fmt.Sprintf("el repositorio '%s' asignado al rol '%s' no existe en la carpeta maestra (ruta: '%s')", repo.Path, roleKey, fullRepoPath))
			}
		}
	}

	// 3b. En monorepo-embedded el único rol admitido es fullstack: los roles
	// especializados pertenecen a multirepo (RF-TOP-04). Es solo una regla de
	// validación; la carga no la aplica, de modo que un axiom.yaml existente sigue
	// resolviéndose y los kickoff.yaml ya sellados se siguen leyendo igual.
	if cfg.Workspace.Topology == TopologyMonorepoEmbedded {
		var specialized []string
		for roleKey := range cfg.Roles {
			if roleKey != monorepoRole {
				specialized = append(specialized, roleKey)
			}
		}
		if len(specialized) > 0 {
			sort.Strings(specialized)
			report.Valid = false
			report.Errors = append(report.Errors, fmt.Sprintf("la topología 'monorepo-embedded' solo admite el rol 'fullstack', pero axiom.yaml declara: %s. Para migrar, deja en 'roles' solo 'fullstack' (con los repositorios y tecnologías de los demás) o pasa a 'workspace.topology: multirepo' con un 'specs_repository' propio; los kickoff.yaml ya sellados se siguen leyendo igual", strings.Join(specialized, ", ")))
		}
	}

	// 3b bis. `acceptance_tests.enabled: true` solo se admite en multirepo (RF-TOP-19).
	// Como la regla de roles, es solo validación: la carga no la aplica, para que
	// Locate y el archivado toleren el error (RF-TOP-20).
	if err := acceptanceTestsTopologyError(cfg.AcceptanceTests, cfg.Workspace.Topology); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, err.Error())
	}

	// 4. Validación de las raíces de skills versionadas. El escaneo del índice ignora
	// en silencio las inválidas, así que este es el único punto donde se avisa.
	for _, root := range cfg.Workspace.SkillRoots {
		if _, ok := CleanSkillRoot(root); !ok {
			report.Valid = false
			report.Errors = append(report.Errors, fmt.Sprintf("la raíz de skills '%s' declarada en 'workspace.skill_roots' no es válida: debe ser una ruta relativa dentro del proyecto y distinta de '.'", root))
		}
	}

	report.Warnings = append(report.Warnings, ObsoleteKeyWarnings(cfg)...)

	if len(report.Errors) > 0 {
		report.Valid = false
	}

	return report, nil
}

// ValidateDeclarationPaths comprueba las declaraciones válidas de cfg
// (`technical_context:`) contra el sistema de ficheros real y el
// almacén declarado, y devuelve los errores y las advertencias para que quien
// llama los fusione en el informe de Validate (REQ-24.14, 24.17, 24.27 y 24.62).
// master es la carpeta maestra (la ruta que recibe `workspace validate`).
//
// No forma parte de Validate: este conserva su firma y su FS abstracto (m9), y
// las comprobaciones de identidad y de enlaces simbólicos necesitan el sistema de
// ficheros real (filepath.EvalSymlinks y pathidentity). Las reglas sintácticas de
// las declaraciones ya se validaron al cargar la configuración, así que una
// declaración inválida no llega aquí: cfg no la lleva.
//
// Las reglas son las de D-04:
//   - cada ruta de contexto que existe, resuelta, sigue dentro de la carpeta
//     maestra (error); una ruta inexistente solo avisa;
//   - los solapes se comprueban de nuevo por identidad sobre lo que existe, para
//     detectar grafías distintas del mismo lugar;
//   - con almacén declarado `engram` o `none`, un error por cada declaración que
//     exige ficheros.
func ValidateDeclarationPaths(cfg *WorkspaceConfig, master string) (errs []string, warnings []string) {
	if cfg == nil {
		return nil, nil
	}
	master = filepath.Clean(master)
	workspaceRoot, _ := Roots(cfg, master)

	// Si workspaceRoot no existe, Validate ya lo reporta: comprobar cada ruta
	// contra él solo añadiría un error por declaración con la misma causa.
	if info, err := os.Stat(workspaceRoot); err == nil && info.IsDir() && cfg.TechnicalContext != nil {
		var located []declaredLocation
		if cfg.TechnicalContext != nil {
			contextErrs, contextWarnings, contexts := checkContextPaths(cfg.TechnicalContext, workspaceRoot, master)
			errs = append(errs, contextErrs...)
			warnings = append(warnings, contextWarnings...)
			located = append(located, contexts...)
		}
		errs = append(errs, identityOverlaps(located)...)
	}

	errs = append(errs, storeErrors(cfg, workspaceRoot)...)
	return errs, warnings
}

// declaredLocation es una ruta declarada que existe, con el lugar al que
// resuelve. owner es el modo del contexto técnico.
type declaredLocation struct {
	owner    string
	declared string
	resolved string
}

// checkContextPaths comprueba las rutas de los dos modos del contexto técnico: la
// que no existe avisa y la que existe y resuelve fuera de la carpeta maestra es
// un error. Solo las rutas de `paths` participan en los solapes: `entry` está
// contenida en ellas por regla de la carga.
func checkContextPaths(context *TechnicalContextConfig, workspaceRoot, master string) (errs, warnings []string, located []declaredLocation) {
	modes := []struct {
		label string
		mode  *ContextMode
	}{
		{"technical_context.maintained", context.Maintained},
		{"technical_context.reference", context.Reference},
	}
	for _, entry := range modes {
		if entry.mode == nil {
			continue
		}
		check := func(raw, field string) (string, bool) {
			clean, err := checkRelative(raw)
			if err != nil {
				return "", false // ya rechazado al cargar la configuración
			}
			full := filepath.Join(workspaceRoot, filepath.FromSlash(clean))
			if _, err := os.Stat(full); err != nil {
				if os.IsNotExist(err) {
					warnings = append(warnings, fmt.Sprintf("la ruta '%s' de '%s.%s' no existe todavía (ruta: '%s'); un documento mantenido puede crearse al archivar", raw, entry.label, field, full))
				} else {
					errs = append(errs, fmt.Sprintf("no se pudo acceder a la ruta '%s' de '%s.%s': %v", raw, entry.label, field, err))
				}
				return "", false
			}
			resolved, err := filepath.EvalSymlinks(full)
			if err != nil {
				errs = append(errs, fmt.Sprintf("no se pudo resolver la ruta '%s' de '%s.%s': %v", raw, entry.label, field, err))
				return "", false
			}
			if !pathidentity.Contains(master, resolved) {
				errs = append(errs, fmt.Sprintf("la ruta '%s' de '%s.%s' resuelve a '%s', fuera de la carpeta maestra '%s' (enlace simbólico): una ruta de contexto técnico debe permanecer dentro de ella", raw, entry.label, field, resolved, master))
				return "", false
			}
			return resolved, true
		}
		for _, raw := range entry.mode.Paths {
			if resolved, ok := check(raw, "paths"); ok {
				located = append(located, declaredLocation{owner: entry.label, declared: raw, resolved: resolved})
			}
		}
		if strings.TrimSpace(entry.mode.Entry) != "" {
			check(entry.mode.Entry, "entry")
		}
	}
	return errs, warnings, located
}

// identityOverlaps repite por identidad la comprobación de solapes de la carga:
// una grafía distinta del mismo directorio, o un enlace, no esquiva la regla de
// que ninguna ruta pertenece a los dos modos. Las rutas de un
// mismo propietario no se comparan entre sí.
func identityOverlaps(located []declaredLocation) []string {
	var errs []string
	for i, a := range located {
		for _, b := range located[i+1:] {
			if a.owner == b.owner {
				continue
			}
			if sameLocation(a.resolved, b.resolved) {
				errs = append(errs, fmt.Sprintf("la ruta '%s' (%s) y la ruta '%s' (%s) solapan por identidad: apuntan al mismo lugar o una contiene a la otra, y una ruta no puede pertenecer a los dos modos del contexto técnico", a.declared, a.owner, b.declared, b.owner))
			}
		}
	}
	return errs
}

// sameLocation dice si a y b son el mismo fichero o directorio, o si uno contiene
// al otro.
func sameLocation(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA == nil && errB == nil && os.SameFile(infoA, infoB) {
		return true
	}
	return pathidentity.Contains(a, b) || pathidentity.Contains(b, a)
}

// storeErrors reporta que openspec/config.yaml declara `engram` o `none`, con
// independencia del contexto técnico (RF-TOP-13), y además un error por cada
// declaración que exige un almacén de ficheros (REQ-24.62, RF-CTX-17). Sin
// `artifact_store` declarado, o con `openspec` o `hybrid`, no reporta nada: el
// status cubre el valor por defecto y esta regla no cambia su resolución.
func storeErrors(cfg *WorkspaceConfig, workspaceRoot string) []string {
	store, declared := DeclaredArtifactStore(workspaceRoot)
	if !declared || (store != "engram" && store != "none") {
		return nil
	}
	maintained := cfg.TechnicalContext != nil && cfg.TechnicalContext.Maintained != nil
	errs := []string{legacyStoreError(store, maintained)}
	if maintained {
		errs = append(errs, storeError("technical_context.maintained", "el contexto técnico mantenido", store))
	}
	return errs
}

// legacyStoreError es el error incondicional de RF-TOP-13: en Axiom el almacén de
// una sesión SDD es siempre hybrid. Con contexto mantenido nombra también la
// declaración afectada (RF-CTX-17).
func legacyStoreError(store string, maintained bool) string {
	msg := fmt.Sprintf("openspec/config.yaml declara artifact_store '%s', que Axiom ya no admite: el almacén de los cambios es siempre 'hybrid', y con '%s' no hay paradas de tres salidas, publicación de compuertas ni relevo (--hand-off). Para migrar, quita la clave artifact_store o declara 'artifact_store: hybrid'", store, store)
	if maintained {
		msg += "; la declaración 'technical_context.maintained' también lo exige"
	}
	return msg
}

func storeError(declaration, subject, store string) string {
	return fmt.Sprintf("la declaración '%s' exige un almacén de artefactos de ficheros, pero openspec/config.yaml declara artifact_store '%s' y %s necesita leer y escribir la carpeta del cambio. Declara 'artifact_store: openspec' o 'hybrid' en openspec/config.yaml, o retira la declaración '%s' de axiom.yaml", declaration, store, subject, declaration)
}

// ObsoleteKeyWarnings devuelve los avisos de las claves de axiom.yaml que se
// siguen leyendo pero ya no tienen efecto. Nunca invalidan la configuración.
func ObsoleteKeyWarnings(cfg *WorkspaceConfig) []string {
	if cfg == nil {
		return nil
	}
	var warnings []string
	if strings.TrimSpace(cfg.Governance.SemanticAnalysis) != "" {
		warnings = append(warnings, "la clave 'governance.semantic_analysis' de axiom.yaml ya no tiene efecto y se ignora: Axiom retiró su análisis semántico propio y los agentes usan CodeGraph directamente; puedes borrarla")
	}
	return warnings
}
