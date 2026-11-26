package reqtrace

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
)

// Source es una fuente de incrementos: sus ficheros, su prefijo dentro de la raíz git y la
// historia del repositorio que la contiene.
type Source struct {
	// Files es el sistema de ficheros con raíz en el directorio de la fuente (el planning
	// home para SDD, `odd/tasks` para ODD); nil si la fuente está ausente.
	Files fs.FS
	// GitPrefix es la raíz de Files relativa a la raíz git, con `/` y sin barra final.
	GitPrefix string
	// History es el puerto de git del repositorio de la fuente; nil con Files indica que la
	// fuente no está en un repositorio git.
	History isoevidence.History
	// Repo identifica el repositorio git de la fuente (su raíz): dos fuentes con el mismo Repo
	// comparten historia y sus commits son comparables entre sí.
	Repo string
}

var (
	// sddFolderName es el nombre seguro de una carpeta archivada: `<fecha>-<cambio>` sin
	// comodines ni magia de pathspec ni guion inicial (decisión 11).
	sddFolderName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[A-Za-z0-9][A-Za-z0-9._-]*$`)
	// tasksFileName reconoce `tasks.md` y `tasks.<rol>.md`.
	tasksFileName = regexp.MustCompile(`^tasks(\.[^.]+)?\.md$`)
)

// datePrefixLen es la longitud de `AAAA-MM-DD-`.
const datePrefixLen = len("2026-01-01-")

// sddFolder es una carpeta archivada de nombre seguro.
type sddFolder struct {
	// Name es `<fecha>-<cambio>`.
	Name string
	// Change es el nombre del cambio, sin la fecha.
	Change string
	// Dir es la ruta de la carpeta dentro de Source.Files.
	Dir string
	// Path es la ruta de la carpeta relativa a la raíz git.
	Path string
}

// readDirOrEmpty lee un directorio: uno que no existe es vacío y no una advertencia; otro
// fallo se informa como unreadable.
func readDirOrEmpty(files fs.FS, dir, gitPath string) ([]fs.DirEntry, []Warning) {
	entries, err := fs.ReadDir(files, dir)
	if err == nil {
		return entries, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return nil, []Warning{unreadable(gitPath, err)}
}

func unreadable(path string, err error) Warning {
	return Warning{Code: WarnUnreadable, Path: path, Message: fmt.Sprintf("no se pudo leer %s y se omite: %v; comprueba los permisos", path, err)}
}

func unsafeName(path string) Warning {
	return Warning{Code: WarnUnsafeName, Path: path, Message: fmt.Sprintf("el nombre de %s no es un nombre seguro y se omite sin consultar git; renómbralo con letras, cifras, `.`, `_` o `-`", path)}
}

// safeName dice si name cumple la regla de nombre seguro: sin `..` y con la forma re.
func safeName(re *regexp.Regexp, name string) bool {
	return re.MatchString(name) && !strings.Contains(name, "..")
}

// listSDDFolders lista las carpetas archivadas de `changes/archive`. Una carpeta de nombre
// no seguro se omite con unsafe_name; el directorio ausente o vacío da una lista vacía sin
// advertencia. Los cambios activos (`changes/<cambio>`) no se leen.
func listSDDFolders(src Source) ([]sddFolder, []Warning) {
	if src.Files == nil {
		return nil, nil
	}
	archiveGit := joinPath(src.GitPrefix, "changes/archive")
	entries, warnings := readDirOrEmpty(src.Files, "changes/archive", archiveGit)
	var folders []sddFolder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := joinPath(archiveGit, entry.Name())
		if !safeName(sddFolderName, entry.Name()) {
			warnings = append(warnings, unsafeName(path))
			continue
		}
		folders = append(folders, sddFolder{
			Name:   entry.Name(),
			Change: entry.Name()[datePrefixLen:],
			Dir:    joinPath("changes/archive", entry.Name()),
			Path:   path,
		})
	}
	sort.Slice(folders, func(i, j int) bool { return folders[i].Name < folders[j].Name })
	return folders, warnings
}

// readTraces lee las trazas ids.yaml de las carpetas archivadas.
func readSDDTraces(src Source, folders []sddFolder) ([]traceSet, []Warning) {
	var sets []traceSet
	var warnings []Warning
	for _, f := range folders {
		set, w := readSDDTrace(src.Files, f.Dir, f.Path)
		sets = append(sets, set)
		warnings = append(warnings, w...)
	}
	return sets, warnings
}

// oddDocName es el nombre seguro de un documento ODD (decisión 11).
var oddDocName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

// oddDoc es un documento `odd/tasks/<feature>.md` de nombre seguro y ya leído.
type oddDoc struct {
	// Name es la feature, sin `.md`.
	Name string
	// Path es la ruta del documento relativa a la raíz git.
	Path string
	Text string
}

// listODDDocs lee los documentos `*.md` de la raíz de la fuente ODD. Uno de nombre no seguro
// se omite con unsafe_name y uno ilegible con unreadable; el directorio ausente o vacío da
// una lista vacía sin advertencia.
func listODDDocs(src Source) ([]oddDoc, []Warning) {
	if src.Files == nil {
		return nil, nil
	}
	entries, warnings := readDirOrEmpty(src.Files, ".", src.GitPrefix)
	var docs []oddDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := joinPath(src.GitPrefix, entry.Name())
		if !safeName(oddDocName, entry.Name()) {
			warnings = append(warnings, unsafeName(path))
			continue
		}
		data, err := fs.ReadFile(src.Files, entry.Name())
		if err != nil {
			warnings = append(warnings, unreadable(path, err))
			continue
		}
		docs = append(docs, oddDoc{Name: strings.TrimSuffix(entry.Name(), ".md"), Path: path, Text: string(data)})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	return docs, warnings
}

// readODDTraces lee la traza `## Identificadores asignados` de cada documento.
func readODDTraces(docs []oddDoc) ([]traceSet, []Warning) {
	var sets []traceSet
	var warnings []Warning
	for _, d := range docs {
		set, w := readODDTrace(d.Text, d.Path)
		sets = append(sets, set)
		warnings = append(warnings, w...)
	}
	return sets, warnings
}
