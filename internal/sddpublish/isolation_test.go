package sddpublish

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Aislamiento de la operación remota (DD-14, REQ-27.11, REQ-25.1): el único código del producto que
// ejecuta `git commit` y `git push` vive en este paquete, solo lo alcanza `internal/cli/sdd_publish.go`
// y ningún método de `internal/branch/**` publica. Son comprobaciones estáticas sobre los ficheros de
// producción de `internal/` y `cmd/`, con `go/parser`.

const (
	modulePath       = "github.com/IGutierrezZ/axiom/v3"
	sddpublishImport = modulePath + "/internal/sddpublish"
	cliPackage       = "internal/cli"
	cliPublishFile   = "internal/cli/sdd_publish.go"
)

// productionFile es un fichero Go de producción con su ruta relativa a la raíz del repositorio.
type productionFile struct {
	label   string // con «/», relativa a la raíz del repositorio
	content []byte
}

func (f productionFile) dir() string { return path.Dir(f.label) }

// walkProduction devuelve los ficheros Go de producción (sin `_test.go`) bajo `internal/` y `cmd/`.
func walkProduction(t *testing.T) []productionFile {
	t.Helper()
	root := filepath.Join("..", "..")
	var files []productionFile
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			name := entry.Name()
			if entry.IsDir() {
				if name == "testdata" || strings.HasPrefix(name, ".") {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			files = append(files, productionFile{label: filepath.ToSlash(relative), content: content})
			return nil
		})
		if err != nil {
			t.Fatalf("recorrer %s: %v", top, err)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].label < files[j].label })
	return files
}

// sddpublishName devuelve el nombre local con el que el fichero importa sddpublish, o "" si no lo
// importa. Un import con `_` o `.` no cuenta como selector pero sí como importación.
func sddpublishName(file *ast.File) (name string, imports bool) {
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != sddpublishImport {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name, true
		}
		return "sddpublish", true
	}
	return "", false
}

// importViolations analiza un fichero de producción y nombra cada forma en que rompe el aislamiento
// por importaciones: importar sddpublish fuera de `internal/cli`, o usar su selector fuera de
// `internal/cli/sdd_publish.go`. Los ficheros del propio paquete sddpublish quedan exentos.
func importViolations(label string, source []byte) ([]string, error) {
	if path.Dir(label) == "internal/sddpublish" {
		return nil, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), label, source, 0)
	if err != nil {
		return nil, err
	}
	name, imports := sddpublishName(file)
	if !imports {
		return nil, nil
	}
	var violations []string
	if path.Dir(label) != cliPackage {
		violations = append(violations, label+" importa internal/sddpublish: solo lo importa "+cliPackage)
	}
	if label != cliPublishFile {
		violations = append(violations, label+" importa internal/sddpublish: solo "+cliPublishFile+" puede usarlo")
	}
	if name != "_" && name != "." {
		used := false
		ast.Inspect(file, func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok {
				if ident, isIdent := selector.X.(*ast.Ident); isIdent && ident.Name == name {
					used = true
				}
			}
			return true
		})
		if used && label != cliPublishFile {
			violations = append(violations, label+" usa el selector "+name+".")
		}
	}
	return violations, nil
}

// runCalls son las funciones por las que `internal/branch/**` ejecuta git: ninguna puede recibir los
// literales "push" ni "commit" (REQ-25.1).
var runCalls = map[string]bool{"Run": true, "RunWithTimeout": true, "runWrite": true, "runRead": true}

// publishingLiterals son los subcomandos que solo ejecuta este paquete.
var publishingLiterals = map[string]bool{"push": true, "commit": true}

