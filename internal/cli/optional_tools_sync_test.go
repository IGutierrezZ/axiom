package cli

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// CodeGraph es opcional: el preset por defecto no lo incluye; solo entra por
// selección explícita.

func TestNormalizeInstallFlagsDefaultNoCodeGraph(t *testing.T) {
	input, err := NormalizeInstallFlags(InstallFlags{}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("NormalizeInstallFlags() error = %v", err)
	}
	if len(input.Selection.CommunityTools) != 0 {
		t.Fatalf("CodeGraph no debe venir marcado por defecto: %v", input.Selection.CommunityTools)
	}
}
