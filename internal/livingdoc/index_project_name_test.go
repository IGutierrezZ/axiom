package livingdoc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// TestRenderIndexMarkdownUsesTheProjectName cubre m11: el nombre sustituye solo
// al literal «Axiom» del título y de «Proyecto», y sin nombre la salida es la de
// siempre.
func TestRenderIndexMarkdownUsesTheProjectName(t *testing.T) {
	indexer := NewIndexer()
	base := &LivingCatalog{TotalRequirements: 3, TotalScenarios: 4}

	named := *base
	named.ProjectName = "Mi Proyecto"
	got := indexer.RenderIndexMarkdown(&named)
	for _, want := range []string{
		"# Catálogo Maestro de Especificaciones Vivas — Mi Proyecto\n",
		"> **Proyecto:** Mi Proyecto (Spec-Driven Development Platform)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("la cabecera debe contener %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Axiom") {
		t.Errorf("con nombre de proyecto no debe quedar el literal Axiom:\n%s", got)
	}

	// Solo cambian el título y la línea «Proyecto»: el resto del documento es el
	// mismo que el del nombre por defecto.
	defaultRender := indexer.RenderIndexMarkdown(base)
	if !strings.Contains(defaultRender, "# Catálogo Maestro de Especificaciones Vivas — Axiom\n") ||
		!strings.Contains(defaultRender, "> **Proyecto:** Axiom (Spec-Driven Development Platform)\n") {
		t.Fatalf("sin ProjectName la cabecera conserva Axiom:\n%s", defaultRender)
	}
	if strings.ReplaceAll(defaultRender, "Axiom", "Mi Proyecto") != got {
		t.Errorf("el nombre solo debe sustituir el literal Axiom:\n--- por defecto\n%s\n--- con nombre\n%s", defaultRender, got)
	}
}

// TestRenderIndexMarkdownIsUnchangedForThisRepository: este repositorio declara
// workspace.name: Axiom, de modo que su INDEX.md no cambia con el nombre del
// proyecto tomado de la configuración.
func TestRenderIndexMarkdownIsUnchangedForThisRepository(t *testing.T) {
	loc, err := workspace.Locate(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Locate sobre la raíz del repositorio: %v", err)
	}
	if name := projectNameOf(loc); name != "Axiom" {
		t.Fatalf("projectNameOf(raíz del repositorio) = %q, se esperaba Axiom (workspace.name)", name)
	}
	indexer := NewIndexer()
	catalog := &LivingCatalog{TotalRequirements: 1}
	withoutName := indexer.RenderIndexMarkdown(catalog)
	named := *catalog
	named.ProjectName = projectNameOf(loc)
	if got := indexer.RenderIndexMarkdown(&named); got != withoutName {
		t.Errorf("el INDEX.md de este repositorio no debe cambiar:\n--- sin nombre\n%s\n--- con nombre\n%s", withoutName, got)
	}
}

// TestServiceTakesTheProjectNameFromTheWorkspace cubre la procedencia del nombre:
// workspace.name y, sin él, el directorio de la carpeta maestra.
func TestServiceTakesTheProjectNameFromTheWorkspace(t *testing.T) {
	t.Run("workspace.name", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "")
		writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "dominio")
		svc := NewService(ws.Master, nil, nil)

		catalog, err := svc.GetCatalog(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		// workspacetest.Embedded declara workspace.name: ejemplo-embebido.
		if catalog.ProjectName != "ejemplo-embebido" {
			t.Errorf("ProjectName = %q, se esperaba ejemplo-embebido", catalog.ProjectName)
		}
		if _, err := svc.Sync(context.Background()); err != nil {
			t.Fatal(err)
		}
		index := readFile(t, filepath.Join(ws.Home, "INDEX.md"))
		if !strings.Contains(index, "Especificaciones Vivas — ejemplo-embebido\n") ||
			!strings.Contains(index, "> **Proyecto:** ejemplo-embebido (Spec-Driven") {
			t.Errorf("INDEX.md debe llevar el nombre del proyecto:\n%s", index)
		}
	})

	t.Run("respaldo: nombre de la carpeta maestra", func(t *testing.T) {
		master := filepath.Join(t.TempDir(), "mi-proyecto")
		writeLivingSpec(t, filepath.Join(master, "openspec", "specs"), "dominio")
		svc := NewService(master, nil, nil)

		catalog, err := svc.GetCatalog(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if catalog.ProjectName != "mi-proyecto" {
			t.Errorf("ProjectName = %q, se esperaba mi-proyecto", catalog.ProjectName)
		}
		if _, err := svc.Sync(context.Background()); err != nil {
			t.Fatal(err)
		}
		index := readFile(t, filepath.Join(master, "openspec", "INDEX.md"))
		if !strings.Contains(index, "> **Proyecto:** mi-proyecto (Spec-Driven") {
			t.Errorf("INDEX.md debe llevar el nombre de la carpeta maestra:\n%s", index)
		}
	})

	t.Run("canónico: la carpeta maestra no es el repositorio de especificaciones", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		svc := NewService(filepath.Join(ws.Backend, "src"), nil, nil)

		catalog, err := svc.GetCatalog(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		// workspacetest.Canonical declara workspace.name: ejemplo-canonico.
		if catalog.ProjectName != "ejemplo-canonico" {
			t.Errorf("ProjectName = %q, se esperaba ejemplo-canonico", catalog.ProjectName)
		}
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return string(data)
}
