package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// planningHomeDirName es el nombre literal del directorio de cambios y
// especificaciones que cuelga siempre de workspaceRoot (invariante B').
const planningHomeDirName = "openspec"

// Origin indica cómo se halló el workspace.
type Origin string

const (
	// OriginPointer: el marcador más cercano es un .axiom-workspace.
	OriginPointer Origin = "pointer"
	// OriginConfig: el marcador más cercano es un axiom.yaml sin puntero.
	OriginConfig Origin = "config"
	// OriginOpenSpec: el marcador más cercano es un directorio openspec.
	OriginOpenSpec Origin = "openspec"
	// OriginStart: no hay marcadores; la raíz es el directorio de partida.
	OriginStart Origin = "start"
)

// DiscoveryCeilingOverride lo fijan solo las pruebas: directorios que el
// descubrimiento ascendente no evalúa, ni a ellos ni a sus ancestros (misma
// semántica que GIT_CEILING_DIRECTORIES). Es nil en producción y no tiene
// variable de entorno equivalente; sigue el patrón de LookPathOverride.
var DiscoveryCeilingOverride []string

// Location es el único resultado de la resolución del planning home.
type Location struct {
	Start         string           // directorio de partida absoluto y limpio
	MarkerDir     string           // directorio del marcador; "" con OriginStart
	Master        string           // carpeta maestra
	WorkspaceRoot string           // padre del planning home
	PlanningHome  string           // siempre filepath.Join(WorkspaceRoot, "openspec")
	Origin        Origin           //
	ConfigPath    string           // axiom.yaml efectivo; "" si no hay
	Config        *WorkspaceConfig // configuración base; nil si no hay o no es cargable
	ConfigErr     error            // causa de no carga de un axiom.yaml directo
	Declarations  Declarations     // baseline y technical_context de axiom.yaml, con su estado de validación
	Notes         []string         // avisos no bloqueantes
}

// LocateErrorKind clasifica los rechazos de la resolución.
type LocateErrorKind string

const (
	LocateStartInvalid           LocateErrorKind = "start_invalid"
	LocatePointerBroken          LocateErrorKind = "pointer_broken"
	LocateConfigUnloadable       LocateErrorKind = "config_unloadable"
	LocateSpecsRepositoryMissing LocateErrorKind = "specs_repository_missing"
	LocateAmbiguousPlanningHome  LocateErrorKind = "ambiguous_planning_home"
)

// LocateError es el rechazo tipado de la resolución: nombra siempre la causa y
// una continuación accionable, con las rutas citadas por pathquote.Quote.
type LocateError struct {
	Kind         LocateErrorKind
	Path         string // ruta principal implicada
	Cause        string // causa en castellano
	Continuation string // acción o comando accionable
	Err          error
}

// Error compone la causa y la continuación en un único mensaje.
func (e *LocateError) Error() string {
	switch {
	case e.Cause != "" && e.Continuation != "":
		return e.Cause + ". " + e.Continuation
	case e.Cause != "":
		return e.Cause
	default:
		return e.Continuation
	}
}

// Unwrap expone la causa original, si la hay.
func (e *LocateError) Unwrap() error { return e.Err }

// Locate resuelve, desde start, la carpeta maestra, workspaceRoot, el planning
// home y el origen. Es determinista y de solo lectura: nunca crea ni modifica
// nada, tampoco openspec/ (REQ-24.1). Evalúa start y después sus ancestros y se
// detiene en el primer marcador (REQ-24.2).
func Locate(start string) (Location, error) {
	return locate(start, true)
}

// LocateAt aplica las mismas reglas que Locate pero evalúa solo dir, sin
// ascender. Sin marcador en dir, Master y WorkspaceRoot son dir. Lo usan las
// llamadas internas que ya conocen la raíz y no deben cambiar de comportamiento.
func LocateAt(dir string) (Location, error) {
	return locate(dir, false)
}

