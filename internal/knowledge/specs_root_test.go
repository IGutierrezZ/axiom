package knowledge

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

// writeGoModule escribe un paquete Go mínimo en <root>/internal/<name>.
func writeGoModule(t *testing.T, root, name, comment string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(root, "internal", name, name+".go"),
		"package "+name+"\n// "+comment+"\n")
}

// writeLegacySpec crea la ubicación antigua <raíz>/specs/specs/antiguo/spec.md de
// un proyecto embebido con specs_repository: specs.
func writeLegacySpec(t *testing.T, root string) string {
	t.Helper()
	legacy := filepath.Join(root, "specs", "specs")
	workspacetest.WriteFile(t, filepath.Join(legacy, "antiguo", "spec.md"), "# Antiguo\n")
	return legacy
}

// TestDefaultSpecsRootTakesThePlanningHome cubre REQ-24.10: con SpecsRoot vacío,
// knowledge toma el planning home de Locate desde la raíz de código analizada.
func TestDefaultSpecsRootTakesThePlanningHome(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	embedded := workspacetest.Embedded(t, "")

	tests := []struct {
		name  string
		start string
		want  string
	}{
		{"canónico desde la carpeta maestra", canonical.Master, canonical.Home},
		{"canónico desde un repositorio de código", canonical.Backend, canonical.Home},
		{"canónico desde un subdirectorio de código", filepath.Join(canonical.Backend, "src"), canonical.Home},
		{"embebido", embedded.Master, embedded.Home},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, notes, err := defaultSpecsRoot(tt.start)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("defaultSpecsRoot = %q, se esperaba el planning home %q", got, tt.want)
			}
			if len(notes) != 0 {
				t.Errorf("sin divergencia no hay avisos: %v", notes)
			}
		})
	}
}

func TestSweepSeedsTheSpecsOfThePlanningHome(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start func(ws workspacetest.Workspace) string
	}{
		{"desde la carpeta maestra", func(ws workspacetest.Workspace) string { return ws.Master }},
		{"desde un repositorio de código", func(ws workspacetest.Workspace) string { return ws.Backend }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ws := workspacetest.Canonical(t)
			writeGoModule(t, ws.Backend, "billing", "regla")
			start := tt.start(ws)

			res, err := RunSweep(context.Background(), SweepOptions{WorkspaceRoot: start, Headless: true})
			if err != nil {
				t.Fatalf("RunSweep: %v", err)
			}
			if len(res.CreatedSpecs) == 0 {
				t.Fatal("se esperaba al menos una especificación sembrada")
			}
			for _, created := range res.CreatedSpecs {
				if !strings.HasPrefix(created, filepath.Join(ws.Home, "specs")+string(filepath.Separator)) {
					t.Errorf("la especificación %q debe sembrarse bajo el planning home %q", created, ws.Home)
				}
			}
			if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
				t.Errorf("INDEX.md debe escribirse en el planning home: %v", err)
			}
			for _, wrong := range []string{ws.Master, ws.Backend} {
				if _, err := os.Stat(filepath.Join(wrong, "openspec")); !os.IsNotExist(err) {
					t.Errorf("no debe crearse openspec/ en %s (err=%v)", wrong, err)
				}
			}
			if len(res.Notes) != 0 {
				t.Errorf("sin divergencia no hay notes: %v", res.Notes)
			}
		})
	}
}

func TestQueryResolvesFromTheSpecsOfThePlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "specs", "billing", "spec.md"),
		"# Facturación\n### Requirement: Regla de facturación (REQ-1)\nEl sistema DEBE aplicar la regla_facturacion_anual a cada cliente.\n")

	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: ws.Backend,
		Question:      "¿Cuál es la regla_facturacion_anual?",
	})
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if !res.ResolvedFromSpec {
		t.Fatalf("la respuesta debe resolverse de la especificación del planning home: %+v", res)
	}
	if len(res.Evidences) == 0 || !strings.Contains(res.Evidences[0].File, "billing") {
		t.Errorf("la evidencia debe ser la especificación del planning home: %+v", res.Evidences)
	}
}

