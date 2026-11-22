package skillcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodSkill = "---\nname: demo\ndescription: \"Hace algo útil. Trigger: cuando pidan demo.\"\nlicence: x\n---\n\n# Demo\n"

func rules(fs []Finding, sev Severity) map[string]bool {
	out := map[string]bool{}
	for _, f := range fs {
		if f.Severity == sev {
			out[f.Rule] = true
		}
	}
	return out
}

func TestCheckContentTabla(t *testing.T) {
	canonicalOK := "---\nname: demo\ndescription: \"Hace algo. Trigger: cuando pidan demo.\"\nmetadata:\n  origin: mined\n  trigger: \"x\"\n---\nbody\n"
	tests := []struct {
		name    string
		content string
		opts    Options
		errors  []string
		warns   []string
	}{
		{name: "canonica limpia con metadata", content: canonicalOK},
		{name: "CRLF y comentario de sección", content: "<!-- section:model-x -->\r\n---\r\nname: demo\r\ndescription: \"A. Trigger: b\"\r\n---\r\nx\r\n"},
		{name: "sin frontmatter", content: "# hola\n", errors: []string{RuleFrontmatterStart}},
		{name: "frontmatter sin cierre", content: "---\nname: demo\n", errors: []string{RuleFrontmatterClose}},
		{name: "secuencia sin sangrar portable", content: "---\nname: demo\ndescription: \"A. Trigger: b\"\nallowed-tools:\n- Read\n- Grep\n---\n", opts: Options{Profile: Portable}, warns: []string{RuleKeysAllowed}},
		{name: "guion sin clave previa", content: "---\n- suelto\nname: demo\ndescription: \"A. Trigger: b\"\n---\n", errors: []string{RuleFrontmatterLine}},
		{name: "linea sin dos puntos", content: "---\nname: demo\nbasura\ndescription: \"A. Trigger: b\"\n---\n", errors: []string{RuleFrontmatterLine}},
		{name: "name ausente", content: "---\ndescription: \"A. Trigger: b\"\n---\n", errors: []string{RuleNameMissing}},
		{name: "name distinto del directorio", content: "---\nname: otro\ndescription: \"A. Trigger: b\"\n---\n", errors: []string{RuleNameMismatch}},
		{name: "description ausente", content: "---\nname: demo\n---\n", errors: []string{RuleDescMissing}},
		{name: "description vacia", content: "---\nname: demo\ndescription:\n---\n", errors: []string{RuleDescMissing}},
		{name: "sin comillas canonico", content: "---\nname: demo\ndescription: Hace algo. Trigger: x\n---\n", errors: []string{RuleDescQuoted}},
		{name: "sin comillas portable es aviso", content: "---\nname: demo\ndescription: Hace algo. Trigger: x\n---\n", opts: Options{Profile: Portable}, warns: []string{RuleDescQuoted}},
		{name: "bloque", content: "---\nname: demo\ndescription: >\n  Hace algo.\n  Trigger: x\n---\n", errors: []string{RuleDescBlock}},
		{name: "multilinea plana", content: "---\nname: demo\ndescription: Hace algo\n  Trigger: x\n---\n", errors: []string{RuleDescMultiline}},
		{name: "sin Trigger", content: "---\nname: demo\ndescription: \"Hace algo.\"\n---\n", errors: []string{RuleDescTrigger}},
		{name: "sin Trigger portable", content: "---\nname: demo\ndescription: \"Hace algo.\"\n---\n", opts: Options{Profile: Portable}, warns: []string{RuleDescTrigger}},
		{name: "excede presupuesto", content: "---\nname: demo\ndescription: \"" + strings.Repeat("a", 170) + " Trigger: x\"\n---\n", errors: []string{RuleDescBudget}},
		{name: "presupuesto ampliado", content: "---\nname: demo\ndescription: \"" + strings.Repeat("a", 170) + " Trigger: x\"\n---\n", opts: Options{DescriptionBudget: 205}},
		{name: "clave no permitida", content: "---\nname: demo\ndescription: \"A. Trigger: b\"\nallowed-tools: x\n---\n", errors: []string{RuleKeysAllowed}},
		{name: "clave no permitida portable", content: "---\nname: demo\ndescription: \"A. Trigger: b\"\nallowed-tools: x\n---\n", opts: Options{Profile: Portable}, warns: []string{RuleKeysAllowed}},
		{name: "claves permitidas", content: "---\nname: demo\ndescription: \"A. Trigger: b\"\nlicense: MIT\nversion: 1\nuser-invocable: true\ndisable-model-invocation: false\n---\n"},
		{name: "descripcion con escapes", content: "---\nname: demo\ndescription: \"Dice \\\"hola\\\". Trigger: x\"\n---\n"},
		{name: "descripcion con comillas simples", content: "---\nname: demo\ndescription: 'It''s ok. Trigger: x'\n---\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckContent(tt.content, "demo", tt.opts)
			gotErr, gotWarn := rules(got, SeverityError), rules(got, SeverityWarning)
			if len(gotErr) != len(tt.errors) {
				t.Fatalf("errores = %v, want %v (%+v)", gotErr, tt.errors, got)
			}
			for _, r := range tt.errors {
				if !gotErr[r] {
					t.Errorf("falta error %q en %+v", r, got)
				}
			}
			if len(gotWarn) != len(tt.warns) {
				t.Fatalf("avisos = %v, want %v (%+v)", gotWarn, tt.warns, got)
			}
			for _, r := range tt.warns {
				if !gotWarn[r] {
					t.Errorf("falta aviso %q en %+v", r, got)
				}
			}
		})
	}
}

