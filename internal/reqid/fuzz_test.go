package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

var fuzzTemplates = []string{
	"REQ-<n>.<m>", "RF-<DOM>-<NN>", "RF-<m>", "HU-<NNN>", "A.B-<DOM>_<n>.<m>-X",
	"REQ-<DOM>-<NNNNNN>", "R-<n>-<m>", "Z9_<m>",
	"RF-<DOM>_<n>-<m>.V", "RF-<DOM>.<n>-<m>_V", "RF-<DOM>-<n>_<m>.V",
	"RF-<DOM>.<n>-<m>V", "RF-<DOM>_<n>.<m>V",
}

func FuzzParse(f *testing.F) {
	for _, s := range fuzzTemplates {
		f.Add(s)
	}
	f.Add("")
	f.Add("RF-<DOM><NN>")
	f.Add("<<<>>>")
	f.Fuzz(func(t *testing.T, template string) {
		p, err := reqid.Parse(template)
		if err != nil {
			return
		}
		// Una plantilla aceptada es coherente consigo misma.
		if p.Template() != template {
			t.Fatalf("Template() = %q, plantilla %q", p.Template(), template)
		}
		_ = p.Compact()
		_ = p.SeriesCompact()
		p.Scan(template)
	})
}

func FuzzSynthetic(f *testing.F) {
	f.Add("facturacion", 3)
	f.Add("USU", 1)
	f.Add("", 0)
	f.Add("A1", -5)
	f.Add("sdd-iso", 12)
	f.Fuzz(func(t *testing.T, domain string, ordinal int) {
		id := reqid.Synthetic(domain, ordinal)
		for _, template := range fuzzTemplates {
			p, err := reqid.Parse(template)
			if err != nil {
				t.Fatal(err)
			}
			if p.Valid(id) {
				t.Fatalf("el sintético %q es válido para %q", id, template)
			}
			if got := p.InTitle("x (" + id + ")"); got != "" {
				t.Fatalf("InTitle devolvió el sintético %q para %q", got, template)
			}
		}
	})
}

// FuzzProvisionalIsNeverAnIdentifier fija REQ-47.1: con cualquier plantilla aceptada y
// cualquier texto, un token provisional completo no es un identificador válido, no sale
// de InTitle ni de TrailingCandidate y Scan no lo devuelve como candidato.
func FuzzProvisionalIsNeverAnIdentifier(f *testing.F) {
	for _, s := range fuzzTemplates {
		f.Add(s, "prov:30:1")
		f.Add(s, "prov:RF-3:1")
		f.Add(s, "prov:USU-b:2")
	}
	f.Add("REQ-<n>.<m>", "prov::1")
	f.Fuzz(func(t *testing.T, template, token string) {
		p, err := reqid.Parse(template)
		if err != nil {
			return
		}
		spans := reqid.ScanProvisional(token)
		if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(token) {
			return // no es un token provisional completo
		}
		if p.Valid(token) {
			t.Fatalf("Valid(%q) con %q", token, template)
		}
		title := "Requisito (" + token + ")"
		if id := p.InTitle(title); id != "" {
			t.Fatalf("InTitle(%q) = %q con %q", title, id, template)
		}
		if _, id, ok := p.TrailingCandidate(title); ok {
			t.Fatalf("TrailingCandidate(%q) = %q con %q", title, id, template)
		}
		valid, malformed := p.Scan("Cubre: " + token + ", " + title)
		if len(valid)+len(malformed) != 0 {
			t.Fatalf("Scan devolvió candidatos %v %v de %q con %q", valid, malformed, token, template)
		}
	})
}
