package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// branchGoneMessage es la frase que la spec fija para una rama que no está en local ni descargada
// (REQ-25.53).
const branchGoneMessage = "la rama ya no está o falta un fetch autorizado"

// Scope calcula, solo con lecturas de git local, qué aporta el cambio en cada repositorio de su
// instantánea de política (REQ-25.51 a REQ-25.54, `axiom.branch-scope/v1`): por repositorio, la rama
// del cambio, la base y la integración, el punto de partida, los commits propios y los ficheros que
// tocan, si la rama está fusionada en la integración y un resultado tipado; y, para todo el cambio,
// el resumen con la oferta de verificación conjunta.
//
// Es de solo lectura y no tiene red: nunca ejecuta `fetch`, no crea ni borra ramas ni worktrees, no
// toca referencias, índice ni árbol de trabajo y no escribe ningún documento del cambio. Rechaza,
// antes de calcular nada, un cambio sin kickoff sellado (`kickoff_not_sealed`), del carril ODD o con
// un kickoff sin instantánea (`scope_not_applicable`) y una configuración ilegible (`config_invalid`);
// un fallo de git que ningún código cubre aborta con `git_failed`, sin degradarse a un resultado por
// repositorio. Un repositorio cuyo alcance no puede calcularse con certeza es `unverifiable`, y no
// impide calcular el resto. cwd puede ser relativa: se resuelve a absoluta.
func (e *Engine) Scope(ctx context.Context, cwd, change string) branch.ScopeResult {
	const operation = "scope"
	refused := func(refusal *branch.Refusal) branch.ScopeResult {
		return branch.NewScopeResult().Refused(refusal)
	}
	if absolute, err := filepath.Abs(cwd); err == nil {
		cwd = absolute
	}
	facts, refusal := e.gather(ctx, operation, cwd, change, branch.LaneSDD)
	if refusal != nil {
		return refused(refusal)
	}
	if !facts.input.HasKickoff {
		if document := filepath.Join(facts.location.WorkspaceRoot, "odd", "tasks", change+".md"); regularFile(document) {
			return refused(branch.ScopeNotApplicableOdd(change, cwd, document))
		}
		return refused(branch.ScopeKickoffNotSealed(change, cwd))
	}
	if facts.input.Snapshot == nil {
		return refused(branch.ScopeNotApplicableWithoutSnapshot(change, cwd))
	}
	plan, _, refusal := e.planWithPromotion(ctx, &facts, promotionRequest{args: branch.EnsureArgs{Operation: operation, Change: change, CWD: cwd, Lane: branch.LaneSDD}})
	if refusal != nil {
		return refused(refusal)
	}

	result := branch.NewScopeResult()
	result.Change, result.Topology, result.WorkspaceRoot = change, facts.input.Topology, facts.location.WorkspaceRoot
	for _, repo := range plan.Repositories {
		entry, refusal := e.scopeRepository(ctx, operation, cwd, facts, repo, change)
		if refusal != nil {
			return refused(refusal)
		}
		result.Repositories = append(result.Repositories, entry)
	}
	// Ningún repositorio de la instantánea se omite (REQ-25.53): uno declarado que no existe en esta
	// máquina es `unverifiable` con la ruta que falta (P1).
	for _, skipped := range plan.Skipped {
		result.Repositories = append(result.Repositories, scopeAbsentRepository(skipped))
	}
	summary := branch.SummarizeScope(result.Repositories)
	result.Summary = &summary
	return result
}

// regularFile dice si path es un fichero regular que existe.
func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// scopeAbsentRepository es la entrada de un repositorio declarado que no existe en esta máquina o no
// es un repositorio git (P1): `unverifiable` con `branch_missing`, `details.path` y la ruta en el
// mensaje. El dominio de motivos no se amplía.
func scopeAbsentRepository(skipped branch.SkippedPath) branch.RepositoryScope {
	entry := branch.RepositoryScope{
		Name:      filepath.Base(skipped.Path),
		GitRoot:   skipped.Path,
		Role:      branch.RepoCode,
		RoleLabel: branch.Label(string(branch.RepoCode)),
	}
	message := fmt.Sprintf("%s: el repositorio %s no está disponible en esta máquina (%s)", branchGoneMessage, skipped.Path, skipped.Message)
	entry.SetUnverifiable(branch.ScopeReasonBranchMissing, message, "", map[string]string{"path": skipped.Path})
	return entry
}

