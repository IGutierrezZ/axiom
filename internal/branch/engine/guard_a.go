package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// WriteTarget es lo que un escritor Go de la carpeta de un cambio declara a la guarda A
// (REQ-25.19, design D-11): la carpeta del cambio, el cambio y el carril, el contexto en el que
// consulta (cada escritor fija el suyo: `archive` para `integrate` y `archive-compose`,
// `ordinary` para el resto), la operación, solo para los mensajes, y, en `kickoff seal`, los roles
// que se van a sellar.
type WriteTarget struct {
	ChangeRoot, Change string
	Lane               branch.Lane
	Context            branch.Context
	Op                 string
	SealRoles          []string
}

// RequireChangeBranch es la guarda A (REQ-25.19, design D-11): un escritor Go de la carpeta de un
// cambio la llama antes de escribir. Consulta VerdictFor con el consultante `write` y el contexto
// del escritor:
//
//   - con `correct` o `not_applicable` devuelve nil y el aviso de la consulta, si lo hay;
//   - con `wrong_branch` devuelve un *branch.RefusalError (`wrong_branch`, con la rama actual, la
//     esperada y la orden exacta de `ensure`): el escritor se niega sin escribir nada;
//   - ante un fallo interno —configuración, kickoff o registro ilegibles, un contexto cancelado—
//     devuelve `config_invalid`: la guarda falla en cerrado, a diferencia de `guard` (D), que lo
//     hace en abierto.
//
// No puede vivir dentro de `kickoff.Seal`, `kickoff.AppendGate` ni `handoff.WriteFile`: `engine`
// importa `kickoff` y esos escritores no conocen ni el cambio ni el contexto.
func (e *Engine) RequireChangeBranch(ctx context.Context, t WriteTarget) (warning string, err error) {
	verdict, err := e.VerdictFor(ctx, VerdictQuery{
		Path:       t.ChangeRoot,
		Change:     t.Change,
		Lane:       t.Lane,
		Context:    t.Context,
		Consultant: branch.ConsultantWrite,
		SealRoles:  t.SealRoles,
	})
	if err != nil {
		var typed *branch.RefusalError
		if errors.As(err, &typed) {
			return "", typed
		}
		return "", failClosed(t, err)
	}
	if verdict.Kind == branch.VerdictWrongBranch {
		return verdict.Warning, &branch.RefusalError{Refusal: *branch.WrongBranch(verdict, t.Change)}
	}
	return verdict.Warning, nil
}

// failClosed es el rechazo `config_invalid` de un fallo interno de la consulta: la escritura no
// se hace, y la continuación lleva a `axiom branch status`, que informa del estado del cambio.
func failClosed(t WriteTarget, cause error) *branch.RefusalError {
	op := t.Op
	if op == "" {
		op = "escribir en la carpeta del cambio"
	}
	return &branch.RefusalError{Refusal: branch.Refusal{
		Code:    branch.CodeConfigInvalid,
		Message: fmt.Sprintf("no se pudo comprobar la rama del cambio %s antes de %s: %v; no se ha escrito nada", t.Change, op, cause),
		Continuation: fmt.Sprintf("comprueba el estado del cambio con `axiom branch status --change %s --cwd %s` y repite la orden",
			pathquote.ShellWord(t.Change), pathquote.Quote(t.ChangeRoot)),
		Details: map[string]string{"change": t.Change, "operation": op},
	}}
}
