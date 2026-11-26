package engine

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/nametemplate"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Relevancia del autofetch (RF-BRA-39 paso 3, design D-07): resuelve el cambio actual y calcula
// los avisos (a), commits entrantes de la integración que tocan lo del cambio, y (b), commits
// nuevos en las ramas del cambio. Solo lee referencias locales y se calcula antes del avance de
// la integración, de modo que el rango «local frente a remota» se deduplica solo. Nunca mueve
// una rama ni trae los repositorios de código.

// autofetchHandoffLimit acota el listado de ramas remotas de la cesión sin registro (D-07, regla 3).
const autofetchHandoffLimit = 50

// autofetchRelevance son los avisos de relevancia de un ciclo, ya ordenados: primero los de la
// integración y después los de las ramas del cambio (D-07, punto 5). El ciclo los añade al
// resultado en ese orden, con los de no avance de T6 entre unos y otros.
type autofetchRelevance struct {
	incoming []branch.AutofetchNotice
	branches []branch.AutofetchNotice
}

// currentChange es el cambio actual resuelto y las ramas que se vigilan de él.
type currentChange struct {
	lane branch.Lane
	name string
	// repositories son los repositorios del registro del cambio (o, en una cesión sin registro,
	// el de specs con la rama esperada).
	repositories []branch.RecordRepo
}

// autofetchRelevance resuelve el cambio actual y calcula los avisos. Fija result.Change. Un fallo
// de lectura no aborta el ciclo: queda como aviso y se omite lo que no pudo calcularse.
func (e *Engine) autofetchRelevance(ctx context.Context, project AutofetchProject, result *branch.AutofetchResult) autofetchRelevance {
	var relevance autofetchRelevance
	current, ok := e.resolveCurrentChange(ctx, project, result)
	if !ok {
		return relevance
	}
	result.Change = &branch.AutofetchChange{Lane: current.lane, Name: current.name}

	prefixes := e.relevantPrefixes(ctx, project, current, result)
	relevance.incoming = e.incomingNotices(ctx, project, current, prefixes, result)
	relevance.branches = e.changeBranchNotices(ctx, project, current, result)
	return relevance
}

// resolveCurrentChange aplica las cuatro reglas de D-07 y se detiene en la primera que da un
// resultado. Con varios candidatos (o ninguno) no hay cambio actual y no se avisa.
func (e *Engine) resolveCurrentChange(ctx context.Context, project AutofetchProject, result *branch.AutofetchResult) (currentChange, bool) {
	var records []branch.Record
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		listed, err := e.registry.List(project.CommonDir, lane)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo leer el registro de cambios (%v): no se calcula la relevancia", err))
			return currentChange{}, false
		}
		records = append(records, listed...)
	}

	// Regla 1: el cambio activo que tiene la rama de HEAD en el repositorio de cwd.
	if head, detached, err := gitcommon.ReadHead(project.GitRoot); err == nil && !detached {
		var matches []branch.Record
		for _, record := range records {
			if recordOwnsBranch(record, project, head) {
				matches = append(matches, record)
			}
		}
		switch len(matches) {
		case 0:
		case 1:
			return currentOf(matches[0]), true
		default:
			return currentChange{}, false
		}
	}
	// Regla 2: un único cambio activo registrado.
	if len(records) == 1 {
		return currentOf(records[0]), true
	}
	if len(records) > 1 {
		return currentChange{}, false
	}
	// Regla 3: ningún cambio registrado; cesión sin registro ni rama local.
	return e.handedOffChange(ctx, project, result)
}

func currentOf(record branch.Record) currentChange {
	return currentChange{lane: record.Lane, name: record.Change, repositories: record.Repositories}
}

// recordOwnsBranch dice si el registro tiene, en el repositorio de cwd, una rama de ese nombre.
func recordOwnsBranch(record branch.Record, project AutofetchProject, name string) bool {
	for _, repo := range record.Repositories {
		if !sameRepository(repo, project) {
			continue
		}
		if slices.ContainsFunc(repo.Branches, func(registered branch.RecordBranch) bool { return registered.Name == name }) {
			return true
		}
	}
	return false
}

func sameRepository(repo branch.RecordRepo, project AutofetchProject) bool {
	return pathidentity.SameDirectory(repo.CommonDir, project.CommonDir) || pathidentity.SameDirectory(repo.GitRoot, project.GitRoot)
}

