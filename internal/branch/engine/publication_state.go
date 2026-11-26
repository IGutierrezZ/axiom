package engine

import (
	"context"
	"fmt"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// PublicationState calcula, solo con lecturas, los hechos de la publicación pendiente de un cambio:
// si hay documentación sin confirmar y cuántos commits faltan por subir, y cuáles tocan código. Es
// lo que haría `axiom sdd publish` sin ejecutarlo: parte del mismo destino (`PublicationTarget`), la
// misma lista cerrada de rutas (`PublicationTarget.DocPaths`) y las mismas exclusiones del rango
// (REQ-27.8). No toma el cerrojo, no escribe, no hace `fetch`, `pull`, `ls-remote`, `commit`, `add`
// ni `push`: el puerto de git no tiene ningún método que publique (REQ-25.1).
//
// Límite aceptado: `CommitFiles` no lista rutas en un commit de fusión, mientras que `publish` lee
// las que resolvió la fusión (`--cc`). En una fusión limpia coinciden (ninguna); solo una fusión con
// conflictos resueltos sobre código puede dar aquí «sin código» donde `publish` se negaría.
func (e *Engine) PublicationState(ctx context.Context, req branch.PublicationRequest) (branch.PublicationState, *branch.Refusal) {
	const operation = "status"
	target, location, refusal := e.resolvePublication(ctx, req)
	if refusal != nil {
		return branch.PublicationState{}, refusal
	}
	state := branch.PublicationState{Target: target}
	dir := target.GitRoot
	failed := func(what string, err error) (branch.PublicationState, *branch.Refusal) {
		return branch.PublicationState{}, branch.GitFailed(operation, dir, what, err)
	}

	set, err := target.DocPaths(location.Declarations, req.Change)
	if err != nil {
		return branch.PublicationState{}, &branch.Refusal{
			Code:         branch.CodeConfigInvalid,
			Message:      fmt.Sprintf("las rutas de documentación declaradas en axiom.yaml no son válidas: %v", err),
			Continuation: "corrige `technical_context` en axiom.yaml y comprueba con `axiom workspace validate`",
		}
	}

	configured, err := e.git.RemoteConfigured(ctx, dir, "origin")
	if err != nil {
		return failed("leer la configuración del remoto origin", err)
	}
	if !configured {
		state.RemoteMissing = true
		return state, nil
	}

	// Rango sin subir: la misma regla que el paso 5 de `publish`.
	_, state.RemoteBranch, err = e.git.ResolveCommit(ctx, dir, "refs/remotes/origin/"+target.Branch)
	if err != nil {
		return failed("resolver la rama remota", err)
	}
	remoteIntegration := "refs/remotes/origin/" + target.IntegrationBranch
	_, remoteIntegrationExists, err := e.git.ResolveCommit(ctx, dir, remoteIntegration)
	if err != nil {
		return failed("resolver la integración remota", err)
	}
	var exclude []string
	switch {
	case state.RemoteBranch:
		exclude = append(exclude, "refs/remotes/origin/"+target.Branch)
		if remoteIntegrationExists {
			exclude = append(exclude, remoteIntegration)
		}
	case remoteIntegrationExists:
		exclude = append(exclude, remoteIntegration)
	default:
		localIntegration := "refs/heads/" + target.IntegrationBranch
		_, localExists, err := e.git.ResolveCommit(ctx, dir, localIntegration)
		if err != nil {
			return failed("resolver la rama de integración", err)
		}
		if !localExists {
			state.RangeUnknown = true
			return state, nil
		}
		exclude = append(exclude, localIntegration)
	}

	// Documentación sin confirmar: lo que la lista cerrada acepta, sin el estado local.
	dirty, err := e.git.DirtyPathsUnder(ctx, dir, set.Candidates())
	if err != nil {
		return failed("leer el estado de las rutas de documentación", err)
	}
	for _, rel := range dirty {
		if _, isDoc := set.Root(rel); isDoc && !set.LocalState(rel) {
			state.DirtyDocs = append(state.DirtyDocs, rel)
		}
	}
	slices.Sort(state.DirtyDocs)

	// Commits sin subir y cuáles tocan código, en una sola lectura.
	revisions := []string{"HEAD"}
	for _, ref := range exclude {
		revisions = append(revisions, "^"+ref)
	}
	commits, err := e.git.CommitFiles(ctx, dir, revisions...)
	if err != nil {
		return failed("listar los commits sin subir", err)
	}
	state.Unpushed = len(commits)
	if state.Unpushed > branch.PublicationMaxUnpushed {
		state.RangeTooBig = true
		return state, nil
	}
	// CommitFiles devuelve del más reciente al más antiguo; el estado los da en el orden contrario.
	for i := len(commits) - 1; i >= 0; i-- {
		if slices.ContainsFunc(commits[i].Paths, func(rel string) bool { return !set.Contains(rel) }) {
			state.CodeCommits = append(state.CodeCommits, shortSHA(commits[i].SHA))
		}
	}
	return state, nil
}

// shortSHA son los siete primeros caracteres de un commit, como los cita `axiom sdd publish`.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
