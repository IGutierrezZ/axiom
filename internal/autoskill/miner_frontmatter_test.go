package autoskill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Las reglas de este test replican TestSkillFrontmatterIsLintClean
// (internal/assets/skills_frontmatter_test.go) hasta que exista skillcheck.

// frontmatterMinado es el resultado del parser mínimo por líneas.
type frontmatterMinado struct {
	claves      []string // claves de primer nivel, en orden
	name        string
	descRaw     string // valor crudo tras "description:"
	metadataLns []string
}

// extraerFrontmatter parsea el bloque inicial delimitado por "---".
func extraerFrontmatter(t *testing.T, md string) frontmatterMinado {
	t.Helper()
	if !strings.HasPrefix(md, "---\n") {
		t.Fatalf("el SKILL.md no empieza con ---: %.40q", md)
	}
	resto := md[len("---\n"):]
	fin := strings.Index(resto, "\n---")
	if fin < 0 {
		t.Fatalf("falta el bloque de cierre ---")
	}

	var fm frontmatterMinado
	enMetadata := false
	for _, ln := range strings.Split(resto[:fin], "\n") {
		if ln == "" {
			continue
		}
		if ln[0] == ' ' || ln[0] == '\t' {
			if enMetadata {
				fm.metadataLns = append(fm.metadataLns, strings.TrimSpace(ln))
			}
			continue
		}
		enMetadata = false
		clave, valor, ok := strings.Cut(ln, ":")
		if !ok {
			t.Fatalf("línea de frontmatter sin clave: %q", ln)
		}
		fm.claves = append(fm.claves, clave)
		valor = strings.TrimSpace(valor)
		switch clave {
		case "name":
			fm.name = valor
		case "description":
			fm.descRaw = valor
		case "metadata":
			enMetadata = true
		}
	}
	return fm
}

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

	permitidas := map[string]bool{
		"name": true, "description": true, "license": true, "metadata": true,
		"version": true, "user-invocable": true, "disable-model-invocation": true,
	}

	for _, id := range patrones {
		p := porPatron[id]
		t.Run(id, func(t *testing.T) {
			fm := extraerFrontmatter(t, p.SkillMD)

			// El gestor deposita la skill en .axiom/skills/inbox/<Name>/SKILL.md.
			if fm.name != p.Name {
				t.Errorf("name = %q, want %q (directorio de la skill)", fm.name, p.Name)
			}

			if len(fm.descRaw) < 2 || !strings.HasPrefix(fm.descRaw, `"`) || !strings.HasSuffix(fm.descRaw, `"`) {
				t.Fatalf("description debe ir entrecomillada y en una línea; crudo: %q", fm.descRaw)
			}
			desc := fm.descRaw[1 : len(fm.descRaw)-1]
			if n := len([]rune(desc)); n > 160 {
				t.Errorf("description mide %d runas, want <=160: %q", n, desc)
			}
			if !strings.Contains(desc, "Trigger:") {
				t.Errorf("description debe contener `Trigger:`: %q", desc)
			}

			for _, k := range fm.claves {
				if !permitidas[k] {
					t.Errorf("clave de primer nivel no permitida: %q", k)
				}
			}

			meta := strings.Join(fm.metadataLns, "\n")
			if !strings.Contains(meta, "origin: mined") {
				t.Errorf("metadata debe incluir `origin: mined`; got %q", meta)
			}
			if !strings.Contains(meta, "trigger:") {
				t.Errorf("metadata debe incluir `trigger:`; got %q", meta)
			}
		})
	}
}