// publishingCallViolations nombra cada llamada a una función de ejecución de git (`gitcommon.Run`,
// `gitcommon.RunWithTimeout`, `runWrite`, `runRead`) que lleva el literal "push" o "commit" entre sus
// argumentos, y devuelve cuántas llamadas de ejecución vio (para que el recorrido no sea vacuo).
func publishingCallViolations(label string, source []byte) (violations []string, calls int, err error) {
	file, err := parser.ParseFile(token.NewFileSet(), label, source, 0)
	if err != nil {
		return nil, 0, err
	}
	ast.Inspect(file, func(node ast.Node) bool {
		// Una lista de argumentos armada aparte (`args := []string{"push", ...}`) y pasada después con
		// `args...` tampoco puede llevar los subcomandos que solo ejecuta la publicación.
		if composite, isComposite := node.(*ast.CompositeLit); isComposite {
			if array, isArray := composite.Type.(*ast.ArrayType); isArray {
				if element, isIdent := array.Elt.(*ast.Ident); isIdent && element.Name == "string" {
					for _, item := range composite.Elts {
						literal, isLiteral := item.(*ast.BasicLit)
						if !isLiteral || literal.Kind != token.STRING {
							continue
						}
						if value, err := strconv.Unquote(literal.Value); err == nil && publishingLiterals[value] {
							violations = append(violations, label+" arma una orden de git con "+literal.Value)
						}
					}
				}
			}
			return true
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		var function string
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if pkg, isIdent := fun.X.(*ast.Ident); isIdent && pkg.Name == "gitcommon" {
				function = fun.Sel.Name
			} else {
				function = fun.Sel.Name // `g.runWrite(...)`
			}
		case *ast.Ident:
			function = fun.Name
		}
		if !runCalls[function] {
			return true
		}
		calls++
		for _, arg := range call.Args {
			ast.Inspect(arg, func(inner ast.Node) bool {
				literal, isLiteral := inner.(*ast.BasicLit)
				if !isLiteral || literal.Kind != token.STRING {
					return true
				}
				if value, err := strconv.Unquote(literal.Value); err == nil && publishingLiterals[value] {
					violations = append(violations, label+" pasa "+literal.Value+" a "+function)
				}
				return true
			})
		}
		return true
	})
	return violations, calls, nil
}

// REQ-27.11: el único paquete que importa `internal/sddpublish` es `internal/cli`, y dentro de él el
// único fichero que lo usa es `sdd_publish.go`; ni `internal/branch/**` ni `internal/docpaths` lo
// importan.
func TestIsolationOnlyTheCLIVerbImportsSddpublish(t *testing.T) {
	files := walkProduction(t)
	importers := map[string]bool{}
	for _, file := range files {
		violations, err := importViolations(file.label, file.content)
		if err != nil {
			t.Fatalf("analizar %s: %v", file.label, err)
		}
		for _, violation := range violations {
			t.Errorf("%s", violation)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file.label, file.content, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("analizar %s: %v", file.label, err)
		}
		if _, imports := sddpublishName(parsed); imports && file.dir() != "internal/sddpublish" {
			importers[file.label] = true
		}
	}
	// El recorrido no es vacuo: ve el verbo, su paquete y el dominio de ramas.
	seen := map[string]bool{}
	for _, file := range files {
		seen[file.label] = true
	}
	for _, want := range []string{cliPublishFile, "internal/sddpublish/publish.go", "internal/docpaths/docpaths.go", "internal/branch/engine/engine.go", "cmd/axiom/main.go"} {
		if !seen[want] {
			t.Errorf("el recorrido no vio %s", want)
		}
	}
	if len(importers) != 1 || !importers[cliPublishFile] {
		t.Errorf("importan internal/sddpublish %v, se esperaba solo %s", importers, cliPublishFile)
	}
}

