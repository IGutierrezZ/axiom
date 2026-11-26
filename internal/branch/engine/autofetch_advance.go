package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Avance de la integración del autofetch (RF-BRA-39 paso 2, design D-08): la rama local de
// integración del repositorio de specs solo se mueve por `merge --ff-only`, y solo cuando está
// estrictamente por detrás de `origin/<int>`, desplegada en un worktree, con el árbol limpio, sin
// operación git en curso y sin divergencia. Todo lo demás es un aviso con la orden exacta que la
// persona puede ejecutar. Nunca se mueve la rama de un cambio, la de archivado ni la de un
// worktree de feature, y `main_branch` solo cuando coincide con la integración. Ninguna orden
// publica ni poda.

// Valores de AutofetchIntegration.State.
const (
	integrationUpToDate      = "up_to_date"
	integrationAdvanced      = "advanced"
	integrationAhead         = "ahead"
	integrationNotAdvanced   = "not_advanced"
	integrationNoRemoteRef   = "no_remote_ref"
	integrationNoLocalBranch = "no_local_branch"
)

// autofetchAdvance aplica la tabla de D-08 y devuelve los avisos de no avance. Fija
// result.Integration. Tras un fetch fallido o con el plazo vencido no avanza nada: el ciclo
// devuelve lo que ya tenía, con el aviso del fetch. Un fallo de lectura es un aviso del
// resultado y no mueve nada.
func (e *Engine) autofetchAdvance(ctx context.Context, project AutofetchProject, result *branch.AutofetchResult) []branch.AutofetchNotice {
	if result.Fetch == branch.AutofetchFetchFailed || result.Fetch == branch.AutofetchFetchTimeout || ctx.Err() != nil {
		return nil
	}
	integration := integrationBranchOf(project.Location, project.GitRoot)
	state := &branch.AutofetchIntegration{Branch: integration, Checkout: project.Checkout, State: integrationNotAdvanced}
	result.Integration = state

	local, haveLocal, err := e.git.ResolveCommit(ctx, project.Checkout, "refs/heads/"+integration)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	remote, haveRemote, err := e.git.ResolveCommit(ctx, project.Checkout, "refs/remotes/origin/"+integration)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	// Fila 1: sin R, sin L o L == R.
	switch {
	case !haveRemote:
		state.State = integrationNoRemoteRef
		return nil
	case !haveLocal:
		state.State = integrationNoLocalBranch
		return nil
	case local == remote:
		state.State = integrationUpToDate
		state.Before, state.After = local, local
		return nil
	}
	state.Before = local
	// Fila 2: la local va por delante.
	remoteBehind, err := e.git.IsAncestor(ctx, project.Checkout, remote, local)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	if remoteBehind {
		state.State = integrationAhead
		state.After = local
		return nil
	}
	worktrees, err := e.git.Worktrees(ctx, project.Checkout)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	deployed, isDeployed := deployedWorktree(worktrees, integration)
	remoteName := originRemote + "/" + integration
	ffCommand := func(checkout string) string {
		return fmt.Sprintf("git -C %s merge --ff-only %s", pathquote.Quote(checkout), remoteName)
	}
	notice := func(kind, checkout, message, command string) []branch.AutofetchNotice {
		state.Checkout = checkout
		return []branch.AutofetchNotice{{Kind: kind, Branch: integration, Message: message, Command: command}}
	}

	// Fila 3: divergencia (L no es ancestro de R).
	localBehind, err := e.git.IsAncestor(ctx, project.Checkout, local, remote)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	if !localBehind {
		checkout := project.Checkout
		if isDeployed {
			checkout = deployed
		}
		return notice(branch.AutofetchNoticeIntegrationDiverged, checkout,
			fmt.Sprintf("la rama %s ha divergido de %s: no se avanza sola", integration, remoteName),
			fmt.Sprintf("git -C %s rebase %s", pathquote.Quote(checkout), remoteName))
	}
	// Fila 4: operación git en curso sobre la integración, en cualquier worktree.
	if operation, checkout := operationInProgress(worktrees, integration); operation != "" {
		return notice(branch.AutofetchNoticeOperationInProgress, checkout,
			fmt.Sprintf("hay un %s en curso sobre %s en %s: termínalo o abórtalo y el siguiente ciclo avanzará la rama", operation, integration, pathquote.Quote(checkout)),
			fmt.Sprintf("git -C %s status", pathquote.Quote(checkout)))
	}
	if !isDeployed {
		main := pathquote.Quote(project.Checkout)
		// Fila 5: sin desplegar y HEAD desacoplado en el checkout principal.
		if _, detached, err := gitcommon.ReadHead(project.Checkout); err == nil && detached {
			return notice(branch.AutofetchNoticeIntegrationDetached, project.Checkout,
				fmt.Sprintf("la rama %s no está desplegada y HEAD está desacoplado en %s: despliégala para que el siguiente ciclo la avance", integration, main),
				fmt.Sprintf("git -C %s switch %s && git -C %s merge --ff-only %s", main, integration, main, remoteName))
		}
		// Fila 6: sin desplegar (el refspec va sin `+`, así que git rechaza lo que no sea ff).
		return notice(branch.AutofetchNoticeNotDeployed, project.Checkout,
			fmt.Sprintf("la rama %s no está desplegada en ningún worktree: no se avanza sola", integration),
			fmt.Sprintf("git -C %s fetch . refs/remotes/%s:refs/heads/%s", main, remoteName, integration))
	}
	// Fila 7: árbol sucio.
	dirty, err := e.git.DirtyPaths(ctx, deployed)
	if err != nil {
		return advanceReadFailed(result, integration, err)
	}
	if len(dirty) > 0 {
		return notice(branch.AutofetchNoticeIntegrationDirty, deployed,
			fmt.Sprintf("la rama %s tiene cambios sin confirmar en %s: no se avanza sola", integration, pathquote.Quote(deployed)),
			ffCommand(deployed))
	}
	// Fila 8: sin presupuesto para iniciar el avance.
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < autofetchAdvanceReserve {
		return notice(branch.AutofetchNoticeFastForwardFailed, deployed,
			fmt.Sprintf("no se avanzó %s: sin tiempo dentro de la cota del ciclo", integration), ffCommand(deployed))
	}
	// Filas 9 y 10: fast-forward estricto, sin reintento.
	if err := e.writer.FastForward(ctx, deployed, remoteName); err != nil {
		return notice(branch.AutofetchNoticeFastForwardFailed, deployed,
			fmt.Sprintf("no se pudo avanzar %s por fast-forward (%s)", integration, firstLine(err.Error())), ffCommand(deployed))
	}
	state.State, state.Advanced, state.Checkout = integrationAdvanced, true, deployed
	state.After = remote
	if after, ok, err := e.git.ResolveCommit(ctx, deployed, "refs/heads/"+integration); err == nil && ok {
		state.After = after
	}
	return nil
}