// handedOffChange es la regla 3 de D-07: con exactamente una rama `origin/<prefijo>/<c>` sin rama
// local cuyo árbol contiene `<planRel>/changes/<c>/kickoff.yaml`, `<c>` es el cambio actual
// (carril SDD). Se leen las primeras autofetchHandoffLimit referencias.
func (e *Engine) handedOffChange(ctx context.Context, project AutofetchProject, result *branch.AutofetchResult) (currentChange, bool) {
	policy := policyOf(project)
	prefix := policy.BranchPrefix
	planRel, inside := relativeWithin(project.GitRoot, project.Location.PlanningHome)
	if !inside {
		return currentChange{}, false
	}
	if policy.BranchTemplate != "" {
		return e.templatedHandedOffChange(ctx, project, policy, planRel, result)
	}
	refs, err := e.git.RemoteBranches(ctx, project.Checkout, prefix)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudieron listar las ramas remotas (%v): no se calcula la relevancia", err))
		return currentChange{}, false
	}
	if len(refs) > autofetchHandoffLimit {
		refs = refs[:autofetchHandoffLimit]
	}
	var candidates []string
	for _, ref := range refs {
		name, ok := strings.CutPrefix(ref, "refs/remotes/origin/"+prefix+"/")
		if !ok || name == "" || strings.Contains(name, "/") || strings.HasSuffix(name, branch.ArchiveSuffix) {
			continue
		}
		if local, err := e.git.BranchExists(ctx, project.Checkout, branch.ResolveChangeBranch(name, prefix)); err != nil || local {
			continue
		}
		kickoff := path.Join(planRel, "changes", name, "kickoff.yaml")
		if _, exists, err := e.git.FileAt(ctx, project.Checkout, ref, kickoff); err != nil || !exists {
			continue
		}
		candidates = append(candidates, name)
	}
	if len(candidates) != 1 {
		return currentChange{}, false
	}
	name := candidates[0]
	return handedOffOf(project, name, branch.ResolveChangeBranch(name, prefix)), true
}

func handedOffOf(project AutofetchProject, change, changeBranch string) currentChange {
	return currentChange{
		lane: branch.LaneSDD,
		name: change,
		repositories: []branch.RecordRepo{{
			Name:     path.Base(project.Checkout),
			GitRoot:  project.Checkout,
			Branches: []branch.RecordBranch{{Name: changeBranch, Kind: branch.KindChange}},
		}},
	}
}

// templatedHandedOffChange es la regla 3 con plantilla (D-16): lista las referencias remotas bajo
// el prefijo literal de la plantilla (con la cota autofetchHandoffLimit en total) y toma el cambio
// cuya instantánea (`kickoff.yaml` de esa referencia) nombra esa misma rama como rama del cambio.
// Solo lee referencias ya traídas: no crea ninguna rama local. Se detiene en el segundo candidato.
// Sin prefijo literal la regla no se aplica y queda un aviso.
func (e *Engine) templatedHandedOffChange(ctx context.Context, project AutofetchProject, policy branch.RepoPolicy, planRel string, result *branch.AutofetchResult) (currentChange, bool) {
	parsed, err := nametemplate.Parse(policy.BranchTemplate)
	if err != nil {
		return currentChange{}, false
	}
	prefixes := nametemplate.LiteralPrefixes(parsed, policy.BranchPrefix)
	if len(prefixes) == 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("la plantilla de rama %q no tiene prefijo literal: no se reconoce un cambio cedido sin registro", policy.BranchTemplate))
		return currentChange{}, false
	}
	var refs []string
	for _, prefix := range prefixes {
		listed, err := e.git.RemoteBranches(ctx, project.Checkout, prefix)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudieron listar las ramas remotas (%v): no se calcula la relevancia", err))
			return currentChange{}, false
		}
		refs = append(refs, listed...)
	}
	slices.Sort(refs)
	refs = slices.Compact(refs)
	if len(refs) > autofetchHandoffLimit {
		refs = refs[:autofetchHandoffLimit]
	}
	changesDir := path.Join(planRel, "changes")
	var found []currentChange
	for _, ref := range refs {
		name, ok := strings.CutPrefix(ref, "refs/remotes/origin/")
		if !ok || name == "" || strings.HasSuffix(name, branch.ArchiveSuffix) {
			continue
		}
		if local, err := e.git.BranchExists(ctx, project.Checkout, name); err != nil || local {
			continue
		}
		dirs, _, err := e.git.TreeDirs(ctx, project.Checkout, ref, changesDir)
		if err != nil {
			continue
		}
		for _, dir := range dirs {
			content, exists, err := e.git.FileAt(ctx, project.Checkout, ref, path.Join(changesDir, dir, "kickoff.yaml"))
			if err != nil || !exists {
				continue
			}
			sealed, err := kickoff.ParseKickoff(content)
			if err != nil || sealed.Git == nil || sealed.Git.ChangeBranch != name {
				continue
			}
			found = append(found, handedOffOf(project, dir, name))
			if len(found) > 1 {
				return currentChange{}, false
			}
		}
	}
	if len(found) != 1 {
		return currentChange{}, false
	}
	return found[0], true
}

