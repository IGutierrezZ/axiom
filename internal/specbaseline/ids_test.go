package specbaseline

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de la asignación y la sustitución de IDs provisionales (REQ-24.43,
// REQ-24.59 a-d, D-11). Todos los textos son inventados.

// idsDocument es un documento con IDs no ordenados: el mayor es RF-007 aunque
// RF-003 vaya después.
var idsDocument = join(
	"# Documento de prueba",
	"",
	"## Catálogo",
	"",
	"### RF-001: Primero",
	"",
	"Cuerpo.",
	"",
	"### RF-002: Segundo",
	"",
	"Cuerpo.",
	"",
	"### RF-007: Séptimo",
	"",
	"Cuerpo.",
	"",
	"### RF-003: Tercero",
	"",
	"Cuerpo.",
)

// addedDelta escribe un delta con un ADDED por nombre.
func addedDelta(names ...string) string {
	lines := []string{"## ADDED Requirements", ""}
	for _, name := range names {
		lines = append(lines, "### Requirement: "+name, "", "(Placement: after RF-001)", "", "Cuerpo de "+name+".", "")
	}
	return join(lines...)
}

func plannedDoc(domain string, profile Profile, original, deltaText string) PlannedDocument {
	return PlannedDocument{
		Domain:   domain,
		Path:     "/espacio/docs/" + domain + ".md",
		RelPath:  "docs/" + domain + ".md",
		Profile:  profile,
		Original: []byte(original),
		Delta:    []byte(deltaText),
	}
}

func definitives(m IDMap) map[string]string {
	out := map[string]string{}
	for _, a := range m.Assignments() {
		out[a.Provisional] = a.Definitive
	}
	return out
}

func TestSplitID(t *testing.T) {
	tests := []struct {
		id, family, digits string
	}{
		{"RF-123", "RF-", "123"},
		{"RF-0099", "RF-", "0099"},
		{"NFR-9001", "NFR-", "9001"},
		{"V2-009", "V2-", "009"},
		{"RF-A1B2", "RF-A1B", "2"},
		{"RF-NUEVO", "RF-NUEVO", ""},
		{"RF-12-bis", "RF-12-bis", ""},
		{"7", "", "7"},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			family, digits := splitID(tt.id)
			if family != tt.family || digits != tt.digits {
				t.Errorf("splitID(%q) = (%q, %q), se esperaba (%q, %q)", tt.id, family, digits, tt.family, tt.digits)
			}
		})
	}
}

func TestPlanIDsAssignsConsecutiveIDsAfterTheFamilyMaximum(t *testing.T) {
	profile := requirementProfile(t)
	m, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos-funcionales", profile, idsDocument, addedDelta("RF-9001: Reintento", "RF-9002: Aviso"))})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	want := []IDAssignment{
		{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos-funcionales"},
		{Provisional: "RF-9002", Definitive: "RF-009", Domain: "requisitos-funcionales"},
	}
	if got := m.Assignments(); !reflect.DeepEqual(got, want) {
		t.Errorf("Assignments() = %+v\nse esperaba %+v", got, want)
	}
	if got, ok := m.Lookup("RF-9002"); !ok || got.Definitive != "RF-009" {
		t.Errorf("Lookup(RF-9002) = (%+v, %v)", got, ok)
	}
	if _, ok := m.Lookup("RF-008"); ok {
		t.Error("Lookup solo conoce los IDs provisionales")
	}
}

