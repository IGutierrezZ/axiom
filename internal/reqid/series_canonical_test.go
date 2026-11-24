package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Con el patrón por defecto la serie se identificaba por su valor entero
// (`strconv.Atoi`): `REQ-09.1` y `REQ-9.2` pertenecían a la misma serie 9.
func TestSeriesCanonicalizesLeadingZerosInSeriesNumber(t *testing.T) {
	p := reqid.Default()
	a, okA := p.Series("REQ-09.1")
	b, okB := p.Series("REQ-9.2")
	if !okA || !okB {
		t.Fatalf("Series: okA=%v okB=%v", okA, okB)
	}
	if a != b {
		t.Fatalf("REQ-09.1 y REQ-9.2 deben ser la misma serie: %q frente a %q", a, b)
	}
	if got := a.Key(); got != "9" {
		t.Errorf("Key() = %q; se esperaba %q", got, "9")
	}
	if got := a.String(); got != "REQ-9.*" {
		t.Errorf("String() = %q; se esperaba %q", got, "REQ-9.*")
	}
	if got := p.Format(a, 3); got != "REQ-9.3" {
		t.Errorf("Format = %q; se esperaba %q", got, "REQ-9.3")
	}
	zero, ok := p.Series("REQ-00.1")
	if !ok || zero.Key() != "0" {
		t.Errorf("REQ-00.1: ok=%v key=%q; se esperaba la serie 0", ok, zero.Key())
	}
}

// El valor de `<n>` que desborda int no forma serie, como `livespec.SeriesOf` antes.
func TestSeriesRejectsOverflowingSeriesNumber(t *testing.T) {
	if s, ok := reqid.Default().Series("REQ-99999999999999999999.1"); ok {
		t.Fatalf("se esperaba que no hubiera serie; se obtuvo %q", s)
	}
}

// `<DOM>` es texto y se conserva tal cual; solo `<n>` se normaliza.
func TestSeriesCanonicalizesOnlyNumericSeriesSegments(t *testing.T) {
	p, err := reqid.Parse("RF-<DOM>-<n>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	a, okA := p.Series("RF-USU-007-01")
	b, okB := p.Series("RF-USU-7-02")
	if !okA || !okB || a != b {
		t.Fatalf("se esperaba la misma serie: %q (%v) frente a %q (%v)", a, okA, b, okB)
	}
	if got := a.Key(); got != "USU-7" {
		t.Errorf("Key() = %q; se esperaba %q", got, "USU-7")
	}
}
