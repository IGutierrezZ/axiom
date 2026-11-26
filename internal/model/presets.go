package model

// ComponentsForPreset returns the managed components implied by a preset.
//
// CodeGraph no forma parte de ningún preset predefinido: es una herramienta de
// la comunidad que se elige aparte (Selection.CommunityTools) y nunca viene
// marcada por defecto.
func ComponentsForPreset(preset PresetID) []ComponentID {
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
	return append(components, ComponentPersona)
}
