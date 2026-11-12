package tui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	livingCanonicalConfig  = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"
	livingResidualBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
)

// livingDocsOf abre la pantalla de documentación viva de un espacio canónico con
// el bloque dado, desde el repositorio de código (donde no hay marcadores).
func livingDocsOf(t *testing.T, block string) (workspacetest.Workspace, Model) {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), livingCanonicalConfig+block)
	m := governanceModel(t, filepath.Join(ws.Backend, "src"))
	return ws, openGovernanceEntry(t, m, governanceLivingDocs, ScreenLivingDoc)
}

func requireNoSpecsWritten(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	for _, absent := range []string{filepath.Join(ws.Home, "INDEX.md"), filepath.Join(ws.Home, "specs")} {
		if _, err := os.Stat(absent); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe existir %s (stat = %v)", absent, err)
		}
	}
}

// REQ-17.10 y REQ-30.21: con un `baseline:` residual la pantalla no lo oculta ni
// lo convierte en un catálogo externo: lee openspec/specs/ como siempre y avisa
// del bloque retirado con la guía de migración.
func TestLivingDocScreenWarnsAboutAResidualBaseline(t *testing.T) {
	ws, state := livingDocsOf(t, livingResidualBaseline)

	for _, want := range []string{"baseline_retired", "axiom archive adopt scan"} {
		if !strings.Contains(state.GovernanceMessage, want) {
			t.Errorf("el mensaje de gobernanza no contiene %q:\n%s", want, state.GovernanceMessage)
		}
	}
	view := state.View()
	if !strings.Contains(view, "Especificaciones consolidadas en openspec/specs/") {
		t.Errorf("la pantalla conserva su presentación sobre openspec/specs/:\n%s", view)
	}
	requireNoSpecsWritten(t, ws)
}

// Sin baseline residual la pantalla conserva su presentación.
func TestLivingDocScreenWithoutAResidualBaselineKeepsItsPresentation(t *testing.T) {
	ws := workspacetest.Canonical(t)
	m := governanceModel(t, ws.Master)
	state := openGovernanceEntry(t, m, governanceLivingDocs, ScreenLivingDoc)
	if state.GovernanceMessage != "" {
		t.Errorf("sin baseline residual no hay aviso: mensaje=%q", state.GovernanceMessage)
	}
	view := state.View()
	if !strings.Contains(view, "No se encontraron especificaciones consolidadas") || !strings.Contains(view, "sincronizadas en INDEX.md") {
		t.Errorf("la pantalla por defecto debe conservar su texto:\n%s", view)
	}
}

// REQ-17.10 y REQ-30.21: la tecla `s` y la opción de sincronizar muestran el
// rechazo con su continuación y no escriben INDEX.md.
func TestLivingDocSyncShowsTheRejectionWithItsContinuation(t *testing.T) {
	tests := []struct {
		name  string
		wants []string
	}{
		{"baseline residual", []string{"baseline_retired", "axiom archive adopt scan"}},
	}
	for _, tt := range tests {
		for _, trigger := range []string{"tecla s", "opción sincronizar"} {
			t.Run(tt.name+" con "+trigger, func(t *testing.T) {
				ws, state := livingDocsOf(t, livingResidualBaseline)

				var updated tea.Model
				if trigger == "tecla s" {
					updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
				} else {
					// La opción de sincronizar va tras la lista de especificaciones.
					state.Cursor = len(state.LivingSpecs)
					updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
				}
				state = updated.(Model)

				if state.Screen != ScreenLivingDoc {
					t.Fatalf("pantalla = %v, se esperaba ScreenLivingDoc", state.Screen)
				}
				if !strings.HasPrefix(state.GovernanceMessage, "Error al sincronizar catálogo: ") {
					t.Errorf("mensaje = %q, se esperaba el rechazo de la sincronización", state.GovernanceMessage)
				}
				for _, want := range tt.wants {
					if !strings.Contains(state.GovernanceMessage, want) {
						t.Errorf("el mensaje no contiene %q:\n%s", want, state.GovernanceMessage)
					}
				}
				requireNoSpecsWritten(t, ws)
			})
		}
	}
}
