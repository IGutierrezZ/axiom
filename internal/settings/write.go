package settings

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// WriteRequest es una petición de escritura sobre un axiom.yaml.
type WriteRequest struct {
	ConfigPath       string // ruta de Locate(...).ConfigPath
	Master           string // carpeta maestra, base de las comprobaciones contra el disco
	ExpectedRevision string // revisión que el llamante leyó
	Patch            Patch
}

// WriteResult es el resultado de una escritura correcta.
type WriteResult struct {
	Changed  bool
	Revision string
	Warnings []string // errores del motor que ya existían antes del parche
}

// beforeReplace es la costura de las pruebas: se ejecuta tras validar y antes de
// comprobar de nuevo la revisión y reemplazar el fichero. En producción no hace nada.
var beforeReplace = func(path string) {}

// Write aplica el parche sobre el axiom.yaml de req.ConfigPath. Secuencia:
// leer y calcular la revisión; compararla con la esperada; parchear el árbol,
// serializar con `SetIndent(2)` y restaurar el formato; validar el documento
// resultante entero (ParseConfig y las comprobaciones del motor contra el disco,
// de las que solo bloquean los errores nuevos); comprobar de nuevo la revisión y
// reemplazar el fichero de forma atómica. Si algo falla, el fichero no cambia.
func Write(req WriteRequest) (*WriteResult, error) {
	if strings.TrimSpace(req.ExpectedRevision) == "" {
		return nil, newError(CodeInvalidRequest, "falta la revisión esperada de axiom.yaml: lee los ajustes y envía la revisión recibida")
	}
	if len(req.Patch) == 0 {
		return nil, newError(CodeInvalidRequest, "no hay ningún cambio que aplicar: el parche está vacío")
	}
	original, err := os.ReadFile(req.ConfigPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, newError(CodeConfigMissing, "el proyecto no tiene un axiom.yaml en "+req.ConfigPath)
		}
		return nil, newError(CodeWriteFailed, fmt.Sprintf("no se pudo leer axiom.yaml: %v; no se ha escrito nada", err))
	}
	rev0 := revisionOf(original)
	if rev0 != req.ExpectedRevision {
		return nil, newError(CodeRevisionMismatch, "axiom.yaml ha cambiado desde que se leyó: recarga los ajustes y vuelve a intentarlo")
	}

	doc := parseDocument(original)
	if doc.blocked != "" {
		return nil, newError(CodeUnsupportedDocument, "axiom.yaml no se puede editar desde Ajustes: "+doc.blocked)
	}
	if perr := doc.applyPatch(req.Patch); perr != nil {
		return nil, perr
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc.top); err != nil {
		return nil, newError(CodeValidationFailed, fmt.Sprintf("no se pudo serializar el axiom.yaml resultante: %v", err))
	}
	if err := enc.Close(); err != nil {
		return nil, newError(CodeValidationFailed, fmt.Sprintf("no se pudo serializar el axiom.yaml resultante: %v", err))
	}
	updated := restoreLayout(original, buf.Bytes())
	if bytes.Equal(updated, original) {
		return &WriteResult{Changed: false, Revision: rev0}, nil
	}

	cfg, err := workspace.ParseConfig(updated)
	if err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: "el axiom.yaml resultante no es válido; no se ha escrito nada", Violations: errorLines(err.Error())}
	}
	// Sin un original válido no hay conjunto de errores previos que comparar: todos
	// los errores del motor del documento nuevo cuentan como introducidos.
	var previous []string
	if oldCfg, oldErr := workspace.ParseConfig(original); oldErr == nil {
		previous = engineErrors(oldCfg, req.Master)
	}
	var introduced, warnings []string
	for _, msg := range engineErrors(cfg, req.Master) {
		if slices.Contains(previous, msg) {
			warnings = append(warnings, msg)
		} else {
			introduced = append(introduced, msg)
		}
	}
	if len(introduced) > 0 {
		return nil, &Error{Code: CodeValidationFailed, Message: "el cambio introduce errores en la validación del proyecto; no se ha escrito nada", Violations: introduced}
	}

	beforeReplace(req.ConfigPath)
	current, err := os.ReadFile(req.ConfigPath)
	if err != nil {
		return nil, newError(CodeWriteFailed, fmt.Sprintf("no se pudo releer axiom.yaml antes de reemplazarlo: %v; no se ha escrito nada", err))
	}
	if revisionOf(current) != rev0 {
		return nil, newError(CodeRevisionMismatch, "axiom.yaml cambió mientras se validaba el cambio: recarga los ajustes y vuelve a intentarlo; no se ha escrito nada")
	}
	result, err := filemerge.WriteFileAtomic(req.ConfigPath, updated, 0o644)
	if err != nil {
		state := "axiom.yaml conserva su contenido anterior"
		if result.Changed {
			state = "el reemplazo llegó a producirse: revisa el contenido de axiom.yaml"
		}
		return nil, newError(CodeWriteFailed, fmt.Sprintf("no se pudo escribir axiom.yaml: %v; %s", err, state))
	}
	return &WriteResult{Changed: true, Revision: revisionOf(updated), Warnings: warnings}, nil
}

// errorLines separa un error acumulado en una violación por línea.
func errorLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// engineErrors reúne los errores de las comprobaciones del motor contra el disco, las
// mismas que `axiom workspace validate`: Validate, ValidateDeclarationPaths y
// ValidateRepositoryPolicies, con master como base.
func engineErrors(cfg *workspace.WorkspaceConfig, master string) []string {
	var out []string
	report, err := workspace.Validate(workspace.DefaultFS(), cfg, master)
	if err != nil {
		out = append(out, err.Error())
	} else {
		out = append(out, report.Errors...)
	}
	declarationErrors, _ := workspace.ValidateDeclarationPaths(cfg, master)
	out = append(out, declarationErrors...)
	repositoryErrors, _ := workspace.ValidateRepositoryPolicies(cfg, master)
	out = append(out, repositoryErrors...)
	return out
}
