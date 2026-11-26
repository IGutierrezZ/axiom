package reqid_test

import (
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestSeriesOf(t *testing.T) {
	cases := []struct {
		template, key, want string
		ok                  bool
	}{
		{"REQ-<n>.<m>", "30", "REQ-30.*", true},
		{"REQ-<n>.<m>", "030", "REQ-30.*", true},
		{"RF-<DOM>-<NN>", "USU", "RF-USU-*", true},
		{"RF-<DOM>-<n>.<m>", "USU-3", "RF-USU-3.*", true},
		{"RF-<m>", "", "RF-*", true},
		{"REQ-<n>.<m>", "", "", false},
		{"REQ-<n>.<m>", "USU", "", false},
		{"RF-<DOM>-<NN>", "30", "", false},
		{"REQ-<n>.<m>", "30-31", "", false},
		{"REQ-<n>.<m>", "99999999999999999999", "", false},
	}
	for _, tc := range cases {
		p := mustParse(t, tc.template)
		got, ok := p.SeriesOf(tc.key)
		if ok != tc.ok || (ok && got.String() != tc.want) {
			t.Errorf("%s SeriesOf(%q) = %q, %v; quería %q, %v", tc.template, tc.key, got, ok, tc.want, tc.ok)
		}
		if ok {
			if again, _ := p.Series(p.Format(got, 1)); again != got {
				t.Errorf("%s SeriesOf(%q) no coincide con Series del identificador formado", tc.template, tc.key)
			}
		}
	}
}

func TestAllocatorNext(t *testing.T) {
	t.Run("serie vacía da 1 con el relleno de la plantilla", func(t *testing.T) {
		cases := []struct{ template, key, want string }{
			{"REQ-<n>.<m>", "30", "REQ-30.1"},
			{"RF-<DOM>-<NN>", "USU", "RF-USU-01"},
			{"HU-<NNN>", "", "HU-001"},
			{"RF-<m>", "", "RF-1"},
		}
		for _, tc := range cases {
			p := mustParse(t, tc.template)
			series, _ := p.SeriesOf(tc.key)
			got, err := reqid.NewAllocator(p).Next(series)
			if err != nil || got != tc.want {
				t.Errorf("%s Next = %q, %v; quería %q", tc.template, got, err, tc.want)
			}
		}
	})
	t.Run("máximo más uno, sin rellenar huecos ni contar otras series", func(t *testing.T) {
		p := mustParse(t, "RF-<DOM>-<NN>")
		a := reqid.NewAllocator(p)
		for _, id := range []string{"RF-USU-01", "RF-USU-09", "RF-USU-03", "RF-FAC-20"} {
			if err := a.Occupy(id); err != nil {
				t.Fatal(err)
			}
		}
		series, _ := p.SeriesOf("USU")
		for _, want := range []string{"RF-USU-10", "RF-USU-11"} {
			if got, err := a.Next(series); err != nil || got != want {
				t.Fatalf("Next = %q, %v; quería %q", got, err, want)
			}
		}
	})
	t.Run("ocupar repetidos no altera el máximo", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		for _, id := range []string{"REQ-3.2", "REQ-3.2", "REQ-3.1", "REQ-3.2"} {
			if err := a.Occupy(id); err != nil {
				t.Fatal(err)
			}
		}
		series, _ := p.SeriesOf("3")
		if got, _ := a.Next(series); got != "REQ-3.3" {
			t.Errorf("Next = %q", got)
		}
	})
	t.Run("desbordamiento", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		if err := a.Occupy("REQ-3." + strconv.Itoa(math.MaxInt)); err != nil {
			t.Fatal(err)
		}
		series, _ := p.SeriesOf("3")
		_, err := a.Next(series)
		var aerr *reqid.AllocationError
		if !errors.As(err, &aerr) {
			t.Fatalf("error = %v, quería *AllocationError", err)
		}
	})
	t.Run("Occupy de un identificador no válido", func(t *testing.T) {
		a := reqid.NewAllocator(mustParse(t, "REQ-<n>.<m>"))
		var aerr *reqid.AllocationError
		if err := a.Occupy("REQ-x"); !errors.As(err, &aerr) {
			t.Fatalf("error = %v, quería *AllocationError", err)
		}
	})
	t.Run("Clone es independiente", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		_ = a.Occupy("REQ-3.1")
		series, _ := p.SeriesOf("3")
		c := a.Clone()
		if got, _ := c.Next(series); got != "REQ-3.2" {
			t.Fatalf("clon Next = %q", got)
		}
		if got, _ := a.Next(series); got != "REQ-3.2" {
			t.Errorf("el original quedó alterado: Next = %q", got)
		}
	})
}

