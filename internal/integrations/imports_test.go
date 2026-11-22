package integrations

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/IGutierrezZ/axiom/v3/"

// repoRoot es la raíz del módulo: este paquete vive en internal/integrations.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("raíz del módulo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("no se encuentra go.mod en %s: %v", root, err)
	}
	return root
}

// nonTestImports devuelve, por fichero .go que no es de prueba bajo dir (recursivo), sus
// importaciones.
func nonTestImports(t *testing.T, dir string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			out[path] = append(out[path], p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", dir, err)
	}
	return out
}

// importers devuelve los ficheros bajo dir que importan alguna ruta que cumpla match.
func importers(t *testing.T, dir string, match func(string) bool) []string {
	t.Helper()
	var found []string
	for file, imports := range nonTestImports(t, dir) {
		for _, p := range imports {
			if match(p) {
				found = append(found, file)
				break
			}
		}
	}
	sort.Strings(found)
	return found
}

func pkg(rel string) func(string) bool {
	full := modulePath + rel
	return func(p string) bool { return p == full || strings.HasPrefix(p, full+"/") }
}

// pkgExact coincide solo con el paquete exacto (sin subpaquetes).
func pkgExact(rel string) func(string) bool {
	full := modulePath + rel
	return func(p string) bool { return p == full }
}

func assertNoImporters(t *testing.T, dir string, match func(string) bool, rule string) {
	t.Helper()
	if found := importers(t, dir, match); len(found) != 0 {
		t.Errorf("regla de importaciones incumplida (%s): %s", rule, strings.Join(found, ", "))
	}
}

func TestImportsDirection(t *testing.T) {
	root := repoRoot(t)

	t.Run("manifest solo importa la biblioteca estándar", func(t *testing.T) {
		assertNoImporters(t, filepath.Join(root, "internal", "integrations", "manifest"),
			func(p string) bool { return strings.Contains(strings.SplitN(p, "/", 2)[0], ".") },
			"manifest es una hoja pura")
	})

	t.Run("internal/integrations no importa internal/cli", func(t *testing.T) {
		assertNoImporters(t, filepath.Join(root, "internal", "integrations"), pkg("internal/cli"),
			"el núcleo no depende de la capa de verbos")
	})

	t.Run("internal/branch no importa internal/integrations", func(t *testing.T) {
		assertNoImporters(t, filepath.Join(root, "internal", "branch"), pkg("internal/integrations"),
			"el motor de ramas no conoce las integraciones")
	})

	t.Run("internal/workspace no importa internal/integrations salvo manifest", func(t *testing.T) {
		assertNoImporters(t, filepath.Join(root, "internal", "workspace"), func(p string) bool {
			return pkg("internal/integrations")(p) && !pkgExact("internal/integrations/manifest")(p)
		}, "workspace solo puede depender de la hoja manifest")
	})

	t.Run("solo internal/integrations/secrets importa go-keyring", func(t *testing.T) {
		isKeyring := func(p string) bool {
			return p == "github.com/zalando/go-keyring" || strings.HasPrefix(p, "github.com/zalando/go-keyring/")
		}
		for _, top := range []string{"internal", "cmd"} {
			for _, user := range importers(t, filepath.Join(root, top), isKeyring) {
				if !strings.Contains(filepath.ToSlash(user), "/internal/integrations/secrets/") {
					t.Errorf("%s importa go-keyring: solo internal/integrations/secrets puede hacerlo", user)
				}
			}
		}
	})

	t.Run("secrets no importa internal/integrations ni workspace", func(t *testing.T) {
		dir := filepath.Join(root, "internal", "integrations", "secrets")
		assertNoImporters(t, dir, pkgExact("internal/integrations"), "secrets no importa el núcleo")
		assertNoImporters(t, dir, pkg("internal/workspace"), "secrets no importa workspace")
	})
}

// TestImportsCheckerDetectsViolations fija que el comprobador no es vacuo: sobre un árbol con
// una importación prohibida la detecta.
func TestImportsCheckerDetectsViolations(t *testing.T) {
	dir := t.TempDir()
	bad := "package x\n\nimport _ \"" + modulePath + "internal/cli\"\n"
	ok := "package y\n\nimport _ \"fmt\"\n"
	test := "package x\n\nimport _ \"" + modulePath + "internal/cli\"\n"
	for name, src := range map[string]string{"bad.go": bad, "ok.go": ok, "bad_test.go": test} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	found := importers(t, dir, pkg("internal/cli"))
	if len(found) != 1 || filepath.Base(found[0]) != "bad.go" {
		t.Fatalf("debe detectar solo bad.go (los ficheros de prueba no cuentan), obtuvo %v", found)
	}
	if got := importers(t, dir, pkgExact("internal/clix")); len(got) != 0 {
		t.Fatalf("pkgExact no debe coincidir con otro paquete: %v", got)
	}
}

// usersOf devuelve los ficheros que no son de prueba bajo dir que usan el identificador
// <pkgName>.<symbol> (por ejemplo kickoff.RefuseArchivedWriteExceptLocalState).
func usersOf(t *testing.T, dir, pkgName, symbol string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return err
		}
		used := false
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == symbol {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkgName {
					used = true
				}
			}
			return !used
		})
		if used {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", dir, err)
	}
	sort.Strings(found)
	return found
}

// TestOnlyIntegrationsUsesTheArchivedWriteGuard fija la excepción acotada de MODIFIED REQ-21.18: la
// guarda hermana que autoriza la cola bajo archive/ solo la llama internal/integrations (D-18).
func TestOnlyIntegrationsUsesTheArchivedWriteGuard(t *testing.T) {
	root := repoRoot(t)
	allowed := filepath.Join(root, "internal", "integrations") + string(filepath.Separator)
	allowedUsers := 0
	for _, top := range []string{"internal", "cmd"} {
		for _, user := range usersOf(t, filepath.Join(root, top), "kickoff", "RefuseArchivedWriteExceptLocalState") {
			if !strings.HasPrefix(user, allowed) {
				t.Errorf("%s usa kickoff.RefuseArchivedWriteExceptLocalState: solo internal/integrations puede hacerlo", user)
				continue
			}
			allowedUsers++
		}
	}
	// Desde la cola local, internal/integrations es el único importador: la guarda no queda sin uso.
	if allowedUsers == 0 {
		t.Error("internal/integrations debe usar kickoff.RefuseArchivedWriteExceptLocalState antes de cada escritura en una carpeta del cambio")
	}
}

func TestUsersOfDetectsTheSymbol(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"uses.go":      "package x\n\nimport \"example.test/kickoff\"\n\nvar _ = kickoff.RefuseArchivedWriteExceptLocalState\n",
		"other.go":     "package x\n\nimport \"example.test/kickoff\"\n\nvar _ = kickoff.RefuseArchivedRoot\n",
		"uses_test.go": "package x\n\nimport \"example.test/kickoff\"\n\nvar _ = kickoff.RefuseArchivedWriteExceptLocalState\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	found := usersOf(t, dir, "kickoff", "RefuseArchivedWriteExceptLocalState")
	if len(found) != 1 || filepath.Base(found[0]) != "uses.go" {
		t.Fatalf("debe detectar solo uses.go, obtuvo %v", found)
	}
}