// REQ-25.1 y REQ-27.11: ninguna ejecución de git de `internal/branch/**` lleva "push" ni "commit"; el
// puerto de publicación sí los lleva, y por eso la comprobación no es vacua.
func TestIsolationBranchPackageNeverPassesPushOrCommitToGit(t *testing.T) {
	files := walkProduction(t)
	branchCalls, publishCalls := 0, 0
	for _, file := range files {
		inBranch := strings.HasPrefix(file.label, "internal/branch/")
		if !inBranch && file.label != "internal/sddpublish/gitport.go" {
			continue
		}
		violations, calls, err := publishingCallViolations(file.label, file.content)
		if err != nil {
			t.Fatalf("analizar %s: %v", file.label, err)
		}
		if inBranch {
			branchCalls += calls
			for _, violation := range violations {
				t.Errorf("%s: solo internal/sddpublish ejecuta commit y push", violation)
			}
			continue
		}
		publishCalls += calls
		got := strings.Join(violations, "\n")
		for _, want := range []string{`"push"`, `"commit"`} {
			if !strings.Contains(got, want) {
				t.Errorf("el adaptador de la publicación debe pasar %s a git, y no se vio:\n%s", want, got)
			}
		}
	}
	if branchCalls < 10 || publishCalls < 10 {
		t.Errorf("el recorrido vio %d ejecuciones de git en internal/branch y %d en la publicación: es vacuo", branchCalls, publishCalls)
	}
}

// Las comprobaciones fallan si alguien rompe el aislamiento: se prueban con fuentes sintéticas.
func TestIsolationChecksFlagSyntheticViolations(t *testing.T) {
	importCases := []struct {
		name       string
		label      string
		source     string
		violations int
	}{
		{"otro paquete importa", "internal/branch/engine/x.go", "package engine\n\nimport \"" + sddpublishImport + "\"\n\nvar _ = sddpublish.Schema\n", 3},
		{"otro fichero de cli lo usa", "internal/cli/sdd_gate.go", "package cli\n\nimport \"" + sddpublishImport + "\"\n\nvar _ = sddpublish.Schema\n", 2},
		{"alias", "internal/cli/otro.go", "package cli\n\nimport pub \"" + sddpublishImport + "\"\n\nvar _ = pub.Schema\n", 2},
		{"el verbo", cliPublishFile, "package cli\n\nimport \"" + sddpublishImport + "\"\n\nvar _ = sddpublish.Schema\n", 0},
		{"su propio paquete", "internal/sddpublish/publish.go", "package sddpublish\n\nvar _ = 1\n", 0},
		{"sin importarlo", "internal/cli/sdd_gate.go", "package cli\n\nvar _ = 1\n", 0},
	}
	for _, tc := range importCases {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := importViolations(tc.label, []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(violations) != tc.violations {
				t.Errorf("%d violaciones %v, se esperaban %d", len(violations), violations, tc.violations)
			}
		})
	}

	callCases := []struct {
		name       string
		source     string
		violations int
		calls      int
	}{
		{"push a gitcommon.Run", "package gitcli\n\nfunc f() { gitcommon.Run(ctx, dir, \"push\", \"origin\") }\n", 1, 1},
		{"commit a RunWithTimeout", "package gitcli\n\nfunc f() { gitcommon.RunWithTimeout(ctx, dir, t, \"commit\", \"-m\", m) }\n", 1, 1},
		{"commit a runWrite", "package engine\n\nfunc f() { g.runWrite(ctx, dir, \"commit\") }\n", 1, 1},
		{"push anidado en los argumentos", "package gitcli\n\nfunc f() { runRead(ctx, dir, append([]string{\"push\"}, x...)...) }\n", 2, 1},
		{"una lectura", "package gitcli\n\nfunc f() { gitcommon.Run(ctx, dir, \"rev-parse\", \"HEAD\") }\n", 0, 1},
		{"una lista de argumentos armada aparte", "package gitcli\n\nfunc f() { args := []string{\"commit\", \"-m\", m}; gitcommon.Run(ctx, dir, args...) }\n", 1, 1},
		{"un literal que no es una orden", "package gitcli\n\nvar nombres = map[string]string{\"push\": \"subir\"}\n", 0, 0},
	}
	for _, tc := range callCases {
		t.Run(tc.name, func(t *testing.T) {
			violations, calls, err := publishingCallViolations("internal/branch/gitcli/x.go", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(violations) != tc.violations || calls != tc.calls {
				t.Errorf("%d violaciones %v y %d llamadas, se esperaban %d y %d", len(violations), violations, calls, tc.violations, tc.calls)
			}
		})
	}
}
