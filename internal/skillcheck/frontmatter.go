package skillcheck

import (
	"strconv"
	"strings"
)

// frontmatter es el resultado del parser mínimo por líneas. No usa una
// dependencia YAML: las reglas solo necesitan inspección a nivel de línea.
type frontmatter struct {
	name        string
	description string // representación lógica, en una línea
	descRaw     string // valor crudo tras `description:`
	descPresent bool
	descBlock   bool // `>` o `|`
	descMulti   bool // continuación indentada de un escalar plano
	keys        []string
	metadata    map[string]string // hijos `clave: valor` de `metadata:`
	body        string            // contenido tras el cierre del frontmatter
}

// parseFrontmatter lee el bloque inicial delimitado por `---`. Tolera BOM,
// CRLF y los comentarios `<!-- section:model-... -->` que preceden al bloque en
// los assets embebidos. Devuelve los hallazgos estructurales que encuentre.
func parseFrontmatter(content string) (frontmatter, []Finding) {
	var fm frontmatter
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	for {
		trimmed := strings.TrimSpace(content)
		if !strings.HasPrefix(trimmed, "<!-- section:model-") {
			break
		}
		idx := strings.Index(content, "-->")
		if idx == -1 {
			break
		}
		content = strings.TrimLeft(content[idx+3:], "\n")
	}

	if !strings.HasPrefix(content, "---\n") {
		return fm, []Finding{{Rule: RuleFrontmatterStart, Severity: SeverityError,
			Message: "el fichero no empieza con `---` (falta el frontmatter)"}}
	}
	lines := strings.Split(content[len("---\n"):], "\n")
	closeAt := -1
	for i, ln := range lines {
		if strings.TrimRight(ln, " \t") == "---" {
			closeAt = i
			break
		}
	}
	if closeAt == -1 {
		return fm, []Finding{{Rule: RuleFrontmatterClose, Severity: SeverityError,
			Message: "falta el `---` de cierre del frontmatter"}}
	}
	fm.body = strings.Join(lines[closeAt+1:], "\n")
	block := lines[:closeAt]

	var findings []Finding
	current := ""
	var descParts []string
	for _, line := range block {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		if indented {
			child := strings.TrimSpace(line)
			switch current {
			case "description":
				descParts = append(descParts, child)
				if !fm.descBlock {
					fm.descMulti = true
				}
			case "metadata":
				if k, v, ok := strings.Cut(child, ":"); ok {
					if fm.metadata == nil {
						fm.metadata = map[string]string{}
					}
					fm.metadata[strings.TrimSpace(k)] = unquote(strings.TrimSpace(v))
				}
			}
			continue
		}
		if line[0] == '#' {
			continue
		}
		// Una secuencia YAML sin sangrar bajo una clave (`tags:` seguida de `- a`)
		// es válida: sus elementos pertenecen a la clave anterior.
		if current != "" && (line == "-" || strings.HasPrefix(line, "- ")) {
			continue
		}
		key, rawValue, ok := strings.Cut(line, ":")
		if !ok {
			findings = append(findings, Finding{Rule: RuleFrontmatterLine, Severity: SeverityError,
				Message: "línea de frontmatter sin `:`: " + strconv.Quote(line)})
			current = ""
			continue
		}
		key = strings.TrimSpace(key)
		value := strings.TrimSpace(rawValue)
		current = key
		fm.keys = append(fm.keys, key)
		switch key {
		case "name":
			fm.name = unquote(value)
		case "description":
			fm.descPresent = true
			fm.descRaw = value
			fm.descBlock = strings.HasPrefix(value, ">") || strings.HasPrefix(value, "|")
			if !fm.descBlock {
				fm.description = unquote(value)
			}
		}
	}
	if len(descParts) > 0 {
		joined := strings.Join(descParts, " ")
		if fm.descBlock || fm.description == "" {
			fm.description = joined
		} else {
			fm.description += " " + joined
		}
	}
	return fm, findings
}

// unquote quita las comillas simples o dobles que rodean un escalar y resuelve
// sus escapes (las skills minadas usan strconv.Quote).
func unquote(s string) string {
	if !isQuoted(s) {
		return s
	}
	if s[0] == '"' {
		if v, err := strconv.Unquote(s); err == nil {
			return v
		}
		return s[1 : len(s)-1]
	}
	return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
}

func isQuoted(s string) bool {
	if len(s) < 2 {
		return false
	}
	q := s[0]
	if q != '"' && q != '\'' {
		return false
	}
	return s[len(s)-1] == q
}
