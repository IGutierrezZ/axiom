package reqid_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestParseNotation(t *testing.T) {
	cases := []struct {
		template, compact, series string
	}{
		{"REQ-<n>.<m>", "REQ-n.m", "REQ-n.*"},
		{"RF-<DOM>-<NN>", "RF-DOM-NN", "RF-DOM-*"},
		{"RF-<m>", "RF-m", "RF-*"},
		{"HU-<NNN>", "HU-NNN", "HU-*"},
		{"RF-<DOM>-<NNNNNN>", "RF-DOM-NNNNNN", "RF-DOM-*"},
		{"A.B-<n>.<m>-X", "A.B-n.m-X", "A.B-n.*-X"},
	}
	for _, tc := range cases {
		t.Run(tc.template, func(t *testing.T) {
			p, err := reqid.Parse(tc.template)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.template, err)
			}
			if got := p.Template(); got != tc.template {
				t.Errorf("Template() = %q, want %q", got, tc.template)
			}
			if got := p.Compact(); got != tc.compact {
				t.Errorf("Compact() = %q, want %q", got, tc.compact)
			}
			if got := p.SeriesCompact(); got != tc.series {
				t.Errorf("SeriesCompact() = %q, want %q", got, tc.series)
			}
		})
	}
}

func TestParseDefaultIsZeroValue(t *testing.T) {
	var zero reqid.Pattern
	if zero != reqid.Default() {
		t.Fatal("el valor cero debe ser Default()")
	}
	p, err := reqid.Parse("REQ-<n>.<m>")
	if err != nil || p != reqid.Default() {
		t.Fatalf("Parse del defecto = %v, %v", p, err)
	}
	if zero.Template() != "REQ-<n>.<m>" || zero.Compact() != "REQ-n.m" || zero.SeriesCompact() != "REQ-n.*" {
		t.Fatalf("notaciones del valor cero: %q %q %q", zero.Template(), zero.Compact(), zero.SeriesCompact())
	}
	a, _ := reqid.Parse("RF-<DOM>-<NN>")
	b, _ := reqid.Parse("RF-<DOM>-<NN>")
	if a != b {
		t.Fatal("dos Parse de la misma plantilla deben ser iguales")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		template, reason string
	}{
		{"", "vacía"},
		{"RF-", "falta el marcador de número"},
		{"RF-<m>-<m>", "un único marcador de número"},
		{"RF-<XYZ>-<m>", "<XYZ>"},
		{"RF-<N>", "<N>"},
		{"RF-<NNNNNNN>", "<NNNNNNN>"},
		{"RF-<DOM-<m>", "sin cerrar"},
		{"RF-<m>>", "`>` suelto"},
		{"RF-<n>-<n>-<m>", "<n> solo puede aparecer una vez"},
		{"RF-<DOM>-<DOM>-<m>", "<DOM> solo puede aparecer una vez"},
		{"<DOM>-<m>", "literal inicial"},
		{"RF<DOM>-<m>", "literal inicial"},
		{"RF-<DOM><NN>", "seguidos"},
		{"RF-<m>0", "alfabeto"},
		{"RF-A<DOM>-<m>", "alfabeto"},
		{"rf-<m>", "literal inicial"},
		{"RF <m>", "literal inicial"},
		{"RF-(<m>)", "carácter"},
		{"RF-<m>,", "carácter"},
		{"RF-" + strings.Repeat("A", 60) + "<m>", "64 caracteres"},
	}
	for _, tc := range cases {
		t.Run(tc.template, func(t *testing.T) {
			_, err := reqid.Parse(tc.template)
			if err == nil {
				t.Fatalf("Parse(%q) debía fallar", tc.template)
			}
			var te *reqid.TemplateError
			if !errors.As(err, &te) {
				t.Fatalf("error %T, quería *reqid.TemplateError", err)
			}
			if te.Template != tc.template {
				t.Errorf("Template = %q, quería %q", te.Template, tc.template)
			}
			if !strings.Contains(te.Reason, tc.reason) {
				t.Errorf("Reason = %q, debía contener %q", te.Reason, tc.reason)
			}
			want := "la plantilla \"" + tc.template + "\" no es válida: " + te.Reason
			if err.Error() != want {
				t.Errorf("Error() = %q, quería %q", err.Error(), want)
			}
		})
	}
}
