package branch

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de `axiom branch scope` (REQ-25.51, D3). Todos se devuelven antes de calcular nada y
// sin mutar el repositorio.

// ScopeKickoffNotSealed es el rechazo `kickoff_not_sealed` de `scope` ante un cambio inexistente o
// sin `kickoff.yaml`: sin kickoff no hay instantánea de repositorios que calcular.
func ScopeKickoffNotSealed(change, cwd string) *Refusal {
	return &Refusal{
		Code:         CodeKickoffNotSealed,
		Message:      fmt.Sprintf("el cambio %s no existe o no tiene ningún kickoff.yaml sellado: sin kickoff no hay instantánea de repositorios sobre la que calcular el alcance", change),
		Continuation: fmt.Sprintf("comprueba el cambio con `axiom sdd status --cwd %s --change %s` y sella su kickoff con `axiom sdd kickoff seal` antes de repetir `axiom branch scope`", pathquote.Quote(cwd), pathquote.ShellWord(change)),
		Details:      map[string]string{"change": change},
	}
}

// ScopeNotApplicableOdd es el rechazo `scope_not_applicable` de un cambio del carril ODD: su
// documento existe y no tiene kickoff ni instantánea.
func ScopeNotApplicableOdd(change, cwd, document string) *Refusal {
	return &Refusal{
		Code:         CodeScopeNotApplicable,
		Message:      fmt.Sprintf("el cambio %s es del carril odd (su documento es %s) y `axiom branch scope` solo opera sobre un cambio SDD con kickoff sellado", change, document),
		Continuation: fmt.Sprintf("consulta las ramas del cambio con `axiom branch status --lane odd --change %s --cwd %s`", pathquote.ShellWord(change), pathquote.Quote(cwd)),
		Details:      map[string]string{"change": change, "lane": string(LaneODD)},
	}
}

// ScopeNotApplicableWithoutSnapshot es el rechazo `scope_not_applicable` de un kickoff sellado sin
// bloque `git:` (un cambio retro-sellado o anterior a la instantánea de política, REQ-25.36).
func ScopeNotApplicableWithoutSnapshot(change, cwd string) *Refusal {
	return &Refusal{
		Code:         CodeScopeNotApplicable,
		Message:      fmt.Sprintf("el kickoff del cambio %s se selló sin instantánea de repositorios (bloque git:), de modo que no hay repositorios cuyo alcance calcular", change),
		Continuation: fmt.Sprintf("consulta el estado del cambio con `axiom sdd status --cwd %s --change %s`", pathquote.Quote(cwd), pathquote.ShellWord(change)),
		Details:      map[string]string{"change": change},
	}
}
