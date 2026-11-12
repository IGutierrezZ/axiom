package workspace

import (
	"strings"
	"testing"
)

// Pin del manual de inicio (REQ-30.31, REQ-30.32): la spec viva es la de OpenSpec,
// la adopción y la integración ODD tienen su verbo, y los verbos `axiom odd` que
// retiró INC-20 no reaparecen.
func TestDocsPinManualDeInicio(t *testing.T) {
	manual := readRepoFile(t, "docs/manual-de-inicio.md")
	if strings.Contains(manual, "axiom odd") {
		t.Error("docs/manual-de-inicio.md no debe mencionar `axiom odd`: sus verbos los retiró INC-20")
	}
	for _, want := range []string{
		"axiom archive adopt scan",
		"axiom archive adopt check",
		"axiom archive integrate --odd",
		"axiom archive context",
		"## Spec viva",
		"## Impacto en la spec",
		"Sin impacto en la spec",
	} {
		if !strings.Contains(manual, want) {
			t.Errorf("docs/manual-de-inicio.md debe citar %q", want)
		}
	}
}

// REQ-30.32 (parte documental): el reparto entre axiom.yaml y openspec/config.yaml.
func TestDocsPinOpenspecConfigBoundary(t *testing.T) {
	doc := readRepoFile(t, "docs/openspec-config.md")
	for _, want := range []string{
		"`technical_context` lives in `axiom.yaml`",
		"`rules.<phase>`, `strict_tdd` and `artifact_store` stay in `openspec/config.yaml`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/openspec-config.md debe contener la frase %q", want)
		}
	}
}
