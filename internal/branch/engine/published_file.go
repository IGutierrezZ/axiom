package engine

import (
	"context"
	"path"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// PublishedChangeFile lee, solo con lecturas y sin red, la versión publicada de name (un fichero
// directamente bajo la carpeta del cambio) en `refs/remotes/origin/<rama que se publica>`. Parte
// del mismo destino que `PublicationTarget`, así que comparte sus rechazos (`wrong_branch`,
// `detached_head`, …). No toma el cerrojo, no escribe y no hace `fetch`: la referencia de
// seguimiento la actualizan un `push` con éxito o un `fetch` de la persona. Sin referencia o sin
// el fichero en ella, devuelve Exists falso y contenido vacío.
func (e *Engine) PublishedChangeFile(ctx context.Context, req branch.PublicationRequest, name string) (branch.PublishedChangeFile, *branch.Refusal) {
	target, _, refusal := e.resolvePublication(ctx, req)
	if refusal != nil {
		return branch.PublishedChangeFile{}, refusal
	}
	published := branch.PublishedChangeFile{Target: target, RemoteRef: "refs/remotes/origin/" + target.Branch}
	dir := target.GitRoot
	_, exists, err := e.git.ResolveCommit(ctx, dir, published.RemoteRef)
	if err != nil {
		return branch.PublishedChangeFile{}, branch.GitFailed("status", dir, "resolver la rama remota", err)
	}
	if !exists {
		return published, nil
	}
	published.RemoteRefExists = true
	rel := path.Join(target.PlanningRel, "changes", req.Change, name)
	content, found, err := e.git.FileAt(ctx, dir, published.RemoteRef, rel)
	if err != nil {
		return branch.PublishedChangeFile{}, branch.GitFailed("status", dir, "leer "+rel+" en "+published.RemoteRef, err)
	}
	published.Exists, published.Content = found, content
	return published, nil
}
