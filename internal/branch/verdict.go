package branch

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// El veredicto único de rama (REQ-25.18, design D-10) es una función pura: a partir del papel
// del repositorio, el contexto de la consulta, los nombres esperados y la rama en la que está
// el repositorio responde «¿esta rama es la que corresponde a este cambio?» con `correct`,
// `wrong_branch` o `not_applicable`. No lee el disco ni lanza nada: obtener la rama actual y
// los nombres esperados es cosa del motor (engine.VerdictFor, solo con lecturas de ficheros),
// de modo que las reglas se prueban con tablas.

// ExpectedNames son los nombres de rama que el motor deriva de la instantánea, del registro o
// de la política (design D-10, «Nombres esperados»).
type ExpectedNames struct {
	// Change es la rama del cambio.
	Change string
	// Roles son los nombres completos de las ramas de rol del cambio; en el contexto ordinario
	// cualquiera de ellas es correcta además de la del cambio.
	Roles []string
	// Archive es la rama de archivado; vacía donde no existe (el carril ODD).
	Archive string
	// Integration es la rama de integración del repositorio.
	Integration string
}

// VerdictInput es todo lo que Decide necesita.
type VerdictInput struct {
	Role       RepoRole
	Repo       string // nombre del repositorio, para el mensaje
	Change     string
	Lane       Lane
	Context    Context
	Consultant Consultant
	Expected   ExpectedNames
	Current    string // rama actual; vacía con HEAD desacoplado
	Detached   bool
	// Exempt es el motivo de una exención (cambio sin instantánea, sumidero acumulativo, ruta
	// fuera de git…): con él el veredicto es `not_applicable` sea cual sea la rama.
	Exempt string
	// Master es la carpeta maestra del checkout principal que cita la continuación con `--cwd`;
	// vacía, la continuación deja la plantilla `<ruta>`.
	Master string
}

// Verdict es la respuesta del veredicto. Los tokens de Kind llevan su etiqueta en Label.
type Verdict struct {
	Kind     VerdictKind
	Repo     string
	Current  string
	Detached bool
	// Expected son las ramas correctas, la del cambio primero.
	Expected []string
	// Detail explica el veredicto en castellano (el porqué de un wrong_branch o de una
	// exención).
	Detail string
	// Warning lo rellena el motor cuando no pudo comprobar la rama y no bloquea.
	Warning string
	// Continuation es la orden exacta de `axiom branch ensure` que pone el repositorio en la rama
	// que corresponde; solo la lleva `wrong_branch`.
	Continuation string
}

// Decide calcula el veredicto de rama (REQ-25.18). Las reglas, por papel del repositorio:
//
//   - `change_branch`, contexto ordinario: la rama del cambio o cualquiera de rol; cualquier otra
//     (la de integración, la de archivado, una ajena y un HEAD desacoplado) es `wrong_branch`;
//   - `change_branch`, contexto de archivado: solo la rama de archivado;
//   - `specs_in_place`: solo la rama de integración, también en el contexto de archivado;
//   - `code`: la rama del cambio o de rol con el consultante `apply`, y `not_applicable` con
//     cualquier otro: un repositorio de código de un multirepo solo se evalúa al aplicar.
//
// El carril ODD no tiene archive: su contexto de archivado se evalúa como el ordinario. Una
// exención gana a todo lo demás. Decide no altera su entrada ni comparte memoria con ella.
func Decide(in VerdictInput) Verdict {
	verdict := Verdict{Kind: VerdictNotApplicable, Repo: in.Repo, Current: in.Current, Detached: in.Detached}
	switch {
	case in.Exempt != "":
		verdict.Detail = in.Exempt
		return verdict
	case in.Role == RepoCode && in.Consultant != ConsultantApply:
		verdict.Detail = fmt.Sprintf("el repositorio de código %s de un multirepo solo se evalúa al aplicar una tarea que lo toca", in.Repo)
		return verdict
	}

	archive := in.Context == ContextArchive && in.Lane != LaneODD && in.Role == RepoChangeBranch
	verdict.Expected = correctBranches(in, archive)
	if !in.Detached && slices.Contains(verdict.Expected, in.Current) {
		verdict.Kind = VerdictCorrect
		return verdict
	}
	verdict.Kind = VerdictWrongBranch
	verdict.Detail = wrongBranchDetail(in, verdict.Expected, archive)
	verdict.Continuation = verdictContinuation(in, archive)
	return verdict
}

