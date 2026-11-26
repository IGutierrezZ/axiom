package openspecdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// RF-ARC-11 y RF-RID-04: no regresión sobre el corpus de las specs vivas. Para cada
// requisito de cada spec de openspec/specs/ (lectura, nunca escritura):
//   - Resolve lo encuentra a sí mismo con el patrón por defecto, y
//   - un MODIFIED identidad (la delta repite el bloque tal cual) compone la spec
//     byte a byte.
func TestComposeCorpusIdentityModified(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "openspec", "specs", "*", "spec.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no hay specs vivas: %v %v", files, err)
	}
	p := reqid.Default()
	checked := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		doc := parseSpecDocument(text)
		var names []string
		for _, s := range doc.Segments {
			if s.IsRequirement {
				names = append(names, s.Name)
			}
		}
		domain := filepath.Base(filepath.Dir(file))
		k := 0
		for _, s := range doc.Segments {
			if !s.IsRequirement {
				continue
			}
			idx, match := Resolve(names, s.Name, p)
			if match == MatchNone {
				t.Errorf("%s: Resolve no encuentra %q", domain, s.Name)
			} else if match == MatchFound && names[idx] != s.Name && !reqid.IsSuperseded(names[idx]) && p.InTitle(names[idx]) != p.InTitle(s.Name) {
				t.Errorf("%s: Resolve(%q) devolvió %q", domain, s.Name, names[idx])
			}
			k++

			// El bloque debe sobrevivir al análisis de deltas (que ignora lo que hay dentro de vallas).
			block := parseDeltaRequirementBlocks(s.Text, fencedRanges(s.Text), 0)
			if len(block) != 1 || block[0].Text != s.Text {
				continue
			}
			delta := "## MODIFIED Requirements\n\n" + s.Text
			// Un nombre repetido o con identificador repetido pasa por Resolve igual que hoy;
			// aquí solo interesa la identidad, así que se salta lo ambiguo.
			if _, m := Resolve(names, s.Name, p); m != MatchFound {
				continue
			}
			out, err := Compose(text, delta, ComposeOptions{IDPattern: p})
			if err != nil {
				t.Errorf("%s: MODIFIED identidad de %q: %v", domain, s.Name, err)
				continue
			}
			if out != ensureTrailingNewline(text) && out != text {
				t.Errorf("%s: MODIFIED identidad de %q cambió la spec", domain, s.Name)
			}
			checked++
		}
		if k == 0 {
			t.Errorf("%s: sin requisitos", domain)
		}
	}
	if checked == 0 {
		t.Fatal("ningún MODIFIED identidad comprobado")
	}
}
