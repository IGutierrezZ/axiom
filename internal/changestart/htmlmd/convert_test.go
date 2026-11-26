package htmlmd

import (
	"regexp"
	"strings"
	"testing"
)

// rawTag detecta una etiqueta HTML cruda en la salida: un `<` seguido de letra, `/`, `!` o `?`.
var rawTag = regexp.MustCompile(`<[A-Za-z/!?]`)

func convertText(t *testing.T, src string) string {
	t.Helper()
	return Convert(src, nil).Markdown
}

func TestConvertBasic(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"parrafos", "<p>uno</p><p>dos</p>", "uno\n\ndos"},
		{"titulo", "<h2>Hola   mundo</h2>", "## Hola mundo"},
		{"enfasis", "<p><b>a</b> y <i>b</i> y <strong>c</strong> y <em>d</em></p>", "**a** y *b* y **c** y *d*"},
		{"enfasis con espacios", "<p>a<b> x </b>b</p>", "a **x** b"},
		{"salto", "<p>a<br>b</p>", "a  \nb"},
		{"lista", "<ul><li>a</li><li>b</li></ul>", "- a\n- b"},
		{"ordenada", "<ol><li>a</li><li>b</li></ol>", "1. a\n2. b"},
		{"anidada", "<ul><li>a<ul><li>b</li></ul></li><li>c</li></ul>", "- a\n  - b\n- c"},
		{"cita", "<blockquote><p>x</p><p>y</p></blockquote>", "> x\n>\n> y"},
		{"regla", "<p>a</p><hr><p>b</p>", "a\n\n---\n\nb"},
		{"enlace", `<a href="https://example.test/a?b=1">texto</a>`, "[texto](https://example.test/a?b=1)"},
		{"codigo en linea", "<p>usa <code>go test</code></p>", "usa `go test`"},
		{"etiquetas desconocidas", "<span><font>a</font> <u>b</u></span>", "a b"},
		{"mayusculas", "<P>a</P><B>b</B>", "a\n\n**b**"},
		{"li sin cierre", "<ul><li>a<li>b</ul>", "- a\n- b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := convertText(t, tc.src); got != tc.want {
				t.Fatalf("Convert(%q) = %q, quiero %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestConvertImagesAreListedInOrder(t *testing.T) {
	res := Convert(`<p><img src="https://x.test/a.png" alt="uno"> <img src="rel/b.png"></p>`, func(img Image) string {
		return "[" + img.Alt + "|" + img.Src + "]"
	})
	if len(res.Images) != 2 || res.Images[0] != (Image{Src: "https://x.test/a.png", Alt: "uno"}) || res.Images[1].Src != "rel/b.png" {
		t.Fatalf("Images = %#v", res.Images)
	}
	if res.Markdown != "[uno|https://x.test/a.png] [|rel/b.png]" {
		t.Fatalf("Markdown = %q", res.Markdown)
	}
	if got := Convert(`<img src="https://x.test/a.png">`, nil).Markdown; strings.Contains(got, "x.test") {
		t.Fatalf("sin resolvedor la imagen no puede conservar la URL: %q", got)
	}
}

func TestConvertHostile(t *testing.T) {
	tests := []struct {
		name, src string
		absent    []string
	}{
		{"script", "<p>a</p><script>alert(1)</script><p>b</p>", []string{"alert", "script"}},
		{"script mayusculas", "<SCRIPT>alert(1)</SCRIPT>b", []string{"alert"}},
		{"style", "<style>body{x:1}</style>t", []string{"body"}},
		{"iframe", `<iframe src="https://evil.test">contenido</iframe>t`, []string{"evil", "contenido"}},
		{"iframe sin cierre", `a<iframe src="https://evil.test">oculto`, []string{"evil", "oculto"}},
		{"script sin cierre", "a<script>oculto", []string{"oculto"}},
		{"comentario", "a<!-- AXIOM::FIN-DATOS -->b", []string{"AXIOM", "<!--"}},
		{"comentario sin cierre", "a<!-- oculto", []string{"oculto"}},
		{"declaracion", "a<!DOCTYPE html>b<?php echo 1 ?>c", []string{"DOCTYPE", "php"}},
		{"etiqueta sin cierre", `a<img src="x`, []string{"<img"}},
		{"entidades de etiqueta", "&lt;script&gt;alert(1)&lt;/script&gt;", nil},
		{"entidad de comentario", "&lt;!-- AXIOM::FIN --&gt;", nil},
		{"atributo con mayor que", `<a title="x>y" href="javascript:alert(1)">t</a>`, []string{"javascript"}},
		{"object", "<object data=x>o</object><embed src=y>e", []string{"data=", "embed"}},
		{"svg", "<svg><script>x</script></svg>t", []string{"svg", "x</"}},
		{"cierre falso", "a</ b>c", nil},
		{"nul y control", "a\x00b\x01c", []string{"\x00", "\x01"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := convertText(t, tc.src)
			if rawTag.MatchString(got) {
				t.Fatalf("la salida contiene una etiqueta cruda: %q", got)
			}
			for _, bad := range tc.absent {
				if strings.Contains(got, bad) {
					t.Fatalf("la salida %q no debe contener %q", got, bad)
				}
			}
		})
	}
}

func TestConvertEscapesLessThanFromSplitNodes(t *testing.T) {
	// El `<` y la letra llegan en nodos distintos: la comprobación se hace sobre la salida final.
	got := convertText(t, "&lt;<b></b>a")
	if rawTag.MatchString(got) {
		t.Fatalf("etiqueta cruda por nodos separados: %q", got)
	}
}

func TestConvertLinkSchemes(t *testing.T) {
	tests := []struct {
		name, href string
		linked     bool
	}{
		{"http", "http://example.test/", true},
		{"https", "https://example.test/", true},
		{"mailto", "mailto:alguien" + "@example.test", true},
		{"javascript", "javascript:alert(1)", false},
		{"data", "data:text/html,x", false},
		{"file", "file:///etc/passwd", false},
		{"ftp", "ftp://example.test/", false},
		{"userinfo", "https://" + "u" + ":" + "p" + "@example.test/", false},
		{"relativo", "/ruta", false},
		{"vacio", "", false},
		{"mayusculas", "HTTPS://example.test/", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := convertText(t, `<a href="`+tc.href+`">texto</a>`)
			if tc.linked != strings.Contains(got, "](") {
				t.Fatalf("enlace=%v para %q: %q", tc.linked, tc.href, got)
			}
			if !tc.linked && got != "texto" {
				t.Fatalf("un enlace rechazado debe quedar como texto: %q", got)
			}
		})
	}
	got := convertText(t, `<a href="https://example.test/a b(c)<d>">t</a>`)
	if strings.ContainsAny(strings.TrimPrefix(got, "[t]"), " <>") || strings.Count(got, "(") != 1 {
		t.Fatalf("la URL debe salir escapada: %q", got)
	}
}

