package tui

import (
	"reflect"
	"slices"
	"testing"

	componentuninstall "github.com/IGutierrezZ/axiom/v3/internal/components/uninstall"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// La selección por defecto de la desinstalación parcial retira el agente entero
// (Engram incluido, para que salga de state.json): todo el catálogo desinstalable.
func TestDefaultUninstallComponentsCoverWholeCatalog(t *testing.T) {
	m := NewModel(system.DetectionResult{}, "dev")

	if !slices.Contains(m.UninstallComponents, model.ComponentEngram) {
		t.Fatalf("default UninstallComponents = %v, want %q", m.UninstallComponents, model.ComponentEngram)
	}

	reset := m.withResetUninstallState()
	if !reflect.DeepEqual(reset.UninstallComponents, m.UninstallComponents) {
		t.Fatalf("reset UninstallComponents = %v, want the default %v", reset.UninstallComponents, m.UninstallComponents)
	}
}

// Con la selección por defecto, el TUI no pasa componentes: el servicio aplica lo
// mismo que `axiom uninstall --agent X` sin `--components` (conserva el registro
// MCP de Engram). Cualquier otra selección es explícita y viaja tal cual.
func TestStartUninstallPassesComponentsOnlyForNonDefaultSelection(t *testing.T) {
	defaultSelection := defaultUninstallComponents()
	reordered := slices.Clone(defaultSelection)
	slices.Reverse(reordered)
	onlyEngram := []model.ComponentID{model.ComponentEngram}

	cases := []struct {
		name         string
		mode         model.UninstallMode
		selection    []model.ComponentID
		wantPassed   []model.ComponentID
		withProfiles bool
	}{
		{name: "default selection", mode: model.UninstallModePartial, selection: defaultSelection, wantPassed: nil},
		{name: "default selection in another order", mode: model.UninstallModePartial, selection: reordered, wantPassed: nil},
		{name: "default selection with profiles", mode: model.UninstallModePartial, selection: defaultSelection, wantPassed: nil, withProfiles: true},
		{name: "only Engram", mode: model.UninstallModePartial, selection: onlyEngram, wantPassed: onlyEngram},
		{name: "full uninstall", mode: model.UninstallModeFull, selection: defaultSelection, wantPassed: defaultSelection},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(system.DetectionResult{}, "dev")
			m.UninstallMode = tc.mode
			m.UninstallAgents = []model.AgentID{model.AgentClaudeCode}
			m.UninstallComponents = slices.Clone(tc.selection)

			var got []model.ComponentID
			called := false
			if tc.withProfiles {
				m.UninstallProfilesToRemove = []string{"cheap"}
				m.UninstallWithProfilesFn = func(_ []model.AgentID, componentIDs []model.ComponentID, _ []string, _ model.EngramUninstallScope) (componentuninstall.Result, error) {
					called = true
					got = componentIDs
					return componentuninstall.Result{}, nil
				}
			} else {
				m.UninstallFn = func(_ []model.AgentID, componentIDs []model.ComponentID) (componentuninstall.Result, error) {
					called = true
					got = componentIDs
					return componentuninstall.Result{}, nil
				}
			}

			msg := m.startUninstall()().(UninstallDoneMsg)
			if msg.Err != nil {
				t.Fatalf("UninstallDoneMsg.Err = %v, want nil", msg.Err)
			}
			if !called {
				t.Fatal("uninstall function was not called")
			}
			if len(tc.wantPassed) == 0 {
				if len(got) != 0 {
					t.Fatalf("componentIDs = %v, want none (service default)", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tc.wantPassed) {
				t.Fatalf("componentIDs = %v, want %v", got, tc.wantPassed)
			}
		})
	}
}
