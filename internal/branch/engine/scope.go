package engine

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// branchGoneMessage es la frase que la spec fija para una rama que no está en local ni descargada
// (RF-BRA-37).
const branchGoneMessage = "la rama ya no está o falta un fetch autorizado"

// Scope calcula, solo con lecturas de git local, qué aporta el cambio en cada repositorio de su
// instantánea de política (RF-BRA-35 a RF-BRA-38, `axiom.branch-scope/v1`): por repositorio, la rama
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
	// Ningún repositorio de la instantánea se omite (RF-BRA-37): uno declarado que no existe en esta
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

	if refusal := e.scopeCommits(ctx, operation, cwd, facts, repo, change, &entry, base, integration, tipRef); refusal != nil {
		return entry, refusal
	}
	return entry, nil
}

// scopeCommits fija el punto de partida estable, los commits propios, los excluidos, los ficheros y
// la fusión en la integración de una rama ya resuelta (D5, D7, D8) y deja en la entrada el
// resultado: `ok`, `empty` o `unverifiable` con `fork_point_undetermined`.
func (e *Engine) scopeCommits(ctx context.Context, operation, cwd string, facts changeFacts, repo branch.RepoPlan, change string, entry *branch.RepositoryScope, base baseReference, integration, tip scopeReference) *branch.Refusal {
	root := repo.GitRoot
	failed := func(what string, err error) *branch.Refusal {
		return gitRefusal(operation, cwd, what+" de "+root, err)
	}
	undetermined := func(reason string) {
		entry.SetUnverifiable(branch.ScopeReasonForkPointUndetermined, reason, "", nil)
	}
	empty := func(start *branch.ScopeStart) {
		entry.Start = start
		entry.SetOutcome(branch.ScopeOutcomeEmpty)
	}

	fB, determined, err := e.firstParentFork(ctx, root, base.sha, tip.sha)
	if err != nil {
		return failed("calcular el punto de bifurcación", err)
	}
	if !determined {
		undetermined(fmt.Sprintf("no se puede fijar el punto de bifurcación de %s respecto de %s: no tienen historia común (¿clon superficial o historias sin relación?)", tip.shown, base.ref))
		return nil
	}
	sameBase := repo.BaseRef == "" && repo.Base() == repo.IntegrationBranch
	start := fB
	startSource := branch.ScopeStartFirstParent
	if fB == tip.sha {
		if !sameBase {
			empty(branch.NewScopeStart(fB, startSource))
			return nil
		}
		// Rama recién creada o fusionada por avance rápido en una sola etapa: solo el reflog de la
		// rama local lo distingue (D6). Sin una creación válida no puede fijarse el punto de partida.
		created, creation, err := e.branchCreation(ctx, root, tip)
		if err != nil {
			return failed("leer el reflog de "+tip.shown, err)
		}
		switch {
		case !creation:
			undetermined(fmt.Sprintf("la rama %s coincide con el punto de bifurcación y su reflog no permite saber si no tiene commits o se fusionó por avance rápido (rama solo remota, adoptada o con el reflog caducado)", tip.shown))
			return nil
		case created == tip.sha:
			empty(branch.NewScopeStart(created, branch.ScopeStartReflog))
			return nil
		}
		ancestor, err := e.git.IsAncestor(ctx, root, created, tip.sha)
		if err != nil {
			return failed("comparar la creación de "+tip.shown+" con su punta", err)
		}
		if !ancestor {
			undetermined(fmt.Sprintf("la creación de la rama %s en el reflog (%s) no es ancestro de su punta: no se puede fijar el punto de partida", tip.shown, created))
			return nil
		}
		fB, start, startSource = created, created, branch.ScopeStartReflog
	}

	revisions := []string{tip.sha, "^" + start}
	var excluded, warnings []string
	if sameBase {
		// La integración que entró en la rama no es del cambio: los commits que van de la bifurcación
		// original a `fB` se excluyen (D5, columna izquierda).
		chain, err := e.git.RevList(ctx, root, true, tip.sha, "^"+fB)
		if err != nil {
			return failed("leer la cadena de primeros padres", err)
		}
		if len(chain) > 0 {
			oldest := chain[len(chain)-1]
			if len(oldest.Parents) > 0 && oldest.Parents[0] != fB {
				if excluded, err = e.revListShas(ctx, root, fB, "^"+oldest.Parents[0]); err != nil {
					return failed("listar los commits excluidos", err)
				}
			}
		}
	} else {
		fI, determinedIntegration, err := e.firstParentFork(ctx, root, integration.sha, tip.sha)
		if err != nil {
			return failed("calcular el punto de bifurcación de la integración", err)
		}
		if !determinedIntegration {
			undetermined(fmt.Sprintf("no se puede fijar el punto de bifurcación de %s respecto de %s: no tienen historia común", tip.shown, integration.shown))
			return nil
		}
		chain, err := e.git.RevList(ctx, root, true, tip.sha, "^"+fB)
		if err != nil {
			return failed("leer la cadena de primeros padres", err)
		}
		onChain := fI != fB && slices.ContainsFunc(chain, func(entry branch.RevEntry) bool { return entry.SHA == fI })
		switch {
		case onChain:
			// La integración entró en la rama por avance rápido (o la rama entró así en la
			// integración): sus commits no se distinguen de los propios. Si `fI` es la propia punta
			// la rama es la integración y no hay nada ajeno que temer.
			if fI != tip.sha {
				warnings = append(warnings, branch.ScopeWarningForeignCommitsPossible)
			}
		default:
			if fI != fB {
				revisions = append(revisions, "^"+fI)
			}
			if excluded, err = e.revListShas(ctx, root, fI, "^"+fB); err != nil {
				return failed("listar los commits excluidos", err)
			}
		}
	}

	commits, err := e.git.CommitFiles(ctx, root, revisions...)
	if err != nil {
		return failed("leer los commits propios", err)
	}
	prefix := changeFolderPrefix(facts, repo, change)
	files, own, relevant := ownFiles(commits, prefix)
	entry.Start = branch.NewScopeStart(start, startSource)
	if !relevant {
		// Sin commits propios, o todos tocan solo la carpeta del cambio (RF-BRA-37).
		entry.SetOutcome(branch.ScopeOutcomeEmpty)
		return nil
	}
	entry.SetOutcome(branch.ScopeOutcomeOK)
	entry.Range, entry.Revisions = start+".."+tip.sha, revisions
	entry.Commits, entry.Excluded, entry.Files, entry.Warnings = own, excluded, files, warnings

	_, evidence, _, err := e.mergeLayers(ctx, root, tip.full, integrationReferences(repo.IntegrationBranch))
	if err != nil {
		return failed("comprobar si la rama "+tip.shown+" está fusionada en "+repo.IntegrationBranch, err)
	}
	switch {
	case evidence.Ancestry:
		entry.SetMerged(true, branch.LayerAncestry)
	case evidence.TreeLayerSupported && evidence.TreeEquivalent:
		entry.SetMerged(true, branch.LayerEquivalentTree)
	default:
		entry.SetMerged(false, "")
	}
	return nil
}

