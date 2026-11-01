package sdd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// installManagedSkills escribe en dir las skills de Axiom tal como las dejaba una
// instalación workspace anterior: SDD con _shared (identidad del agente) y una
// ordinaria.
func installManagedSkills(t *testing.T, dir string, agent model.AgentID) {
	t.Helper()
	if _, err := InjectSkillDirectoryForAgent(dir, "", agent); err != nil {
		t.Fatalf("InjectSkillDirectoryForAgent(%s): %v", dir, err)
	}
	if _, err := skills.InjectDirectoryWithCapability(dir, []model.SkillID{"go-testing"}, ""); err != nil {
		t.Fatalf("InjectDirectoryWithCapability(%s): %v", dir, err)
	}
}

func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// Una copia anterior idéntica a lo que Axiom genera hoy se retira, y con ella la
// carpeta contenedora si queda vacía; la copia compartida no se toca.
func TestRetireLegacyWorkspaceSkillsRemovesProvablyOwnedCopies(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".opencode", "skills")
	shared := filepath.Join(root, ".agents", "skills")
	installManagedSkills(t, legacy, model.AgentOpenCode)
	installManagedSkills(t, shared, "")

	result := RetireLegacyWorkspaceSkills(legacy, shared, model.AgentOpenCode)
	if len(result.Notes) != 0 {
		t.Errorf("notas inesperadas: %v", result.Notes)
	}
	if len(result.Removed) == 0 {
		t.Fatal("no se retiró nada")
	}
	if exists(legacy) {
		t.Errorf("la carpeta anterior debería haberse borrado al quedar vacía")
	}
	if !exists(filepath.Join(root, ".opencode")) {
		t.Errorf(".opencode es del agente y no debe borrarse")
	}
	for _, rel := range []string{"sdd-apply/SKILL.md", "go-testing/SKILL.md", "_shared/README.md"} {
		if !exists(filepath.Join(shared, filepath.FromSlash(rel))) {
			t.Errorf("la copia compartida perdió %s", rel)
		}
	}
	// Idempotente: una segunda pasada no hace nada.
	if again := RetireLegacyWorkspaceSkills(legacy, shared, model.AgentOpenCode); len(again.Removed) != 0 || len(again.Notes) != 0 {
		t.Errorf("segunda pasada = %+v", again)
	}
}

// Una skill de Axiom que el usuario modificó se conserva con un aviso; las demás, que
// siguen siendo de Axiom, se retiran. La skill ajena ni se mira.
func TestRetireLegacyWorkspaceSkillsKeepsModifiedAndForeign(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".github", "skills")
	shared := filepath.Join(root, ".agents", "skills")
	installManagedSkills(t, legacy, model.AgentVSCodeCopilot)
	installManagedSkills(t, shared, "")

	edited := filepath.Join(legacy, "go-testing", "SKILL.md")
	original, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	writeFileT(t, edited, string(original)+"\nMi nota.\n")
	foreign := filepath.Join(legacy, "mi-skill", "SKILL.md")
	writeFileT(t, foreign, "---\nname: mi-skill\ndescription: mía\n---\n")

	result := RetireLegacyWorkspaceSkills(legacy, shared, model.AgentVSCodeCopilot)
	if len(result.Notes) != 1 || !strings.Contains(result.Notes[0], filepath.Join(legacy, "go-testing")) {
		t.Fatalf("quiero un aviso sobre go-testing, tengo %v", result.Notes)
	}
	if !exists(edited) {
		t.Errorf("la copia modificada no debe borrarse")
	}
	if !exists(foreign) {
		t.Errorf("la skill ajena no debe tocarse")
	}
	if exists(filepath.Join(legacy, "sdd-apply")) || exists(filepath.Join(legacy, "_shared")) {
		t.Errorf("las copias de Axiom sin cambios deberían haberse retirado")
	}
	if !exists(legacy) {
		t.Errorf("la carpeta contiene contenido que se conserva: no puede borrarse")
	}
}

// Un fichero extra dentro de una skill de Axiom la hace no retirable: puede ser del
// usuario.
func TestRetireLegacyWorkspaceSkillsKeepsFolderWithExtraFile(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".gemini", "skills")
	shared := filepath.Join(root, ".agents", "skills")
	installManagedSkills(t, legacy, model.AgentGeminiCLI)
	installManagedSkills(t, shared, "")
	writeFileT(t, filepath.Join(legacy, "go-testing", "notas.md"), "mías")

	result := RetireLegacyWorkspaceSkills(legacy, shared, model.AgentGeminiCLI)
	if len(result.Notes) != 1 || !strings.Contains(result.Notes[0], "notas.md") {
		t.Fatalf("quiero un aviso que nombre el fichero extra, tengo %v", result.Notes)
	}
	if !exists(filepath.Join(legacy, "go-testing", "SKILL.md")) || !exists(filepath.Join(legacy, "go-testing", "notas.md")) {
		t.Errorf("la skill con fichero extra debe quedar entera")
	}
}

// Si la copia compartida aún no está, no se retira la única copia que hay.
func TestRetireLegacyWorkspaceSkillsKeepsCopyWithoutReplacement(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".cursor", "skills")
	shared := filepath.Join(root, ".agents", "skills")
	installManagedSkills(t, legacy, model.AgentCursor)

	result := RetireLegacyWorkspaceSkills(legacy, shared, model.AgentCursor)
	if len(result.Removed) != 0 || len(result.Notes) != 0 {
		t.Errorf("resultado = %+v, quiero sin cambios ni avisos", result)
	}
	if !exists(filepath.Join(legacy, "go-testing", "SKILL.md")) {
		t.Errorf("la única copia no debe borrarse")
	}
}

// La lista de la copia de seguridad es el máximo que se podría borrar y no exige que
// la copia compartida exista (la copia se hace antes de instalar).
func TestLegacyWorkspaceSkillFilesDoNotNeedSharedCopy(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".kilo", "skills")
	shared := filepath.Join(root, ".agents", "skills")
	installManagedSkills(t, legacy, model.AgentKilocode)

	files := LegacyWorkspaceSkillFiles(legacy, shared, model.AgentKilocode)
	want := filepath.Join(legacy, "go-testing", "SKILL.md")
	found := false
	for _, f := range files {
		found = found || f == want
	}
	if !found {
		t.Errorf("%s no está en la lista de copia de seguridad: %v", want, files)
	}
	if !exists(want) {
		t.Errorf("listar no debe borrar nada")
	}
}

// Sin carpeta anterior no hay nada que hacer ni que avisar.
func TestRetireLegacyWorkspaceSkillsLeavesMissingFolderAlone(t *testing.T) {
	root := t.TempDir()
	result := RetireLegacyWorkspaceSkills(filepath.Join(root, ".opencode", "skills"), filepath.Join(root, ".agents", "skills"), model.AgentOpenCode)
	if len(result.Removed) != 0 || len(result.Notes) != 0 {
		t.Errorf("resultado = %+v, quiero vacío", result)
	}
}
