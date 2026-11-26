package explainer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// La validación del contrato del HTML no se ejecuta en producción (la neutralizan la CSP y el
// iframe en sandbox del dashboard); vive aquí para que el ratchet de código muerto no la cuente.

var (
	digestValue = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	refAttr     = regexp.MustCompile(`(?is)\b` + RefAttribute + `\s*=\s*["']([^"']+)["']`)

	scriptTag   = regexp.MustCompile(`(?i)<script\b`)
	externalURL = regexp.MustCompile(`(?is)\b(?:src|href|xlink:href|action|poster|srcset)\s*=\s*["']?\s*(?:https?:)?//`)
	cssURL      = regexp.MustCompile(`(?i)url\(\s*["']?\s*(?:https?:)?//`)
	cssImport   = regexp.MustCompile(`(?i)@import\b`)
	spaces      = regexp.MustCompile(`\s+`)
)

const approvalNotice = "lo que se aprueba es el md"

// Kinds devuelve los tipos válidos, en este orden.
func Kinds() []Kind { return []Kind{KindSpec, KindDesign} }

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

// Validate comprueba el contrato del HTML.
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

// fencedBlock devuelve el contenido del primer bloque con la etiqueta dada.
func fencedBlock(t *testing.T, text, label string) string {
	t.Helper()
	open := "```" + label + "\n"
	from := strings.Index(text, open)
	if from < 0 {
		t.Fatalf("no hay bloque ```%s", label)
	}
	rest := text[from+len(open):]
	to := strings.Index(rest, "\n```")
	if to < 0 {
		t.Fatalf("el bloque ```%s no se cierra", label)
	}
	return rest[:to]
}

// El esqueleto que lee el modelo es el que valida el contrato, y cita exactamente los requisitos del md de ejemplo.
func TestExplainerHTMLSkeletonMatchesExample(t *testing.T) {
	shared := assets.MustRead("skills/_shared/sdd-explainer-html.md")
	md := fencedBlock(t, shared, "markdown")
	html := []byte(fencedBlock(t, shared, "html"))

	if findings := Validate(html); len(findings) != 0 {
		t.Fatalf("el esqueleto incumple el contrato: %+v", findings)
	}
	cited := CitedIDs(html)
	if len(cited) == 0 {
		t.Fatal("el esqueleto no cita ningún ID")
	}
	var required []string
	for _, match := range regexp.MustCompile(`(?m)^#{2,4} (RF-[A-Z]+-\d+)`).FindAllStringSubmatch(md, -1) {
		required = append(required, match[1])
	}
	if len(required) == 0 {
		t.Fatal("el md de ejemplo no tiene requisitos con ID")
	}
	inMD := map[string]bool{}
	for _, id := range required {
		inMD[id] = true
	}
	inHTML := map[string]bool{}
	for _, id := range cited {
		inHTML[id] = true
		if !inMD[id] {
			t.Errorf("el esqueleto cita %q, que no existe en el md de ejemplo", id)
		}
	}
	for _, id := range required {
		if !inHTML[id] {
			t.Errorf("el requisito %q del md de ejemplo no está citado en el esqueleto", id)
		}
	}
}
