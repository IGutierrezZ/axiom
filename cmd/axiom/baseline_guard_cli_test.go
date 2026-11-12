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
	cliCanonicalConfig  = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n  frontend:\n    name: Frontend\n    repositories:\n      - path: repo-frontend\n"
	cliResidualBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
)

// baselineWorkspace monta el canónico con el bloque dado y un módulo de código
// con una regla que las consultas pueden encontrar.
func baselineWorkspace(t *testing.T, block string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), cliCanonicalConfig+block)
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

// REQ-30.21: `archive sync` y `archive coldstart` se rechazan con código distinto
// de cero desde cualquier punto de partida del canónico, con el error tipado de la
// retirada y la migración, sin crear openspec/specs ni INDEX.md.
func TestArchiveSyncAndColdstartAreRejectedWithAResidualBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliResidualBaseline)
	wants := []string{"baseline_retired", "axiom archive adopt scan"}
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "archive", "sync", "--path", start)
			if code == 0 {
				t.Fatalf("archive sync debe salir con código distinto de cero:\n%s", output)
			}
			requireContains(t, output, wants...)

			output, code = runAxiom(t, "archive", "coldstart", "--change", planningHomeChange, "--path", start)
			if code == 0 {
				t.Fatalf("archive coldstart debe salir con código distinto de cero:\n%s", output)
			}
			requireContains(t, output, wants...)
			requireNoLivingSpecs(t, ws)
		})
	}
}

// REQ-30.21 y REQ-1.1: con un `baseline:` residual `archive list` lee
// openspec/specs/ como siempre (código 0) y avisa del bloque retirado; ya no hay
// catálogo externo.
func TestArchiveListWarnsAboutAResidualBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliResidualBaseline)
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "archive", "list", "--path", start)
			if code != 0 {
				t.Fatalf("archive list salió con %d:\n%s", code, output)
			}
			requireContains(t, output, "[AVISO]", "baseline_retired", "axiom archive adopt scan", "No se encontraron especificaciones vivas")
			if strings.Contains(output, "baseline de especificaciones es externa") {
				t.Errorf("la baseline externa ya no existe:\n%s", output)
			}

			output, code = runAxiom(t, "archive", "show", "--path", start, "modelo-de-datos")
			if code == 0 {
				t.Fatalf("un dominio de la baseline retirada no existe en openspec/specs:\n%s", output)
			}
		})
	}
	requireNoLivingSpecs(t, ws)
}

// REQ-30.21: `knowledge sweep` completa el análisis con código 0, devuelve
// `created_specs` vacío y omite la siembra y el índice.
func TestKnowledgeSweepSkipsTheSeedingWithAResidualBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliResidualBaseline)
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
			if !strings.Contains(strings.Join(res.Notes, " "), "baseline_retired") {
				t.Errorf("notes = %q, se esperaba la omisión con el error tipado", res.Notes)
			}
		})
	}
	requireNoLivingSpecs(t, ws)

	t.Run("texto", func(t *testing.T) {
		output, code := runAxiom(t, "knowledge", "sweep", "--cwd", ws.Backend)
		if code != 0 {
			t.Fatalf("knowledge sweep salió con %d:\n%s", code, output)
		}
		requireContains(t, output, "[AVISO]", "baseline_retired", "No se sembraron especificaciones")
		if strings.Contains(output, "Catálogo maestro actualizado") {
			t.Errorf("no debe afirmar que actualizó INDEX.md:\n%s", output)
		}
		requireNoLivingSpecs(t, ws)
	})
}

// REQ-30.21: `knowledge query --enrich` responde sin enriquecer y anota la omisión.
func TestKnowledgeQueryEnrichIsSkippedWithAResidualBaseline(t *testing.T) {
	ws := baselineWorkspace(t, cliResidualBaseline)
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
			if res.SpecUpdated || !strings.Contains(notes, "Se omite el enriquecimiento") || !strings.Contains(notes, "baseline_retired") {
				t.Errorf("SpecUpdated = %v, notes = %q", res.SpecUpdated, res.Notes)
			}
		})
	}
	requireNoLivingSpecs(t, ws)
}

// REQ-30.21: `knowledge crawl --record-unit` y `--finalize` se rechazan con código
// distinto de cero sin escribir en openspec/specs ni INDEX.md.
func TestKnowledgeCrawlRecordAndFinalizeAreRejectedWithAResidualBaseline(t *testing.T) {
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
	// El bloque se declara después del plan: la guarda se resuelve al escribir.
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), cliCanonicalConfig+cliResidualBaseline)
	payload := filepath.Join(t.TempDir(), "analisis.json")
	workspacetest.WriteFile(t, payload, `{"unit_id":"`+plan.Units[0].ID+`","summary":"Resumen del análisis."}`)

	output, code := runAxiom(t, "knowledge", "crawl", "--record-unit", "--job", job, "--unit", plan.Units[0].ID, "--input", payload)
	if code == 0 {
		t.Fatalf("crawl --record-unit debe salir con código distinto de cero:\n%s", output)
	}
	requireContains(t, output, "baseline_retired", "axiom archive adopt scan")

	output, code = runAxiom(t, "knowledge", "crawl", "--finalize", "--job", job)
	if code == 0 {
		t.Fatalf("crawl --finalize debe salir con código distinto de cero:\n%s", output)
	}
	requireContains(t, output, "baseline_retired", "axiom archive adopt scan")
	requireNoLivingSpecs(t, ws)
}

// REQ-30.21 y REQ-2.2: `axiom init` sobre un `axiom.yaml` con un `baseline:`
// residual se rechaza con código distinto de cero y la migración, sin escribir ni
// registrar nada (tampoco con --force); sobre una carpeta maestra con puntero hacia
// otro planning home se rechaza igualmente.
func TestInitRejectsAResidualBaselineAndAForeignPointer(t *testing.T) {
	const embedded = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	for name, args := range map[string][]string{
		"perfil completo":          {},
		"perfil knowledge":         {"--profile", "knowledge"},
		"con --force":              {"--force"},
		"perfil knowledge forzado": {"--profile", "knowledge", "--force"},
	} {
		t.Run("baseline residual "+name, func(t *testing.T) {
			dir := t.TempDir()
			config := embedded + cliResidualBaseline
			workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config)

			output, code := runAxiom(t, append([]string{"init", "--path", dir}, args...)...)
			if code == 0 {
				t.Fatalf("init debe rechazarse con código distinto de cero:\n%s", output)
			}
			requireContains(t, output, "baseline_retired", "axiom archive adopt scan")
			if _, err := os.Stat(filepath.Join(dir, "openspec")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("no debe crearse openspec/ (stat = %v)", err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "axiom.yaml"))
			if err != nil || string(got) != config {
				t.Errorf("el axiom.yaml existente no se toca (%v):\n%s", err, got)
			}
		})
	}

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
		config := embedded + "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n"
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
