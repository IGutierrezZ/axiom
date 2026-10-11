package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	cliCanonicalConfig = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n  frontend:\n    name: Frontend\n    repositories:\n      - path: repo-frontend\n"
	cliProjectBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: docs/04_Flujos_Negocio.md\n"
	cliInvalidBaseline = "baseline:\n  documents: []\n"
)

// baselineWorkspace monta el canónico con la baseline dada y un módulo de código
// con una regla que las consultas pueden encontrar.
func baselineWorkspace(t *testing.T, baseline string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), cliCanonicalConfig+baseline)
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "internal", "billing", "invoice.go"),
		"package billing\n// regla_facturacion_anual activa\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", planningHomeChange, "spec.md"),
		"# Spec\n### Requirement: Regla (REQ-1)\n#### Scenario: S\n")
	return ws
}

func requireNoLivingSpecs(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	for _, absent := range []string{filepath.Join(ws.Home, "specs"), filepath.Join(ws.Home, "INDEX.md")} {
		if _, err := os.Stat(absent); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe existir %s (stat = %v)", absent, err)
		}
	}
}

// jsonOf recorta la salida del subproceso auxiliar, que el marco de pruebas
// cierra con una línea PASS, hasta la llave que cierra el objeto JSON.
func jsonOf(t *testing.T, output string) []byte {
	t.Helper()
	end := strings.LastIndex(output, "}")
	if end < 0 {
		t.Fatalf("la salida no contiene un objeto JSON:\n%s", output)
	}
	return []byte(output[:end+1])
}

func requireContains(t *testing.T, output string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, output)
		}
	}
}

// REQ-24.23: `archive sync` y `archive coldstart` se rechazan con código distinto
// de cero desde cualquier punto de partida del canónico, nombrando la baseline
// declarada y la continuación, sin crear openspec/specs ni INDEX.md.
func TestArchiveSyncAndColdstartAreRejectedWithAProjectBaseline(t *testing.T) {
	for _, tt := range []struct {
		name     string
		baseline string
		wants    []string
	}{
		{"del proyecto", cliProjectBaseline, []string{"baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`"}},
		{"indeterminada", cliInvalidBaseline, []string{"no se puede determinar", "al menos un documento", "axiom workspace validate --path"}},
	} {
		ws := baselineWorkspace(t, tt.baseline)
		for name, start := range canonicalStarts(ws) {
			t.Run(tt.name+" desde "+name, func(t *testing.T) {
				output, code := runAxiom(t, "archive", "sync", "--path", start)
				if code == 0 {
					t.Fatalf("archive sync debe salir con código distinto de cero:\n%s", output)
				}
				requireContains(t, output, tt.wants...)

				output, code = runAxiom(t, "archive", "coldstart", "--change", planningHomeChange, "--path", start)
				if code == 0 {
					t.Fatalf("archive coldstart debe salir con código distinto de cero:\n%s", output)
				}
				requireContains(t, output, tt.wants...)
				requireNoLivingSpecs(t, ws)
			})
		}
	}
}

// REQ-24.25: `archive list` informa de que la baseline es externa y lista los
// dominios con sus rutas, sin indexar y con código 0; `archive show` de un
// dominio declarado informa de que pertenece a la baseline externa y su ruta.
func TestArchiveListAndShowReportAnExternalBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliProjectBaseline)
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "archive", "list", "--path", start)
			if code != 0 {
				t.Fatalf("archive list salió con %d:\n%s", code, output)
			}
			requireContains(t, output,
				"baseline de especificaciones es externa",
				"modelo-de-datos", filepath.Join(ws.Specs, "03_Modelo_Datos.md"),
				"flujos-de-negocio", filepath.Join(ws.Specs, "docs", "04_Flujos_Negocio.md"))
			if strings.Contains(output, "No se encontraron especificaciones vivas") {
				t.Errorf("un catálogo vacío no debe interpretarse como ausencia de especificación:\n%s", output)
			}

			output, code = runAxiom(t, "archive", "show", "--path", start, "modelo-de-datos")
			if code != 0 {
				t.Fatalf("archive show salió con %d:\n%s", code, output)
			}
			requireContains(t, output, "baseline externa", "modelo-de-datos", filepath.Join(ws.Specs, "03_Modelo_Datos.md"))
		})
	}
	requireNoLivingSpecs(t, ws)

	t.Run("dominio no declarado", func(t *testing.T) {
		output, code := runAxiom(t, "archive", "show", "--path", ws.Master, "dominio-no-declarado")
		if code == 0 {
			t.Fatalf("un dominio no declarado sigue siendo un error:\n%s", output)
		}
	})
}

