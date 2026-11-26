package model

import (
	"slices"
	"testing"
)

func TestExistingPresetsDoNotImplyCommunityTools(t *testing.T) {
	for _, preset := range []PresetID{PresetFullGentleman, PresetEcosystemOnly, PresetMinimal, PresetCustom} {
		t.Run(string(preset), func(t *testing.T) {
			selection := Selection{
				Preset:     preset,
				Components: ComponentsForPreset(preset, PersonaGentleman),
			}
			if len(selection.CommunityTools) != 0 {
				t.Fatalf("preset %q community tools = %v, want none", preset, selection.CommunityTools)
			}
		})
	}
}

func TestFullGentlemanKeepsPermissionComponent(t *testing.T) {
	if got := ComponentsForPreset(PresetFullGentleman, PersonaAxiom); !slices.Contains(got, ComponentPermission) {
		t.Fatalf("full-gentleman debe conservar el componente de permisos (solo deny): %v", got)
	}
}
