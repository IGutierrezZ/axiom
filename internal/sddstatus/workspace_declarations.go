package sddstatus

import "github.com/IGutierrezZ/axiom/v3/internal/workspace"

// decorateWorkspaceDeclarations proyecta sobre el status lo que la resolución
// del workspace (workspace.Locate) tiene que comunicar y no bloquea: en esta
// unidad, los avisos de Location (un axiom.yaml directo que no se pudo cargar y
// un planning home divergente de <maestra>/openspec). Es un decorador que
// actúa fuera de baseStatus, de modo que la firma que usa
// status_v2_clean_break_test no cambia.
//
// Un aviso es informativo: nunca modifica la ruta recomendada ni las
// dependencias. Si el status ya lleva instrucciones de fase, se vuelven a
// generar para que incluyan los avisos añadidos.
func decorateWorkspaceDeclarations(status *Status, location workspace.Location) {
	if len(location.Notes) == 0 {
		return
	}
	status.Notes = append(status.Notes, location.Notes...)
	if status.PhaseInstructions != nil {
		instructions := renderPhaseInstructions(*status)
		status.PhaseInstructions = &instructions
	}
}
