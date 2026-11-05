package gga

import (
	"regexp"
	"strings"
)

var configAssignment = regexp.MustCompile(`^(?:export[ \t]+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
var bareLiteral = regexp.MustCompile(`^[A-Za-z0-9_./:@%+,=-]+$`)

// existingProvider reconoce solo asignaciones literales independientes y
// comentarios. No es un parser de shell: ante sintaxis dudosa devuelve
// safe=false y el llamador conserva el fichero entero. La palabra se devuelve
// tal cual, sin decodificar ni evaluar. word vacío con safe=true significa que
// el config no define ningún proveedor.
func existingProvider(content []byte) (word string, safe bool) {
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		// Solo el espacio y el tabulador ASCII delimitan palabras de shell.
		line = strings.Trim(line, " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		match := configAssignment.FindStringSubmatch(line)
		if match == nil {
			return "", false
		}
		literal, ok := literalWord(match[2])
		if !ok || seen[match[1]] {
			return "", false
		}
		seen[match[1]] = true
		if match[1] == "PROVIDER" {
			if literal == "''" || literal == `""` {
				return "", false
			}
			word = literal
		}
	}
	return word, true
}

func literalWord(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return "", false
	}
	if value[0] == '\'' || value[0] == '"' {
		end := strings.IndexByte(value[1:], value[0])
		if end < 0 {
			return "", false
		}
		end++
		tail := value[end+1:]
		if tail != "" {
			if tail[0] != ' ' && tail[0] != '\t' {
				return "", false
			}
			tail = strings.Trim(tail, " \t")
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return "", false
			}
		}
		if value[0] == '"' && strings.ContainsAny(value[1:end], "$`\\") {
			return "", false
		}
		return value[:end+1], true
	}
	// Una palabra sin comillas no admite metacaracteres, expansiones ni escapes.
	end := strings.IndexAny(value, " \t")
	if end < 0 {
		end = len(value)
	}
	word := value[:end]
	if !bareLiteral.MatchString(word) {
		return "", false
	}
	tail := strings.Trim(value[end:], " \t")
	if tail != "" && !strings.HasPrefix(tail, "#") {
		return "", false
	}
	return word, true
}
