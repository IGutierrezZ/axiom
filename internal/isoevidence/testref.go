package isoevidence

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// maxReferenceFileBytes acota la lectura de un fichero referenciado (D11). Un
// símbolo que solo aparece más allá de este límite no resuelve.
const maxReferenceFileBytes = 4 << 20

// ResolveOutcome es el resultado de resolver una referencia `ruta:símbolo`.
type ResolveOutcome string

const (
	OutcomeResolved      ResolveOutcome = "resolved"
	OutcomeFileMissing   ResolveOutcome = "file_missing"
	OutcomeSymbolMissing ResolveOutcome = "symbol_missing"
	OutcomeOutsideTree   ResolveOutcome = "outside_tree"
)

// TreeResolver resuelve las referencias de prueba de la matriz contra el árbol
// de trabajo (REQ-29.7). Solo lee: nunca ejecuta ni interpreta el contenido.
type TreeResolver interface {
	// ResolvePathSymbol comprueba que relPath es un fichero del árbol y que
	// symbol figura en él como identificador completo. Una ruta absoluta, con
	// unidad, UNC, con `..` o que atraviesa un enlace simbólico fuera del árbol
	// devuelve OutcomeOutsideTree, sin error.
	ResolvePathSymbol(relPath, symbol string) (ResolveOutcome, error)
	// ResolveGoTest comprueba que algún `*_test.go` del árbol declara la
	// función de nivel superior name.
	ResolveGoTest(name string) (bool, error)
}

// ClosableTreeResolver es el resolutor basado en `os.Root`, que mantiene un
// descriptor abierto hasta Close.
type ClosableTreeResolver interface {
	TreeResolver
	io.Closer
}

// skippedDirs son los directorios que el recorrido de `TestX` no visita.
var skippedDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	"testdata":     true,
}

type rootResolver struct {
	root *os.Root

	indexOnce sync.Once
	goTests   map[string]bool
	indexErr  error
}

// NewTreeResolver abre root con `os.Root`, que rechaza rutas absolutas, `..` y
// enlaces simbólicos que salgan del árbol. El llamador cierra el resolutor.
func NewTreeResolver(root string) (ClosableTreeResolver, error) {
	opened, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la raíz del árbol %q: %w", root, err)
	}
	return &rootResolver{root: opened}, nil
}

func (r *rootResolver) Close() error { return r.root.Close() }

// outsideTreeLexically dice si la ruta, antes de tocar el disco, no es una ruta
// relativa con `/` dentro del árbol.
func outsideTreeLexically(rel string) bool {
	if rel == "" {
		return false
	}
	if strings.Contains(rel, `\`) {
		return true // no es una ruta con `/`: cubre `C:\x` y `\\srv\x`
	}
	if strings.HasPrefix(rel, "/") || hasVolumePrefix(rel) {
		return true
	}
	cleaned := path.Clean(rel)
	return cleaned == ".." || strings.HasPrefix(cleaned, "../")
}

// hasVolumePrefix reconoce `C:` y `C:/...` al principio de la ruta.
func hasVolumePrefix(rel string) bool {
	if len(rel) < 2 || rel[1] != ':' {
		return false
	}
	c := rel[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (r *rootResolver) ResolvePathSymbol(relPath, symbol string) (ResolveOutcome, error) {
	if outsideTreeLexically(relPath) {
		return OutcomeOutsideTree, nil
	}
	if relPath == "" {
		return OutcomeFileMissing, nil
	}
	f, err := r.root.Open(path.Clean(relPath))
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return OutcomeFileMissing, nil
		case strings.Contains(err.Error(), "escapes"):
			return OutcomeOutsideTree, nil
		default:
			return "", fmt.Errorf("no se pudo abrir %q: %w", relPath, err)
		}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("no se pudo leer %q: %w", relPath, err)
	}
	if info.IsDir() {
		return OutcomeFileMissing, nil
	}
	content, err := io.ReadAll(io.LimitReader(f, maxReferenceFileBytes))
	if err != nil {
		return "", fmt.Errorf("no se pudo leer %q: %w", relPath, err)
	}
	if containsIdentifier(string(content), symbol) {
		return OutcomeResolved, nil
	}
	return OutcomeSymbolMissing, nil
}

// isIdentifierRune son los caracteres que forman parte de un identificador:
// letras, dígitos, `_` y `$`.
func isIdentifierRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// containsIdentifier busca symbol como identificador completo: ni el carácter
// anterior ni el siguiente forman parte de un identificador.
func containsIdentifier(content, symbol string) bool {
	if symbol == "" {
		return false
	}
	for from := 0; from <= len(content)-len(symbol); {
		i := strings.Index(content[from:], symbol)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(symbol)
		beforeOK, afterOK := true, true
		if start > 0 {
			prev, _ := utf8.DecodeLastRuneInString(content[:start])
			beforeOK = !isIdentifierRune(prev)
		}
		if end < len(content) {
			next, _ := utf8.DecodeRuneInString(content[end:])
			afterOK = !isIdentifierRune(next)
		}
		if beforeOK && afterOK {
			return true
		}
		from = start + 1
	}
	return false
}

// isGoTestName dice si name tiene la forma de una función de prueba de Go:
// `Test` seguido de nada o de un carácter que no sea una minúscula.
func isGoTestName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Test")
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	first, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(first)
}

func (r *rootResolver) ResolveGoTest(name string) (bool, error) {
	if !isGoTestName(name) {
		return false, nil
	}
	r.indexOnce.Do(func() { r.goTests, r.indexErr = r.buildGoTestIndex() })
	if r.indexErr != nil {
		return false, r.indexErr
	}
	return r.goTests[name], nil
}

// buildGoTestIndex recorre el árbol una sola vez y recoge las funciones de
// prueba de nivel superior de los `*_test.go`. El prefiltro textual evita
// analizar los ficheros que no declaran ninguna `func Test`; la confirmación
// con `go/parser` descarta comentarios, cadenas y métodos.
func (r *rootResolver) buildGoTestIndex() (map[string]bool, error) {
	index := map[string]bool{}
	fsys := r.root.FS()
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != "." && skippedDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, err := fsys.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		src, err := io.ReadAll(io.LimitReader(f, maxReferenceFileBytes))
		if err != nil || !strings.Contains(string(src), "func Test") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), p, src, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
				index[fn.Name.Name] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo recorrer el árbol: %w", err)
	}
	return index, nil
}
