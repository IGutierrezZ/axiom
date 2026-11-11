package model

import (
	"strings"
	"testing"
)

func TestValidateCopilotPhaseAssignments(t *testing.T) {
	tests := []struct {
		name        string
		assignments map[string]CopilotPhaseAssignment
		wantErr     []string
	}{
		{name: "sin asignaciones", assignments: nil},
		{
			name: "válidas",
			assignments: map[string]CopilotPhaseAssignment{
				"orchestrator": {Model: "Claude Opus 4.6 (copilot)", Effort: "high"},
				"sdd-apply":    {Model: "GPT-5.2 (copilot)"},
			},
		},
		{
			name:        "fase desconocida",
			assignments: map[string]CopilotPhaseAssignment{"sdd-nope": {Model: "GPT-5.2 (copilot)"}, "default": {Model: "x"}},
			wantErr:     []string{"fases desconocidas: default, sdd-nope"},
		},
		{
			name:        "modelo vacío",
			assignments: map[string]CopilotPhaseAssignment{"sdd-apply": {Model: "   "}},
			wantErr:     []string{"sdd-apply: el modelo no puede estar vacío"},
		},
		{
			name:        "salto de línea en el modelo",
			assignments: map[string]CopilotPhaseAssignment{"sdd-apply": {Model: "GPT\ntools: []"}},
			wantErr:     []string{"sdd-apply", "caracteres de control"},
		},
		{
			name:        "salto de línea en el esfuerzo",
			assignments: map[string]CopilotPhaseAssignment{"sdd-apply": {Model: "GPT-5.2 (copilot)", Effort: "high\nx: y"}},
			wantErr:     []string{"sdd-apply", "caracteres de control"},
		},
		{
			name: "informa de todas las violaciones",
			assignments: map[string]CopilotPhaseAssignment{
				"sdd-apply": {Model: ""},
				"sdd-zzz":   {Model: "x"},
			},
			wantErr: []string{"sdd-apply: el modelo no puede estar vacío", "fases desconocidas: sdd-zzz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCopilotPhaseAssignments(tt.assignments)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("error inesperado: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			for _, fragment := range tt.wantErr {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("el error %q no contiene %q", err, fragment)
				}
			}
		})
	}
}

func TestCopilotPhaseNamesReturnsCopy(t *testing.T) {
	names := CopilotPhaseNames()
	if len(names) == 0 || names[0] != "orchestrator" {
		t.Fatalf("CopilotPhaseNames() = %v", names)
	}
	names[0] = "otra"
	if CopilotPhaseNames()[0] != "orchestrator" {
		t.Error("CopilotPhaseNames() debe devolver una copia")
	}
}
