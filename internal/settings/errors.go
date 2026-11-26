// Package settings es el dueño del catálogo de claves editables de axiom.yaml y del
// escritor que las modifica sin perder comentarios ni formato (C-06-e). Es un
// paquete hoja respecto del servidor HTTP: el dashboard solo traduce entre HTTP y
// esta API, de modo que otros clientes pueden reutilizarla.
package settings

import "strings"

// Code identifica el motivo tipado de un rechazo del paquete.
type Code string

const (
	// CodeInvalidRequest: petición mal formada (revisión ausente, parche vacío).
	CodeInvalidRequest Code = "invalid_request"
	// CodeConfigMissing: el proyecto no tiene un axiom.yaml efectivo.
	CodeConfigMissing Code = "config_missing"
	// CodeRevisionMismatch: el fichero cambió respecto de la revisión esperada.
	CodeRevisionMismatch Code = "revision_mismatch"
	// CodeUnsupportedDocument: anclas, alias, `<<`, multidocumento o clave repetida.
	CodeUnsupportedDocument Code = "unsupported_document"
	// CodeUnknownKey: clave que no figura en el catálogo.
	CodeUnknownKey Code = "unknown_key"
	// CodeKeyNotEditable: clave de solo lectura del catálogo.
	CodeKeyNotEditable Code = "key_not_editable"
	// CodeKeyNotApplicable: clave que no aplica a este documento (topología, rol
	// o repositorio inexistente).
	CodeKeyNotApplicable Code = "key_not_applicable"
	// CodeInvalidValue: tipo incorrecto, fuera de vocabulario o `null` en una clave
	// obligatoria.
	CodeInvalidValue Code = "invalid_value"
	// CodeValidationFailed: el documento resultante no supera la validación.
	CodeValidationFailed Code = "validation_failed"
	// CodeWriteFailed: fallo de E/S al reemplazar el fichero.
	CodeWriteFailed Code = "write_failed"
)

// Error es el error tipado del paquete. Message va siempre en castellano y
// Violations, cuando existe, lleva una violación por línea.
type Error struct {
	Code       Code
	Message    string
	Violations []string
}

// Error implementa error: el mensaje seguido de sus violaciones, una por línea.
func (e *Error) Error() string {
	if len(e.Violations) == 0 {
		return e.Message
	}
	return e.Message + ":\n" + strings.Join(e.Violations, "\n")
}

func newError(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}