// firstParentFork es el commit de la cadena de primeros padres de ref más reciente que ya contiene
// tip (D5): no se mueve al fusionar tip en ref, con commit de fusión ni con squash, porque la fusión
// y todo commit ajeno posterior a la bifurcación quedan fuera de esa cadena. Si ref ya está contenido
// en tip, es el propio ref. determined es falso cuando la cadena acaba en un commit sin padre
// (historias sin relación o clon superficial).
func (e *Engine) firstParentFork(ctx context.Context, dir, ref, tip string) (sha string, determined bool, err error) {
	entries, err := e.git.RevList(ctx, dir, true, ref, "^"+tip)
	if err != nil {
		return "", false, err
	}
	if len(entries) == 0 {
		return ref, true, nil
	}
	oldest := entries[len(entries)-1]
	if len(oldest.Parents) == 0 {
		return "", false, nil
	}
	return oldest.Parents[0], true, nil
}

// revListShas son los SHAs de `git rev-list <revisiones>`, del más reciente al más antiguo.
func (e *Engine) revListShas(ctx context.Context, dir string, revisions ...string) ([]string, error) {
	entries, err := e.git.RevList(ctx, dir, false, revisions...)
	if err != nil {
		return nil, err
	}
	shas := make([]string, 0, len(entries))
	for _, entry := range entries {
		shas = append(shas, entry.SHA)
	}
	return shas, nil
}

// changeFolderPrefix es la carpeta del cambio (`<WorkspaceRel>/openspec/changes/<cambio>/`) en el
// repositorio del planning home, cuyos ficheros no entran en el alcance; vacía en los demás
// repositorios, que no la tienen (D7).
func changeFolderPrefix(facts changeFacts, repo branch.RepoPlan, change string) string {
	if !pathidentity.SameDirectory(repo.GitRoot, facts.input.PlanningRepo.GitRoot) {
		return ""
	}
	return path.Join(facts.input.WorkspaceRel, "openspec", "changes", change) + "/"
}

// ownFiles reúne los commits propios y la unión ordenada y sin repetidos de sus rutas, sin las de
// prefix. relevant dice si hay algún commit propio que aporte algo: el resultado se decide por los
// commits y no por la lista de ficheros (un commit vacío o de fusión no lista ficheros y cuenta); no
// es relevante el commit que toca únicamente la carpeta del cambio.
func ownFiles(commits []branch.CommitPaths, prefix string) (files []string, own []branch.ScopeCommit, relevant bool) {
	seen := map[string]bool{}
	for _, commit := range commits {
		own = append(own, branch.ScopeCommit{SHA: commit.SHA, Merge: len(commit.Parents) > 1})
		outside := len(commit.Paths) == 0 || len(commit.Parents) > 1
		for _, file := range commit.Paths {
			if prefix != "" && strings.HasPrefix(file, prefix) {
				continue
			}
			outside = true
			if !seen[file] {
				seen[file] = true
				files = append(files, file)
			}
		}
		relevant = relevant || outside
	}
	slices.Sort(files)
	return files, own, relevant
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

// branchCreation lee el commit en que se creó la rama local del cambio (D6). Solo una rama local
// tiene reflog propio: una rama que solo existe como `origin/<rama>` ya descargada no lo tiene, y
// una creación que no es de `git branch` (adopción de `origin/<rama>`, RF-BRA-27) no vale.
func (e *Engine) branchCreation(ctx context.Context, dir string, tip scopeReference) (sha string, ok bool, err error) {
	if tip.source != branch.ScopeSourceLocal {
		return "", false, nil
	}
	sha, _, ok, err = e.git.BranchCreation(ctx, dir, tip.shown)
	return sha, ok, err
}
