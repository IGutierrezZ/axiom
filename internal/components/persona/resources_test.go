package persona_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
)

func TestResourcePlanOutputStylePaths(t *testing.T) {
	dir := t.TempDir()
	axiom := filepath.Join(dir, "axiom.md")

	got := persona.ResourcePlanFor().OutputStylePaths(dir)
	want := persona.OutputStylePaths{Write: axiom, Backup: []string{axiom}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResourcePlanFor().OutputStylePaths() = %#v, want %#v", got, want)
	}
	if empty := persona.ResourcePlanFor().OutputStylePaths(""); !reflect.DeepEqual(empty, persona.OutputStylePaths{}) {
		t.Fatalf("OutputStylePaths(\"\") = %#v, want vacío", empty)
	}
}

func TestManagedOutputStylesOnlyAxiom(t *testing.T) {
	styles := persona.ManagedOutputStyles()
	if len(styles) != 1 || styles[0].Name != "Axiom" || styles[0].File != "axiom.md" {
		t.Fatalf("ManagedOutputStyles() = %#v, want solo el estilo de Axiom", styles)
	}
}
