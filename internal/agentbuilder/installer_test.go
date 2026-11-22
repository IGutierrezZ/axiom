package agentbuilder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

// makeAgent creates a minimal GeneratedAgent for testing.
func makeAgent(name, content string) *GeneratedAgent {
	return &GeneratedAgent{
		Name:        name,
		Title:       "Test Agent",
		Description: "Agente de prueba",
		Trigger:     "cuando se pida una prueba",
		Content:     content,
	}
}

func TestInstall_HappyPath_WritesFilesToBothDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	agent := makeAgent("my-agent", "# My Agent\n\nContent here.")

	adapters := []AdapterInfo{
		{AgentID: model.AgentClaudeCode, SkillsDir: dir1},
		{AgentID: model.AgentOpenCode, SkillsDir: dir2},
	}

	results, err := Install(agent, adapters, "")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("result for %s: Success=false, err=%v", r.AgentID, r.Err)
		}
	}
}

func TestInstall_FileContentMatchesAgentContent(t *testing.T) {
	dir := t.TempDir()
	content := "# My Agent\n\n## Description\nDoes things.\n"
	agent := makeAgent("my-agent", content)

	adapters := []AdapterInfo{
		{AgentID: model.AgentClaudeCode, SkillsDir: dir},
	}

	results, err := Install(agent, adapters, "")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	skillFile := filepath.Join(dir, "my-agent", "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// Install antepone el frontmatter compuesto en código y conserva el cuerpo.
	want := "---\nname: my-agent\ndescription: \"Agente de prueba. Trigger: cuando se pida una prueba\"\n---\n\n" + content
	if string(data) != want {
		t.Errorf("file content = %q, want %q", string(data), want)
	}
	if findings := skillcheck.CheckContent(string(data), "my-agent", skillcheck.Options{Profile: skillcheck.Canonical}); len(findings) != 0 {
		t.Errorf("el frontmatter compuesto debe ser válido incluso en perfil canónico: %+v", findings)
	}

	if results[0].Path != skillFile {
		t.Errorf("Path = %q, want %q", results[0].Path, skillFile)
	}
}

func TestInstall_ColapsaDescripcionMultilineaYEscapaComillas(t *testing.T) {
	dir := t.TempDir()
	agent := makeAgent("my-agent", "# Body\n")
	agent.Description = "Revisa \"cosas\"\n  en varias   líneas."
	agent.Trigger = "al\nrevisar"

	if _, err := Install(agent, []AdapterInfo{{AgentID: model.AgentClaudeCode, SkillsDir: dir}}, ""); err != nil {
		t.Fatalf("Install error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "my-agent", "SKILL.md"))
	want := "description: \"Revisa \\\"cosas\\\" en varias líneas. Trigger: al revisar\"\n"
	if !strings.Contains(string(data), want) {
		t.Errorf("SKILL.md = %q, want it to contain %q", string(data), want)
	}
}

func TestInstall_ContenidoConFrontmatterPropioSeRespeta(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: my-agent\ndescription: \"Propio. Trigger: x\"\n---\n\n# Body\n"
	if _, err := Install(makeAgent("my-agent", content), []AdapterInfo{{AgentID: model.AgentClaudeCode, SkillsDir: dir}}, ""); err != nil {
		t.Fatalf("Install error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "my-agent", "SKILL.md"))
	if string(data) != content {
		t.Errorf("SKILL.md = %q, want the original content", string(data))
	}
}

func TestInstall_RechazaSinEscribirNada(t *testing.T) {
	tests := []struct {
		name    string
		agent   *GeneratedAgent
		wantMsg string
	}{
		{"nombre con separador de ruta", makeAgent("../evil", "# x\n"), "nombre de skill no válido"},
		{"nombre con subcarpeta", makeAgent("a/b", "# x\n"), "nombre de skill no válido"},
		{"nombre vacío", makeAgent("", "# x\n"), "nombre de skill no válido"},
		{"descripción vacía", &GeneratedAgent{Name: "my-agent", Content: "# x\n"}, "description-missing"},
		{"frontmatter propio con otro nombre", makeAgent("my-agent", "---\nname: otro\ndescription: \"A. Trigger: b\"\n---\n"), "name-mismatch"},
		{"nombre reservado", makeAgent("sdd-propio", "# x\n"), "name-reserved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			results, err := Install(tt.agent, []AdapterInfo{{AgentID: model.AgentClaudeCode, SkillsDir: dir}}, "")
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantMsg)
			}
			if len(results) != 0 {
				t.Errorf("results = %+v, want none", results)
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 0 {
				t.Errorf("no debe escribirse nada, hay %d entradas en %s", len(entries), dir)
			}
		})
	}
}

func TestInstall_MissingDirectory_CreatedAutomatically(t *testing.T) {
	base := t.TempDir()
	// nested subdirectory that doesn't exist yet
	skillsDir := filepath.Join(base, "does", "not", "exist")

	agent := makeAgent("auto-dir-agent", "# Agent\n")
	adapters := []AdapterInfo{
		{AgentID: model.AgentClaudeCode, SkillsDir: skillsDir},
	}

	results, err := Install(agent, adapters, "")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if !results[0].Success {
		t.Errorf("expected success even for missing parent dir")
	}

	skillFile := filepath.Join(skillsDir, "auto-dir-agent", "SKILL.md")
	if _, statErr := os.Stat(skillFile); statErr != nil {
		t.Errorf("SKILL.md not created: %v", statErr)
	}
}

func TestInstall_RollbackOnSecondWriteFailure(t *testing.T) {
	// dir1: writable — first write succeeds
	dir1 := t.TempDir()

	// dir2: make it read-only so the mkdir inside it fails
	dir2 := t.TempDir()
	// Create a file with the same name as the agent skill dir so MkdirAll fails
	blocker := filepath.Join(dir2, "my-agent")
	if err := os.WriteFile(blocker, []byte("block"), 0444); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	// Make blocker read-only to prevent overwrite
	if err := os.Chmod(blocker, 0444); err != nil {
		t.Fatalf("chmod blocker: %v", err)
	}

	agent := makeAgent("my-agent", "# Agent\n")
	adapters := []AdapterInfo{
		{AgentID: model.AgentClaudeCode, SkillsDir: dir1},
		{AgentID: model.AgentOpenCode, SkillsDir: dir2},
	}

	_, err := Install(agent, adapters, "")
	if err == nil {
		t.Fatal("expected error when second write fails")
	}

	// Verify rollback: the first file should be removed.
	firstFile := filepath.Join(dir1, "my-agent", "SKILL.md")
	if _, statErr := os.Stat(firstFile); statErr == nil {
		t.Errorf("expected first file to be rolled back, but it still exists: %s", firstFile)
	}
}

func TestInstall_EmptyAdapters_NoErrorAndEmptyResults(t *testing.T) {
	agent := makeAgent("my-agent", "# Agent\n")

	results, err := Install(agent, []AdapterInfo{}, "")
	if err != nil {
		t.Fatalf("unexpected error with empty adapters: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results len = %d, want 0", len(results))
	}
}

func TestInstall_NilAgent_ReturnsError(t *testing.T) {
	_, err := Install(nil, []AdapterInfo{}, "")
	if err == nil {
		t.Fatal("expected error for nil agent")
	}
}
