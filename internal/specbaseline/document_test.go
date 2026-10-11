package specbaseline

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// fixturesLF son los documentos de testdata/reference que se recorren en las
// pruebas de ida y vuelta.
var fixturesLF = []string{
	"01_Requisitos_Funcionales.md",
	"01_Requisitos_Funcionales_sin_salto_final.md",
	"02_Requisitos_No_Funcionales.md",
	"03_Modelo_Datos.md",
	"05_Interfaces_Usuario.md",
	"REGISTRO_INCREMENTOS.md",
	"context/ARQUITECTURA.md",
	"context/INTEGRACIONES.md",
}

func TestDocumentRoundTripKeepsEveryByte(t *testing.T) {
	for _, name := range fixturesLF {
		t.Run(name, func(t *testing.T) {
			lf := readFixture(t, name)
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				data := v.apply(lf)
				doc := ParseDocument(data)
				if got := doc.Bytes(); !bytes.Equal(got, data) {
					t.Fatalf("la serialización no reproduce el documento (%d bytes frente a %d)", len(got), len(data))
				}
				if want := v.bom || strings.HasPrefix(lf, string(utf8BOM)); doc.BOM != want {
					t.Errorf("BOM = %v, se esperaba %v", doc.BOM, want)
				}
			})
		})
	}
}

func TestDocumentLinesKeepTheirOwnEOL(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []Line
		bom  bool
	}{
		{name: "vacío", data: "", want: nil},
		{name: "una línea sin salto", data: "uno", want: []Line{{Text: "uno"}}},
		{name: "una línea con LF", data: "uno\n", want: []Line{{Text: "uno", EOL: "\n"}}},
		{name: "una línea con CRLF", data: "uno\r\n", want: []Line{{Text: "uno", EOL: "\r\n"}}},
		{name: "final mixto", data: "a\r\nb\nc", want: []Line{{Text: "a", EOL: "\r\n"}, {Text: "b", EOL: "\n"}, {Text: "c"}}},
		{name: "línea en blanco final", data: "a\n\n", want: []Line{{Text: "a", EOL: "\n"}, {Text: "", EOL: "\n"}}},
		{name: "un CR suelto es texto", data: "a\rb\n", want: []Line{{Text: "a\rb", EOL: "\n"}}},
		{name: "BOM retirado", data: "\xef\xbb\xbfuno\n", want: []Line{{Text: "uno", EOL: "\n"}}, bom: true},
		{name: "solo el BOM", data: "\xef\xbb\xbf", want: nil, bom: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := ParseDocument([]byte(tt.data))
			if !reflect.DeepEqual(doc.Lines, tt.want) {
				t.Errorf("líneas = %#v, se esperaba %#v", doc.Lines, tt.want)
			}
			if doc.BOM != tt.bom {
				t.Errorf("BOM = %v, se esperaba %v", doc.BOM, tt.bom)
			}
			if got := string(doc.Bytes()); got != tt.data {
				t.Errorf("Bytes() = %q, se esperaba %q", got, tt.data)
			}
		})
	}
}

func TestDocumentEOLFallback(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "sin líneas", data: "", want: "\n"},
		{name: "ninguna línea con final", data: "solo", want: "\n"},
		{name: "primer final CRLF", data: "a\r\nb\n", want: "\r\n"},
		{name: "primer final LF", data: "a\nb\r\n", want: "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDocument([]byte(tt.data)).EOL(); got != tt.want {
				t.Errorf("EOL() = %q, se esperaba %q", got, tt.want)
			}
		})
	}
}

// headingSummary resume los encabezados de un documento como «nivel título».
func headingSummary(doc Document) []string {
	var out []string
	for _, h := range doc.Headings() {
		out = append(out, strings.Repeat("#", h.Level)+" "+h.Title)
	}
	return out
}

