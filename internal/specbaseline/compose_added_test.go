package specbaseline

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de ADDED con ubicación explícita, de la asignación y sustitución de IDs
// en la composición y de `scenarios: omit` (REQ-24.40, 24.43, 24.59 y 24.61).
// Todos los textos son inventados.

// addedDocument tiene IDs no ordenados (el mayor es RF-007, aunque RF-003 va
// después), una sección contigua entre dos requisitos y tres secciones h2.
var addedDocument = join(
	"# Documento de prueba",
	"",
	"## Catálogo",
	"",
	"### RF-001: Primero",
	"",
	"Cuerpo primero.",
	"",
	"### RF-002: Segundo",
	"",
	"Cuerpo segundo.",
	"",
	"### Notas del catálogo",
	"",
	"Sección contigua.",
	"",
	"### RF-007: Séptimo",
	"",
	"Cuerpo séptimo.",
	"",
	"### RF-003: Tercero",
	"",
	"Cuerpo tercero.",
	"",
	"## Pagos",
	"",
	"### RF-004: Cobro",
	"",
	"Cuerpo cobro.",
	"",
	"## Anexos",
	"",
	"Texto de anexos.",
)

// added escribe un requisito ADDED con la nota de ubicación pegada al encabezado.
func added(name, placement string, body ...string) []string {
	lines := []string{"### Requirement: " + name}
	if placement != "" {
		lines = append(lines, "(Placement: "+placement+")")
	}
	lines = append(lines, "")
	lines = append(lines, body...)
	return append(lines, "")
}

// addedDeltaOf arma un delta con la sección ADDED y las secciones de operaciones
// que se indiquen, en LF.
func addedDeltaOf(blocks ...[]string) string {
	lines := []string{"## ADDED Requirements", ""}
	for _, block := range blocks {
		lines = append(lines, block...)
	}
	return join(lines...)
}

// composeAdded planifica los IDs del documento y lo compone, en la variante de bytes.
func composeAdded(t *testing.T, profile Profile, v byteVariant, original, deltaText string, configure ...func(*ComposeInput)) (ComposeResult, error) {
	t.Helper()
	input := composeInput(profile, v.apply(original), []byte(deltaText))
	ids, err := PlanIDs([]PlannedDocument{input.Document})
	if err != nil {
		return ComposeResult{}, err
	}
	input.IDs = ids
	for _, apply := range configure {
		apply(&input)
	}
	return ComposeDocument(input)
}

func TestComposeAddedIsInsertedAfterTheBlockOfItsAnchorWithTheDefinitiveID(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(
		added("RF-9001: Reintento de pago", "after RF-001", "El sistema DEBE reintentar el pago. Depende de RF-9002."),
		added("RF-9002: Aviso de fallo", "after RF-001", "Cuerpo del aviso."),
	)
	want := join(
		"# Documento de prueba",
		"",
		"## Catálogo",
		"",
		"### RF-001: Primero",
		"",
		"Cuerpo primero.",
		"",
		"### RF-008: Reintento de pago",
		"",
		"El sistema DEBE reintentar el pago. Depende de RF-009.",
		"",
		"### RF-009: Aviso de fallo",
		"",
		"Cuerpo del aviso.",
		"",
		"### RF-002: Segundo",
		"",
		"Cuerpo segundo.",
		"",
		"### Notas del catálogo",
		"",
		"Sección contigua.",
		"",
		"### RF-007: Séptimo",
		"",
		"Cuerpo séptimo.",
		"",
		"### RF-003: Tercero",
		"",
		"Cuerpo tercero.",
		"",
		"## Pagos",
		"",
		"### RF-004: Cobro",
		"",
		"Cuerpo cobro.",
		"",
		"## Anexos",
		"",
		"Texto de anexos.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, addedDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		for _, forbidden := range []string{"RF-9001", "RF-9002", "(Placement"} {
			if bytes.Contains(got.Output, []byte(forbidden)) {
				t.Errorf("el documento no debe contener %q", forbidden)
			}
		}
		if len(got.Warnings) != 0 {
			t.Errorf("un ADDED no emite avisos de cola: %q", got.Warnings)
		}
	})
}

func TestComposeAddedTraceRecordsTheProvisionalToDefinitiveMapping(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(
		added("RF-9001: Reintento de pago", "after RF-001", "Cuerpo."),
		added("RF-9002: Aviso de fallo", `section "Pagos"`, "Cuerpo."),
	)
	got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entradas = %d, se esperaban 2: %+v", len(got.Entries), got.Entries)
	}
	first, second := got.Entries[0], got.Entries[1]
	wantFirst := TraceEntry{
		ID: "RF-008", Provisional: "RF-9001", Kind: EntryCompose, Domain: "requisitos-funcionales",
		Document: "docs/01_Requisitos_Funcionales.md", Anchor: "### RF-008: Reintento de pago", Action: ActionAdded, State: StatePending,
	}
	if first.ID != wantFirst.ID || first.Provisional != wantFirst.Provisional || first.Kind != wantFirst.Kind || first.Domain != wantFirst.Domain ||
		first.Document != wantFirst.Document || first.Anchor != wantFirst.Anchor || first.Action != wantFirst.Action || first.State != wantFirst.State || first.ScenariosOmitted {
		t.Errorf("primera entrada = %+v\nse esperaba %+v", first, wantFirst)
	}
	if second.ID != "RF-009" || second.Provisional != "RF-9002" || second.Anchor != "### RF-009: Aviso de fallo" {
		t.Errorf("segunda entrada = %+v", second)
	}
}