// scopeRepository calcula la entrada de un repositorio del plan: exenta si es el de specs in situ,
// `unverifiable` si falta la rama, la base o la integración, y calculada en otro caso. Un fallo de
// git que ningún código cubre se devuelve como rechazo.
func (e *Engine) scopeRepository(ctx context.Context, operation, cwd string, facts changeFacts, repo branch.RepoPlan, change string) (branch.RepositoryScope, *branch.Refusal) {
	entry := branch.RepositoryScope{
		Name:      repo.Name,
		GitRoot:   repo.GitRoot,
		Role:      repo.Role,
		RoleLabel: branch.Label(string(repo.Role)),
		Promotion: repo.Promotion,
	}
	if repo.Role == branch.RepoSpecsInPlace {
		entry.SetOutcome(branch.ScopeOutcomeExempt)
		return entry, nil
	}
	fetch := "git -C " + pathquote.Quote(repo.GitRoot) + " fetch origin"
	failed := func(what string, err error) *branch.Refusal {
		return gitRefusal(operation, cwd, what+" de "+repo.GitRoot, err)
	}

	name := changeBranchName(repo)
	tip, tipRef, found, err := e.scopeTip(ctx, repo.GitRoot, name)
	if err != nil {
		return entry, failed("resolver la rama del cambio", err)
	}
	if !found {
		entry.Branch = &branch.ScopeRef{Name: name}
		entry.SetUnverifiable(branch.ScopeReasonBranchMissing, fmt.Sprintf("%s: %s no existe en local ni como origin/%s ya descargada", branchGoneMessage, name, name), fetch, nil)
		return entry, nil
	}
	entry.Branch = branch.NewScopeRef(name, tipRef.full, tipRef.source, tip)
	entry.Tip = tip

	base, haveBase, err := e.resolveBase(ctx, repo)
	if err != nil {
		return entry, failed("resolver la base", err)
	}
	if !haveBase {
		missing := repo.Base()
		if repo.BaseRef != "" {
			missing = repo.BaseRef
		}
		entry.SetUnverifiable(branch.ScopeReasonBaseMissing, fmt.Sprintf("la base %s del cambio no existe en local ni como referencia remota ya descargada", missing), fetch, map[string]string{"missingRef": missing})
		return entry, nil
	}
	entry.Base = branch.NewScopeRef(base.ref, base.full, base.source, base.sha)

	integration, haveIntegration, err := e.scopeIntegration(ctx, repo)
	if err != nil {
		return entry, failed("resolver la rama de integración", err)
	}
	if !haveIntegration {
		entry.SetUnverifiable(branch.ScopeReasonBaseMissing, fmt.Sprintf("la rama de integración %s no existe en local ni como referencia remota ya descargada", repo.IntegrationBranch), fetch, map[string]string{"missingRef": repo.IntegrationBranch})
		return entry, nil
	}
	entry.Integration = integration.scopeRef(repo.IntegrationBranch)

	// El cálculo de los commits propios y los ficheros llega con la siguiente unidad: mientras tanto
	// la rama resuelta no aporta nada.
	entry.SetOutcome(branch.ScopeOutcomeEmpty)
	return entry, nil
}

// scopeReference es una referencia resuelta del alcance: la completa, de dónde sale y el commit.
type scopeReference struct {
	shown  string
	full   string
	source string
	sha    string
}

func (r scopeReference) scopeRef(name string) *branch.ScopeRef {
	return branch.NewScopeRef(name, r.full, r.source, r.sha)
}

// scopeSource es la fuente de una referencia completa: remota ya descargada o local.
func scopeSource(full string) string {
	if strings.HasPrefix(full, "refs/remotes/") {
		return branch.ScopeSourceRemoteTracking
	}
	return branch.ScopeSourceLocal
}

// scopeTip resuelve la punta de la rama del cambio sin red (D4): `refs/heads/<rama>` y, si no
// existe, `refs/remotes/origin/<rama>` ya descargada.
func (e *Engine) scopeTip(ctx context.Context, gitRoot, name string) (sha string, ref scopeReference, found bool, err error) {
	if name == "" {
		return "", scopeReference{}, false, nil
	}
	for _, full := range []string{"refs/heads/" + name, "refs/remotes/" + originRemote + "/" + name} {
		sha, ok, err := e.git.ResolveCommit(ctx, gitRoot, full)
		if err != nil {
			return "", scopeReference{}, false, err
		}
		if ok {
			return sha, scopeReference{shown: name, full: full, source: scopeSource(full), sha: sha}, true, nil
		}
	}
	return "", scopeReference{}, false, nil
}

// scopeIntegration resuelve la rama de integración sin red: la primera de integrationReferences
// que existe en local (D4).
func (e *Engine) scopeIntegration(ctx context.Context, repo branch.RepoPlan) (scopeReference, bool, error) {
	for _, candidate := range integrationReferences(repo.IntegrationBranch) {
		sha, ok, err := e.git.ResolveCommit(ctx, repo.GitRoot, candidate.full)
		if err != nil {
			return scopeReference{}, false, err
		}
		if ok {
			return scopeReference{shown: candidate.shown, full: candidate.full, source: scopeSource(candidate.full), sha: sha}, true, nil
		}
	}
	return scopeReference{}, false, nil
}
