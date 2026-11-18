package skillcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/skillregistry"
)

func skillBody(name, tag string) string {
	return "---\nname: " + name + "\ndescription: \"Hace algo. Trigger: cuando haga falta.\"\n---\n\nBody " + tag + "\n"
}

func findRule(fs []Finding, rule string) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func TestScanSetSkillRotaYReferenciaRota(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".claude", "skills"), "buena", skillBody("buena", "x"), nil)
	writeSkill(t, filepath.Join(cwd, ".claude", "skills"), "rota", "# sin frontmatter\n", nil)
	writeSkill(t, filepath.Join(cwd, ".claude", "skills"), "refs", skillBody("refs", "x")+"Mira [guía](references/falta.md).\n", nil)

	rep := ScanSet(cwd, home, SetOptions{})
	if len(rep.Skills) != 3 {
		t.Fatalf("skills = %d, want 3", len(rep.Skills))
	}
	if rep.Count(SeverityError) != 1 || !rep.HasErrors() {
		t.Errorf("errores = %d, want 1 (la skill sin frontmatter)", rep.Count(SeverityError))
	}
	var refWarn int
	for _, r := range rep.Skills {
		refWarn += len(findRule(r.Findings, RuleReference))
	}
	if refWarn != 1 {
		t.Errorf("referencias rotas = %d, want 1", refWarn)
	}
}

func TestScanSetColisiones(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".agents", "skills"), "dup", skillBody("dup", "a"), nil)
	writeSkill(t, filepath.Join(cwd, ".claude", "skills"), "dup", skillBody("dup", "b"), nil)
	writeSkill(t, filepath.Join(cwd, ".agents", "skills"), "igual", skillBody("igual", "a"), nil)
	writeSkill(t, filepath.Join(home, ".claude", "skills"), "igual", skillBody("igual", "a"), nil)

	rep := ScanSet(cwd, home, SetOptions{})
	col := findRule(rep.SetFindings, RuleCollision)
	if len(col) != 2 {
		t.Fatalf("colisiones = %+v, want 2", col)
	}
	for _, f := range col {
		switch {
		case strings.Contains(f.Message, `"dup"`):
			if f.Severity != SeverityWarning || !strings.Contains(f.Message, "contenido distinto") {
				t.Errorf("dup: %+v", f)
			}
			// El registro prefiere la copia compartida del proyecto.
			want := filepath.Join(cwd, ".agents", "skills", "dup", "SKILL.md")
			if !strings.Contains(f.Message, "el registro elige "+want) {
				t.Errorf("dup debe indicar la copia elegida %s: %s", want, f.Message)
			}
		case strings.Contains(f.Message, `"igual"`):
			if f.Severity != SeverityInfo {
				t.Errorf("igual: %+v", f)
			}
		}
	}
}

func TestScanSetNombreReservadoEsInformativo(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	writeSkill(t, filepath.Join(cwd, "skills"), "sdd-propia", skillBody("sdd-propia", "x"), nil)

	rep := ScanSet(cwd, home, SetOptions{})
	got := findRule(rep.SetFindings, RuleReservedDrop)
	if len(got) != 1 || got[0].Severity != SeverityInfo {
		t.Fatalf("reserved-dropped = %+v", got)
	}
	if rep.HasErrors() {
		t.Errorf("un nombre reservado no es un error en el escaneo: %+v", rep)
	}
}

func TestScanSetAlcanceYAvisoDePi(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	shared := filepath.Join(cwd, ".agents", "skills")
	orphan := filepath.Join(cwd, "skills")
	piDir := filepath.Join(cwd, ".pi", "skills")
	writeSkill(t, shared, "a", skillBody("a", "x"), nil)
	writeSkill(t, orphan, "b", skillBody("b", "x"), nil)
	writeSkill(t, piDir, "c", skillBody("c", "x"), nil)

	rep := ScanSet(cwd, home, SetOptions{FolderRuntimes: map[string][]string{
		shared: {"opencode", "codex"},
		piDir:  {PiRuntime},
	}})
	scope := findRule(rep.SetFindings, RuleFolderScope)
	if len(scope) != 3 {
		t.Fatalf("folder-scope = %+v, want 3", scope)
	}
	var sawShared, sawOrphan bool
	for _, f := range scope {
		if f.Severity != SeverityInfo {
			t.Errorf("el alcance es informativo: %+v", f)
		}
		if strings.Contains(f.Message, shared) && strings.Contains(f.Message, "codex, opencode") {
			sawShared = true
		}
		if strings.Contains(f.Message, orphan) && strings.Contains(f.Message, "ningún runtime") {
			sawOrphan = true
		}
	}
	if !sawShared || !sawOrphan {
		t.Errorf("alcance inesperado: %+v", scope)
	}
	pi := findRule(rep.SetFindings, RulePiNotice)
	if len(pi) != 1 || pi[0].Severity != SeverityInfo || !strings.Contains(pi[0].Message, "gentle-pi") {
		t.Errorf("pi-notice = %+v", pi)
	}
	if rep.Count(SeverityError)+rep.Count(SeverityWarning) != 0 {
		t.Errorf("Pi y alcance nunca bloquean ni avisan: %+v", rep)
	}

	// Sin carpetas de Pi, solo con PiRelevant.
	rep = ScanSet(t.TempDir(), home, SetOptions{PiRelevant: true})
	if len(findRule(rep.SetFindings, RulePiNotice)) != 1 {
		t.Errorf("PiRelevant debe emitir el aviso: %+v", rep.SetFindings)
	}
}

func TestScanSetFrescuraDelRegistro(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	dir := writeSkill(t, filepath.Join(cwd, "skills"), "demo", skillBody("demo", "x"), nil)

	// Sin registro: informativo.
	rep := ScanSet(cwd, home, SetOptions{})
	got := findRule(rep.SetFindings, RuleRegistryState)
	if len(got) != 1 || got[0].Severity != SeverityInfo {
		t.Fatalf("absent: %+v", got)
	}

	if _, err := skillregistry.Regenerate(cwd, home, skillregistry.RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}
	if got := findRule(ScanSet(cwd, home, SetOptions{}).SetFindings, RuleRegistryState); len(got) != 0 {
		t.Fatalf("fresco no debe dar hallazgo: %+v", got)
	}

	// Una edición deja la caché desfasada: aviso.
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillBody("demo", "cambiado")), 0o644); err != nil {
		t.Fatal(err)
	}
	got = findRule(ScanSet(cwd, home, SetOptions{}).SetFindings, RuleRegistryState)
	if len(got) != 1 || got[0].Severity != SeverityWarning || !strings.Contains(got[0].Message, "axiom skill index refresh") {
		t.Fatalf("stale: %+v", got)
	}
}
