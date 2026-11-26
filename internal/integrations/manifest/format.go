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

// Formatos de la lista cerrada que se añaden para el ciclo de vida de las Tasks.
const (
	// FormatNonEmpty: string no vacía tras recortar.
	FormatNonEmpty ValueFormat = "non_empty"
	// FormatTagPrefix: string de 1 a 100 caracteres, sin ',' ni ';', que no empieza por "axiom:" ni
	// por "axiom-task:" (sin distinguir mayúsculas).
	FormatTagPrefix ValueFormat = "tag_prefix"
	// FormatWeekdaySet: string_list no vacía, sin duplicados, con valores de mon a sun.
	FormatWeekdaySet ValueFormat = "weekday_set"
)

const maxTagPrefixLength = 100

// weekdays son los días admitidos por weekday_set, en orden de semana.
var weekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// reservedTagPrefixes son los prefijos de etiqueta que Axiom se reserva.
var reservedTagPrefixes = []string{"axiom:", "axiom-task:"}

// Formats devuelve la lista cerrada de formatos en orden estable.
func Formats() []ValueFormat {
	return []ValueFormat{FormatTypePrefixPairs, FormatNonEmpty, FormatTagPrefix, FormatWeekdaySet}
}

// ValidFormat indica si f pertenece a la lista cerrada de formatos.
func ValidFormat(f ValueFormat) bool {
	for _, k := range Formats() {
		if k == f {
			return true
		}
	}
	return false
}

// FormatNames devuelve los formatos admitidos separados por coma.
func FormatNames() string {
	names := make([]string, 0, len(Formats()))
	for _, f := range Formats() {
		names = append(names, string(f))
	}
	return strings.Join(names, ", ")
}

// FormatType indica el tipo de campo al que se aplica un formato.
func FormatType(f ValueFormat) FieldType {
	switch f {
	case FormatNonEmpty, FormatTagPrefix:
		return FieldString
	}
	return FieldStringList
}

// CheckNonEmpty comprueba el formato non_empty y devuelve el motivo del fallo, o "".
func CheckNonEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "no puede estar vacío"
	}
	return ""
}

// CheckTagPrefix comprueba el formato tag_prefix y devuelve el motivo del fallo, o "".
func CheckTagPrefix(s string) string {
	switch {
	case s == "":
		return "no puede estar vacío"
	case len([]rune(s)) > maxTagPrefixLength:
		return fmt.Sprintf("supera los %d caracteres", maxTagPrefixLength)
	case strings.ContainsAny(s, ",;"):
		return "no puede contener ',' ni ';'"
	}
	lower := strings.ToLower(s)
	for _, r := range reservedTagPrefixes {
		if strings.HasPrefix(lower, r) {
			return fmt.Sprintf("no puede empezar por %q, prefijo reservado de Axiom", r)
		}
	}
	return ""
}

// ParseWeekdaySet comprueba el formato weekday_set. Devuelve los problemas por elemento y, si la
// lista está vacía, un problema con Index -1.
func ParseWeekdaySet(values []string) []FormatProblem {
	if len(values) == 0 {
		return []FormatProblem{{Index: -1, Reason: "no puede estar vacía"}}
	}
	var problems []FormatProblem
	seen := map[string]bool{}
	for i, v := range values {
		switch {
		case !containsStr(weekdays, v):
			problems = append(problems, FormatProblem{Index: i, Element: v, Reason: "día no admitido; admitidos: " + strings.Join(weekdays, ", ")})
		case seen[v]:
			problems = append(problems, FormatProblem{Index: i, Element: v, Reason: "día repetido"})
		}
		seen[v] = true
	}
	return problems
}

// Weekdays devuelve los días admitidos por weekday_set.
func Weekdays() []string { return append([]string(nil), weekdays...) }

func containsStr(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

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
