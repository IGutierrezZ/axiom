package adoption

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// sourceOfIDs construye un origen con una unidad por identificador, titulada con él.
func sourceOfIDs(path string, ids ...string) Source {
	s := Source{Path: path, Kind: KindMarkdown}
	for i, id := range ids {
		s.Units = append(s.Units, Unit{Anchor: fmt.Sprintf("u%d", i), Title: id + " requisito", Line: i + 1})
	}
	return s
}

func rangeIDs(format string, from, to int) []string {
	var ids []string
	for i := from; i <= to; i++ {
		ids = append(ids, fmt.Sprintf(format, i))
	}
	return ids
}

func TestProposePatternTemplates(t *testing.T) {
	cases := []struct {
		name string
		ids  []string
		want string
	}{
		{"ancho 3 con ceros", rangeIDs("RF-%03d", 1, 120), "RF-<NNN>"},
		{"ancho 2 con ceros", rangeIDs("RF-%02d", 1, 45), "RF-<NN>"},
		{"desbordamiento de <NN>", []string{"RF-01", "RF-100"}, "RF-<NN>"},
		{"ancho 1", rangeIDs("RF-%d", 1, 9), "RF-<m>"},
		{"sin prueba de relleno", rangeIDs("RF-%d", 10, 99), "RF-<m>"},
		{"anchos mixtos", []string{"RF-1", "RF-2", "RF-010"}, "RF-<m>"},
		{"relleno y mas ancho con cero", []string{"RF-001", "RF-01"}, "RF-<m>"},
		{"un unico relleno", []string{"RF-001"}, "RF-<NNN>"},
		{"un unico sin relleno", []string{"RF-7"}, "RF-<m>"},
		{"dominio con relleno", []string{"RF-USU-01", "RF-FAC-02"}, "RF-<DOM>-<NN>"},
		{"dominio con anchos mixtos", []string{"RF-USU-1", "RF-FAC-010"}, "RF-<DOM>-<m>"},
		{"separador guion bajo", []string{"RF_001", "RF_002"}, "RF_<NNN>"},
		{"REQ sin punto", []string{"REQ-12", "REQ-13"}, "REQ-<m>"},
		{"cero con relleno", []string{"RF-000", "RF-001"}, "RF-<NNN>"},
		{"cero sin relleno mezclado", []string{"RF-0", "RF-000"}, "RF-<m>"},
		{"ancho 5", rangeIDs("RF-%05d", 1, 120), "RF-<NNNNN>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := ProposePattern([]Source{sourceOfIDs("docs/r.md", tc.ids...)})
			if p == nil {
				t.Fatal("ProposePattern = nil")
			}
			if p.Template != tc.want {
				t.Fatalf("Template = %q, se esperaba %q (conflictos %+v)", p.Template, tc.want, p.Conflicts)
			}
			pattern, err := reqid.Parse(p.Template)
			if err != nil {
				t.Fatalf("reqid.Parse(%q): %v", p.Template, err)
			}
			for _, id := range tc.ids {
				if !pattern.Valid(id) {
					t.Errorf("%q no es válido para %q", id, p.Template)
				}
			}
			if len(p.SchemeConflicts()) != 0 {
				t.Errorf("no debería haber conflictos de esquema: %+v", p.Conflicts)
			}
		})
	}
}

func TestProposePatternSixWidthIsOutOfReach(t *testing.T) {
	// foreignIDPattern admite como máximo 5 dígitos, así que <NNNNNN> (k=6) no es alcanzable.
	if got := foreignIDs("RF-000001"); len(got) != 0 {
		t.Fatalf("foreignIDs reconoce 6 dígitos: %v", got)
	}
}

func TestProposePatternEvidence(t *testing.T) {
	p := ProposePattern([]Source{sourceOfIDs("docs/r.md", rangeIDs("RF-%03d", 1, 120)...)})
	want := []PatternEvidence{
		{"prefix", "RF", 120}, {"separator", "-", 120}, {"domain", "", 120}, {"width", "3", 120},
	}
	if p.IDs != 120 || !reflect.DeepEqual(p.Evidence, want) {
		t.Fatalf("IDs=%d Evidence=%+v", p.IDs, p.Evidence)
	}
}

func TestProposePatternSchemeConflicts(t *testing.T) {
	cases := []struct {
		name string
		ids  []string
		code string
		dim  string
	}{
		{"prefijos", []string{"RF-001", "RNF-001"}, ConflictMixedPrefixes, "prefix"},
		{"dominio", []string{"RF-001", "RF-USU-01"}, ConflictMixedDomain, "domain"},
		{"separadores", []string{"RF-001", "RF_002"}, ConflictMixedSeparators, "separator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := ProposePattern([]Source{sourceOfIDs("docs/r.md", tc.ids...)})
			if p == nil || p.Template != "" {
				t.Fatalf("con conflicto de esquema no hay plantilla: %+v", p)
			}
			got := p.SchemeConflicts()
			if len(got) != 1 || got[0].Code != tc.code {
				t.Fatalf("conflictos = %+v, se esperaba %s", p.Conflicts, tc.code)
			}
			var dims []string
			for _, e := range p.Evidence {
				dims = append(dims, e.Dimension)
			}
			if dims[0] != "prefix" || dims[len(dims)-1] != "width" {
				t.Errorf("la evidencia debe seguir el orden fijo de dimensiones: %v", dims)
			}
		})
	}
}

