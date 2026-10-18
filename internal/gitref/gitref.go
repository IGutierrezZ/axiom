// Package gitref valida nombres de referencia git (ramas y prefijos de rama).
//
// Es una hoja pura: solo importa la biblioteca estándar, no ejecuta nada y no
// lee nada del disco. Existe aparte de cualquier paquete que planifique ramas para
// que `workspace` (bloques `git:` y `repositories:` de axiom.yaml), `kickoff` y
// el propio planificador compartan una única definición de «nombre válido» sin
// formar ciclos de importación (design INC-25, B1 y §4.6).
//
// Los errores describen la regla incumplida sin nombrar el valor ni su origen:
// quien llama los envuelve con la clave de configuración o el argumento de la
// invocación a los que pertenece el valor.
package gitref

import (
	"errors"
	"fmt"
	"strings"
)

// forbiddenChars son los caracteres imprimibles que `git check-ref-format`
// rechaza en cualquier posición de un nombre. El espacio, los caracteres de
// control y la barra invertida se tratan aparte por llevar un motivo propio.
const forbiddenChars = "~^:?*["

// ValidRefName comprueba que name es un nombre de rama aceptable para git, con
// las reglas de `git check-ref-format`:
//
//   - no está vacío ni es `@`;
//   - ningún componente separado por `/` empieza por `.` ni termina en `.lock`;
//   - no contiene `..`, `@{`, `//`, espacios, caracteres de control (ASCII
//     inferior a 0x20 y DEL), `~`, `^`, `:`, `?`, `*`, `[` ni `\`;
//   - no empieza ni termina en `/` y no termina en `.`.
//
// Añade una regla propia de los nombres de rama: no empieza por `-`, como
// rechaza `git check-ref-format --branch` (una rama así se confundiría con una
// opción en cualquier invocación posterior de git).
//
// No exige un `/` ni limita la longitud, y acepta caracteres no ASCII, igual
// que git.
func ValidRefName(name string) error {
	switch {
	case name == "":
		return errors.New("no puede estar vacío")
	case name == "@":
		return errors.New("no puede ser «@»")
	case strings.HasPrefix(name, "-"):
		return errors.New("no puede empezar por «-»: git lo confundiría con una opción")
	case strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/"):
		return errors.New("no puede empezar ni terminar en «/»")
	}

	for i := 0; i < len(name); i++ {
		switch c := name[i]; {
		case c < 0x20 || c == 0x7f:
			return errors.New("no puede contener caracteres de control")
		case c == ' ':
			return errors.New("no puede contener espacios")
		case c == '\\':
			return errors.New("no puede contener la barra invertida «\\»")
		case strings.IndexByte(forbiddenChars, c) >= 0:
			return fmt.Errorf("no puede contener «%c»", c)
		}
	}

	switch {
	case strings.Contains(name, "//"):
		return errors.New("no puede contener «//»: hay un componente vacío")
	case strings.Contains(name, ".."):
		return errors.New("no puede contener «..»")
	case strings.Contains(name, "@{"):
		return errors.New("no puede contener «@{»")
	case strings.HasSuffix(name, "."):
		return errors.New("no puede terminar en «.»")
	}

	for _, component := range strings.Split(name, "/") {
		switch {
		case strings.HasPrefix(component, "."):
			return fmt.Errorf("ningún componente puede empezar por «.» (componente «%s»)", component)
		case strings.HasSuffix(component, ".lock"):
			return fmt.Errorf("ningún componente puede terminar en «.lock» (componente «%s»)", component)
		}
	}
	return nil
}

// ValidPrefix comprueba el prefijo de rama `git.branch_prefix`: una cadena no
// vacía, sin `/` inicial ni final, que además cumple todas las reglas de
// ValidRefName (en particular, sin `..` ni espacios). Admite `/` internas, de
// modo que `equipo/inc` es un prefijo válido.
func ValidPrefix(prefix string) error {
	if prefix == "" {
		return errors.New("no puede estar vacío")
	}
	if strings.HasPrefix(prefix, "/") || strings.HasSuffix(prefix, "/") {
		return errors.New("no puede empezar ni terminar en «/»")
	}
	return ValidRefName(prefix)
}
