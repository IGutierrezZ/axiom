package specbaseline

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// baseDocument es el documento de las pruebas de composición: dos secciones h2,
// un requisito con subsección h4 seguido de una sección contigua que no es
// requisito, y requisitos de dos secciones.
var baseDocument = join(
	"# Documento de prueba",
	"",
	"## Catálogo",
	"",
	"### RF-100: Alta",
	"",
	"Cuerpo original de alta.",
	"",
	"#### Escenario viejo",
	"",
	"Texto viejo.",
	"",
	"### Notas de implantación",
	"",
	"Sección contigua.",
	"",
	"### RF-101: Baja",
	"",
	"Cuerpo de baja.",
	"",
	"## Pagos",
	"",
	"### RF-200: Cobro",
	"",
	"Cuerpo de cobro.",
)

// composeInput arma la entrada de composición de un documento de prueba.
func composeInput(profile Profile, original, deltaData []byte) ComposeInput {
	return ComposeInput{Document: PlannedDocument{
		Domain:   "requisitos-funcionales",
		Path:     "/espacio/docs/01_Requisitos_Funcionales.md",
		RelPath:  "docs/01_Requisitos_Funcionales.md",
		Profile:  profile,
		Original: original,
		Delta:    deltaData,
	}}
}

// composeVariant compone un delta en LF sobre el documento en la variante de bytes.
func composeVariant(t *testing.T, profile Profile, v byteVariant, original, deltaText string) (ComposeResult, error) {
	t.Helper()
	return ComposeDocument(composeInput(profile, v.apply(original), []byte(deltaText)))
}

// assertOutput comprueba que la composición produjo el texto esperado, escrito en
// LF, en los bytes de la variante.
func assertOutput(t *testing.T, v byteVariant, got ComposeResult, want string) {
	t.Helper()
	if expected := v.apply(want); !bytes.Equal(got.Output, expected) {
		t.Errorf("salida distinta de la esperada\n--- obtenida ---\n%q\n--- esperada ---\n%q", got.Output, expected)
	}
}

func assertComposeError(t *testing.T, err error, wantParts ...string) *ComposeError {
	t.Helper()
	var composeErr *ComposeError
	if !errors.As(err, &composeErr) {
		t.Fatalf("error = %v (%T), se esperaba *ComposeError", err, err)
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("el error no menciona %q: %v", part, err)
		}
	}
	return composeErr
}

func TestComposeModifiedReplacesOnlyTheBlockAndDropsTheNotes(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-100: Alta de cliente con validación",
		"",
		"Nuevo cuerpo de alta.",
		"",
		"#### Escenario nuevo",
		"",
		"Texto nuevo.",
		"",
		"(Placement: after RF-001)",
		"(Previously: cuerpo anterior)",
		"(Migration: sin migración)",
	)
	want := join(
		"# Documento de prueba",
		"",
		"## Catálogo",
		"",
		"### RF-100: Alta de cliente con validación",
		"",
		"Nuevo cuerpo de alta.",
		"",
		"#### Escenario nuevo",
		"",
		"Texto nuevo.",
		"",
		"### Notas de implantación",
		"",
		"Sección contigua.",
		"",
		"### RF-101: Baja",
		"",
		"Cuerpo de baja.",
		"",
		"## Pagos",
		"",
		"### RF-200: Cobro",
		"",
		"Cuerpo de cobro.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, baseDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		for _, note := range []string{"(Placement", "(Previously", "(Migration"} {
			if bytes.Contains(got.Output, []byte(note)) {
				t.Errorf("la nota %q no debe integrarse", note)
			}
		}
		if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "### Notas de implantación") || !strings.Contains(got.Warnings[0], "línea 13") {
			t.Errorf("avisos = %q, se esperaba uno que nombre la sección contigua y su línea", got.Warnings)
		}
	})
}

