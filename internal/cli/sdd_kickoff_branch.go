package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// La política de ramas de `axiom sdd kickoff seal` (REQ-25.36, REQ-25.37, REQ-25.38, design D-11
// y D-12). Sin `--infer`, y por este orden:
//
//  1. la guarda A: se consulta el veredicto de rama del repositorio del planning home. En un
//     monorepo embebido exige estar en la rama del cambio o en una de rol de los roles que se
//     van a sellar; en un multirepo, que el repositorio de specs esté en su integración (los de
//     código no se comprueban al sellar). Ante `wrong_branch` se niega sin escribir
//     `kickoff.yaml`, con el error tipado y la orden exacta de `axiom branch ensure`;
//  2. la instantánea: `engine.Snapshot` calcula la política git del cambio y entra en el mismo
//     `kickoff.Seal` que escribe el resto de la configuración, que la publica una sola vez.
//
// `--infer` no pasa por aquí: ni comprueba la rama ni escribe instantánea, de modo que el cambio
// retro-sellado queda exento de las guardas de rama.

// sealBranchWarnings recibe los avisos de la guarda (ruta fuera de git, `HEAD` ilegible): la
// guarda no bloquea lo que no puede comprobar, pero lo dice. Es una variable para que las
// pruebas los recojan.
var sealBranchWarnings io.Writer = os.Stderr

// applySealBranchPolicy aplica la guarda A al sellado de sealing en changeRoot y, si la rama es la
// que corresponde, le añade la instantánea de política git. Solo se calcula la instantánea
// cuando la guarda comprobó de verdad el repositorio: sin repositorio git o con un `HEAD` ilegible
// (aviso) y con un kickoff que ya existe (`kickoff.Seal` devolverá al ganador sin reescribirlo) no
// hay nada que escribir.
func applySealBranchPolicy(loc workspace.Location, changeRoot string, sealing *kickoff.Kickoff) error {
	roles := make([]string, 0, len(sealing.Config.Roles))
	for _, role := range sealing.Config.Roles {
		roles = append(roles, role.Role)
	}
	ctx := context.Background()
	branches := engine.Default()
	warning, err := branches.RequireChangeBranch(ctx, engine.WriteTarget{
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
	if err != nil || warning != "" {
		return err
	}
	if existing, loadErr := kickoff.Load(changeRoot); loadErr != nil || existing != nil {
		return loadErr
	}
	snapshot, err := branches.Snapshot(ctx, loc, sealing.Change, roles)
	if err != nil {
		return err
	}
	sealing.Git = snapshot
	return nil
}
