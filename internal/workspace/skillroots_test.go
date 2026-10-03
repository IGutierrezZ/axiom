package workspace

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCleanSkillRoot(t *testing.T) {
	valid := []struct {
		name string
		in   string
		want string
	}{
		{"ruta simple", "internal/assets/skills", "internal/assets/skills"},
		{"un solo nivel", "docs", "docs"},
		{"barra final", "docs/skills/", "docs/skills"},
		{"espacios alrededor", "  docs/skills  ", "docs/skills"},
		{"prefijo ./", "./docs/skills", "docs/skills"},
		{"separadores de Windows", `internal\assets\skills`, "internal/assets/skills"},
		{"retroceso que se queda dentro", "a/../b", "b"},
		{"nombre que empieza por dos puntos", "..hidden/skills", "..hidden/skills"},
	}
	for _, tt := range valid {
		t.Run("valida/"+tt.name, func(t *testing.T) {
			got, ok := CleanSkillRoot(tt.in)
			if !ok {
				t.Fatalf("CleanSkillRoot(%q) rechazada, se esperaba válida", tt.in)
			}
			if want := filepath.FromSlash(tt.want); got != want {
				t.Errorf("CleanSkillRoot(%q) = %q, esperado %q", tt.in, got, want)
			}
		})
	}

	invalid := []struct {
		name string
		in   string
	}{
		{"vacía", ""},
		{"solo espacios", "   "},
		{"el propio proyecto", "."},
		{"el propio proyecto con barra", "./"},
		{"vuelve al propio proyecto", "a/.."},
		{"padre", ".."},
		{"sale del proyecto", "../x"},
		{"sale tras descender", "a/../../x"},
		{"sale con separador de Windows", `..\x`},
		{"absoluta POSIX", "/opt/skills"},
		{"absoluta con separador de Windows", `\skills`},
		{"unidad de Windows", `C:\skills`},
		{"unidad de Windows con barra", "c:/skills"},
		{"unidad de Windows sin barra", "C:skills"},
		{"UNC", `\\server\share\skills`},
	}
	for _, tt := range invalid {
		t.Run("invalida/"+tt.name, func(t *testing.T) {
			if got, ok := CleanSkillRoot(tt.in); ok {
				t.Errorf("CleanSkillRoot(%q) = %q, se esperaba rechazada", tt.in, got)
			}
		})
	}
}

func TestValidSkillRootsKeepsDeclarationOrderAndDropsInvalid(t *testing.T) {
	section := WorkspaceSection{SkillRoots: []string{
		"internal/assets/skills",
		"/abs",
		"",
		"../fuera",
		".",
		"docs/skills",
	}}

	got := section.ValidSkillRoots()
	want := []string{filepath.FromSlash("internal/assets/skills"), filepath.FromSlash("docs/skills")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidSkillRoots() = %v, esperado %v", got, want)
	}

	if got := (WorkspaceSection{}).ValidSkillRoots(); len(got) != 0 {
		t.Errorf("sin raíces declaradas se esperaba una lista vacía, obtenido %v", got)
	}
}

func TestSkillRootsRoundTrip(t *testing.T) {
	original := &WorkspaceConfig{
		Workspace: WorkspaceSection{
			Name:            "RoundTrip",
			Topology:        TopologyMonorepoEmbedded,
			SpecsRepository: ".",
			SkillRoots:      []string{"internal/assets/skills", "docs/skills"},
		},
		Roles: map[string]RoleConfig{
			"main": {Name: "Main", Repositories: []RepositoryEntry{{Path: "."}}},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("error al serializar: %v", err)
	}
	if !strings.Contains(string(data), "skill_roots:") {
		t.Fatalf("la clave debe serializarse como 'workspace.skill_roots', obtenido:\n%s", data)
	}

	parsed, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("error al releer lo serializado: %v", err)
	}
	if !reflect.DeepEqual(parsed.Workspace.SkillRoots, original.Workspace.SkillRoots) {
		t.Errorf("skill_roots tras el ciclo = %v, esperado %v", parsed.Workspace.SkillRoots, original.Workspace.SkillRoots)
	}

	// Sin raíces declaradas la clave no aparece: los axiom.yaml existentes no cambian.
	original.Workspace.SkillRoots = nil
	data, err = yaml.Marshal(original)
	if err != nil {
		t.Fatalf("error al serializar: %v", err)
	}
	if strings.Contains(string(data), "skill_roots") {
		t.Errorf("sin raíces no debe serializarse la clave, obtenido:\n%s", data)
	}
}
