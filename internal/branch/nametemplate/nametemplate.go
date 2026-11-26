// Package nametemplate implementa las plantillas de nombre de rama
// (`branch_template`): un analizador, un renderizador y las comprobaciones de
// forma, todo puro (sin procesos ni ficheros). Solo depende de internal/gitref
// y de la biblioteca estándar, de modo que `workspace`, `kickoff` y `branch`
// comparten una única gramática.
//
// La gramática es mínima: texto literal y variables entre llaves, `{nombre}`.
// Una llave sin cerrar, una de cierre suelta, unas llaves anidadas o una
// variable desconocida son errores de sintaxis. No hay escape de llaves.
// Render nunca sanea: un valor que no cumple su regla se rechaza con su
// variable.
package nametemplate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// Variable es el nombre de una variable de plantilla.
type Variable string

// Las siete variables admitidas. La de `slug` se llama VarSlug para no chocar
// con la función Slug que calcula su valor.
const (
	VarPrefix  Variable = "prefix"
	VarKind    Variable = "kind"
	VarChange  Variable = "change"
	VarSlug    Variable = "slug"
	VarRole    Variable = "role"
	VarRef     Variable = "ref"
	VarTaskRef Variable = "task_ref"
)

// Variables devuelve las siete variables, en el orden del vocabulario.
func Variables() []Variable {
	return []Variable{VarPrefix, VarKind, VarChange, VarSlug, VarRole, VarRef, VarTaskRef}
}

func isKnown(name string) bool {
	for _, v := range Variables() {
		if string(v) == name {
			return true
		}
	}
	return false
}

// variableList enumera las variables para los mensajes de error.
func variableList() string {
	names := make([]string, 0, 7)
	for _, v := range Variables() {
		names = append(names, "{"+string(v)+"}")
	}
	return strings.Join(names, ", ")
}

// segment es un trozo de la plantilla: texto literal o una variable.
type segment struct {
	literal  string
	variable Variable // vacío si el trozo es literal
}

// Template es una plantilla analizada. El valor cero (IsZero) es «sin
// plantilla».
type Template struct {
	raw      string
	segments []segment
}

// Raw devuelve el texto original de la plantilla.
func (t Template) Raw() string { return t.raw }

// IsZero dice si la plantilla está vacía (no hay plantilla declarada).
func (t Template) IsZero() bool { return t.raw == "" }

// Uses dice si la plantilla usa la variable v.
func (t Template) Uses(v Variable) bool {
	for _, s := range t.segments {
		if s.variable == v {
			return true
		}
	}
	return false
}

// Parse analiza raw. Devuelve *SyntaxError o *UnknownVariableError. Una
// cadena vacía es un error de sintaxis: una plantilla declarada no puede estar
// vacía (la ausencia se expresa con el valor cero de Template).
func Parse(raw string) (Template, error) {
	if raw == "" {
		return Template{}, &SyntaxError{Template: raw, Offset: 0, Reason: "la plantilla no puede estar vacía"}
	}
	var segs []segment
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			segs = append(segs, segment{literal: lit.String()})
			lit.Reset()
		}
	}
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '}':
			return Template{}, &SyntaxError{Template: raw, Offset: i, Reason: "llave de cierre «}» sin abrir"}
		case '{':
			end := -1
			for j := i + 1; j < len(raw); j++ {
				if raw[j] == '{' {
					return Template{}, &SyntaxError{Template: raw, Offset: j, Reason: "llaves anidadas: una «{» dentro de otra"}
				}
				if raw[j] == '}' {
					end = j
					break
				}
			}
			if end < 0 {
				return Template{}, &SyntaxError{Template: raw, Offset: i, Reason: "llave «{» sin cerrar"}
			}
			name := raw[i+1 : end]
			if name == "" {
				return Template{}, &SyntaxError{Template: raw, Offset: i, Reason: "variable vacía «{}»"}
			}
			if !isKnown(name) {
				return Template{}, &UnknownVariableError{Template: raw, Name: name}
			}
			flush()
			segs = append(segs, segment{variable: Variable(name)})
			i = end
		default:
			lit.WriteByte(raw[i])
		}
	}
	flush()
	return Template{raw: raw, segments: segs}, nil
}

// Values asigna un valor a cada variable que se va a sustituir.
type Values map[Variable]string

// checkValue aplica la regla de valor de la variable v.
func checkValue(v Variable, value string) error {
	switch v {
	case VarPrefix:
		return gitref.ValidPrefix(value)
	case VarKind:
		if value != "feature" && value != "fix" {
			return errors.New("debe ser «feature» o «fix»")
		}
		return nil
	default:
		return gitref.ValidBranchComponent(value)
	}
}

