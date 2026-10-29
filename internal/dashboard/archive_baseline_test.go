package dashboard

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/livingdoc"
)

const (
	archiveBaselineConfig  = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	archiveBaselineBlock   = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: docs/04_Flujos_Negocio.md\n"
	archiveInvalidBaseline = "baseline:\n  documents: []\n"
)

// archiveRouter monta el servidor sobre un proyecto temporal con el axiom.yaml
// dado, de modo que nada se escribe en el checkout (repo_guard).
func archiveRouter(t *testing.T, axiomYAML string) (root string, router http.Handler) {
	t.Helper()
	root = t.TempDir()
	writeFixtureFile(t, root, "axiom.yaml", axiomYAML)
	writeFixtureFile(t, root, "openspec/changes/.keep", "")
	return root, NewServer(NewService(root)).Router()
}

func serve(router http.Handler, method, target string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(method, target, nil))
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON inválido (%d): %v\n%s", rr.Code, err, rr.Body.String())
	}
	return body
}

// REQ-4.1: con baseline del proyecto, GET /api/archive/specs responde 200 con
// la lista vacía, el indicador de baseline externa y los dominios y rutas
// declarados.
func TestArchiveSpecsWithAProjectBaselineReportsItAsExternal(t *testing.T) {
	root, router := archiveRouter(t, archiveBaselineConfig+archiveBaselineBlock)
	// Una spec en el planning home no se indexa: la baseline es del proyecto.
	writeFixtureFile(t, root, "openspec/specs/heredado/spec.md", "# Heredado\n\n### Requirement: R (REQ-1)\n\n#### Scenario: S\n")

	rr := serve(router, http.MethodGet, "/api/archive/specs")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/archive/specs = %d: %s", rr.Code, rr.Body.String())
	}
	var catalog livingdoc.LivingCatalog
	if err := json.Unmarshal(rr.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Specs == nil || len(catalog.Specs) != 0 {
		t.Errorf("Specs = %#v, se esperaba una lista vacía", catalog.Specs)
	}
	if catalog.ExternalBaseline == nil || len(catalog.ExternalBaseline.Documents) != 2 {
		t.Fatalf("ExternalBaseline = %+v, se esperaban los dos documentos declarados", catalog.ExternalBaseline)
	}
	if got := catalog.ExternalBaseline.Documents[0]; got.Domain != "modelo-de-datos" || !sameFile(t, got.Path, filepath.Join(root, "03_Modelo_Datos.md")) {
		t.Errorf("primer documento = %+v", got)
	}
	if !strings.Contains(rr.Body.String(), `"specs":[]`) {
		t.Errorf("el cuerpo debe llevar la lista vacía de especificaciones: %s", rr.Body.String())
	}
}

// sameFile compara dos rutas que pueden no existir, sin distinguir las
// diferencias de grafía del directorio temporal (nombres cortos en Windows).
func sameFile(t *testing.T, got, want string) bool {
	t.Helper()
	return sameDirectory(t, filepath.Dir(got), filepath.Dir(want)) && filepath.Base(got) == filepath.Base(want)
}

// REQ-4.1: el detalle de un dominio declarado responde 4xx con el mensaje de
// baseline externa y la ruta; un dominio no declarado responde 404 como siempre.
func TestArchiveSpecDetailOfADeclaredDomain(t *testing.T) {
	_, router := archiveRouter(t, archiveBaselineConfig+archiveBaselineBlock)

	rr := serve(router, http.MethodGet, "/api/archive/specs/modelo-de-datos")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET del dominio declarado = %d, se esperaba un error de cliente 404", rr.Code)
	}
	body := decodeBody(t, rr)
	message, _ := body["error"].(string)
	for _, want := range []string{"baseline externa", "modelo-de-datos", "03_Modelo_Datos.md"} {
		if !strings.Contains(message, want) {
			t.Errorf("el mensaje no contiene %q: %s", want, message)
		}
	}
	if body["external_baseline"] != true || body["domain"] != "modelo-de-datos" {
		t.Errorf("cuerpo = %v, se esperaban el indicador y el dominio", body)
	}
	if path, _ := body["path"].(string); !strings.HasSuffix(path, "03_Modelo_Datos.md") {
		t.Errorf("path = %q", path)
	}

	rr = serve(router, http.MethodGet, "/api/archive/specs/no-declarado")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET de un dominio no declarado = %d, se esperaba 404", rr.Code)
	}
	if body := decodeBody(t, rr); body["external_baseline"] != nil {
		t.Errorf("un dominio no declarado no es baseline externa: %v", body)
	}
}

