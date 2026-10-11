package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IGutierrezZ/axiom/v3/internal/handoff"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Posiciones del menú de gobernanza (ScreenGovernance).
const (
	governanceSDDIncrements = 1
	governanceMultiRole     = 2
	governanceHandoffs      = 3
	governanceLivingDocs    = 4
)

// governanceModel crea un modelo en la pantalla de gobernanza, con el directorio
// actual en start. Las pruebas usan Model.Update directo, sin teatest.
func governanceModel(t *testing.T, start string) Model {
	t.Helper()
	t.Chdir(start)
	m := NewModel(system.DetectionResult{}, "dev")
	m.Screen = ScreenGovernance
	return m
}

// openGovernanceEntry pulsa Enter sobre la entrada indicada del menú de
// gobernanza y devuelve el modelo resultante.
func openGovernanceEntry(t *testing.T, m Model, entry int, want Screen) Model {
	t.Helper()
	m.Screen = ScreenGovernance
	m.Cursor = entry
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state := updated.(Model)
	if state.Screen != want {
		t.Fatalf("entrada %d: pantalla = %v, se esperaba %v", entry, state.Screen, want)
	}
	return state
}

func writeValidHandoff(t *testing.T, path, change string) {
	t.Helper()
	ho := &handoff.Handoff{
		Metadata: handoff.Metadata{
			Change:    change,
			FromPhase: handoff.PhaseDesign,
			ToPhase:   handoff.PhaseTasks,
			FromRole:  "architect",
			ToRole:    "developer",
			Timestamp: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			Status:    handoff.StatusReady,
		},
		Sections: handoff.Sections{
			ExecutiveSummary:   "Resumen de prueba",
			Artifacts:          "changes/" + change,
			Decisions:          "Ninguna",
			RisksAndBlockers:   "Ninguno",
			DirectInstructions: "Continuar",
		},
	}
	if err := handoff.WriteFile(path, ho); err != nil {
		t.Fatalf("escribir handoff de prueba: %v", err)
	}
}

// TestGovernanceLoadersResolveThePlanningHome cubre REQ-24.10 y REQ-17.10: los
// cuatro cargadores resuelven el planning home con Locate desde la carpeta
// maestra y desde un repositorio de código del espacio canónico, donde no hay
// ni openspec/ ni axiom.yaml propios.
func TestGovernanceLoadersResolveThePlanningHome(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start func(workspacetest.Workspace) string
	}{
		{"carpeta maestra", func(ws workspacetest.Workspace) string { return ws.Master }},
		{"repositorio de código", func(ws workspacetest.Workspace) string { return filepath.Join(ws.Backend, "src") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ws := workspacetest.Canonical(t)
			workspacetest.WriteFile(t, filepath.Join(ws.Home, "specs", "billing", "spec.md"),
				"# Facturación\n\n### Requirement: Regla de facturación (REQ-1)\nTexto.\n")
			writeValidHandoff(t, filepath.Join(ws.Home, "changes", "inc-99-ejemplo", "handoff.md"), "inc-99-ejemplo")
			m := governanceModel(t, tt.start(ws))

			state := openGovernanceEntry(t, m, governanceSDDIncrements, ScreenSDDIncrements)
			if len(state.SDDIncrements) != 1 || state.SDDIncrements[0].Name != "inc-99-ejemplo" || state.SDDIncrements[0].Type != "active" {
				t.Fatalf("SDDIncrements = %+v, se esperaba el cambio del planning home", state.SDDIncrements)
			}
			if state.SDDActiveChange != "inc-99-ejemplo" {
				t.Errorf("SDDActiveChange = %q", state.SDDActiveChange)
			}
			if state.GovernanceMessage != "" {
				t.Errorf("sin divergencia no hay mensaje: %q", state.GovernanceMessage)
			}

			state = openGovernanceEntry(t, state, governanceMultiRole, ScreenMultiRole)
			roles := map[string]bool{}
			for _, role := range state.MultiRoles {
				roles[role.ID] = true
			}
			if !roles["backend"] || !roles["frontend"] || len(roles) != 2 {
				t.Errorf("MultiRoles = %+v, se esperaban los roles de la configuración del puntero", state.MultiRoles)
			}
			if state.MultiRoleBarrier == nil {
				t.Error("la barrera debe evaluarse sobre el cambio del planning home")
			}

			state = openGovernanceEntry(t, state, governanceHandoffs, ScreenHandoffs)
			if state.HandoffErr != "" || state.ActiveHandoff == nil || state.ActiveHandoff.Metadata.Change != "inc-99-ejemplo" {
				t.Errorf("handoff: err=%q, handoff=%+v", state.HandoffErr, state.ActiveHandoff)
			}

			state = openGovernanceEntry(t, state, governanceLivingDocs, ScreenLivingDoc)
			if len(state.LivingSpecs) != 1 || state.LivingSpecs[0].Domain != "billing" {
				t.Errorf("LivingSpecs = %+v, se esperaba el dominio del planning home", state.LivingSpecs)
			}
			if state.GovernanceMessage != "" {
				t.Errorf("sin divergencia no hay mensaje: %q", state.GovernanceMessage)
			}

			// Las cargas son de solo lectura: no aparece openspec/ fuera del
			// repositorio de especificaciones.
			for _, dir := range []string{ws.Master, ws.Backend, ws.Frontend} {
				if _, err := os.Stat(filepath.Join(dir, "openspec")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("no debe existir openspec/ en %s (err=%v)", dir, err)
				}
			}
		})
	}
}

