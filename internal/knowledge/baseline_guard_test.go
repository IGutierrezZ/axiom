package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	guardCanonicalConfig = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"
	guardProjectBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: docs/04_Flujos_Negocio.md\n"
	guardInvalidBaseline = "baseline:\n  documents: []\n"
)

// guardWorkspace monta el canónico con la baseline dada y un módulo de código en
// repo-backend con una regla que las consultas pueden encontrar.
func guardWorkspace(t *testing.T, baseline string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), guardCanonicalConfig+baseline)
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "internal", "billing", "invoice.go"),
		"package billing\n// regla_facturacion_anual activa\n")
	return ws
}

// requireNoSpecsWritten comprueba que no existen openspec/specs ni INDEX.md en el
// planning home.
func requireNoSpecsWritten(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	for _, absent := range []string{filepath.Join(ws.Home, "specs"), filepath.Join(ws.Home, "INDEX.md")} {
		if _, err := os.Stat(absent); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe existir %s (stat = %v)", absent, err)
		}
	}
}

func hasNote(notes []string, fragment string) bool {
	for _, note := range notes {
		if strings.Contains(note, fragment) {
			return true
		}
	}
	return false
}

// REQ-24.23: con baseline del proyecto, o indeterminada, el barrido completa el
// análisis, omite la siembra y el índice, devuelve `created_specs: []` y anota la
// omisión, sin error.
func TestSweepWithAProjectOrIndeterminateBaselineSkipsTheSeeding(t *testing.T) {
	tests := []struct {
		name     string
		baseline string
		wants    []string
	}{
		{"del proyecto", guardProjectBaseline, []string{"baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`", "Se omite la siembra"}},
		{"indeterminada", guardInvalidBaseline, []string{"no se puede determinar", "al menos un documento", "axiom workspace validate --path", "Se omite la siembra"}},
	}
	for _, tt := range tests {
		for _, start := range []string{"repositorio de código", "carpeta maestra"} {
			t.Run(tt.name+" desde "+start, func(t *testing.T) {
				ws := guardWorkspace(t, tt.baseline)
				root := ws.Backend
				if start == "carpeta maestra" {
					root = ws.Master
				}

				res, err := RunSweep(context.Background(), SweepOptions{WorkspaceRoot: root, Headless: true})
				if err != nil {
					t.Fatalf("RunSweep = %v, con la guarda completa el resto del resultado", err)
				}
				if len(res.Modules) == 0 {
					t.Errorf("el análisis debe completarse: %+v", res)
				}
				if res.CreatedSpecs == nil || len(res.CreatedSpecs) != 0 || !res.SpecsSkipped {
					t.Errorf("CreatedSpecs = %#v, SpecsSkipped = %v", res.CreatedSpecs, res.SpecsSkipped)
				}
				for _, want := range tt.wants {
					if !hasNote(res.Notes, want) {
						t.Errorf("ninguna nota contiene %q: %q", want, res.Notes)
					}
				}
				raw, err := json.Marshal(res)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(raw), `"created_specs":[]`) || strings.Contains(string(raw), "SpecsSkipped") {
					t.Errorf("el JSON debe llevar created_specs vacío y no el indicador interno: %s", raw)
				}
				requireNoSpecsWritten(t, ws)
			})
		}
	}
}

// Sin baseline del proyecto el barrido siembra como siempre.
func TestSweepWithTheDefaultBaselineStillSeeds(t *testing.T) {
	ws := guardWorkspace(t, "")
	res, err := RunSweep(context.Background(), SweepOptions{WorkspaceRoot: ws.Backend, Headless: true})
	if err != nil {
		t.Fatalf("RunSweep = %v", err)
	}
	if res.SpecsSkipped || len(res.CreatedSpecs) == 0 {
		t.Errorf("CreatedSpecs = %v, SpecsSkipped = %v: debe sembrar", res.CreatedSpecs, res.SpecsSkipped)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
		t.Errorf("debe sincronizar INDEX.md: %v", err)
	}
}