func TestComposeModifiedBodyArrivesWholeWithTheTemplateHeading(t *testing.T) {
	profile := profileFor(t, `^## (?P<id>RF-\d+) - (?P<title>.+)$`, 2, workspace.IdentityID, "## {id} - {title}", workspace.ScenariosInclude)
	original := join(
		"# Documento",
		"",
		"## RF-120 - Alta de cliente",
		"",
		"Cuerpo.",
		"",
		"## RF-121 - Baja de cliente",
		"",
		"Otro cuerpo.",
	)
	deltaText := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-120 - Alta de cliente con validación",
		"",
		"Primer párrafo.",
		"",
		"Segundo párrafo con `código`.",
		"",
		"Tercer párrafo.",
	)
	want := join(
		"# Documento",
		"",
		"## RF-120 - Alta de cliente con validación",
		"",
		"Primer párrafo.",
		"",
		"Segundo párrafo con `código`.",
		"",
		"Tercer párrafo.",
		"",
		"## RF-121 - Baja de cliente",
		"",
		"Otro cuerpo.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, original, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
	})
}

func TestComposeModifiedKeepsFencedHeadingsInsideTheBody(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-101: Baja",
		"",
		"Ejemplo:",
		"",
		"```md",
		"## Ejemplo",
		"### Notas",
		"```",
	)
	want := join(
		"# Documento de prueba",
		"",
		"## Catálogo",
		"",
		"### RF-100: Alta",
		"",
		"Cuerpo original de alta.",
		"",
		"#### Escenario viejo",
		"",
		"Texto viejo.",
		"",
		"### Notas de implantación",
		"",
		"Sección contigua.",
		"",
		"### RF-101: Baja",
		"",
		"Ejemplo:",
		"",
		"```md",
		"## Ejemplo",
		"### Notas",
		"```",
		"",
		"## Pagos",
		"",
		"### RF-200: Cobro",
		"",
		"Cuerpo de cobro.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, baseDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		if len(got.Warnings) != 0 {
			t.Errorf("RF-101 no tiene cola contigua; avisos = %q", got.Warnings)
		}
	})
}

func TestComposeRejectsBodyHeadingsThatWouldCloseTheBlock(t *testing.T) {
	profile := requirementProfile(t)
	tests := []struct {
		name string
		body []string
		want []string
	}{
		{name: "encabezado del mismo nivel", body: []string{"### Notas de implantación"}, want: []string{"### Notas de implantación", "línea 5 del delta", "nivel 3", "rebaja el encabezado"}},
		{name: "encabezado de nivel menor", body: []string{"## Ejemplo"}, want: []string{"## Ejemplo", "línea 5 del delta"}},
		{name: "un encabezado de nivel mayor no es infracción", body: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := []string{"## MODIFIED Requirements", "", "### Requirement: RF-101: Baja", ""}
			if tt.body == nil {
				lines = append(lines, "#### Escenario: válido", "", "Texto.")
			} else {
				lines = append(lines, tt.body...)
			}
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				got, err := composeVariant(t, profile, v, baseDocument, join(lines...))
				if tt.want == nil {
					if err != nil {
						t.Fatalf("ComposeDocument() error = %v", err)
					}
					return
				}
				composeErr := assertComposeError(t, err, append([]string{"requisitos-funcionales", "MODIFIED", "RF-101: Baja"}, tt.want...)...)
				if composeErr.Continuation == "" {
					t.Error("el error debe indicar la continuación")
				}
				if got.Output != nil {
					t.Error("un rechazo no debe devolver salida")
				}
			})
		})
	}
}