func locate(start string, ascend bool) (Location, error) {
	abs, err := normalizeStart(start)
	if err != nil {
		return Location{}, err
	}
	dir := abs
	for {
		if ascend && coversCeiling(dir) {
			break
		}
		if origin, found := markerAt(dir); found {
			return locateFromMarker(abs, dir, origin)
		}
		parent := filepath.Dir(dir)
		if !ascend || parent == dir {
			break
		}
		dir = parent
	}
	return Location{
		Start:         abs,
		Master:        abs,
		WorkspaceRoot: abs,
		PlanningHome:  filepath.Join(abs, planningHomeDirName),
		Origin:        OriginStart,
	}, nil
}

// normalizeStart devuelve el directorio de partida absoluto y limpio, o el
// rechazo start_invalid si no existe, no es un directorio o es la raíz del
// sistema de ficheros.
func normalizeStart(start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", startInvalid(start, "no se pudo determinar el directorio de trabajo actual", err)
		}
		start = wd
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", startInvalid(start, "no se pudo convertir a una ruta absoluta", err)
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", startInvalid(abs, "no existe", err)
	case err != nil:
		return "", startInvalid(abs, "no es accesible", err)
	case !info.IsDir():
		return "", startInvalid(abs, "no es un directorio", nil)
	case filepath.Dir(abs) == abs:
		return "", startInvalid(abs, "es la raíz del sistema de ficheros, que nunca contiene un proyecto", nil)
	}
	return abs, nil
}

func startInvalid(start, reason string, err error) *LocateError {
	return &LocateError{
		Kind:         LocateStartInvalid,
		Path:         start,
		Cause:        fmt.Sprintf("el directorio de partida %s es incorrecto: %s", pathquote.Quote(start), reason),
		Continuation: "Vuelve a lanzar el comando con el directorio del proyecto: `axiom sdd status --cwd \"<directorio-del-proyecto>\" --json` (o `--path` en handoff y role)",
		Err:          err,
	}
}

// coversCeiling informa de si dir es, o es ancestro de, algún directorio del
// techo de pruebas. Compara por identidad, no por cadena.
func coversCeiling(dir string) bool {
	for _, ceiling := range DiscoveryCeilingOverride {
		if pathidentity.Contains(dir, ceiling) {
			return true
		}
	}
	return false
}

