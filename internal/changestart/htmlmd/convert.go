// Package htmlmd convierte el HTML de un work item en Markdown de solo lectura para un agente.
//
// La salida es dato, no un documento que pinte un navegador: el riesgo que se contiene es la
// inyección de instrucciones, no XSS. Por eso el conversor es propio, conservador y sobre la
// biblioteca estándar: lo que no entiende lo descarta o lo deja como texto, y la salida final nunca
// contiene una etiqueta HTML cruda.
package htmlmd

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxInput es el tope de entrada por campo (1 MiB). Lo que sobra se corta con una nota.
const MaxInput = 1 << 20

const (
	maxDepth = 128
	maxNodes = 200000

	hardBreak = "\x01" // los caracteres de control de la entrada se eliminan, así que no hay colisión

	truncatedNote = "\n\n[contenido truncado: superaba 1 MiB]"
)

// Image es una imagen encontrada en el HTML, en orden de documento.
type Image struct{ Src, Alt string }

// Result es la conversión de un campo.
type Result struct {
	Markdown  string
	Images    []Image
	Truncated bool
}

// Convert convierte src. resolve decide el Markdown de cada imagen (la reescritura a un adjunto local
// o a una nota de omisión); sin él, cada imagen queda como «[imagen omitida]» sin la URL.
func Convert(src string, resolve func(Image) string) Result {
	var res Result
	if len(src) > MaxInput {
		cut := MaxInput
		for cut > 0 && !utf8.RuneStart(src[cut]) {
			cut--
		}
		src = src[:cut]
		res.Truncated = true
	}
	src = strings.ToValidUTF8(src, "�")
	src = strings.Map(func(r rune) rune {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0x7f {
			return -1
		}
		return r
	}, src)

	root, capped := parse(src)
	c := &conv{resolve: resolve}
	md := strings.TrimSpace(c.blocks(root.kids))
	md = escapeRawTags(md)
	if res.Truncated || capped {
		res.Truncated = true
		md += truncatedNote
	}
	res.Markdown = md
	res.Images = c.images
	return res
}

