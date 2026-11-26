// Package docpaths fija la lista cerrada de rutas que `axiom sdd publish` considera
// documentación de un cambio (S6, RF-PUB-01, DD-06). Es un paquete hoja y puro: no
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
	change      string         // <x>, nombre simple de la carpeta del cambio
	changeDir   string         // <p>/changes/<x>
	archiveDir  string         // <p>/changes/archive
	archiveName *regexp.Regexp // <fecha>-<x>, nombre completo de la carpeta archivada
	specsDir    string         // <p>/specs
	indexFile   string         // <p>/INDEX.md
	maintained  []string
	reference   []string
}

// New compone el conjunto. planningRel es el planning home relativo a la raíz git
// ("" si es la raíz, como en el repositorio de specs de un multirepo); maintained y reference son las rutas declaradas en axiom.yaml, ya relativas a la
// raíz git (las que caen fuera de ella se descartan antes de llamar). Devuelve un
// error si el cambio no es un nombre de carpeta simple o si una ruta declarada es
// vacía, absoluta, sale de la raíz o es la raíz misma.
func New(planningRel, change string, maintained, reference []string) (Set, error) {
	if change == "" || change == "." || change == ".." || strings.ContainsAny(change, `/\`) {
		return Set{}, fmt.Errorf("el nombre de cambio %q no es un nombre de carpeta simple", change)
	}
	home, err := normalizeRoot(planningRel, true)
	if err != nil {
		return Set{}, fmt.Errorf("planning home: %w", err)
	}
	set := Set{
		change:      change,
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
// archivada, `<p>/specs`, `<p>/INDEX.md` o la ruta
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

// LocalState dice si rel es estado local del cambio que nunca se versiona: el contenido de `context/` y
// `.context-tmp/` (datos importados de un work item, D-16) o un fichero cuyo nombre
// empieza por punto y que está directamente en la raíz de la carpeta del cambio o de su carpeta
// archivada (`.gates.lock`, `.axiom-instance`, `.integrations-outbox.yaml`, temporales de
// escritura atómica). Un fichero con punto anidado, o en la carpeta archivada de otro cambio, sigue
// siendo documentación (MODIFIED RF-PUB-01).
func (s Set) LocalState(rel string) bool {
	if s.changeDir == "" || !valid(rel) {
		return false
	}
	if under(rel, s.changeDir) {
		return localFile(strings.TrimPrefix(rel, s.changeDir+"/"))
	}
	if under(rel, s.archiveDir) {
		name, file, found := strings.Cut(strings.TrimPrefix(rel, s.archiveDir+"/"), "/")
		return found && s.archiveName.MatchString(name) && localFile(file)
	}
	return false
}

// Carpetas de contexto importado de un work item (D-16): el contenido de `context/` y los temporales de
// su copia son datos de un tercero que nunca se versionan.
const (
	contextDir    = "context"
	contextTmpDir = ".context-tmp"
)

// localFile dice si name es estado local de la raíz de un cambio: un fichero de una sola componente cuyo
// nombre empieza por punto, o cualquier ruta bajo `context/` o `.context-tmp/`.
func localFile(name string) bool {
	if first, _, nested := strings.Cut(name, "/"); nested {
		return first == contextDir || first == contextTmpDir
	}
	return strings.HasPrefix(name, ".")
}

// LocalStateGlobs devuelve los globs, relativos a la raíz git, que dejan fuera el estado local del
// cambio: `<p>/changes/<x>/.*` para la carpeta ordinaria y
// `<p>/changes/archive/AAAA-MM-DD-<x>/.*` para la carpeta archivada de ese cambio. Quien confirma
// los añade como exclusiones de pathspec a las raíces que publica. Un nombre de cambio válido no
// contiene metacaracteres de glob.
func (s Set) LocalStateGlobs() []string {
	archived := s.archiveDir + "/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-" + s.change
	return []string{
		s.changeDir + "/.*",
		archived + "/.*",
		s.changeDir + "/" + contextDir + "/**",
		s.changeDir + "/" + contextTmpDir + "/**",
		archived + "/" + contextDir + "/**",
		archived + "/" + contextTmpDir + "/**",
	}
}

// FromWorkspace compone el conjunto a partir de lo que declara axiom.yaml: las rutas de
// `technical_context` son relativas al workspaceRoot y se convierten aquí a relativas a la raíz
// git. planningRel es el planning home relativo a la raíz git, que siempre es
// `<workspaceRoot>/openspec`, así que el workspaceRoot se deriva de él. Las rutas declaradas que
// caen fuera de la raíz git o son la propia raíz se descartan. Es el único punto que compone la
// lista cerrada: `axiom sdd publish` y el estado de publicación del status la comparten, de modo
// que no pueden divergir.
func FromWorkspace(planningRel, change string, maintained, reference []string) (Set, error) {
	wsRel := workspaceRelative(planningRel)
	return New(planningRel, change, rebase(wsRel, maintained), rebase(wsRel, reference))
}

// rebase convierte cada ruta declarada a relativa a la raíz git y descarta las que no caben.
func rebase(wsRel string, declared []string) []string {
	var rebased []string
	for _, raw := range declared {
		if rel, ok := fromWorkspace(wsRel, raw); ok {
			rebased = append(rebased, rel)
		}
	}
	return rebased
}

// workspaceRelative es el workspaceRoot relativo a la raíz git, derivado del planning home, que es
// siempre `<workspaceRoot>/openspec`: vacío si ambos coinciden con la raíz.
func workspaceRelative(planningRel string) string {
	if planningRel == "" {
		return ""
	}
	if dir := path.Dir(planningRel); dir != "." {
		return dir
	}
	return ""
}

// fromWorkspace convierte una ruta declarada, relativa al workspaceRoot, a una relativa a la raíz
// git. Las que caen fuera de la raíz git o son la propia raíz se descartan.
func fromWorkspace(wsRel, declared string) (string, bool) {
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(declared), `\`, "/"))
	switch {
	case clean == "." || clean == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../"):
		return "", false
	case len(clean) > 1 && clean[1] == ':':
		return "", false
	}
	joined := path.Join(wsRel, clean)
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", false
	}
	return joined, true
}

// exactAllowedName es el único nombre que el modo acotado de la publicación (`Only`) acepta: el
// fichero de marcas de ejecución de `pruebas.md`, relativo a la carpeta del cambio.
const exactAllowedName = "pruebas-estado.yaml"

// ExactAllowed dice si name es una ruta que el modo acotado de la publicación puede confirmar.
// name es relativo a la carpeta del cambio y solo se acepta el nombre exacto de
// `pruebas-estado.yaml`: ni otros documentos, ni código, ni rutas con separadores, `..` o
// absolutas. La comparación es pura (sin git ni disco) y distingue mayúsculas, como git.
func ExactAllowed(name string) bool { return name == exactAllowedName }