func TestComposeAddedKeepsTheContiguousTailWithItsRequirement(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-002", "Cuerpo nuevo."))
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, addedDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		want := join(
			"### RF-002: Segundo",
			"",
			"Cuerpo segundo.",
			"",
			"### Notas del catálogo",
			"",
			"Sección contigua.",
			"",
			"### RF-008: Nuevo",
			"",
			"Cuerpo nuevo.",
			"",
			"### RF-007: Séptimo",
		)
		if !bytes.Contains(got.Output, v.snippet(want)) {
			t.Errorf("el orden debe ser RF-002, la sección contigua, el nuevo y RF-007:\n%s", got.Output)
		}
	})
}

func TestComposeAddedKeepsTheOrderOfTheDeltaForTheSameAnchor(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(
		added("RF-9003: Tercero nuevo", "after RF-002", "C."),
		added("RF-9001: Primero nuevo", "after RF-002", "A."),
		added("RF-9002: Segundo nuevo", "after RF-002", "B."),
	)
	got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	var order []string
	for _, line := range strings.Split(string(got.Output), "\n") {
		if strings.HasPrefix(line, "### ") {
			order = append(order, strings.TrimPrefix(line, "### "))
		}
	}
	want := []string{"RF-001: Primero", "RF-002: Segundo", "Notas del catálogo", "RF-008: Tercero nuevo", "RF-009: Primero nuevo", "RF-010: Segundo nuevo", "RF-007: Séptimo", "RF-003: Tercero", "RF-004: Cobro"}
	if strings.Join(order, "|") != strings.Join(want, "|") {
		t.Errorf("orden de los encabezados = %q\nse esperaba %q", order, want)
	}
}

func TestComposeAddedWithSectionPlacementGoesAtTheEndOfTheSection(t *testing.T) {
	profile := requirementProfile(t)
	t.Run("antes del siguiente encabezado de nivel 2", func(t *testing.T) {
		deltaText := addedDeltaOf(added("RF-9001: Reembolso", `section "Pagos"`, "Cuerpo."))
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			got, err := composeAdded(t, profile, v, addedDocument, deltaText)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			want := join(
				"## Pagos",
				"",
				"### RF-004: Cobro",
				"",
				"Cuerpo cobro.",
				"",
				"### RF-008: Reembolso",
				"",
				"Cuerpo.",
				"",
				"## Anexos",
			)
			if !bytes.Contains(got.Output, v.snippet(want)) {
				t.Errorf("el nuevo requisito debe ir al final de la sección «Pagos»:\n%s", got.Output)
			}
		})
	})

	t.Run("al final del fichero", func(t *testing.T) {
		deltaText := addedDeltaOf(added("RF-9001: Anexo nuevo", `section "Anexos"`, "Cuerpo."))
		got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if want := addedDocument + "\n### RF-008: Anexo nuevo\n\nCuerpo.\n"; string(got.Output) != want {
			t.Errorf("salida = %q\nse esperaba %q", got.Output, want)
		}
	})

	t.Run("tras los requisitos y la cola de la sección", func(t *testing.T) {
		reference := readFixture(t, "01_Requisitos_Funcionales.md")
		deltaText := addedDeltaOf(added("RF-9001: Cobro con vale", `section "Pagos"`, "Cuerpo."))
		got, err := composeAdded(t, profile, byteVariants[1], reference, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		want := "### RF-021: Cobro con transferencia\n\nEl sistema DEBE admitir el pago por transferencia bancaria.\n\n### RF-051: Cobro con vale\n\nCuerpo.\n\n## Devoluciones\n"
		if !bytes.Contains(got.Output, byteVariants[1].apply(want)) {
			t.Errorf("el nuevo requisito debe ir antes de «## Devoluciones»:\n%s", got.Output)
		}
	})
}

func TestComposeAddedRejectsASectionTitleThatIsMissingOrRepeated(t *testing.T) {
	profile := requirementProfile(t)
	repeated := addedDocument + "\n## Pagos\n\nOtra sección con el mismo título.\n"
	tests := []struct {
		name, document, title, want string
	}{
		{name: "inexistente", document: addedDocument, title: "Notas", want: "0 veces"},
		{name: "repetido", document: repeated, title: "Pagos", want: "2 veces"},
		{name: "un encabezado de otro nivel no cuenta", document: addedDocument, title: "RF-004: Cobro", want: "0 veces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deltaText := addedDeltaOf(added("RF-9001: Nuevo", `section "`+tt.title+`"`, "Cuerpo."))
			got, err := composeAdded(t, profile, byteVariants[0], tt.document, deltaText)
			assertComposeError(t, err, "ADDED", tt.title, tt.want, "Continuación")
			if got.Output != nil {
				t.Error("un rechazo no devuelve resultado parcial")
			}
		})
	}
}

func TestComposeAddedRequiresAnExplicitPlacementWithIDIdentity(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Sin ubicación", "", "Cuerpo."))
	got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
	assertComposeError(t, err, "ADDED", "RF-9001", "ubicación", "explícita", "(Placement: after <ID>)")
	if got.Output != nil {
		t.Error("un rechazo no devuelve resultado parcial")
	}
}

