package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// ValueFormat declara el formato de los elementos de un campo string_list. La lista es cerrada.
type ValueFormat string

// FormatTypePrefixPairs es la lista de pares "Tipo=Prefijo" (por ejemplo "User Story=US").
const FormatTypePrefixPairs ValueFormat = "type_prefix_pairs"

// TypePrefix es un par tipo de work item y prefijo del nombre del cambio.
type TypePrefix struct {
	Type   string
	Prefix string
}

// FormatProblem describe un elemento de una lista que no cumple su formato. Index es la posición
// del elemento (desde 0) y Element su texto tal como se escribió.
type FormatProblem struct {
	Index   int
	Element string
	Reason  string
}

var prefixPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,10}$`)

// ValidFormat indica si f pertenece a la lista cerrada de formatos.
func ValidFormat(f ValueFormat) bool { return f == FormatTypePrefixPairs }

// ParseTypePrefixes interpreta una lista de elementos "Tipo=Prefijo". Cada elemento lleva un solo
// "="; el tipo no puede quedar vacío tras recortarlo; el prefijo cumple ^[A-Za-z0-9]{1,10}$; ni el
// tipo ni el prefijo se repiten, sin distinguir mayúsculas. Los elementos válidos se devuelven en
// su orden aunque otros fallen.
func ParseTypePrefixes(values []string) ([]TypePrefix, []FormatProblem) {
	var out []TypePrefix
	var problems []FormatProblem
	types := map[string]bool{}
	prefixes := map[string]bool{}
	for i, raw := range values {
		fail := func(reason string) {
			problems = append(problems, FormatProblem{Index: i, Element: raw, Reason: reason})
		}
		if strings.Count(raw, "=") != 1 {
			fail("debe tener la forma Tipo=PREFIJO con un único '='")
			continue
		}
		left, right, _ := strings.Cut(raw, "=")
		typ := strings.TrimSpace(left)
		prefix := strings.TrimSpace(right)
		switch {
		case typ == "":
			fail("el tipo no puede estar vacío")
			continue
		case !prefixPattern.MatchString(prefix):
			fail(fmt.Sprintf("el prefijo debe cumplir %s", prefixPattern.String()))
			continue
		case types[strings.ToLower(typ)]:
			fail(fmt.Sprintf("el tipo %q está repetido", typ))
			continue
		case prefixes[strings.ToLower(prefix)]:
			fail(fmt.Sprintf("el prefijo %q está repetido", prefix))
			continue
		}
		types[strings.ToLower(typ)] = true
		prefixes[strings.ToLower(prefix)] = true
		out = append(out, TypePrefix{Type: typ, Prefix: prefix})
	}
	return out, problems
}
