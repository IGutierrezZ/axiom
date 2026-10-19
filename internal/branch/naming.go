package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Validación de nombres (REQ-25.3, REQ-25.11, design D-04). Los rechazos distinguen
// el origen del valor: lo que viene de la invocación (`--change`, `--prefix`,
// `--branch`) o del kickoff (roles) es `invalid_name`; lo que viene de la política de
// axiom.yaml o del hub es `config_invalid` (la carga ya lo rechaza, y esta es la red
// de seguridad de quien construye una política a mano). Todo se comprueba antes de
// crear nada.

// archiveRole es el nombre de rol que colisionaría con la rama de archivado
// `<prefijo>/<cambio>-archive` (REQ-25.4, supuesto S-15).
const archiveRole = "archive"

// ValidateChangeName rechaza con `invalid_name` un nombre de cambio que no es un
// componente válido a la vez como rama git y como directorio de Windows. operation
// es la orden de `axiom branch` que se repite en la continuación y dir el directorio
// de partida que esta cita. El motor lo comprueba antes de usar el nombre para
// componer una ruta del disco.
func ValidateChangeName(operation, dir, change string) *Refusal {
	if err := gitref.ValidBranchComponent(change); err != nil {
		return invalidName("--change", "el nombre del cambio", change, err,
			fmt.Sprintf("repite la orden con otro nombre de cambio, que use letras, dígitos, «.», «_» y «-»: `axiom branch %s --change <cambio> --cwd %s`", operation, pathquote.Quote(dir)))
	}
	return nil
}

// validateRoleName rechaza con `invalid_name` un rol cuyo nombre no puede formar
// parte de una rama ni de un directorio (`<prefijo>/<cambio>-<rol>`,
// `<raíz>/<cambio>-<rol>/<repo>`), o que se llama `archive`: colisionaría con la
// rama de archivado del cambio (supuesto S-15). Solo se comprueba para los roles que
// generan rama de rol.
func validateRoleName(in PlanInput, role, changeBranch string) *Refusal {
	continuation := fmt.Sprintf("renombra el rol en axiom.yaml (roles.<rol>) y en el kickoff del cambio, y comprueba con `axiom branch %s --change %s --cwd %s`",
		in.Operation, in.Change, pathquote.Quote(in.Master))
	if strings.EqualFold(role, archiveRole) {
		return invalidName("role", "el nombre del rol", role,
			fmt.Errorf("colisiona con la rama de archivado %s-%s del cambio", changeBranch, archiveRole), continuation)
	}
	if err := gitref.ValidBranchComponent(role); err != nil {
		return invalidName("role", "el nombre del rol", role, err, continuation)
	}
	return nil
}

// validateExceptions comprueba los valores de las excepciones por cambio que son
// nombres: un `--prefix` o un `--branch` inválidos de la invocación son
// `invalid_name`. `--base` no es un nombre sino una referencia, y la resuelve git.
func validateExceptions(in PlanInput) *Refusal {
	if prefix := in.Exceptions.Prefix; prefix != "" {
		if err := gitref.ValidPrefix(prefix); err != nil {
			return invalidName("--prefix", "el prefijo indicado con --prefix", prefix, err,
				fmt.Sprintf("repite la orden con un prefijo válido: `axiom branch %s --change %s --prefix <prefijo> --cwd %s`",
					in.Operation, in.Change, pathquote.Quote(in.Master)))
		}
	}
	if name := in.Exceptions.Branch; name != "" {
		if err := gitref.ValidRefName(name); err != nil {
			return invalidName("--branch", "el nombre de rama indicado con --branch", name, err,
				fmt.Sprintf("repite la orden con un nombre de rama válido: `axiom branch %s --change %s --branch <rama> --cwd %s`",
					in.Operation, in.Change, pathquote.Quote(in.Master)))
		}
	}
	return nil
}

// validatePolicyValue rechaza con `config_invalid` un valor de la política de ramas
// que no es un nombre válido. field es la clave de axiom.yaml (`integration_branch` o
// `branch_prefix`) y repository, el repositorio al que se aplica.
func validatePolicyValue(in PlanInput, repository, field, value string, validate func(string) error) *Refusal {
	err := validate(value)
	if err == nil {
		return nil
	}
	return &Refusal{
		Code:         CodeConfigInvalid,
		Message:      fmt.Sprintf("la política de ramas del repositorio %s tiene un %s «%s» no válido: %v", repository, field, value, err),
		Continuation: validateConfigContinuation(in.Master),
		Details:      map[string]string{"field": field, "repository": repository},
	}
}

// invalidName compone el rechazo `invalid_name`: field nombra el argumento o el
// elemento inválido y value, su valor, ambos también en Details para quien lee el
// JSON por máquina.
func invalidName(field, what, value string, cause error, continuation string) *Refusal {
	return &Refusal{
		Code:         CodeInvalidName,
		Message:      fmt.Sprintf("%s «%s» no es válido: %v", what, value, cause),
		Continuation: continuation,
		Details:      map[string]string{"field": field, "value": value},
	}
}