func TestComposeRejectsOperationsOnDuplicatedIDsButKeepsTheRestOperable(t *testing.T) {
	profile := requirementProfile(t)
	original := join(
		"# Documento",
		"",
		"## Cobros",
		"",
		"### RF-020: Cobro con tarjeta",
		"",
		"Cuerpo A.",
		"",
		"### RF-300: Único",
		"",
		"Cuerpo único.",
		"",
		"## Devoluciones",
		"",
		"### RF-020: Devolución parcial",
		"",
		"Cuerpo B.",
	)

	t.Run("MODIFIED sobre un ID duplicado nombra todas las apariciones", func(t *testing.T) {
		deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-020: Cobro", "", "Cuerpo nuevo.")
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			got, err := composeVariant(t, profile, v, original, deltaText)
			composeErr := assertComposeError(t, err, "RF-020", "línea 5", "Cobro con tarjeta", "línea 15", "Devolución parcial", "normaliza")
			if composeErr.Section != "MODIFIED" || got.Output != nil {
				t.Errorf("sección = %q, salida = %q", composeErr.Section, got.Output)
			}
		})
	})

	t.Run("REMOVED sobre un ID duplicado se rechaza", func(t *testing.T) {
		deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-020: Cobro", "", "(Reason: obsoleto)")
		_, err := composeVariant(t, profile, byteVariants[0], original, deltaText)
		assertComposeError(t, err, "REMOVED", "RF-020", "línea 5", "línea 15")
	})

	t.Run("un ID único sigue siendo operable", func(t *testing.T) {
		deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-300: Único actualizado", "", "Cuerpo único nuevo.")
		want := strings.Replace(strings.Replace(original, "### RF-300: Único", "### RF-300: Único actualizado", 1), "Cuerpo único.", "Cuerpo único nuevo.", 1)
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			got, err := composeVariant(t, profile, v, original, deltaText)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			assertOutput(t, v, got, want)
		})
	})
}

func TestComposeRenamedWithIDIdentityIsRejected(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## RENAMED Requirements", "", "### Requirement: RF-100: Alta → RF-100: Alta de cliente", "", "(Reason: nombre más claro)")
	got, err := composeVariant(t, profile, byteVariants[0], baseDocument, deltaText)
	assertComposeError(t, err, "RENAMED", "MODIFIED", "renumerar")
	if got.Output != nil {
		t.Error("un rechazo no debe devolver salida")
	}
}

