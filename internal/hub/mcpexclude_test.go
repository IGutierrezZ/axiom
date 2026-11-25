package hub

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newMCPExclusionManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	return mgr
}

func TestNormalizeMCPExclusions(t *testing.T) {
	cases := []struct {
		name string
		in   map[string][]string
		want map[string][]string
	}{
		{"nil", nil, nil},
		{"solo vacíos", map[string][]string{"claude-code": {" ", ""}}, nil},
		{
			"recorta, deduplica y ordena",
			map[string][]string{"claude-code": {" work-copilot ", "claude.ai Microsoft 365", "work-copilot"}},
			map[string][]string{"claude-code": {"claude.ai Microsoft 365", "work-copilot"}},
		},
		{
			"junta claves que solo difieren en espacios",
			map[string][]string{"claude-code": {"b"}, " claude-code ": {"a", "b"}},
			map[string][]string{"claude-code": {"a", "b"}},
		},
		{"agente vacío sin nombres", map[string][]string{"": nil}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeMCPExclusions(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NormalizeMCPExclusions(%v) = %v, se esperaba %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeMCPExclusionsRejectsInvalid(t *testing.T) {
	cases := map[string]map[string][]string{
		"agente vacío con nombres": {"": {"work-copilot"}},
		"caracter de control":      {"claude-code": {"work\ncopilot"}},
		"nombre demasiado largo":   {"claude-code": {strings.Repeat("ñ", maxMCPServerNameLen+1)}},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeMCPExclusions(in); err == nil {
				t.Errorf("NormalizeMCPExclusions(%q) debía fallar", in)
			}
		})
	}
}

func TestProjectMCPExclusionsPersistAndClear(t *testing.T) {
	mgr := newMCPExclusionManager(t)
	project := t.TempDir()
	rec, err := mgr.Register(project, "p", "")
	if err != nil {
		t.Fatal(err)
	}

	if got, found, err := mgr.LookupProjectMCPExclusions(project); err != nil || got != nil || found == nil {
		t.Fatalf("sin exclusiones: %v %v %v", got, found, err)
	}

	first := map[string][]string{"claude-code": {"work-copilot", "claude.ai Microsoft 365"}}
	previous, updated, err := mgr.SetProjectMCPExclusions(rec.ID, first)
	if err != nil {
		t.Fatal(err)
	}
	if previous != nil {
		t.Errorf("la primera vez no hay exclusiones previas: %v", previous)
	}
	want := map[string][]string{"claude-code": {"claude.ai Microsoft 365", "work-copilot"}}
	if !reflect.DeepEqual(updated.MCPExclusions, want) {
		t.Errorf("entrada devuelta = %v, se esperaba %v", updated.MCPExclusions, want)
	}

	// Se resuelve también desde una subcarpeta del proyecto.
	sub := filepath.Join(project, "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, _, err := mgr.LookupProjectMCPExclusions(sub); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("desde una subcarpeta = %v (%v), se esperaba %v", got, err, want)
	}

	// Re-registrar el proyecto conserva la preferencia.
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := mgr.LookupProjectMCPExclusions(project); !reflect.DeepEqual(got, want) {
		t.Errorf("re-registrar no debe perder las exclusiones: %v", got)
	}

	// Cambiar devuelve las anteriores, para retirar las que sobran.
	previous, _, err = mgr.SetProjectMCPExclusions(project, map[string][]string{"claude-code": {"work-copilot"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(previous, want) {
		t.Errorf("previas = %v, se esperaba %v", previous, want)
	}

	// Vaciar retira el campo del JSON.
	if _, _, err := mgr.SetProjectMCPExclusions(project, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(mgr.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "mcp_exclusions") {
		t.Errorf("vaciar debe retirar el campo del JSON:\n%s", raw)
	}
}

func TestProjectMCPExclusionsLookupReturnsACopy(t *testing.T) {
	mgr := newMCPExclusionManager(t)
	project := t.TempDir()
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.SetProjectMCPExclusions(project, map[string][]string{"claude-code": {"work-copilot"}}); err != nil {
		t.Fatal(err)
	}
	got, _, err := mgr.LookupProjectMCPExclusions(project)
	if err != nil {
		t.Fatal(err)
	}
	got["claude-code"][0] = "cambiado"
	again, _, _ := mgr.LookupProjectMCPExclusions(project)
	if again["claude-code"][0] != "work-copilot" {
		t.Errorf("modificar el resultado no debe tocar lo guardado: %v", again)
	}
}

func TestProjectMCPExclusionsRejectInvalidAndUnregistered(t *testing.T) {
	mgr := newMCPExclusionManager(t)
	project := t.TempDir()
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(mgr.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := mgr.SetProjectMCPExclusions(project, map[string][]string{"claude-code": {"a\tb"}}); err == nil {
		t.Error("un nombre con caracteres de control debía fallar")
	}
	other := t.TempDir()
	if _, _, err := mgr.SetProjectMCPExclusions(other, map[string][]string{"claude-code": {"work-copilot"}}); err == nil || !strings.Contains(err.Error(), "no está registrado") {
		t.Errorf("un proyecto sin registrar debía fallar diciéndolo: %v", err)
	}

	after, err := os.ReadFile(mgr.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("un rechazo no debe modificar el hub:\nantes: %s\ndespués: %s", before, after)
	}
}
