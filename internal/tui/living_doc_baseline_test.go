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
	livingCanonicalConfig = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"
	livingProjectBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: docs/04_Flujos_Negocio.md\n"
	livingInvalidBaseline = "baseline:\n  documents: []\n"
)

// livingDocsOf abre la pantalla de documentación viva de un espacio canónico con
// la baseline dada, desde el repositorio de código (donde no hay marcadores).
func livingDocsOf(t *testing.T, baseline string) (workspacetest.Workspace, Model) {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), livingCanonicalConfig+baseline)
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

// REQ-17.10 y REQ-24.25: con baseline del proyecto la pantalla no muestra el
// catálogo vacío como ausencia de especificación, sino el aviso de baseline
// externa con los dominios y las rutas declarados.
func TestLivingDocScreenShowsAnExternalBaseline(t *testing.T) {
	ws, state := livingDocsOf(t, livingProjectBaseline)

	if !state.LivingExternalBaseline || len(state.LivingSpecs) != 0 {
		t.Fatalf("LivingExternalBaseline = %v, LivingSpecs = %+v", state.LivingExternalBaseline, state.LivingSpecs)
	}
	for _, want := range []string{
		"baseline de especificaciones es externa",
		"modelo-de-datos → " + filepath.Join(ws.Specs, "03_Modelo_Datos.md"),
		"flujos-de-negocio → " + filepath.Join(ws.Specs, "docs", "04_Flujos_Negocio.md"),
	} {
		if !strings.Contains(state.GovernanceMessage, want) {
			t.Errorf("el mensaje de gobernanza no contiene %q:\n%s", want, state.GovernanceMessage)
		}
	}

	view := state.View()
	if !strings.Contains(view, "baseline de especificaciones es del proyecto") {
		t.Errorf("la pantalla debe indicar que la baseline es del proyecto:\n%s", view)
	}
	if strings.Contains(view, "No se encontraron especificaciones consolidadas") {
		t.Errorf("con baseline externa el catálogo vacío no significa ausencia de especificación:\n%s", view)
	}
	requireNoSpecsWritten(t, ws)
}

// Sin baseline del proyecto la pantalla conserva su presentación.
func TestLivingDocScreenWithoutAProjectBaselineKeepsItsPresentation(t *testing.T) {
	ws := workspacetest.Canonical(t)
	m := governanceModel(t, ws.Master)
	state := openGovernanceEntry(t, m, governanceLivingDocs, ScreenLivingDoc)
	if state.LivingExternalBaseline || state.GovernanceMessage != "" {
		t.Errorf("sin baseline externa no hay aviso: externa=%v, mensaje=%q", state.LivingExternalBaseline, state.GovernanceMessage)
	}
	view := state.View()
	if !strings.Contains(view, "No se encontraron especificaciones consolidadas") || !strings.Contains(view, "sincronizadas en INDEX.md") {
		t.Errorf("la pantalla por defecto debe conservar su texto:\n%s", view)
	}
}

// REQ-17.10 y REQ-24.23: la tecla `s` y la opción de sincronizar muestran el
// rechazo con su continuación y no escriben INDEX.md.
func TestLivingDocSyncShowsTheRejectionWithItsContinuation(t *testing.T) {
	tests := []struct {
		name     string
		baseline string
		wants    []string
	}{
		{"baseline del proyecto", livingProjectBaseline, []string{"baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`"}},
		{"baseline indeterminada", livingInvalidBaseline, []string{"no se puede determinar", "al menos un documento", "axiom workspace validate --path"}},
	}
	for _, tt := range tests {
		for _, trigger := range []string{"tecla s", "opción sincronizar"} {
			t.Run(tt.name+" con "+trigger, func(t *testing.T) {
				ws, state := livingDocsOf(t, tt.baseline)

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