func TestComposeRenamedWithTextIdentityRetitlesTheBlock(t *testing.T) {
	profile := profileFor(t, `^### (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityText, "### {id}: {title}", workspace.ScenariosInclude)
	renamed := join(
		"## RENAMED Requirements",
		"",
		"### Requirement: RF-100: Alta → RF-100: Alta de cliente",
		"",
		"(Reason: nombre más claro)",
	)
	want := strings.Replace(baseDocument, "### RF-100: Alta\n", "### RF-100: Alta de cliente\n", 1)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, baseDocument, renamed)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
	})

	t.Run("el nombre de destino ya existe", func(t *testing.T) {
		clash := join("## RENAMED Requirements", "", "### Requirement: RF-100: Alta → RF-101: Baja", "", "(Reason: choque)")
		_, err := composeVariant(t, profile, byteVariants[0], baseDocument, clash)
		assertComposeError(t, err, "RENAMED", "ya existe")
	})
	t.Run("sin nota de motivo", func(t *testing.T) {
		missing := join("## RENAMED Requirements", "", "### Requirement: RF-100: Alta → RF-100: Alta de cliente")
		_, err := composeVariant(t, profile, byteVariants[0], baseDocument, missing)
		assertComposeError(t, err, "RENAMED", "(Reason")
	})
	t.Run("sin flecha", func(t *testing.T) {
		_, err := composeVariant(t, profile, byteVariants[0], baseDocument, join("## RENAMED Requirements", "", "### Requirement: RF-100: Alta", "", "(Reason: x)"))
		assertComposeError(t, err, "RENAMED", "Nombre antiguo")
	})
}

func TestComposeModifiedOnAMissingIDNamesSectionAndRequirement(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## MODIFIED Requirements", "", "### Requirement: RF-999: No existe", "", "Cuerpo.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, baseDocument, deltaText)
		composeErr := assertComposeError(t, err, "MODIFIED", "RF-999", "requisitos-funcionales")
		if composeErr.Section != "MODIFIED" || composeErr.Requirement != "RF-999: No existe" {
			t.Errorf("error = %+v", composeErr)
		}
		if got.Output != nil || got.Entries != nil {
			t.Error("un rechazo no debe devolver salida ni traza")
		}
	})
}

func TestComposeRejectsANameThatDoesNotMatchTheHeadingPattern(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: Reintento de pago", "", "Cuerpo.")
	got, err := composeVariant(t, profile, byteVariants[0], baseDocument, deltaText)
	composeErr := assertComposeError(t, err, "Reintento de pago", "heading", "### Reintento de pago")
	if composeErr.Section != "MODIFIED" || got.Output != nil {
		t.Errorf("error = %+v, salida = %q", composeErr, got.Output)
	}
}

func TestComposeRemovedDeletesOnlyTheBlockAndKeepsItsTail(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-100: Alta", "", "(Reason: sustituido por el nuevo alta)")
	want := join(
		"# Documento de prueba",
		"",
		"## Catálogo",
		"",
		"### Notas de implantación",
		"",
		"Sección contigua.",
		"",
		"### RF-101: Baja",
		"",
		"Cuerpo de baja.",
		"",
		"## Pagos",
		"",
		"### RF-200: Cobro",
		"",
		"Cuerpo de cobro.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, baseDocument, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "### Notas de implantación") || !strings.Contains(got.Warnings[0], "REMOVED") {
			t.Errorf("avisos = %q, se esperaba uno que nombre la sección contigua", got.Warnings)
		}
		if bytes.Contains(got.Output, []byte("(Reason")) {
			t.Error("la nota de motivo no debe integrarse")
		}
	})
}

func TestComposeRemovedRequiresTheReasonNote(t *testing.T) {
	profile := requirementProfile(t)
	_, err := composeVariant(t, profile, byteVariants[0], baseDocument, join("## REMOVED Requirements", "", "### Requirement: RF-100: Alta"))
	assertComposeError(t, err, "REMOVED", "RF-100", "(Reason")
}

func TestComposeRemovedWithSupersedeKeepsTheBlockMarked(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-101: Baja", "", "(Reason: sustituido por el nuevo flujo)")
	want := join(
		"# Documento de prueba",
		"",
		"## Catálogo",
		"",
		"### RF-100: Alta",
		"",
		"Cuerpo original de alta.",
		"",
		"#### Escenario viejo",
		"",
		"Texto viejo.",
		"",
		"### Notas de implantación",
		"",
		"Sección contigua.",
		"",
		"### RF-101: Baja [SUPERSEDED / DEPRECADO]",
		"> [!CAUTION]",
		"> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.",
		"> Motivo: sustituido por el nuevo flujo",
		"",
		"",
		"Cuerpo de baja.",
		"",
		"## Pagos",
		"",
		"### RF-200: Cobro",
		"",
		"Cuerpo de cobro.",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		input := composeInput(profile, v.apply(baseDocument), []byte(deltaText))
		input.Supersede = true
		got, err := ComposeDocument(input)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		if len(got.Entries) != 1 || got.Entries[0].Action != ActionRemoved || got.Entries[0].Reason != "sustituido por el nuevo flujo" {
			t.Errorf("entradas = %+v", got.Entries)
		}
	})
}

func TestComposeExplicitSupersededAcceptsIDsOrTitles(t *testing.T) {
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-200: Cobro", "", "Cuerpo de cobro actualizado.")

	t.Run("identidad id", func(t *testing.T) {
		profile := requirementProfile(t)
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			input := composeInput(profile, v.apply(baseDocument), []byte(deltaText))
			input.ExplicitSuperseded = []string{"RF-101", " RF-100 ", "RF-777"}
			got, err := ComposeDocument(input)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			text := string(bytes.TrimPrefix(got.Output, utf8BOM))
			text = strings.ReplaceAll(text, "\r\n", "\n")
			for _, heading := range []string{"### RF-101: Baja [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n", "### RF-100: Alta [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n"} {
				if !strings.Contains(text, heading) {
					t.Errorf("falta el encabezado marcado %q en:\n%s", heading, text)
				}
			}
			if !strings.Contains(text, "> Motivo: Superado por consolidación de arquitectura") {
				t.Error("falta el motivo del motor nativo")
			}
			if len(got.Warnings) < 1 || !strings.Contains(strings.Join(got.Warnings, "\n"), "RF-777") {
				t.Errorf("un identificador inexistente debe avisar: %q", got.Warnings)
			}
		})
	})

	t.Run("identidad text", func(t *testing.T) {
		profile := profileFor(t, `^### (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityText, "### {id}: {title}", workspace.ScenariosInclude)
		textDelta := join("## MODIFIED Requirements", "", "### Requirement: RF-200: Cobro", "", "Cuerpo de cobro actualizado.")
		input := composeInput(profile, []byte(baseDocument), []byte(textDelta))
		input.ExplicitSuperseded = []string{"Baja"}
		got, err := ComposeDocument(input)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if !strings.Contains(string(got.Output), "### RF-101: Baja [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n") {
			t.Errorf("el título debe identificar el requisito:\n%s", got.Output)
		}
	})

	t.Run("un requisito ya marcado por REMOVED con supersede no se vuelve a marcar", func(t *testing.T) {
		profile := requirementProfile(t)
		removed := join("## REMOVED Requirements", "", "### Requirement: RF-101: Baja", "", "(Reason: obsoleto)")
		input := composeInput(profile, []byte(baseDocument), []byte(removed))
		input.Supersede = true
		input.ExplicitSuperseded = []string{"RF-101"}
		got, err := ComposeDocument(input)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if count := strings.Count(string(got.Output), "[SUPERSEDED / DEPRECADO]"); count != 1 {
			t.Errorf("marcas = %d, se esperaba 1", count)
		}
		if len(got.Warnings) != 0 {
			t.Errorf("avisos = %q, no se esperaba ninguno", got.Warnings)
		}
	})

	t.Run("un identificador duplicado se rechaza", func(t *testing.T) {
		profile := requirementProfile(t)
		duplicated := join("## Cobros", "", "### RF-020: A", "", "x", "", "### RF-020: B", "", "y", "", "### RF-200: Cobro", "", "z")
		input := composeInput(profile, []byte(duplicated), []byte(deltaText))
		input.ExplicitSuperseded = []string{"RF-020"}
		_, err := ComposeDocument(input)
		assertComposeError(t, err, "RF-020", "ambiguo")
	})
}

