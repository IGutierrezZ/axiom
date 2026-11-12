package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// REQ-30.27: el alta de proyecto devuelve el informe de adopción y la instrucción para lanzar
// la adopción desde el agente; el dashboard no ejecuta nada ni crea el cambio de adopción.

type projectInitAdoptionBody struct {
	Adoption *struct {
		SourceCount int `json:"source_count"`
		UnitCount   int `json:"unit_count"`
		Sources     []struct {
			Path  string `json:"path"`
			Units int    `json:"units"`
		} `json:"sources"`
		ContextCandidates []struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		} `json:"context_candidates"`
	} `json:"adoption"`
	AdoptionInstruction string `json:"adoption_instruction"`
}

func writeAdoptionFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectsInitReturnsTheAdoptionReportAndInstruction(t *testing.T) {
	router, _, _ := newPrefsServer(t, t.TempDir())
	dir := filepath.Join(t.TempDir(), "con-requisitos")
	writeAdoptionFile(t, dir, "docs/requisitos.md", "# Requisitos\n\n## RF-001 Alta\n\nTexto.\n")
	writeAdoptionFile(t, dir, "ARCHITECTURE.md", "# Arquitectura\n")

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	var body projectInitAdoptionBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Adoption == nil || body.Adoption.SourceCount != 1 || body.Adoption.UnitCount != 2 || body.Adoption.Sources[0].Path != "docs/requisitos.md" {
		t.Fatalf("informe de adopción = %+v", body.Adoption)
	}
	if len(body.Adoption.ContextCandidates) != 1 || body.Adoption.ContextCandidates[0].Mode != "maintained" {
		t.Fatalf("candidatos = %+v", body.Adoption.ContextCandidates)
	}
	for _, want := range []string{"axiom archive adopt scan --path", "--change adopcion-openspec"} {
		if !strings.Contains(body.AdoptionInstruction, want) {
			t.Errorf("la instrucción debe contener %q: %s", want, body.AdoptionInstruction)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "openspec", "changes", "adopcion-openspec")); !os.IsNotExist(err) {
		t.Errorf("el dashboard no crea el cambio de adopción (%v)", err)
	}
}

func TestProjectsInitWithoutSourcesOmitsTheAdoptionInstruction(t *testing.T) {
	router, _, _ := newPrefsServer(t, t.TempDir())
	dir := filepath.Join(t.TempDir(), "vacio")

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	var body projectInitAdoptionBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Adoption != nil || body.AdoptionInstruction != "" {
		t.Fatalf("un alta sin orígenes no lleva informe ni instrucción: %s", rr.Body.String())
	}
}
