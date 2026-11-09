// Package docpaths fija la lista cerrada de rutas que `axiom sdd publish` considera
// documentación de un cambio (S6, REQ-27.5, DD-06). Es un paquete hoja y puro: no
// ejecuta git ni toca el sistema de ficheros, de modo que la política de rutas se
// prueba en tabla.
//
// La clasificación es siempre por raíz de ruta y nunca por extensión ni por nombre de
// fichero: `requirements.txt`, `CMakeLists.txt`, `docs/x.mdx` sin declarar o un
// `README.sh` en la raíz del repositorio son código y nunca se publican como
// documentación. Todo lo que no está en la lista es código.
package docpaths

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Set es la lista cerrada de rutas de documentación de un cambio en un repositorio.
// Todas las rutas son relativas a la raíz git y usan «/». El valor cero no reconoce
// ninguna ruta: se compone siempre con New.
type Set struct {
	changeDir   string         // <p>/changes/<x>
	archiveDir  string         // <p>/changes/archive
	archiveName *regexp.Regexp // <fecha>-<x>, nombre completo de la carpeta archivada
	specsDir    string         // <p>/specs
	indexFile   string         // <p>/INDEX.md
	baseline    []string
	maintained  []string
	reference   []string
}

// New compone el conjunto. planningRel es el planning home relativo a la raíz git
// ("" si es la raíz, como en el repositorio de specs de un multirepo); baseline,
// maintained y reference son las rutas declaradas en axiom.yaml, ya relativas a la
// raíz git (las que caen fuera de ella se descartan antes de llamar). Devuelve un
// error si el cambio no es un nombre de carpeta simple o si una ruta declarada es
// vacía, absoluta, sale de la raíz o es la raíz misma.
func New(planningRel, change string, baseline, maintained, reference []string) (Set, error) {
	if change == "" || change == "." || change == ".." || strings.ContainsAny(change, `/\`) {
		return Set{}, fmt.Errorf("el nombre de cambio %q no es un nombre de carpeta simple", change)
	}
	home, err := normalizeRoot(planningRel, true)
	if err != nil {
		return Set{}, fmt.Errorf("planning home: %w", err)
	}
	set := Set{
		changeDir:   join(home, "changes", change),
		archiveDir:  join(home, "changes", "archive"),
		archiveName: regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-` + regexp.QuoteMeta(change) + `$`),
		specsDir:    join(home, "specs"),
		indexFile:   join(home, "INDEX.md"),
	}
	for _, group := range []struct {
		name string
		in   []string
		out  *[]string
	}{
		{"baseline", baseline, &set.baseline},
		{"technical_context.maintained", maintained, &set.maintained},
		{"technical_context.reference", reference, &set.reference},
	} {
		for _, raw := range group.in {
			clean, err := normalizeRoot(raw, false)
			if err != nil {
				return Set{}, fmt.Errorf("%s: %w", group.name, err)
			}
			*group.out = append(*group.out, clean)
		}
	}
	return set, nil
}

// normalizeRoot limpia una ruta relativa a la raíz git. Con allowEmpty, la cadena
// vacía y «.» significan la raíz y devuelven "".
func normalizeRoot(raw string, allowEmpty bool) (string, error) {
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(raw), `\`, "/"))
	switch {
	case clean == "." && allowEmpty:
		return "", nil
	case clean == ".", clean == "/", strings.HasPrefix(clean, "/"), clean == ".." || strings.HasPrefix(clean, "../"):
		return "", fmt.Errorf("la ruta %q no está dentro de la raíz del repositorio", raw)
	case len(clean) > 1 && clean[1] == ':':
		return "", fmt.Errorf("la ruta %q es absoluta", raw)
	}
	return clean, nil
}

func join(parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "/")
}

// valid dice si rel es una ruta de git bien formada: relativa, limpia y con «/».
func valid(rel string) bool {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, `\`) {
		return false
	}
	return path.Clean(rel) == rel
}

func under(rel, dir string) bool { return strings.HasPrefix(rel, dir+"/") }

func underAny(rel string, dirs []string) bool {
	for _, dir := range dirs {
		if rel == dir || under(rel, dir) {
			return true
		}
	}
	return false
}

// Root devuelve la raíz de pathspec que cubre rel (la carpeta del cambio, la carpeta
// archivada, `<p>/specs`, `<p>/INDEX.md`, el documento de baseline o la ruta
// mantenida declarada) y false si rel no es documentación. Una ruta bajo
// `technical_context.reference` nunca lo es, aunque coincida con otra regla.
func (s Set) Root(rel string) (string, bool) {
	if s.changeDir == "" || !valid(rel) || underAny(rel, s.reference) {
		return "", false
	}
	switch {
	case under(rel, s.changeDir):
		return s.changeDir, true
	case under(rel, s.archiveDir):
		name, _, _ := strings.Cut(strings.TrimPrefix(rel, s.archiveDir+"/"), "/")
		if rel != s.archiveDir+"/"+name && s.archiveName.MatchString(name) {
			return s.archiveDir + "/" + name, true
		}
	case under(rel, s.specsDir):
		return s.specsDir, true
	case rel == s.indexFile:
		return s.indexFile, true
	}
	for _, document := range s.baseline {
		if rel == document {
			return document, true
		}
	}
	for _, root := range s.maintained {
		if rel == root || under(rel, root) {
			return root, true
		}
	}
	return "", false
}

// Contains dice si rel es documentación del cambio. Toda ruta que no lo es es código.
// Las comparaciones distinguen mayúsculas, como git.
func (s Set) Contains(rel string) bool {
	_, ok := s.Root(rel)
	return ok
}

// Candidates devuelve las raíces de pathspec donde puede haber documentación sucia:
// el conjunto de partida de la consulta de estado. Una raíz candidata puede contener
// rutas que Contains rechaza (la carpeta `archive` de otros cambios, por ejemplo):
// quien la consulta filtra el resultado con Contains. La lista es estable y sin
// repetidos.
func (s Set) Candidates() []string {
	if s.changeDir == "" {
		return nil
	}
	roots := []string{s.changeDir, s.archiveDir, s.specsDir, s.indexFile}
	roots = append(roots, s.baseline...)
	roots = append(roots, s.maintained...)
	seen := make(map[string]bool, len(roots))
	var unique []string
	for _, root := range roots {
		if !seen[root] {
			seen[root] = true
			unique = append(unique, root)
		}
	}
	return unique
}

// LocalState dice si rel es estado local de la carpeta del cambio que nunca se
// versiona: un fichero de su raíz cuyo nombre empieza por punto (`.gates.lock`,
// `.axiom-instance`, temporales de escritura atómica).
func (s Set) LocalState(rel string) bool {
	if s.changeDir == "" || !valid(rel) || !under(rel, s.changeDir) {
		return false
	}
	name := strings.TrimPrefix(rel, s.changeDir+"/")
	return !strings.Contains(name, "/") && strings.HasPrefix(name, ".")
}

// LocalStateGlob devuelve el glob, relativo a la raíz git, que deja fuera el estado local de la
// carpeta del cambio (`<p>/changes/<x>/.*`): quien confirma lo añade como exclusión de pathspec
// a las raíces que publica.
func (s Set) LocalStateGlob() string {
	return s.changeDir + "/.*"
}