func TestComposeAddedRejectsAMalformedPlacement(t *testing.T) {
	profile := requirementProfile(t)
	for name, placement := range map[string]string{
		"sin forma conocida":   "before RF-001",
		"after sin ID":         "after",
		"section sin comillas": "section Pagos",
		"section vacío":        `section ""`,
	} {
		t.Run(name, func(t *testing.T) {
			deltaText := addedDeltaOf(added("RF-9001: Nuevo", placement, "Cuerpo."))
			_, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
			assertComposeError(t, err, "ADDED", "formato", "(Placement: "+placement+")")
		})
	}

	t.Run("varias notas", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "", "### Requirement: RF-9001: Nuevo", "(Placement: after RF-001)", "(Placement: after RF-002)", "", "Cuerpo.")
		_, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
		assertComposeError(t, err, "ADDED", "2 notas")
	})
}

func TestComposeAddedAnchorMustBeAUniqueExistingRequirement(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales.md")

	t.Run("duplicado", func(t *testing.T) {
		deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-020", "Cuerpo."))
		_, err := composeAdded(t, profile, byteVariants[0], reference, deltaText)
		assertComposeError(t, err, "ADDED", "RF-020", "ambiguo", "Cobro con tarjeta", "Devolución parcial", "Continuación")
	})

	t.Run("un ID único sigue siendo operable aunque haya otros duplicados", func(t *testing.T) {
		deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-050", "Cuerpo."))
		got, err := composeAdded(t, profile, byteVariants[0], reference, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if !strings.HasSuffix(string(got.Output), "### RF-050: Informe mensual\n\nEl sistema DEBE generar un informe mensual de ventas.\n\n### RF-051: Nuevo\n\nCuerpo.\n") {
			t.Errorf("salida = %q", got.Output)
		}
	})

	t.Run("inexistente", func(t *testing.T) {
		deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-999", "Cuerpo."))
		_, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
		assertComposeError(t, err, "ADDED", "RF-999", "no contiene")
	})

	t.Run("eliminado por el mismo delta sin supersede", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "") + strings.Join(added("RF-9001: Nuevo", "after RF-002", "Cuerpo."), "\n") +
			"\n## REMOVED Requirements\n\n### Requirement: RF-002: Segundo\n\n(Reason: se retira)\n"
		_, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
		assertComposeError(t, err, "ADDED", "RF-002", "elimina", "--supersede")
	})

	t.Run("eliminado con supersede sigue siendo un ancla", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "") + strings.Join(added("RF-9001: Nuevo", "after RF-002", "Cuerpo."), "\n") +
			"\n## REMOVED Requirements\n\n### Requirement: RF-002: Segundo\n\n(Reason: se retira)\n"
		got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText, func(in *ComposeInput) { in.Supersede = true })
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if !strings.Contains(string(got.Output), "### RF-002: Segundo [SUPERSEDED / DEPRECADO]") ||
			strings.Index(string(got.Output), "### RF-008: Nuevo") < strings.Index(string(got.Output), "Sección contigua.") {
			t.Errorf("el nuevo requisito debe ir tras el bloque superado y su cola:\n%s", got.Output)
		}
	})

	t.Run("un ancla ambigua no impide operar sobre otros", func(t *testing.T) {
		deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-050: Informe mensual", "", "Cuerpo nuevo.")
		if _, err := composeAdded(t, profile, byteVariants[0], reference, deltaText); err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
	})
}

func TestComposeAddedLocationIsEvaluatedAfterTheOtherOperations(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## ADDED Requirements", "",
		"### Requirement: RF-9001: Nuevo",
		"(Placement: after RF-001)", "",
		"Cuerpo nuevo.", "",
		"## MODIFIED Requirements", "",
		"### Requirement: RF-001: Primero ampliado", "",
		"Cuerpo primero ampliado.", "",
		"#### Escenario nuevo", "",
		"Texto.", "",
		"## REMOVED Requirements", "",
		"### Requirement: RF-002: Segundo", "",
		"(Reason: se retira)",
	)
	got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	want := join(
		"## Catálogo",
		"",
		"### RF-001: Primero ampliado",
		"",
		"Cuerpo primero ampliado.",
		"",
		"#### Escenario nuevo",
		"",
		"Texto.",
		"",
		"### Notas del catálogo",
		"",
		"Sección contigua.",
		"",
		"### RF-008: Nuevo",
		"",
		"Cuerpo nuevo.",
		"",
		"### RF-007: Séptimo",
	)
	// Eliminado RF-002, su sección contigua pasa a seguir a RF-001: la cola se
	// evalúa sobre el documento tras las demás operaciones.
	if !strings.Contains(string(got.Output), want) {
		t.Errorf("el ADDED debe ir tras el bloque modificado y la cola que ahora le sigue:\n%s", got.Output)
	}
	if strings.Contains(string(got.Output), "RF-002") {
		t.Error("RF-002 debe haberse eliminado")
	}
	if got.Entries[len(got.Entries)-1].Action != ActionAdded {
		t.Errorf("la entrada del ADDED va tras las de MODIFIED y REMOVED: %+v", got.Entries)
	}
}