func TestAllocatorNewSeries(t *testing.T) {
	label := func(t *testing.T, p reqid.Pattern, token string) reqid.Provisional {
		t.Helper()
		pv, err := p.ParseProvisional(token)
		if err != nil {
			t.Fatal(err)
		}
		return pv
	}
	t.Run("máximo de series más uno", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		for _, id := range []string{"REQ-3.3", "REQ-46.1", "REQ-30.2"} {
			_ = a.Occupy(id)
		}
		s, err := a.NewSeries(label(t, p, "prov:a:1"))
		if err != nil || s.String() != "REQ-47.*" {
			t.Fatalf("NewSeries = %q, %v", s, err)
		}
	})
	t.Run("etiquetas distintas dan series distintas, consecutivas", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		_ = a.Occupy("REQ-46.1")
		s1, _ := a.NewSeries(label(t, p, "prov:a:1"))
		s2, _ := a.NewSeries(label(t, p, "prov:b:1"))
		if s1.String() != "REQ-47.*" || s2.String() != "REQ-48.*" {
			t.Errorf("series = %q, %q", s1, s2)
		}
	})
	t.Run("nunca una serie fijada por el propio cambio", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		_ = a.Occupy("REQ-46.1")
		fixed, _ := p.SeriesOf("47")
		if id, _ := a.Next(fixed); id != "REQ-47.1" {
			t.Fatalf("Next = %q", id)
		}
		s, _ := a.NewSeries(label(t, p, "prov:a:1"))
		if s.String() != "REQ-48.*" {
			t.Errorf("NewSeries = %q, quería REQ-48.*", s)
		}
	})
	t.Run("sin series ocupadas empieza en 1", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		s, err := reqid.NewAllocator(p).NewSeries(label(t, p, "prov:a:1"))
		if err != nil || s.String() != "REQ-1.*" {
			t.Errorf("NewSeries = %q, %v", s, err)
		}
	})
	t.Run("solo cuentan las series con el mismo dominio", func(t *testing.T) {
		p := mustParse(t, "RF-<DOM>-<n>.<m>")
		a := reqid.NewAllocator(p)
		for _, id := range []string{"RF-USU-3.1", "RF-USU-5.2", "RF-FAC-9.1"} {
			_ = a.Occupy(id)
		}
		s, err := a.NewSeries(label(t, p, "prov:USU-b:1"))
		if err != nil || s.String() != "RF-USU-6.*" {
			t.Errorf("NewSeries = %q, %v", s, err)
		}
	})
	t.Run("desbordamiento de la serie", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		a := reqid.NewAllocator(p)
		_ = a.Occupy("REQ-" + strconv.Itoa(math.MaxInt) + ".1")
		var aerr *reqid.AllocationError
		if _, err := a.NewSeries(label(t, p, "prov:a:1")); !errors.As(err, &aerr) {
			t.Fatalf("error = %v, quería *AllocationError", err)
		}
	})
	t.Run("un provisional de serie fijada no es una etiqueta", func(t *testing.T) {
		p := mustParse(t, "REQ-<n>.<m>")
		var aerr *reqid.AllocationError
		if _, err := reqid.NewAllocator(p).NewSeries(label(t, p, "prov:30:1")); !errors.As(err, &aerr) {
			t.Fatalf("error = %v, quería *AllocationError", err)
		}
	})
}
