package techcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// PathMode es la clasificación de una ruta frente a `technical_context`.
type PathMode string

const (
	// Maintained: la ruta está bajo `technical_context.maintained.paths`.
	Maintained PathMode = "maintained"
	// Reference: la ruta está bajo `technical_context.reference.paths`; nunca se modifica.
	Reference PathMode = "reference"
	// Outside: la ruta no está bajo ninguna ruta declarada.
	Outside PathMode = "outside"
)

// Scope reúne las rutas que deciden si el documento de una entrada puede
// editarse (REQ-24.30, D-16). Todas son relativas a WorkspaceRoot, con `/`; las
// de contexto pueden salir de él hacia un repositorio hermano (`../repo/docs/`).
type Scope struct {
	WorkspaceRoot string   // absoluta
	Maintained    []string // technical_context.maintained.paths
	Reference     []string // technical_context.reference.paths
}

// Resolve devuelve la ruta absoluta de una ruta declarada o de un documento del
// delta. Una ruta relativa, también la que sale hacia un repositorio hermano
// (`../repo/docs/`), se resuelve contra WorkspaceRoot y nunca contra el
// directorio de trabajo del proceso; una absoluta se respeta.
func (s Scope) Resolve(document string) string {
	native := filepath.FromSlash(document)
	if filepath.IsAbs(native) {
		return filepath.Clean(native)
	}
	return filepath.Join(s.WorkspaceRoot, native)
}

// CheckDocument comprueba, por identidad y no por texto, que el documento de la
// entrada está bajo alguna ruta mantenida y que no está bajo una ruta de
// referencia (REQ-24.30, REQ-24.31, D-16).
func (s Scope) CheckDocument(entry Entry) error {
	target := resolveDeepest(s.Resolve(entry.Document))
	for _, declared := range s.Reference {
		if underPath(s.Resolve(declared), target) {
			return fmt.Errorf("la entrada %s (línea %d) apunta a %s, que está bajo la ruta de referencia %q (modo reference): las rutas de referencia nunca se modifican", entry.ID, entry.Line, entry.Document, declared)
		}
	}
	for _, declared := range s.Maintained {
		if underPath(s.Resolve(declared), target) {
			return nil
		}
	}
	return fmt.Errorf("la entrada %s (línea %d) apunta a %s, que no está bajo ninguna ruta mantenida; las rutas mantenidas declaradas son: %s", entry.ID, entry.Line, entry.Document, quoteList(s.Maintained))
}

func quoteList(items []string) string {
	if len(items) == 0 {
		return "(ninguna)"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}

// underPath dice si target es declared o está dentro de él. Decide por identidad
// (pathidentity.Contains para un directorio, os.SameFile para un fichero), nunca
// comparando cadenas, para tolerar mayúsculas y enlaces simbólicos (H14). Una
// ruta declarada que aún no existe no tiene identidad: solo entonces se compara
// de forma léxica, porque no hay nada que identificar (workspace validate solo
// avisa de las rutas de contexto inexistentes).
func underPath(declared, target string) bool {
	info, err := os.Stat(declared)
	switch {
	case err == nil && info.IsDir():
		return pathidentity.Contains(declared, target)
	case err == nil:
		other, statErr := os.Stat(target)
		return statErr == nil && os.SameFile(info, other)
	default:
		return lexicallyUnder(resolveDeepest(declared), target)
	}
}

// lexicallyUnder es la comparación léxica de respaldo de underPath, sin
// distinguir mayúsculas en Windows.
func lexicallyUnder(declared, target string) bool {
	declared, target = filepath.Clean(declared), filepath.Clean(target)
	if runtime.GOOS == "windows" {
		declared, target = strings.ToLower(declared), strings.ToLower(target)
	}
	return target == declared || strings.HasPrefix(target, declared+string(filepath.Separator))
}

// resolveDeepest resuelve los enlaces simbólicos del tramo existente de una ruta
// y conserva tal cual el resto, de modo que un documento que aún no existe se
// juzga por el lugar real de su directorio.
func resolveDeepest(p string) string {
	p = filepath.Clean(p)
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(p, rest)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = parent
	}
}

// ScopeOf lee las rutas declaradas en `technical_context` de la ubicación, con
// `/` como separador y sin espacios sobrantes.
func ScopeOf(loc workspace.Location) Scope {
	scope := Scope{WorkspaceRoot: loc.WorkspaceRoot}
	normalize := func(paths []string) []string {
		out := make([]string, 0, len(paths))
		for _, p := range paths {
			out = append(out, strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
		}
		return out
	}
	if technical := loc.Declarations.TechnicalContext; technical != nil {
		if technical.Maintained != nil {
			scope.Maintained = normalize(technical.Maintained.Paths)
		}
		if technical.Reference != nil {
			scope.Reference = normalize(technical.Reference.Paths)
		}
	}
	return scope
}

// Classify dice si la ruta absoluta está bajo una ruta de referencia, bajo una
// mantenida o fuera de ambas. La pertenencia se decide por identidad y no por
// texto, y la referencia prevalece: lo que está bajo las dos nunca se modifica.
func (s Scope) Classify(abs string) PathMode {
	target := resolveDeepest(abs)
	for _, declared := range s.Reference {
		if underPath(s.Resolve(declared), target) {
			return Reference
		}
	}
	for _, declared := range s.Maintained {
		if underPath(s.Resolve(declared), target) {
			return Maintained
		}
	}
	return Outside
}

// RealPath resuelve los enlaces simbólicos del tramo existente de una ruta y conserva
// tal cual el resto; es la forma en que Classify compara las rutas.
func RealPath(path string) string { return resolveDeepest(path) }
