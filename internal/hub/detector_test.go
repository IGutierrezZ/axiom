package hub

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Las pruebas del informe de adopción del detector (REQ-30.26): reutiliza el inventario de
// `axiom archive adopt scan`, sin escribir ni modificar axiom.yaml.

const adoptionRequisitos = "# Requisitos\n\nIntroducción.\n\n## RF-001 Alta de usuario\n\nEl sistema permite el alta.\n\n## RF-002 Baja de usuario\n\nEl sistema permite la baja.\n"

func writeHubFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hubTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			files[rel] = "<dir>"
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestDetectorProposesMigrationSourcesWithoutWriting(t *testing.T) {
	root := t.TempDir()
	writeHubFile(t, root, "docs/requisitos.md", adoptionRequisitos)
	before := hubTreeSnapshot(t, root)

	detection, err := NewDetector().Detect(root)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if detection.Adoption == nil {
		t.Fatal("el detector debe rellenar el informe de adopción")
	}
	sum := detection.Adoption
	if sum.SourceCount != 1 || sum.UnitCount != 3 || len(sum.Sources) != 1 || sum.Sources[0].Path != "docs/requisitos.md" || sum.Sources[0].Units != 3 {
		t.Fatalf("informe de adopción = %+v", sum)
	}
	if after := hubTreeSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("el detector no escribe ningún fichero:\nantes %v\ndespués %v", before, after)
	}
}

func TestDetectorProposesTechnicalContextCandidatesWithoutTouchingAxiomYaml(t *testing.T) {
	root := t.TempDir()
	writeHubFile(t, root, "ARCHITECTURE.md", "# Arquitectura\n")
	writeHubFile(t, root, "docs/adr/0001-decision.md", "# ADR\n")
	writeHubFile(t, root, "axiom.yaml", "workspace:\n  name: ejemplo\n")
	before := hubTreeSnapshot(t, root)

	detection, err := NewDetector().Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, c := range detection.Adoption.ContextCandidates {
		modes[c.Path] = c.Mode
	}
	if modes["ARCHITECTURE.md"] != "maintained" || modes["docs/adr/"] != "reference" {
		t.Fatalf("candidatos = %v", modes)
	}
	if after := hubTreeSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("axiom.yaml y el resto del árbol no se modifican")
	}
}

func TestDetectorEmptyAdoptionReportWithoutError(t *testing.T) {
	root := t.TempDir()
	detection, err := NewDetector().Detect(root)
	if err != nil {
		t.Fatalf("un directorio sin orígenes no es un error: %v", err)
	}
	if detection.Adoption == nil || !detection.Adoption.Empty() || detection.Adoption.Sources == nil {
		t.Fatalf("el informe debe existir y estar vacío: %+v", detection.Adoption)
	}
}

func TestDetectorAdoptionIgnoresOpenSpec(t *testing.T) {
	root := t.TempDir()
	writeHubFile(t, root, "openspec/specs/pagos/spec.md", "# RF-001\n")
	detection, err := NewDetector().Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !detection.Adoption.Empty() {
		t.Fatalf("openspec/ no cuenta como origen: %+v", detection.Adoption)
	}
}
