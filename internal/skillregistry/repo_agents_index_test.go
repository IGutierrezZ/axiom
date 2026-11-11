package skillregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepoAgentsIndexMatchesVersionedSkills guarda la deriva del índice de skills
// versionado en el AGENTS.md de este repositorio.
//
// El hook UserPromptSubmit de Claude Code y el plugin de OpenCode ejecutan
// `axiom skill-registry refresh` en el checkout, que reescribe el bloque
// axiom:skills-index a partir de la description de cada SKILL.md. Si un cambio
// toca una description sin regenerar AGENTS.md, ese refresh deja AGENTS.md
// modificado en el checkout de quien abra la siguiente sesión. Este test hace
// que el desfase falle en el PR que lo provoca.
//
// Reproduce la vista versionada de Regenerate (solo skills del proyecto que git
// versiona) con un home vacío, para que las skills de usuario de la máquina no
// influyan; Regenerate las descarta igualmente para AGENTS.md.
func TestRepoAgentsIndexMatchesVersionedSkills(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	agentsPath := filepath.Join(root, AgentsRelPath)
	current, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Skipf("sin AGENTS.md en la raíz del repositorio: %v", err)
	}

	files, err := findAllSkillFiles(uniqueExistingDirs(ProjectSkillDirs(root)))
	if err != nil {
		t.Fatal(err)
	}
	versionable := versionableFiles(root, files)
	entries := make([]SkillEntry, 0, len(files))
	for _, file := range files {
		if entry, ok := LoadSkill(file); ok {
			entries = append(entries, entry)
		}
	}
	versioned := dedupeBySkillName(filterVersionable(entries, versionable), root)
	if len(versioned) == 0 {
		t.Skip("git no informa de skills versionadas (¿checkout sin .git?)")
	}

	want := injectSkillsIndex(AdoptSkillsIndexMarkers(string(current)), skillsIndexBody(root, versioned))
	if want != string(current) {
		t.Fatalf("el índice de skills de AGENTS.md está desfasado respecto a los SKILL.md versionados.\n"+
			"Regenéralo con `axiom skill-registry refresh --force --no-gitignore --cwd <repo>` y versiona AGENTS.md en el mismo cambio.\n"+
			"Primera diferencia: %s", firstDiff(string(current), want))
	}
}

// firstDiff devuelve la primera línea en que difieren got y want.
func firstDiff(got, want string) string {
	gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(gl) || i < len(wl); i++ {
		var g, w string
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			return fmt.Sprintf("línea %d\n  AGENTS.md: %s\n  esperado:  %s", i+1, g, w)
		}
	}
	return "(ninguna)"
}