// escapeRawTags escribe `&lt;` en todo `<` seguido de letra, `/`, `!` o `?`. Se aplica a la salida
// final, no a cada nodo, para que dos nodos contiguos no puedan formar una etiqueta.
func escapeRawTags(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '<' && i+1 < len(s) && (isLetter(s[i+1]) || s[i+1] == '/' || s[i+1] == '!' || s[i+1] == '?') {
			b.WriteString("&lt;")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

type node struct {
	tag   string // vacío: nodo de texto
	text  string
	attrs tagAttrs
	kids  []*node
}

var known = map[string]bool{
	"p": true, "div": true, "br": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true, "b": true, "strong": true, "i": true, "em": true, "a": true,
	"img": true, "table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "td": true,
	"th": true, "pre": true, "code": true, "blockquote": true, "hr": true,
}

var void = map[string]bool{"br": true, "img": true, "hr": true}

func isHeading(tag string) bool {
	return len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6'
}

func isBlock(tag string) bool {
	switch tag {
	case "p", "div", "ul", "ol", "li", "table", "thead", "tbody", "tfoot", "tr", "td", "th", "pre", "blockquote", "hr":
		return true
	}
	return isHeading(tag)
}

// opensBlock son las etiquetas cuya apertura cierra un <p> abierto.
func opensBlock(tag string) bool {
	switch tag {
	case "p", "div", "ul", "ol", "table", "pre", "blockquote", "hr":
		return true
	}
	return isHeading(tag)
}

// parse construye el árbol con cierres implícitos mínimos. Solo entran las etiquetas conocidas: las
// demás son transparentes. Devuelve capped si se alcanzó el tope de nodos.
func parse(src string) (*node, bool) {
	root := &node{}
	stack := []*node{root}
	count := 0
	capped := false
	tz := newTokenizer(src)

	closeNearest := func(tag string, boundary ...string) {
		for i := len(stack) - 1; i > 0; i-- {
			if stack[i].tag == tag {
				stack = stack[:i]
				return
			}
			for _, b := range boundary {
				if stack[i].tag == b {
					return
				}
			}
		}
	}
	add := func(n *node) {
		top := stack[len(stack)-1]
		top.kids = append(top.kids, n)
	}

	for {
		tok, ok := tz.next()
		if !ok {
			break
		}
		if count >= maxNodes {
			capped = true
			break
		}
		switch tok.kind {
		case tokText:
			count++
			add(&node{text: tok.text})
		case tokStart:
			if !known[tok.name] {
				continue
			}
			if opensBlock(tok.name) && stack[len(stack)-1].tag == "p" {
				stack = stack[:len(stack)-1]
			}
			switch tok.name {
			case "li":
				closeNearest("li", "ul", "ol")
			case "td", "th":
				closeNearest("td", "tr", "table")
				closeNearest("th", "tr", "table")
			case "tr":
				closeNearest("tr", "table")
			case "thead", "tbody", "tfoot":
				for len(stack) > 1 {
					t := stack[len(stack)-1].tag
					if t != "td" && t != "th" && t != "tr" && t != "thead" && t != "tbody" && t != "tfoot" {
						break
					}
					stack = stack[:len(stack)-1]
				}
			}
			count++
			n := &node{tag: tok.name, attrs: tok.attrs}
			add(n)
			if !void[tok.name] && !tok.selfClosing && len(stack) < maxDepth {
				stack = append(stack, n)
			}
		case tokEnd:
			if !known[tok.name] || void[tok.name] {
				continue
			}
			for i := len(stack) - 1; i > 0; i-- {
				if stack[i].tag == tok.name {
					stack = stack[:i]
					break
				}
			}
		}
	}
	return root, capped
}

type conv struct {
	resolve func(Image) string
	images  []Image
	flat    int // >0 mientras se renderiza una celda de una tabla degradada
}

// blocks renderiza una secuencia de nodos como bloques separados por una línea en blanco.
func (c *conv) blocks(nodes []*node) string { return strings.Join(c.blockParts(nodes), "\n\n") }

func (c *conv) blockParts(nodes []*node) []string {
	var out []string
	var run []*node
	flush := func() {
		if len(run) == 0 {
			return
		}
		if s := c.inline(run); s != "" {
			out = append(out, s)
		}
		run = nil
	}
	for _, n := range nodes {
		if n.tag != "" && isBlock(n.tag) {
			flush()
			if s := c.block(n); s != "" {
				out = append(out, s)
			}
			continue
		}
		run = append(run, n)
	}
	flush()
	return out
}

func (c *conv) block(n *node) string {
	switch {
	case isHeading(n.tag):
		text := oneLine(c.inline(n.kids))
		if text == "" {
			return ""
		}
		return strings.Repeat("#", int(n.tag[1]-'0')) + " " + text
	case n.tag == "ul" || n.tag == "ol":
		return c.list(n)
	case n.tag == "li":
		return c.item("- ", n.kids)
	case n.tag == "table":
		return c.table(n)
	case n.tag == "pre":
		return fence(rawText(n))
	case n.tag == "blockquote":
		inner := c.blocks(n.kids)
		if inner == "" {
			return ""
		}
		lines := strings.Split(inner, "\n")
		for i, l := range lines {
			if l == "" {
				lines[i] = ">"
			} else {
				lines[i] = "> " + l
			}
		}
		return strings.Join(lines, "\n")
	case n.tag == "hr":
		return "---"
	}
	return c.blocks(n.kids) // p, div y contenedores de tabla sueltos
}

func (c *conv) list(n *node) string {
	ordered := n.tag == "ol"
	var items [][]*node
	var loose []*node
	flushLoose := func() {
		if len(loose) > 0 {
			items = append(items, loose)
			loose = nil
		}
	}
	for _, k := range n.kids {
		if k.tag == "li" {
			flushLoose()
			items = append(items, k.kids)
			continue
		}
		loose = append(loose, k)
	}
	flushLoose()

	var lines []string
	num := 0
	for _, it := range items {
		marker := "- "
		if ordered {
			marker = fmt.Sprintf("%d. ", num+1)
		}
		if s := c.item(marker, it); s != "" {
			lines = append(lines, s)
			num++
		}
	}
	return strings.Join(lines, "\n")
}

func (c *conv) item(marker string, kids []*node) string {
	parts := c.blockParts(kids)
	if len(parts) == 0 {
		return ""
	}
	text := strings.Join(parts, "\n")
	pad := strings.Repeat(" ", len(marker))
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		switch {
		case i == 0:
			lines[i] = marker + l
		case l != "":
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

func (c *conv) table(n *node) string {
	var rows [][]*node
	var collect func(kids []*node)
	collect = func(kids []*node) {
		for _, k := range kids {
			switch k.tag {
			case "thead", "tbody", "tfoot":
				collect(k.kids)
			case "tr":
				var cells []*node
				for _, cell := range k.kids {
					if cell.tag == "td" || cell.tag == "th" {
						cells = append(cells, cell)
					}
				}
				rows = append(rows, cells)
			}
		}
	}
	collect(n.kids)

	degrade := c.flat > 0
	for _, r := range rows {
		for _, cell := range r {
			if cell.attrs.Span || hasTag(cell.kids, "table") {
				degrade = true
			}
		}
	}
	if degrade {
		c.flat++
		defer func() { c.flat-- }()
	}

	text := make([][]string, 0, len(rows))
	cols := 0
	for _, r := range rows {
		cells := make([]string, 0, len(r))
		for _, cell := range r {
			s := oneLine(c.blocks(cell.kids))
			if !degrade {
				s = strings.ReplaceAll(s, "|", `\|`)
			}
			cells = append(cells, s)
		}
		if len(cells) > 0 {
			text = append(text, cells)
		}
		if len(cells) > cols {
			cols = len(cells)
		}
	}
	if len(text) == 0 {
		return ""
	}
	if degrade {
		lines := make([]string, 0, len(text))
		for _, cells := range text {
			var nonEmpty []string
			for _, s := range cells {
				if s != "" {
					nonEmpty = append(nonEmpty, s)
				}
			}
			if len(nonEmpty) > 0 {
				lines = append(lines, strings.Join(nonEmpty, " | "))
			}
		}
		return strings.Join(lines, "\n")
	}
	line := func(cells []string) string {
		padded := make([]string, cols)
		copy(padded, cells)
		return "| " + strings.Join(padded, " | ") + " |"
	}
	sep := make([]string, cols)
	for i := range sep {
		sep[i] = "---"
	}
	out := []string{line(text[0]), "| " + strings.Join(sep, " | ") + " |"}
	for _, cells := range text[1:] {
		out = append(out, line(cells))
	}
	return strings.Join(out, "\n")
}

func hasTag(nodes []*node, tag string) bool {
	for _, n := range nodes {
		if n.tag == tag || hasTag(n.kids, tag) {
			return true
		}
	}
	return false
}

// inline renderiza una secuencia en línea: el espacio en blanco se colapsa y <br> es un salto duro.
func (c *conv) inline(nodes []*node) string {
	raw := c.inlineRaw(nodes)
	parts := strings.Split(raw, hardBreak)
	for i, p := range parts {
		parts[i] = oneLine(p)
	}
	for len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, "  \n")
}

func (c *conv) inlineRaw(nodes []*node) string {
	var b strings.Builder
	for _, n := range nodes {
		switch {
		case n.tag == "":
			b.WriteString(n.text)
		case n.tag == "br":
			b.WriteString(hardBreak)
		case n.tag == "img":
			b.WriteString(" " + c.image(n) + " ")
		case n.tag == "b" || n.tag == "strong":
			b.WriteString(c.emph("**", n.kids))
		case n.tag == "i" || n.tag == "em":
			b.WriteString(c.emph("*", n.kids))
		case n.tag == "a":
			b.WriteString(c.link(n))
		case n.tag == "code":
			b.WriteString(codeSpan(rawText(n)))
		case isBlock(n.tag):
			// Un bloque dentro de un elemento en línea se aplana a una línea.
			b.WriteString(" " + oneLine(c.block(n)) + " ")
		}
	}
	return b.String()
}

// span renderiza el interior de un elemento en línea sin saltos duros.
func (c *conv) span(kids []*node) string {
	return oneLine(strings.ReplaceAll(c.inlineRaw(kids), hardBreak, " "))
}

func (c *conv) image(n *node) string {
	img := Image{Src: strings.TrimSpace(n.attrs.Src), Alt: n.attrs.Alt}
	c.images = append(c.images, img)
	if c.resolve == nil {
		return "[imagen omitida]"
	}
	return c.resolve(img)
}

func (c *conv) link(n *node) string {
	text := c.span(n.kids)
	target, ok := safeURL(n.attrs.Href)
	if !ok {
		return text
	}
	if text == "" {
		text = target
	}
	text = strings.NewReplacer("[", `\[`, "]", `\]`).Replace(text)
	return "[" + text + "](" + target + ")"
}

// safeURL acepta solo http, https y mailto, sin userinfo, y devuelve la URL con los caracteres que
// rompen el Markdown o forman marcado escritos en porcentaje.
func safeURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return "", false
	}
	switch u.Scheme {
	case "http", "https":
		if u.Host == "" {
			return "", false
		}
	case "mailto":
		if u.Opaque == "" {
			return "", false
		}
	default:
		return "", false
	}
	var b strings.Builder
	for _, r := range raw {
		switch r {
		case ' ':
			b.WriteString("%20")
		case '<':
			b.WriteString("%3C")
		case '>':
			b.WriteString("%3E")
		case '(':
			b.WriteString("%28")
		case ')':
			b.WriteString("%29")
		case '[':
			b.WriteString("%5B")
		case ']':
			b.WriteString("%5D")
		default:
			if unicode.IsControl(r) {
				return "", false
			}
			b.WriteRune(r)
		}
	}
	return b.String(), true
}

// emph envuelve el interior con una marca conservando un espacio inicial o final.
func (c *conv) emph(mark string, kids []*node) string {
	raw := strings.ReplaceAll(c.inlineRaw(kids), hardBreak, " ")
	inner := oneLine(raw)
	if inner == "" {
		return raw
	}
	lead, trail := "", ""
	if r, _ := utf8.DecodeRuneInString(raw); unicode.IsSpace(r) {
		lead = " "
	}
	if r, _ := utf8.DecodeLastRuneInString(raw); unicode.IsSpace(r) {
		trail = " "
	}
	return lead + mark + inner + mark + trail
}

// oneLine colapsa todo el espacio en blanco, saltos incluidos, a un único espacio.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// rawText devuelve el texto de n y sus descendientes sin colapsar espacio; <br> es un salto.
func rawText(n *node) string {
	var b strings.Builder
	var walk func(nodes []*node)
	walk = func(nodes []*node) {
		for _, k := range nodes {
			switch {
			case k.tag == "":
				b.WriteString(k.text)
			case k.tag == "br":
				b.WriteByte('\n')
			default:
				walk(k.kids)
			}
		}
	}
	walk(n.kids)
	return b.String()
}

// longestRun es la racha más larga de comillas invertidas de s.
func longestRun(s string) int {
	best, cur := 0, 0
	for _, r := range s {
		if r == '`' {
			cur++
			if cur > best {
				best = cur
			}
			continue
		}
		cur = 0
	}
	return best
}

func fence(content string) string {
	content = strings.TrimPrefix(content, "\n")
	content = strings.TrimRight(content, "\n")
	if strings.TrimSpace(content) == "" {
		return ""
	}
	ticks := strings.Repeat("`", max(3, longestRun(content)+1))
	return ticks + "\n" + content + "\n" + ticks
}

func codeSpan(content string) string {
	content = oneLine(content)
	if content == "" {
		return ""
	}
	ticks := strings.Repeat("`", longestRun(content)+1)
	if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
		content = " " + content + " "
	}
	return ticks + content + ticks
}