func TestComposeAddedKeepsEveryByteOutsideTheInsertedBlocks(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales.md")
	deltaText := addedDeltaOf(
		added("RF-9001: Tras la cola", "after RF-003", "Cuerpo uno."),
		added("RF-9002: En informes", `section "Informes"`, "Cuerpo dos."),
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, reference, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		original := v.apply(reference)
		output := got.Output
		if v.bom {
			original, output = original[len(utf8BOM):], output[len(utf8BOM):]
		}
		for _, block := range []string{
			"### RF-051: Tras la cola\n\nCuerpo uno.\n\n",
			"\n### RF-052: En informes\n\nCuerpo dos.\n",
		} {
			if !bytes.Contains(output, []byte(strings.ReplaceAll(block, "\n", v.eol()))) {
				t.Fatalf("falta el bloque %q en la salida", block)
			}
			output = bytes.Replace(output, []byte(strings.ReplaceAll(block, "\n", v.eol())), nil, 1)
		}
		if !bytes.Equal(output, original) {
			t.Errorf("fuera de los bloques insertados, la salida debe conservar los bytes del documento")
		}
		// RF-003 va seguido de dos secciones contiguas: el nuevo bloque va tras ellas.
		text := strings.ReplaceAll(string(got.Output), v.eol(), "\n")
		if strings.Index(text, "### RF-051") < strings.Index(text, "### Glosario del catálogo") || strings.Index(text, "### RF-051") > strings.Index(text, "### RF-040") {
			t.Errorf("RF-051 debe ir tras la cola contigua de RF-003 y antes de RF-040:\n%s", text)
		}
	})
}

func TestComposeAddedInheritsTheBlankLinesOfTheReferenceBlock(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-1", "Cuerpo."))
	tests := []struct {
		name, original, want string
	}{
		{
			name:     "dos líneas en blanco",
			original: "### RF-1: Uno\n\ntexto\n\n\n### RF-2: Dos\n\ntexto\n",
			want:     "### RF-1: Uno\n\ntexto\n\n\n### RF-3: Nuevo\n\nCuerpo.\n\n\n### RF-2: Dos\n\ntexto\n",
		},
		{
			name:     "sin líneas en blanco entre bloques: una delante y mínimo una detrás",
			original: "### RF-1: Uno\ntexto\n### RF-2: Dos\ntexto\n",
			want:     "### RF-1: Uno\ntexto\n\n### RF-3: Nuevo\n\nCuerpo.\n\n### RF-2: Dos\ntexto\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				got, err := composeAdded(t, profile, v, tt.original, deltaText)
				if err != nil {
					t.Fatalf("ComposeDocument() error = %v", err)
				}
				assertOutput(t, v, got, tt.want)
			})
		})
	}
}

func TestComposeAddedAtTheEndOfAFileWithoutFinalNewline(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-2", "Cuerpo."))
	original := "### RF-1: Uno\n\ntexto\n\n### RF-2: Dos\n\ntexto"
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		// La última línea recibe el final de línea del documento (el único byte
		// añadido fuera del bloque insertado) y el nuevo bloque termina sin él.
		assertOutput(t, v, got, "### RF-1: Uno\n\ntexto\n\n### RF-2: Dos\n\ntexto\n\n### RF-3: Nuevo\n\nCuerpo.")
	})

	t.Run("con final de línea", func(t *testing.T) {
		got, err := composeAdded(t, profile, byteVariants[1], original+"\n", deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, byteVariants[1], got, "### RF-1: Uno\n\ntexto\n\n### RF-2: Dos\n\ntexto\n\n### RF-3: Nuevo\n\nCuerpo.\n")
	})
}

func TestComposeAddedWithTextIdentityDefaultsToAfterTheLastBlockAndItsTail(t *testing.T) {
	profile := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)
	original := join(
		"# Documento", "",
		"## Requisitos", "",
		"### Alfa", "", "Cuerpo alfa.", "",
		"### Beta", "", "Cuerpo beta.", "",
		"## Anexos", "", "Texto.",
	)
	deltaText := join("## ADDED Requirements", "", "### Requirement: Gamma", "", "Cuerpo gamma.", "", "### Requirement: Delta", "", "Cuerpo delta.")
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, join(
			"# Documento", "",
			"## Requisitos", "",
			"### Alfa", "", "Cuerpo alfa.", "",
			"### Beta", "", "Cuerpo beta.", "",
			"### Gamma", "", "Cuerpo gamma.", "",
			"### Delta", "", "Cuerpo delta.", "",
			"## Anexos", "", "Texto.",
		))
		if got.Entries[0].ID != "Gamma" || got.Entries[0].Provisional != "" || got.Entries[0].Action != ActionAdded {
			t.Errorf("con identidad text no hay ID provisional: %+v", got.Entries[0])
		}
	})

	t.Run("con ubicación explícita", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "", "### Requirement: Gamma", "(Placement: after Alfa)", "", "Cuerpo gamma.")
		got, err := composeAdded(t, profile, byteVariants[0], original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if !strings.Contains(string(got.Output), "### Alfa\n\nCuerpo alfa.\n\n### Gamma\n\nCuerpo gamma.\n\n### Beta") {
			t.Errorf("salida = %q", got.Output)
		}
	})

	t.Run("un título existente se rechaza", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "", "### Requirement: Alfa", "", "Cuerpo.")
		_, err := composeAdded(t, profile, byteVariants[0], original, deltaText)
		assertComposeError(t, err, "ADDED", "Alfa", "ya existe")
	})

	t.Run("sin ningún requisito va al final del documento", func(t *testing.T) {
		deltaText := join("## ADDED Requirements", "", "### Requirement: Gamma", "", "Cuerpo gamma.")
		got, err := composeAdded(t, profile, byteVariants[0], "# Documento vacío\n", deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if want := "# Documento vacío\n\n### Gamma\n\nCuerpo gamma.\n"; string(got.Output) != want {
			t.Errorf("salida = %q, se esperaba %q", got.Output, want)
		}
	})
}

