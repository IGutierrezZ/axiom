package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

var fuzzTemplates = []string{
	"REQ-<n>.<m>", "RF-<DOM>-<NN>", "RF-<m>", "HU-<NNN>", "A.B-<DOM>_<n>.<m>-X",
	"REQ-<DOM>-<NNNNNN>", "R-<n>-<m>", "Z9_<m>",
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
