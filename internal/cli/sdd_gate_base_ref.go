package cli

import (
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// completeGateBaseRef fija la rama contra la que se comprueba la evidencia `pr_merged` cuando la
// invocación no trae --base-ref (design D-22, REQ-25.23): la rama de integración resuelta del
// repositorio del planning home, no un `main` fijo. `kickoff.ParseGateRecordArgs` no la conoce y
// deja el campo vacío; solo `pr_merged` tiene comprobación de ancestro, así que el resto de
// evidencias conserva el campo vacío como antes. Un --base-ref explícito no se toca.
func completeGateBaseRef(parsed kickoff.GateRecordArgs, loc workspace.Location) kickoff.GateRecordArgs {
	if parsed.EvidenceKind == kickoff.EvidencePRMerged && parsed.BaseRef == "" {
		parsed.BaseRef = engine.PlanningIntegration(loc)
	}
	return parsed
}
