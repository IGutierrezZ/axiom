package model

// VisualPolishComponents returns the complete managed visual-polish inventory.
// Cleanup flows use this inventory, which intentionally includes the generic theme
// even though presets do not install it.
func VisualPolishComponents() []ComponentID {
	return []ComponentID{ComponentTheme, ComponentClaudeTheme, ComponentOpenCodeGentleLogo}
}

// ComponentsForPreset returns the managed components implied by a preset/persona
// pair. Visual themes are retained as legacy cleanup IDs, never installed.
//
// GGA no forma parte de ningún preset predefinido: es opcional y solo se instala
// por selección explícita (`--component gga`, preset custom o la TUI). CodeGraph
// tampoco: es una herramienta de la comunidad que se elige aparte
// (Selection.CommunityTools) y nunca viene marcada por defecto.
func ComponentsForPreset(preset PresetID, persona PersonaID) []ComponentID {
	var components []ComponentID
	switch preset {
	case PresetMinimal:
		components = []ComponentID{ComponentEngram}
	case PresetEcosystemOnly:
		components = []ComponentID{ComponentEngram, ComponentSDD, ComponentSkills, ComponentContext7}
	case PresetCustom:
		return nil
	default: // full-gentleman
		components = []ComponentID{
			ComponentEngram,
			ComponentSDD,
			ComponentSkills,
			ComponentContext7,
			ComponentPermission,
		}
	}
	if persona != PersonaCustom {
		components = append(components, ComponentPersona)
	}
	return components
}
