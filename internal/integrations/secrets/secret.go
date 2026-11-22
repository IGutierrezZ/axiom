// Package secrets resuelve y guarda los secretos de los plugins de integración. El valor de un
// secreto vive en un tipo redactado (Secret) que nunca se imprime ni se serializa: solo Reveal
// lo devuelve. Los secretos nunca se escriben en axiom.yaml, en la cola ni en ningún fichero.
package secrets

import (
	"fmt"
	"log/slog"
)

// redacted es el marcador con el que se sustituye el valor en cualquier salida.
const redacted = "[REDACTED]"

// Secret guarda un valor sensible en un campo no exportado. Todas las formas de imprimirlo o
// serializarlo devuelven [REDACTED]; el valor solo sale con Reveal.
type Secret struct {
	value string
}

// NewSecret envuelve un valor sensible.
func NewSecret(value string) Secret { return Secret{value: value} }

// Reveal devuelve el valor. Es la única salida del valor: llamarla es una decisión explícita.
func (s Secret) Reveal() string { return s.value }

// IsZero indica que el secreto no lleva valor.
func (s Secret) IsZero() bool { return s.value == "" }

// String implementa fmt.Stringer.
func (Secret) String() string { return redacted }

// GoString implementa fmt.GoStringer (%#v).
func (Secret) GoString() string { return redacted }

// Format implementa fmt.Formatter y cubre cualquier verbo (%s %v %+v %#v %q %x %d ...).
func (Secret) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(redacted)) }

// MarshalJSON implementa json.Marshaler.
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// MarshalYAML implementa yaml.Marshaler.
func (Secret) MarshalYAML() (any, error) { return redacted, nil }

// MarshalText implementa encoding.TextMarshaler.
func (Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }

// LogValue implementa slog.LogValuer.
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }
