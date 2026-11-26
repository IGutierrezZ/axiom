package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectPersonaAxiomContent(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}

	content, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	// Prohibiciones explícitas: no voseo ni regionalismos rioplatenses
	for _, forbidden := range []string{"Rioplatense", "voseo", "tenés", "podés", "hacé"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("PersonaAxiom contains forbidden term %q; content:\n%s", forbidden, text)
		}
	}

	// Verificación de directivas de Axiom
	for _, want := range []string{
		"Castellano de España (peninsular)",
		"all technical artifacts",
		"Spanish (castellano peninsular)",
		"English is strictly preserved for code syntax and source identifiers",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("PersonaAxiom missing expected directive %q; content:\n%s", want, text)
		}
	}
}
