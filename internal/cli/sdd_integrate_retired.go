package cli

import (
	"fmt"
	"io"
	"strings"
)

// SDDIntegrateRetiredError es la negativa tipada de `axiom sdd integrate`, retirado
// con la baseline ajena (REQ-30.25). Nombra los verbos que lo sustituyen y no
// escribe nada: el contexto técnico se integra con `axiom archive context` y una
// baseline residual se migra con `axiom archive adopt`.
type SDDIntegrateRetiredError struct {
	// Subcommand es el subcomando que se pidió (check, prepare, verify, show), si lo hubo.
	Subcommand string
}

// Error nombra el verbo retirado y sus sustitutos.
func (e *SDDIntegrateRetiredError) Error() string {
	requested := "axiom sdd integrate"
	if e.Subcommand != "" {
		requested += " " + e.Subcommand
	}
	return fmt.Sprintf("%s ya no existe: se retiró con la baseline ajena y no se ha escrito nada. Para el contexto técnico usa `axiom archive context check|prepare|verify|show --path <ruta> --change <cambio>` y, para migrar requisitos o un `baseline:` residual, `axiom archive adopt scan --path <ruta> --change <cambio>`", requested)
}

// RunSDDIntegrate es el punto de entrada retirado de `axiom sdd integrate`: nunca
// escribe y siempre termina con *SDDIntegrateRetiredError (la ayuda lo explica con
// código 0).
func RunSDDIntegrate(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		_, _ = fmt.Fprintln(stdout, "axiom sdd integrate está retirado (REQ-30.25): se retiró con la baseline ajena.")
		_, _ = fmt.Fprintln(stdout, "  Contexto técnico: axiom archive context check|prepare|verify|show --path <ruta> --change <cambio>")
		_, _ = fmt.Fprintln(stdout, "  Migración de requisitos o de un `baseline:` residual: axiom archive adopt scan --path <ruta> --change <cambio>")
		return nil
	}
	retired := &SDDIntegrateRetiredError{}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		retired.Subcommand = args[0]
	}
	return retired
}