func TestDocumentATXHeadings(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "niveles de 1 a 6", text: "# a\n## b\n### c\n#### d\n##### e\n###### f\n", want: []string{"# a", "## b", "### c", "#### d", "##### e", "###### f"}},
		{name: "siete almohadillas no son encabezado", text: "####### a\n", want: nil},
		{name: "sin espacio no es encabezado", text: "#a\n###b\n", want: nil},
		{name: "de 0 a 3 espacios iniciales", text: "# a\n # b\n  # c\n   # d\n", want: []string{"# a", "# b", "# c", "# d"}},
		{name: "cuatro espacios no es encabezado", text: "    # a\n", want: nil},
		{name: "almohadillas de cierre", text: "## a ##\n## b #\n### c ####   \n", want: []string{"## a", "## b", "### c"}},
		{name: "almohadilla pegada al texto no es cierre", text: "## a#\n## C#\n", want: []string{"## a#", "## C#"}},
		{name: "título vacío", text: "###\n### \n### ###\n", want: []string{"### ", "### ", "### "}},
		{name: "setext no se reconoce", text: "Título\n======\n\nOtro\n------\n", want: nil},
		{name: "valla de tres acentos graves", text: "# a\n```\n# falso\n```\n# b\n", want: []string{"# a", "# b"}},
		{name: "valla de tildes", text: "# a\n~~~md\n# falso\n~~~\n# b\n", want: []string{"# a", "# b"}},
		{name: "el cierre exige el mismo carácter", text: "# a\n```\n~~~\n# falso\n```\n# b\n", want: []string{"# a", "# b"}},
		{name: "el cierre exige una serie igual de larga", text: "# a\n````\n```\n# falso\n````\n# b\n", want: []string{"# a", "# b"}},
		{name: "una línea con texto no cierra la valla", text: "# a\n```\n``` texto\n# falso\n```\n# b\n", want: []string{"# a", "# b"}},
		{name: "valla sin cerrar llega al final", text: "# a\n```\n# falso\n# otro\n", want: []string{"# a"}},
		{name: "la valla debe empezar en la primera columna", text: "# a\n  ```\n# real\n", want: []string{"# a", "# real"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forEachVariant(t, func(t *testing.T, v byteVariant) {
				got := headingSummary(ParseDocument(v.apply(tt.text)))
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("encabezados = %q, se esperaba %q", got, tt.want)
				}
			})
		})
	}
}

func TestDocumentHeadingIndexesAreZeroBased(t *testing.T) {
	doc := ParseDocument([]byte("texto\n\n## Dos\n"))
	headings := doc.Headings()
	if len(headings) != 1 || headings[0].Index+1 != 3 || headings[0].Index != 2 {
		t.Fatalf("encabezados = %#v, se esperaba uno en el índice 2", headings)
	}
}

func TestDocumentFixturesKeepTheShapesOfTheReference(t *testing.T) {
	t.Run("los documentos de prosa no tienen requisitos con el perfil de RF", func(t *testing.T) {
		profile := requirementProfile(t)
		for _, name := range []string{"05_Interfaces_Usuario.md", "REGISTRO_INCREMENTOS.md", "context/ARQUITECTURA.md"} {
			doc := ParseDocument([]byte(readFixture(t, name)))
			if got := doc.Requirements(profile); len(got) != 0 {
				t.Errorf("%s: %d requisitos reconocidos, se esperaban 0", name, len(got))
			}
		}
	})
	t.Run("un documento de prosa con historias reconoce sus secciones por título", func(t *testing.T) {
		doc := ParseDocument([]byte(readFixture(t, "03_Modelo_Datos.md")))
		got := headingSummary(doc)
		want := []string{"# Modelo de datos de la tienda demo", "## Entidades", "### US-014: Cliente", "### BUG-007: Pedido sin cliente", "## Relaciones"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("encabezados = %q, se esperaba %q", got, want)
		}
	})
	t.Run("los títulos repetidos de la interfaz y del contexto se conservan", func(t *testing.T) {
		for name, repeated := range map[string]string{
			"05_Interfaces_Usuario.md": "### Listado",
			"context/ARQUITECTURA.md":  "### Servicio de catálogo",
		} {
			doc := ParseDocument([]byte(readFixture(t, name)))
			count := 0
			for _, heading := range headingSummary(doc) {
				if heading == repeated {
					count++
				}
			}
			if count != 2 {
				t.Errorf("%s: %q aparece %d veces, se esperaban 2", name, repeated, count)
			}
		}
	})
	t.Run("el fichero de contexto con BOM lo conserva", func(t *testing.T) {
		data := []byte(readFixture(t, "context/INTEGRACIONES.md"))
		doc := ParseDocument(data)
		if !doc.BOM {
			t.Fatal("el fixture debe empezar con BOM")
		}
		if got := headingSummary(doc); len(got) == 0 || got[0] != "# Integraciones" {
			t.Errorf("el BOM no debe formar parte del primer encabezado: %q", got)
		}
	})
	t.Run("la variante sin salto final termina sin final de línea", func(t *testing.T) {
		doc := ParseDocument([]byte(readFixture(t, "01_Requisitos_Funcionales_sin_salto_final.md")))
		if last := doc.Lines[len(doc.Lines)-1]; last.EOL != "" {
			t.Errorf("la última línea tiene final %q, se esperaba ninguno", last.EOL)
		}
	})
}
