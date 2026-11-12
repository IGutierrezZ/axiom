package livespec

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Las pruebas de ParseFeatureDoc (REQ-30.1 y el formato de REQ-30.5). Todos los textos
// son inventados.

const addedDelta = `## ADDED Requirements

### Requirement: Alta de factura (REQ-31.1)

El sistema DEBE dar de alta una factura.

#### Scenario: Alta correcta

- **DADO** un formulario válido
- **CUANDO** se envía
- **ENTONCES** se crea la factura
`

func fence(delta string) string { return "```openspec\n" + delta + "```\n" }

func featureDoc(header, body string) string {
	return "# ODD · mi-feature\n\n" + header + "## Objetivo\n\nTexto.\n\n" + body
}

func capability(name, block string) string { return "### Capacidad: " + name + "\n\n" + block + "\n" }

func parseOK(t *testing.T, text string) FeatureDoc {
	t.Helper()
	doc, err := ParseFeatureDoc("mi-feature", []byte(text))
	if err != nil {
		t.Fatalf("ParseFeatureDoc: %v", err)
	}
	return doc
}

func parseProblems(t *testing.T, text string) []Problem {
	t.Helper()
	_, err := ParseFeatureDoc("mi-feature", []byte(text))
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("error = %v, se esperaba *DocError", err)
	}
	return docErr.Problems
}

func wantProblem(t *testing.T, problems []Problem, code, subject string) {
	t.Helper()
	for _, p := range problems {
		if p.Code == code && p.Subject == subject {
			if p.Message == "" || p.Line <= 0 {
				t.Errorf("el problema %+v debe llevar mensaje y línea", p)
			}
			return
		}
	}
	t.Errorf("faltan el código %q y el sujeto %q en %+v", code, subject, problems)
}

func TestParseFeatureDocReadsOneDeltaPerCapability(t *testing.T) {
	body := "## Spec viva\n\n" + capability("facturacion", fence(addedDelta)) +
		capability("pagos", fence("## ADDED Requirements\n\n### Requirement: Cobro (REQ-32.1)\n\nCuerpo.\n"))

	doc := parseOK(t, featureDoc("", body))

	if !doc.SpecSection || doc.Kind != KindIncrement || doc.Feature != "mi-feature" {
		t.Errorf("doc = %+v", doc)
	}
	if len(doc.Capabilities) != 2 || doc.Capabilities[0].Name != "facturacion" || doc.Capabilities[1].Name != "pagos" {
		t.Fatalf("capacidades = %+v", doc.Capabilities)
	}
	first := doc.Capabilities[0]
	if len(first.Delta.Added) != 1 || first.Delta.Added[0].ID != "REQ-31.1" {
		t.Errorf("delta de facturacion = %+v", first.Delta)
	}
	if first.Text != addedDelta {
		t.Errorf("Text debe ser el contenido de la valla sin las vallas:\n%q", first.Text)
	}
	if first.Line <= 0 {
		t.Errorf("Line de la capacidad = %d", first.Line)
	}
}

func TestParseFeatureDocWithoutTheSectionIntegratesNothing(t *testing.T) {
	doc := parseOK(t, featureDoc("", "## Notas\n\nnada\n"))
	if doc.SpecSection || len(doc.Capabilities) != 0 || doc.Impact != nil || doc.Kind != KindIncrement {
		t.Errorf("doc = %+v", doc)
	}
}

func TestParseFeatureDocFenceVariants(t *testing.T) {
	tests := []struct {
		name  string
		block string
	}{
		{"valla de tres", "```openspec\n" + addedDelta + "```\n"},
		{"valla de cuatro con una valla de tres dentro", "````openspec\n" + addedDelta + "\n```go\nfunc f() {}\n```\n````\n"},
		{"tildes", "~~~openspec\n" + addedDelta + "~~~\n"},
		{"etiqueta con texto detrás", "```openspec título\n" + addedDelta + "```\n"},
		{"otra valla con otra etiqueta se ignora", "```go\nfunc f() {}\n```\n\n```openspec\n" + addedDelta + "```\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseOK(t, featureDoc("", "## Spec viva\n\n"+capability("facturacion", tt.block)))
			if len(doc.Capabilities) != 1 || len(doc.Capabilities[0].Delta.Added) != 1 {
				t.Fatalf("capacidades = %+v", doc.Capabilities)
			}
		})
	}
}

func TestParseFeatureDocHeadingsInsideTheFenceDoNotCloseTheSection(t *testing.T) {
	body := "## Spec viva\n\n" + capability("facturacion", fence(addedDelta+"\n## MODIFIED Requirements\n\n### Requirement: Otro\n\nCuerpo.\n")) + "## Cierre\n\nTexto.\n"
	doc := parseOK(t, featureDoc("", body))
	if len(doc.Capabilities) != 1 || len(doc.Capabilities[0].Delta.Modified) != 1 {
		t.Fatalf("la valla debe contener ambas secciones de la delta: %+v", doc.Capabilities)
	}
}

