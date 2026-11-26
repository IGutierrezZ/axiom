package communitytool

import (
	"strings"
	"testing"
)

// Un hijo de Pi con los bloques que escribió el nombre anterior acaba con un
// único par de bloques con el marcador actual, y repetir el render no cambia nada.
func TestRenderPiChildReplacesLegacyBlocksWithAxiomBlocks(t *testing.T) {
	legacy := "---\nname: child\ntools: read, bash\n---\nBody\n\n" +
		legacyPiCodeGraphToolMarker + "\nold tool text\n" + legacyPiCodeGraphEndMarker + "\n\n" +
		legacyPiCodeGraphGuidanceMarker + "\nold guidance\n" + legacyPiCodeGraphEndMarker + "\n"

	first, err := renderPiChild(legacy, []string{"read", "bash", "mcp"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(first, "gentle-ai:") || strings.Contains(first, "old tool text") || strings.Contains(first, "old guidance") {
		t.Fatalf("legacy block survived:\n%s", first)
	}
	if strings.Count(first, piCodeGraphToolMarker) != 1 || strings.Count(first, piCodeGraphGuidanceMarker) != 1 || strings.Count(first, piCodeGraphEndMarker) != 2 {
		t.Fatalf("want one axiom tool block and one axiom guidance block:\n%s", first)
	}
	second, err := renderPiChild(first, []string{"read", "bash", "mcp"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("render is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestStripPiCodeGraphBlocksRejectsAnUnclosedLegacyBlock(t *testing.T) {
	if _, err := stripPiCodeGraphBlocks("x\n" + legacyPiCodeGraphToolMarker + "\nno end"); err == nil {
		t.Fatal("an unclosed legacy block must be reported, not silently kept")
	}
}

// La huella de doctor reconoce la sección que escribe Axiom y también la del
// nombre anterior (que la siguiente sincronización sustituye).
func TestHasManagedCodeGraphSectionRecognizesAxiomAndLegacyMarkers(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		want    bool
	}{
		"axiom":  {"intro\n<!-- axiom:" + codeGraphGuidanceSectionID + " -->\nx", true},
		"legacy": {"intro\n<!-- gentle-ai:" + codeGraphGuidanceSectionID + " -->\nx", true},
		"none":   {"intro only", false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := hasManagedCodeGraphSection(tc.content); got != tc.want {
				t.Fatalf("hasManagedCodeGraphSection = %t, want %t", got, tc.want)
			}
		})
	}
}
