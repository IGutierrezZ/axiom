package reqtrace

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// CodeIDInvalid es el código estable del error de un ID que no cumple el patrón.
const CodeIDInvalid = "id_invalid"

// Query es el ID consultado, ya validado.
type Query struct {
	ID          string
	Provisional bool
}

// IDError es el error tipado de un ID que no es válido ni provisional válido bajo el patrón.
type IDError struct {
	ID       string
	Template string
	Code     string
	// Cause es el *reqid.ProvisionalError de la lectura como provisional.
	Cause error
}

func (e *IDError) Error() string {
	return fmt.Sprintf("el identificador %q no es válido para el patrón %s ni un provisional válido (%v)", e.ID, e.Template, e.Cause)
}

// Unwrap devuelve la causa de la lectura como provisional.
func (e *IDError) Unwrap() error { return e.Cause }

// ParseID valida raw contra el patrón del proyecto: un ID válido, un provisional válido o
// *IDError. El ID se compara tal cual, sin normalizar mayúsculas, espacios ni ceros.
func ParseID(raw string, p reqid.Pattern) (Query, error) {
	if p.Valid(raw) {
		return Query{ID: raw}, nil
	}
	prov, err := p.ParseProvisional(raw)
	if err != nil {
		return Query{}, &IDError{ID: raw, Template: p.Template(), Code: CodeIDInvalid, Cause: err}
	}
	return Query{ID: prov.String(), Provisional: true}, nil
}
