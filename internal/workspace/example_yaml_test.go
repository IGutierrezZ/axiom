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

// REQ-29.1 y REQ-30.28 (R7): la plantilla axiom.example.yaml documenta el bloque
// `sdd:` y, una vez descomentado, es válido y declara ISO.
func TestExampleYAMLDocumentsTheSDDBlock(t *testing.T) {
	var block []string
	for _, line := range strings.Split(strings.ReplaceAll(readRepoFile(t, "axiom.example.yaml"), "\r\n", "\n"), "\n") {
		if len(block) == 0 {
			if line == "# sdd:" {
				block = append(block, "sdd:")
			}
			continue
		}
		if !strings.HasPrefix(line, "#   ") {
			break
		}
		block = append(block, strings.TrimPrefix(line, "# "))
	}
	if len(block) < 2 {
		t.Fatalf("axiom.example.yaml debe documentar un bloque `# sdd:` con su clave, y se halló %q", block)
	}
	_, baseErr, declarations := parseDeclarations(t, strings.Join(block, "\n")+"\n")
	if baseErr != nil || declarations.SDDErr != nil {
		t.Fatalf("el bloque de la plantilla no es válido: base=%v sdd=%v\n%s", baseErr, declarations.SDDErr, block)
	}
	if declarations.SDD == nil || !declarations.SDD.ISO {
		t.Errorf("la plantilla debe documentar `sdd.iso: true`: %+v", declarations.SDD)
	}
}
