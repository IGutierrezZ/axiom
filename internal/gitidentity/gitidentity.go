// Package gitidentity resuelve la identidad git del operador (user.name y
// user.email) para registrarla como actor de los artefactos versionados
// (capacidad change-actor-identity). Es un paquete hoja: solo depende de la
// biblioteca estándar y de internal/gitcommon, que es el único punto de
// ejecución de git del producto.
package gitidentity

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

const (
	// KeyName y KeyEmail son las claves de configuración de git que se leen.
	KeyName  = "user.name"
	KeyEmail = "user.email"

	// MaxPartRunes es el tope de cada parte tras el saneado: el actor
	// "Nombre <email>" nunca supera 2*120+3 = 243 runes (< 256).
	MaxPartRunes = 120

	// resolveTimeout acota cada lectura: la identidad es opcional y no debe
	// retener el registro de una compuerta ni un sellado.
	resolveTimeout = 5 * time.Second
)

// Identity es la identidad git saneada. Una parte vacía está ausente.
type Identity struct {
	Name  string
	Email string
}

// New sanea name y email con Sanitize. Pura.
func New(name, email string) Identity {
	return Identity{Name: Sanitize(name), Email: Sanitize(email)}
}

// Complete dice si las dos partes están presentes.
func (i Identity) Complete() bool {
	return i.Name != "" && i.Email != ""
}

// Actor devuelve "Nombre <email>" y true si la identidad está completa; ""
// y false si no. Una identidad parcial se trata como ausente.
func (i Identity) Actor() (string, bool) {
	if !i.Complete() {
		return "", false
	}
	return i.Name + " <" + i.Email + ">", true
}

// Missing devuelve las claves ausentes en orden fijo (KeyName, KeyEmail).
func (i Identity) Missing() []string {
	var missing []string
	if i.Name == "" {
		missing = append(missing, KeyName)
	}
	if i.Email == "" {
		missing = append(missing, KeyEmail)
	}
	return missing
}

var angleBrackets = strings.NewReplacer("<", "", ">", "")

// Sanitize sanea una parte de la identidad; "" significa ausente. Pura.
//
// Pasos, en este orden: recorte; rechazo de UTF-8 no válido y de cualquier
// carácter de control (también CR, LF, TAB y ESC); retirada de '<' y '>';
// recorte; truncado a MaxPartRunes runes sin dejar espacio final.
func Sanitize(value string) string {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) {
		return ""
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ""
		}
	}
	value = strings.TrimSpace(angleBrackets.Replace(value))
	if utf8.RuneCountInString(value) > MaxPartRunes {
		runes := []rune(value)
		value = string(runes[:MaxPartRunes])
	}
	return strings.TrimRightFunc(value, unicode.IsSpace)
}

// Resolve lee user.name y user.email de la configuración de git desde dir
// (absoluto y existente) y devuelve New(name, email). Nunca falla: cualquier
// error, código de salida distinto de cero o salida vacía deja la clave
// ausente. No lee GIT_AUTHOR_* ni GIT_COMMITTER_*.
func Resolve(ctx context.Context, dir string) Identity {
	return New(readKey(ctx, dir, KeyName), readKey(ctx, dir, KeyEmail))
}

func readKey(ctx context.Context, dir, key string) string {
	result, err := gitcommon.RunWithTimeout(ctx, dir, resolveTimeout, "config", "--get", key)
	if err != nil || result.ExitCode != 0 {
		return ""
	}
	return string(result.Stdout)
}
