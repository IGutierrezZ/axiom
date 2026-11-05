package tui

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	tea "github.com/charmbracelet/bubbletea"
)

// View y optionCount se ejecutan en cada fotograma y pulsación: deben reutilizar
// la disponibilidad de motores detectada al construir el modelo, sin consultar el
// PATH.
func TestWelcomeRenderDoesNotReprobeAgentBuilderEngines(t *testing.T) {
	probes := 0
	original := agentBuilderEngineAvailableFn
	agentBuilderEngineAvailableFn = func(id model.AgentID) bool {
		probes++
		return id == model.AgentClaudeCode
	}
	t.Cleanup(func() { agentBuilderEngineAvailableFn = original })

	m := NewModel(system.DetectionResult{}, "test")
	afterConstruction := probes
	if afterConstruction == 0 {
		t.Fatal("NewModel no detectó los motores del constructor de agentes")
	}

	var current tea.Model = m
	for range 5 {
		_ = current.View()
		_ = current.(Model).optionCount()
		current, _ = current.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if probes != afterConstruction {
		t.Fatalf("consultas al PATH tras construir = %d, want 0 (render y navegación deben reutilizar la detección)", probes-afterConstruction)
	}
	if !current.(Model).hasAgentBuilderEngines() {
		t.Fatal("hasAgentBuilderEngines() = false, want true por el motor claude detectado")
	}
}

// Crear un agente es una acción explícita: vuelve a detectar y recoge un motor
// instalado después de arrancar la TUI.
func TestCreateAgentRefreshesAgentBuilderEngines(t *testing.T) {
	available := false
	original := agentBuilderEngineAvailableFn
	agentBuilderEngineAvailableFn = func(id model.AgentID) bool {
		return available && id == model.AgentCodex
	}
	t.Cleanup(func() { agentBuilderEngineAvailableFn = original })

	m := NewModel(system.DetectionResult{}, "test")
	if m.hasAgentBuilderEngines() {
		t.Fatal("hasAgentBuilderEngines() = true antes de instalar ningún motor")
	}

	available = true
	m.Screen = ScreenWelcome
	m.Cursor = 5 // «Crear agente personalizado»
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)

	if got.Screen != ScreenAgentBuilderEngine {
		t.Fatalf("screen = %v, want ScreenAgentBuilderEngine tras refrescar", got.Screen)
	}
	if len(got.AgentBuilder.AvailableEngines) != 1 || got.AgentBuilder.AvailableEngines[0] != model.AgentCodex {
		t.Fatalf("AvailableEngines = %v, want [codex]", got.AgentBuilder.AvailableEngines)
	}
}
