package engine

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// axiomDocument es una ruta que la clasificación léxica reconoce como documento de Axiom: Change
// es el cambio al que pertenece (en `changes/archive/<AAAA-MM-DD>-<x>/…`, `<x>`) y Lane, el carril
// (`sdd` para la carpeta de un cambio y `odd` para `odd/tasks/<f>.md`).
type axiomDocument struct {
	Path   string
	Change string
	Lane   branch.Lane
}

// archivedFolder es la carpeta de un cambio archivado, `<AAAA-MM-DD>-<x>`: el grupo 1 es `<x>`.
var archivedFolder = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

// classifyAxiomDocument clasifica path por sus segmentos, sin tocar el disco ni lanzar procesos y
// sin interpretar jamás el contenido ni la extensión salvo para `odd/tasks/` (design D-18, paso 2).
// Un documento de Axiom tiene los segmentos `openspec/changes/<x>/…` —cualquier fichero bajo la
// carpeta de un cambio, sea cual sea su nombre—, `openspec/changes/archive/<AAAA-MM-DD>-<x>/…`
// (el cambio es `<x>`) u `odd/tasks/<f>.md`, que exige la extensión `.md`: `odd/tasks/README.sh`
// es otro fichero. Cualquier otra ruta es «otro fichero» (false).
//
// La ruta se limpia antes de clasificarla (`openspec/changes/x/../../a.go` es otro fichero) y,
// con foldCase —Windows—, los segmentos fijos no distinguen mayúsculas. Que la ruta esté de verdad
// bajo las carpetas del proyecto lo confirma después `Locate` (containsDocument).
func classifyAxiomDocument(path string, foldCase bool) (axiomDocument, bool) {
	if path == "" {
		return axiomDocument{}, false
	}
	cleaned := filepath.Clean(path)
	segments := strings.Split(filepath.ToSlash(cleaned), "/")
	same := func(segment, fixed string) bool {
		if foldCase {
			return strings.EqualFold(segment, fixed)
		}
		return segment == fixed
	}

	for i := 0; i+1 < len(segments); i++ {
		switch {
		case same(segments[i], "openspec") && same(segments[i+1], "changes"):
			if change, ok := changeFolderName(segments[i+2:], same); ok {
				return axiomDocument{Path: cleaned, Change: change, Lane: branch.LaneSDD}, true
			}
		case same(segments[i], "odd") && same(segments[i+1], "tasks") && i+3 == len(segments):
			file := segments[i+2]
			if len(file) > len(".md") && same(file[len(file)-len(".md"):], ".md") {
				return axiomDocument{Path: cleaned, Change: file[:len(file)-len(".md")], Lane: branch.LaneODD}, true
			}
		}
	}
	return axiomDocument{}, false
}

// changeFolderName devuelve el cambio de los segmentos que siguen a `openspec/changes`: hace falta
// la carpeta del cambio y al menos un segmento más (un fichero o una subcarpeta suya). La carpeta
// `archive` no es un cambio: el cambio archivado es la carpeta `<AAAA-MM-DD>-<x>` que contiene.
func changeFolderName(rest []string, same func(segment, fixed string) bool) (string, bool) {
	if len(rest) < 2 {
		return "", false
	}
	if !same(rest[0], "archive") {
		return rest[0], true
	}
	if len(rest) < 3 {
		return "", false
	}
	match := archivedFolder.FindStringSubmatch(rest[1])
	if match == nil {
		return "", false
	}
	return match[1], true
}
