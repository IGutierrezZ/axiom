package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
)

const doctorSkillOK = "---\nname: %s\ndescription: \"Hace algo. Trigger: cuando haga falta.\"\n---\n\nBody\n"

func writeDoctorSkill(t *testing.T, root, name, content string) {
	t.Helper()
	writeWorkspaceDoctorFile(t, filepath.Join(root, name, "SKILL.md"), content)
}

func okSkill(name string) string { return strings.ReplaceAll(doctorSkillOK, "%s", name) }

func TestProjectSkillsSinRaizNoSeAnade(t *testing.T) {
	stubDoctorWorld(t)
	t.Chdir(t.TempDir())
	if _, ok := runDoctorChecks(t, doctor.Options{Home: t.TempDir()})[doctor.CheckProjectSkills]; ok {
		t.Fatal("sin raíz no debe haber project:skills")
	}
}

func TestProjectSkillsLimpioEsPass(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeDoctorSkill(t, filepath.Join(root, ".agents", "skills"), "buena", okSkill("buena"))

	r := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectSkills]
	if r.Status != doctor.StatusPass || r.Scope != doctor.ScopeProject {
		t.Fatalf("project:skills = %+v, want pass de ámbito proyecto", r)
	}
}

func TestProjectSkillsFrontmatterRotoYReferenciaRotaAvisan(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	dir := filepath.Join(root, ".agents", "skills")
	writeDoctorSkill(t, dir, "rota", "# sin frontmatter\n")
	writeDoctorSkill(t, dir, "refs", okSkill("refs")+"Ver [x](references/falta.md)\n")

	r := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectSkills]
	if r.Status != doctor.StatusWarn || r.Remedy == nil || !strings.Contains(r.Detail, "1 error(s), 1 warning(s)") {
		t.Fatalf("project:skills = %+v, want warn con 1 error y 1 aviso", r)
	}
}

func TestProjectSkillsColisionYRegistroDesactualizado(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeDoctorSkill(t, filepath.Join(root, ".agents", "skills"), "dup", okSkill("dup")+"A\n")
	writeDoctorSkill(t, filepath.Join(root, ".claude", "skills"), "dup", okSkill("dup")+"B\n")

	r := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectSkills]
	if r.Status != doctor.StatusWarn || !strings.Contains(r.Detail, "collision") {
		t.Fatalf("colisión con contenido distinto = %+v, want warn", r)
	}
}

func TestProjectSkillsPiSoloInformaYNoAvisa(t *testing.T) {
	stubDoctorWorld(t)
	home, root := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeDoctorSkill(t, filepath.Join(root, ".pi", "skills"), "solo-pi", okSkill("solo-pi"))

	r := runDoctorChecks(t, doctor.Options{Home: home, Root: root})[doctor.CheckProjectSkills]
	if r.Status != doctor.StatusPass || !strings.Contains(r.Detail, "gentle-pi") {
		t.Fatalf("project:skills = %+v, want pass con el aviso de Pi", r)
	}
}

func TestSkillFolderRuntimesPorCarpeta(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	m := SkillFolderRuntimes(home, root)

	shared := m[filepath.Join(root, ".agents", "skills")]
	if len(shared) < 2 {
		t.Fatalf(".agents/skills debe compartirse entre varios runtimes: %v", shared)
	}
	if got := m[filepath.Join(root, ".claude", "skills")]; len(got) != 1 || got[0] != "claude-code" {
		t.Errorf(".claude/skills = %v, want [claude-code]", got)
	}
	if got := m[filepath.Join(root, ".pi", "skills")]; len(got) != 1 || got[0] != "pi" {
		t.Errorf(".pi/skills = %v, want [pi]", got)
	}
	if got := m[filepath.Join(home, ".pi", "agent", "skills")]; len(got) != 1 || got[0] != "pi" {
		t.Errorf("~/.pi/agent/skills = %v, want [pi]", got)
	}
}
