package docpaths

import "testing"

// Matriz de amenazas del modo acotado: solo el nombre exacto del fichero de marcas.
func TestExactAllowed(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"pruebas-estado.yaml", true},
		{"design.md", false},
		{"internal/x.go", false},
		{"../pruebas-estado.yaml", false},
		{"/pruebas-estado.yaml", false},
		{`C:\pruebas-estado.yaml`, false},
		{"sub/pruebas-estado.yaml", false},
		{"./pruebas-estado.yaml", false},
		{"Pruebas-Estado.yaml", false},
		{"pruebas-estado.yaml ", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ExactAllowed(c.name); got != c.want {
			t.Errorf("ExactAllowed(%q) = %v, quería %v", c.name, got, c.want)
		}
	}
}