// REQ-24.23: `knowledge sweep` completa el análisis con código 0, devuelve
// `created_specs` vacío y omite la siembra y el índice.
func TestKnowledgeSweepSkipsTheSeedingWithAProjectBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliProjectBaseline)
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "knowledge", "sweep", "--cwd", start, "--json")
			if code != 0 {
				t.Fatalf("knowledge sweep salió con %d:\n%s", code, output)
			}
			var res struct {
				Modules      []json.RawMessage `json:"modules"`
				CreatedSpecs []string          `json:"created_specs"`
				Notes        []string          `json:"notes"`
			}
			if err := json.Unmarshal(jsonOf(t, output), &res); err != nil {
				t.Fatalf("la salida no es JSON: %v\n%s", err, output)
			}
			if res.CreatedSpecs == nil || len(res.CreatedSpecs) != 0 {
				t.Errorf("created_specs = %#v, se esperaba []", res.CreatedSpecs)
			}
			if !strings.Contains(strings.Join(res.Notes, " "), "baseline de especificaciones es del proyecto") {
				t.Errorf("notes = %q, se esperaba la omisión", res.Notes)
			}
		})
	}
	requireNoLivingSpecs(t, ws)

	t.Run("texto", func(t *testing.T) {
		output, code := runAxiom(t, "knowledge", "sweep", "--cwd", ws.Backend)
		if code != 0 {
			t.Fatalf("knowledge sweep salió con %d:\n%s", code, output)
		}
		requireContains(t, output, "[AVISO]", "baseline de especificaciones es del proyecto", "No se sembraron especificaciones")
		if strings.Contains(output, "Catálogo maestro actualizado") {
			t.Errorf("no debe afirmar que actualizó INDEX.md:\n%s", output)
		}
		requireNoLivingSpecs(t, ws)
	})
}

// REQ-24.23 y REQ-24.25: `knowledge query --enrich` responde sin enriquecer y
// anota la omisión; la búsqueda spec-first avisa de que no cubre la baseline.
func TestKnowledgeQueryEnrichIsSkippedWithAProjectBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliProjectBaseline)
	// Solo la carpeta maestra y el repositorio de código contienen código con la
	// regla; el repositorio de especificaciones no tiene evidencias que enriquecer.
	for name, start := range map[string]string{"carpeta maestra": ws.Master, "repositorio de código": ws.Backend} {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "knowledge", "query", "¿Cuál es la regla_facturacion_anual?", "--cwd", start, "--enrich", "--json")
			if code != 0 {
				t.Fatalf("knowledge query salió con %d:\n%s", code, output)
			}
			var res struct {
				SpecUpdated bool     `json:"spec_updated"`
				Notes       []string `json:"notes"`
			}
			if err := json.Unmarshal(jsonOf(t, output), &res); err != nil {
				t.Fatalf("la salida no es JSON: %v\n%s", err, output)
			}
			notes := strings.Join(res.Notes, " ")
			if res.SpecUpdated || !strings.Contains(notes, "Se omite el enriquecimiento") || !strings.Contains(notes, "La búsqueda spec-first no cubre la baseline externa") {
				t.Errorf("SpecUpdated = %v, notes = %q", res.SpecUpdated, res.Notes)
			}
		})
	}
	requireNoLivingSpecs(t, ws)

	t.Run("texto", func(t *testing.T) {
		output, code := runAxiom(t, "knowledge", "query", "¿Cuál es la regla_facturacion_anual?", "--cwd", ws.Backend)
		if code != 0 {
			t.Fatalf("knowledge query salió con %d:\n%s", code, output)
		}
		requireContains(t, output, "[AVISO]", "La búsqueda spec-first no cubre la baseline externa")
	})
}

