package assets

import (
	"strings"
	"testing"
)

const (
	independentReviewTemplateTag = "{{AXIOM_SDD_SECTION:SDD Independent Review}}"
	blockGatesTemplateTag        = "{{AXIOM_SDD_SECTION:SDD Change Kickoff and Block Gates}}"
)

var independentReviewRuntimes = []string{
	"antigravity", "claude", "codex", "cursor", "gemini", "generic", "opencode", "qwen",
}

// TestIndependentReviewBoundaryTemplateParity: toda plantilla que proyecta las puertas proyecta una vez la sección nueva,
// detrás de ellas.
func TestIndependentReviewBoundaryTemplateParity(t *testing.T) {
	for _, runtime := range independentReviewRuntimes {
		path := runtime + "/sdd-orchestrator.md"
		content := MustRead(path)
		gates := strings.Index(content, blockGatesTemplateTag)
		if gates < 0 {
			continue
		}
		if got := strings.Count(content, independentReviewTemplateTag); got != 1 {
			t.Errorf("%s: la sección SDD Independent Review se proyecta %d veces, se esperaba 1", path, got)
			continue
		}
		if strings.Index(content, independentReviewTemplateTag) < gates {
			t.Errorf("%s: la sección SDD Independent Review debe ir detrás de las puertas", path)
		}
	}
}
