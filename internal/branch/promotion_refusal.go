package branch

import (
	"fmt"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos y decisión de la promoción de ramas (RF-BRA-29). Como los del modo de trabajo, cada
// uno nombra una orden `axiom branch ensure` ejecutable con las rutas citadas por
// pathquote.Quote, y conserva en ella lo que la invocación ya daba (excepciones, `--repo` y
// las respuestas de `--promotion` anteriores).

// EnsureArgs son los argumentos de una invocación de `ensure` que las continuaciones de sus
// rechazos repiten.
type EnsureArgs struct {
	Operation  string
	Change     string
	CWD        string
	Lane       Lane
	Mode       Mode
	Exceptions Exceptions
	// Repos son los repositorios incluidos con `--repo` (carril ODD).
	Repos      []string
	Promotions []PromotionRequest
}

// Invocation compone la orden: `axiom branch <op> --change c [--lane odd] [--mode m] [--prefix p]
// [--branch b] [--base r] [--repo q]… [--promotion x]… --cwd q`. Sin banderas opcionales produce
// exactamente la cadena de modeInvocation; el carril `sdd`, el de por defecto, no se repite.
func (a EnsureArgs) Invocation() string {
	var command strings.Builder
	fmt.Fprintf(&command, "axiom branch %s --change %s", a.Operation, a.Change)
	if a.Lane == LaneODD {
		command.WriteString(" --lane " + string(a.Lane))
	}
	if a.Mode != "" {
		command.WriteString(" --mode " + string(a.Mode))
	}
	for _, flag := range []struct{ name, value string }{
		{"--prefix", a.Exceptions.Prefix}, {"--branch", a.Exceptions.Branch}, {"--base", a.Exceptions.Base},
	} {
		if flag.value != "" {
			command.WriteString(" " + flag.name + " " + flagValue(flag.value))
		}
	}
	for _, repo := range a.Repos {
		command.WriteString(" --repo " + pathquote.Quote(repo))
	}
	for _, request := range a.Promotions {
		value := string(request.Mode)
		if request.Repository != "" {
			value = request.Repository + "=" + value
		}
		command.WriteString(" --promotion " + flagValue(value))
	}
	command.WriteString(" --cwd " + pathquote.Quote(a.CWD))
	return command.String()
}

// flagValue cita un valor de bandera solo cuando lo necesita (espacios o comillas).
func flagValue(value string) string {
	if strings.ContainsAny(value, " \t\"") {
		return pathquote.Quote(value)
	}
	return value
}

// withPromotion devuelve una copia de los argumentos con la respuesta dada: sustituye las
// peticiones que ya nombraban ese repositorio y añade la nueva. qualify cualifica la respuesta
// con `<repo>=`; sin él, la petición queda sin repositorio y sustituye también a las demás
// sin cualificar.
func (a EnsureArgs) withPromotion(repo RepoPlan, mode Promotion, qualify bool) EnsureArgs {
	name := ""
	if qualify {
		name = repo.Name
	}
	kept := make([]PromotionRequest, 0, len(a.Promotions)+1)
	for _, request := range a.Promotions {
		if request.Repository != name {
			kept = append(kept, request)
		}
	}
	a.Promotions = append(kept, PromotionRequest{Repository: name, Mode: mode})
	return a
}

// withoutPromotions devuelve una copia de los argumentos sin ningún `--promotion`.
func (a EnsureArgs) withoutPromotions() EnsureArgs {
	a.Promotions = nil
	return a
}

// PromotionDecisionRequired es el resultado `decision_required` de un repositorio de dos etapas
// cuyo modo de promoción es `ask` y el cambio no tiene respuesta registrada (RF-BRA-29): no se ha
// creado nada. Pregunta por un solo repositorio (el primero del plan que lo necesita); las
// respuestas de los demás llegan en llamadas sucesivas, cada una con las anteriores en su
// invocación. qualify es cierto cuando el plan tiene dos o más repositorios con ramas: la
// respuesta se cualifica con `--promotion <repo>=<modo>`.
func PromotionDecisionRequired(args EnsureArgs, repo RepoPlan, qualify bool) *Refusal {
	branchArgs := args.withPromotion(repo, PromotionBranch, qualify)
	releaseArgs := args.withPromotion(repo, PromotionRelease, qualify)
	branchInvocation, releaseInvocation := branchArgs.Invocation(), releaseArgs.Invocation()
	return &Refusal{
		Code: CodeDecisionRequired,
		Message: fmt.Sprintf("la promoción del repositorio %s es «%s» y el cambio %s no tiene respuesta registrada: no se ha creado nada ni se ha escrito el registro",
			repo.Name, PromotionLabel(string(PromotionAsk)), args.Change),
		Continuation: fmt.Sprintf("elige un modo y repite `%s` (%s) o `%s` (%s): la respuesta se registra para este cambio",
			branchInvocation, PromotionLabel(string(PromotionBranch)), releaseInvocation, PromotionLabel(string(PromotionRelease))),
		Details: map[string]string{"change": args.Change, "repository": repo.GitRoot, "name": repo.Name, "promotion": string(PromotionAsk)},
		Decision: &Decision{
			Question: fmt.Sprintf("El repositorio %s tiene dos ramas permanentes (integración %s, principal %s) y el cambio %s no tiene modo de promoción registrado: ¿%s o %s?",
				repo.Name, repo.IntegrationBranch, repo.MainBranch, args.Change, PromotionLabel(string(PromotionBranch)), PromotionLabel(string(PromotionRelease))),
			Choices: []DecisionChoice{
				{
					Answer: string(PromotionBranch),
					Label:  PromotionLabel(string(PromotionBranch)),
					Effect: fmt.Sprintf("la rama del cambio sale de origin/%s: su primera PR va a %s sin borrarse y la segunda a %s; `done` la elimina al fusionarse en %s",
						repo.MainBranch, repo.IntegrationBranch, repo.MainBranch, repo.MainBranch),
					Invocation: branchInvocation,
				},
				{
					Answer: string(PromotionRelease),
					Label:  PromotionLabel(string(PromotionRelease)),
					Effect: fmt.Sprintf("la rama del cambio sale de %s y su PR va a %s; el paso a %s es la release",
						repo.IntegrationBranch, repo.IntegrationBranch, repo.MainBranch),
					Invocation: releaseInvocation,
				},
			},
		},
	}
}

// PromotionConflict es el rechazo `plan_conflict` de un `--promotion` distinto del modo con el
// que el repositorio del cambio ya está materializado (RF-BRA-29): un cambio no cambia de modo y
// nunca se rehace la base. No se ha tocado nada; source es el origen del modo materializado.
func PromotionConflict(args EnsureArgs, repo RepoPlan, materialized, requested Promotion, source string) *Refusal {
	// La respuesta se cualifica siempre: la cualificada prevalece sobre una sin cualificar, que
	// puede estar pensada para otros repositorios del plan.
	keep := args.withPromotion(repo, materialized, true)
	message := fmt.Sprintf("el repositorio %s del cambio %s ya está materializado en modo de promoción %s (%s) y se pidió --promotion %s: un cambio no cambia de modo de promoción",
		repo.Name, args.Change, materialized, PromotionLabel(source), requested)
	return &Refusal{
		Code:         CodePlanConflict,
		Message:      message,
		Continuation: fmt.Sprintf("repite con el modo materializado, `%s`, o sin --promotion, `%s`", keep.Invocation(), args.withoutPromotions().Invocation()),
		Details: map[string]string{
			"change": args.Change, "repository": repo.GitRoot, "name": repo.Name,
			"promotion": string(materialized), "requested": string(requested),
		},
	}
}

// UnknownPromotionRepository es el rechazo `invalid_name` de un `--promotion <repo>=…` cuyo
// `<repo>` no es ningún repositorio del plan del cambio. available son los nombres válidos.
func UnknownPromotionRepository(args EnsureArgs, name string, available []string) *Refusal {
	available = slices.Clone(available)
	slices.Sort(available)
	listed := strings.Join(available, ", ")
	if listed == "" {
		listed = "ninguno"
	}
	return &Refusal{
		Code:    CodeInvalidName,
		Message: fmt.Sprintf("el repositorio «%s» de --promotion no es ninguno del plan del cambio %s: los repositorios del plan son %s", name, args.Change, listed),
		Continuation: fmt.Sprintf("usa `--promotion <repo>=branch` o `--promotion <repo>=release` con uno de esos repositorios y repite `%s`",
			args.withoutPromotions().Invocation()),
		Details: map[string]string{"change": args.Change, "repository": name, "available": strings.Join(available, "\n")},
	}
}

// InvalidPromotion es el rechazo `config_invalid` de una petición cuyo modo de promoción no es una
// respuesta válida: solo `branch` y `release` se materializan (`ask` es una política). No se ha
// tocado nada.
func InvalidPromotion(args EnsureArgs, value string) *Refusal {
	branchArgs := args.withoutPromotions()
	branchArgs.Promotions = []PromotionRequest{{Mode: PromotionBranch}}
	releaseArgs := args.withoutPromotions()
	releaseArgs.Promotions = []PromotionRequest{{Mode: PromotionRelease}}
	return &Refusal{
		Code:         CodeConfigInvalid,
		Message:      fmt.Sprintf("el modo de promoción «%s» no es una respuesta válida: usa %s o %s", value, PromotionBranch, PromotionRelease),
		Continuation: fmt.Sprintf("repite `%s` o `%s`", branchArgs.Invocation(), releaseArgs.Invocation()),
		Details:      map[string]string{"change": args.Change, "promotion": value},
	}
}

// withMode devuelve una copia de los argumentos con el modo de trabajo dado.
func (a EnsureArgs) withMode(mode Mode) EnsureArgs {
	a.Mode = mode
	return a
}