func TestGenerateCrawlPlanTakesTheSpecsRootFromThePlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t)
	writeGoModule(t, ws.Backend, "billing", "regla")

	job, err := GenerateCrawlPlan(context.Background(), CrawlPlanOptions{
		WorkspaceRoot: ws.Backend,
		JobPath:       filepath.Join(t.TempDir(), "crawl-job.json"),
		Force:         true,
	})
	if err != nil {
		t.Fatalf("GenerateCrawlPlan: %v", err)
	}
	if job.SpecsRoot != ws.Home {
		t.Errorf("SpecsRoot del job = %q, se esperaba el planning home %q", job.SpecsRoot, ws.Home)
	}
	if job.WorkspaceRoot != ws.Backend {
		t.Errorf("WorkspaceRoot del job = %q, sigue siendo la raíz de código %q", job.WorkspaceRoot, ws.Backend)
	}
}

// TestSweepAndQueryEmitTheLegacyLocationNote cubre el aviso Q14 en las notes de
// sweep y de query, y que no aparece sin divergencia.
func TestSweepAndQueryEmitTheLegacyLocationNote(t *testing.T) {
	ws := workspacetest.Embedded(t, "specs")
	legacy := writeLegacySpec(t, ws.Master)
	writeGoModule(t, ws.Master, "billing", "regla_facturacion_anual activa")

	sweep, err := RunSweep(context.Background(), SweepOptions{WorkspaceRoot: ws.Master, Headless: true})
	if err != nil {
		t.Fatalf("RunSweep: %v", err)
	}
	query, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: ws.Master,
		Question:      "¿Cuál es la regla_facturacion_anual?",
	})
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	for name, notes := range map[string][]string{"sweep": sweep.Notes, "query": query.Notes} {
		if len(notes) != 1 {
			t.Fatalf("%s: notes = %v, se esperaba el aviso Q14", name, notes)
		}
		for _, want := range []string{legacy, filepath.Join(ws.Home, "specs")} {
			if !strings.Contains(notes[0], want) {
				t.Errorf("%s: el aviso debe nombrar %q: %q", name, want, notes[0])
			}
		}
	}
	// Las dos lecturas usan solo el planning home, no la ubicación antigua.
	for _, created := range sweep.CreatedSpecs {
		if strings.HasPrefix(created, legacy) {
			t.Errorf("sweep no debe sembrar en la ubicación antigua: %s", created)
		}
	}
}

func TestExplicitSpecsRootSkipsTheResolution(t *testing.T) {
	// Con una raíz de especificaciones explícita no se invoca Locate: el puntero
	// roto no afecta y no hay notes.
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	writeGoModule(t, ws.Backend, "billing", "regla")
	specsRoot := filepath.Join(t.TempDir(), "openspec")

	res, err := RunSweep(context.Background(), SweepOptions{
		WorkspaceRoot: ws.Backend,
		SpecsRoot:     specsRoot,
		Headless:      true,
	})
	if err != nil {
		t.Fatalf("RunSweep con raíz explícita: %v", err)
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %v, sin resolución no hay avisos", res.Notes)
	}
	if _, err := os.Stat(filepath.Join(specsRoot, "specs")); err != nil {
		t.Errorf("la raíz explícita debe usarse tal cual: %v", err)
	}
}

func TestKnowledgeRejectsAnUnresolvablePlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	ctx := context.Background()

	calls := map[string]func() error{
		"RunSweep": func() error {
			_, err := RunSweep(ctx, SweepOptions{WorkspaceRoot: ws.Backend})
			return err
		},
		"RunQuery": func() error {
			_, err := RunQuery(ctx, QueryOptions{WorkspaceRoot: ws.Backend, Question: "regla_facturacion_anual"})
			return err
		},
		"GenerateCrawlPlan": func() error {
			_, err := GenerateCrawlPlan(ctx, CrawlPlanOptions{
				WorkspaceRoot: ws.Backend,
				JobPath:       filepath.Join(t.TempDir(), "crawl-job.json"),
				Force:         true,
			})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var locateErr *workspace.LocateError
			if err := call(); !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocatePointerBroken {
				t.Fatalf("err = %v, se esperaba el *LocateError pointer_broken", err)
			}
		})
	}
	for _, dir := range []string{ws.Master, ws.Backend} {
		if _, err := os.Stat(filepath.Join(dir, "openspec")); !os.IsNotExist(err) {
			t.Errorf("un rechazo de Locate no debe crear openspec/ en %s (err=%v)", dir, err)
		}
	}
}