// toProfileHeadings convierte un texto en formato nativo («### Requirement: X»)
// al formato de un documento con perfil de identidad `text` («### X»).
func toProfileHeadings(text string) string {
	return strings.ReplaceAll(text, "### Requirement: ", "### ")
}

func TestComposeTextIdentityMatchesTheNativeEngine(t *testing.T) {
	canonical := join(
		"## Requirements",
		"",
		"### Requirement: Alpha",
		"",
		"Alpha body.",
		"",
		"### Requirement: Beta",
		"",
		"Beta body.",
		"",
		"### Requirement: Gamma",
		"",
		"Gamma body.",
		"",
		"### Requirement: Delta",
		"",
		"Delta body.",
		"",
		"### Requirement: Epsilon",
		"",
		"Epsilon body.",
	)
	deltaText := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: Beta",
		"",
		"Beta body changed.",
		"",
		"#### Scenario: changed",
		"",
		"- **WHEN** it changes",
		"",
		"## REMOVED Requirements",
		"",
		"### Requirement: Gamma",
		"",
		"(Reason: replaced by the new flow)",
		"",
		"## RENAMED Requirements",
		"",
		"### Requirement: Delta → Delta renamed",
		"",
		"(Reason: clearer name)",
	)
	profile := profileFor(t, `^### (?P<id>)(?P<title>.+)$`, 3, workspace.IdentityText, "### {id}{title}", workspace.ScenariosInclude)

	tests := []struct {
		name      string
		supersede bool
		explicit  []string
	}{
		{name: "sin opciones"},
		{name: "con supersede", supersede: true},
		{name: "con requisitos superados explícitos", explicit: []string{"Alpha"}},
		{name: "con supersede y requisitos explícitos", supersede: true, explicit: []string{"Epsilon", "Missing"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			native, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonical, deltaText, sddstatus.ComposeOptions{
				SupersedeRemoved:   tt.supersede,
				ExplicitSuperseded: tt.explicit,
			})
			if err != nil {
				t.Fatalf("motor nativo: %v", err)
			}
			input := composeInput(profile, []byte(toProfileHeadings(canonical)), []byte(deltaText))
			input.Supersede = tt.supersede
			input.ExplicitSuperseded = tt.explicit
			got, err := ComposeDocument(input)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			if want := toProfileHeadings(native); string(got.Output) != want {
				t.Errorf("la composición con identidad text debe coincidir con el motor nativo\n--- con perfil ---\n%s\n--- nativo ---\n%s", got.Output, want)
			}
		})
	}
}