func TestPlanIDsPadsWithTheZerosOfTheLargestIDOfTheFamily(t *testing.T) {
	profile := requirementProfile(t)
	tests := []struct {
		name     string
		existing []string
		want     []string
	}{
		{name: "tres dígitos", existing: []string{"RF-007"}, want: []string{"RF-008", "RF-009"}},
		{name: "cuatro dígitos", existing: []string{"RF-0099"}, want: []string{"RF-0100", "RF-0101"}},
		{name: "sin relleno", existing: []string{"RF-99"}, want: []string{"RF-100", "RF-101"}},
		{name: "el ancho es el del mayor", existing: []string{"RF-0003", "RF-12"}, want: []string{"RF-13", "RF-14"}},
		{name: "con números iguales, el ancho mayor", existing: []string{"RF-10", "RF-0010"}, want: []string{"RF-0011", "RF-0012"}},
		{name: "un número mayor que el máximo de uint64", existing: []string{"RF-18446744073709551615"}, want: []string{"RF-18446744073709551616", "RF-18446744073709551617"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []string
			for _, id := range tt.existing {
				lines = append(lines, "### "+id+": Existente", "", "Cuerpo.", "")
			}
			m, err := PlanIDs([]PlannedDocument{plannedDoc("d", profile, join(lines...), addedDelta("RF-9001: Uno", "RF-9002: Dos"))})
			if err != nil {
				t.Fatalf("PlanIDs() error = %v", err)
			}
			got := m.Assignments()
			if len(got) != 2 || got[0].Definitive != tt.want[0] || got[1].Definitive != tt.want[1] {
				t.Errorf("definitivos = %+v, se esperaba %v", got, tt.want)
			}
		})
	}
}

func TestPlanIDsComputesTheMaximumOnTheOriginalDocumentCountingDuplicates(t *testing.T) {
	profile := requirementProfile(t)
	original := join(
		"### RF-020: Primero", "", "Cuerpo.", "",
		"### RF-021: Segundo", "", "Cuerpo.", "",
		"### RF-020: Duplicado con otro título", "", "Cuerpo.", "",
		"### RF-050: Máximo", "", "Cuerpo.", "",
		"### RF-050: Máximo duplicado", "", "Cuerpo.",
	)
	// El delta elimina el máximo, pero la asignación se calcula antes de aplicar
	// ninguna operación.
	deltaText := join(
		"## REMOVED Requirements", "",
		"### Requirement: RF-021: Segundo", "", "(Reason: se retira)", "",
		"## ADDED Requirements", "",
		"### Requirement: RF-9001: Nuevo", "", "(Placement: after RF-020)", "", "Cuerpo.",
	)
	m, err := PlanIDs([]PlannedDocument{plannedDoc("d", profile, original, deltaText)})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	if got := definitives(m)["RF-9001"]; got != "RF-051" {
		t.Errorf("definitivo = %q, se esperaba RF-051", got)
	}
}

func TestPlanIDsKeepsEachFamilyAndEachDocumentApart(t *testing.T) {
	profile := requirementProfile(t)
	functional := join("### RF-010: Uno", "", "Cuerpo.", "", "### NFR-004: Rendimiento", "", "Cuerpo.")
	nonFunctional := join("### NFR-100: Disponibilidad", "", "Cuerpo.")
	prose := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)

	m, err := PlanIDs([]PlannedDocument{
		plannedDoc("funcionales", profile, functional, addedDelta("RF-9001: Funcional uno", "NFR-9001: Rendimiento nuevo", "RF-9002: Funcional dos")),
		plannedDoc("no-funcionales", profile, nonFunctional, addedDelta("NFR-9002: Disponibilidad nueva")),
		// Un documento con identidad text no tiene IDs provisionales.
		plannedDoc("texto", prose, "### Alpha\n\nCuerpo.\n", addedDelta("Beta")),
	})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	want := []IDAssignment{
		{Provisional: "RF-9001", Definitive: "RF-011", Domain: "funcionales"},
		{Provisional: "NFR-9001", Definitive: "NFR-005", Domain: "funcionales"},
		{Provisional: "RF-9002", Definitive: "RF-012", Domain: "funcionales"},
		{Provisional: "NFR-9002", Definitive: "NFR-101", Domain: "no-funcionales"},
	}
	if got := m.Assignments(); !reflect.DeepEqual(got, want) {
		t.Errorf("Assignments() = %+v\nse esperaba %+v", got, want)
	}
}

func TestPlanIDsWithoutAddedRequirementsIsEmpty(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-001: Primero", "", "Cuerpo nuevo.")
	m, err := PlanIDs([]PlannedDocument{plannedDoc("d", profile, idsDocument, deltaText)})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	if got := m.Assignments(); len(got) != 0 {
		t.Errorf("Assignments() = %+v, se esperaba vacío", got)
	}
	if got := m.Substitute("RF-001 y RF-9001"); got != "RF-001 y RF-9001" {
		t.Errorf("una correspondencia vacía no sustituye nada: %q", got)
	}
}

