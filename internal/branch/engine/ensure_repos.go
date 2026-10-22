package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Excepciones por cambio y selección de repositorios de `ensure` (REQ-25.12, REQ-25.13, design
// D-04): lo que la invocación pide con `--prefix`, `--branch`, `--base` y `--repo` se contrasta con
// el registro del cambio y se traduce en los hechos que entran al planificador.

// gatherOptions son las peticiones de `ensure` que condicionan los hechos del plan: las
// excepciones de la invocación y los repositorios de `--repo`.
type gatherOptions struct {
	exceptions branch.Exceptions
	repos      []string
}

// mergeExceptions devuelve las excepciones efectivas del cambio (REQ-25.12). Sin registro rigen
// las pedidas. Con registro, las no indicadas son las registradas y una pedida igual a la
// registrada es una repetición idempotente; una pedida distinta es `plan_conflict` cuando el cambio
// ya tiene ramas materializadas (una excepción no cambia lo que ya existe) y, mientras no las
// tiene (un multirepo antes del sellado solo registra el repositorio de specs, sin ramas), aún no
// contradice nada y sustituye a la registrada.
func mergeExceptions(operation, cwd, change string, lane branch.Lane, record *branch.Record, requested branch.Exceptions) (branch.Exceptions, *branch.Refusal) {
	if record == nil {
		return requested, nil
	}
	effective := record.Exceptions
	// La rama de archivado (`ensure --archive`) no es del plan del cambio: un registro que solo
	// la tiene no materializa nada que una excepción del cambio pueda contradecir.
	materialized := false
	for _, repo := range record.Repositories {
		for _, recorded := range repo.Branches {
			materialized = materialized || recorded.Kind != branch.KindArchive
		}
	}
	conflict := false
	for _, field := range []struct {
		effective *string
		requested string
	}{
		{&effective.Prefix, requested.Prefix}, {&effective.Branch, requested.Branch}, {&effective.Base, requested.Base},
	} {
		switch {
		case field.requested == "" || field.requested == *field.effective:
		case materialized:
			conflict = true
		default:
			*field.effective = field.requested
		}
	}
	if conflict {
		return branch.Exceptions{}, branch.ExceptionConflict(operation, change, cwd, lane, record.Exceptions, requested)
	}
	return effective, nil
}

// chosenRepositories son los repositorios de código del plan tras `--repo`, los que el carril ODD
// incluye, las rutas omitidas y los avisos.
type chosenRepositories struct {
	code     []branch.RepoFacts
	included []branch.RepoFacts
	skipped  []branch.SkippedPath
	warnings []string
}

// chooseRepositories aplica `--repo` (REQ-25.12, REQ-25.13):
//
//   - en el carril ODD, que no divide en roles, incluye los repositorios pedidos y los que el
//     registro del cambio ya guarda (de modo que repetir `ensure` o `status` sin `--repo` los
//     conserva); cada uno recibirá solo la rama de la feature;
//   - en el carril SDD acota a los pedidos los repositorios de código que declaran los roles del
//     cambio, solo en esta ejecución; uno pedido que no es del plan se ignora con un aviso.
//
// Una ruta que no existe o no es un repositorio git se omite sin abortar (REQ-25.6). Sin `--repo`
// y sin registro, los repositorios son los declarados.
func (e *Engine) chooseRepositories(ctx context.Context, operation string, location workspace.Location, planning branch.RepoFacts, lane branch.Lane, code []branch.RepoFacts, record *branch.Record, repos []string) (chosenRepositories, *branch.Refusal) {
	chosen := chosenRepositories{code: code}
	if lane == branch.LaneODD {
		return chosen, e.includeRepositories(ctx, operation, location, planning, record, repos, &chosen)
	}
	if len(repos) == 0 {
		return chosen, nil
	}
	var roots []string
	for _, path := range repos {
		_, root, reason, err := e.resolveRepositoryPath(ctx, location, path)
		if err != nil {
			return chosenRepositories{}, gitRefusal(operation, location.Master, "localizar la raíz git de "+path, err)
		}
		if reason != "" {
			chosen.skipped = append(chosen.skipped, skipRepository(path, location, "indicada con --repo", reason))
			continue
		}
		roots = append(roots, root)
		if !pathidentity.SameDirectory(root, planning.GitRoot) && repositoryAt(pointers(code), root) == nil {
			chosen.warnings = append(chosen.warnings, fmt.Sprintf("la ruta %s indicada con --repo no es un repositorio del plan del cambio: se ignora", path))
		}
	}
	chosen.code = nil
	for _, facts := range code {
		for _, root := range roots {
			if pathidentity.SameDirectory(root, facts.GitRoot) {
				chosen.code = append(chosen.code, facts)
				break
			}
		}
	}
	return chosen, nil
}

// includeRepositories resuelve los repositorios que el carril ODD incluye: los pedidos con
// `--repo` y los que el registro del cambio guarda, sin repetir el del planning home ni ninguno ya
// incluido.
func (e *Engine) includeRepositories(ctx context.Context, operation string, location workspace.Location, planning branch.RepoFacts, record *branch.Record, repos []string, chosen *chosenRepositories) *branch.Refusal {
	type wanted struct{ path, declared, origin string }
	var list []wanted
	for _, path := range repos {
		list = append(list, wanted{path: path, declared: path, origin: "indicada con --repo"})
	}
	if record != nil {
		for _, registered := range record.Repositories {
			if !pathidentity.SameDirectory(registered.GitRoot, planning.GitRoot) {
				list = append(list, wanted{path: registered.GitRoot, declared: registered.Name, origin: "registrada para el cambio"})
			}
		}
	}
	known := []*branch.RepoFacts{&planning}
	for _, item := range list {
		dir, root, reason, err := e.resolveRepositoryPath(ctx, location, item.path)
		if err != nil {
			return gitRefusal(operation, location.Master, "localizar la raíz git de "+item.path, err)
		}
		if reason != "" {
			chosen.skipped = append(chosen.skipped, skipRepository(dir, location, item.origin, reason))
			continue
		}
		if repositoryAt(known, root) != nil {
			continue
		}
		facts, refusal := e.completeFacts(ctx, operation, location, dir, root, item.declared, location.Master)
		if refusal != nil {
			return refusal
		}
		known = append(known, &facts)
		chosen.included = append(chosen.included, facts)
	}
	return nil
}

// resolveRepositoryPath resuelve una ruta de `--repo` (relativa a la carpeta maestra o absoluta) a
// la raíz git que la contiene. reason no está vacío cuando la ruta no sirve: no existe o no
// pertenece a ningún repositorio git.
func (e *Engine) resolveRepositoryPath(ctx context.Context, location workspace.Location, path string) (dir, root, reason string, err error) {
	dir = path
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(location.Master, filepath.FromSlash(strings.ReplaceAll(dir, `\`, "/")))
	}
	dir = filepath.Clean(dir)
	if reason := missingPath(dir); reason != "" {
		return dir, "", reason, nil
	}
	root, err = e.git.TopLevel(ctx, dir)
	if errors.Is(err, gitcommon.ErrNotGitRepository) {
		return dir, "", "no pertenece a ningún repositorio git", nil
	}
	if err != nil {
		return "", "", "", err
	}
	return dir, root, "", nil
}

// skipRepository compone la ruta omitida de `--repo` (o del registro): nombra la ruta, de dónde
// sale y el motivo (REQ-25.6).
func skipRepository(path string, location workspace.Location, origin, reason string) branch.SkippedPath {
	dir := path
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(location.Master, filepath.FromSlash(strings.ReplaceAll(dir, `\`, "/")))
	}
	dir = filepath.Clean(dir)
	return branch.SkippedPath{
		Path:    dir,
		Code:    branch.CodeNotAGitRepository,
		Message: fmt.Sprintf("la ruta %s %s %s; se omite del plan", dir, origin, reason),
	}
}

// pointers devuelve punteros a los repositorios de la lista, para repositoryAt.
func pointers(repositories []branch.RepoFacts) []*branch.RepoFacts {
	list := make([]*branch.RepoFacts, 0, len(repositories))
	for i := range repositories {
		list = append(list, &repositories[i])
	}
	return list
}

// checkSpecsRepository es la comprobación del repositorio de specs de un multirepo (REQ-25.5): se
// trabaja in situ, en su rama de integración, y `ensure` nunca hace checkout en él. Fuera de ella
// (otra rama, una rama de rol del cambio o HEAD desacoplado) rechaza con
// `specs_repo_not_on_integration`. La rama actual se lee de ficheros, sin lanzar procesos.
func checkSpecsRepository(operation, cwd string, repo branch.RepoPlan, facts changeFacts) *branch.Refusal {
	current, detached, err := gitcommon.ReadHead(repo.GitRoot)
	if err != nil {
		return branch.GitFailed(operation, cwd, "leer la rama actual del repositorio de specs "+repo.GitRoot, err)
	}
	if !detached && current == repo.IntegrationBranch {
		return nil
	}
	return branch.SpecsRepoNotOnIntegration(operation, facts.input.Change, cwd, facts.input.Lane, repo, current, detached)
}
