package livingdoc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const sampleSpec = `# Especificación de ejemplo
### Requirement: Regla de ejemplo (REQ-01)
#### Scenario: Escenario de ejemplo
- DADO un sistema en estado inicial
`

// writeLivingSpec escribe una especificación viva mínima en <specsDir>/<domain>.
func writeLivingSpec(t *testing.T, specsDir, domain string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(specsDir, domain, "spec.md"), sampleSpec)
}

func catalogDomains(catalog *LivingCatalog) []string {
	domains := make([]string, 0, len(catalog.Specs))
	for _, spec := range catalog.Specs {
		domains = append(domains, spec.Domain)
	}
	return domains
}

// TestServiceReadsThePlanningHomeRegardlessOfSpecsRepository cubre REQ-1.1 y
// REQ-1.2: la raíz de las especificaciones y el índice salen del planning home.
func TestServiceReadsThePlanningHomeRegardlessOfSpecsRepository(t *testing.T) {
	for _, specsRepository := range []string{"", ".", "openspec", "specs"} {
		t.Run("embebido con specs_repository="+specsRepository, func(t *testing.T) {
			ws := workspacetest.Embedded(t, specsRepository)
			svc := NewService(ws.Master, nil, nil)

			specsRoot, err := svc.GetSpecsRoot()
			if err != nil {
				t.Fatal(err)
			}
			indexPath, err := svc.GetIndexPath()
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(ws.Home, "specs"); specsRoot != want {
				t.Errorf("GetSpecsRoot() = %q, se esperaba %q", specsRoot, want)
			}
			if want := filepath.Join(ws.Home, "INDEX.md"); indexPath != want {
				t.Errorf("GetIndexPath() = %q, se esperaba %q", indexPath, want)
			}
		})
	}

	t.Run("canónico desde la carpeta maestra", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "workspace-topology")
		svc := NewService(ws.Master, nil, nil)

		catalog, err := svc.GetCatalog(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got := catalogDomains(catalog); len(got) != 1 || got[0] != "workspace-topology" {
			t.Fatalf("dominios = %v, se esperaba [workspace-topology]", got)
		}
		if len(catalog.Warnings) != 0 {
			t.Errorf("el canónico no emite avisos: %v", catalog.Warnings)
		}

		report, err := svc.Sync(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(ws.Home, "INDEX.md"); report.IndexPath != want {
			t.Errorf("IndexPath = %q, se esperaba %q", report.IndexPath, want)
		}
		if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
			t.Errorf("INDEX.md debe escribirse en el planning home: %v", err)
		}
		if _, err := os.Stat(filepath.Join(ws.Master, "openspec")); !os.IsNotExist(err) {
			t.Errorf("no debe crearse openspec/ en la carpeta maestra (err=%v)", err)
		}
	})

	t.Run("canónico desde un repositorio de código", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "workspace-topology")
		svc := NewService(filepath.Join(ws.Backend, "src"), nil, nil)

		catalog, err := svc.GetCatalog(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got := catalogDomains(catalog); len(got) != 1 {
			t.Fatalf("dominios = %v, se esperaba el catálogo del planning home", got)
		}
		if _, raw, err := svc.GetSpecDetail("workspace-topology"); err != nil || raw == "" {
			t.Fatalf("GetSpecDetail desde el repositorio de código: %v", err)
		}
	})
}

// TestServiceWarnsAboutADivergentLegacyLocation cubre el escenario «Ubicación
// antigua divergente emite un aviso» de REQ-1.1 y REQ-24.10 (Q14).
func TestServiceWarnsAboutADivergentLegacyLocation(t *testing.T) {
	ws := workspacetest.Embedded(t, "specs")
	legacySpecs := filepath.Join(ws.Master, "specs", "specs")
	currentSpecs := filepath.Join(ws.Home, "specs")
	writeLivingSpec(t, legacySpecs, "antiguo")
	writeLivingSpec(t, currentSpecs, "vigente")
	svc := NewService(ws.Master, nil, nil)

	catalog, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := catalogDomains(catalog); len(got) != 1 || got[0] != "vigente" {
		t.Fatalf("dominios = %v, se esperaba solo [vigente]", got)
	}
	if len(catalog.Warnings) != 1 {
		t.Fatalf("Warnings = %v, se esperaba el aviso Q14", catalog.Warnings)
	}
	for _, want := range []string{legacySpecs, currentSpecs} {
		if !strings.Contains(catalog.Warnings[0], want) {
			t.Errorf("el aviso debe nombrar %q: %q", want, catalog.Warnings[0])
		}
	}

	report, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.SpecsCount != 1 || len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], legacySpecs) {
		t.Errorf("Sync debe indexar solo el planning home y repetir el aviso Q14: %+v", report)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
		t.Errorf("INDEX.md debe escribirse en el planning home: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Master, "specs", "INDEX.md")); !os.IsNotExist(err) {
		t.Errorf("no debe escribirse un INDEX.md en la ubicación antigua (err=%v)", err)
	}
}

