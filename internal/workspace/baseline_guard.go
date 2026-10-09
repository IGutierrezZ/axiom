package workspace

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// BaselineOwnership es la respuesta de la consulta única «¿la baseline declarada
// es del proyecto?» (REQ-24.22).
type BaselineOwnership string

const (
	// BaselineDefault: no hay axiom.yaml, o es legible y no contiene `baseline:`
	// (aunque tenga errores de validación ajenos a ese bloque). Los creadores
	// operan como siempre sobre <planning home>/specs.
	BaselineDefault BaselineOwnership = "default"
	// BaselineProject: axiom.yaml es legible y su bloque `baseline:` es válido.
	BaselineProject BaselineOwnership = "project"
	// BaselineIndeterminate: axiom.yaml no se puede leer como YAML o su bloque
	// `baseline:` es inválido. Los creadores rechazan su escritura (REQ-24.18).
	BaselineIndeterminate BaselineOwnership = "indeterminate"
)

// Ownership es la única implementación de la consulta de baseline (REQ-24.22):
// Location.BaselineOwnership delega en ella, y axiom init la usa directamente
// sobre las declaraciones del propio destino. Un error de otra sección de
// axiom.yaml, o un `technical_context:` inválido, no altera la respuesta, y
// tampoco una causa de almacén (REQ-24.62), que no forma parte de Declarations.
//
// Con BaselineIndeterminate el error devuelto es la causa.
func (d Declarations) Ownership() (BaselineOwnership, error) {
	switch {
	case d.ParseErr != nil:
		return BaselineIndeterminate, d.ParseErr
	case d.BaselineErr != nil:
		return BaselineIndeterminate, d.BaselineErr
	case d.Baseline != nil:
		return BaselineProject, nil
	default:
		return BaselineDefault, nil
	}
}

// BaselineOwnership responde la consulta de baseline para la configuración
// resuelta. Cuando el primitivo de resolución rechaza, no hay Location ni
// respuesta (REQ-24.22): quien llama propaga ese *LocateError.
func (l Location) BaselineOwnership() (BaselineOwnership, error) {
	return l.Declarations.Ownership()
}

// DeclaredDomains devuelve los dominios del bloque `baseline:`, en el orden en
// que se declaran.
func (d Declarations) DeclaredDomains() []string {
	if d.Baseline == nil {
		return nil
	}
	domains := make([]string, 0, len(d.Baseline.Documents))
	for _, document := range d.Baseline.Documents {
		domains = append(domains, document.Domain)
	}
	return domains
}

// SecondBaselineError es el rechazo de una operación que habría creado o
// actualizado <planning home>/specs o <planning home>/INDEX.md, es decir, una
// segunda baseline, cuando la baseline es del proyecto o no se puede determinar.
// Es distinguible con errors.As para que cada canal (CLI, HTTP, TUI) lo presente
// a su manera.
type SecondBaselineError struct {
	Operation  string            // operación rechazada, por ejemplo `axiom archive sync`
	Ownership  BaselineOwnership // project o indeterminate
	ConfigPath string            // axiom.yaml que declara la baseline
	Master     string            // carpeta maestra, para la continuación `workspace validate`
	Domains    []string          // dominios declarados; vacío con indeterminate
	Cause      error             // causa con indeterminate
}

// Error compone la baseline declarada o la causa y la continuación.
func (e *SecondBaselineError) Error() string {
	if e.Ownership == BaselineIndeterminate {
		return fmt.Sprintf("%s se rechaza: no se puede determinar si la baseline de especificaciones es del proyecto (%v). Ejecuta `axiom workspace validate --path %s` y corrige %s antes de volver a intentarlo",
			e.Operation, e.Cause, pathquote.Quote(e.Master), pathquote.Quote(e.ConfigPath))
	}
	return fmt.Sprintf("%s se rechaza: la baseline de especificaciones es del proyecto (%s declara los dominios %s) y escribir en openspec/specs o en openspec/INDEX.md produciría una segunda baseline. Integra el cambio con el flujo de archive o retira el bloque `baseline:` de %s para volver a la baseline de OpenSpec",
		e.Operation, pathquote.Quote(e.ConfigPath), strings.Join(e.Domains, ", "), pathquote.Quote(e.ConfigPath))
}

// Unwrap expone la causa de una baseline indeterminada.
func (e *SecondBaselineError) Unwrap() error { return e.Cause }

// GuardSpecsWrite devuelve nil si op puede crear o actualizar
// <planning home>/specs o <planning home>/INDEX.md, y un *SecondBaselineError si
// la baseline es del proyecto o indeterminada (REQ-24.23).
func (l Location) GuardSpecsWrite(op string) error {
	ownership, cause := l.BaselineOwnership()
	if ownership == BaselineDefault {
		return nil
	}
	return &SecondBaselineError{
		Operation:  op,
		Ownership:  ownership,
		ConfigPath: l.ConfigPath,
		Master:     l.Master,
		Domains:    l.Declarations.DeclaredDomains(),
		Cause:      cause,
	}
}

// SamePlanningHome compara dos planning homes por identidad
// (pathidentity.SameDirectory) y, cuando alguno no existe, por ruta absoluta
// limpia, sin distinguir mayúsculas en Windows. REQ-2.2 compara por ruta y no
// exige que existan: axiom init es quien crea <ruta>/openspec.
//
// Ese respaldo léxico queda fuera de la política de internal/pathidentity, que
// pide declararlo donde se use. Aquí es aceptable porque las dos rutas se derivan
// léxicamente de la misma <ruta> y de su configuración antes de que exista nada
// que comparar por identidad.
func SamePlanningHome(a, b string) bool {
	if pathidentity.SameDirectory(a, b) {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	absA, absB = filepath.Clean(absA), filepath.Clean(absB)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(absA, absB)
	}
	return absA == absB
}
