// Package specsummary renderiza de forma determinista el resumen de la spec de un cambio
// como HTML simple (D12 de eventos-ado). Es un paquete hoja: solo usa la biblioteca
// estándar, no lee ficheros ni conoce el tracker, y quien lo llama le pasa un modelo ya
// extraído.
//
// El HTML emitido se limita a `h2`, `h3`, `p`, `ul`, `li`, `code` y `strong`, sin ningún
// atributo y con todo el texto escapado. Hoy ninguna regla necesita `ul` ni `li`, así que
// el renderizador no los emite.
package specsummary

import (
	"html"
	"sort"
	"strconv"
	"strings"
)

// MaxBytes es el tamaño máximo del HTML renderizado (64 KiB).
const MaxBytes = 64 * 1024

// Operation es la operación de la delta que declara un requisito.
type Operation string

// Operaciones de una delta OpenSpec.
const (
	OpAdded    Operation = "ADDED"
	OpModified Operation = "MODIFIED"
	OpRemoved  Operation = "REMOVED"
	OpRenamed  Operation = "RENAMED"
)

// Requirement es un requisito del modelo. Title es el título tal como debe verse (en un
// RENAMED, el nombre nuevo); RenamedFrom es el título anterior, solo en un RENAMED;
// Paragraph es el primer párrafo normativo, ya extraído (puede ser vacío).
type Requirement struct {
	Title       string
	Paragraph   string
	Operation   Operation
	RenamedFrom string
}

// Capability es una capacidad con sus requisitos, en el orden del fichero.
type Capability struct {
	Name         string
	Requirements []Requirement
}

// Model es lo que el renderizador recibe.
type Model struct {
	Capabilities []Capability
}

// Render devuelve el HTML del modelo: capacidades por orden alfabético (`h2`) y, dentro de
// cada una, todos los requisitos en el orden recibido (`h3` y, si procede, `p`). Si el
// resultado supera MaxBytes se vuelve a renderizar sin los `p` y, si aún no cabe, se
// cortan requisitos y se cierra con `<p>… y N requisitos más</p>`. Con el mismo modelo
// devuelve siempre los mismos bytes.
func Render(m Model) string {
	caps := append([]Capability(nil), m.Capabilities...)
	sort.SliceStable(caps, func(i, j int) bool { return caps[i].Name < caps[j].Name })

	full := write(caps, true, -1)
	if len(full) <= MaxBytes {
		return full
	}
	bare := write(caps, false, -1)
	if len(bare) <= MaxBytes {
		return bare
	}
	total := 0
	for _, c := range caps {
		total += len(c.Requirements)
	}
	// Mayor número de requisitos que caben con el cierre; búsqueda binaria determinista.
	lo, hi := 0, total-1
	best := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if len(write(caps, false, mid)) <= MaxBytes {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return write(caps, false, best)
}

// write emite el HTML. Con keep >= 0 solo se emiten los primeros keep requisitos y se
// añade el cierre con el resto; una capacidad sin ningún requisito emitido no lleva `h2`.
func write(caps []Capability, paragraphs bool, keep int) string {
	var b strings.Builder
	emitted, total := 0, 0
	for _, c := range caps {
		total += len(c.Requirements)
	}
	for _, c := range caps {
		headed := false
		for _, r := range c.Requirements {
			if keep >= 0 && emitted >= keep {
				break
			}
			if !headed {
				b.WriteString("<h2>" + html.EscapeString(c.Name) + "</h2>\n")
				headed = true
			}
			b.WriteString("<h3>" + html.EscapeString(r.Title) + "</h3>\n")
			if p := paragraph(r, paragraphs); p != "" {
				b.WriteString("<p>" + p + "</p>\n")
			}
			emitted++
		}
	}
	if keep >= 0 && total > emitted {
		b.WriteString("<p>… y " + strconv.Itoa(total-emitted) + " requisitos más</p>\n")
	}
	return b.String()
}

// paragraph compone el contenido del `p` de un requisito: el marcador de la operación en
// `strong` (REMOVED y RENAMED) seguido del primer párrafo con su marcado en línea.
func paragraph(r Requirement, withText bool) string {
	var marker string
	switch r.Operation {
	case OpRemoved:
		marker = "<strong>Retirado.</strong>"
	case OpRenamed:
		marker = "<strong>Renombrado desde:</strong> " + html.EscapeString(r.RenamedFrom)
	}
	text := ""
	if withText && strings.TrimSpace(r.Paragraph) != "" {
		text = inline(r.Paragraph)
	}
	switch {
	case marker != "" && text != "":
		return marker + " " + text
	case marker != "":
		return marker
	default:
		return text
	}
}

// inline escapa el texto y convierte los pares completos de `**` y de “ ` “ en
// `strong` y `code`. Un delimitador sin cerrar (o un par vacío) se emite literal y
// escapado. El contenido de un par no se interpreta de nuevo.
func inline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], "**"):
			if end := strings.Index(s[i+2:], "**"); end > 0 {
				b.WriteString("<strong>" + html.EscapeString(s[i+2:i+2+end]) + "</strong>")
				i += end + 4
				continue
			}
			b.WriteString("**")
			i += 2
		case s[i] == '`':
			if end := strings.IndexByte(s[i+1:], '`'); end > 0 {
				b.WriteString("<code>" + html.EscapeString(s[i+1:i+1+end]) + "</code>")
				i += end + 2
				continue
			}
			b.WriteString("`")
			i++
		default:
			j := i
			for j < len(s) && s[j] != '`' && !strings.HasPrefix(s[j:], "**") {
				j++
			}
			b.WriteString(html.EscapeString(s[i:j]))
			i = j
		}
	}
	return b.String()
}

// FirstParagraph extrae el primer párrafo normativo del cuerpo de un requisito (con su
// línea de título): las líneas no vacías consecutivas tras el título, antes de cualquier
// línea que empiece por `#`, `-`, `|`, `>`, una valla o `(`. Si el primer bloque no es un
// párrafo devuelve "". Las líneas se unen con un espacio.
func FirstParagraph(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	var para []string
	for _, line := range lines[1:] {
		t := strings.TrimSpace(line)
		if t == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		if startsBlock(t) {
			break
		}
		para = append(para, t)
	}
	return strings.Join(para, " ")
}

// startsBlock dice si la línea (sin espacios en los extremos) abre algo que no es párrafo.
func startsBlock(t string) bool {
	if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
		return true
	}
	switch t[0] {
	case '#', '-', '|', '>', '(':
		return true
	}
	return false
}
