package techcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func scopeTree(t *testing.T) (Scope, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "especificaciones")
	for _, dir := range []string{
		filepath.Join(root, "context"), filepath.Join(root, "referencia", "legado"),
		filepath.Join(base, "repo-backend", "docs"), filepath.Join(base, "ajeno"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{
		filepath.Join(root, "01_Requisitos.md"), filepath.Join(root, "context", "ARQUITECTURA.md"),
		filepath.Join(root, "context", "NOTAS.txt"), filepath.Join(root, "referencia", "legado", "VIEJO.md"),
		filepath.Join(base, "repo-backend", "docs", "API.md"), filepath.Join(base, "ajeno", "X.md"),
	} {
		if err := os.WriteFile(file, []byte("# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Scope{
		WorkspaceRoot: root,
		Maintained:    []string{"context/", "../repo-backend/docs/", "NOTAS-RAIZ.md"},
		Reference:     []string{"referencia/"},
	}, base
}

func scopeEntry(document string) Entry {
	return Entry{ID: "C4", Document: document, Line: 12}
}

func TestScopeAcceptsMaintainedDocuments(t *testing.T) {
	scope, _ := scopeTree(t)
	for _, document := range []string{
		"context/ARQUITECTURA.md",            // existente
		"context/PAGOS.md",                   // por crear (REQ-24.55)
		"context/nuevo/dir/SERVICIO.md",      // por crear, con directorios intermedios
		"../repo-backend/docs/API.md",        // repositorio hermano con `..`
		"../repo-backend/docs/NUEVO.md",      // por crear en el repositorio hermano
		"context/../context/ARQUITECTURA.md", // la ruta se limpia antes de decidir
	} {
		if err := scope.CheckDocument(scopeEntry(document)); err != nil {
			t.Errorf("%s debía estar permitido: %v", document, err)
		}
	}
}

func TestScopeRejectsDocumentsOutsideTheMaintainedPaths(t *testing.T) {
	scope, _ := scopeTree(t)
	tests := []struct {
		document string
		want     []string
	}{
		{document: "referencia/legado/VIEJO.md", want: []string{"C4", "línea 12", "ruta de referencia", "modo reference"}},
		{document: "referencia/NUEVO.md", want: []string{"modo reference"}},
		{document: "../ajeno/X.md", want: []string{"no está bajo ninguna ruta mantenida", `"context/"`, `"../repo-backend/docs/"`}},
		{document: "docs/otro.md", want: []string{"no está bajo ninguna ruta mantenida"}},
		{document: "context-ajeno/X.md", want: []string{"no está bajo ninguna ruta mantenida"}}, // prefijo textual, otro directorio
	}
	for _, tt := range tests {
		err := scope.CheckDocument(scopeEntry(tt.document))
		if err == nil {
			t.Errorf("%s debía rechazarse", tt.document)
			continue
		}
		for _, fragment := range tt.want {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("%s: el error %q debe contener %q", tt.document, err, fragment)
			}
		}
	}
}

func TestScopeDecidesByIdentityNotByText(t *testing.T) {
	scope, _ := scopeTree(t)
	// Una ruta mantenida declarada como un fichero concreto solo permite ese fichero.
	if err := os.WriteFile(scope.Resolve("NOTAS-RAIZ.md"), []byte("# n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := scope.CheckDocument(scopeEntry("NOTAS-RAIZ.md")); err != nil {
		t.Errorf("el fichero mantenido debía permitirse: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("NOTAS-RAIZ.md.bak")); err == nil {
		t.Error("un fichero con el mismo prefijo no es el fichero mantenido")
	}
	// Con una ruta de referencia que existe, lo que resuelve a ella es referencia
	// aunque se escriba con otra grafía.
	if err := scope.CheckDocument(scopeEntry("context/../referencia/legado/VIEJO.md")); err == nil || !strings.Contains(err.Error(), "referencia") {
		t.Errorf("la grafía con `..` sigue siendo referencia: %v", err)
	}
}

func TestScopeRejectsASymlinkThatLeavesTheMaintainedPath(t *testing.T) {
	scope, base := scopeTree(t)
	link := filepath.Join(scope.WorkspaceRoot, "context", "salida")
	if err := os.Symlink(filepath.Join(base, "ajeno"), link); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos en este sistema: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("context/salida/X.md")); err == nil {
		t.Fatal("un enlace simbólico bajo la ruta mantenida que sale de ella no debe permitir el documento")
	}
	if err := scope.CheckDocument(scopeEntry("context/salida/NUEVO.md")); err == nil {
		t.Fatal("tampoco un documento por crear tras el enlace")
	}
}

func TestScopeComparesAnUndeclaredPathLexically(t *testing.T) {
	scope, _ := scopeTree(t)
	// La ruta mantenida aún no existe: no tiene identidad, así que se compara de
	// forma léxica y el primer documento de un directorio nuevo se permite.
	scope.Maintained = []string{"contexto-nuevo/"}
	if err := scope.CheckDocument(scopeEntry("contexto-nuevo/PRIMERO.md")); err != nil {
		t.Errorf("el primer documento de una ruta mantenida inexistente debía permitirse: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("contexto-nuevo-otro/X.md")); err == nil {
		t.Error("un directorio hermano con el mismo prefijo no está bajo la ruta mantenida")
	}
}

func TestScopeWithoutMaintainedPaths(t *testing.T) {
	scope, _ := scopeTree(t)
	scope.Maintained = nil
	err := scope.CheckDocument(scopeEntry("context/ARQUITECTURA.md"))
	if err == nil || !strings.Contains(err.Error(), "(ninguna)") {
		t.Fatalf("sin rutas mantenidas se rechaza nombrando que no hay ninguna: %v", err)
	}
}

func locationWith(root string, technical *workspace.TechnicalContextConfig) workspace.Location {
	return workspace.Location{
		WorkspaceRoot: root,
		Declarations:  workspace.Declarations{TechnicalContext: technical},
	}
}

func TestScopeOfReadsTheDeclaredPathsWithSlashes(t *testing.T) {
	loc := locationWith(`C:\ws`, &workspace.TechnicalContextConfig{
		Maintained: &workspace.ContextMode{Entry: "context/A.md", Paths: []string{` context\ `, "../repo/docs/"}},
		Reference:  &workspace.ContextMode{Paths: []string{"referencia/"}},
	})
	scope := ScopeOf(loc)
	if scope.WorkspaceRoot != `C:\ws` {
		t.Errorf("WorkspaceRoot = %q", scope.WorkspaceRoot)
	}
	if got := strings.Join(scope.Maintained, "|"); got != "context/|../repo/docs/" {
		t.Errorf("Maintained = %q, quería las rutas con `/` y sin espacios", got)
	}
	if got := strings.Join(scope.Reference, "|"); got != "referencia/" {
		t.Errorf("Reference = %q", got)
	}
	if empty := ScopeOf(locationWith("/x", nil)); len(empty.Maintained) != 0 || len(empty.Reference) != 0 {
		t.Errorf("sin technical_context el ámbito debe estar vacío: %+v", empty)
	}
}

// Amenaza «selección de repositorio git» (REQ-30.15, REQ-30.29): las rutas
// declaradas, relativas, absolutas o hacia un repositorio hermano, se resuelven
// contra workspaceRoot y nunca contra el directorio de trabajo del proceso.
func TestClassifyResolvesDeclaredPathsAgainstTheWorkspaceRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "especificaciones")
	sibling := filepath.Join(base, "repo-hermano")
	absolute := filepath.Join(base, "absoluta")
	for _, dir := range []string{filepath.Join(root, "context"), filepath.Join(root, "ref"), filepath.Join(sibling, "docs"), absolute, filepath.Join(base, "otro")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	loc := locationWith(root, &workspace.TechnicalContextConfig{
		Maintained: &workspace.ContextMode{Paths: []string{"context/", "../repo-hermano/docs/", filepath.ToSlash(absolute)}},
		Reference:  &workspace.ContextMode{Paths: []string{"ref/"}},
	})
	scope := ScopeOf(loc)

	// El directorio del proceso es otro: no puede influir.
	t.Chdir(filepath.Join(base, "otro"))

	tests := []struct {
		name string
		path string
		want PathMode
	}{
		{"relativa mantenida", filepath.Join(root, "context", "A.md"), Maintained},
		{"relativa mantenida por crear", filepath.Join(root, "context", "nuevo", "B.md"), Maintained},
		{"repositorio hermano con ..", filepath.Join(sibling, "docs", "API.md"), Maintained},
		{"absoluta declarada", filepath.Join(absolute, "X.md"), Maintained},
		{"referencia", filepath.Join(root, "ref", "Y.md"), Reference},
		{"fuera", filepath.Join(base, "otro", "Z.md"), Outside},
		{"prefijo textual de otro directorio", filepath.Join(root, "context-ajeno", "W.md"), Outside},
		{"fuera del hermano declarado", filepath.Join(sibling, "src", "main.go"), Outside},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scope.Classify(tt.path); got != tt.want {
				t.Errorf("Classify(%s) = %q, quería %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestClassifyGivesReferencePrecedenceOverMaintained(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "context", "legado"), 0o755); err != nil {
		t.Fatal(err)
	}
	scope := Scope{WorkspaceRoot: root, Maintained: []string{"context/"}, Reference: []string{"context/legado/"}}
	if got := scope.Classify(filepath.Join(root, "context", "legado", "V.md")); got != Reference {
		t.Errorf("una ruta bajo ambas es de referencia (nunca se modifica): %q", got)
	}
	if got := scope.Classify(filepath.Join(root, "context", "A.md")); got != Maintained {
		t.Errorf("Classify = %q, quería %q", got, Maintained)
	}
}