// TestServiceResolvesThePlanningHomeOnEveryCall comprueba que el servicio no
// cachea la raíz: si el proyecto cambia entre dos llamadas, la segunda lee el
// nuevo (D-07, alternativa descartada «cachear Location en el constructor»).
func TestServiceResolvesThePlanningHomeOnEveryCall(t *testing.T) {
	const roles = "roles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	master := t.TempDir()
	for _, repo := range []string{"repo-a", "repo-b"} {
		workspacetest.WriteFile(t, filepath.Join(master, repo, "axiom.yaml"),
			"workspace:\n  name: "+repo+"\n  topology: multirepo\n  specs_repository: "+repo+"\n"+roles)
	}
	writeLivingSpec(t, filepath.Join(master, "repo-a", "openspec", "specs"), "dominio-a")
	writeLivingSpec(t, filepath.Join(master, "repo-b", "openspec", "specs"), "dominio-b")

	pointer := filepath.Join(master, ".axiom-workspace")
	workspacetest.WriteFile(t, pointer, "config: repo-a/axiom.yaml\n")
	svc := NewService(master, nil, nil)

	first, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	workspacetest.WriteFile(t, pointer, "config: repo-b/axiom.yaml\n")
	second, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := catalogDomains(first); len(got) != 1 || got[0] != "dominio-a" {
		t.Errorf("primera llamada = %v, se esperaba [dominio-a]", got)
	}
	if got := catalogDomains(second); len(got) != 1 || got[0] != "dominio-b" {
		t.Errorf("segunda llamada = %v, se esperaba [dominio-b]: el servicio no debe cachear la raíz", got)
	}
}

// TestServicePropagatesLocateFailures: si la resolución se rechaza, todos los
// métodos públicos devuelven ese error sin escribir nada.
func TestServicePropagatesLocateFailures(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	svc := NewService(ws.Master, nil, nil)
	ctx := context.Background()

	calls := map[string]func() error{
		"GetCatalog":    func() error { _, err := svc.GetCatalog(ctx); return err },
		"GetSpecDetail": func() error { _, _, err := svc.GetSpecDetail("x"); return err },
		"Sync":          func() error { _, err := svc.Sync(ctx); return err },
		"ColdStart":     func() error { _, err := svc.ColdStart("inc-99-ejemplo", ""); return err },
		"GetSpecsRoot":  func() error { _, err := svc.GetSpecsRoot(); return err },
		"GetIndexPath":  func() error { _, err := svc.GetIndexPath(); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var locateErr *workspace.LocateError
			err := call()
			if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocatePointerBroken {
				t.Fatalf("err = %v, se esperaba el *LocateError pointer_broken", err)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "specs")); !os.IsNotExist(err) {
		t.Errorf("un rechazo de Locate no debe crear nada (err=%v)", err)
	}
}

// TestColdStartUsesTheResolvedWorkspaceRoot: el sintetizador recibe el
// workspaceRoot ya resuelto, también cuando el servicio parte de la carpeta
// maestra del canónico.
func TestColdStartUsesTheResolvedWorkspaceRoot(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "inc-99-ejemplo", "spec.md"), sampleSpec)
	svc := NewService(ws.Master, nil, nil)

	entry, err := svc.ColdStart("inc-99-ejemplo", "ejemplo")
	if err != nil {
		t.Fatalf("ColdStart: %v", err)
	}
	// El indexador normaliza FilePath con barras, también en Windows.
	if want := filepath.ToSlash(filepath.Join(ws.Home, "specs", "ejemplo", "spec.md")); filepath.ToSlash(entry.FilePath) != want {
		t.Errorf("FilePath = %q, se esperaba %q", entry.FilePath, want)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
		t.Errorf("ColdStart debe sincronizar el INDEX.md del planning home: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Master, "openspec")); !os.IsNotExist(err) {
		t.Errorf("no debe crearse openspec/ en la carpeta maestra (err=%v)", err)
	}
}
