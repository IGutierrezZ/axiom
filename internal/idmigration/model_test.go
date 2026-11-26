// Package idmigration_test aloja la herramienta de un solo uso que migra las specs propias de
// Axiom de `REQ-n.m` a `RF-<DOM>-<NN>` (C-03-e). Es solo código de prueba: no hay ficheros de
// producción, así que no cuenta para el trinquete de código muerto ni deja binarios. La
// herramienta se ejecuta en proceso con `AXIOM_IDMIGRATION=map|plan|write go test`.
//
// Este fichero reúne las piezas de bajo nivel que comparten el generador, el reescritor y el
// modelo de residuos: lectura de tokens, líneas, Markdown, Go y sustitución por desplazamientos.
package idmigration_test

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const rfTemplate = "RF-<DOM>-<NN>"

// rfPattern es el patrón con el que Axiom numera los IDs nuevos.
var rfPattern = func() reqid.Pattern {
	p, err := reqid.Parse(rfTemplate)
	if err != nil {
		panic(err)
	}
	return p
}()

// ---- tokens heredados ----

// tokenRe lee un ID heredado, con el sufijo de una letra de `REQ-27.2b`.
var tokenRe = regexp.MustCompile(`REQ-(\d+)\.(\d+)([a-z]?)`)

// hit es una aparición de un ID heredado en un texto, con desplazamientos relativos a él.
type hit struct {
	Start, End int
	Text       string
	Series     string
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

func digitAt(s string, i int) bool { return i < len(s) && s[i] >= '0' && s[i] <= '9' }

// findTokens devuelve los IDs heredados completos del texto: ni precedidos de una letra, una
// cifra o `_`, ni seguidos de ellos ni de `.` y una cifra (`REQ-1.1.2` no es un token). Así
// `REQ-1.1` nunca se lee dentro de `REQ-1.10` ni de `REQ-11.1`.
func findTokens(text string) []hit {
	var out []hit
	for _, m := range tokenRe.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[0], m[1]
		if s > 0 {
			r, size := utf8.DecodeLastRuneInString(text[:s])
			if isWordRune(r) {
				continue
			}
			if r == '-' && s-size > 0 {
				if rr, _ := utf8.DecodeLastRuneInString(text[:s-size]); unicode.IsLetter(rr) || rr == '_' {
					continue
				}
			}
		}
		if e < len(text) {
			r, _ := utf8.DecodeRuneInString(text[e:])
			if isWordRune(r) || (r == '.' && digitAt(text, e+1)) {
				continue
			}
		}
		out = append(out, hit{Start: s, End: e, Text: text[s:e], Series: text[m[2]:m[3]]})
	}
	return out
}

// contRe lee una continuación sin prefijo tras un ID: `/21.6`, `, 21.6`, ` y 24.27`, ` a 30.14`,
// ` al 30.14` o `-21.12` (también con el guion largo).
var contRe = regexp.MustCompile(`^(/|, | y | al | a |-|–)(\d+)\.(\d+)([a-z]?)`)

// elem es una continuación: su separador y el tramo `a.c` que lo sigue.
type elem struct {
	Sep      string
	SepStart int
	Start    int
	End      int
	Series   string
	Num      string
	Suffix   string
}

func (e elem) token() string { return "REQ-" + e.Series + "." + e.Num + e.Suffix }

// continuations devuelve las continuaciones de la misma serie que siguen a h.
func continuations(text string, h hit) []elem {
	var out []elem
	pos := h.End
	for pos < len(text) {
		m := contRe.FindStringSubmatchIndex(text[pos:])
		if m == nil || text[pos+m[4]:pos+m[5]] != h.Series {
			break
		}
		end := pos + m[1]
		if end < len(text) {
			r, _ := utf8.DecodeRuneInString(text[end:])
			if isWordRune(r) || (r == '.' && digitAt(text, end+1)) {
				break
			}
		}
		out = append(out, elem{
			Sep: text[pos : pos+m[3]], SepStart: pos, Start: pos + m[3], End: end,
			Series: h.Series, Num: text[pos+m[6] : pos+m[7]], Suffix: text[pos+m[8] : pos+m[9]],
		})
		pos = end
	}
	return out
}

// ---- líneas ----

// line es una línea sin su terminador (`\n` o `\r\n`); Start y End son absolutos.
type line struct {
	Start, End int
	Text       string
}

func splitLines(s string) []line {
	var out []line
	start := 0
	for start <= len(s) {
		i := strings.IndexByte(s[start:], '\n')
		end, next := len(s), len(s)+1
		if i >= 0 {
			end = start + i
			next = end + 1
		}
		te := end
		if te > start && s[te-1] == '\r' {
			te--
		}
		out = append(out, line{Start: start, End: te, Text: s[start:te]})
		start = next
	}
	return out
}

// lineIndex localiza la línea de un desplazamiento.
type lineIndex struct{ lines []line }

func newLineIndex(s string) lineIndex { return lineIndex{lines: splitLines(s)} }

