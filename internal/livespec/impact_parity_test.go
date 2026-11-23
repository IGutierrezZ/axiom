package livespec

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
)

// REQ-44.8: el marcador «Sin impacto en la spec» y su `Motivo:` los lee igual el carril ISO
// (isoevidence.ParseBugDeclaration) y el ODD (livespec.ParseFeatureDoc), con una única
// constante compartida para la etiqueta del motivo.
func TestImpactParityWithISO(t *testing.T) {
	if impactReasonPrefix != isoevidence.NoImpactReasonLabel || isoevidence.NoImpactReasonLabel != "Motivo:" {
		t.Fatalf("la etiqueta del motivo debe ser la constante compartida: %q / %q", impactReasonPrefix, isoevidence.NoImpactReasonLabel)
	}
	if impactMarkerLine != isoevidence.NoImpactMark {
		t.Fatalf("la línea del marcador debe ser la constante compartida")
	}

	const mark = "Sin impacto en la spec"
	bodies := []struct {
		name, body string
	}{
		{"completo", mark + "\nMotivo: el fallo era tipográfico\n"},
		{"sin marcador", "Motivo: el fallo era tipográfico\n"},
		{"sin Motivo", mark + "\n"},
		{"Motivo vacío", mark + "\nMotivo:\n"},
		{"Motivo vacío seguido de otro con texto", mark + "\nMotivo:\nMotivo: razón\n"},
		{"sangría con espacios", "   " + mark + "\n   Motivo: razón\n"},
		{"sangría con tabulador", "\t" + mark + "\n\tMotivo: razón\n"},
		{"espacios finales", mark + "   \nMotivo: razón   \n"},
		{"marcador y Motivo dentro de una valla", "```\n" + mark + "\nMotivo: razón\n```\n"},
		{"marcador con texto adicional", mark + ": razón\nMotivo: razón\n"},
		{"marcador con punto final", mark + ".\nMotivo: razón\n"},
		{"otra capitalización", "sin impacto en la spec\nMotivo: razón\n"},
	}
	for _, tc := range bodies {
		t.Run(tc.name, func(t *testing.T) {
			iso := isoevidence.ParseBugDeclaration("# Bug\n\n## Informe de bug\n\nObservado: x\nEsperado: y\n\n## Impacto en la spec\n\n" + tc.body)
			if iso.Kind != isoevidence.BugKindNoImpact {
				t.Fatalf("ISO: kind = %s", iso.Kind)
			}
			isoMarker := true
			for _, missing := range iso.Missing {
				if strings.Contains(missing, mark) {
					isoMarker = false
				}
			}

			doc, err := ParseFeatureDoc("mi-feature", []byte(featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\n"+tc.body)))
			if err != nil {
				t.Fatalf("ODD: %v", err)
			}
			if doc.Impact == nil {
				t.Fatal("ODD: la sección de impacto debe leerse")
			}
			oddMarker := doc.Impact.Defect != ImpactMarkerMissing

			if isoMarker != oddMarker {
				t.Errorf("marcador presente: ISO %v, ODD %v", isoMarker, oddMarker)
			}
			if iso.Valid != doc.Impact.NoImpact {
				t.Errorf("validez: ISO %v (faltan %v), ODD %v (defecto %q)", iso.Valid, iso.Missing, doc.Impact.NoImpact, doc.Impact.Defect)
			}
			if isoMarker && iso.Motivo != doc.Impact.Reason {
				t.Errorf("motivo: ISO %q, ODD %q", iso.Motivo, doc.Impact.Reason)
			}
		})
	}
}