func TestProposePatternConflictTemplateInvalid(t *testing.T) {
	owned := []ownedID{}
	for _, id := range []string{"RF-1", "RF-2"} {
		parts, _ := splitID(id)
		owned = append(owned, ownedID{unit: "x", parts: parts})
	}
	bad := validateTemplate("RF-<NNN>", owned)
	if bad == nil || bad.Code != ConflictTemplateInvalid {
		t.Fatalf("validateTemplate = %+v", bad)
	}
	bad = validateTemplate("rf-<m>", owned)
	if bad == nil || bad.Code != ConflictTemplateInvalid {
		t.Fatalf("una plantilla que no parsea es template_invalid: %+v", bad)
	}
	if validateTemplate("RF-<m>", owned) != nil {
		t.Fatal("RF-<m> es válida para RF-1 y RF-2")
	}
}

func TestIdentityIDs(t *testing.T) {
	cases := []struct {
		name string
		unit Unit
		want []string
	}{
		{"id en el titulo, el cuerpo cita otros", Unit{Title: "RF-001 Alta", ForeignIDs: []string{"RF-001", "RF-003"}}, []string{"RF-001"}},
		{"titulo sin id, cuerpo con uno", Unit{Title: "Alta", ForeignIDs: []string{"RF-004"}}, []string{"RF-004"}},
		{"varios ids propios", Unit{Title: "RF-001 y RF-002", ForeignIDs: []string{"RF-001", "RF-002"}}, []string{"RF-001", "RF-002"}},
		{"ninguno", Unit{Title: "Alta"}, nil},
		{"fichero entero sin secciones", Unit{Anchor: "", Title: "RF-010.md", ForeignIDs: []string{"RF-010"}}, []string{"RF-010"}},
		{"origen binario", Unit{Anchor: "", Title: "requisitos.docx", Line: 1}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IdentityIDs(tc.unit); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("IdentityIDs = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}

func TestProposePatternUnitConflicts(t *testing.T) {
	t.Run("unidad con varios ids propios", func(t *testing.T) {
		src := Source{Path: "docs/r.md", Units: []Unit{
			{Anchor: "a", Title: "RF-001 y RF-002"},
			{Anchor: "b", Title: "RF-003"},
		}}
		p := ProposePattern([]Source{src})
		if p.Template != "RF-<NNN>" {
			t.Fatalf("la plantilla se sigue proponiendo: %+v", p)
		}
		got := p.UnitConflicts()
		if len(got) != 1 || got[0].Code != ConflictAmbiguousUnitID || got[0].Units[0] != "docs/r.md#a" {
			t.Fatalf("conflictos = %+v", p.Conflicts)
		}
	})
	t.Run("mismo id en dos unidades", func(t *testing.T) {
		src := Source{Path: "docs/r.md", Units: []Unit{{Anchor: "a", Title: "RF-001 uno"}, {Anchor: "b", Title: "RF-001 dos"}}}
		p := ProposePattern([]Source{src})
		got := p.UnitConflicts()
		if len(got) != 1 || got[0].Code != ConflictDuplicateForeignID || len(got[0].Units) != 2 {
			t.Fatalf("conflictos = %+v", p.Conflicts)
		}
	})
	t.Run("mismo id en dos origenes", func(t *testing.T) {
		p := ProposePattern([]Source{sourceOfIDs("a.md", "RF-001"), sourceOfIDs("b.md", "RF-001")})
		got := p.UnitConflicts()
		if len(got) != 1 || got[0].Code != ConflictDuplicateForeignID {
			t.Fatalf("conflictos = %+v", p.Conflicts)
		}
	})
	t.Run("por serie y numero con plantilla", func(t *testing.T) {
		p := ProposePattern([]Source{sourceOfIDs("a.md", "RF-1", "RF-001", "RF-20")})
		if p.Template != "RF-<m>" {
			t.Fatalf("Template = %q", p.Template)
		}
		got := p.UnitConflicts()
		if len(got) != 1 || got[0].Code != ConflictDuplicateForeignID || !reflect.DeepEqual(got[0].IDs, []string{"RF-001", "RF-1"}) {
			t.Fatalf("RF-1 y RF-001 se repiten con <m>: %+v", p.Conflicts)
		}
	})
	t.Run("por cadena sin plantilla", func(t *testing.T) {
		p := ProposePattern([]Source{sourceOfIDs("a.md", "RF-1", "RNF-1", "RF-001")})
		if p.Template != "" {
			t.Fatal("hay conflicto de esquema")
		}
		if len(p.UnitConflicts()) != 0 {
			t.Fatalf("sin plantilla se compara por cadena y RF-1 != RF-001: %+v", p.UnitConflicts())
		}
	})
}

func TestProposePatternDeterministicAndNil(t *testing.T) {
	srcs := []Source{sourceOfIDs("a.md", "RF-002", "RF-001"), sourceOfIDs("b.md", "RF-003")}
	a, b := ProposePattern(srcs), ProposePattern(srcs)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("no es determinista: %+v vs %+v", a, b)
	}
	if got := ProposePattern([]Source{{Path: "a.md", Units: []Unit{{Title: "Intro"}}}}); got != nil {
		t.Fatalf("sin IDs propios debe ser nil: %+v", got)
	}
	if got := ProposePattern(nil); got != nil {
		t.Fatalf("ProposePattern(nil) = %+v", got)
	}
	if !strings.Contains(ConflictMixedPrefixes, "prefix") {
		t.Fatal("código estable")
	}
}
