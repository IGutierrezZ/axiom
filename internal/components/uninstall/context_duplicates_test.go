package uninstall

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

const (
	managedBlockV1 = "<!-- axiom:sdd-orchestrator -->\nreglas v1\n<!-- /axiom:sdd-orchestrator -->\n"
	managedBlockV2 = "<!-- axiom:sdd-orchestrator -->\nreglas v2\n<!-- /axiom:sdd-orchestrator -->\n"
)

// duplicateAdapters devuelve el adaptador de usuario y el ligado a root.
func duplicateAdapters(t *testing.T, agent model.AgentID, root string) (agents.Adapter, agents.Adapter) {
	t.Helper()
	adapter, err := agents.NewAdapter(agent)
	if err != nil {
		t.Fatalf("NewAdapter(%s) error = %v", agent, err)
	}
	return adapter, agents.BindWorkspace(adapter, root)
}

func findDuplicate(found []ContextDuplicate, agent model.AgentID, kind string) (ContextDuplicate, bool) {
	for _, duplicate := range found {
		if duplicate.Kind == kind && slices.Contains(duplicate.Agents, agent) {
			return duplicate, true
		}
	}
	return ContextDuplicate{}, false
}

func TestContextDuplicatesInstructions(t *testing.T) {
	// Claude Code (sin ligar) y Codex y Cursor (ligados): el mismo criterio para todos.
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCodex, model.AgentCursor} {
		t.Run(string(agent), func(t *testing.T) {
			home, root := t.TempDir(), t.TempDir()
			user, project := duplicateAdapters(t, agent, root)
			userFile, projectFile := user.SystemPromptFile(home), project.SystemPromptFile(root)

			// Solo la copia de usuario: no hay duplicado.
			writeGlobalFile(t, userFile, managedBlockV1)
			if _, ok := findDuplicate(ContextDuplicates(home, root), agent, DuplicateInstructions); ok {
				t.Fatal("con una sola copia no debe haber duplicado")
			}

			// Las dos con el mismo contenido: duplicado sin divergencia.
			writeGlobalFile(t, projectFile, managedBlockV1)
			duplicate, ok := findDuplicate(ContextDuplicates(home, root), agent, DuplicateInstructions)
			if !ok {
				t.Fatal("dos copias gestionadas deben ser un duplicado")
			}
			if duplicate.UserPath != userFile || duplicate.WorkspacePath != projectFile {
				t.Errorf("rutas = %q y %q, quiero %q y %q", duplicate.UserPath, duplicate.WorkspacePath, userFile, projectFile)
			}
			if duplicate.Diverges {
				t.Error("copias iguales no divergen")
			}

			// Contenido gestionado distinto: divergencia.
			writeGlobalFile(t, projectFile, managedBlockV2)
			duplicate, ok = findDuplicate(ContextDuplicates(home, root), agent, DuplicateInstructions)
			if !ok || !duplicate.Diverges {
				t.Fatalf("copias distintas deben marcar divergencia, got %+v ok=%v", duplicate, ok)
			}
		})
	}
}

func TestContextDuplicatesIgnoresUserEditedFilesWithoutMarkers(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	user, project := duplicateAdapters(t, model.AgentClaudeCode, root)
	writeGlobalFile(t, user.SystemPromptFile(home), managedBlockV1)
	writeGlobalFile(t, project.SystemPromptFile(root), "# Mis notas\nSin marcas de Axiom.\n")

	if _, ok := findDuplicate(ContextDuplicates(home, root), model.AgentClaudeCode, DuplicateInstructions); ok {
		t.Fatal("un fichero del usuario sin marcas de Axiom no debe informarse")
	}
}

func TestContextDuplicatesEmptyRootReportsNothing(t *testing.T) {
	home := t.TempDir()
	user, _ := duplicateAdapters(t, model.AgentClaudeCode, "")
	writeGlobalFile(t, user.SystemPromptFile(home), managedBlockV1)

	if got := ContextDuplicates(home, ""); len(got) != 0 {
		t.Fatalf("sin raíz de proyecto no se comprueba nada, got %+v", got)
	}
}