// REQ-24.23 y REQ-24.25: con --enrich la consulta se responde sin enriquecer y
// anota la omisión; con baseline del proyecto avisa además de que la búsqueda
// spec-first no cubre la baseline externa, también sin --enrich.
func TestQueryEnrichWithAProjectOrIndeterminateBaseline(t *testing.T) {
	question := "¿Cuál es la regla_facturacion_anual?"
	tests := []struct {
		name         string
		baseline     string
		wants        []string
		wantSpecNote bool
	}{
		{"del proyecto", guardProjectBaseline, []string{"baseline de especificaciones es del proyecto", "Se omite el enriquecimiento"}, true},
		{"indeterminada", guardInvalidBaseline, []string{"no se puede determinar", "al menos un documento", "Se omite el enriquecimiento"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := guardWorkspace(t, tt.baseline)

			res, err := RunQuery(context.Background(), QueryOptions{WorkspaceRoot: ws.Backend, Question: question, Enrich: true})
			if err != nil {
				t.Fatalf("RunQuery = %v, la consulta se responde igualmente", err)
			}
			if len(res.Evidences) == 0 || res.DirectAnswer == "" {
				t.Errorf("la consulta debe responderse con sus evidencias: %+v", res)
			}
			if res.SpecUpdated || res.TargetSpecPath != "" {
				t.Errorf("no debe enriquecerse: %+v", res)
			}
			for _, want := range tt.wants {
				if !hasNote(res.Notes, want) {
					t.Errorf("ninguna nota contiene %q: %q", want, res.Notes)
				}
			}
			if got := hasNote(res.Notes, "La búsqueda spec-first no cubre la baseline externa"); got != tt.wantSpecNote {
				t.Errorf("aviso de búsqueda spec-first = %v, se esperaba %v: %q", got, tt.wantSpecNote, res.Notes)
			}
			requireNoSpecsWritten(t, ws)

			// Sin --enrich la consulta es de solo lectura, pero el aviso de la
			// búsqueda spec-first sigue apareciendo con baseline del proyecto.
			res, err = RunQuery(context.Background(), QueryOptions{WorkspaceRoot: ws.Backend, Question: question})
			if err != nil {
				t.Fatalf("RunQuery sin --enrich = %v", err)
			}
			if hasNote(res.Notes, "Se omite el enriquecimiento") {
				t.Errorf("sin --enrich no hay nada que omitir: %q", res.Notes)
			}
			if got := hasNote(res.Notes, "La búsqueda spec-first no cubre la baseline externa"); got != tt.wantSpecNote {
				t.Errorf("sin --enrich, aviso de búsqueda spec-first = %v, se esperaba %v: %q", got, tt.wantSpecNote, res.Notes)
			}
		})
	}

	t.Run("sin baseline del proyecto enriquece como siempre", func(t *testing.T) {
		ws := guardWorkspace(t, "")
		res, err := RunQuery(context.Background(), QueryOptions{WorkspaceRoot: ws.Backend, Question: question, Enrich: true})
		if err != nil {
			t.Fatalf("RunQuery = %v", err)
		}
		if !res.SpecUpdated {
			t.Errorf("se esperaba SpecUpdated=true: %+v", res)
		}
		if hasNote(res.Notes, "spec-first") {
			t.Errorf("sin baseline del proyecto no hay aviso de búsqueda spec-first: %q", res.Notes)
		}
	})
}

// crawlJobOf genera el plan de crawl del repositorio de código y devuelve la
// ruta del job y el identificador de una unidad.
func crawlJobOf(t *testing.T, ws workspacetest.Workspace) (jobPath, unitID string) {
	t.Helper()
	jobPath = filepath.Join(t.TempDir(), "crawl-job.json")
	job, err := GenerateCrawlPlan(context.Background(), CrawlPlanOptions{WorkspaceRoot: ws.Backend, JobPath: jobPath, Force: true})
	if err != nil {
		t.Fatalf("GenerateCrawlPlan: %v", err)
	}
	if len(job.Units) == 0 {
		t.Fatalf("el plan debe tener unidades: %+v", job)
	}
	return jobPath, job.Units[0].ID
}

