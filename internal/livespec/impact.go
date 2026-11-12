package livespec

import "fmt"

// impactSubject es el sujeto que nombran los problemas de impacto: la sección del documento.
const impactSubject = "## " + impactSectionHeading

// CheckImpact aplica la obligación de impacto de los bugs ODD (REQ-30.5). Un documento
// `Tipo: bug` DEBE aportar una `## Spec viva` con al menos una capacidad (se evalúa como
// un cambio con delta y no necesita marcador) o el marcador válido de `## Impacto en la
// spec`: la línea exacta `Sin impacto en la spec` y una línea `Motivo:` con texto. Un
// marcador junto con una delta no vacía se ignora. En un documento que no es un bug
// la sección y el marcador son opcionales y nunca producen un rechazo. El rechazo es un
// *DocError que nombra la sección y la regla incumplida.
func CheckImpact(doc FeatureDoc) error {
	if doc.Kind != KindBug || doc.HasDelta() {
		return nil
	}
	problem := Problem{Subject: impactSubject, Line: 1}
	switch {
	case doc.Impact == nil:
		problem.Code = "impact_missing"
		problem.Message = "el documento es un bug (`Tipo: bug`) y no declara su impacto en la spec; añade una `## Spec viva` con una `### Capacidad:` y su delta, o una sección `## Impacto en la spec` con la línea `Sin impacto en la spec` y una línea `Motivo: <una línea>`"
	case doc.Impact.Defect == ImpactReasonMissing:
		problem.Code, problem.Line = "impact_reason_missing", doc.Impact.Line
		problem.Message = "la sección `## Impacto en la spec` tiene la línea `Sin impacto en la spec` pero falta la línea `Motivo: <una línea>` o está vacía; escribe el motivo en una línea que empiece por `Motivo:`"
	case doc.Impact.Defect != "":
		problem.Code, problem.Line = "impact_marker_invalid", doc.Impact.Line
		problem.Message = fmt.Sprintf("la sección `## Impacto en la spec` no contiene la línea exacta `%s` (se compara de forma literal y con mayúsculas, sin punto final, sin prefijo ni texto detrás) y el bug no aporta una `## Spec viva` con delta; corrige la línea del marcador y añade `Motivo: <una línea>`, o aporta la delta", impactMarkerLine)
	default:
		return nil
	}
	return &DocError{Feature: doc.Feature, Problems: []Problem{problem}}
}
