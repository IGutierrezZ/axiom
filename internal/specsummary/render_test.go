package specsummary

import (
	"regexp"
	"strings"
	"testing"
)

func sampleModel() Model {
	return Model{Capabilities: []Capability{
		{Name: "zeta", Requirements: []Requirement{
			{Title: "Uno (prov:ZZ:1)", Paragraph: "El sistema DEBE hacer `x`.", Operation: OpAdded},
		}},
		{Name: "alfa", Requirements: []Requirement{
			{Title: "Cambiado (prov:AA:2)", Paragraph: "Texto **fuerte** y `código`.", Operation: OpModified},
			{Title: "Viejo (prov:AA:3)", Paragraph: "Ya no aplica.", Operation: OpRemoved},
			{Title: "Nuevo nombre (prov:AA:4)", Paragraph: "Ahora se llama así.", Operation: OpRenamed, RenamedFrom: "Nombre antiguo (prov:AA:4)"},
			{Title: "Sin texto (prov:AA:5)", Operation: OpAdded},
		}},
	}}
}

func TestRenderStructureAndOrder(t *testing.T) {
	want := strings.Join([]string{
		"<h2>alfa</h2>",
		"<h3>Cambiado (prov:AA:2)</h3>",
		"<p>Texto <strong>fuerte</strong> y <code>código</code>.</p>",
		"<h3>Viejo (prov:AA:3)</h3>",
		"<p><strong>Retirado.</strong> Ya no aplica.</p>",
		"<h3>Nuevo nombre (prov:AA:4)</h3>",
		"<p><strong>Renombrado desde:</strong> Nombre antiguo (prov:AA:4) Ahora se llama así.</p>",
		"<h3>Sin texto (prov:AA:5)</h3>",
		"<h2>zeta</h2>",
		"<h3>Uno (prov:ZZ:1)</h3>",
		"<p>El sistema DEBE hacer <code>x</code>.</p>",
		"",
	}, "\n")
	if got := Render(sampleModel()); got != want {
		t.Fatalf("Render =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderKeepsFileOrderWithMixedOperations(t *testing.T) {
	m := Model{Capabilities: []Capability{{Name: "c", Requirements: []Requirement{
		{Title: "B", Operation: OpModified},
		{Title: "R", Operation: OpRemoved},
		{Title: "A", Operation: OpAdded},
	}}}}
	got := Render(m)
	if i, j, k := strings.Index(got, "<h3>B"), strings.Index(got, "<h3>R"), strings.Index(got, "<h3>A"); !(i < j && j < k) {
		t.Fatalf("el orden del fichero no se respeta:\n%s", got)
	}
}

func TestRenderRemovedAndRenamedHaveH3AndStrongMarker(t *testing.T) {
	m := Model{Capabilities: []Capability{{Name: "c", Requirements: []Requirement{
		{Title: "R", Operation: OpRemoved},
		{Title: "N", Operation: OpRenamed, RenamedFrom: "V"},
	}}}}
	want := "<h2>c</h2>\n<h3>R</h3>\n<p><strong>Retirado.</strong></p>\n<h3>N</h3>\n<p><strong>Renombrado desde:</strong> V</p>\n"
	if got := Render(m); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestRenderEmitsNoListsAndOnlyWhitelistedTagsWithoutAttributes(t *testing.T) {
	got := Render(sampleModel())
	tags := regexp.MustCompile(`</?([a-zA-Z0-9]+)([^>]*)>`).FindAllStringSubmatch(got, -1)
	allowed := map[string]bool{"h2": true, "h3": true, "p": true, "code": true, "strong": true}
	for _, m := range tags {
		if !allowed[m[1]] {
			t.Errorf("etiqueta fuera de la lista blanca: %s", m[0])
		}
		if m[2] != "" {
			t.Errorf("etiqueta con atributos: %s", m[0])
		}
	}
	if strings.Contains(got, "<ul") || strings.Contains(got, "<li") {
		t.Fatalf("no deben emitirse ul ni li:\n%s", got)
	}
}

func TestRenderEscapesEverything(t *testing.T) {
	m := Model{Capabilities: []Capability{{Name: "<c&>", Requirements: []Requirement{
		{Title: `<script>"x"</script>`, Paragraph: "a <b> & `<i>` **<u>** 'q'", Operation: OpRenamed, RenamedFrom: "<old>"},
	}}}}
	got := Render(m)
	for _, bad := range []string{"<script", "<b>", "<i>", "<u>", "<old>", "<c&>"} {
		if strings.Contains(got, bad) {
			t.Errorf("sin escapar %q en:\n%s", bad, got)
		}
	}
	if !strings.Contains(got, "<code>&lt;i&gt;</code>") || !strings.Contains(got, "<strong>&lt;u&gt;</strong>") {
		t.Errorf("el marcado en línea no escapa su contenido:\n%s", got)
	}
}

func TestRenderIsByteDeterministic(t *testing.T) {
	first := Render(sampleModel())
	for i := 0; i < 20; i++ {
		if got := Render(sampleModel()); got != first {
			t.Fatalf("la ejecución %d difiere", i)
		}
	}
}

func TestRenderUnclosedMarkupIsLiteralAndEscaped(t *testing.T) {
	cases := map[string]string{
		"a **b":        "a **b",
		"a `b":         "a `b",
		"a ** b":       "a ** b",
		"****":         "****",
		"``":           "``",
		"x < **y**":    "x &lt; <strong>y</strong>",
		"`a` y `b":     "<code>a</code> y `b",
		"**a** **b":    "<strong>a</strong> **b",
		"`**` literal": "<code>**</code> literal",
	}
	for in, want := range cases {
		if got := inline(in); got != want {
			t.Errorf("inline(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstParagraph(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{"simple", "### Requirement: X\nLinea uno\nlinea dos\n\nOtro bloque\n", "Linea uno linea dos"},
		{"blancas previas", "### Requirement: X\n\n\nTexto\n", "Texto"},
		{"primer bloque lista", "### Requirement: X\n- a\n- b\n\nTexto\n", ""},
		{"primer bloque escenario", "### Requirement: X\n#### Scenario: s\ntexto\n", ""},
		{"tabla", "### Requirement: X\n| a | b |\n", ""},
		{"cita", "### Requirement: X\n> nota\n", ""},
		{"valla", "### Requirement: X\n```go\ncodigo\n```\n", ""},
		{"valla tilde", "### Requirement: X\n~~~\n", ""},
		{"parentesis", "### Requirement: X\n(nota)\n", ""},
		{"corta en cabecera", "### Requirement: X\ntexto\n#### Scenario: s\nmas\n", "texto"},
		{"corta en lista", "### Requirement: X\ntexto\n- item\n", "texto"},
		{"sin cuerpo", "### Requirement: X", ""},
		{"crlf", "### Requirement: X\r\nuno\r\ndos\r\n", "uno dos"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FirstParagraph(tt.body); got != tt.want {
				t.Fatalf("FirstParagraph = %q, want %q", got, tt.want)
			}
		})
	}
}

func bigModel(n int) Model {
	reqs := make([]Requirement, n)
	for i := range reqs {
		reqs[i] = Requirement{
			Title:     "Requisito número " + strings.Repeat("x", 40) + " " + itoa(i),
			Paragraph: strings.Repeat("palabra ", 40),
			Operation: OpAdded,
		}
	}
	return Model{Capabilities: []Capability{{Name: "grande", Requirements: reqs}}}
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{digits[i%10]}, b...)
	}
	return string(b)
}

func TestRenderDropsParagraphsBeforeCuttingRequirements(t *testing.T) {
	// Con párrafos supera el tope; sin ellos cabe: no se corta ningún requisito.
	m := bigModel(400)
	if len(write(m.Capabilities, true, -1)) <= MaxBytes {
		t.Fatal("el modelo de prueba debe superar el tope con párrafos")
	}
	got := Render(m)
	if len(got) > MaxBytes {
		t.Fatalf("len = %d supera el tope", len(got))
	}
	if strings.Contains(got, "<p>") {
		t.Fatalf("debe haberse descartado los p:\n%.200s", got)
	}
	if strings.Count(got, "<h3>") != 400 {
		t.Fatalf("no debe cortar requisitos: %d", strings.Count(got, "<h3>"))
	}
}

func TestRenderCutsRequirementsWithTrailer(t *testing.T) {
	m := bigModel(2000)
	got := Render(m)
	if len(got) > MaxBytes {
		t.Fatalf("len = %d supera el tope", len(got))
	}
	kept := strings.Count(got, "<h3>")
	if kept == 0 || kept >= 2000 {
		t.Fatalf("recorte inesperado: %d requisitos", kept)
	}
	want := "<p>… y " + itoa(2000-kept) + " requisitos más</p>\n"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("cierre inesperado, se esperaba %q al final", want)
	}
	if Render(m) != got {
		t.Fatal("el recorte no es determinista")
	}
}

func TestRenderEmptyModel(t *testing.T) {
	if got := Render(Model{}); got != "" {
		t.Fatalf("Render(vacío) = %q", got)
	}
}

func FuzzRenderEscapes(f *testing.F) {
	for _, s := range []string{"", "<script>", "a & b", "**x", "`y", "`<i>`", "\"'", "**`**`", "\x00\xff"} {
		f.Add(s)
	}
	tag := regexp.MustCompile(`</?([a-z0-9]+)>`)
	f.Fuzz(func(t *testing.T, s string) {
		m := Model{Capabilities: []Capability{{Name: s, Requirements: []Requirement{
			{Title: s, Paragraph: s, Operation: OpRenamed, RenamedFrom: s},
		}}}}
		got := Render(m)
		allowed := map[string]bool{"h2": true, "h3": true, "p": true, "code": true, "strong": true}
		for _, m := range tag.FindAllStringSubmatch(got, -1) {
			if !allowed[m[1]] {
				t.Fatalf("etiqueta no permitida %q para %q", m[0], s)
			}
		}
		// Ningún `<` ni `"` crudos fuera de las etiquetas de la lista blanca.
		stripped := tag.ReplaceAllString(got, "")
		rest := regexp.MustCompile(`&(amp|lt|gt|#34|#39);`).ReplaceAllString(stripped, "")
		if strings.ContainsAny(rest, "<>\"&'") {
			t.Fatalf("texto sin escapar para %q: %q", s, got)
		}
		if Render(m) != got {
			t.Fatal("no determinista")
		}
	})
}
