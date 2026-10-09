package specbaseline

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// requirementView es la parte de un requisito que comprueban las pruebas:
// número de línea del encabezado, identificador, primera y última línea del
// bloque (desde 1) y encabezados de su cola contigua.
type requirementView struct {
	line  int
	key   string
	first int
	last  int
	tail  []string
}

func viewOf(requirements []Requirement) []requirementView {
	var out []requirementView
	for _, r := range requirements {
		view := requirementView{line: r.LineNumber(), key: r.Key, first: r.Start + 1, last: r.End}
		for _, section := range r.Tail {
			view.tail = append(view.tail, section.Line)
		}
		out = append(out, view)
	}
	return out
}

func TestBlockRequirementsOfTheFunctionalReference(t *testing.T) {
	profile := requirementProfile(t)
	want := []requirementView{
		{line: 7, key: "RF-001", first: 7, last: 20},
		{line: 21, key: "RF-002", first: 21, last: 24, tail: []string{"### Notas de implantación del catálogo"}},
		{line: 29, key: "RF-003", first: 29, last: 39, tail: []string{"### Criterios de aceptación generales", "### Glosario del catálogo"}},
		{line: 48, key: "RF-040", first: 48, last: 51},
		{line: 52, key: "RF-030", first: 52, last: 55},
		{line: 58, key: "RF-020", first: 58, last: 61},
		{line: 62, key: "RF-021", first: 62, last: 65},
		{line: 68, key: "RF-020", first: 68, last: 71},
		{line: 72, key: "RF-021", first: 72, last: 80},
		{line: 83, key: "RF-050", first: 83, last: 85},
	}
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		doc := ParseDocument(v.apply(readFixture(t, "01_Requisitos_Funcionales.md")))
		got := viewOf(doc.Requirements(profile))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("requisitos = %+v\nse esperaba   %+v", got, want)
		}
	})
}

func TestBlockSubsectionsAndFencesStayInsideTheBlock(t *testing.T) {
	profile := requirementProfile(t)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		doc := ParseDocument(v.apply(readFixture(t, "01_Requisitos_Funcionales.md")))
		requirements := doc.Requirements(profile)

		first := requirements[0]
		if first.Key != "RF-001" || first.End-first.Start != 14 {
			t.Fatalf("RF-001 debe incluir sus dos subsecciones h4: %+v", first)
		}
		third := requirements[2]
		if third.Key != "RF-003" {
			t.Fatalf("requisito inesperado: %+v", third)
		}
		fenced := false
		for i := third.Start; i < third.End; i++ {
			if doc.Lines[i].Text == "## Encabezado falso dentro de una valla" {
				fenced = true
			}
		}
		if !fenced {
			t.Error("el bloque de RF-003 debe incluir la valla con encabezados falsos")
		}
		for _, r := range requirements {
			if r.Key == "RF-900" {
				t.Error("un requisito dentro de una valla no debe reconocerse")
			}
		}
	})
}

