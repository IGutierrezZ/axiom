package workspace

import (
	"strings"
	"testing"
)

// REQ-30.32 (parte documental): la plantilla axiom.example.yaml documenta el bloque
// `technical_context:` con sus dos modos y, una vez descomentado, es válido. No
// documenta `baseline:`, que se retira en favor de la spec viva de OpenSpec.
func TestExampleYAMLDocumentsTechnicalContext(t *testing.T) {
	block := uncommentedExampleBlock(t, "technical_context")
	_, baseErr, declarations := parseDeclarations(t, block)
	if baseErr != nil || declarations.TechnicalContextErr != nil {
		t.Fatalf("el bloque de la plantilla no es válido: base=%v technical_context=%v\n%s", baseErr, declarations.TechnicalContextErr, block)
	}
	tc := declarations.TechnicalContext
	if tc == nil || tc.Maintained == nil || tc.Reference == nil {
		t.Fatalf("la plantilla debe documentar los modos maintained y reference: %+v", tc)
	}
	if tc.Maintained.Entry == "" || len(tc.Maintained.Paths) == 0 || len(tc.Reference.Paths) == 0 {
		t.Errorf("los modos de la plantilla deben llevar rutas (y entry en maintained): %+v", tc)
	}
}

func TestExampleYAMLDoesNotDocumentBaseline(t *testing.T) {
	raw := readRepoFile(t, "axiom.example.yaml")
	for number, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(strings.TrimLeft(line, "# ")), "baseline:") {
			t.Errorf("axiom.example.yaml:%d documenta `baseline:`, que se retira: %s", number+1, line)
		}
	}
}
