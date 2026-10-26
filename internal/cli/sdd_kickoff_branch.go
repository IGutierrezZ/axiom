package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// La guarda A de `axiom sdd kickoff seal` (REQ-25.38, design D-11 y D-12): antes de sellar,
// sin `--infer`, se consulta el veredicto de rama del repositorio del planning home. En un
// monorepo embebido exige estar en la rama del cambio o en una de rol de los roles que se van
// a sellar; en un multirepo, que el repositorio de specs esté en su integración (los de código
// no se comprueban al sellar). Ante `wrong_branch` se niega sin escribir `kickoff.yaml`, con el
// error tipado y la orden exacta de `axiom branch ensure`.

// sealBranchWarnings recibe los avisos de la guarda (ruta fuera de git, `HEAD` ilegible): la
// guarda no bloquea lo que no puede comprobar, pero lo dice. Es una variable para que las
// pruebas los recojan.
var sealBranchWarnings io.Writer = os.Stderr

// requireSealBranch aplica la guarda A al sellado de sealing en changeRoot.
func requireSealBranch(changeRoot string, sealing kickoff.Kickoff) error {
	roles := make([]string, 0, len(sealing.Config.Roles))
	for _, role := range sealing.Config.Roles {
		roles = append(roles, role.Role)
	}
	warning, err := engine.Default().RequireChangeBranch(context.Background(), engine.WriteTarget{
		ChangeRoot: changeRoot,
		Change:     sealing.Change,
		Lane:       branch.LaneSDD,
		Context:    branch.ContextOrdinary,
		Op:         "kickoff seal",
		SealRoles:  roles,
	})
	if warning != "" {
		_, _ = fmt.Fprintln(sealBranchWarnings, "Aviso: "+warning)
	}
	return err
}
