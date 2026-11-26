package model

// Fixtures de prueba: los presets de Codex ya no tienen otro llamador que las
// pruebas (la TUI que los ofrecía se retiró), así que viven solo en _test.go.

// CodexModelPresetPowerful devuelve el preset Powerful.
func CodexModelPresetPowerful() map[string]CodexEffort {
	return codexPresetEfforts(string(CodexPresetPowerful))
}

// CodexModelPresetLowCost devuelve el preset Low-cost.
func CodexModelPresetLowCost() map[string]CodexEffort {
	return codexPresetEfforts(string(CodexPresetLowCost))
}

// CodexPresetOrchestratorAssignment devuelve la política de sesión principal de un preset.
func CodexPresetOrchestratorAssignment(preset string) *CodexOrchestratorAssignment {
	assignment, ok := codexPresetOrchestrator[CodexPresetKey(preset)]
	if !ok {
		assignment = codexPresetOrchestrator[CodexPresetRecommended]
	}
	return &assignment
}
