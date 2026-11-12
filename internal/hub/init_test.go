package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Las pruebas del informe de adopción del alta de proyecto (REQ-30.27): el alta devuelve el
// informe y la instrucción para lanzar la adopción desde el agente, sin ejecutar nada ni crear
// el cambio de adopción.

func TestInitReturnsTheAdoptionReportAndInstruction(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proyecto")
	writeHubFile(t, root, "docs/requisitos.md", adoptionRequisitos)
	writeHubFile(t, root, "ARCHITECTURE.md", "# Arquitectura\n")

	res, err := NewInitializer(nil, nil).Init(InitOptions{Path: root, Name: "Ejemplo"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res.Adoption == nil || res.Adoption.SourceCount != 1 || res.Adoption.UnitCount != 3 || res.Adoption.Sources[0].Path != "docs/requisitos.md" {
		t.Fatalf("informe de adopción = %+v", res.Adoption)
	}
	if len(res.Adoption.ContextCandidates) != 1 || res.Adoption.ContextCandidates[0].Path != "ARCHITECTURE.md" {
		t.Fatalf("candidatos = %+v", res.Adoption.ContextCandidates)
	}
	for _, want := range []string{"axiom archive adopt scan --path", root, "--change adopcion-openspec", "sdd-spec"} {
		if !strings.Contains(res.AdoptionInstruction, want) {
			t.Errorf("la instrucción debe contener %q: %s", want, res.AdoptionInstruction)
		}
	}
	// El alta no ejecuta la adopción ni crea el cambio.
	for _, rel := range []string{"openspec/changes/adopcion-openspec", "openspec/changes/adopcion-openspec/adoption-inventory.yaml"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("el alta no debe crear %s (%v)", rel, err)
		}
	}
}

func TestInitWithoutSourcesHasNoAdoptionInstruction(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vacio")
	res, err := NewInitializer(nil, nil).Init(InitOptions{Path: root, Name: "Vacio"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res.Adoption != nil || res.AdoptionInstruction != "" {
		t.Fatalf("sin orígenes ni candidatos no hay informe ni instrucción: %+v %q", res.Adoption, res.AdoptionInstruction)
	}
}

func TestInitWithOnlyContextCandidatesShowsTheReportButNoInstruction(t *testing.T) {
	root := filepath.Join(t.TempDir(), "solo-contexto")
	writeHubFile(t, root, "ARCHITECTURE.md", "# Arquitectura\n")
	res, err := NewInitializer(nil, nil).Init(InitOptions{Path: root, Name: "Contexto"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Adoption == nil || res.Adoption.SourceCount != 0 || len(res.Adoption.ContextCandidates) != 1 {
		t.Fatalf("informe = %+v", res.Adoption)
	}
	if res.AdoptionInstruction != "" {
		t.Fatalf("sin orígenes de requisitos no hay nada que adoptar: %q", res.AdoptionInstruction)
	}
}

func TestInitMultirepoAdoptionCoversTheSpecsRepository(t *testing.T) {
	master := filepath.Join(t.TempDir(), "maestra")
	writeHubFile(t, master, "repo-specs/docs/requisitos.md", adoptionRequisitos)
	writeHubFile(t, master, "docs/prd-maestra.md", adoptionRequisitos)

	res, err := NewInitializer(nil, nil).Init(InitOptions{
		Path: master, Name: "Multi", Topology: "multirepo", SpecsRepository: "repo-specs", Roles: multirepoRoles(),
	})
	if err != nil {
		t.Fatalf("Init multirepo: %v", err)
	}
	if res.Adoption == nil || res.Adoption.SourceCount != 1 || res.Adoption.Sources[0].Path != "docs/requisitos.md" {
		t.Fatalf("el informe cubre el repositorio de especificaciones: %+v", res.Adoption)
	}
	if !strings.Contains(res.AdoptionInstruction, master) {
		t.Errorf("la instrucción usa la carpeta maestra, que resuelve al repositorio de especificaciones: %s", res.AdoptionInstruction)
	}
}