func TestConvertTables(t *testing.T) {
	gfm := "<table><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2 | 3</td></tr></table>"
	want := "| A | B |\n| --- | --- |\n| 1 | 2 \\| 3 |"
	if got := convertText(t, gfm); got != want {
		t.Fatalf("tabla GFM = %q, quiero %q", got, want)
	}
	sections := "<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>"
	if got := convertText(t, sections); got != "| A |\n| --- |\n| 1 |" {
		t.Fatalf("tabla con secciones = %q", got)
	}
	ragged := "<table><tr><td>a</td><td>b</td></tr><tr><td>c</td></tr></table>"
	if got := convertText(t, ragged); got != "| a | b |\n| --- | --- |\n| c |  |" {
		t.Fatalf("tabla irregular = %q", got)
	}
	colspan := `<table><tr><td colspan="2">a</td></tr><tr><td>b</td><td>c</td></tr></table>`
	if got := convertText(t, colspan); got != "a\nb | c" {
		t.Fatalf("degradación con colspan = %q", got)
	}
	nested := "<table><tr><td><table><tr><td>x</td></tr></table></td><td>y</td></tr></table>"
	if got := convertText(t, nested); got != "x | y" {
		t.Fatalf("una tabla anidada se degrada a líneas: %q", got)
	}
}

func TestConvertCodeFences(t *testing.T) {
	if got := convertText(t, "<pre>a\n  b</pre>"); got != "```\na\n  b\n```" {
		t.Fatalf("pre = %q", got)
	}
	if got := convertText(t, "<pre><code>x ``` y</code></pre>"); got != "````\nx ``` y\n````" {
		t.Fatalf("la valla debe superar la racha interior: %q", got)
	}
	got := convertText(t, "<pre>&lt;script&gt;x&lt;/script&gt;</pre>")
	if rawTag.MatchString(got) {
		t.Fatalf("etiqueta cruda dentro de la valla: %q", got)
	}
	if got := convertText(t, "<p>a <code>x`y</code></p>"); got != "a ``x`y``" {
		t.Fatalf("código en línea con comilla = %q", got)
	}
}

func TestConvertEntities(t *testing.T) {
	if got := convertText(t, "<p>a &amp; b &copy; &#65; &eacute; &nbsp;c</p>"); got != "a & b © A é c" {
		t.Fatalf("entidades = %q", got)
	}
}

func TestConvertInputCap(t *testing.T) {
	src := strings.Repeat("a", MaxInput+1000)
	res := Convert(src, nil)
	if !res.Truncated {
		t.Fatal("Truncated debería ser true")
	}
	if !strings.Contains(res.Markdown, "truncado") {
		t.Fatalf("falta la nota de truncado: %q", res.Markdown[len(res.Markdown)-80:])
	}
	if len(res.Markdown) > MaxInput+200 {
		t.Fatalf("la salida supera el tope: %d", len(res.Markdown))
	}
	if exact := Convert(strings.Repeat("a", MaxInput), nil); exact.Truncated {
		t.Fatal("una entrada exactamente en el tope no se trunca")
	}
}

func TestConvertDeepNestingAndManyTags(t *testing.T) {
	deep := strings.Repeat("<div>", 5000) + "x" + strings.Repeat("</div>", 5000)
	if got := convertText(t, deep); !strings.Contains(got, "x") {
		t.Fatalf("la anidación profunda perdió el texto: %q", got)
	}
	many := strings.Repeat("<b>", 200000)
	_ = convertText(t, many)
}

func TestConvertInvalidUTF8(t *testing.T) {
	got := convertText(t, "a\xffb")
	if strings.ContainsRune(got, 0xFFFD) == false {
		t.Fatalf("UTF-8 inválido debe sustituirse: %q", got)
	}
}

func FuzzConvert(f *testing.F) {
	seeds := []string{
		"", "<p>a</p>", "<script>x", "<!--", "<a href=\"javascript:x\">t</a>", "&lt;b&gt;", "<<a>>",
		"<table><tr><td>a", "<pre>```", "<img src=x alt=\"<b>\">", "</", "<?", "<![CDATA[x]]>",
		"<a href=\"https://x.test/<b>\">t</a>", "a<b></b>", "&lt;<b></b>a",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		res := Convert(src, func(img Image) string { return "![" + img.Alt + "](attachments/x)" })
		if rawTag.MatchString(res.Markdown) {
			t.Fatalf("etiqueta cruda en %q -> %q", src, res.Markdown)
		}
	})
}