// advanceReadFailed anota un fallo de lectura como aviso del resultado: no se avanza nada.
func advanceReadFailed(result *branch.AutofetchResult, integration string, err error) []branch.AutofetchNotice {
	result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo evaluar el avance de %s (%v): no se avanza nada", integration, err))
	return nil
}

// deployedWorktree es el worktree que tiene desplegada la rama, sea el principal o uno enlazado.
func deployedWorktree(worktrees []branch.WorktreeEntry, name string) (string, bool) {
	for _, entry := range worktrees {
		if !entry.Detached && entry.Branch == name {
			return entry.Path, true
		}
	}
	return "", false
}

// operationInProgress busca una operación git en curso sobre la integración, solo con lecturas
// de ficheros: `MERGE_HEAD` o `CHERRY_PICK_HEAD` con `HEAD` en la rama, o `rebase-merge/` o
// `rebase-apply/` cuyo `head-name` es la rama (durante un rebase git informa del worktree como
// desacoplado). Devuelve la operación (`merge`, `rebase`, `cherry-pick`) y el worktree.
func operationInProgress(worktrees []branch.WorktreeEntry, name string) (operation, checkout string) {
	for _, entry := range worktrees {
		gitDir, err := gitcommon.GitDirFromFiles(entry.Path)
		if err != nil {
			continue
		}
		for _, dir := range []string{"rebase-merge", "rebase-apply"} {
			if head, err := os.ReadFile(filepath.Join(gitDir, dir, "head-name")); err == nil &&
				strings.TrimSpace(string(head)) == "refs/heads/"+name {
				return "rebase", entry.Path
			}
		}
		if current, detached, err := gitcommon.ReadHead(entry.Path); err != nil || detached || current != name {
			continue
		}
		for _, candidate := range []struct{ file, label string }{{"MERGE_HEAD", "merge"}, {"CHERRY_PICK_HEAD", "cherry-pick"}} {
			if _, err := os.Stat(filepath.Join(gitDir, candidate.file)); err == nil {
				return candidate.label, entry.Path
			}
		}
	}
	return "", ""
}

// firstLine es la primera línea no vacía de un texto: el resumen del error de git.
func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return text
}
