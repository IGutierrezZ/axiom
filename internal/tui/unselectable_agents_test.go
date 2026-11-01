package tui

import (
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/tui/screens"
)

// retiredAgents se conservan en el código pero no se pueden elegir al instalar.
var retiredAgents = []model.AgentID{model.AgentQwenCode, model.AgentHermes, model.AgentKimi}

// La pantalla de selección de agentes no lista Qwen Code, Hermes ni Kimi, y sí el
// resto; la de desinstalación los sigue listando.
func TestAgentSelectionScreenOmitsRetiredAgents(t *testing.T) {
	options := screens.AgentOptions()
	for _, id := range retiredAgents {
		if slices.Contains(options, id) {
			t.Errorf("la selección de agentes no debe listar %s: %v", id, options)
		}
	}
	if !slices.Contains(options, model.AgentClaudeCode) || !slices.Contains(options, model.AgentCodex) {
		t.Errorf("la selección debe conservar los agentes seleccionables: %v", options)
	}

	var uninstall []model.AgentID
	for _, agent := range screens.UninstallAgentOptions() {
		uninstall = append(uninstall, agent.ID)
	}
	for _, id := range retiredAgents {
		if !slices.Contains(uninstall, id) {
			t.Errorf("la desinstalación debe seguir listando %s", id)
		}
	}
}

// Ni la detección por carpetas ni el estado persistido preseleccionan un agente
// retirado: la pantalla no lo muestra y no debe instalarse a escondidas.
func TestPreselectedAgentsExcludesRetiredAgents(t *testing.T) {
	detected := preselectedAgents(makeDetectionWithAgents("qwen-code", "hermes", "claude-code"), state.InstallState{})
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(detected, want) {
		t.Errorf("preselectedAgents(detección) = %v, want %v", detected, want)
	}

	// Solo agentes retirados detectados: se cae al catálogo seleccionable.
	fallback := preselectedAgents(makeDetectionWithAgents("qwen-code", "hermes"), state.InstallState{})
	if len(fallback) == 0 {
		t.Fatal("sin agentes seleccionables detectados debe usarse el catálogo seleccionable")
	}
	for _, id := range retiredAgents {
		if slices.Contains(fallback, id) {
			t.Errorf("el catálogo por defecto no debe incluir %s: %v", id, fallback)
		}
	}

	persisted := state.InstallState{InstalledAgents: []string{"kimi", "claude-code", "qwen-code", "hermes"}, SelectionConfigured: true}
	got := preselectedAgents(makeDetectionWithAgents(), persisted)
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(got, want) {
		t.Errorf("preselectedAgents(estado) = %v, want %v", got, want)
	}
}