func TestContextDuplicatesSkills(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCodex} {
		t.Run(string(agent), func(t *testing.T) {
			home, root := t.TempDir(), t.TempDir()
			user, project := duplicateAdapters(t, agent, root)
			names := embeddedDirNames("skills")
			if len(names) == 0 {
				t.Fatal("no hay skills incrustadas")
			}
			skill := names[0]

			writeGlobalFile(t, filepath.Join(user.SkillsDir(home), skill, "SKILL.md"), "v1")
			writeGlobalFile(t, filepath.Join(user.SkillsDir(home), "skill-del-usuario", "SKILL.md"), "mía")
			if _, ok := findDuplicate(ContextDuplicates(home, root), agent, DuplicateSkills); ok {
				t.Fatal("con una sola copia no debe haber duplicado")
			}

			writeGlobalFile(t, filepath.Join(project.SkillsDir(root), skill, "SKILL.md"), "v1")
			writeGlobalFile(t, filepath.Join(project.SkillsDir(root), "skill-del-usuario", "SKILL.md"), "mía")
			duplicate, ok := findDuplicate(ContextDuplicates(home, root), agent, DuplicateSkills)
			if !ok {
				t.Fatal("la misma skill en las dos carpetas debe ser un duplicado")
			}
			if !slices.Equal(duplicate.Items, []string{skill}) {
				t.Errorf("Items = %v, quiero solo la skill de Axiom %q", duplicate.Items, skill)
			}
			if duplicate.Diverges {
				t.Error("copias iguales no divergen")
			}

			writeGlobalFile(t, filepath.Join(project.SkillsDir(root), skill, "SKILL.md"), "v2")
			duplicate, _ = findDuplicate(ContextDuplicates(home, root), agent, DuplicateSkills)
			if !duplicate.Diverges {
				t.Error("skills distintas deben marcar divergencia")
			}
		})
	}
}

func TestContextDuplicatesSubAgentsCommandsAndOutputStyles(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	user, project := duplicateAdapters(t, model.AgentClaudeCode, root)

	for _, item := range []struct {
		kind, userDir, projectDir, file string
	}{
		{DuplicateSubAgents, user.SubAgentsDir(home), project.SubAgentsDir(root), "sdd-apply.md"},
		{DuplicateCommands, user.CommandsDir(home), project.CommandsDir(root), "sdd-new.md"},
		{DuplicateOutputStyles, user.OutputStyleDir(home), project.OutputStyleDir(root), "axiom.md"},
	} {
		writeGlobalFile(t, filepath.Join(item.userDir, item.file), "v1")
		writeGlobalFile(t, filepath.Join(item.projectDir, item.file), "v2")
		// Un fichero ajeno repetido en las dos carpetas no cuenta.
		writeGlobalFile(t, filepath.Join(item.userDir, "ajeno.md"), "x")
		writeGlobalFile(t, filepath.Join(item.projectDir, "ajeno.md"), "x")
	}

	found := ContextDuplicates(home, root)
	for kind, want := range map[string]string{
		DuplicateSubAgents:    "sdd-apply",
		DuplicateCommands:     "sdd-new",
		DuplicateOutputStyles: "axiom",
	} {
		duplicate, ok := findDuplicate(found, model.AgentClaudeCode, kind)
		if !ok {
			t.Errorf("falta el duplicado de %s", kind)
			continue
		}
		if !slices.Equal(duplicate.Items, []string{want}) || !duplicate.Diverges {
			t.Errorf("%s: %+v, quiero Items=[%s] y divergencia", kind, duplicate, want)
		}
	}
}

func TestContextDuplicatesNoneWhenOnlyProjectInstalled(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	_, project := duplicateAdapters(t, model.AgentClaudeCode, root)
	writeGlobalFile(t, project.SystemPromptFile(root), managedBlockV1)

	if got := ContextDuplicates(home, root); len(got) != 0 {
		t.Fatalf("sin instalación de usuario no hay duplicados, got %+v", got)
	}
}