// policyOf es la política de ramas del repositorio de specs del proyecto.
func policyOf(project AutofetchProject) branch.RepoPolicy {
	policy := branch.ResolvePolicy(project.Location.Config)
	if relative, inside := relativeWithin(project.Location.Master, project.GitRoot); inside {
		return policy.For(relative)
	}
	return branch.RepoPolicy{IntegrationBranch: policy.IntegrationBranch, BranchPrefix: policy.BranchPrefix}
}

// relevantPrefixes son las rutas, relativas a la raíz git del repositorio de specs, que hacen
// relevante un commit para el cambio (D-07, punto 2). Salen ordenadas y sin repetir.
func (e *Engine) relevantPrefixes(ctx context.Context, project AutofetchProject, current currentChange, result *branch.AutofetchResult) []string {
	if current.lane == branch.LaneODD {
		wsRel, inside := relativeWithin(project.GitRoot, project.Location.WorkspaceRoot)
		if !inside {
			return nil
		}
		return []string{path.Join(wsRel, "odd", "tasks", current.name+".md")}
	}
	planRel, inside := relativeWithin(project.GitRoot, project.Location.PlanningHome)
	if !inside {
		return nil
	}
	changeDir := path.Join(planRel, "changes", current.name)
	prefixes := []string{changeDir}
	capabilities := map[string]bool{}
	revisions := []string{"refs/remotes/origin/" + integrationBranchOf(project.Location, project.GitRoot)}
	for _, repo := range current.repositories {
		if sameRepository(repo, project) || pathidentity.SameDirectory(repo.GitRoot, project.Checkout) {
			for _, registered := range repo.Branches {
				revisions = append(revisions, "refs/heads/"+registered.Name)
			}
		}
	}
	for _, revision := range revisions {
		if resolved, ok, err := e.git.ResolveCommit(ctx, project.Checkout, revision); err != nil || !ok {
			continue
		} else {
			dirs, _, err := e.git.TreeDirs(ctx, project.Checkout, resolved, path.Join(changeDir, "specs"))
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudieron leer las capacidades de %s en %s (%v)", current.name, revision, err))
				continue
			}
			for _, capability := range dirs {
				capabilities[capability] = true
			}
		}
	}
	for capability := range capabilities {
		prefixes = append(prefixes, path.Join(planRel, "specs", capability))
	}
	slices.Sort(prefixes)
	return slices.Compact(prefixes)
}

// underPrefix dice si una ruta es el prefijo o está dentro de él.
func underPrefix(file, prefix string) bool {
	return file == prefix || strings.HasPrefix(file, prefix+"/")
}