// Render sustituye cada variable por su valor y comprueba el nombre
// resultante con gitref.ValidRefName. Nunca sanea. Errores: *MissingValueError,
// *UnsafeValueError e *InvalidNameError.
func (t Template) Render(v Values) (string, error) {
	var out strings.Builder
	for _, s := range t.segments {
		if s.variable == "" {
			out.WriteString(s.literal)
			continue
		}
		value, ok := v[s.variable]
		if !ok || value == "" {
			return "", &MissingValueError{Variable: s.variable}
		}
		if err := checkValue(s.variable, value); err != nil {
			return "", &UnsafeValueError{Variable: s.variable, Value: value, Cause: err}
		}
		out.WriteString(value)
	}
	name := out.String()
	if err := gitref.ValidRefName(name); err != nil {
		return "", &InvalidNameError{Name: name, Cause: err}
	}
	return name, nil
}

// CheckDistinguishing exige que la plantilla use `{change}`, `{slug}`,
// `{ref}` o `{task_ref}`: sin ellas, todos los cambios compartirían un nombre
// constante. Devuelve *ConstantTemplateError.
func CheckDistinguishing(t Template) error {
	for _, v := range []Variable{VarChange, VarSlug, VarRef, VarTaskRef} {
		if t.Uses(v) {
			return nil
		}
	}
	return &ConstantTemplateError{Template: t.raw}
}

// sampleValues son valores seguros con los que se valida la forma al cargar.
var sampleValues = Values{
	VarPrefix:  "feat",
	VarKind:    "feature",
	VarChange:  "sample-change",
	VarSlug:    "sample",
	VarRole:    "role",
	VarRef:     "1",
	VarTaskRef: "1",
}

// CheckSample renderiza la plantilla con valores de muestra seguros y
// comprueba que el nombre cumple gitref.ValidRefName. Detecta formas como
// `-{ref}` o `{prefix}/`, que ningún valor válido puede arreglar.
func CheckSample(t Template) error {
	_, err := t.Render(sampleValues)
	return err
}

// Slug devuelve el nombre del cambio sin su referencia de work item. Solo
// actúa con ref no vacío: quita del principio de change el prefijo
// `^[A-Za-z]+-<ref>-`, sin distinguir mayúsculas. Si el prefijo no casa o al
// quitarlo no queda nada, devuelve change: el slug nunca está vacío.
func Slug(change, ref string) string {
	if ref == "" {
		return change
	}
	re := regexp.MustCompile(`(?i)^[A-Za-z]+-` + regexp.QuoteMeta(ref) + `-`)
	loc := re.FindStringIndex(change)
	if loc == nil {
		return change
	}
	rest := change[loc[1]:]
	if rest == "" {
		return change
	}
	return rest
}

// SyntaxError es un error de sintaxis de la plantilla.
type SyntaxError struct {
	Template string
	Offset   int
	Reason   string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("plantilla de rama %q inválida en la posición %d: %s (variables admitidas: %s)",
		e.Template, e.Offset, e.Reason, variableList())
}

// UnknownVariableError indica una variable que no existe.
type UnknownVariableError struct {
	Template string
	Name     string
}

func (e *UnknownVariableError) Error() string {
	return fmt.Sprintf("plantilla de rama %q: la variable {%s} no existe (variables admitidas: %s)",
		e.Template, e.Name, variableList())
}

// ConstantTemplateError indica una plantilla sin variable distintiva.
type ConstantTemplateError struct {
	Template string
}

func (e *ConstantTemplateError) Error() string {
	return fmt.Sprintf("plantilla de rama %q sin variable distintiva: usa al menos una de {change}, {slug}, {ref} o {task_ref}", e.Template)
}

// MissingValueError indica una variable sin valor al renderizar.
type MissingValueError struct {
	Variable Variable
}

func (e *MissingValueError) Error() string {
	return fmt.Sprintf("la variable {%s} no tiene valor", e.Variable)
}

// UnsafeValueError indica un valor que incumple la regla de su variable.
type UnsafeValueError struct {
	Variable Variable
	Value    string
	Cause    error
}

func (e *UnsafeValueError) Error() string {
	return fmt.Sprintf("el valor %q de la variable {%s} no es válido: %v", e.Value, e.Variable, e.Cause)
}

func (e *UnsafeValueError) Unwrap() error { return e.Cause }

// InvalidNameError indica un nombre de rama resultante que git rechazaría.
type InvalidNameError struct {
	Name  string
	Cause error
}

func (e *InvalidNameError) Error() string {
	return fmt.Sprintf("el nombre de rama %q no es válido: %v", e.Name, e.Cause)
}

func (e *InvalidNameError) Unwrap() error { return e.Cause }