func TestComposeAddedWithTextIdentityMatchesTheNativeEngine(t *testing.T) {
	canonical := join(
		"## Requirements", "",
		"### Requirement: Alpha", "", "Alpha body.", "",
		"### Requirement: Beta", "", "Beta body.", "",
		"## Appendix", "", "Appendix text.",
	)
	deltaText := join(
		"## ADDED Requirements", "",
		"### Requirement: Gamma", "", "Gamma body.", "",
		"### Requirement: Delta", "", "Delta body.", "",
		"## MODIFIED Requirements", "",
		"### Requirement: Alpha", "", "Alpha body changed.", "",
	)
	native, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonical, deltaText, sddstatus.ComposeOptions{})
	if err != nil {
		t.Fatalf("motor nativo: %v", err)
	}
	profile := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)
	got, err := ComposeDocument(composeInput(profile, []byte(toProfileHeadings(canonical)), []byte(deltaText)))
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if want := toProfileHeadings(native); string(got.Output) != want {
		t.Errorf("la composición con identidad text debe coincidir con el motor nativo\n--- con perfil ---\n%s\n--- nativo ---\n%s", got.Output, want)
	}
}

func TestComposeAddedIsAllOrNothing(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(
		added("RF-9001: Bueno", "after RF-001", "Cuerpo."),
		added("RF-9002: Malo", "after RF-999", "Cuerpo."),
	)
	original := []byte(addedDocument)
	input := composeInput(profile, original, []byte(deltaText))
	ids, err := PlanIDs([]PlannedDocument{input.Document})
	if err != nil {
		t.Fatalf("PlanIDs() error = %v", err)
	}
	input.IDs = ids
	got, err := ComposeDocument(input)
	assertComposeError(t, err, "ADDED", "RF-999")
	if got.Output != nil || got.Entries != nil || got.Warnings != nil {
		t.Errorf("un rechazo no debe devolver resultado parcial: %+v", got)
	}
	if string(original) != addedDocument {
		t.Error("la composición no debe modificar la entrada")
	}
}

func TestComposeAddedWithoutTheIDMapIsRefused(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Nuevo", "after RF-001", "Cuerpo."))
	_, err := ComposeDocument(composeInput(profile, []byte(addedDocument), []byte(deltaText)))
	assertComposeError(t, err, "ADDED", "RF-9001", "PlanIDs")
}

func TestComposeSubstitutesProvisionalIDsOnlyInTheIntegratedText(t *testing.T) {
	profile := requirementProfile(t)
	original := join(
		"### RF-001: Primero", "",
		"Texto preexistente que cita RF-9001 con otro sentido.", "",
		"### RF-002: Segundo", "",
		"Cuerpo segundo.", "",
		"### RF-003: Tercero", "",
		"Cuerpo tercero.",
	)
	deltaText := join(
		"## ADDED Requirements", "",
		"### Requirement: RF-9001: Reintento",
		"(Placement: after RF-001)", "",
		"Cuerpo del reintento.", "",
		"### Requirement: RF-9002: Aviso",
		"(Placement: after RF-001)", "",
		"Depende de RF-9001, de RF-90012 y de ARF-9001. Ver RF-9002.", "",
		"## MODIFIED Requirements", "",
		"### Requirement: RF-002: Segundo con referencia a RF-9001", "",
		"Cuerpo segundo que reutiliza RF-9001.", "",
		"## REMOVED Requirements", "",
		"### Requirement: RF-003: Tercero", "",
		"(Reason: superado por RF-9002)",
	)
	got, err := composeAdded(t, profile, byteVariants[0], original, deltaText, func(in *ComposeInput) { in.Supersede = true })
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	want := join(
		"### RF-001: Primero", "",
		"Texto preexistente que cita RF-9001 con otro sentido.", "",
		"### RF-004: Reintento", "",
		"Cuerpo del reintento.", "",
		"### RF-005: Aviso", "",
		"Depende de RF-004, de RF-90012 y de ARF-9001. Ver RF-005.", "",
		"### RF-002: Segundo con referencia a RF-004", "",
		"Cuerpo segundo que reutiliza RF-004.", "",
		"### RF-003: Tercero [SUPERSEDED / DEPRECADO]",
		"> [!CAUTION]",
		"> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.",
		"> Motivo: superado por RF-005",
		"",
		"",
		"Cuerpo tercero.",
	)
	if string(got.Output) != want {
		t.Errorf("salida distinta de la esperada\n--- obtenida ---\n%s\n--- esperada ---\n%s", got.Output, want)
	}
	var removed TraceEntry
	for _, entry := range got.Entries {
		if entry.Action == ActionRemoved {
			removed = entry
		}
	}
	if removed.Reason != "superado por RF-005" {
		t.Errorf("la traza del REMOVED registra el motivo sustituido: %+v", removed)
	}
}