// markerAt devuelve el marcador de dir por precedencia: puntero, configuración
// y openspec. El fallback de subdirectorios de ResolveConfigFile nunca actúa
// como marcador (REQ-24.2).
func markerAt(dir string) (Origin, bool) {
	switch {
	case fileExists(filepath.Join(dir, WorkspacePointerFilename)):
		return OriginPointer, true
	case fileExists(filepath.Join(dir, DefaultConfigFilename)):
		return OriginConfig, true
	case isDir(filepath.Join(dir, planningHomeDirName)):
		return OriginOpenSpec, true
	}
	return "", false
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func locateFromMarker(start, markerDir string, origin Origin) (Location, error) {
	switch origin {
	case OriginPointer:
		loc, _, err := ResolveOwnPointer(markerDir)
		if err != nil {
			return Location{}, err
		}
		loc.Start = start
		return loc, nil
	case OriginConfig:
		return locateFromConfig(start, markerDir)
	default:
		return Location{
			Start:         start,
			MarkerDir:     markerDir,
			Master:        markerDir,
			WorkspaceRoot: markerDir,
			PlanningHome:  filepath.Join(markerDir, planningHomeDirName),
			Origin:        OriginOpenSpec,
		}, nil
	}
}

// ResolveOwnPointer aplica a <dir>/.axiom-workspace, sin ascender, las reglas
// de Locate para un puntero. Nunca exige que exista el planning home. found es
// false si dir no contiene puntero. Los errores son *LocateError.
func ResolveOwnPointer(dir string) (loc Location, found bool, err error) {
	absDir, absErr := filepath.Abs(dir)
	if absErr != nil {
		return Location{}, false, startInvalid(dir, "no se pudo convertir a una ruta absoluta", absErr)
	}
	absDir = filepath.Clean(absDir)
	pointerPath := filepath.Join(absDir, WorkspacePointerFilename)
	if !fileExists(pointerPath) {
		return Location{}, false, nil
	}

	configPath, err := resolveFromPointerFile(pointerPath)
	if err != nil {
		return Location{}, true, &LocateError{
			Kind:         LocatePointerBroken,
			Path:         pointerPath,
			Cause:        err.Error(),
			Continuation: fmt.Sprintf("Corrige el puntero %s para que `config` o `specs` apunte a un axiom.yaml existente, o retíralo", pathquote.Quote(pointerPath)),
			Err:          err,
		}
	}
	data, err := os.ReadFile(configPath)
	if err == nil {
		// Los errores de las declaraciones nunca rechazan (D-03): sin la
		// configuración base no se sabe dónde está el planning home, pero un
		// bloque baseline o technical_context inválido no lo oculta.
		var cfg *WorkspaceConfig
		var declarations Declarations
		cfg, err, declarations = parseLayered(data)
		if err == nil {
			loc = Location{
				Start:        absDir,
				MarkerDir:    absDir,
				Master:       absDir,
				Origin:       OriginPointer,
				ConfigPath:   configPath,
				Config:       cfg,
				Declarations: declarations,
			}
			if rootsErr := completeRoots(&loc); rootsErr != nil {
				return Location{}, true, rootsErr
			}
			return loc, true, nil
		}
	}
	return Location{}, true, &LocateError{
		Kind:         LocateConfigUnloadable,
		Path:         configPath,
		Cause:        fmt.Sprintf("el puntero %s apunta a %s, que no se puede cargar: %v", pathquote.Quote(pointerPath), pathquote.Quote(configPath), err),
		Continuation: fmt.Sprintf("Ejecuta `axiom workspace validate --path %s` y corrige la configuración; sin ella el planning home es desconocido", pathquote.Quote(absDir)),
		Err:          err,
	}
}

// locateFromConfig resuelve un axiom.yaml directo (sin puntero) hallado en dir.
func locateFromConfig(start, dir string) (Location, error) {
	configPath := filepath.Join(dir, DefaultConfigFilename)
	data, err := os.ReadFile(configPath)
	var cfg *WorkspaceConfig
	var declarations Declarations
	if err == nil {
		cfg, err, declarations = parseLayered(data)
	} else {
		// Un axiom.yaml que existe y no se puede leer deja la causa en ParseErr.
		declarations.ParseErr = err
	}
	if err != nil {
		// Caso 3 de REQ-24.7: un axiom.yaml directo que no se puede cargar no
		// se rechaza; se conserva la resolución embebida y se avisa. Las
		// declaraciones se evalúan igualmente, de modo que el status pueda
		// bloquear el archivado por su causa.
		return Location{
			Start:         start,
			MarkerDir:     dir,
			Master:        dir,
			WorkspaceRoot: dir,
			PlanningHome:  filepath.Join(dir, planningHomeDirName),
			Origin:        OriginConfig,
			ConfigPath:    configPath,
			ConfigErr:     err,
			Declarations:  declarations,
			Notes: []string{fmt.Sprintf(
				"El axiom.yaml %s no se pudo cargar (%v); se conserva la resolución embebida en %s. Ejecuta `axiom workspace validate --path %s` para ver la causa.",
				pathquote.Quote(configPath), err, pathquote.Quote(dir), pathquote.Quote(dir)),
			},
		}, nil
	}
	loc := Location{
		Start:        start,
		MarkerDir:    dir,
		Master:       deduceMaster(cfg, dir),
		Origin:       OriginConfig,
		ConfigPath:   configPath,
		Config:       cfg,
		Declarations: declarations,
	}
	if err := completeRoots(&loc); err != nil {
		return Location{}, err
	}
	return loc, nil
}

// completeRoots rellena WorkspaceRoot y PlanningHome a partir de Master y
// Config, y aplica las comprobaciones de los pasos 8 a 10 de D-02.
func completeRoots(loc *Location) error {
	loc.WorkspaceRoot, loc.PlanningHome = Roots(loc.Config, loc.Master)

	// Paso 8: se exige workspaceRoot, nunca el planning home, que no tiene que
	// existir para resolver.
	if separatesSpecs(loc.Config) && !isDir(loc.WorkspaceRoot) {
		return &LocateError{
			Kind:         LocateSpecsRepositoryMissing,
			Path:         loc.WorkspaceRoot,
			Cause:        fmt.Sprintf("el repositorio de especificaciones %s no existe (specs_repository: %q)", pathquote.Quote(loc.WorkspaceRoot), loc.Config.Workspace.SpecsRepository),
			Continuation: fmt.Sprintf("Crea el repositorio de especificaciones en %s o corrige `workspace.specs_repository` en %s", pathquote.Quote(loc.WorkspaceRoot), pathquote.Quote(loc.ConfigPath)),
		}
	}

	// Pasos 9 y 10: un <maestra>/openspec que no es el planning home resuelto.
	masterOpenSpec := filepath.Join(loc.Master, planningHomeDirName)
	if !isDir(masterOpenSpec) || pathidentity.SameDirectory(loc.PlanningHome, masterOpenSpec) {
		return nil
	}
	if fileExists(filepath.Join(loc.Master, WorkspacePointerFilename)) {
		return &LocateError{
			Kind:         LocateAmbiguousPlanningHome,
			Path:         masterOpenSpec,
			Cause:        fmt.Sprintf("la carpeta maestra %s tiene un puntero %s y un directorio %s propio, pero el planning home que resuelve el puntero es %s", pathquote.Quote(loc.Master), WorkspacePointerFilename, pathquote.Quote(masterOpenSpec), pathquote.Quote(loc.PlanningHome)),
			Continuation: fmt.Sprintf("Consolida los cambios de %s en %s y retira el directorio sobrante, o retira el puntero %s", pathquote.Quote(masterOpenSpec), pathquote.Quote(loc.PlanningHome), WorkspacePointerFilename),
		}
	}
	if separatesSpecs(loc.Config) {
		loc.Notes = append(loc.Notes, fmt.Sprintf(
			"Existe %s, pero el planning home resuelto es %s. Consolida los cambios de %s en el planning home resuelto para que no queden ocultos.",
			pathquote.Quote(masterOpenSpec), pathquote.Quote(loc.PlanningHome), pathquote.Quote(masterOpenSpec)))
	}
	return nil
}

// LegacySpecsDivergence devuelve el aviso Q14 de los consumidores de catálogo
// (REQ-24.10, D-07). Se emite solo en monorepo-embedded, con specs_repository
// fuera de {"", ".", "openspec"}, cuando existe <raíz>/<specs_repository>/specs y
// no es el mismo directorio que <planning home>/specs. Los consumidores leen
// únicamente el planning home; el aviso evita que la ubicación antigua quede
// oculta sin que nadie lo sepa. Es de solo lectura y se calcula en un único
// sitio para que los cuatro consumidores no reproduzcan la divergencia.
func (l Location) LegacySpecsDivergence() (string, bool) {
	if l.Config == nil || l.Config.Workspace.Topology != TopologyMonorepoEmbedded {
		return "", false
	}
	segments := specsSegments(l.Config.Workspace.SpecsRepository)
	if len(segments) == 0 || (len(segments) == 1 && sameSegment(segments[0], planningHomeDirName)) {
		return "", false
	}
	legacy := filepath.Join(specsRepositoryPath(l.Config.Workspace.SpecsRepository, l.WorkspaceRoot), "specs")
	current := filepath.Join(l.PlanningHome, "specs")
	if !isDir(legacy) || pathidentity.SameDirectory(legacy, current) {
		return "", false
	}
	return fmt.Sprintf(
		"Existe %s (workspace.specs_repository: %q), pero el catálogo se lee solo de %s y las especificaciones de la ubicación antigua quedan fuera. Consolídalas en %s o corrige `workspace.specs_repository` en %s.",
		pathquote.Quote(legacy), l.Config.Workspace.SpecsRepository, pathquote.Quote(current), pathquote.Quote(current), pathquote.Quote(l.ConfigPath)), true
}

// separatesSpecs informa de si la topología guarda las especificaciones en un
// repositorio distinto de la carpeta maestra (monorepo-decoupled y multirepo).
func separatesSpecs(cfg *WorkspaceConfig) bool {
	if cfg == nil {
		return false
	}
	switch cfg.Workspace.Topology {
	case TopologyMonorepoDecoupled, TopologyMultirepo:
		return true
	}
	return false
}

// Roots aplica el enraizado por topología (REQ-24.3, REQ-24.4). En
// monorepo-embedded, o sin configuración, workspaceRoot es la carpeta maestra.
// En monorepo-decoupled y multirepo es el repositorio de especificaciones o, si
// el último segmento de specs_repository es openspec, su padre. openspec/
// permanece siempre literal bajo workspaceRoot.
func Roots(cfg *WorkspaceConfig, master string) (workspaceRoot, planningHome string) {
	if !separatesSpecs(cfg) {
		return master, filepath.Join(master, planningHomeDirName)
	}
	repo := specsRepositoryPath(cfg.Workspace.SpecsRepository, master)
	if sameSegment(filepath.Base(repo), planningHomeDirName) {
		return filepath.Dir(repo), repo
	}
	return repo, filepath.Join(repo, planningHomeDirName)
}

// specsRepositoryPath devuelve la ruta del repositorio de especificaciones:
// vacío o "." es la propia carpeta maestra y una ruta absoluta se usa tal cual.
func specsRepositoryPath(specsRepository, master string) string {
	segments := specsSegments(specsRepository)
	if filepath.IsAbs(strings.TrimSpace(specsRepository)) {
		return filepath.Clean(strings.TrimSpace(specsRepository))
	}
	if len(segments) == 0 {
		return master
	}
	return filepath.Join(master, filepath.Join(segments...))
}

// specsSegments normaliza specs_repository tratando "/" y "\" como equivalentes
// y devuelve sus segmentos; vacío y "." no tienen segmentos.
func specsSegments(specsRepository string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(specsRepository), `\`, "/")
	cleaned := path.Clean(normalized)
	if normalized == "" || cleaned == "." {
		return nil
	}
	return strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
}

// deduceMaster determina la carpeta maestra de un axiom.yaml directo hallado en
// dir (REQ-24.5). En monorepo-embedded es siempre dir. En monorepo-decoupled y
// multirepo: (1) specs_repository vacío o "." → dir; (2) dir contiene un
// directorio con esa ruta → dir; (3) la ruta de dir termina con los segmentos
// de specs_repository → dir sin ese sufijo; (4) en otro caso, dir.
func deduceMaster(cfg *WorkspaceConfig, dir string) string {
	if !separatesSpecs(cfg) {
		return dir
	}
	raw := cfg.Workspace.SpecsRepository
	segments := specsSegments(raw)
	if len(segments) == 0 || filepath.IsAbs(strings.TrimSpace(raw)) || segments[0] == ".." {
		return dir
	}
	if isDir(filepath.Join(dir, filepath.Join(segments...))) {
		return dir
	}
	dirSegments := strings.Split(filepath.ToSlash(strings.TrimPrefix(dir, filepath.VolumeName(dir))), "/")
	if len(dirSegments) >= len(segments) {
		tail := dirSegments[len(dirSegments)-len(segments):]
		matches := true
		for i := range segments {
			if !sameSegment(tail[i], segments[i]) {
				matches = false
				break
			}
		}
		if matches {
			master := dir
			for range segments {
				master = filepath.Dir(master)
			}
			return master
		}
	}
	return dir
}

// sameSegment compara dos segmentos de ruta; no distingue mayúsculas en Windows.
func sameSegment(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
