package changestart

import (
	"errors"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// maxChangeNameLen es el tope del nombre completo del cambio (D-08): deja margen bajo MAX_PATH de
// Windows para el adjunto más profundo sin depender de core.longpaths.
const maxChangeNameLen = 48

// reservedChangeName es el nombre de la carpeta de archivo de los cambios SDD.
const reservedChangeName = "archive"

// ValidChangeName comprueba que name sirve como nombre de cambio: un componente de rama válido y
// distinto del nombre reservado archive (D-28). Lo usan la CLI para --change y el escaneo de vínculos
// para el nombre que reclama una etiqueta: un nombre que no lo supera nunca llega a una ruta.
func ValidChangeName(name string) error {
	if err := gitref.ValidBranchComponent(name); err != nil {
		return err
	}
	if strings.EqualFold(name, reservedChangeName) {
		return errors.New("«archive» es un nombre reservado")
	}
	return nil
}

// DeriveName compone <prefijo>-<id>-<slug> (D-08). El slug pliega el título a ASCII, pasa a minúsculas,
// convierte cualquier carácter fuera de [a-z0-9] en un guion, colapsa los guiones y recorta los
// extremos. El nombre completo no supera 48 caracteres: el slug se corta en el último guion que quepa
// (o en seco si no hay ninguno). Con el slug vacío, el nombre es <prefijo>-<id>.
func DeriveName(prefix, id, title string) string {
	head := prefix + "-" + id
	slug := slugify(title)
	room := maxChangeNameLen - len(head) - 1
	if slug == "" || room < 1 {
		return head
	}
	if len(slug) > room {
		cut := slug[:room]
		if slug[room] != '-' {
			if i := strings.LastIndexByte(cut, '-'); i > 0 {
				cut = cut[:i]
			}
		}
		slug = strings.Trim(cut, "-")
	}
	if slug == "" {
		return head
	}
	name := head + "-" + slug
	if ValidChangeName(name) != nil {
		return head
	}
	return name
}

// slugify devuelve el slug de title: solo [a-z0-9] y guiones simples, sin guiones en los extremos.
func slugify(title string) string {
	var b strings.Builder
	dash := true // evita un guion inicial
	for _, r := range foldASCII(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			dash = false
		default:
			if !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}