func TestCheckContentNombreReservado(t *testing.T) {
	for _, dir := range []string{"_shared", "skill-registry", "sdd-apply"} {
		content := "---\nname: " + dir + "\ndescription: \"A. Trigger: b\"\n---\n"
		got := CheckContent(content, dir, Options{})
		if !rules(got, SeverityError)[RuleNameReserved] {
			t.Errorf("%s: se esperaba name-reserved, got %+v", dir, got)
		}
		got = CheckContent(content, dir, Options{SkipReservedName: true})
		if len(got) != 0 {
			t.Errorf("%s: con SkipReservedName no debe haber hallazgos, got %+v", dir, got)
		}
	}
}

func writeSkill(t *testing.T, root, name, content string, extra map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, c := range extra {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCheckDirReferencias(t *testing.T) {
	body := "---\nname: demo\ndescription: \"A. Trigger: b\"\n---\n" +
		"Lee [guía](references/guia.md) y [rota](references/rota.md).\n" +
		"Usa `scripts/run.sh` y `scripts/falta.sh`.\n" +
		"Ignora [web](https://example.com/x), [ancla](#a), [abs](/etc/x), references/<nombre> y `assets/*`.\n" +
		"```\nreferences/en-codigo.md\n```\n"
	dir := writeSkill(t, t.TempDir(), "demo", body, map[string]string{
		"references/guia.md": "x",
		"scripts/run.sh":     "x",
	})
	rep := CheckDir(dir, Options{})
	if rep.HasErrors() {
		t.Fatalf("no debe haber errores: %+v", rep.Findings)
	}
	var refs []string
	for _, f := range rep.Findings {
		if f.Rule == RuleReference && f.Severity == SeverityWarning {
			refs = append(refs, f.Message)
		}
	}
	if len(refs) != 2 {
		t.Fatalf("referencias rotas = %v, want rota.md y falta.sh", refs)
	}
	joined := strings.Join(refs, "|")
	if !strings.Contains(joined, "references/rota.md") || !strings.Contains(joined, "scripts/falta.sh") {
		t.Errorf("referencias inesperadas: %v", refs)
	}
	if rep.Name != "demo" || rep.Profile != Canonical {
		t.Errorf("Report = %+v", rep)
	}
}

func TestCheckDirSinSkill(t *testing.T) {
	rep := CheckDir(filepath.Join(t.TempDir(), "nada"), Options{})
	if !rep.HasErrors() || rep.Findings[0].Rule != RuleUnreadable {
		t.Fatalf("Report = %+v", rep)
	}
}

func TestCheckDirLimpia(t *testing.T) {
	dir := writeSkill(t, t.TempDir(), "demo", goodSkill, nil)
	rep := CheckDir(dir, Options{Profile: Portable})
	// `licence` no es una clave permitida: aviso en Portable, no error.
	if rep.HasErrors() || rep.Count(SeverityWarning) != 1 {
		t.Fatalf("Report = %+v", rep)
	}
}

func TestFormatErrors(t *testing.T) {
	got := FormatErrors([]Finding{
		{Rule: "a", Severity: SeverityError, Message: "uno"},
		{Rule: "b", Severity: SeverityWarning, Message: "dos"},
		{Rule: "c", Severity: SeverityError, Message: "tres"},
	})
	if got != "a: uno\nc: tres" {
		t.Errorf("FormatErrors = %q", got)
	}
}