// TestGovernanceMessageShowsTheLegacyLocationWarning cubre el aviso Q14: en un
// proyecto embebido con specs_repository distinto, que conserva specs en la
// ubicación antigua, los cargadores lo muestran en el mensaje de gobernanza.
func TestGovernanceMessageShowsTheLegacyLocationWarning(t *testing.T) {
	ws := workspacetest.Embedded(t, "specs")
	legacy := filepath.Join(ws.Master, "specs", "specs")
	workspacetest.WriteFile(t, filepath.Join(legacy, "antiguo", "spec.md"), "# Antiguo\n")
	m := governanceModel(t, ws.Master)

	for _, tt := range []struct {
		name   string
		entry  int
		screen Screen
	}{
		{"incrementos SDD", governanceSDDIncrements, ScreenSDDIncrements},
		{"multirrol", governanceMultiRole, ScreenMultiRole},
		{"documentación viva", governanceLivingDocs, ScreenLivingDoc},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := openGovernanceEntry(t, m, tt.entry, tt.screen)
			for _, want := range []string{legacy, filepath.Join(ws.Home, "specs")} {
				if !strings.Contains(state.GovernanceMessage, want) {
					t.Errorf("el mensaje de gobernanza debe nombrar %q: %q", want, state.GovernanceMessage)
				}
			}
		})
	}
}

// TestGovernanceLoadersReportAnUnresolvablePlanningHome cubre que un fallo de
// Locate llega a la pantalla con su causa y su continuación, sin listas
// obsoletas.
func TestGovernanceLoadersReportAnUnresolvablePlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	m := governanceModel(t, ws.Master)
	_, locateErr := workspace.Locate(ws.Master)
	if locateErr == nil {
		t.Fatal("el puntero roto debe rechazarse")
	}
	m.SDDIncrements = nil
	m.SDDActiveChange = "inc-obsoleto"

	for _, tt := range []struct {
		name   string
		entry  int
		screen Screen
	}{
		{"incrementos SDD", governanceSDDIncrements, ScreenSDDIncrements},
		{"multirrol", governanceMultiRole, ScreenMultiRole},
		{"documentación viva", governanceLivingDocs, ScreenLivingDoc},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := openGovernanceEntry(t, m, tt.entry, tt.screen)
			if !strings.Contains(state.GovernanceMessage, locateErr.Error()) {
				t.Errorf("el mensaje debe llevar la causa y la continuación %q: %q", locateErr.Error(), state.GovernanceMessage)
			}
		})
	}

	t.Run("handoffs", func(t *testing.T) {
		state := openGovernanceEntry(t, m, governanceHandoffs, ScreenHandoffs)
		if !strings.Contains(state.HandoffErr, locateErr.Error()) || state.ActiveHandoff != nil {
			t.Errorf("HandoffErr = %q, handoff = %+v", state.HandoffErr, state.ActiveHandoff)
		}
	})
}