func TestComposeScenariosOmitIntegratesOnlyTheDescription(t *testing.T) {
	omit := profileFor(t, `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosOmit)
	original := join(
		"### RF-001: Primero", "", "Cuerpo primero.", "",
		"### RF-002: Segundo", "", "Cuerpo segundo.",
	)
	deltaText := join(
		"## ADDED Requirements", "",
		"### Requirement: RF-9001: Con escenarios",
		"(Placement: after RF-001)", "",
		"Descripción del requisito.", "",
		"```md", "#### Scenario: dentro de una valla", "```", "",
		"#### Scenario: primero", "",
		"- **CUANDO** ocurre", "- **ENTONCES** pasa", "",
		"#### Scenario: segundo", "",
		"```md", "#### Notas falsas dentro de una valla del escenario", "```", "",
		"#### Notas de implantación", "",
		"Texto de las notas.", "",
		"#### Scenario: tercero", "",
		"- **CUANDO** otra vez", "",
		"### Requirement: RF-9002: Sin escenarios",
		"(Placement: after RF-001)", "",
		"Solo descripción.", "",
		"## MODIFIED Requirements", "",
		"### Requirement: RF-002: Segundo modificado", "",
		"Descripción nueva.", "",
		"#### Scenario: único", "",
		"- **CUANDO** cambia", "",
		"### Requirement: RF-001: Primero sin cambios de escenarios", "",
		"Solo cambia la descripción.",
	)
	want := join(
		"### RF-001: Primero sin cambios de escenarios", "",
		"Solo cambia la descripción.", "",
		"### RF-003: Con escenarios", "",
		"Descripción del requisito.", "",
		"```md", "#### Scenario: dentro de una valla", "```", "",
		"#### Notas de implantación", "",
		"Texto de las notas.", "",
		"### RF-004: Sin escenarios", "",
		"Solo descripción.", "",
		"### RF-002: Segundo modificado", "",
		"Descripción nueva.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, omit, v, original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		if bytes.Contains(got.Output, []byte("**CUANDO**")) {
			t.Error("el documento no debe contener el contenido de los escenarios")
		}
		omitted := map[string]bool{}
		for _, entry := range got.Entries {
			omitted[entry.ID] = entry.ScenariosOmitted
		}
		wantOmitted := map[string]bool{"RF-002": true, "RF-001": false, "RF-003": true, "RF-004": false}
		for id, flag := range wantOmitted {
			if omitted[id] != flag {
				t.Errorf("scenarios_omitted de %s = %v, se esperaba %v", id, omitted[id], flag)
			}
		}
	})
}

func TestComposeScenariosOmitDoesNotChangeRemoved(t *testing.T) {
	omit := profileFor(t, `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosOmit)
	original := join("### RF-001: Primero", "", "Cuerpo.", "", "#### Scenario: uno", "", "- **CUANDO** x", "", "### RF-002: Segundo", "", "Cuerpo.")
	deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-001: Primero", "", "(Reason: se retira)")
	got, err := composeAdded(t, omit, byteVariants[0], original, deltaText, func(in *ComposeInput) { in.Supersede = true })
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if !strings.Contains(string(got.Output), "#### Scenario: uno\n\n- **CUANDO** x\n") {
		t.Errorf("REMOVED conserva el bloque original con sus escenarios:\n%s", got.Output)
	}
	if got.Entries[0].ScenariosOmitted {
		t.Error("REMOVED no registra omisión de escenarios")
	}
}

func TestComposeScenariosIncludeIntegratesTheWholeBody(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := addedDeltaOf(added("RF-9001: Con escenarios", "after RF-001", "Descripción.", "", "#### Scenario: uno", "", "- **CUANDO** x", "", "#### Scenario: dos", "", "- **CUANDO** y"))
	got, err := composeAdded(t, profile, byteVariants[0], addedDocument, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	want := "### RF-008: Con escenarios\n\nDescripción.\n\n#### Scenario: uno\n\n- **CUANDO** x\n\n#### Scenario: dos\n\n- **CUANDO** y\n\n### RF-002"
	if !strings.Contains(string(got.Output), want) {
		t.Errorf("con include el cuerpo pasa íntegro:\n%s", got.Output)
	}
	if got.Entries[0].ScenariosOmitted {
		t.Error("con include no se registra omisión")
	}
}

func TestComposeBodyRuleAppliesToTheIntegratedText(t *testing.T) {
	// Con nivel 4 y omit los escenarios (nivel 4) no cuentan: se omiten antes de
	// comprobar el cuerpo. Con include el mismo cuerpo cortaría el bloque.
	body := []string{"Descripción.", "", "#### Scenario: uno", "", "- **CUANDO** x"}
	document := join("#### RF-001: Primero", "", "Cuerpo.", "", "#### RF-002: Segundo", "", "Cuerpo.")
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-001: Primero nuevo", "", body[0], "", body[2], "", body[4])

	omit := profileFor(t, `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, 4, workspace.IdentityID, "#### {id}: {title}", workspace.ScenariosOmit)
	got, err := composeAdded(t, omit, byteVariants[0], document, deltaText)
	if err != nil {
		t.Fatalf("con omit: ComposeDocument() error = %v", err)
	}
	if want := "#### RF-001: Primero nuevo\n\nDescripción.\n\n#### RF-002: Segundo"; !strings.Contains(string(got.Output), want) {
		t.Errorf("salida = %q", got.Output)
	}

	include := profileFor(t, `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, 4, workspace.IdentityID, "#### {id}: {title}", workspace.ScenariosInclude)
	_, err = composeAdded(t, include, byteVariants[0], document, deltaText)
	assertComposeError(t, err, "MODIFIED", "RF-001", "#### Scenario: uno", "línea")

	t.Run("un encabezado que no es un escenario sigue contando", func(t *testing.T) {
		withNotes := join("## MODIFIED Requirements", "", "### Requirement: RF-001: Primero nuevo", "", "Descripción.", "", "#### Notas de implantación", "", "Texto.")
		_, err := composeAdded(t, omit, byteVariants[0], document, withNotes)
		assertComposeError(t, err, "MODIFIED", "#### Notas de implantación")
	})
}

func TestComposeRefusesADocumentWithoutACompiledProfile(t *testing.T) {
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-101: Baja", "", "Cuerpo.")
	_, err := composeVariant(t, Profile{}, byteVariants[0], baseDocument, deltaText)
	assertComposeError(t, err, "perfil")
}

func TestOmitScenarios(t *testing.T) {
	lines := func(texts ...string) []deltaLine {
		out := make([]deltaLine, len(texts))
		for i, text := range texts {
			out[i] = deltaLine{Text: text, Number: i + 1}
		}
		return out
	}
	texts := func(body []deltaLine) []string {
		var out []string
		for _, line := range body {
			out = append(out, line.Text)
		}
		return out
	}
	tests := []struct {
		name        string
		body        []string
		want        []string
		wantOmitted bool
	}{
		{name: "sin escenarios", body: []string{"Descripción.", "", "Más texto."}, want: []string{"Descripción.", "", "Más texto."}},
		{name: "un escenario al final", body: []string{"Descripción.", "", "#### Scenario: a", "", "- x"}, want: []string{"Descripción."}, wantOmitted: true},
		{name: "escenario entre texto y notas", body: []string{"Desc.", "", "#### Scenario: a", "- x", "", "#### Notas", "Texto."}, want: []string{"Desc.", "", "#### Notas", "Texto."}, wantOmitted: true},
		{name: "valla con un escenario falso", body: []string{"```", "#### Scenario: falso", "```", "Fin."}, want: []string{"```", "#### Scenario: falso", "```", "Fin."}},
		{name: "un encabezado de nivel menor cierra el escenario", body: []string{"#### Scenario: a", "- x", "### Otro", "Texto."}, want: []string{"### Otro", "Texto."}, wantOmitted: true},
		{name: "un encabezado de nivel 5 pertenece al escenario", body: []string{"Desc.", "#### Scenario: a", "##### Detalle", "- x"}, want: []string{"Desc."}, wantOmitted: true},
		{name: "nivel 3 no es un escenario", body: []string{"### Scenario: no", "Texto."}, want: []string{"### Scenario: no", "Texto."}},
		{name: "sin los dos puntos no es un escenario", body: []string{"#### Scenario sin dos puntos", "Texto."}, want: []string{"#### Scenario sin dos puntos", "Texto."}},
		{name: "solo escenarios", body: []string{"#### Scenario: a", "- x"}, want: nil, wantOmitted: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, omitted := omitScenarios(lines(tt.body...))
			if strings.Join(texts(got), "\n") != strings.Join(tt.want, "\n") || omitted != tt.wantOmitted {
				t.Errorf("omitScenarios() = (%q, %v), se esperaba (%q, %v)", texts(got), omitted, tt.want, tt.wantOmitted)
			}
		})
	}
}

