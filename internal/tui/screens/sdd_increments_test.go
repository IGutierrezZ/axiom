package screens

import (
	"testing"
)

// TestSDDIncrementsOptionsBadgeForPendingArchive fija la insignia de un cambio pendiente de archivar
// (INC-25, REQ-25.34): solo la lleva el marcado, y la longitud de las opciones no cambia.
func TestSDDIncrementsOptionsBadgeForPendingArchive(t *testing.T) {
	increments := []SDDIncrementInfo{
		{Name: "y", Type: "active", Phase: "spec", PendingArchive: true},
		{Name: "x", Type: "active", Phase: "spec"},
	}

	options := SDDIncrementsOptions(increments)

	if len(options) != len(increments)+4 {
		t.Fatalf("SDDIncrementsOptions() longitud = %d, esperado %d", len(options), len(increments)+4)
	}
	if want := "▶ y [SPEC] (0/0 tareas, 0%) · pendiente de archivar"; options[0] != want {
		t.Errorf("opción del pendiente = %q, esperado %q", options[0], want)
	}
	if want := "▶ x [SPEC] (0/0 tareas, 0%)"; options[1] != want {
		t.Errorf("opción sin pendiente = %q, esperado %q", options[1], want)
	}
}

// TestSDDIncrementsOptionsUnchangedByLaneHeader protege D-13: la cabecera
// de conmutación de carril es de solo renderizado y NO debe alterar la
// longitud ni el contenido de SDDIncrementsOptions, cuyo manejador en
// model.go (ScreenSDDIncrements) tiene un misrouting preexistente y ajeno
// a este incremento que una opción nueva convertiría en activo.
func TestSDDIncrementsOptionsUnchangedByLaneHeader(t *testing.T) {
	tests := []struct {
		name       string
		increments []SDDIncrementInfo
		wantTail   []string
	}{
		{
			name:       "sin incrementos: solo las 4 opciones globales",
			increments: nil,
			wantTail: []string{
				"+ Crear nuevo incremento SDD",
				"▶ Avanzar fase del cambio seleccionado (sdd continue)",
				"✓ Validar reporte de verificación (sdd verify-validate)",
				"Volver a gobernanza",
			},
		},
		{
			name: "con un incremento: 1 fila + 4 opciones globales",
			increments: []SDDIncrementInfo{
				{Name: "demo", Type: "active", Phase: "spec", TasksTotal: 2, TasksCompleted: 1, ProgressPct: 50},
			},
			wantTail: []string{
				"+ Crear nuevo incremento SDD",
				"▶ Avanzar fase del cambio seleccionado (sdd continue)",
				"✓ Validar reporte de verificación (sdd verify-validate)",
				"Volver a gobernanza",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := SDDIncrementsOptions(tt.increments)

			wantLen := len(tt.increments) + len(tt.wantTail)
			if len(options) != wantLen {
				t.Fatalf("SDDIncrementsOptions() longitud = %d, esperado %d: %v", len(options), wantLen, options)
			}

			got := options[len(options)-len(tt.wantTail):]
			for i, want := range tt.wantTail {
				if got[i] != want {
					t.Errorf("SDDIncrementsOptions() cola[%d] = %q, esperado %q", i, got[i], want)
				}
			}
		})
	}
}