// REQ-4.1 y REQ-24.23: POST /api/archive/sync responde 409 con la continuación y
// no escribe nada, ni con baseline del proyecto ni con una indeterminada.
func TestArchiveSyncIsRejectedWithAProjectOrIndeterminateBaseline(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		ownership string
		wants     []string
	}{
		{"del proyecto", archiveBaselineConfig + archiveBaselineBlock, "project", []string{"baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`"}},
		{"indeterminada", archiveBaselineConfig + archiveInvalidBaseline, "indeterminate", []string{"no se puede determinar", "al menos un documento", "axiom workspace validate --path"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, router := archiveRouter(t, tt.config)

			rr := serve(router, http.MethodPost, "/api/archive/sync")
			if rr.Code != http.StatusConflict {
				t.Fatalf("POST /api/archive/sync = %d, se esperaba 409: %s", rr.Code, rr.Body.String())
			}
			body := decodeBody(t, rr)
			message, _ := body["error"].(string)
			for _, want := range tt.wants {
				if !strings.Contains(message, want) {
					t.Errorf("el mensaje no contiene %q: %s", want, message)
				}
			}
			if body["ownership"] != tt.ownership {
				t.Errorf("ownership = %v, se esperaba %q", body["ownership"], tt.ownership)
			}
			for _, absent := range []string{filepath.Join(root, "openspec", "INDEX.md"), filepath.Join(root, "openspec", "specs")} {
				if _, err := os.Stat(absent); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("no debe existir %s (stat = %v)", absent, err)
				}
			}
		})
	}
}

// Sin baseline del proyecto el endpoint conserva su comportamiento.
func TestArchiveSyncWithTheDefaultBaselineStillWrites(t *testing.T) {
	root, router := archiveRouter(t, archiveBaselineConfig)
	writeFixtureFile(t, root, "openspec/specs/dominio/spec.md", "# Dominio\n\n### Requirement: R (REQ-1)\n\n#### Scenario: S\n")

	if rr := serve(router, http.MethodPost, "/api/archive/sync"); rr.Code != http.StatusOK {
		t.Fatalf("POST /api/archive/sync = %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "openspec", "INDEX.md")); err != nil {
		t.Errorf("el sync debe escribir INDEX.md: %v", err)
	}
}

// REQ-4.2: la pantalla de specs vivas muestra el aviso de baseline externa en
// lugar del catálogo vacío y el botón de sincronización muestra el rechazo con su
// continuación. Sin navegador en las pruebas, js/screens/livingdoc.js y el helper
// api() se inspeccionan por marcadores: api() extrae el campo `error` del cuerpo
// del 409 (que nombra la baseline y la continuación) y la pantalla lo presenta.
func TestLivingDocsScreenPresentsTheExternalBaseline(t *testing.T) {
	js := readAsset(t, "js/screens/livingdoc.js")
	for name, want := range map[string]string{
		"lee el indicador del catálogo":           "catalog.external_baseline",
		"avisa de que la baseline es externa":     "La baseline de especificaciones es externa",
		"lista dominios y rutas":                  "doc.domain",
		"consulta el catálogo con api()":          "api('/api/archive/specs')",
		"sincroniza con api()":                    "api('/api/archive/sync'",
		"muestra el rechazo de la sincronización": "Fallo al sincronizar: ${err.message}",
		"muestra el error del catálogo":           "stateError(ctx.catalogError, 'Error cargando el catálogo')",
		"muestra el error del detalle":            "stateError(d.error, 'No se pudo cargar la especificación')",
		"explica la baseline externa en el visor": "d.external",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("la pantalla no %s (%q)", name, want)
		}
	}

	// El mensaje del rechazo llega a la pantalla porque api() lo toma del cuerpo.
	apiJS := readAsset(t, "js/core/api.js")
	if !strings.Contains(apiJS, "if (data.error) return String(data.error)") {
		t.Errorf("api() debe usar el campo error del cuerpo como mensaje de ApiError")
	}
}
