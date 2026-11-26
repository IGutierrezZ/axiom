// Package explainer define el contrato del HTML explicativo de spec y design
// (`resumen-spec.html` y `resumen-design.html`): nombres de fichero, lectura
// acotada, lectura del `<meta>` del digest de origen, IDs citados y validación.
//
// Es un paquete hoja: solo usa la biblioteca estándar, de modo que sddstatus,
// el dashboard y los tests de assets lo comparten sin ciclos de importación.
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

// Kinds devuelve los tipos válidos, en este orden.
func Kinds() []Kind { return []Kind{KindSpec, KindDesign} }

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
	digestValue = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	refAttr     = regexp.MustCompile(`(?is)\b` + RefAttribute + `\s*=\s*["']([^"']+)["']`)

	scriptTag   = regexp.MustCompile(`(?i)<script\b`)
	externalURL = regexp.MustCompile(`(?is)\b(?:src|href|xlink:href|action|poster|srcset)\s*=\s*["']?\s*(?:https?:)?//`)
	cssURL      = regexp.MustCompile(`(?i)url\(\s*["']?\s*(?:https?:)?//`)
	cssImport   = regexp.MustCompile(`(?i)@import\b`)
	spaces      = regexp.MustCompile(`\s+`)
)

const approvalNotice = "lo que se aprueba es el md"

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

// CitedIDs devuelve, en orden de aparición y sin repetir, los valores de data-axiom-ref.
func CitedIDs(html []byte) []string {
	var ids []string
	seen := map[string]bool{}
	for _, m := range refAttr.FindAllSubmatch(html, -1) {
		id := strings.TrimSpace(string(m[1]))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// Finding es un incumplimiento del contrato detectado por Validate.
type Finding struct{ Code, Detail string }

// Códigos de Finding.
const (
	CodeScriptTag             = "script_tag"
	CodeExternalURL           = "external_url"
	CodeDigestMetaMissing     = "digest_meta_missing"
	CodeApprovalNoticeMissing = "approval_notice_missing"
	CodeCitedIDsMissing       = "cited_ids_missing"
)

// Validate comprueba el contrato del HTML. No se ejecuta en producción: lo usan los tests.
func Validate(html []byte) []Finding {
	var out []Finding
	if scriptTag.Match(html) {
		out = append(out, Finding{CodeScriptTag, "el HTML contiene una etiqueta <script>"})
	}
	if externalURL.Match(html) || cssURL.Match(html) || cssImport.Match(html) {
		out = append(out, Finding{CodeExternalURL, "el HTML referencia un recurso externo (URL absoluta, // o @import)"})
	}
	if v, ok := SourceDigest(html); !ok || !digestValue.MatchString(v) {
		out = append(out, Finding{CodeDigestMetaMissing, `falta <meta name="` + MetaName + `"> con un valor sha256:<64 hex>`})
	}
	text := strings.ToLower(spaces.ReplaceAllString(string(html), " "))
	if !strings.Contains(text, approvalNotice) {
		out = append(out, Finding{CodeApprovalNoticeMissing, "falta el aviso «lo que se aprueba es el md»"})
	}
	if len(CitedIDs(html)) == 0 {
		out = append(out, Finding{CodeCitedIDsMissing, "ningún bloque cita su ID con " + RefAttribute})
	}
	return out
}