func TestComposeKeepsEveryByteOutsideTheEditedBlocks(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales.md")
	deltaText := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-002: Consulta de producto por código",
		"",
		"El sistema DEBE devolver los datos de un producto por su código o por su alias.",
		"",
		"### Requirement: RF-050: Informe mensual",
		"",
		"El sistema DEBE generar un informe mensual y otro trimestral.",
	)
	want := strings.Replace(reference, "### RF-002: Consulta de producto\n\nEl sistema DEBE devolver los datos de un producto por su código.\n",
		"### RF-002: Consulta de producto por código\n\nEl sistema DEBE devolver los datos de un producto por su código o por su alias.\n", 1)
	want = strings.Replace(want, "El sistema DEBE generar un informe mensual de ventas.\n", "El sistema DEBE generar un informe mensual y otro trimestral.\n", 1)
	if want == reference {
		t.Fatal("el fixture ha cambiado: la sustitución esperada no se aplicó")
	}
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, reference, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
		if v.crlf {
			if lone := bytes.Count(got.Output, []byte("\n")) - bytes.Count(got.Output, []byte("\r\n")); lone != 0 {
				t.Errorf("la salida CRLF contiene %d saltos LF sueltos", lone)
			}
		}
	})
}

func TestComposeEmitsTheEditedBlocksWithTheDocumentEOL(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-101: Baja", "", "Cuerpo nuevo.", "", "Segunda línea.")

	t.Run("documento CRLF y delta LF", func(t *testing.T) {
		original := byteVariants[1].apply(baseDocument)
		got, err := ComposeDocument(composeInput(profile, original, []byte(deltaText)))
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if lone := bytes.Count(got.Output, []byte("\n")) - bytes.Count(got.Output, []byte("\r\n")); lone != 0 {
			t.Errorf("el fichero contiene %d saltos LF sueltos", lone)
		}
		if !bytes.Contains(got.Output, []byte("### RF-101: Baja\r\n\r\nCuerpo nuevo.\r\n\r\nSegunda línea.\r\n")) {
			t.Errorf("el bloque sustituido debe usar CRLF:\n%q", got.Output)
		}
	})

	t.Run("documento LF y delta CRLF", func(t *testing.T) {
		got, err := ComposeDocument(composeInput(profile, []byte(baseDocument), byteVariants[1].apply(deltaText)))
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		if bytes.Contains(got.Output, []byte("\r")) {
			t.Errorf("el documento LF no debe ganar CR:\n%q", got.Output)
		}
	})
}

func TestComposeKeepsTheBOMAndNeverAddsOne(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-101: Baja", "", "Cuerpo nuevo.")

	withBOM, err := ComposeDocument(composeInput(profile, byteVariants[2].apply(baseDocument), []byte(deltaText)))
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if !bytes.HasPrefix(withBOM.Output, utf8BOM) || bytes.HasPrefix(withBOM.Output[len(utf8BOM):], utf8BOM) {
		t.Error("el documento con BOM debe conservar exactamente uno")
	}

	withoutBOM, err := ComposeDocument(composeInput(profile, byteVariants[1].apply(baseDocument), append(append([]byte{}, utf8BOM...), deltaText...)))
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if bytes.HasPrefix(withoutBOM.Output, utf8BOM) || bytes.Contains(withoutBOM.Output, utf8BOM) {
		t.Error("un documento sin BOM no debe ganar BOM, aunque el delta lo tenga")
	}
}

