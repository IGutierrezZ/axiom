package assets

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

// TestSkillFrontmatterIsLintClean valida cada SKILL.md embebido con el perfil
// canónico de skillcheck, la única fuente de reglas de skills. Los errores
// fallan el test; los avisos (referencias rotas) solo se registran.
//
// Dos excepciones deliberadas:
//   - systemic-issue-triage se copia byte a byte de su origen público, con un
//     texto de trigger más largo: presupuesto de 205 runas.
//   - Los assets contienen skills `sdd-*` y `skill-registry`, que el registro
//     descarta en silencio en otros contextos: no se aplica la regla de nombres
//     reservados.
func TestSkillFrontmatterIsLintClean(t *testing.T) {
	for _, p := range embeddedSkillPaths(t) {
		t.Run(p, func(t *testing.T) {
			dir := path.Dir(p)
			opts := skillcheck.Options{
				Profile:          skillcheck.Canonical,
				SkipReservedName: true,
				Exists: func(rel string) bool {
					_, err := fs.Stat(FS, path.Join(dir, rel))
					return err == nil
				},
			}
			if p == "skills/systemic-issue-triage/SKILL.md" {
				opts.DescriptionBudget = 205
			}
			for _, f := range skillcheck.CheckContent(MustRead(p), path.Base(dir), opts) {
				switch f.Severity {
				case skillcheck.SeverityError:
					t.Errorf("%s: %s", f.Rule, f.Message)
				default:
					t.Logf("%s (%s): %s", f.Rule, f.Severity, f.Message)
				}
			}
		})
	}
}

func embeddedSkillPaths(t *testing.T) []string {
	t.Helper()

	var paths []string
	if err := fs.WalkDir(FS, "skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, "/SKILL.md") {
			return nil
		}
		paths = append(paths, p)
		return nil
	}); err != nil {
		t.Fatalf("walk embedded skills: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("no embedded SKILL.md files found")
	}
	sort.Strings(paths)
	return paths
}