// incomingNotices es el aviso (a): commits que no son de fusión y tocan algún prefijo relevante
// en el rango `refs/heads/<int>..refs/remotes/origin/<int>`. Sin rama local de integración, o
// sin remota, se omite. La orden usa el SHA local previo, que sigue siendo válido tras el avance.
func (e *Engine) incomingNotices(ctx context.Context, project AutofetchProject, current currentChange, prefixes []string, result *branch.AutofetchResult) []branch.AutofetchNotice {
	if len(prefixes) == 0 {
		return nil
	}
	integration := integrationBranchOf(project.Location, project.GitRoot)
	local, haveLocal, err := e.git.ResolveCommit(ctx, project.Checkout, "refs/heads/"+integration)
	if err != nil || !haveLocal {
		return nil
	}
	remote, haveRemote, err := e.git.ResolveCommit(ctx, project.Checkout, "refs/remotes/origin/"+integration)
	if err != nil || !haveRemote || local == remote {
		return nil
	}
	commits, err := e.git.CommitFiles(ctx, project.Checkout, remote, "^"+local)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudieron leer los commits entrantes de origin/%s (%v)", integration, err))
		return nil
	}
	count := 0
	touched := map[string]bool{}
	for _, commit := range commits {
		if len(commit.Parents) > 1 {
			continue
		}
		relevant := false
		for _, file := range commit.Paths {
			for _, prefix := range prefixes {
				if underPrefix(file, prefix) {
					relevant, touched[prefix] = true, true
				}
			}
		}
		if relevant {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	scope := make([]string, 0, len(touched))
	for prefix := range touched {
		scope = append(scope, prefix)
	}
	slices.Sort(scope)
	quoted := make([]string, len(scope))
	for i, prefix := range scope {
		quoted[i] = pathquote.Quote(prefix)
	}
	remoteName := originRemote + "/" + integration
	return []branch.AutofetchNotice{{
		Kind:    branch.AutofetchNoticeIncomingRelevant,
		Branch:  remoteName,
		Commits: count,
		Scope:   scope,
		Message: fmt.Sprintf("%s trae %d commits que tocan el cambio %s (%s)", remoteName, count, current.name, strings.Join(scope, ", ")),
		Command: fmt.Sprintf("git -C %s log --oneline %s..%s -- %s", pathquote.Quote(project.Checkout), local, remoteName, strings.Join(quoted, " ")),
	}}
}

// changeBranchNotices es el aviso (b): por cada rama del registro del cambio, repositorio y nombre
// ordenados, los commits que la remota ya traída tiene y la local no, o la rama pendiente de
// adoptar. Solo lee referencias locales: la rama del cambio nunca se mueve.
func (e *Engine) changeBranchNotices(ctx context.Context, project AutofetchProject, current currentChange, result *branch.AutofetchResult) []branch.AutofetchNotice {
	repositories := slices.Clone(current.repositories)
	slices.SortFunc(repositories, func(a, b branch.RecordRepo) int { return strings.Compare(a.Name, b.Name) })
	var notices []branch.AutofetchNotice
	for _, repo := range repositories {
		if directoryProblem(repo.GitRoot) != "" {
			continue
		}
		registered := slices.Clone(repo.Branches)
		slices.SortFunc(registered, func(a, b branch.RecordBranch) int { return strings.Compare(a.Name, b.Name) })
		for _, planned := range registered {
			if notice, ok := e.changeBranchNotice(ctx, project, current, repo, planned, result); ok {
				notices = append(notices, notice)
			}
		}
	}
	return notices
}

func (e *Engine) changeBranchNotice(ctx context.Context, project AutofetchProject, current currentChange, repo branch.RecordRepo, planned branch.RecordBranch, result *branch.AutofetchResult) (branch.AutofetchNotice, bool) {
	dir := repo.GitRoot
	remoteRef := "refs/remotes/origin/" + planned.Name
	remoteName := originRemote + "/" + planned.Name
	if _, haveRemote, err := e.git.ResolveCommit(ctx, dir, remoteRef); err != nil || !haveRemote {
		return branch.AutofetchNotice{}, false
	}
	localRef := "refs/heads/" + planned.Name
	_, haveLocal, err := e.git.ResolveCommit(ctx, dir, localRef)
	if err != nil {
		return branch.AutofetchNotice{}, false
	}
	if !haveLocal {
		return e.notAdoptedNotice(ctx, project, current, repo, planned, remoteRef, remoteName), true
	}
	ahead, behind, err := e.git.AheadBehind(ctx, dir, localRef, remoteRef)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo comparar %s con %s (%v)", planned.Name, remoteName, err))
		return branch.AutofetchNotice{}, false
	}
	if ahead == 0 {
		return branch.AutofetchNotice{}, false
	}
	where := dir
	if planned.Worktree != "" {
		where = planned.Worktree
	}
	command := fmt.Sprintf("git -C %s merge --ff-only %s", pathquote.Quote(where), remoteName)
	if behind > 0 {
		command = fmt.Sprintf("git -C %s log --oneline %s..%s", pathquote.Quote(where), planned.Name, remoteName)
	}
	return branch.AutofetchNotice{
		Kind:    branch.AutofetchNoticeChangeBranchBehind,
		Branch:  planned.Name,
		Commits: ahead,
		Message: fmt.Sprintf("la rama %s del cambio %s tiene %d commits nuevos en %s", planned.Name, current.name, ahead, remoteName),
		Command: command,
	}, true
}

// notAdoptedNotice es el aviso de una rama del cambio que existe solo como referencia remota (una
// cesión pendiente de adoptar): cuenta los commits que tiene sobre la integración y propone
// `axiom branch ensure`, que la adopta.
func (e *Engine) notAdoptedNotice(ctx context.Context, project AutofetchProject, current currentChange, repo branch.RecordRepo, planned branch.RecordBranch, remoteRef, remoteName string) branch.AutofetchNotice {
	commits := 0
	integration := integrationBranchOf(project.Location, repo.GitRoot)
	for _, base := range []string{"refs/heads/" + integration, "refs/remotes/origin/" + integration} {
		if _, ok, err := e.git.ResolveCommit(ctx, repo.GitRoot, base); err != nil || !ok {
			continue
		}
		if entries, err := e.git.RevList(ctx, repo.GitRoot, false, remoteRef, "^"+base); err == nil {
			commits = len(entries)
		}
		break
	}
	laneFlag := ""
	if current.lane == branch.LaneODD {
		laneFlag = " --lane " + string(branch.LaneODD)
	}
	return branch.AutofetchNotice{
		Kind:    branch.AutofetchNoticeChangeBranchNotAdopted,
		Branch:  planned.Name,
		Commits: commits,
		Message: fmt.Sprintf("la rama %s del cambio %s existe en %s y aún no está adoptada en local", planned.Name, current.name, remoteName),
		Command: fmt.Sprintf("axiom branch ensure --change %s%s --cwd %s", current.name, laneFlag, pathquote.Quote(project.Master)),
	}
}
