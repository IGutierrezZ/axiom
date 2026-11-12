package adoption

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct{ title, want string }{
		{"RF-001", "rf-001"},
		{"Requisitos funcionales", "requisitos-funcionales"},
		{"Gestión de usuarios (v2)", "gestión-de-usuarios-v2"},
		{"  ¿Qué es esto?  ", "qué-es-esto"},
		{"Año nuevo: ñandú", "año-nuevo-ñandú"},
		{"snake_case y `código`", "snake_case-y-código"},
		{"Títle   con   espacios", "títle---con---espacios"},
		{"A -- B", "a----b"},
		{"!!!", ""},
	}
	for _, c := range cases {
		if got := Slug(c.title); got != c.want {
			t.Errorf("Slug(%q) = %q, quería %q", c.title, got, c.want)
		}
	}
}

func TestSlugger_DuplicadosAlEstiloGitHub(t *testing.T) {
	s := newSlugger()
	got := []string{s.next("Glosario"), s.next("Glosario"), s.next("Glosario"), s.next("Otro")}
	want := []string{"glosario", "glosario-1", "glosario-2", "otro"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ancla %d = %q, quería %q (todas: %v)", i, got[i], want[i], got)
		}
	}
}

func TestSlugger_NoChocaConUnSlugExistente(t *testing.T) {
	s := newSlugger()
	got := []string{s.next("Glosario"), s.next("Glosario 1"), s.next("Glosario"), s.next("Glosario")}
	want := []string{"glosario", "glosario-1", "glosario-2", "glosario-3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ancla %d = %q, quería %q (todas: %v)", i, got[i], want[i], got)
		}
	}
}

func TestSlugger_TituloSinLetrasNiDigitos(t *testing.T) {
	s := newSlugger()
	if a, b := s.next("***"), s.next("***"); a != "" || b != "-1" {
		t.Fatalf("anclas de un título vacío = %q y %q, quería \"\" y \"-1\"", a, b)
	}
}