// snippet convierte un fragmento escrito en LF al final de línea de la variante,
// sin anteponer el BOM: sirve para buscar un tramo dentro de la salida.
func (v byteVariant) snippet(lf string) []byte {
	return byteVariant{crlf: v.crlf}.apply(lf)
}

func TestComposeAddedSplitsTheTextThatFollowsTheAnchorBlock(t *testing.T) {
	profile := requirementProfile(t)
	original := join(
		"## Catálogo", "",
		"### RF-001: Uno", "", "Cuerpo.", "",
		"### Notas", "", "Texto de notas.", "",
		"## Anexos", "", "Texto de anexos.", "",
		"## Fin", "", "Cierre.",
	)
	deltaText := addedDeltaOf(
		added("RF-9001: Tras la cola", "after RF-001", "A."),
		added("RF-9002: En anexos", `section "Anexos"`, "B."),
		added("RF-9003: Otra tras la cola", "after RF-001", "C."),
		added("RF-9004: Otra en anexos", `section "Anexos"`, "D."),
	)
	want := join(
		"## Catálogo", "",
		"### RF-001: Uno", "", "Cuerpo.", "",
		"### Notas", "", "Texto de notas.", "",
		"### RF-002: Tras la cola", "", "A.", "",
		"### RF-004: Otra tras la cola", "", "C.", "",
		"## Anexos", "", "Texto de anexos.", "",
		"### RF-003: En anexos", "", "B.", "",
		"### RF-005: Otra en anexos", "", "D.", "",
		"## Fin", "", "Cierre.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeAdded(t, profile, v, original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
	})
}