func TestComposeKeepsTheEndOfFileWhenTheLastBlockIsNotEdited(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales_sin_salto_final.md")
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-100: Primer requisito editado", "", "Cuerpo editado.")
	want := strings.Replace(reference, "### RF-100: Primer requisito\n\nCuerpo del primer requisito.\n", "### RF-100: Primer requisito editado\n\nCuerpo editado.\n", 1)
	if want == reference || strings.HasSuffix(want, "\n") {
		t.Fatal("el fixture ha cambiado: debe terminar sin salto de línea y admitir la sustitución")
	}
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		got, err := composeVariant(t, profile, v, reference, deltaText)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		assertOutput(t, v, got, want)
	})
}

func TestComposeEditingTheLastBlockWithoutNewlineKeepsTheFileWithoutIt(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales_sin_salto_final.md")

	t.Run("MODIFIED", func(t *testing.T) {
		deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-101: Último requisito", "", "Último cuerpo editado.")
		want := strings.Replace(reference, "Último cuerpo, sin salto de línea al final del fichero.", "Último cuerpo editado.", 1)
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			got, err := composeVariant(t, profile, v, reference, deltaText)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			assertOutput(t, v, got, want)
		})
	})

	t.Run("REMOVED del último bloque", func(t *testing.T) {
		deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-101: Último requisito", "", "(Reason: obsoleto)")
		want := reference[:strings.Index(reference, "### RF-101")]
		forEachVariant(t, func(t *testing.T, v byteVariant) {
			got, err := composeVariant(t, profile, v, reference, deltaText)
			if err != nil {
				t.Fatalf("ComposeDocument() error = %v", err)
			}
			assertOutput(t, v, got, want)
		})
	})

	t.Run("supersede de un encabezado que es la última línea", func(t *testing.T) {
		original := "### RF-1: A\n\nx\n\n### RF-2: B"
		deltaText := join("## REMOVED Requirements", "", "### Requirement: RF-2: B", "", "(Reason: obsoleto)")
		input := composeInput(profile, []byte(original), []byte(deltaText))
		input.Supersede = true
		got, err := ComposeDocument(input)
		if err != nil {
			t.Fatalf("ComposeDocument() error = %v", err)
		}
		want := "### RF-1: A\n\nx\n\n### RF-2: B [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.\n> Motivo: obsoleto\n"
		if string(got.Output) != want {
			t.Errorf("salida = %q\nse esperaba %q", got.Output, want)
		}
	})
}

func TestComposeIsAllOrNothing(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## MODIFIED Requirements", "",
		"### Requirement: RF-100: Alta nueva", "", "Cuerpo nuevo.", "",
		"### Requirement: RF-999: No existe", "", "Cuerpo.",
	)
	original := []byte(baseDocument)
	got, err := ComposeDocument(composeInput(profile, original, []byte(deltaText)))
	assertComposeError(t, err, "MODIFIED", "RF-999")
	if got.Output != nil || got.Entries != nil || got.Warnings != nil {
		t.Errorf("un rechazo no debe devolver resultado parcial: %+v", got)
	}
	if string(original) != baseDocument {
		t.Error("la composición no debe modificar la entrada")
	}
}

func TestComposeRefusesADeltaWithoutRequirements(t *testing.T) {
	profile := requirementProfile(t)
	for name, deltaText := range map[string]string{
		"vacío":          "",
		"solo secciones": "## MODIFIED Requirements\n\nTexto.\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := composeVariant(t, profile, byteVariants[0], baseDocument, deltaText)
			composeErr := assertComposeError(t, err, "DELTA", "ADDED, MODIFIED, REMOVED ni RENAMED")
			if composeErr.Domain != "requisitos-funcionales" {
				t.Errorf("dominio = %q", composeErr.Domain)
			}
		})
	}
}