func TestBlockBoundaryDependsOnTheHeadingLevel(t *testing.T) {
	profile := profileFor(t, `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosInclude)
	tests := []struct {
		name string
		text string
		want []requirementView
	}{
		{
			name: "un encabezado de nivel menor cierra el bloque",
			text: "## Pagos\n\n### RF-001: A\n\nCuerpo.\n\n## Otra\n",
			want: []requirementView{{line: 3, key: "RF-001", first: 3, last: 6}},
		},
		{
			name: "un encabezado del mismo nivel que no es requisito cierra el bloque y abre la cola",
			text: "### RF-001: A\n\nCuerpo.\n\n### Notas\n\nTexto.\n\n### RF-002: B\n",
			want: []requirementView{
				{line: 1, key: "RF-001", first: 1, last: 4, tail: []string{"### Notas"}},
				{line: 9, key: "RF-002", first: 9, last: 9},
			},
		},
		{
			name: "el nivel declarado filtra los encabezados",
			text: "### RF-100: A\n\nCuerpo.\n\n#### RF-101: B\n\nSubsección.\n",
			want: []requirementView{{line: 1, key: "RF-100", first: 1, last: 7}},
		},
		{
			name: "un encabezado de nivel mayor que coincide con el patrón no es requisito",
			text: "##### RF-100: A\n",
			want: nil,
		},
		{
			name: "un encabezado de nivel menor que coincide con el patrón no es requisito",
			text: "## RF-100: A\n",
			want: nil,
		},
		{
			name: "la cola se detiene en un encabezado de nivel menor",
			text: "### RF-001: A\n\n### Notas\n\n## Siguiente sección\n\n### Otra nota\n",
			want: []requirementView{{line: 1, key: "RF-001", first: 1, last: 2, tail: []string{"### Notas"}}},
		},
		{
			name: "sin cola cuando sigue directamente otro requisito",
			text: "### RF-001: A\n### RF-002: B\n",
			want: []requirementView{
				{line: 1, key: "RF-001", first: 1, last: 1},
				{line: 2, key: "RF-002", first: 2, last: 2},
			},
		},
		{
			name: "un encabezado con espacios iniciales cierra el bloque",
			text: "### RF-001: A\n\n  ## Subtítulo\n",
			want: []requirementView{{line: 1, key: "RF-001", first: 1, last: 2}},
		},
		{
			name: "una valla sin cerrar absorbe el resto del bloque",
			text: "### RF-001: A\n\n```\n### RF-002: B\n## Otra\n",
			want: []requirementView{{line: 1, key: "RF-001", first: 1, last: 5}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				got := viewOf(ParseDocument(v.apply(tt.text)).Requirements(profile))
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("requisitos = %+v\nse esperaba   %+v", got, tt.want)
				}
			})
		})
	}
}

func TestBlockRequirementWithoutFinalNewlineKeepsItsExtent(t *testing.T) {
	profile := requirementProfile(t)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		doc := ParseDocument(v.apply(readFixture(t, "01_Requisitos_Funcionales_sin_salto_final.md")))
		got := viewOf(doc.Requirements(profile))
		want := []requirementView{
			{line: 5, key: "RF-100", first: 5, last: 8},
			{line: 9, key: "RF-101", first: 9, last: 11},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("requisitos = %+v\nse esperaba   %+v", got, want)
		}
	})
}

func TestBlockIdentityByTextUsesTheTitleAsKey(t *testing.T) {
	profile := profileFor(t, `^### (?P<id>[A-Z]+-\d+): (?P<title>.+)$`, 3, workspace.IdentityText, "### {id}: {title}", workspace.ScenariosInclude)
	doc := ParseDocument([]byte("### RF-1: Alta\n\n### RF-2: Baja\n"))
	requirements := doc.Requirements(profile)
	if len(requirements) != 2 || requirements[0].Key != "Alta" || requirements[1].Key != "Baja" {
		t.Fatalf("claves = %+v, se esperaban los títulos", viewOf(requirements))
	}
	if requirements[0].ID != "RF-1" || requirements[0].Title != "Alta" {
		t.Errorf("id y título = (%q, %q)", requirements[0].ID, requirements[0].Title)
	}
}

func TestBlockDuplicatesAreGroupedByKey(t *testing.T) {
	profile := requirementProfile(t)

	doc := ParseDocument([]byte(readFixture(t, "01_Requisitos_Funcionales.md")))
	groups := Duplicates(doc.Requirements(profile))
	if len(groups) != 2 {
		t.Fatalf("grupos duplicados = %d, se esperaban 2 (RF-020 y RF-021)", len(groups))
	}
	for key, wantTitles := range map[string][]string{
		"RF-020": {"Cobro con tarjeta", "Devolución parcial"},
		"RF-021": {"Cobro con transferencia", "Devolución total"},
	} {
		var titles []string
		for _, r := range groups[key] {
			titles = append(titles, r.Title)
		}
		if !reflect.DeepEqual(titles, wantTitles) {
			t.Errorf("%s: títulos = %q, se esperaba %q", key, titles, wantTitles)
		}
	}

	clean := ParseDocument([]byte(readFixture(t, "02_Requisitos_No_Funcionales.md")))
	if groups := Duplicates(clean.Requirements(profile)); len(groups) != 0 {
		t.Errorf("el documento de no funcionales no tiene duplicados: %v", groups)
	}
	if got := len(clean.Requirements(profile)); got != 3 {
		t.Errorf("requisitos no funcionales = %d, se esperaban 3", got)
	}
}
