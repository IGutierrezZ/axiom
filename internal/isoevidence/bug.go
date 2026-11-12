package isoevidence

import "strings"

// Textos literales del perfil de bug (REQ-29.15). Son un contrato compartido
// con otros carriles: no se traducen ni se reformulan.
const (
	bugReportHeading = "## Informe de bug"
	bugImpactHeading = "## Impacto en la spec"
	// NoImpactMark es la línea literal que declara un bug sin impacto en la spec.
	NoImpactMark = "Sin impacto en la spec"
)

// BugKind es la variante de bug sin delta que declara un proposal.md.
type BugKind string

const (
	// BugKindNone: el proposal.md no declara ninguna de las dos secciones.
	BugKindNone BugKind = "none"
	// BugKindExistingRequirement: informe de bug sin sección de impacto.
	BugKindExistingRequirement BugKind = "existing_requirement"
	// BugKindNoImpact: sección `## Impacto en la spec` presente.
	BugKindNoImpact BugKind = "no_impact"
)

// BugDeclaration es el resultado de leer las secciones de bug de un
// proposal.md.
type BugDeclaration struct {
	Kind  BugKind
	Valid bool
	// Missing lista lo que falta, en orden de aparición esperado: primero lo
	// del informe y después lo de la sección de impacto.
	Missing []string
	// Motivo es el texto de la línea `Motivo:` (vacío si falta).
	Motivo string

	reportMissing []string
	impactMissing []string
}

// ParseBugDeclaration lee `## Informe de bug` y `## Impacto en la spec` de un
// proposal.md según design §5.6. Los encabezados cuentan solo como nivel 2
// exacto y fuera de vallas; cada sección termina en el siguiente `#` o `##`.
// Con delta las secciones se ignoran, pero eso lo decide el llamador: esta
// función solo analiza el texto.
func ParseBugDeclaration(proposal string) BugDeclaration {
	report, hasReport := sectionLines(proposal, bugReportHeading)
	impact, hasImpact := sectionLines(proposal, bugImpactHeading)

	d := BugDeclaration{Kind: BugKindNone}
	switch {
	case hasImpact:
		d.Kind = BugKindNoImpact
	case hasReport:
		d.Kind = BugKindExistingRequirement
	default:
		return d
	}

	if !hasReport {
		d.reportMissing = append(d.reportMissing, "la sección `"+bugReportHeading+"`")
	} else {
		d.reportMissing = append(d.reportMissing, missingLabels(report, "Observado:", "Esperado:")...)
	}

	if hasImpact {
		if !hasMark(impact) {
			d.impactMissing = append(d.impactMissing, "la línea literal `"+NoImpactMark+"`")
		}
		value, found := labelValue(impact, "Motivo:")
		switch {
		case !found:
			d.impactMissing = append(d.impactMissing, "la línea `Motivo:`")
		case value == "":
			d.impactMissing = append(d.impactMissing, "la línea `Motivo:` con texto (está vacía)")
		default:
			d.Motivo = value
		}
	}

	d.Missing = append(append([]string{}, d.reportMissing...), d.impactMissing...)
	d.Valid = len(d.Missing) == 0
	return d
}

// Findings devuelve un hallazgo por cada cosa que falta, con el token que
// corresponde a su sección (D9). Una declaración válida o ausente no produce
// ninguno.
func (d BugDeclaration) Findings() []Finding {
	var findings []Finding
	for _, missing := range d.reportMissing {
		findings = append(findings, Finding{
			Code:         CodeBugReportIncomplete,
			Stage:        StageSpecs,
			Subject:      bugReportHeading,
			Detail:       "el informe de bug de proposal.md está incompleto: falta " + missing,
			Continuation: "completa `Observado:` y `Esperado:` con texto, o añade `" + bugReportHeading + "` en proposal.md",
		})
	}
	for _, missing := range d.impactMissing {
		findings = append(findings, Finding{
			Code:         CodeBugNoImpactIncomplete,
			Stage:        StageSpecs,
			Subject:      bugImpactHeading,
			Detail:       "la sección `" + bugImpactHeading + "` de proposal.md está incompleta: falta " + missing,
			Continuation: "añade la línea `" + NoImpactMark + "` y un `Motivo:` no vacío en `" + bugImpactHeading + "`",
		})
	}
	return findings
}

// sectionLines devuelve las líneas, fuera de vallas, de la primera sección cuyo
// encabezado de nivel 2 es exactamente heading, hasta el siguiente encabezado
// de nivel 1 o 2.
func sectionLines(text, heading string) ([]string, bool) {
	var lines []string
	in, found := false, false
	for _, line := range splitLines(text) {
		if line.Fenced {
			continue
		}
		if strings.HasPrefix(line.Text, "# ") || strings.HasPrefix(line.Text, "## ") {
			if in {
				break
			}
			if !found && strings.TrimRight(line.Text, " \t") == heading {
				in, found = true, true
			}
			continue
		}
		if in {
			lines = append(lines, line.Text)
		}
	}
	return lines, found
}

// labelValue busca la primera línea que, sin el espacio inicial, empieza por la
// etiqueta, y devuelve su valor recortado.
func labelValue(lines []string, label string) (string, bool) {
	for _, l := range lines {
		trimmed := strings.TrimLeft(l, " \t")
		if strings.HasPrefix(trimmed, label) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, label)), true
		}
	}
	return "", false
}

func missingLabels(lines []string, labels ...string) []string {
	var missing []string
	for _, label := range labels {
		value, found := labelValue(lines, label)
		switch {
		case !found:
			missing = append(missing, "la línea `"+label+"`")
		case value == "":
			missing = append(missing, "la línea `"+label+"` con texto (está vacía)")
		}
	}
	return missing
}

// hasMark compara de forma exacta, sensible a mayúsculas y sin más texto, la
// línea literal de la marca, ignorando solo el espacio inicial y final.
func hasMark(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == NoImpactMark {
			return true
		}
	}
	return false
}
