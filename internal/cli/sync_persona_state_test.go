package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// TestRunSyncWarnsAboutRetiredPersonaInState verifica el lector tolerante: un
// estado con una persona retirada carga, sync avisa y la escritura posterior del
// estado ya no conserva la clave. "axiom" no avisa.
func TestRunSyncWarnsAboutRetiredPersonaInState(t *testing.T) {
	for _, tc := range []struct {
		persona  string
		wantWarn bool
	}{
		{persona: "gentleman", wantWarn: true},
		{persona: "neutral", wantWarn: true},
		{persona: "custom", wantWarn: true},
		{persona: "axiom", wantWarn: false},
	} {
		t.Run(tc.persona, func(t *testing.T) {
			home := t.TempDir()
			setSyncTestHome(t, home)
			notice := captureWorkspaceAgentsNotice(t)
			if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
			mustWriteFile(t, state.Path(home), []byte(`{"installed_agents":["claude-code"],"persona":"`+tc.persona+`"}`))

			if _, err := RunSync([]string{"--agents", "claude-code", "--sdd-mode", "single"}); err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}

			warned := strings.Contains(notice.String(), "la persona \""+tc.persona+"\" guardada en el estado ya no existe")
			if warned != tc.wantWarn {
				t.Fatalf("aviso de persona retirada = %t, want %t; salida: %q", warned, tc.wantWarn, notice.String())
			}
			if got := readTextFile(t, state.Path(home)); strings.Contains(got, `"persona"`) {
				t.Fatalf("la escritura del estado conserva la persona retirada:\n%s", got)
			}
		})
	}
}

// TestRunSyncWithSelectionPiWritesAxiomPersona comprueba que la persona de Pi es
// siempre la de Axiom y que no se toca la configuración del proyecto.
func TestRunSyncWithSelectionPiWritesAxiomPersona(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	t.Chdir(workspace)
	piPath := filepath.Join(workspace, ".pi", "gentle-ai", "persona.json")
	mustWriteFile(t, piPath, []byte("{\n  \"mode\": \"gentleman\"\n}\n"))
	mustWriteFile(t, state.Path(home), []byte(`{"installed_agents":["pi"]}`))

	if _, err := RunSyncWithSelection(home, model.Selection{
		Agents:     []model.AgentID{model.AgentPi},
		Components: []model.ComponentID{model.ComponentPersona},
	}); err != nil {
		t.Fatalf("RunSyncWithSelection() error = %v", err)
	}
	homePiPath := filepath.Join(home, ".pi", "gentle-ai", "persona.json")
	if got, want := readTextFile(t, homePiPath), "{\n  \"mode\": \"axiom\"\n}\n"; got != want {
		t.Fatalf("global Pi persona config = %q, want %q", got, want)
	}
	if got, want := readTextFile(t, piPath), "{\n  \"mode\": \"gentleman\"\n}\n"; got != want {
		t.Fatalf("workspace Pi persona config changed = %q, want %q", got, want)
	}
}

// TestRunSyncWarnsAboutRetiredPresetInState verifica el lector tolerante: un
// estado con un preset retirado carga, sync avisa y la escritura posterior del
// estado ya no conserva la clave ni deja de sincronizar la selección guardada.
func TestRunSyncWarnsAboutRetiredPresetInState(t *testing.T) {
	for _, preset := range []string{"full-gentleman", "ecosystem-only", "minimal", "custom"} {
		t.Run(preset, func(t *testing.T) {
			home := t.TempDir()
			setSyncTestHome(t, home)
			notice := captureWorkspaceAgentsNotice(t)
			if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
			mustWriteFile(t, state.Path(home), []byte(`{"installed_agents":["claude-code"],"selection_configured":true,"components":["persona"],"preset":"`+preset+`"}`))

			result, err := RunSync([]string{"--agents", "claude-code", "--sdd-mode", "single"})
			if err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}

			if want := "el preset \"" + preset + "\" guardado en el estado ya no existe"; !strings.Contains(notice.String(), want) {
				t.Fatalf("falta el aviso %q; salida: %q", want, notice.String())
			}
			if !reflect.DeepEqual(result.Selection.Components, []model.ComponentID{model.ComponentPersona}) {
				t.Fatalf("componentes sincronizados = %v, want [persona]", result.Selection.Components)
			}
			if got := readTextFile(t, state.Path(home)); strings.Contains(got, `"preset"`) {
				t.Fatalf("la escritura del estado conserva el preset retirado:\n%s", got)
			}
		})
	}
}
