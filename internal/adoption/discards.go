package adoption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Discard es una unidad del inventario que el cambio de adopción no migra
// (`- <ruta>#<ancla>: <motivo>` en la sección `## Descartado` de adoption.md).
type Discard struct {
	Path   string
	Anchor string
	Reason string // "" si falta el motivo: Check lo reporta
	Line   int
}

// Origin es una línea `Origen: <ruta>#<ancla> (<ID original>)` del cuerpo de un requisito.
type Origin struct {
	Path       string
	Anchor     string
	OriginalID string
	Line       int
}

var (
	discardsHeading = regexp.MustCompile(`^## Descartado[ \t]*$`)
	h2Heading       = regexp.MustCompile(`^## \S`)
	bulletPrefix    = regexp.MustCompile(`^\s*[-*]\s+`)
	originLine      = regexp.MustCompile(`^\s*(?:[-*]\s+)?Origen:[ \t]*(.*)$`)
	originIDGroup   = regexp.MustCompile(`^\s*\((.*)\)\s*$`)
)

// ReadDiscards lee los descartes de <changeRoot>/adoption.md; sin el fichero no hay
// descartes y no es un error.
func ReadDiscards(changeRoot string) ([]Discard, error) {
	path := filepath.Join(changeRoot, DiscardsFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", path, err)
	}
	return ParseDiscards(string(data)), nil
}

// ParseDiscards analiza la sección `## Descartado` de un adoption.md: una viñeta por
// unidad, hasta el siguiente encabezado de nivel 2. Una viñeta sin motivo se devuelve con
// Reason vacío. Las vallas de código se ignoran.
func ParseDiscards(text string) []Discard {
	var out []Discard
	inSection, fence := false, ""
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if m := fenceLine.FindStringSubmatch(line); m != nil {
			switch marker := m[1][:1]; {
			case fence == "":
				fence = marker
			case fence == marker:
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		switch {
		case discardsHeading.MatchString(line):
			inSection = true
		case h2Heading.MatchString(line):
			inSection = false
		case inSection && bulletPrefix.MatchString(line):
			ref, reason := splitRef(bulletPrefix.ReplaceAllString(line, ""))
			path, anchor := splitAnchor(ref)
			out = append(out, Discard{Path: path, Anchor: anchor, Reason: reason, Line: i + 1})
		}
	}
	return out
}

// splitRef separa `<ruta>#<ancla>: <motivo>`; la referencia puede ir entre comillas
// invertidas si contiene espacios.
func splitRef(s string) (ref, reason string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "`") {
		if end := strings.Index(s[1:], "`"); end >= 0 {
			ref = s[1 : 1+end]
			rest := strings.TrimSpace(s[2+end:])
			return ref, strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		}
	}
	if idx := strings.Index(s, ":"); idx >= 0 {
		return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+1:])
	}
	return s, ""
}

func splitAnchor(ref string) (path, anchor string) {
	if idx := strings.LastIndex(ref, "#"); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

// ParseOrigins devuelve las líneas `Origen:` de un cuerpo de requisito, en orden y fuera
// de vallas. Un requisito puede llevar varias. Line es relativa al cuerpo (desde 1).
func ParseOrigins(body string) []Origin {
	var out []Origin
	fence := ""
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if m := fenceLine.FindStringSubmatch(line); m != nil {
			switch marker := m[1][:1]; {
			case fence == "":
				fence = marker
			case fence == marker:
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		m := originLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rest := strings.TrimSpace(m[1])
		token := ""
		switch {
		case strings.HasPrefix(rest, "`"):
			if end := strings.Index(rest[1:], "`"); end >= 0 {
				token, rest = rest[1:1+end], rest[2+end:]
			} else {
				token, rest = rest[1:], ""
			}
		default:
			if end := strings.IndexAny(rest, " \t"); end >= 0 {
				token, rest = rest[:end], rest[end:]
			} else {
				token, rest = rest, ""
			}
		}
		origin := Origin{Line: i + 1}
		origin.Path, origin.Anchor = splitAnchor(token)
		if g := originIDGroup.FindStringSubmatch(rest); g != nil {
			origin.OriginalID = strings.TrimSpace(g[1])
		}
		out = append(out, origin)
	}
	return out
}
