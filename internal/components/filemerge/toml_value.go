package filemerge

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

// Ayudas TOML para los valores de texto largo que Axiom registra en la
// configuración de proyecto de Codex (developer_instructions). Complementan a
// UpsertTopLevelTOMLString, que escribe cadenas cortas con %q (una cadena de Go,
// no de TOML) y cambia de sitio la clave en cada pasada.

// FormatTOMLString devuelve el valor de la derecha de una asignación TOML con value
// como texto. Un valor de una sola línea va entre comillas dobles con los escapes
// de TOML; uno con saltos de línea va como cadena multilínea, que es la forma
// legible de un texto largo:
//
//   - literal (delimitada por tres comillas simples), sin escapes, cuando el texto no
//     contiene tres comillas simples seguidas ni caracteres de control salvo el salto
//     de línea y el tabulador;
//   - básica (delimitada por tres comillas dobles), con escapes, en cualquier otro
//     caso.
//
// En ambas, TOML descarta el primer salto de línea tras el delimitador de apertura,
// así que el texto que se vuelve a leer es exactamente value. Los saltos CRLF se
// normalizan a LF.
func FormatTOMLString(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	if !strings.Contains(value, "\n") {
		return formatTOMLBasicString(value, false)
	}
	if tomlLiteralMultilineSafe(value) {
		return "'''\n" + value + "'''"
	}
	return "\"\"\"\n" + formatTOMLBasicString(value, true) + "\"\"\""
}

