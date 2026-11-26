package reqid_test

import (
	"errors"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestParseProvisionalGrammar(t *testing.T) {
	cases := []struct {
		name, template, token string
		wantErr               string // "" = válido; si no, el código de ProvisionalError
		wantLabel             string
		wantSeries            string // Series().String() si la serie está fijada
	}{
		{"serie fijada", "REQ-<n>.<m>", "prov:30:1", "", "", "REQ-30.*"},
		{"serie sin ceros iniciales", "REQ-<n>.<m>", "prov:030:2", "", "", "REQ-30.*"},
		{"etiqueta de serie nueva", "REQ-<n>.<m>", "prov:a:1", "", "a", ""},
		{"etiqueta con cifras", "REQ-<n>.<m>", "prov:ab2:7", "", "ab2", ""},
		{"dominio fijado", "RF-<DOM>-<NN>", "prov:USU:1", "", "", "RF-USU-*"},
		{"dominio con cifras", "RF-<DOM>-<NN>", "prov:U2:3", "", "", "RF-U2-*"},
		{"dominio y serie fijados", "RF-<DOM>-<n>.<m>", "prov:USU-3:1", "", "", "RF-USU-3.*"},
		{"dominio fijado y etiqueta", "RF-<DOM>-<n>.<m>", "prov:USU-b:1", "", "b", ""},
		{"serie única", "RF-<m>", "prov::1", "", "", "RF-*"},
		{"HU de tres cifras", "HU-<NNN>", "prov::4", "", "", "HU-*"},
		{"dominio con patrón sin DOM", "REQ-<n>.<m>", "prov:USU:1", reqid.CodeProvisionalMalformed, "", ""},
		{"serie con patrón DOM", "RF-<DOM>-<NN>", "prov:30:1", reqid.CodeProvisionalMalformed, "", ""},
		{"etiqueta con patrón DOM sin n", "RF-<DOM>-<NN>", "prov:a:1", reqid.CodeProvisionalNewSeriesUnsupported, "", ""},
		{"etiqueta con patrón de serie única", "RF-<m>", "prov:a:1", reqid.CodeProvisionalNewSeriesUnsupported, "", ""},
		{"etiqueta en posición de dominio", "RF-<DOM>-<n>.<m>", "prov:b-3:1", reqid.CodeProvisionalMalformed, "", ""},
		{"distinto número de componentes", "RF-<DOM>-<n>.<m>", "prov:USU:1", reqid.CodeProvisionalMalformed, "", ""},
		{"demasiados componentes", "REQ-<n>.<m>", "prov:30-31:1", reqid.CodeProvisionalMalformed, "", ""},
		{"clave vacía con patrón con serie", "REQ-<n>.<m>", "prov::1", reqid.CodeProvisionalMalformed, "", ""},
		{"componente mezclado", "REQ-<n>.<m>", "prov:a1B:1", reqid.CodeProvisionalMalformed, "", ""},
		{"componente vacío", "RF-<DOM>-<n>.<m>", "prov:USU-:1", reqid.CodeProvisionalMalformed, "", ""},
		{"ordinal cero", "REQ-<n>.<m>", "prov:30:0", reqid.CodeProvisionalMalformed, "", ""},
		{"ordinal con ceros", "REQ-<n>.<m>", "prov:30:01", reqid.CodeProvisionalMalformed, "", ""},
		{"ordinal desbordado", "REQ-<n>.<m>", "prov:30:99999999999999999999", reqid.CodeProvisionalMalformed, "", ""},
		{"serie desbordada", "REQ-<n>.<m>", "prov:99999999999999999999:1", reqid.CodeProvisionalMalformed, "", ""},
		{"sin ordinal", "REQ-<n>.<m>", "prov:30:", reqid.CodeProvisionalMalformed, "", ""},
		{"no es provisional", "REQ-<n>.<m>", "REQ-30.1", reqid.CodeProvisionalMalformed, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := mustParse(t, tc.template)
			pv, err := p.ParseProvisional(tc.token)
			if tc.wantErr != "" {
				var perr *reqid.ProvisionalError
				if !errors.As(err, &perr) {
					t.Fatalf("error = %v, quería *ProvisionalError", err)
				}
				if perr.Code != tc.wantErr || perr.Token != tc.token {
					t.Fatalf("error = {%s %s}, quería {%s %s}", perr.Code, perr.Token, tc.wantErr, tc.token)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseProvisional(%q): %v", tc.token, err)
			}
			if pv.String() != tc.token {
				t.Errorf("String() = %q", pv.String())
			}
			label, hasLabel := pv.Label()
			if label != tc.wantLabel || hasLabel != (tc.wantLabel != "") {
				t.Errorf("Label() = %q, %v", label, hasLabel)
			}
			series, fixed := pv.Series()
			if fixed != (tc.wantSeries != "") || (fixed && series.String() != tc.wantSeries) {
				t.Errorf("Series() = %q, %v; quería %q", series, fixed, tc.wantSeries)
			}
		})
	}
}

func TestProvisionalNeverValidTitleOrCandidate(t *testing.T) {
	cases := []struct{ template, token string }{
		{"REQ-<n>.<m>", "prov:30:1"},
		{"REQ-<n>.<m>", "prov:a:1"},
		{"RF-<DOM>-<NN>", "prov:USU:1"},
		{"RF-<m>", "prov::1"},
		{"HU-<NNN>", "prov::1"},
		{"RF-<DOM>-<n>.<m>", "prov:USU-3:1"},
		// Con el dominio RF la clave contiene `RF-`: sin máscara sería un candidato.
		{"RF-<DOM>-<n>.<m>", "prov:RF-3:1"},
		{"RF-<DOM>-<n>.<m>", "prov:RF-b:1"},
	}
	for _, tc := range cases {
		t.Run(tc.template+" "+tc.token, func(t *testing.T) {
			p := mustParse(t, tc.template)
			if _, err := p.ParseProvisional(tc.token); err != nil {
				t.Fatalf("el caso debe ser un provisional válido: %v", err)
			}
			if p.Valid(tc.token) {
				t.Errorf("Valid(%q) = true", tc.token)
			}
			title := "Cobro con IVA (" + tc.token + ")"
			if id := p.InTitle(title); id != "" {
				t.Errorf("InTitle = %q", id)
			}
			if rest, id, ok := p.TrailingCandidate(title); ok {
				t.Errorf("TrailingCandidate = %q, %q", rest, id)
			}
			valid, malformed := p.Scan("Cubre: " + tc.token + ", y " + title)
			if len(valid) != 0 || len(malformed) != 0 {
				t.Errorf("Scan = %v, %v; ningún candidato", valid, malformed)
			}
			if got, ok := reqid.ProvisionalInTitle(title); !ok || got != tc.token {
				t.Errorf("ProvisionalInTitle = %q, %v", got, ok)
			}
		})
	}
}

func TestScanStillFindsDefinitivesNextToProvisionals(t *testing.T) {
	p := mustParse(t, "REQ-<n>.<m>")
	valid, malformed := p.Scan("REQ-30.1 junto a prov:30:2 y REQ-x")
	if len(valid) != 1 || valid[0] != "REQ-30.1" {
		t.Errorf("valid = %v", valid)
	}
	if len(malformed) != 1 || malformed[0] != "REQ-x" {
		t.Errorf("malformed = %v", malformed)
	}
}

func TestScanProvisionalWholeTokenRule(t *testing.T) {
	cases := []struct {
		name, text string
		want       []string
		wantLax    []bool
	}{
		{"entre paréntesis", "Cobro (prov:30:1)", []string{"prov:30:1"}, []bool{false}},
		{"con puntuación", "Cubre: prov:30:1, prov:a:2.", []string{"prov:30:1", "prov:a:2"}, []bool{false, false}},
		{"precedido de letra", "xprov:30:1", nil, nil},
		{"precedido de guion", "-prov:30:1", nil, nil},
		{"seguido de letra", "prov:30:1a", nil, nil},
		{"seguido de guion bajo", "prov:30:1_", nil, nil},
		{"seguido de cifra no existe: el ordinal la absorbe", "prov:30:12", []string{"prov:30:12"}, []bool{false}},
		{"clave inválida es laxo", "prov:a1B:1", []string{"prov:a1B:1"}, []bool{true}},
		{"ordinal con cero inicial es laxo", "prov:30:01", []string{"prov:30:01"}, []bool{true}},
		{"componente vacío es laxo", "prov:USU-:1", []string{"prov:USU-:1"}, []bool{true}},
		{"clave vacía", "prov::1", []string{"prov::1"}, []bool{false}},
		{"forma con ángulos no es token", "prov:<serie>:<ordinal>", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spans := reqid.ScanProvisional(tc.text)
			if len(spans) != len(tc.want) {
				t.Fatalf("spans = %+v, quería %v", spans, tc.want)
			}
			for i, s := range spans {
				if s.Token != tc.want[i] || s.Lax != tc.wantLax[i] || tc.text[s.Start:s.End] != s.Token {
					t.Errorf("span %d = %+v, quería %q lax=%v", i, s, tc.want[i], tc.wantLax[i])
				}
			}
		})
	}
}

func TestProvisionalInTitleTakesTheLastOne(t *testing.T) {
	if got, ok := reqid.ProvisionalInTitle("A (prov:1:1) y B (prov:2:3)"); !ok || got != "prov:2:3" {
		t.Errorf("= %q, %v", got, ok)
	}
	if _, ok := reqid.ProvisionalInTitle("Cobro (REQ-30.1)"); ok {
		t.Error("un definitivo no es un provisional")
	}
	if _, ok := reqid.ProvisionalInTitle("Cobro prov:30:1 sin paréntesis"); ok {
		t.Error("sin paréntesis no es el provisional del título")
	}
}
