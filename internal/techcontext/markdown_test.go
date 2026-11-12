package techcontext

import (
	"reflect"
	"strings"
	"testing"
)

func TestMarkdownDocumentLinesKeepTheirOwnEOL(t *testing.T) {
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

func TestMarkdownDocumentATXHeadings(t *testing.T) {
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

func TestMarkdownDocumentHeadingIndexesAreZeroBased(t *testing.T) {
	doc := ParseDocument([]byte("texto\n\n## Dos\n"))
	headings := doc.Headings()
	if len(headings) != 1 || headings[0].Index+1 != 3 || headings[0].Index != 2 {
		t.Fatalf("encabezados = %#v, se esperaba uno en el índice 2", headings)
	}
}