// REQ-24.23: con baseline del proyecto o indeterminada, registrar una unidad y
// finalizar el crawl se rechazan sin escribir en openspec/specs ni INDEX.md.
func TestCrawlRecordAndFinalizeAreRejectedWithAProjectOrIndeterminateBaseline(t *testing.T) {
	tests := []struct {
		name      string
		baseline  string
		ownership workspace.BaselineOwnership
	}{
		{"del proyecto", guardProjectBaseline, workspace.BaselineProject},
		{"indeterminada", guardInvalidBaseline, workspace.BaselineIndeterminate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// El plan se genera con la baseline por defecto y la baseline se declara
			// después: la guarda se resuelve en el momento de escribir.
			ws := guardWorkspace(t, "")
			jobPath, unitID := crawlJobOf(t, ws)
			workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), guardCanonicalConfig+tt.baseline)

			_, err := RecordUnit(context.Background(), jobPath, UnitAnalysisInput{UnitID: unitID, Summary: "Resumen del análisis."})
			var second *workspace.SecondBaselineError
			if !errors.As(err, &second) || second.Ownership != tt.ownership {
				t.Fatalf("RecordUnit = %v, se esperaba un *workspace.SecondBaselineError %s", err, tt.ownership)
			}
			if !strings.Contains(err.Error(), "axiom knowledge crawl --record-unit") {
				t.Errorf("el mensaje debe nombrar la operación rechazada: %v", err)
			}

			_, err = FinalizeCrawl(context.Background(), jobPath)
			if !errors.As(err, &second) || second.Ownership != tt.ownership {
				t.Fatalf("FinalizeCrawl = %v, se esperaba un *workspace.SecondBaselineError %s", err, tt.ownership)
			}
			if !strings.Contains(err.Error(), "axiom knowledge crawl --finalize") {
				t.Errorf("el mensaje debe nombrar la operación rechazada: %v", err)
			}

			requireNoSpecsWritten(t, ws)
			job, loadErr := LoadJob(jobPath)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if job.CompletedUnits != 0 {
				t.Errorf("el rechazo no debe actualizar el job: %+v", job)
			}
		})
	}
}

// REQ-24.22: si Locate rechaza en el momento de escribir, no hay respuesta de
// baseline y RecordUnit y FinalizeCrawl no escriben.
func TestCrawlWritesAreRejectedWhenLocateRejects(t *testing.T) {
	ws := guardWorkspace(t, "")
	jobPath, unitID := crawlJobOf(t, ws)
	// El puntero se rompe después de generar el plan.
	workspacetest.WriteFile(t, filepath.Join(ws.Master, ".axiom-workspace"), "config: repo-inexistente/axiom.yaml\n")

	var locateErr *workspace.LocateError
	if _, err := RecordUnit(context.Background(), jobPath, UnitAnalysisInput{UnitID: unitID, Summary: "Resumen."}); !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocatePointerBroken {
		t.Errorf("RecordUnit = %v, se esperaba el *workspace.LocateError pointer_broken", err)
	}
	if _, err := FinalizeCrawl(context.Background(), jobPath); !errors.As(err, &locateErr) {
		t.Errorf("FinalizeCrawl = %v, se esperaba el *workspace.LocateError", err)
	}
	requireNoSpecsWritten(t, ws)
}

// Sin baseline del proyecto, RecordUnit y FinalizeCrawl escriben como siempre.
func TestCrawlWritesWithTheDefaultBaselineStillWork(t *testing.T) {
	ws := guardWorkspace(t, "")
	jobPath, unitID := crawlJobOf(t, ws)

	if _, err := RecordUnit(context.Background(), jobPath, UnitAnalysisInput{UnitID: unitID, Summary: "Resumen del análisis."}); err != nil {
		t.Fatalf("RecordUnit = %v", err)
	}
	res, err := FinalizeCrawl(context.Background(), jobPath)
	if err != nil {
		t.Fatalf("FinalizeCrawl = %v", err)
	}
	if !res.IndexUpdated {
		t.Errorf("FinalizeCrawl debe actualizar el índice: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "INDEX.md")); err != nil {
		t.Errorf("debe existir INDEX.md: %v", err)
	}
}