// Un ADDED insertado tras una línea que no está en blanco recibe una línea en
// blanco delante; detrás hereda la serie del bloque de referencia con un mínimo de
// una salvo al final del fichero. Solo se insertan bytes (D-12, REQ-24.45).
func TestComposeAddedSeparatesTheBlockFromANonBlankPreviousLine(t *testing.T) {
	idProfile := requirementProfile(t)
	textProfile := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)
	afterRF1 := addedDeltaOf(added("RF-9001: Nuevo", "after RF-1", "Cuerpo."))
	sectionPagos := addedDeltaOf(added("RF-9001: Nuevo", `section "Pagos"`, "Cuerpo."))
	textDelta := join("## ADDED Requirements", "", "### Requirement: Gamma", "", "Cuerpo gamma.")
	tests := []struct {
		name, original, delta, want string
		profile                     Profile
	}{
		{
			name:     "after ID con referencia sin blancos finales",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\ntexto\n### RF-2: Dos\ntexto\n",
			want:     "### RF-1: Uno\ntexto\n\n### RF-3: Nuevo\n\nCuerpo.\n\n### RF-2: Dos\ntexto\n",
		},
		{
			name:     "after ID con referencia sin blancos finales y varios bloques insertados",
			profile:  idProfile,
			delta:    addedDeltaOf(added("RF-9001: Uno nuevo", "after RF-1", "A."), added("RF-9002: Otro nuevo", "after RF-1", "B.")),
			original: "### RF-1: Uno\ntexto\n### RF-2: Dos\ntexto\n",
			want:     "### RF-1: Uno\ntexto\n\n### RF-3: Uno nuevo\n\nA.\n\n### RF-4: Otro nuevo\n\nB.\n\n### RF-2: Dos\ntexto\n",
		},
		{
			name:     "control: after ID con referencia con blancos finales no cambia",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\n\ntexto\n\n### RF-2: Dos\n\ntexto\n",
			want:     "### RF-1: Uno\n\ntexto\n\n### RF-3: Nuevo\n\nCuerpo.\n\n### RF-2: Dos\n\ntexto\n",
		},
		{
			name:     "control: la serie de dos blancos se hereda detrás y delante no se añade",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\n\ntexto\n\n\n### RF-2: Dos\n\ntexto\n",
			want:     "### RF-1: Uno\n\ntexto\n\n\n### RF-3: Nuevo\n\nCuerpo.\n\n\n### RF-2: Dos\n\ntexto\n",
		},
		{
			name:     "section cuyo último bloque no tiene blancos finales",
			profile:  idProfile,
			delta:    sectionPagos,
			original: "## Pagos\n\n### RF-1: Uno\ntexto\n## Anexos\n\ntexto\n",
			want:     "## Pagos\n\n### RF-1: Uno\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.\n\n## Anexos\n\ntexto\n",
		},
		{
			name:     "section cuyo último bloque es el encabezado de la sección",
			profile:  idProfile,
			delta:    sectionPagos,
			original: "## Previo\n\n### RF-1: Uno\n\ntexto\n\n## Pagos\n## Anexos\n\ntexto\n",
			want:     "## Previo\n\n### RF-1: Uno\n\ntexto\n\n## Pagos\n\n### RF-2: Nuevo\n\nCuerpo.\n\n## Anexos\n\ntexto\n",
		},
		{
			name:     "control: section con blancos finales no cambia",
			profile:  idProfile,
			delta:    sectionPagos,
			original: "## Pagos\n\n### RF-1: Uno\n\ntexto\n\n## Anexos\n\ntexto\n",
			want:     "## Pagos\n\n### RF-1: Uno\n\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.\n\n## Anexos\n\ntexto\n",
		},
		{
			name:     "final del fichero con salto final tras un bloque sin blancos",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\ntexto\n",
			want:     "### RF-1: Uno\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.\n",
		},
		{
			name:     "final del fichero sin salto final tras un bloque sin blancos",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\ntexto",
			want:     "### RF-1: Uno\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.",
		},
		{
			name:     "section al final del fichero sin salto final tras un bloque sin blancos",
			profile:  idProfile,
			delta:    sectionPagos,
			original: "## Pagos\n\n### RF-1: Uno\ntexto",
			want:     "## Pagos\n\n### RF-1: Uno\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.",
		},
		{
			name:     "control: final del fichero tras un bloque con blancos hereda la serie",
			profile:  idProfile,
			delta:    afterRF1,
			original: "### RF-1: Uno\n\ntexto\n\n",
			want:     "### RF-1: Uno\n\ntexto\n\n### RF-2: Nuevo\n\nCuerpo.\n\n",
		},
		{
			name:     "identidad text por defecto tras un último bloque sin blancos",
			profile:  textProfile,
			delta:    textDelta,
			original: "### Alfa\ntexto\n### Beta\ntexto\n",
			want:     "### Alfa\ntexto\n### Beta\ntexto\n\n### Gamma\n\nCuerpo gamma.\n",
		},
		{
			name:     "identidad text por defecto al final del fichero sin salto final",
			profile:  textProfile,
			delta:    textDelta,
			original: "### Alfa\ntexto\n### Beta\ntexto",
			want:     "### Alfa\ntexto\n### Beta\ntexto\n\n### Gamma\n\nCuerpo gamma.",
		},
		{
			name:     "identidad text por defecto sin requisitos tras una línea sin blancos",
			profile:  textProfile,
			delta:    textDelta,
			original: "# Documento\ntexto",
			want:     "# Documento\ntexto\n\n### Gamma\n\nCuerpo gamma.",
		},
		{
			name:     "control: identidad text por defecto con blancos finales no cambia",
			profile:  textProfile,
			delta:    textDelta,
			original: "### Alfa\n\ntexto\n\n### Beta\n\ntexto\n\n",
			want:     "### Alfa\n\ntexto\n\n### Beta\n\ntexto\n\n### Gamma\n\nCuerpo gamma.\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				got, err := composeAdded(t, tt.profile, v, tt.original, tt.delta)
				if err != nil {
					t.Fatalf("ComposeDocument() error = %v", err)
				}
				assertOutput(t, v, got, tt.want)
			})
		})
	}
}

func TestComposeAddedInAnEmptyDocumentDoesNotPrependABlankLine(t *testing.T) {
	profile := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)
	deltaText := join("## ADDED Requirements", "", "### Requirement: Gamma", "", "Cuerpo gamma.")
	got, err := composeAdded(t, profile, byteVariants[0], "", deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if want := "### Gamma\n\nCuerpo gamma.\n"; string(got.Output) != want {
		t.Errorf("salida = %q, se esperaba %q", got.Output, want)
	}
}
