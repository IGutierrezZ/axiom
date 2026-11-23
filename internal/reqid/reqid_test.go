package reqid_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func mustParse(t testing.TB, template string) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatalf("Parse(%q): %v", template, err)
	}
	return p
}

func TestValid(t *testing.T) {
	cases := []struct {
		template, id string
		want         bool
	}{
		{"REQ-<n>.<m>", "REQ-30.4", true},
		{"REQ-<n>.<m>", "REQ-30.01", true},
		{"REQ-<n>.<m>", "REQ-30", false},
		{"REQ-<n>.<m>", "REQ-x.1", false},
		{"REQ-<n>.<m>", "req-30.4", false},
		{"REQ-<n>.<m>", " REQ-30.4", false},
		{"RF-<DOM>-<NN>", "RF-USU-01", true},
		{"RF-<DOM>-<NN>", "RF-USU-99", true},
		{"RF-<DOM>-<NN>", "RF-USU-100", true},
		{"RF-<DOM>-<NN>", "RF-USU-123", true},
		{"RF-<DOM>-<NN>", "RF-USU-001", false},
		{"RF-<DOM>-<NN>", "RF-USU-1", false},
		{"RF-<DOM>-<NN>", "RF-usu-01", false},
		{"RF-<DOM>-<NN>", "RF-USU-01-bis", false},
		{"RF-<DOM>-<NN>", "REQ-30.1", false},
		{"RF-<m>", "RF-121", true},
		{"RF-<m>", "RF-0", true},
		{"RF-<m>", "RF-USU-1", false},
		{"HU-<NNN>", "HU-007", true},
		{"HU-<NNN>", "HU-7", false},
		{"HU-<NNN>", "HU-1000", true},
		{"HU-<NNN>", "HU-0007", false},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		if got := p.Valid(tc.id); got != tc.want {
			t.Errorf("%s: Valid(%q) = %v, quería %v", tc.template, tc.id, got, tc.want)
		}
	}
}

func TestOverflow(t *testing.T) {
	rf := mustParse(t, "RF-<DOM>-<NN>")
	def := reqid.Default()
	if rf.Valid("RF-USU-99999999999999999999") {
		t.Error("RF-USU-99999999999999999999 no debe ser válido")
	}
	if def.Valid("REQ-1.99999999999999999999") {
		t.Error("REQ-1.99999999999999999999 no debe ser válido")
	}
	if got := rf.InTitle("Alta (RF-USU-99999999999999999999)"); got != "" {
		t.Errorf("InTitle devolvió %q", got)
	}
	if got := def.InTitle("Alta (REQ-1.99999999999999999999)"); got != "" {
		t.Errorf("InTitle devolvió %q", got)
	}
	valid, malformed := def.Scan("REQ-1.99999999999999999999")
	if len(valid) != 0 || !reflect.DeepEqual(malformed, []string{"REQ-1.99999999999999999999"}) {
		t.Errorf("Scan = %v, %v", valid, malformed)
	}
	if _, ok := def.Number("REQ-1.99999999999999999999"); ok {
		t.Error("Number no debe devolver un desbordamiento")
	}
}

func TestInTitle(t *testing.T) {
	cases := []struct {
		template, title, want string
	}{
		{"REQ-<n>.<m>", "Cobro (REQ-30.4)", "REQ-30.4"},
		{"REQ-<n>.<m>", "Cobro", ""},
		{"REQ-<n>.<m>", "Cobro REQ-30.4", ""},
		{"REQ-<n>.<m>", "Cobro (REQ-1.1) (REQ-2.2)", "REQ-2.2"},
		{"REQ-<n>.<m>", "Cobro (REQ-30)", ""},
		{"REQ-<n>.<m>", "Cobro (REQ-030.01)", "REQ-030.01"},
		{"RF-<DOM>-<NN>", "Alta (RF-USU-01)", "RF-USU-01"},
		{"RF-<DOM>-<NN>", "Alta (RF-USU-1)", ""},
		{"RF-<DOM>-<NN>", "Alta (REQ-30.1)", ""},
		{"RF-<m>", "Alta (RF-121)", "RF-121"},
		{"HU-<NNN>", "Historia (HU-007)", "HU-007"},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		if got := p.InTitle(tc.title); got != tc.want {
			t.Errorf("%s: InTitle(%q) = %q, quería %q", tc.template, tc.title, got, tc.want)
		}
	}
}

func TestScan(t *testing.T) {
	cases := []struct {
		template, text string
		valid, bad     []string
	}{
		{"REQ-<n>.<m>", "REQ-30.4 y REQ-30, REQ-30.4.", []string{"REQ-30.4"}, []string{"REQ-30"}},
		{"REQ-<n>.<m>", "REQ-30.1,REQ-2.2.", []string{"REQ-30.1", "REQ-2.2"}, nil},
		{"REQ-<n>.<m>", "XREQ-30.1 req-30.4", nil, nil},
		{"REQ-<n>.<m>", "REQ-x.1 REQ-30.4.1", nil, []string{"REQ-x.1", "REQ-30.4.1"}},
		{"RF-<DOM>-<NN>", "RF-USU-01, RF-FAC-02 RF-USU-01", []string{"RF-USU-01", "RF-FAC-02"}, nil},
		{"RF-<DOM>-<NN>", "Alta (RF-USU-01-bis) RF-USU-1", nil, []string{"RF-USU-01-bis", "RF-USU-1"}},
		{"RF-<DOM>-<NN>", "rf-usu-01 XRF-USU-01 REQ-30.1", nil, nil},
		{"RF-<m>", "RF-121; RF-12a", []string{"RF-121"}, []string{"RF-12a"}},
		{"HU-<NNN>", "HU-007 HU-7", []string{"HU-007"}, []string{"HU-7"}},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		valid, bad := p.Scan(tc.text)
		if !reflect.DeepEqual(valid, tc.valid) || !reflect.DeepEqual(bad, tc.bad) {
			t.Errorf("%s: Scan(%q) = %v, %v; quería %v, %v", tc.template, tc.text, valid, bad, tc.valid, tc.bad)
		}
	}
}

