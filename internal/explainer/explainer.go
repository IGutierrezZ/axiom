// Package explainer define el contrato del HTML explicativo de spec y design
// (`resumen-spec.html` y `resumen-design.html`): nombres de fichero, lectura
// acotada y lectura del `<meta>` del digest de origen.
//
// Es un paquete hoja: solo usa la biblioteca estándar, de modo que sddstatus y
// el dashboard lo comparten sin ciclos de importación. La validación del
// contrato (Validate, CitedIDs) no se ejecuta en producción y vive en
// contract_test.go, que también contrasta el esqueleto de la doctrina.
package explainer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Kind es el tipo de documento resumido.
type Kind string

const (
	KindSpec   Kind = "spec"
	KindDesign Kind = "design"
)

// State es la frescura del HTML respecto del md de origen.
type State string

const (
	StateFresh   State = "fresh"
	StateStale   State = "stale"
	StateMissing State = "missing"
)

const (
	// MetaName es el nombre del <meta> que lleva el digest del md de origen.
	MetaName = "axiom-source-digest"
	// DigestPrefix precede al SHA-256 hexadecimal del digest.
	DigestPrefix = "sha256:"
	// RefAttribute es el atributo con el que cada bloque cita el ID de su requisito.
	RefAttribute = "data-axiom-ref"
	// MaxBytes es el límite común del endpoint y de la frescura.
	MaxBytes = 5 << 20
)

var (
	// ErrTooLarge indica que el fichero supera MaxBytes; nunca se trunca.
	ErrTooLarge = errors.New("el resumen supera el límite de 5 MiB")
	// ErrNotRegular indica que la ruta no es un fichero regular (directorio o enlace simbólico).
	ErrNotRegular = errors.New("el resumen no es un fichero regular")
)

// ValidKind indica si k es uno de los tipos admitidos.
func ValidKind(k Kind) bool { return k == KindSpec || k == KindDesign }

// FileName devuelve el nombre del HTML de k, o "" si k no es válido.
func FileName(k Kind) string {
	switch k {
	case KindSpec:
		return "resumen-spec.html"
	case KindDesign:
		return "resumen-design.html"
	}
	return ""
}

// FormatDigest antepone DigestPrefix al SHA-256 hexadecimal.
func FormatDigest(hexDigest string) string { return DigestPrefix + hexDigest }

// ReadFile lee el HTML exigiendo un fichero regular (un enlace simbólico da
// ErrNotRegular). Lee como máximo MaxBytes+1 bytes y, si se supera el límite,
// devuelve ErrTooLarge sin contenido. Un fichero inexistente devuelve un error
// que cumple errors.Is(err, fs.ErrNotExist).
func ReadFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotRegular
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leer el resumen: %w", err)
	}
	if len(data) > MaxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

var (
	metaTag     = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	metaName    = regexp.MustCompile(`(?is)\bname\s*=\s*["']` + MetaName + `["']`)
	metaContent = regexp.MustCompile(`(?is)\bcontent\s*=\s*["']([^"']*)["']`)
)

// SourceDigest devuelve el content del primer <meta name="axiom-source-digest">,
// recortado y en minúsculas. El booleano es falso si no hay ninguna etiqueta con
// ese nombre. Un content ausente o vacío devuelve ("", true).
func SourceDigest(html []byte) (string, bool) {
	for _, tag := range metaTag.FindAll(html, -1) {
		if !metaName.Match(tag) {
			continue
		}
		m := metaContent.FindSubmatch(tag)
		if m == nil {
			return "", true
		}
		return strings.ToLower(strings.TrimSpace(string(m[1]))), true
	}
	return "", false
}
