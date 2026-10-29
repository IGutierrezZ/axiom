package branch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de las excepciones por cambio y del repositorio de specs de un multirepo en `ensure`
// (REQ-25.3, REQ-25.5, REQ-25.12, design D-04 y D-05). Cada uno nombra una orden ejecutable con las
// rutas citadas por pathquote.Quote (tasks.md, regla 11).

// ensureInvocation es la orden de `ensure` con el cambio, el carril (el de por defecto, `sdd`, no
// se repite), las excepciones que no están vacías y la carpeta de partida.
func ensureInvocation(operation, change, cwd string, lane Lane, exceptions Exceptions) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "axiom branch %s --change %s", operation, change)
	if lane == LaneODD {
		builder.WriteString(" --lane " + string(lane))
	}
	for _, flag := range []struct{ name, value string }{
		{"--prefix", exceptions.Prefix}, {"--branch", exceptions.Branch}, {"--base", exceptions.Base},
	} {
		if flag.value != "" {
			builder.WriteString(" " + flag.name + " " + flag.value)
		}
	}
	fmt.Fprintf(&builder, " --cwd %s", pathquote.Quote(cwd))
	return builder.String()
}

// ValidateExceptions rechaza con `invalid_name`, sin efectos, una excepción de la invocación que no
// es válida (REQ-25.3): un `--prefix` o un `--branch` que no son nombres de referencia git, o una
// `--base` que empieza por «-» y git leería como una opción (`--base` es una referencia que
// resuelve git, no un nombre). operation, dir y change componen la continuación. El motor lo
// comprueba antes de bloquear el cambio ni leer nada con él.
func ValidateExceptions(operation, dir, change string, exceptions Exceptions) *Refusal {
	if refusal := validateExceptions(PlanInput{Operation: operation, Change: change, Master: dir, Exceptions: exceptions}); refusal != nil {
		return refusal
	}
	if base := exceptions.Base; strings.HasPrefix(base, "-") {
		return invalidName("--base", "la base indicada con --base", base,
			errors.New("no puede empezar por «-»: git la leería como una opción"),
			fmt.Sprintf("repite la orden con una referencia válida: `axiom branch %s --change %s --base <ref> --cwd %s`", operation, change, pathquote.Quote(dir)))
	}
	return nil
}

// ExceptionConflict es el rechazo `plan_conflict` de una excepción que contradice la ya
// materializada del cambio (REQ-25.12): una excepción no cambia lo que ya existe, y la repetición
// idéntica es idempotente. Nombra cada excepción que difiere, con su valor registrado y el pedido
// (un valor registrado vacío es la política del proyecto). No se ha tocado nada; la continuación
// repite `ensure` con las excepciones registradas.
func ExceptionConflict(operation, change, cwd string, lane Lane, materialized, requested Exceptions) *Refusal {
	var parts, fields []string
	for _, field := range []struct{ name, flag, registered, asked string }{
		{"prefix", "--prefix", materialized.Prefix, requested.Prefix},
		{"branch", "--branch", materialized.Branch, requested.Branch},
		{"base", "--base", materialized.Base, requested.Base},
	} {
		if field.asked == "" || field.asked == field.registered {
			continue
		}
		registered := "la de la política"
		if field.registered != "" {
			registered = "«" + field.registered + "»"
		}
		fields = append(fields, field.name)
		parts = append(parts, fmt.Sprintf("%s: registrada %s, pedida «%s»", field.flag, registered, field.asked))
	}
	return &Refusal{
		Code: CodePlanConflict,
		Message: fmt.Sprintf("el cambio %s ya está materializado y las excepciones pedidas lo contradicen (%s): una excepción no cambia lo que ya existe; no se ha tocado nada",
			change, strings.Join(parts, "; ")),
		Continuation: fmt.Sprintf("repite con las excepciones registradas, `%s`, o sin excepciones", ensureInvocation(operation, change, cwd, lane, materialized)),
		Details:      map[string]string{"change": change, "fields": strings.Join(fields, ",")},
	}
}

// SpecsRepoNotOnIntegration es el rechazo `specs_repo_not_on_integration` (REQ-25.5): en un
// multirepo, `ensure` trabaja el repositorio de specs in situ, en su rama de integración, y nunca
// hace checkout en él; si está en otra rama (una ajena o una rama de rol del cambio) o con HEAD
// desacoplado (detached), no se crea nada en ningún repositorio. current es la rama actual (vacía
// con HEAD desacoplado). La continuación cambia a la integración y repite `ensure`.
func SpecsRepoNotOnIntegration(operation, change, cwd string, lane Lane, repo RepoPlan, current string, detached bool) *Refusal {
	where := "está en la rama " + current
	if detached {
		where = "tiene HEAD desacoplado"
		current = ""
	}
	return &Refusal{
		Code: CodeSpecsRepoNotOnIntegration,
		Message: fmt.Sprintf("el repositorio de specs %s (%s) %s y no está en su rama de integración %s: el repositorio de specs se trabaja in situ y no se hace checkout en él; no se ha creado ninguna rama ni worktree",
			repo.Name, repo.GitRoot, where, repo.IntegrationBranch),
		Continuation: fmt.Sprintf("cambia el repositorio de specs a su integración con `git -C %s switch %s` y repite `%s`",
			pathquote.Quote(repo.GitRoot), repo.IntegrationBranch, ensureInvocation(operation, change, cwd, lane, Exceptions{})),
		Details: map[string]string{"repository": repo.GitRoot, "branch": current, "expected": repo.IntegrationBranch},
	}
}