// at devuelve la línea del desplazamiento y su número (desde 1).
func (li lineIndex) at(off int) (line, int) {
	i := sort.Search(len(li.lines), func(i int) bool { return li.lines[i].Start > off }) - 1
	if i < 0 {
		i = 0
	}
	return li.lines[i], i + 1
}

// ---- Markdown ----

type fenceOpen struct {
	ch   byte
	n    int
	info string
}

func leadingSpaces(t string) int {
	n := 0
	for n < len(t) && t[n] == ' ' {
		n++
	}
	return n
}

// openingFence lee una valla de apertura (CommonMark: hasta tres espacios, tres o más ` o ~).
func openingFence(t string) (fenceOpen, bool) {
	sp := leadingSpaces(t)
	if sp > 3 || sp >= len(t) || (t[sp] != '`' && t[sp] != '~') {
		return fenceOpen{}, false
	}
	ch := t[sp]
	n := 0
	for sp+n < len(t) && t[sp+n] == ch {
		n++
	}
	if n < 3 {
		return fenceOpen{}, false
	}
	info := strings.TrimSpace(t[sp+n:])
	if ch == '`' && strings.Contains(info, "`") {
		return fenceOpen{}, false
	}
	return fenceOpen{ch: ch, n: n, info: info}, true
}

// closesFence dice si t cierra la valla f: mismo carácter, al menos su longitud y nada más.
func closesFence(t string, f fenceOpen) bool {
	sp := leadingSpaces(t)
	if sp > 3 || sp >= len(t) || t[sp] != f.ch {
		return false
	}
	n := 0
	for sp+n < len(t) && t[sp+n] == f.ch {
		n++
	}
	return n >= f.n && strings.TrimSpace(t[sp+n:]) == ""
}

// mdLine es una línea de Markdown con su pertenencia a una valla.
type mdLine struct {
	line
	InFence bool   // marcador de apertura, cuerpo o marcador de cierre
	Marker  bool   // es el marcador de apertura o de cierre
	Open    bool   // es el marcador de apertura
	Info    string // cadena de información de la valla que la contiene
}

func analyzeMarkdown(ls []line) []mdLine {
	out := make([]mdLine, len(ls))
	var cur *fenceOpen
	for i, l := range ls {
		out[i] = mdLine{line: l}
		switch {
		case cur == nil:
			if f, ok := openingFence(l.Text); ok {
				cur = &f
				out[i].InFence, out[i].Marker, out[i].Open, out[i].Info = true, true, true, f.info
			}
		case closesFence(l.Text, *cur):
			out[i].InFence, out[i].Marker, out[i].Info = true, true, cur.info
			cur = nil
		default:
			out[i].InFence, out[i].Info = true, cur.info
		}
	}
	return out
}

// seg es un tramo de una línea (desplazamientos relativos) con su ámbito.
type seg struct {
	Start, End int
	Scope      string
}

const (
	scopeAll     = "all"
	scopeComment = "go-comment"
	scopeString  = "go-string"
	scopeCode    = "md-code"
)

