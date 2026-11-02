package skillregistry

import (
	"path/filepath"
	"testing"
)

// Con varios agentes, el proyecto tiene una copia compartida en .agents/skills y puede
// conservar copias en las carpetas propias de cada agente (instalaciones anteriores,
// Claude Code). El índice apunta a la compartida, no a la primera carpeta de agente
// (.opencode/skills), que era lo que ocurría antes.
func TestListPrefersSharedAgentsSkillsOverPerAgentCopies(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	for _, dir := range []string{".opencode", ".claude", ".agents"} {
		writeSkill(t, filepath.Join(cwd, dir, "skills", "dup", "SKILL.md"), minimalSkill("dup"))
	}
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "solo-claude", "SKILL.md"), minimalSkill("solo-claude"))

	byName := map[string]string{}
	for _, entry := range List(cwd, home) {
		byName[entry.Name] = entry.Path
	}
	if want := filepath.Join(cwd, ".agents", "skills", "dup", "SKILL.md"); byName["dup"] != want {
		t.Errorf("dup apunta a %q, quiero la copia compartida %q", byName["dup"], want)
	}
	// Una skill que solo existe en la carpeta de un agente se indexa donde esté.
	if want := filepath.Join(cwd, ".claude", "skills", "solo-claude", "SKILL.md"); byName["solo-claude"] != want {
		t.Errorf("solo-claude apunta a %q, quiero %q", byName["solo-claude"], want)
	}
}

// La preferencia por .agents/skills no desplaza a skills/ ni a una raíz declarada: la
// copia canónica versionada sigue ganando.
func TestListSharedAgentsSkillsDoNotOutrankCanonicalSkills(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	writeSkill(t, filepath.Join(cwd, "skills", "dup", "SKILL.md"), minimalSkill("dup"))
	writeSkill(t, filepath.Join(cwd, ".agents", "skills", "dup", "SKILL.md"), minimalSkill("dup"))
	writeSkill(t, filepath.Join(cwd, ".opencode", "skills", "dup", "SKILL.md"), minimalSkill("dup"))

	entries := List(cwd, home)
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, quiero 1", len(entries))
	}
	if want := filepath.Join(cwd, "skills", "dup", "SKILL.md"); entries[0].Path != want {
		t.Errorf("dup apunta a %q, quiero la canónica %q", entries[0].Path, want)
	}
}
