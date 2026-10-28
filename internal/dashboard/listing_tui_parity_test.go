package dashboard

import (
	"path/filepath"
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/tui"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// governanceSDDIncrementsEntry es la posición de «Incrementos SDD» en el menú de gobernanza de la TUI.
const governanceSDDIncrementsEntry = 1

// REQ-25.34 escenario «La TUI lista el mismo conjunto» (INC-25 S10-c4): con un cambio solo en su worktree,
// otro fusionado y sin archivar, uno archivado y el sumidero exento, la pantalla de incrementos de la TUI
// y `GET /api/increments` listan el mismo conjunto de cambios, con las mismas marcas de pendiente.
func TestTUIListsTheSameChangesAsGetIncrements(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "y", "proposal.md"), "# y fusionado\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "e2e-cumulative", "tasks.md"), "- [ ] 1.1 a\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktreeChange(t, svc, ws, "x")

	listing := getListing(t, svc)

	t.Chdir(ws.Master)
	model := tui.NewModel(system.DetectionResult{}, "dev")
	model.Screen = tui.ScreenGovernance
	model.Cursor = governanceSDDIncrementsEntry
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state := updated.(tui.Model)
	if state.Screen != tui.ScreenSDDIncrements {
		t.Fatalf("pantalla = %v, se esperaba la de incrementos SDD", state.Screen)
	}

	var fromAPI, fromTUI, pendingAPI, pendingTUI []string
	for _, increment := range listing.list {
		fromAPI = append(fromAPI, increment.Type+":"+increment.Name)
		if increment.PendingArchive {
			pendingAPI = append(pendingAPI, increment.Name)
		}
	}
	for _, increment := range state.SDDIncrements {
		fromTUI = append(fromTUI, increment.Type+":"+increment.Name)
		if increment.PendingArchive {
			pendingTUI = append(pendingTUI, increment.Name)
		}
	}
	slices.Sort(fromAPI)
	slices.Sort(fromTUI)
	if !slices.Equal(fromAPI, fromTUI) {
		t.Errorf("la TUI lista %v y GET /api/increments %v; debían ser el mismo conjunto", fromTUI, fromAPI)
	}
	if want := []string{"active:x", "active:y", "archived:2026-01-01-viejo"}; !slices.Equal(fromAPI, want) {
		t.Errorf("conjunto = %v, se esperaba %v", fromAPI, want)
	}
	if !slices.Equal(pendingAPI, pendingTUI) || !slices.Equal(pendingAPI, []string{"y"}) {
		t.Errorf("pendientes de archivar: TUI %v, API %v; se esperaba solo y en ambos", pendingTUI, pendingAPI)
	}
}