// correctBranches son las ramas correctas de la consulta, la del cambio primero. Siempre es una
// copia: el veredicto no comparte memoria con la entrada.
func correctBranches(in VerdictInput, archive bool) []string {
	names := in.Expected
	switch {
	case in.Role == RepoSpecsInPlace:
		return []string{names.Integration}
	case archive:
		if names.Archive == "" {
			return nil
		}
		return []string{names.Archive}
	}
	return slices.Concat([]string{names.Change}, names.Roles)
}

// wrongBranchDetail explica un wrong_branch nombrando el repositorio, la rama actual (o el HEAD
// desacoplado) y la esperada.
func wrongBranchDetail(in VerdictInput, expected []string, archive bool) string {
	where := fmt.Sprintf("el repositorio %s está en la rama «%s»", in.Repo, in.Current)
	if in.Detached {
		where = fmt.Sprintf("el repositorio %s tiene el HEAD desacoplado (no está en ninguna rama)", in.Repo)
	}
	switch {
	case in.Role == RepoSpecsInPlace:
		return fmt.Sprintf("%s y el repositorio de specs de un multirepo se trabaja in situ en su rama de integración «%s»", where, in.Expected.Integration)
	case archive && len(expected) == 0:
		return fmt.Sprintf("%s y el cambio %s no tiene rama de archivado conocida", where, in.Change)
	case archive:
		return fmt.Sprintf("%s y el archivado del cambio %s se hace en su rama de archivado «%s»", where, in.Change, expected[0])
	case len(expected) > 1:
		return fmt.Sprintf("%s y el cambio %s se trabaja en la rama «%s» o en una de sus ramas de rol (%s)", where, in.Change, expected[0], strings.Join(expected[1:], ", "))
	}
	return fmt.Sprintf("%s y el cambio %s se trabaja en la rama «%s»", where, in.Change, expected[0])
}

// verdictContinuation es la orden exacta que pone el repositorio en la rama que corresponde
// (design D-10, «Continuación»): `axiom branch ensure --cwd <maestra> --change <c>`, con
// `--lane odd` en el carril ODD y `--archive` en el contexto de archivado de un repositorio con
// rama de cambio. La carpeta maestra va citada con pathquote.Quote.
func verdictContinuation(in VerdictInput, archive bool) string {
	master := "<ruta>"
	if in.Master != "" {
		master = pathquote.Quote(in.Master)
	}
	command := fmt.Sprintf("axiom branch ensure --cwd %s --change %s", master, pathquote.ShellWord(in.Change))
	if in.Lane == LaneODD {
		command += " --lane " + string(LaneODD)
	}
	if archive {
		command += " --" + archiveRole
	}
	return command
}

// RefusalError es una negativa tipada como error: lo devuelven la guarda A de los escritores y
// el veredicto cuando no puede comprobar la rama. Error() es el mensaje seguido de la
// continuación, de modo que el CLI que lo imprime nombra siempre una orden ejecutable.
type RefusalError struct{ Refusal }

func (e *RefusalError) Error() string { return e.Message + ". " + e.Continuation }

// WrongBranch es el rechazo `wrong_branch` de un veredicto desfavorable (REQ-25.19): nombra el
// código tipado, el repositorio, la rama actual y la esperada, y repite la orden exacta de
// `ensure` del veredicto. La escritura que se niega no ha tocado nada.
func WrongBranch(verdict Verdict, change string) *Refusal {
	return &Refusal{
		Code:    CodeWrongBranch,
		Message: string(CodeWrongBranch) + ": " + verdict.Detail + "; no se ha escrito nada",
		Continuation: fmt.Sprintf("pon el repositorio en la rama que corresponde con `%s` y repite la orden",
			verdict.Continuation),
		Details: map[string]string{
			"change":     change,
			"repository": verdict.Repo,
			"current":    verdict.Current,
			"expected":   strings.Join(verdict.Expected, ", "),
			"detached":   strconv.FormatBool(verdict.Detached),
		},
	}
}
