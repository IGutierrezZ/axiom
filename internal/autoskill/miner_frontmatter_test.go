package autoskill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

// Las reglas de frontmatter son las de skillcheck (perfil canónico): una sola
// fuente con el test de assets embebidos y las puertas de entrada.

func TestMinerFrontmatterEsValido(t *testing.T) {
	repo := t.TempDir()

	escribir := func(rel, contenido string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(contenido), 0o644); err != nil {
			t.Fatalf("escribir %s: %v", rel, err)
		}
	}

	// Dispara pattern-table-tests.
	escribir("pkg/a_test.go", "package pkg\n\nfunc TestA(t *testing.T) {\n\ttests := []struct{ name string }{}\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {})\n\t}\n}\n")
	// Dispara pattern-error-wrapping (al menos 2 ficheros .go).
	escribir("pkg/b.go", "package pkg\n\nvar _ = fmt.Errorf(\"x: %w\", err)\n")
	escribir("pkg/c.go", "package pkg\n\nvar _ = fmt.Errorf(\"y: %w\", err)\n")
	// Dispara pattern-internal-layering (>=2 subdirectorios en internal/).
	escribir("internal/uno/uno.go", "package uno\n")
	escribir("internal/dos/dos.go", "package dos\n")

	proposals, err := NewMiner().MinePatterns(repo)
	if err != nil {
		t.Fatalf("MinePatterns() falló: %v", err)
	}

	porPatron := map[string]MiningProposal{}
	for _, p := range proposals {
		porPatron[p.PatternID] = p
	}
	patrones := []string{"pattern-table-tests", "pattern-internal-layering", "pattern-error-wrapping"}
	for _, id := range patrones {
		if _, ok := porPatron[id]; !ok {
			t.Fatalf("el minero no propuso %s; propuestas: %d", id, len(proposals))
		}
	}

	for _, id := range patrones {
		p := porPatron[id]
		t.Run(id, func(t *testing.T) {
			// El gestor deposita la skill en .axiom/skills/inbox/<Name>/SKILL.md.
			for _, f := range skillcheck.CheckContent(p.SkillMD, p.Name, skillcheck.Options{Profile: skillcheck.Canonical}) {
				if f.Severity == skillcheck.SeverityError || f.Severity == skillcheck.SeverityWarning {
					t.Errorf("%s (%s): %s", f.Rule, f.Severity, f.Message)
				}
			}
			if !strings.Contains(p.SkillMD, "\n  origin: mined\n") {
				t.Errorf("metadata debe incluir `origin: mined` indentado bajo `metadata:`")
			}
			if !strings.Contains(p.SkillMD, "\n  trigger:") {
				t.Errorf("metadata debe incluir `trigger:` indentado bajo `metadata:`")
			}
		})
	}
}