func TestTrailingCandidate(t *testing.T) {
	cases := []struct {
		template, title, rest, id string
		ok                        bool
	}{
		{"REQ-<n>.<m>", "Cobro (REQ-30.4)", "Cobro", "REQ-30.4", true},
		{"REQ-<n>.<m>", "Cobro (REQ-x)  ", "Cobro", "REQ-x", true},
		{"REQ-<n>.<m>", "Cobro", "Cobro", "", false},
		{"REQ-<n>.<m>", "(REQ-30.4)", "(REQ-30.4)", "", false},
		{"REQ-<n>.<m>", "Cobro (REQ-30.4) más", "Cobro (REQ-30.4) más", "", false},
		{"RF-<DOM>-<NN>", "Alta (RF-USU-01)", "Alta", "RF-USU-01", true},
		{"RF-<DOM>-<NN>", "Alta (REQ-30.1)", "Alta (REQ-30.1)", "", false},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		rest, id, ok := p.TrailingCandidate(tc.title)
		if rest != tc.rest || id != tc.id || ok != tc.ok {
			t.Errorf("%s: TrailingCandidate(%q) = %q, %q, %v; quería %q, %q, %v", tc.template, tc.title, rest, id, ok, tc.rest, tc.id, tc.ok)
		}
	}
}

func TestSeriesNumberFormat(t *testing.T) {
	cases := []struct {
		template, id string
		key, series  string
		number       int
	}{
		{"REQ-<n>.<m>", "REQ-30.4", "30", "REQ-30.*", 4},
		{"REQ-<n>.<m>", "REQ-30.01", "30", "REQ-30.*", 1},
		{"RF-<DOM>-<NN>", "RF-USU-01", "USU", "RF-USU-*", 1},
		{"RF-<DOM>-<NN>", "RF-USU-123", "USU", "RF-USU-*", 123},
		{"RF-<m>", "RF-121", "", "RF-*", 121},
		{"HU-<NNN>", "HU-007", "", "HU-*", 7},
		{"A-<DOM>-<n>.<m>-X", "A-FAC-3.12-X", "FAC-3", "A-FAC-3.*-X", 12},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		s, ok := p.Series(tc.id)
		if !ok {
			t.Fatalf("%s: Series(%q) no ok", tc.template, tc.id)
		}
		if s.Key() != tc.key || s.String() != tc.series {
			t.Errorf("%s: Series(%q) = key %q, %q; quería %q, %q", tc.template, tc.id, s.Key(), s.String(), tc.key, tc.series)
		}
		n, ok := p.Number(tc.id)
		if !ok || n != tc.number {
			t.Errorf("%s: Number(%q) = %d, %v; quería %d", tc.template, tc.id, n, ok, tc.number)
		}
		if tc.id != "REQ-30.01" { // el relleno no es canónico con <m>
			if got := p.Format(s, n); got != tc.id {
				t.Errorf("%s: Format(Series(%q), %d) = %q", tc.template, tc.id, n, got)
			}
		}
	}
	if _, ok := reqid.Default().Series("REQ-30"); ok {
		t.Error("Series de un identificador no válido debe fallar")
	}
	if _, ok := reqid.Default().Number("REQ-30"); ok {
		t.Error("Number de un identificador no válido debe fallar")
	}
}

func TestFormatPadding(t *testing.T) {
	rf := mustParse(t, "RF-<DOM>-<NN>")
	s, _ := rf.Series("RF-USU-01")
	for n, want := range map[int]string{0: "RF-USU-00", 9: "RF-USU-09", 99: "RF-USU-99", 100: "RF-USU-100", 12345: "RF-USU-12345"} {
		if got := rf.Format(s, n); got != want {
			t.Errorf("Format(%d) = %q, quería %q", n, got, want)
		}
	}
	def := reqid.Default()
	ds, _ := def.Series("REQ-30.4")
	if got := def.Format(ds, 5); got != "REQ-30.5" {
		t.Errorf("Format = %q", got)
	}
}

func TestCompare(t *testing.T) {
	def := reqid.Default()
	ids := []string{"REQ-10.1", "REQ-9.1", "REQ-30.1", "REQ-2.1"}
	var series []reqid.Series
	for _, id := range ids {
		s, _ := def.Series(id)
		series = append(series, s)
	}
	sort.Slice(series, func(i, j int) bool { return series[i].Compare(series[j]) < 0 })
	var got []string
	for _, s := range series {
		got = append(got, s.String())
	}
	want := []string{"REQ-2.*", "REQ-9.*", "REQ-10.*", "REQ-30.*"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("orden = %v, quería %v", got, want)
	}
	a, _ := def.Series("REQ-9.1")
	b, _ := def.Series("REQ-9.7")
	if a.Compare(b) != 0 || a != b {
		t.Error("la misma serie debe ser igual")
	}
	rf := mustParse(t, "RF-<DOM>-<NN>")
	x, _ := rf.Series("RF-FAC-01")
	y, _ := rf.Series("RF-USU-01")
	if x.Compare(y) >= 0 || y.Compare(x) <= 0 {
		t.Error("FAC debe ir antes que USU")
	}
}
