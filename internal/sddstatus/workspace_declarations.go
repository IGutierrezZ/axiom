package sddstatus

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// decorateWorkspaceDeclarations proyecta sobre el status lo que la resolución
// del workspace (workspace.Locate) tiene que comunicar: los avisos de Location
// (un axiom.yaml directo que no se pudo cargar y un planning home divergente de
// <maestra>/openspec) y las declaraciones válidas de axiom.yaml, `baseline:` y
// `technical_context:`, como campos aditivos y opcionales (D-05, REQ-24.19,
// REQ-24.28). Es un decorador que actúa fuera de baseStatus, de modo que la
// firma que usa status_v2_clean_break_test no cambia.
//
// Un bloque inválido no se proyecta, ni parcial ni por defecto (REQ-24.18), y sin
// declaración las dos claves no existen en el JSON: un proyecto sin axiom.yaml, o
// con uno que no declara los bloques, produce el mismo status que antes.
//
// Un aviso es informativo: nunca modifica la ruta recomendada ni las
// dependencias. Si el status ya lleva instrucciones de fase, se vuelven a
// generar para que reflejen lo añadido.
func decorateWorkspaceDeclarations(status *Status, location workspace.Location) {
	status.Baseline = projectBaseline(location)
	status.TechnicalContext = projectTechnicalContext(location)

	if len(location.Notes) == 0 {
		return
	}
	status.Notes = append(status.Notes, location.Notes...)
	if status.PhaseInstructions != nil {
		instructions := renderPhaseInstructions(*status)
		status.PhaseInstructions = &instructions
	}
}

// projectBaseline devuelve la proyección de la baseline declarada, o nil si no
// hay declaración o no es válida. Las rutas son absolutas: las declaradas son
// relativas a workspaceRoot.
func projectBaseline(location workspace.Location) *BaselineProjection {
	declarations := location.Declarations
	if declarations.BaselineErr != nil || declarations.Baseline == nil {
		return nil
	}
	documents := make([]BaselineDocumentProjection, 0, len(declarations.Baseline.Documents))
	for _, document := range declarations.Baseline.Documents {
		documents = append(documents, BaselineDocumentProjection{
			Domain:      document.Domain,
			Path:        absoluteDeclaredPath(location.WorkspaceRoot, document.Path),
			Integration: string(document.EffectiveIntegration()),
		})
	}
	return &BaselineProjection{Documents: documents}
}

// projectTechnicalContext devuelve la proyección del contexto técnico declarado,
// o nil si no hay declaración o no es válida. Un modo no declarado no aparece.
func projectTechnicalContext(location workspace.Location) *TechnicalContextProjection {
	declarations := location.Declarations
	if declarations.TechnicalContextErr != nil || declarations.TechnicalContext == nil {
		return nil
	}
	projectMode := func(mode *workspace.ContextMode) *ContextModeProjection {
		if mode == nil {
			return nil
		}
		projected := &ContextModeProjection{Paths: make([]string, 0, len(mode.Paths))}
		for _, declared := range mode.Paths {
			projected.Paths = append(projected.Paths, absoluteDeclaredPath(location.WorkspaceRoot, declared))
		}
		if strings.TrimSpace(mode.Entry) != "" {
			projected.Entry = absoluteDeclaredPath(location.WorkspaceRoot, mode.Entry)
		}
		return projected
	}
	return &TechnicalContextProjection{
		Maintained: projectMode(declarations.TechnicalContext.Maintained),
		Reference:  projectMode(declarations.TechnicalContext.Reference),
	}
}

// absoluteDeclaredPath resuelve una ruta declarada, relativa a workspaceRoot y
// con «/» o «\» como separadores, a una ruta absoluta y limpia del sistema.
func absoluteDeclaredPath(workspaceRoot, declared string) string {
	slashed := strings.ReplaceAll(strings.TrimSpace(declared), `\`, "/")
	return filepath.Clean(filepath.Join(workspaceRoot, filepath.FromSlash(slashed)))
}
