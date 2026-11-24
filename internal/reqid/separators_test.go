package reqid_test

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Los separadores de los literales pueden aparecer en cualquier orden: ninguno
// debe formar un rango dentro de la clase de candidatos.
func TestParseSeparatorOrders(t *testing.T) {
	cases := []struct {
		template string
		id       string
	}{
		{"RF-<DOM>_<n>-<m>.V", "RF-USU_1-2.V"},
		{"RF-<DOM>.<n>-<m>_V", "RF-USU.1-2_V"},
		{"RF-<DOM>-<n>_<m>.V", "RF-USU-1_2.V"},
		{"RF-<DOM>.<n>-<m>V", "RF-USU.1-2V"},
		{"RF-<DOM>_<n>.<m>V", "RF-USU_1.2V"},
	}
	for _, c := range cases {
		t.Run(c.template, func(t *testing.T) {
			p, err := reqid.Parse(c.template)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.template, err)
			}
			valid, malformed := p.Scan("x " + c.id + " y")
			if !reflect.DeepEqual(valid, []string{c.id}) || len(malformed) != 0 {
				t.Errorf("Scan válido = %v, malformado = %v; quería [%s]", valid, malformed, c.id)
			}
			// ':' y ';' quedan fuera del candidato.
			valid, malformed = p.Scan("x " + c.id + ":" + c.id + ";")
			if !reflect.DeepEqual(valid, []string{c.id}) || len(malformed) != 0 {
				t.Errorf("Scan con ':' y ';' válido = %v, malformado = %v", valid, malformed)
			}
		})
	}
}