// REQ-24.23: `knowledge crawl --record-unit` y `--finalize` se rechazan con
// código distinto de cero sin escribir en openspec/specs ni INDEX.md.
func TestKnowledgeCrawlRecordAndFinalizeAreRejectedWithAProjectBaseline(t *testing.T) {
	ws := baselineWorkspace(t, "")
	job := filepath.Join(t.TempDir(), "crawl-job.json")
	if output, code := runAxiom(t, "knowledge", "crawl", "--plan", "--cwd", ws.Backend, "--job", job, "--json"); code != 0 {
		t.Fatalf("crawl --plan salió con %d:\n%s", code, output)
	}
	var plan struct {
		Units []struct {
			ID string `json:"id"`
		} `json:"units"`
	}
	raw, err := os.ReadFile(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plan); err != nil || len(plan.Units) == 0 {
		t.Fatalf("el plan debe tener unidades (%v): %s", err, raw)
	}
	// La baseline se declara después del plan: la guarda se resuelve al escribir.
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), cliCanonicalConfig+cliProjectBaseline)
	payload := filepath.Join(t.TempDir(), "analisis.json")
	workspacetest.WriteFile(t, payload, `{"unit_id":"`+plan.Units[0].ID+`","summary":"Resumen del análisis."}`)

	output, code := runAxiom(t, "knowledge", "crawl", "--record-unit", "--job", job, "--unit", plan.Units[0].ID, "--input", payload)
	if code == 0 {
		t.Fatalf("crawl --record-unit debe salir con código distinto de cero:\n%s", output)
	}
	requireContains(t, output, "axiom knowledge crawl --record-unit", "baseline de especificaciones es del proyecto")

	output, code = runAxiom(t, "knowledge", "crawl", "--finalize", "--job", job)
	if code == 0 {
		t.Fatalf("crawl --finalize debe salir con código distinto de cero:\n%s", output)
	}
	requireContains(t, output, "axiom knowledge crawl --finalize", "baseline de especificaciones es del proyecto")
	requireNoLivingSpecs(t, ws)
}

// REQ-24.23 y REQ-2.2: `axiom init` sobre un `axiom.yaml` con baseline del
// proyecto informa de la omisión con código 0; sobre una carpeta maestra con
// puntero hacia otro planning home se rechaza sin escribir ni registrar.
func TestInitReportsTheProjectBaselineAndRejectsAForeignPointer(t *testing.T) {
	t.Run("baseline del proyecto", func(t *testing.T) {
		dir := t.TempDir()
		config := "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n" + cliProjectBaseline
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config)

		output, code := runAxiom(t, "init", "--path", dir)
		if code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, output)
		}
		requireContains(t, output, "[AVISO]", "baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio")
		if _, err := os.Stat(filepath.Join(dir, "openspec", "specs")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe existir openspec/specs (stat = %v)", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "openspec", "changes")); err != nil {
			t.Errorf("openspec/changes sí se crea: %v", err)
		}
	})

	t.Run("perfil knowledge no declara openspec/specs entre lo creado", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: openspec\nroles:\n  knowledge:\n    name: K\n    repositories:\n      - path: .\n"+cliProjectBaseline)
		output, code := runAxiom(t, "init", "--path", dir, "--profile", "knowledge")
		if code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, output)
		}
		for _, absent := range []string{"openspec/specs", "openspec/INDEX.md"} {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(absent))); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("no debe existir %s (stat = %v)", absent, err)
			}
		}
	})

	t.Run("puntero hacia otro planning home", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		output, code := runAxiom(t, "init", "--path", ws.Master)
		if code == 0 {
			t.Fatalf("init debe rechazarse con código distinto de cero:\n%s", output)
		}
		requireContains(t, output, ".axiom-workspace", ws.Home, "axiom init --path")
		if _, err := os.Stat(filepath.Join(ws.Master, "openspec")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe crearse openspec/ en la carpeta maestra (stat = %v)", err)
		}
		if _, err := os.Stat(filepath.Join(ws.Master, "axiom.yaml")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe crearse axiom.yaml en la carpeta maestra (stat = %v)", err)
		}
		// El subproceso hereda el HOME del sandbox: el hub no debe registrar la carpeta.
		manager, err := hub.NewManager("")
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := manager.Load()
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range cfg.Workspaces {
			if strings.EqualFold(filepath.Clean(record.Path), filepath.Clean(ws.Master)) {
				t.Errorf("el rechazo no debe registrar la carpeta maestra en el hub: %+v", record)
			}
		}
	})

	t.Run("force conserva un axiom.yaml con declaraciones y sale con 0", func(t *testing.T) {
		dir := t.TempDir()
		config := "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n" + cliProjectBaseline
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config)
		output, code := runAxiom(t, "init", "--path", dir, "--force")
		if code != 0 {
			t.Fatalf("init --force salió con %d:\n%s", code, output)
		}
		requireContains(t, output, "Se conserva", "--force")
		got, err := os.ReadFile(filepath.Join(dir, "axiom.yaml"))
		if err != nil || string(got) != config {
			t.Errorf("el axiom.yaml debe conservarse íntegro (%v):\n%s", err, got)
		}
	})
}
