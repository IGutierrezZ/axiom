package specbaseline

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// byteVariant es una de las tres formas de bytes en las que se prueba cada
// documento (REQ-24.45): las variantes CRLF y CRLF con BOM se generan a partir
// de la versión LF, de modo que las pruebas no dependen de .gitattributes.
type byteVariant struct {
	name string
	bom  bool
	crlf bool
}

var byteVariants = []byteVariant{
	{name: "LF"},
	{name: "CRLF", crlf: true},
	{name: "CRLF+BOM", crlf: true, bom: true},
}

// apply convierte un texto en LF a la variante de bytes. Un texto que ya lleva
// BOM lo conserva en todas las variantes y no recibe otro.
func (v byteVariant) apply(lf string) []byte {
	data := []byte(lf)
	if v.crlf {
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	}
	if v.bom && !bytes.HasPrefix(data, utf8BOM) {
		data = append(append([]byte{}, utf8BOM...), data...)
	}
	return data
}

// eol es el final de línea de la variante.
func (v byteVariant) eol() string {
	if v.crlf {
		return "\r\n"
	}
	return "\n"
}

// forEachVariant ejecuta fn como subprueba por cada variante de bytes.
func forEachVariant(t *testing.T, fn func(t *testing.T, v byteVariant)) {
	t.Helper()
	for _, v := range byteVariants {
		t.Run(v.name, func(t *testing.T) { fn(t, v) })
	}
}

// readFixture lee un fixture de testdata/reference como texto.
func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "reference", filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("leer el fixture %s: %v", name, err)
	}
	return string(data)
}

// requirementProfile es el perfil de `RF`/`NFR` de los fixtures: nivel 3, id y
// plantilla `### {id}: {title}`.
func requirementProfile(t *testing.T) Profile {
	t.Helper()
	return profileFor(t, `^#+ (?P<id>(?:RF|NFR)-\d+): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosInclude)
}

func profileFor(t *testing.T, heading string, level int, identity workspace.IdentityKind, render string, scenarios workspace.ScenariosMode) Profile {
	t.Helper()
	return Profile{
		Heading:   regexp.MustCompile(heading),
		Level:     level,
		Identity:  identity,
		Render:    render,
		Scenarios: scenarios,
	}
}