func TestParseFeatureDocRejectsWithTypedProblems(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		code    string
		subject string
	}{
		{"capacidad sin bloque vallado", "## Spec viva\n\n### Capacidad: facturacion\n\nSolo texto.\n", "no_fence", "facturacion"},
		{"valla sin la etiqueta openspec", "## Spec viva\n\n" + capability("facturacion", "```\n"+addedDelta+"```\n"), "no_fence", "facturacion"},
		{"valla sin cerrar", "## Spec viva\n\n### Capacidad: facturacion\n\n```openspec\n" + addedDelta, "unclosed_fence", "facturacion"},
		{"dos bloques openspec", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta)+"\n"+fence(addedDelta)), "multiple_fences", "facturacion"},
		{"delta que no se analiza", "## Spec viva\n\n" + capability("facturacion", fence("Sin operaciones.\n")), "delta_unparseable", "facturacion"},
		{"sección desconocida en la delta", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta+"\n## Notas\n\nlibre\n")), "delta_unknown_section", "facturacion"},
		{"capacidad duplicada", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta)) + capability("facturacion", fence(addedDelta)), "capability_duplicate", "facturacion"},
		{"dominio inválido", "## Spec viva\n\n" + capability("Facturación", fence(addedDelta)), "capability_invalid", "Facturación"},
		{"dominio vacío", "## Spec viva\n\n### Capacidad:\n\n" + fence(addedDelta), "capability_invalid", ""},
		{"casilla sin marcar en la delta", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta+"\n- [ ] pendiente\n")), "checkbox_in_delta", "facturacion"},
		{"casilla marcada en la delta", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta+"\n- [x] hecho\n")), "checkbox_in_delta", "facturacion"},
		{"casilla anidada en la delta", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta+"\n  - [X] hecho\n")), "checkbox_in_delta", "facturacion"},
		{"sección Spec viva repetida", "## Spec viva\n\n" + capability("facturacion", fence(addedDelta)) + "## Spec viva\n\nOtra.\n", "section_duplicate", "## Spec viva"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantProblem(t, parseProblems(t, featureDoc("", tt.body)), tt.code, tt.subject)
		})
	}
}

func TestParseFeatureDocReportsEveryProblemAtOnce(t *testing.T) {
	body := "## Spec viva\n\n### Capacidad: uno\n\nSin valla.\n\n### Capacidad: dos\n\nTampoco.\n"
	problems := parseProblems(t, featureDoc("", body))
	if len(problems) != 2 {
		t.Fatalf("problemas = %+v, se esperaban dos", problems)
	}
}

func TestParseFeatureDocDocErrorNamesEachProblem(t *testing.T) {
	_, err := ParseFeatureDoc("mi-feature", []byte(featureDoc("", "## Spec viva\n\n### Capacidad: facturacion\n\nSolo texto.\n")))
	for _, want := range []string{"mi-feature", "facturacion", "no_fence", "openspec"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("el error %v debe contener %q", err, want)
		}
	}
}