func TestComposeTraceHasOneEntryPerDeltaRequirement(t *testing.T) {
	profile := requirementProfile(t)
	deltaText := join(
		"## MODIFIED Requirements", "",
		"### Requirement: RF-100: Alta de cliente", "", "Cuerpo nuevo.", "",
		"## REMOVED Requirements", "",
		"### Requirement: RF-200: Cobro", "", "(Reason: se elimina el cobro)",
	)
	got, err := composeVariant(t, profile, byteVariants[0], baseDocument, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entradas = %d, se esperaban 2: %+v", len(got.Entries), got.Entries)
	}
	modified, removed := got.Entries[0], got.Entries[1]
	if modified.ID != "RF-100" || modified.Action != ActionModified || modified.Kind != EntryCompose || modified.State != StatePending ||
		modified.Domain != "requisitos-funcionales" || modified.Document != "docs/01_Requisitos_Funcionales.md" || modified.Anchor != "### RF-100: Alta de cliente" {
		t.Errorf("entrada MODIFIED = %+v", modified)
	}
	if len(modified.Warnings) != 1 || !strings.Contains(modified.Warnings[0], "### Notas de implantación") {
		t.Errorf("la traza debe registrar el aviso de la cola: %+v", modified.Warnings)
	}
	if removed.ID != "RF-200" || removed.Action != ActionRemoved || removed.Reason != "se elimina el cobro" || removed.Anchor != "### RF-200: Cobro" || len(removed.Warnings) != 0 {
		t.Errorf("entrada REMOVED = %+v", removed)
	}
}

func TestComposeWarnsOncePerTailSection(t *testing.T) {
	profile := requirementProfile(t)
	reference := readFixture(t, "01_Requisitos_Funcionales.md")
	deltaText := join("## MODIFIED Requirements", "", "### Requirement: RF-003: Baja de producto", "", "Cuerpo nuevo.")
	got, err := composeVariant(t, profile, byteVariants[1], reference, deltaText)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}
	if len(got.Warnings) != 2 {
		t.Fatalf("avisos = %q, se esperaban 2", got.Warnings)
	}
	for i, section := range []string{"### Criterios de aceptación generales", "### Glosario del catálogo"} {
		if !strings.Contains(got.Warnings[i], section) {
			t.Errorf("aviso %d = %q, debe nombrar %q", i, got.Warnings[i], section)
		}
	}
	if !strings.Contains(got.Warnings[0], "línea 40 de docs/01_Requisitos_Funcionales.md") || !strings.Contains(got.Warnings[1], "línea 44") {
		t.Errorf("los avisos deben dar la línea y el documento: %q", got.Warnings)
	}
	if !bytes.Contains(got.Output, byteVariants[1].apply("### Criterios de aceptación generales\n\nSección independiente entre dos requisitos.\n\n### Glosario del catálogo\n\nSegunda sección contigua tras el requisito anterior.\n")) {
		t.Error("la cola contigua debe conservar sus bytes")
	}
}

func TestComposeErrorMessageNamesSectionRequirementAndContinuation(t *testing.T) {
	err := &ComposeError{Domain: "dom", Section: "MODIFIED", Requirement: "RF-1", Reason: "no existe", Continuation: "corrígelo"}
	want := `sdd-archive-compose: dominio "dom": sección MODIFIED, requisito "RF-1": no existe. Continuación: corrígelo`
	if err.Error() != want {
		t.Errorf("Error() = %q, se esperaba %q", err.Error(), want)
	}
	minimal := &ComposeError{Reason: "sin delta"}
	if minimal.Error() != "sdd-archive-compose: sin delta" {
		t.Errorf("Error() = %q", minimal.Error())
	}
}
