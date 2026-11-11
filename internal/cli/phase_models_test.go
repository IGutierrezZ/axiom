package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// phaseModelsFixture deja un HOME aislado y un proyecto con el estado dado.
func phaseModelsFixture(t *testing.T, st *state.InstallState) (project, home string) {
	t.Helper()
	home = t.TempDir()
	original := osUserHomeDir
	osUserHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { osUserHomeDir = original })
	project = t.TempDir()
	if st != nil {
		if err := state.ProjectStore(home, project).Write(*st); err != nil {
			t.Fatal(err)
		}
	}
	return project, home
}

func TestReadProjectPhaseModelsConvertsStateAndFallsBackToTheLegacyClaudeMap(t *testing.T) {
	project, _ := phaseModelsFixture(t, &state.InstallState{
		InstalledAgents:         []string{"claude-code", "opencode"},
		SDDMode:                 model.SDDModeMulti,
		ClaudeModelAssignments:  map[string]string{"sdd-apply": "opus"},
		ModelAssignments:        map[string]state.ModelAssignmentState{"sdd-spec": {ProviderID: "openai", ModelID: "gpt-5", Effort: "low"}},
		CopilotPhaseAssignments: map[string]state.CopilotPhaseAssignmentState{"sdd-verify": {Model: "GPT-5.2 (copilot)"}},
	})
	got, err := ReadProjectPhaseModels(project)
	if err != nil {
		t.Fatal(err)
	}
	if got.SDDMode != "multi" || len(got.Agents) != 2 {
		t.Errorf("agentes y modo = %+v", got)
	}
	if got.Claude["sdd-apply"].Model != model.ClaudeModelOpus {
		t.Errorf("sin mapa por fase debe usar el antiguo: %+v", got.Claude)
	}
	if a := got.OpenCode["sdd-spec"]; a.FullID() != "openai/gpt-5" || a.Effort != "low" {
		t.Errorf("OpenCode = %+v", got.OpenCode)
	}
	if got.Copilot["sdd-verify"].Model != "GPT-5.2 (copilot)" {
		t.Errorf("Copilot = %+v", got.Copilot)
	}
}

func TestReadProjectPhaseModelsWithoutStateIsEmptyAndRejectsAMissingRoot(t *testing.T) {
	project, _ := phaseModelsFixture(t, nil)
	got, err := ReadProjectPhaseModels(project)
	if err != nil || len(got.Agents) != 0 || len(got.Claude) != 0 {
		t.Fatalf("sin estado: %+v %v", got, err)
	}
	if _, err := ReadProjectPhaseModels(filepath.Join(project, "no-existe")); err == nil {
		t.Error("una raíz que no existe debe rechazarse")
	}
	if _, err := ReadProjectPhaseModels("  "); err == nil {
		t.Error("una raíz vacía debe rechazarse")
	}
}

func TestPhaseModelOverridesReplaceOnlyTheAdaptersTheyName(t *testing.T) {
	selection := model.Selection{
		ClaudePhaseAssignments:  map[string]model.ClaudePhaseAssignment{"sdd-apply": {Model: model.ClaudeModelOpus}},
		ClaudeModelAssignments:  map[string]model.ClaudeModelAlias{"sdd-apply": model.ClaudeModelOpus},
		CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-apply": {Model: "x"}},
	}
	(*PhaseModelOverrides)(nil).apply(&selection)
	if len(selection.ClaudePhaseAssignments) != 1 {
		t.Fatal("un receptor nil no cambia nada")
	}

	// Un mapa vacío no nil quita todas las asignaciones de ese adaptador.
	(&PhaseModelOverrides{Claude: map[string]model.ClaudePhaseAssignment{}}).apply(&selection)
	if len(selection.ClaudePhaseAssignments) != 0 || selection.ClaudeModelAssignments != nil {
		t.Errorf("Claude debe quedar sin asignaciones: %+v", selection)
	}
	if len(selection.CopilotPhaseAssignments) != 1 {
		t.Error("Copilot no se nombra y debe conservarse")
	}
}

func TestSaveProjectPhaseModelsWithNoRegenerableAdapterOnlyWritesTheState(t *testing.T) {
	project, home := phaseModelsFixture(t, &state.InstallState{
		InstalledAgents:        []string{"github-copilot-cli"},
		ClaudeModelAssignments: map[string]string{"sdd-apply": "opus"},
	})
	result, err := SaveProjectPhaseModels(project, nil, PhaseModelOverrides{
		Copilot: map[string]model.CopilotPhaseAssignment{"sdd-spec": {Model: "GPT-5.2 (copilot)", Effort: "high"}},
		Claude:  map[string]model.ClaudePhaseAssignment{},
	})
	if err != nil || !result.NoOp {
		t.Fatalf("SaveProjectPhaseModels() = %+v, %v", result, err)
	}
	got, err := state.ProjectStore(home, project).Read()
	if err != nil {
		t.Fatal(err)
	}
	if a := got.CopilotPhaseAssignments["sdd-spec"]; a.Model != "GPT-5.2 (copilot)" || a.Effort != "high" {
		t.Errorf("estado Copilot = %+v", got.CopilotPhaseAssignments)
	}
	if len(got.ClaudePhaseAssignments) != 0 || got.ClaudeModelAssignments != nil {
		t.Errorf("un mapa de Claude vacío debe vaciar también el antiguo: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(project, ".github")); !os.IsNotExist(err) {
		t.Errorf("no debe escribir ficheros del adaptador (err=%v)", err)
	}
}