// tomlLiteralMultilineSafe indica si value cabe tal cual en una cadena multilínea
// literal: UTF-8 válido, sin tres comillas simples seguidas y sin caracteres de
// control distintos de \n y \t.
// Una comilla simple final se rechaza porque se pegaría al delimitador de cierre;
// en ese caso se usa la forma básica.
func tomlLiteralMultilineSafe(value string) bool {
	if !utf8.ValidString(value) || strings.Contains(value, "'''") || strings.HasSuffix(value, "'") {
		return false
	}
	for _, r := range value {
		if r == '\n' || r == '\t' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// formatTOMLBasicString escapa value para una cadena básica de TOML. En una
// cadena multilínea (multiline) conserva los saltos de línea y escapa cualquier
// comilla doble que completaría tres seguidas, que cerrarían la cadena.
func formatTOMLBasicString(value string, multiline bool) string {
	var out strings.Builder
	if !multiline {
		out.WriteByte('"')
	}
	quotes := 0
	rawTrailingQuote := false
	for _, r := range value {
		rawTrailingQuote = false
		if r == '"' {
			quotes++
			if multiline && quotes == 3 {
				out.WriteString(`\"`)
				quotes = 0
				continue
			}
			if !multiline {
				out.WriteString(`\"`)
				continue
			}
			out.WriteByte('"')
			rawTrailingQuote = true
			continue
		}
		quotes = 0
		switch {
		case r == '\\':
			out.WriteString(`\\`)
		case r == '\n' && multiline:
			out.WriteByte('\n')
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\t':
			out.WriteString(`\t`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == utf8.RuneError || r < 0x20 || r == 0x7f:
			fmt.Fprintf(&out, `\u%04X`, r)
		default:
			out.WriteRune(r)
		}
	}
	if !multiline {
		out.WriteByte('"')
		return out.String()
	}
	encoded := out.String()
	if rawTrailingQuote {
		// Una comilla pegada al delimitador de cierre se leería como parte de él.
		encoded = strings.TrimSuffix(encoded, `"`) + `\"`
	}
	return encoded
}

// TopLevelTOMLString lee la clave key del ámbito raíz del documento TOML. declared
// indica si existe y isString si es una cadena: una clave con otro tipo (lista,
// tabla, número) no se puede tratar como texto y el llamante debe dejarla intacta.
// Un documento que no es TOML válido devuelve el error del analizador.
func TopLevelTOMLString(content, key string) (value string, declared, isString bool, err error) {
	var document map[string]any
	if _, err = toml.Decode(content, &document); err != nil {
		return "", false, false, err
	}
	raw, declared := document[key]
	if !declared {
		return "", false, false, nil
	}
	value, isString = raw.(string)
	return value, true, isString, nil
}

// ValidateTOML comprueba que content es un documento TOML válido. Un fichero vacío
// lo es.
func ValidateTOML(content string) error {
	var document map[string]any
	_, err := toml.Decode(content, &document)
	return err
}

// UpsertTopLevelTOMLValue escribe key = rawValue en el ámbito raíz del documento.
// rawValue es el texto ya formateado de la derecha (FormatTOMLString). A diferencia
// de UpsertTopLevelTOMLString, una clave que ya existe se sustituye en su sitio,
// también cuando su valor ocupa varias líneas, de modo que repetir la operación no
// cambia el orden de las claves ni un solo byte. Si la clave no existe se inserta
// antes de la primera tabla, que es donde TOML sigue considerándola de la raíz.
// El resto del documento se conserva tal cual.
func UpsertTopLevelTOMLValue(content, key, rawValue string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	assignment := strings.Split(key+" = "+rawValue, "\n")

	var out []string
	if start, end, found := topLevelTOMLAssignmentSpan(lines, key); found {
		out = append(out, lines[:start]...)
		out = append(out, assignment...)
		out = append(out, lines[end+1:]...)
	} else {
		insertAt := firstTOMLTableLine(lines)
		if insertAt == len(lines) {
			// Sin tablas se añade al final, sin dejar líneas en blanco de más.
			for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
				insertAt--
			}
			lines = lines[:insertAt]
		}
		out = append(out, lines[:insertAt]...)
		out = append(out, assignment...)
		if insertAt < len(lines) && len(assignment) > 1 {
			// Un valor multilínea pegado a la tabla siguiente se lee mal.
			out = append(out, "")
		}
		out = append(out, lines[insertAt:]...)
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}

// RemoveTopLevelTOMLKey quita la asignación de key del ámbito raíz del documento,
// aunque su valor ocupe varias líneas, y deja el resto como estaba. Si la clave no
// existe devuelve el documento sin cambios.
func RemoveTopLevelTOMLKey(content, key string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	start, end, found := topLevelTOMLAssignmentSpan(lines, key)
	if !found {
		return content
	}
	out := append(append([]string{}, lines[:start]...), lines[end+1:]...)
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}

// topLevelTOMLAssignmentSpan devuelve las líneas, de start a end ambas incluidas,
// que ocupa la asignación de key en el ámbito raíz. Sigue las cadenas multilínea y
// las listas para no confundir una línea de su contenido con una clave o una tabla.
func topLevelTOMLAssignmentSpan(lines []string, key string) (start, end int, found bool) {
	arrayDepth := 0
	var multilineQuote byte
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if arrayDepth == 0 && multilineQuote == 0 {
			if isTOMLTableHeader(trimmed) {
				return 0, 0, false
			}
			if isTOMLKeyAssignment(trimmed, key) {
				depth := 0
				quote := updateTOMLMultilineString(line, 0, &depth)
				end = i
				for (quote != 0 || depth != 0) && end+1 < len(lines) {
					end++
					quote = updateTOMLMultilineString(lines[end], quote, &depth)
				}
				return i, end, true
			}
		}
		multilineQuote = updateTOMLMultilineString(line, multilineQuote, &arrayDepth)
	}
	return 0, 0, false
}

// firstTOMLTableLine devuelve el índice de la primera cabecera de tabla del
// documento, o len(lines) si no hay ninguna.
func firstTOMLTableLine(lines []string) int {
	arrayDepth := 0
	var multilineQuote byte
	for i, line := range lines {
		if arrayDepth == 0 && multilineQuote == 0 && isTOMLTableHeader(strings.TrimSpace(line)) {
			return i
		}
		multilineQuote = updateTOMLMultilineString(line, multilineQuote, &arrayDepth)
	}
	return len(lines)
}