func TestPlanIDsStopsOnAnIDWithoutANumericSuffix(t *testing.T) {
	profile := profileFor(t, `^#+ (?P<id>RF-(?:\d+|NUEVO|12-bis)): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosInclude)
	for _, id := range []string{"RF-NUEVO", "RF-12-bis"} {
		t.Run(id, func(t *testing.T) {
			_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta(id+": Nuevo"))})
			composeErr := assertComposeError(t, err, id, "sufijo numérico", "ADDED", "Continuación", "renombra el ID provisional")
			if composeErr.Domain != "requisitos" {
				t.Errorf("dominio = %q", composeErr.Domain)
			}
		})
	}
}

func TestPlanIDsStopsOnAFamilyWithoutIDsWithTheLiteralContinuation(t *testing.T) {
	profile := requirementProfile(t)
	_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta("NFR-9001: Rendimiento"))})
	assertComposeError(t, err, "NFR-9001", `familia de IDs "NFR-"`, "docs/requisitos.md",
		"siembra la numeración añadiendo al documento, en un commit propio del repositorio de especificaciones",
		"(por ejemplo `NFR-001`, con el relleno de ceros que heredará la familia)",
		"o cambia el ADDED a una familia existente",
		`axiom sdd integrate check --cwd "<workspaceRoot>" --change <cambio>`)
}

func TestPlanIDsStopsOnAProvisionalIDDeclaredTwice(t *testing.T) {
	profile := requirementProfile(t)

	t.Run("en el mismo documento", func(t *testing.T) {
		_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta("RF-9001: Uno", "RF-9001: Otro"))})
		assertComposeError(t, err, "RF-9001", "dos requisitos ADDED", `"RF-9001: Uno"`, `"RF-9001: Otro"`)
	})

	t.Run("en documentos distintos", func(t *testing.T) {
		other := join("### RF-300: Otro documento", "", "Cuerpo.")
		_, err := PlanIDs([]PlannedDocument{
			plannedDoc("requisitos", profile, idsDocument, addedDelta("RF-9001: Uno")),
			plannedDoc("otros", profile, other, addedDelta("RF-9001: Otro")),
		})
		composeErr := assertComposeError(t, err, "RF-9001", "dos requisitos ADDED", `dominio "requisitos"`, `dominio "otros"`)
		if composeErr.Domain != "otros" {
			t.Errorf("el rechazo se atribuye al segundo ADDED, dominio = %q", composeErr.Domain)
		}
	})
}

func TestPlanIDsStopsWhenAProvisionalIDEqualsTheDefinitiveOfAnotherRequirement(t *testing.T) {
	profile := requirementProfile(t)
	original := join("### RF-510: Uno", "", "Cuerpo.", "", "### RF-511: Dos", "", "Cuerpo.")

	t.Run("ambos IDs nombrados", func(t *testing.T) {
		_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, original, addedDelta("RF-513: Primero", "RF-512: Segundo"))})
		assertComposeError(t, err, "RF-513", "RF-512", "ambigua")
	})

	t.Run("un provisional igual a su propio definitivo se admite", func(t *testing.T) {
		m, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, original, addedDelta("RF-512: Primero", "RF-513: Segundo"))})
		if err != nil {
			t.Fatalf("PlanIDs() error = %v", err)
		}
		if got, want := definitives(m), map[string]string{"RF-512": "RF-512", "RF-513": "RF-513"}; !reflect.DeepEqual(got, want) {
			t.Errorf("correspondencia = %v, se esperaba %v", got, want)
		}
	})

	t.Run("entre documentos distintos", func(t *testing.T) {
		// El definitivo del primer ADDED es RF-511; el segundo documento declara
		// RF-511 como provisional.
		first := join("### RF-510: Existente", "", "Cuerpo.")
		other := join("### RF-5: Existente", "", "Cuerpo.")
		_, err := PlanIDs([]PlannedDocument{
			plannedDoc("requisitos", profile, first, addedDelta("RF-9001: Primero")),
			plannedDoc("otros", profile, other, addedDelta("RF-511: Segundo")),
		})
		assertComposeError(t, err, "RF-511", "RF-9001", "ambigua")
	})
}

func TestPlanIDsStopsOnAProvisionalIDThatExistsInTheDocument(t *testing.T) {
	profile := requirementProfile(t)
	_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta("RF-002: Colisiona"))})
	assertComposeError(t, err, "RF-002", "coincide con un ID existente", "línea 9", "docs/requisitos.md")
}

func TestPlanIDsRejectsADocumentWithoutACompiledProfile(t *testing.T) {
	_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", Profile{}, idsDocument, addedDelta("RF-9001: Uno"))})
	assertComposeError(t, err, "perfil")
}

func TestPlanIDsRejectsANameThatDoesNotMatchTheHeadingPattern(t *testing.T) {
	profile := requirementProfile(t)
	_, err := PlanIDs([]PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta("Sin identificador"))})
	assertComposeError(t, err, "ADDED", "Sin identificador", "'heading'")
}

func TestPlanIDsIsDeterministic(t *testing.T) {
	profile := requirementProfile(t)
	docs := []PlannedDocument{plannedDoc("requisitos", profile, idsDocument, addedDelta("RF-9003: C", "RF-9001: A", "RF-9002: B"))}
	first, err := PlanIDs(docs)
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	for i := 0; i < 5; i++ {
		again, err := PlanIDs(docs)
		if err != nil {
			t.Fatalf("PlanIDs() error = %v", err)
		}
		if !reflect.DeepEqual(first.Assignments(), again.Assignments()) {
			t.Fatalf("la correspondencia debe ser determinista: %+v frente a %+v", first.Assignments(), again.Assignments())
		}
	}
	// Los definitivos siguen el orden del delta, no el de los IDs provisionales.
	if got, want := definitives(first), map[string]string{"RF-9003": "RF-008", "RF-9001": "RF-009", "RF-9002": "RF-010"}; !reflect.DeepEqual(got, want) {
		t.Errorf("correspondencia = %v, se esperaba %v", got, want)
	}
}

func TestPlanIDsDoesNotChangeItsInput(t *testing.T) {
	profile := requirementProfile(t)
	doc := plannedDoc("requisitos", profile, idsDocument, addedDelta("RF-9001: Uno"))
	original, deltaBytes := string(doc.Original), string(doc.Delta)
	if _, err := PlanIDs([]PlannedDocument{doc}); err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	if string(doc.Original) != original || string(doc.Delta) != deltaBytes {
		t.Error("PlanIDs no debe modificar sus entradas")
	}
}

func TestPlanIDsWorksOnTheThreeByteVariants(t *testing.T) {
	profile := requirementProfile(t)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		doc := PlannedDocument{Domain: "requisitos", Profile: profile, Original: v.apply(idsDocument), Delta: v.apply(addedDelta("RF-9001: Uno"))}
		m, err := PlanIDs([]PlannedDocument{doc})
		if err != nil {
			t.Fatalf("PlanIDs() error = %v", err)
		}
		if got := definitives(m)["RF-9001"]; got != "RF-008" {
			t.Errorf("definitivo = %q, se esperaba RF-008", got)
		}
	})
}

func TestSubstituteReplacesWholeTokensOnly(t *testing.T) {
	m := newIDMap([]IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "d"}})
	tests := []struct {
		name, text, want string
	}{
		{name: "token suelto", text: "RF-9001", want: "RF-008"},
		{name: "ID más largo con el mismo prefijo", text: "RF-90012", want: "RF-90012"},
		{name: "precedido por una letra", text: "ARF-9001", want: "ARF-9001"},
		{name: "precedido por una letra Unicode", text: "ÁRF-9001", want: "ÁRF-9001"},
		{name: "seguido por una letra", text: "RF-9001a", want: "RF-9001a"},
		{name: "seguido por un guion", text: "RF-9001-bis", want: "RF-9001-bis"},
		{name: "precedido por un guion", text: "x-RF-9001", want: "x-RF-9001"},
		{name: "precedido por un guion bajo", text: "_RF-9001", want: "_RF-9001"},
		{name: "seguido por un guion bajo", text: "RF-9001_", want: "RF-9001_"},
		{name: "distinto en mayúsculas", text: "rf-9001", want: "rf-9001"},
		{name: "entre signos de puntuación", text: "(RF-9001), RF-9001.", want: "(RF-008), RF-008."},
		{name: "énfasis de Markdown", text: "**RF-9001** y `RF-9001`", want: "**RF-008** y `RF-008`"},
		{name: "al principio y al final de una línea", text: "RF-9001 depende de RF-9001", want: "RF-008 depende de RF-008"},
		{name: "solo la primera aparición es un token", text: "RF-9001, RF-90012 y ARF-9001", want: "RF-008, RF-90012 y ARF-9001"},
		{name: "con saltos de línea", text: "a\nRF-9001\r\nb", want: "a\nRF-008\r\nb"},
		{name: "sin apariciones", text: "RF-001 y RF-9002", want: "RF-001 y RF-9002"},
		{name: "texto vacío", text: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Substitute(tt.text); got != tt.want {
				t.Errorf("Substitute(%q) = %q, se esperaba %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestSubstituteIsASinglePass(t *testing.T) {
	// Aunque PlanIDs no produce una correspondencia así (caso d), Substitute no
	// debe encadenar sustituciones: el texto sustituido no se vuelve a examinar.
	m := newIDMap([]IDAssignment{
		{Provisional: "RF-9001", Definitive: "RF-9002"},
		{Provisional: "RF-9002", Definitive: "RF-9003"},
	})
	if got, want := m.Substitute("RF-9001 RF-9002"), "RF-9002 RF-9003"; got != want {
		t.Errorf("Substitute() = %q, se esperaba %q", got, want)
	}
}

func TestSubstituteKeepsAProvisionalIDEqualToItsOwnDefinitive(t *testing.T) {
	m := newIDMap([]IDAssignment{{Provisional: "RF-512", Definitive: "RF-512"}})
	if got := m.Substitute("ver RF-512 y RF-5120"); got != "ver RF-512 y RF-5120" {
		t.Errorf("Substitute() = %q", got)
	}
}

func TestSubstituteHandlesSeveralIDsAndCrossReferences(t *testing.T) {
	profile := requirementProfile(t)
	m, err := PlanIDs([]PlannedDocument{plannedDoc("d", profile, idsDocument, addedDelta("RF-9001: Primero", "RF-9002: Segundo"))})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	got := m.Substitute("El requisito RF-9002 depende de RF-9001 y de RF-003.")
	if want := "El requisito RF-009 depende de RF-008 y de RF-003."; got != want {
		t.Errorf("Substitute() = %q, se esperaba %q", got, want)
	}
}

func TestSubstituteNeverTouchesTheOriginalText(t *testing.T) {
	m := newIDMap([]IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008"}})
	text := "RF-9001"
	_ = m.Substitute(text)
	if text != "RF-9001" {
		t.Error("Substitute devuelve un texto nuevo y no modifica el recibido")
	}
	if got := m.Assignments(); len(got) != 1 {
		t.Fatalf("Assignments() = %+v", got)
	}
	m.Assignments()[0].Definitive = "RF-999"
	if got, _ := m.Lookup("RF-9001"); got.Definitive != "RF-008" {
		t.Error("Assignments devuelve una copia")
	}
}

func TestSubstitutePrefersTheLongestProvisionalID(t *testing.T) {
	m := newIDMap([]IDAssignment{
		{Provisional: "A-1", Definitive: "A-10"},
		{Provisional: "A-1.5", Definitive: "A-11"},
	})
	if got, want := m.Substitute("ver A-1.5 y A-1."), "ver A-11 y A-10."; got != want {
		t.Errorf("Substitute() = %q, se esperaba %q", got, want)
	}
	if !strings.Contains(m.Substitute("A-1"), "A-10") {
		t.Error("el ID más corto también se sustituye")
	}
}

// Recuento de provisionales para la verificación (e) (REQ-24.51, D-15, D-20).

func countingMap() IDMap {
	return newIDMap([]IDAssignment{
		{Provisional: "RF-9001", Definitive: "RF-008", Domain: "d"},
		{Provisional: "RF-9002", Definitive: "RF-009", Domain: "d"},
		{Provisional: "RF-010", Definitive: "RF-010", Domain: "d"},
	})
}

func TestCountProvisionalCountsEachCompleteTokenOfAProvisional(t *testing.T) {
	tests := []struct {
		name string
		text string
		want map[string]int
	}{
		{name: "sin texto", text: "", want: map[string]int{}},
		{name: "sin apariciones", text: "Texto sin IDs del cambio.", want: map[string]int{}},
		{name: "una aparición", text: "Véase RF-9001.", want: map[string]int{"RF-9001": 1}},
		{name: "varias apariciones de dos provisionales", text: "RF-9001 depende de RF-9002 y de RF-9001 otra vez", want: map[string]int{"RF-9001": 2, "RF-9002": 1}},
		{name: "entre signos de puntuación", text: "(RF-9001), «RF-9001»; [RF-9001]", want: map[string]int{"RF-9001": 3}},
		{name: "un número más largo no es el token", text: "RF-90010 y RF-90011", want: map[string]int{}},
		{name: "precedido por letra", text: "xRF-9001 y éRF-9001", want: map[string]int{}},
		{name: "precedido o seguido por guion o guion bajo", text: "_RF-9001 RF-9001_ -RF-9001 RF-9001-bis", want: map[string]int{}},
		{name: "precedido por un dígito", text: "7RF-9001", want: map[string]int{}},
		{name: "distingue mayúsculas", text: "rf-9001 Rf-9001", want: map[string]int{}},
		{name: "con salto de línea CRLF", text: "RF-9001\r\nRF-9001\r\n", want: map[string]int{"RF-9001": 2}},
		{name: "un provisional igual a su definitivo no cuenta", text: "RF-010 y RF-010", want: map[string]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countingMap().CountProvisional(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CountProvisional(%q) = %v, se esperaba %v", tt.text, got, tt.want)
			}
		})
	}
}

// El recuento y la sustitución comparten una sola regla de token: tras sustituir,
// no queda ningún provisional que contar, y un texto sin sustituir cuenta
// exactamente las apariciones que la sustitución reemplazaría.
func TestCountProvisionalAgreesWithSubstitute(t *testing.T) {
	m := countingMap()
	text := "RF-9001, RF-9002, RF-90010, xRF-9001, RF-9001-bis, RF-010 y RF-9001.\r\n"
	counts := m.CountProvisional(text)
	if counts["RF-9001"] != 2 || counts["RF-9002"] != 1 {
		t.Fatalf("recuento = %v", counts)
	}
	replaced := m.Substitute(text)
	if after := m.CountProvisional(replaced); len(after) != 0 {
		t.Errorf("tras sustituir no deben quedar provisionales: %v\n%s", after, replaced)
	}
	if got := strings.Count(replaced, "RF-008"); got != counts["RF-9001"] {
		t.Errorf("la sustitución reemplaza %d apariciones de RF-9001 y el recuento dice %d", got, counts["RF-9001"])
	}
}

func TestCountProvisionalPrefersTheLongestProvisionalWithoutOverlaps(t *testing.T) {
	m := newIDMap([]IDAssignment{
		{Provisional: "NFR-9001", Definitive: "NFR-004", Domain: "d"},
		{Provisional: "FR-9001", Definitive: "FR-004", Domain: "d"},
	})
	// FR-9001 aparece dentro de NFR-9001 pero precedido por una letra: no es token.
	got := m.CountProvisional("NFR-9001 y FR-9001")
	if want := map[string]int{"NFR-9001": 1, "FR-9001": 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("CountProvisional() = %v, se esperaba %v", got, want)
	}
}

func TestCountProvisionalOfAnEmptyMapIsEmpty(t *testing.T) {
	if got := (IDMap{}).CountProvisional("RF-9001"); len(got) != 0 {
		t.Errorf("CountProvisional() = %v, se esperaba un mapa vacío", got)
	}
}