func TestParseFeatureDocKind(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   Kind
		code   string
	}{
		{"por defecto, incremento", "", KindIncrement, ""},
		{"bug", "Tipo: bug\n\n", KindBug, ""},
		{"incremento explícito", "Tipo: incremento\n\n", KindIncrement, ""},
		{"espacios alrededor", "Tipo:   bug  \n\n", KindBug, ""},
		{"valor inválido", "Tipo: tarea\n\n", "", "kind_invalid"},
		{"otra capitalización inválida", "Tipo: Bug\n\n", "", "kind_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := featureDoc(tt.header, "")
			if tt.code != "" {
				problems := parseProblems(t, text)
				if problems[0].Code != tt.code {
					t.Errorf("problemas = %+v", problems)
				}
				return
			}
			if got := parseOK(t, text).Kind; got != tt.want {
				t.Errorf("Kind = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFeatureDocOnlyTheHeaderDeclaresTheKind(t *testing.T) {
	text := featureDoc("", "## Notas\n\nTipo: bug\n\n```\nTipo: tarea\n```\n")
	if got := parseOK(t, text).Kind; got != KindIncrement {
		t.Errorf("Kind = %q: una línea Tipo: fuera de la cabecera no cuenta", got)
	}
}

func TestParseFeatureDocNormalizesLineEndings(t *testing.T) {
	text := strings.ReplaceAll(featureDoc("Tipo: bug\n\n", "## Spec viva\n\n"+capability("facturacion", fence(addedDelta))), "\n", "\r\n")
	doc := parseOK(t, text)
	if doc.Kind != KindBug || len(doc.Capabilities) != 1 || strings.Contains(doc.Capabilities[0].Text, "\r") {
		t.Errorf("doc = %+v", doc)
	}
}

func impactDoc(section string) string {
	return featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\n"+section)
}

// El contrato fijo del marcador (REQ-29.15 de carril-sdd-versatil): la línea exacta
// `Sin impacto en la spec` y una línea `Motivo:` con texto, sin rechazar el documento.
func TestImpactMarker(t *testing.T) {
	tests := []struct {
		name       string
		section    string
		wantValid  bool
		wantDefect string
		wantReason string
	}{
		{"marcador y motivo válidos", "Sin impacto en la spec\nMotivo: error tipográfico en un mensaje\n", true, "", "error tipográfico en un mensaje"},
		{"espacios iniciales y finales se ignoran", "  Sin impacto en la spec  \nMotivo:   razón válida  \n", true, "", "razón válida"},
		{"con líneas en blanco y otras líneas", "Contexto previo.\n\nSin impacto en la spec\n\nMotivo: razón\n", true, "", "razón"},
		{"punto final invalida", "Sin impacto en la spec.\nMotivo: razón\n", false, ImpactMarkerMissing, ""},
		{"otra capitalización invalida", "sin impacto en la spec\nMotivo: razón\n", false, ImpactMarkerMissing, ""},
		{"prefijo en la línea invalida", "- Sin impacto en la spec\nMotivo: razón\n", false, ImpactMarkerMissing, ""},
		{"texto delante invalida", "Nota: Sin impacto en la spec\nMotivo: razón\n", false, ImpactMarkerMissing, ""},
		{"forma antigua con dos puntos invalida", "Sin impacto en la spec: razón\n", false, ImpactMarkerMissing, ""},
		{"falta el motivo", "Sin impacto en la spec\n", false, ImpactReasonMissing, ""},
		{"motivo vacío", "Sin impacto en la spec\nMotivo:\n", false, ImpactReasonMissing, ""},
		{"motivo solo con espacios", "Sin impacto en la spec\nMotivo:    \n", false, ImpactReasonMissing, ""},
		{"sin marcador", "Texto libre.\n", false, ImpactMarkerMissing, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseOK(t, impactDoc(tt.section))
			if doc.Impact == nil {
				t.Fatal("Impact debe existir cuando la sección existe")
			}
			if doc.Impact.NoImpact != tt.wantValid || doc.Impact.Defect != tt.wantDefect || doc.Impact.Reason != tt.wantReason {
				t.Errorf("Impact = %+v, want valid=%t defect=%q reason=%q", *doc.Impact, tt.wantValid, tt.wantDefect, tt.wantReason)
			}
			if doc.Impact.Line <= 0 {
				t.Errorf("Line = %d", doc.Impact.Line)
			}
		})
	}
}

func TestImpactMarkerInsideAFenceDoesNotCount(t *testing.T) {
	doc := parseOK(t, impactDoc("```\nSin impacto en la spec\nMotivo: razón\n```\n"))
	if doc.Impact == nil || doc.Impact.NoImpact {
		t.Errorf("Impact = %+v: un marcador dentro de una valla es un ejemplo", doc.Impact)
	}
}

func TestImpactMarkerTogetherWithASpecSectionIsParsedNotRejected(t *testing.T) {
	body := "## Spec viva\n\n" + capability("facturacion", fence(addedDelta)) + "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: razón\n"
	doc := parseOK(t, featureDoc("Tipo: bug\n\n", body))
	if len(doc.Capabilities) != 1 || doc.Impact == nil || !doc.Impact.NoImpact || !doc.HasDelta() {
		t.Errorf("doc = %+v: el marcador junto con una delta no se rechaza al analizar", doc)
	}
}

func TestImpactSectionAbsentLeavesImpactNil(t *testing.T) {
	if doc := parseOK(t, featureDoc("Tipo: bug\n\n", "")); doc.Impact != nil {
		t.Errorf("Impact = %+v, se esperaba nil", doc.Impact)
	}
}

func TestBugWithoutAnInformeDeBugIsNotRejected(t *testing.T) {
	doc := parseOK(t, impactDoc("Sin impacto en la spec\nMotivo: razón\n"))
	if doc.Kind != KindBug {
		t.Errorf("Kind = %q", doc.Kind)
	}
}

func TestDocErrorProblemsKeepTheirOrder(t *testing.T) {
	_, err := ParseFeatureDoc("f", []byte("Tipo: tarea\n\n## Spec viva\n\n### Capacidad: x\n\nSin valla.\n"))
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("error = %v", err)
	}
	var codes []string
	for _, p := range docErr.Problems {
		codes = append(codes, p.Code)
	}
	if !reflect.DeepEqual(codes, []string{"kind_invalid", "no_fence"}) {
		t.Errorf("códigos = %v", codes)
	}
}