// inlineSegments parte una línea de Markdown en tramos de texto y de código en línea.
func inlineSegments(t string) []seg {
	var out []seg
	textStart := 0
	i := 0
	for i < len(t) {
		if t[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < len(t) && t[i+n] == '`' {
			n++
		}
		closeAt := -1
		for j := i + n; j < len(t); {
			if t[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(t) && t[j+m] == '`' {
				m++
			}
			if m == n {
				closeAt = j + m
				break
			}
			j += m
		}
		if closeAt < 0 {
			i += n
			continue
		}
		if textStart < i {
			out = append(out, seg{textStart, i, scopeAll})
		}
		out = append(out, seg{i, closeAt, scopeCode})
		i, textStart = closeAt, closeAt
	}
	if textStart < len(t) {
		out = append(out, seg{textStart, len(t), scopeAll})
	}
	return out
}

// ---- Go ----

// goSegments devuelve los comentarios y los literales de cadena de un fuente Go, con sus
// desplazamientos absolutos. Un literal sin cerrar o un fuente ilegible dan error.
func goSegments(src string) ([]seg, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var errs []string
	var s scanner.Scanner
	s.Init(file, []byte(src), func(pos token.Position, msg string) { errs = append(errs, fmt.Sprintf("%s: %s", pos, msg)) }, scanner.ScanComments)
	var out []seg
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		off := file.Offset(pos)
		switch tok {
		case token.COMMENT:
			end := len(src)
			if strings.HasPrefix(src[off:], "/*") {
				if i := strings.Index(src[off+2:], "*/"); i >= 0 {
					end = off + 2 + i + 2
				}
			} else {
				if i := strings.IndexByte(src[off:], '\n'); i >= 0 {
					end = off + i
				}
				if end > off && src[end-1] == '\r' {
					end--
				}
			}
			out = append(out, seg{off, end, scopeComment})
		case token.STRING:
			out = append(out, seg{off, stringEnd(src, off), scopeString})
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

func stringEnd(src string, off int) int {
	if src[off] == '`' {
		if i := strings.IndexByte(src[off+1:], '`'); i >= 0 {
			return off + 1 + i + 1
		}
		return len(src)
	}
	for i := off + 1; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		case '\n':
			return i
		}
	}
	return len(src)
}

// formatGo aplica gofmt al fuente reescrito solo si el original ya estaba formateado (así no se
// reescriben bytes ajenos, como los CRLF de un fichero sin formatear); comprueba siempre que
// el resultado sigue siendo Go válido.
func formatGo(orig, updated string) (string, error) {
	if _, err := parser.ParseFile(token.NewFileSet(), "x.go", updated, parser.ParseComments); err != nil {
		return "", fmt.Errorf("el Go reescrito no se puede analizar: %w", err)
	}
	if f0, err := format.Source([]byte(orig)); err == nil && string(f0) == orig {
		f1, err := format.Source([]byte(updated))
		if err != nil {
			return "", fmt.Errorf("gofmt del Go reescrito: %w", err)
		}
		return string(f1), nil
	}
	return updated, nil
}

// ---- sustitución por desplazamientos ----

// edit sustituye content[Start:End] por Repl.
type edit struct {
	Start, End int
	Repl       string
}

// applyEdits aplica ediciones sin solapes; el resto de los bytes queda intacto.
func applyEdits(s string, edits []edit) (string, error) {
	sorted := append([]edit(nil), edits...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	var b strings.Builder
	last := 0
	for _, e := range sorted {
		if e.Start < last || e.End < e.Start || e.End > len(s) {
			return "", fmt.Errorf("ediciones solapadas o fuera de rango en [%d,%d)", e.Start, e.End)
		}
		b.WriteString(s[last:e.Start])
		b.WriteString(e.Repl)
		last = e.End
	}
	b.WriteString(s[last:])
	return b.String(), nil
}

// ---- rutas ----

// globMatch compara una ruta con un patrón de segmentos: `*` no cruza `/` y `**` cruza cero o
// más segmentos.
func globMatch(pattern, p string) bool {
	return matchSegs(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegs(pat, parts []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				return true
			}
			for i := 0; i <= len(parts); i++ {
				if matchSegs(pat[1:], parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		ok, err := path.Match(pat[0], parts[0])
		if err != nil || !ok {
			return false
		}
		pat, parts = pat[1:], parts[1:]
	}
	return len(parts) == 0
}

// inputFiles son las dos entradas curadas de la herramienta: citan tokens por definición.
const (
	residuesPath = "internal/idmigration/residues.yaml"
	contextsPath = "internal/idmigration/contexts.yaml"
	idMapPath    = "openspec/id-map.yaml"
	indexPath    = "openspec/INDEX.md"
)

func isInputFile(p string) bool { return p == residuesPath || p == contextsPath }

// skipPath dice si la herramienta nunca reescribe la ruta (D4): historia, mapa, índice
// regenerado, carpeta de este cambio, fixtures e historia fechada fuera de OpenSpec.
func skipPath(p string) bool {
	switch {
	case strings.HasPrefix(p, "openspec/changes/archive/"),
		strings.HasPrefix(p, "openspec/changes/migrar-ids-rf/"),
		strings.HasPrefix(p, "docs/archive/"),
		strings.HasPrefix(p, "docs/audits/"),
		p == idMapPath, p == indexPath, isInputFile(p):
		return true
	}
	for _, c := range strings.Split(p, "/") {
		if c == "testdata" {
			return true
		}
	}
	return false
}

type fileKind int

const (
	kindText fileKind = iota
	kindSpec
	kindODD
	kindGoProd
	kindGoTest
)

var specPathRe = regexp.MustCompile(`^openspec/specs/([^/]+)/spec\.md$`)

// classify devuelve el tipo de fichero y, en una spec viva, su capacidad.
func classify(p string) (fileKind, string) {
	if m := specPathRe.FindStringSubmatch(p); m != nil {
		return kindSpec, m[1]
	}
	// Un `.go` bajo testdata es un fixture, nunca código de producción: el árbol miniatura de
	// la propia herramienta lleva uno con un literal que el residuo de fixtures debe poder cubrir.
	if strings.Contains("/"+p, "/testdata/") && strings.HasSuffix(p, ".go") {
		return kindText, ""
	}
	switch {
	case strings.HasPrefix(p, "odd/tasks/") && strings.HasSuffix(p, ".md") && !strings.Contains(p[len("odd/tasks/"):], "/"):
		return kindODD, ""
	case strings.HasSuffix(p, "_test.go"):
		return kindGoTest, ""
	case strings.HasSuffix(p, ".go"):
		return kindGoProd, ""
	}
	return kindText, ""
}

// safeRel rechaza rutas absolutas o con `..`: nunca se lee ni se escribe fuera de la raíz.
func safeRel(p string) error {
	if p == "" || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) || (len(p) > 1 && p[1] == ':') {
		return fmt.Errorf("la ruta %q es absoluta o vacía", p)
	}
	for _, c := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if c == ".." {
			return fmt.Errorf("la ruta %q sale de la raíz con `..`", p)
		}
	}
	return nil
}
