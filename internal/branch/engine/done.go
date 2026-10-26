package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// DoneRequest es la petición de `done`. CWD debe ser un directorio absoluto. Force salta la
// exigencia de fusión confirmada y, en `worktree`, la de árbol limpio (`--force`, REQ-25.17); nunca
// salta la de árbol limpio en `in_place`. Fetch pide `git fetch origin` en cada repositorio del plan
// antes de evaluar (`--fetch`, REQ-25.10): sin él, `done` no hace ninguna operación remota.
type DoneRequest struct {
	CWD    string
	Change string
	Lane   branch.Lane
	Force  bool
	Fetch  bool
}

// Done elimina las ramas de un cambio y sus worktrees una vez fusionadas (REQ-25.16, REQ-25.17,
// design D-09). Cada rama del plan (la de rol, la del cambio y la de archivado, en ese orden) se
// procesa por separado, con el cambio bloqueado en exclusiva como en `ensure`:
//
//  1. reúne los hechos de solo lectura y, con `--fetch`, trae `origin` en cada repositorio antes
//     de evaluar nada;
//  2. si la rama no existe, no hay nada que hacer; si la tiene un worktree o el propio repositorio
//     (`in_place`) con cambios sin confirmar, rechaza con `dirty_worktree`, salvo `--force` en un
//     worktree (en `in_place` `--force` nunca salta la limpieza);
//  3. confirma la fusión con branch.ConfirmMerge contra su destino (la rama de rol, en la de
//     seguimiento local; la del cambio y la de archivado, en cada referencia de integración que
//     existe entre `origin/<integración>` y `<integración>`) y, si ninguna capa la confirma,
//     rechaza con `merge_not_confirmed`, salvo `--force`, que la da por `forced` y deja en el
//     resultado la punta de la rama;
//  4. desmonta: `git worktree remove` desde el checkout principal y, solo si tiene éxito, `git
//     branch -D`; en `in_place`, `git switch <integración>` y `git branch -D`. Si algo falla, la
//     rama se conserva;
//  5. actualiza el registro local: borra de él las ramas eliminadas y el fichero cuando no queda
//     ninguna.
//
// Procesa las ramas que el plan nombra y las que el registro local guarda aunque el plan ya no las
// conozca (el kickoff de un cambio archivado deja de estar en `changes/<cambio>/`). Nunca toca una
// rama remota ni lanza más operación remota que el `fetch` de `--fetch`. El resultado es `ready`
// si todas las ramas se procesaron, `partial` si algunas sí y otras se conservaron, y el rechazo
// de la primera rama conservada si ninguna se pudo procesar; el detalle de cada rama está en su
// resultado. Todo con salida 1 salvo `ready`.
func (e *Engine) Done(ctx context.Context, r DoneRequest) branch.Result {
	const operation = "done"
	refused := func(refusal *branch.Refusal) branch.Result {
		return branch.NewResult(operation).Refused(refusal)
	}

	// El bloqueo se toma en cuanto se conoce el git common dir y se suelta al terminar.
	var unlock func()
	defer func() {
		if unlock != nil {
			unlock()
		}
	}()
	acquire := func(commonDir string) *branch.Refusal {
		release, err := e.registry.Lock(ctx, commonDir, r.Lane, r.Change)
		if err != nil {
			return lockRefusal(err, operation, r.Change, r.CWD)
		}
		unlock = release
		return nil
	}
	facts, refusal := e.gatherUnder(ctx, operation, r.CWD, r.Change, r.Lane, acquire)
	if refusal != nil {
		return refused(refusal)
	}
	// `done` no coloca ningún worktree: el que tiene cada rama lo dice git. Con `InPlace` el plan no
	// calcula rutas, de modo que su colocación y su longitud (`plan_conflict`, `path_too_long`) no
	// pueden impedir eliminar lo que ya existe.
	facts.input.InPlace = true
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return refused(refusal)
	}
	if plan, refusal = e.withArchiveBranch(ctx, operation, r.CWD, facts, plan); refusal != nil {
		return refused(refusal)
	}

	run := doneRun{
		engine: e, ctx: ctx, request: r, facts: facts,
		inPlace: facts.record != nil && facts.record.Mode == branch.ModeInPlace,
	}
	repositories, refusal := run.repositories(plan)
	if refusal != nil {
		return refused(refusal)
	}
	if r.Fetch {
		if refusal := run.fetch(repositories); refusal != nil {
			return refused(refusal)
		}
	}

	result := branch.NewResult(operation)
	result.Change, result.Lane, result.Topology = plan.Change, plan.Lane, facts.input.Topology
	result.WorkspaceRoot, result.PlanningHome = facts.location.WorkspaceRoot, facts.location.PlanningHome
	result.Skipped = plan.Skipped
	if facts.record != nil {
		result.SetMode(facts.record.Mode, facts.record.ModeSource)
	}
	var failures []*branch.Refusal
	removed := false
	var gone []goneBranches
	for _, repo := range repositories {
		described, repoGone, repoFailures, repoRemoved := run.repository(repo)
		result.Repositories = append(result.Repositories, described)
		gone = append(gone, repoGone)
		failures = append(failures, repoFailures...)
		removed = removed || repoRemoved
	}

	present, warning := e.updateRegistry(facts, gone, len(failures) > 0)
	result.Registry = &branch.RegistryResult{Present: present}
	if warning != "" {
		result.Warnings = append(result.Warnings, warning)
	}
	result.Warnings = append(result.Warnings, run.warnings...)
	result.Warnings = append(result.Warnings, facts.warnings...)
	switch {
	case len(failures) == 0:
		return result
	case removed:
		return result.Partial(summarize(failures))
	}
	return result.Refused(summarize(failures))
}

// summarize resume los rechazos de las ramas en el del resultado: el de la primera, con una nota de
// cuántas más se conservaron (el detalle de cada una está en su rama).
func summarize(failures []*branch.Refusal) *branch.Refusal {
	first := *failures[0]
	if len(failures) > 1 {
		first.Message += fmt.Sprintf(" (y %d ramas más sin procesar: el detalle está en cada rama)", len(failures)-1)
	}
	return &first
}

// doneRun es lo que comparten las ramas de un `done`: el contexto, la petición, los hechos del
// cambio y si está materializado en `in_place`.
type doneRun struct {
	engine  *Engine
	ctx     context.Context
	request DoneRequest
	facts   changeFacts
	inPlace bool
	// warnings son los avisos de la ejecución, sin repetir: la capa 2 omitida y los linajes RDD abiertos.
	warnings []string
	// prMerged guarda, tras leerla la primera vez, si el `gates.yaml` del cambio acredita el PR fusionado.
	prMerged *bool
}

// warn añade un aviso a la ejecución si aún no consta.
func (d *doneRun) warn(message string) {
	if !slices.Contains(d.warnings, message) {
		d.warnings = append(d.warnings, message)
	}
}

// doneRepository es un repositorio del plan con las ramas que `done` procesa en él y su checkout
// principal, desde el que se lanza toda escritura.
type doneRepository struct {
	repo     branch.RepoPlan
	main     string
	branches []branch.BranchPlan
}

// goneBranches son las ramas de un repositorio (por git common dir) que ya no existen tras `done`:
// las eliminadas y las que ya no estaban. El registro local deja de listarlas.
type goneBranches struct {
	commonDir string
	names     []string
}

// repositories reúne los repositorios del plan que tienen ramas que procesar, cada uno con su
// checkout principal. El de specs de un multirepo, que se trabaja in situ y no tiene ramas, no
// cuenta.
func (d *doneRun) repositories(plan branch.ChangePlan) ([]doneRepository, *branch.Refusal) {
	var repositories []doneRepository
	for _, repo := range plan.Repositories {
		branches := doneBranches(repo, d.facts.record)
		if len(branches) == 0 {
			continue
		}
		listed := d.facts.worktrees[repo.GitRoot]
		if len(listed) == 0 {
			return nil, gitRefusal("done", d.request.CWD, "listar los worktrees de "+repo.GitRoot, errors.New("git no devolvió ningún worktree"))
		}
		repositories = append(repositories, doneRepository{repo: repo, main: listed[0].Path, branches: branches})
	}
	return repositories, nil
}

// doneBranches son las ramas que `done` procesa en un repositorio: las del plan y las que el
// registro local guarda para él y el plan ya no nombra (el registro es el plan materializado).
// Van las de rol primero, después la del cambio y por último la de archivado (D-09): una rama de rol
// se confirma contra la del cambio, que por eso se elimina después.
func doneBranches(repo branch.RepoPlan, record *branch.Record) []branch.BranchPlan {
	branches := slices.Clone(repo.Branches)
	if record != nil {
		for _, registered := range record.Repositories {
			if !pathidentity.SameDirectory(registered.CommonDir, repo.CommonDir) {
				continue
			}
			changeBranch := ""
			for _, recorded := range registered.Branches {
				if recorded.Kind == branch.KindChange {
					changeBranch = recorded.Name
				}
			}
			for _, recorded := range registered.Branches {
				if slices.ContainsFunc(branches, func(planned branch.BranchPlan) bool { return planned.Name == recorded.Name }) {
					continue
				}
				added := branch.BranchPlan{Name: recorded.Name, Kind: recorded.Kind, RoleID: recorded.RoleID, PRTarget: repo.IntegrationBranch}
				if recorded.Kind == branch.KindRole {
					added.From, added.PRTarget = changeBranch, changeBranch
				}
				branches = append(branches, added)
			}
		}
	}
	rank := map[branch.BranchKind]int{branch.KindRole: 0, branch.KindChange: 1, branch.KindArchive: 2}
	slices.SortStableFunc(branches, func(a, b branch.BranchPlan) int { return rank[a.Kind] - rank[b.Kind] })
	return branches
}

// fetch trae `origin` en cada repositorio del plan con ramas, antes de evaluar ninguna (`--fetch`,
// REQ-25.10). Un fallo es `fetch_failed` y no se ha eliminado nada.
func (d *doneRun) fetch(repositories []doneRepository) *branch.Refusal {
	for _, repository := range repositories {
		if err := d.engine.writer.Fetch(d.ctx, repository.main, originRemote); err != nil {
			return branch.DoneFetchFailed(d.request.Change, d.request.CWD, d.request.Lane, repository.repo, originRemote, err)
		}
	}
	return nil
}

// repository procesa las ramas de un repositorio, una por una y en su orden, y describe el
// resultado: lo hecho con cada rama, las que ya no existen, los rechazos de las conservadas y si
// se eliminó alguna.
func (d *doneRun) repository(repository doneRepository) (branch.RepositoryResult, goneBranches, []*branch.Refusal, bool) {
	described := branch.NewRepositoryResult(repository.repo)
	described.Branches = nil
	gone := goneBranches{commonDir: repository.repo.CommonDir}
	var failures []*branch.Refusal
	removed := false
	for _, planned := range repository.branches {
		outcome, failure := d.branch(repository, planned)
		described.Branches = append(described.Branches, outcome)
		switch {
		case failure != nil:
			failures = append(failures, failure)
		default:
			gone.names = append(gone.names, planned.Name)
			removed = removed || outcome.Outcome == branch.OutcomeRemoved
		}
	}
	// La rama actual, de ficheros, tras haber cambiado de rama en `in_place` (REQ-25.21).
	if name, detached, err := gitcommon.ReadHead(repository.main); err == nil {
		described.CurrentBranch, described.Detached = name, detached
		if detached {
			described.CurrentBranch = ""
		}
	}
	return described, gone, failures, removed
}

// branch procesa una rama del plan (D-09, pasos 2 a 4 de Engine.Done). Devuelve su descripción y, si
// la rama se conservó, el rechazo que lo explica.
func (d *doneRun) branch(repository doneRepository, planned branch.BranchPlan) (branch.BranchResult, *branch.Refusal) {
	repo, main := repository.repo, repository.main
	request := d.request
	described := branch.BranchResult{
		Name: planned.Name, Kind: planned.Kind, KindLabel: branch.Label(string(planned.Kind)),
		Role: planned.RoleID, PRTarget: planned.PRTarget,
	}
	keep := func(refusal *branch.Refusal) (branch.BranchResult, *branch.Refusal) {
		described.Keep(refusal)
		return described, refusal
	}
	failed := func(gitOperation, fix string, err error) (branch.BranchResult, *branch.Refusal) {
		return keep(branch.DoneGitFailed(request.Change, request.CWD, request.Lane, main, gitOperation, fix, err))
	}
	readFix := fmt.Sprintf("comprueba que git está instalado y que puedes leer %s", pathquote.Quote(main))

	exists, err := d.engine.git.BranchExists(d.ctx, main, planned.Name)
	if err != nil {
		return failed("comprobar si existe la rama "+planned.Name, readFix, err)
	}
	if !exists {
		described.SetOutcome(branch.OutcomeAbsent)
		return described, nil
	}
	described.Exists = true

	// Dónde está desplegada, ahora: una rama anterior de esta misma ejecución pudo cambiar el
	// checkout principal.
	listed, err := d.engine.git.Worktrees(d.ctx, main)
	if err != nil {
		return failed("listar los worktrees de "+main, readFix, err)
	}
	var deployed *branch.WorktreeEntry
	for i := range listed {
		// Las vistas de candidato de RDD son worktrees desacoplados: no cuentan (D-07).
		if listed[i].Branch == planned.Name && !listed[i].Detached {
			deployed = &listed[i]
			break
		}
	}
	inMain := deployed != nil && d.engine.samePlace(deployed.Path, main)
	linked := deployed != nil && !inMain
	if linked {
		described.Worktree = deployed.Path
	}

	// Árbol limpio: el worktree de la rama o, en `in_place`, el propio repositorio.
	var dirty []string
	switch {
	case inMain && !d.inPlace:
		return keep(branch.DoneCheckedOutInMainCheckout(request.Change, request.CWD, request.Lane, repo, planned.Name, main))
	case inMain || linked:
		dir := main
		if linked {
			dir = deployed.Path
		}
		dirty, err = d.engine.git.DirtyPaths(d.ctx, dir)
		if err != nil {
			return failed("leer el estado de "+dir, fmt.Sprintf("comprueba %s (si su directorio ya no existe, `git -C %s worktree prune` elimina su registro)", pathquote.Quote(dir), pathquote.Quote(main)), err)
		}
		described.Dirty = len(dirty) > 0
		// En `in_place`, `--force` nunca salta la limpieza: el cambio de rama pondría en riesgo esos cambios.
		if len(dirty) > 0 && (inMain || !request.Force) {
			return keep(branch.DoneDirtyWorktree(request.Change, request.CWD, request.Lane, repo.Name, planned.Name, dir, dirty, inMain))
		}
	}

	targets, evidence, err := d.mergeEvidence(main, planned, repo)
	if err != nil {
		return failed("comprobar si la rama "+planned.Name+" está fusionada", readFix, err)
	}
	layer, confirmed := branch.ConfirmMerge(planned.Kind, evidence, request.Force)
	if !confirmed {
		return keep(branch.MergeNotConfirmed(request.Change, request.CWD, request.Lane, repo, planned, targets))
	}
	described.SetConfirmedBy(layer)
	// La punta, antes de borrar nada: sin fusión confirmada es lo único que permite recuperar la rama.
	if layer == branch.LayerForced {
		tip, found, err := d.engine.git.ResolveCommit(d.ctx, main, "refs/heads/"+planned.Name)
		if err != nil || !found {
			if err == nil {
				err = errors.New("la rama no se resuelve a ningún commit")
			}
			return failed("leer la punta de la rama "+planned.Name, readFix, err)
		}
		described.TipSha = tip
	}

	// Un linaje RDD abierto en lo que se va a eliminar solo avisa: no impide la operación (REQ-25.17).
	if linked || inMain {
		root := main
		if linked {
			root = deployed.Path
		}
		d.warnOfOpenLineages(planned.Name, root, main)
	}

	switch {
	case linked:
		if err := d.engine.writer.RemoveWorktree(d.ctx, main, deployed.Path, len(dirty) > 0); err != nil {
			return failed(fmt.Sprintf("retirar el worktree %s de la rama %s", deployed.Path, planned.Name), removeFix(main, deployed.Path), err)
		}
		described.Discarded = dirty
	case inMain:
		if err := d.engine.writer.Switch(d.ctx, main, repo.IntegrationBranch); err != nil {
			return failed(fmt.Sprintf("cambiar el repositorio %s a la rama %s", repo.Name, repo.IntegrationBranch),
				fmt.Sprintf("comprueba que existe la rama local %s en %s", repo.IntegrationBranch, pathquote.Quote(main)), err)
		}
	}
	if err := d.engine.writer.DeleteBranch(d.ctx, main, planned.Name); err != nil {
		return failed("eliminar la rama "+planned.Name, readFix, err)
	}
	described.Exists, described.Dirty = false, false
	described.SetOutcome(branch.OutcomeRemoved)
	return described, nil
}

// removeFix es la acción que resuelve un worktree que no se pudo retirar: en Windows, un worktree
// que es el directorio actual de algún proceso no se puede borrar, de modo que se repite desde otro
// directorio y con `--cwd` sobre el checkout principal (D-09); si git ya no lo considera un
// worktree, `git worktree prune` limpia su registro.
func removeFix(main, worktree string) string {
	return fmt.Sprintf("comprueba que ningún proceso tiene %s como directorio actual (en Windows impide borrarlo) y ejecuta la orden desde otro directorio con `--cwd` sobre el checkout principal %s; si git ya no lo reconoce como worktree, `git -C %s worktree prune` limpia su registro",
		pathquote.Quote(worktree), pathquote.Quote(main), pathquote.Quote(main))
}

// mergeEvidence reúne la evidencia de fusión de una rama contra su destino (D-09): las referencias
// que existen donde buscar su fusión (targets, para el mensaje) y el resultado de las tres capas, que
// se evalúan por orden y solo hasta la primera que confirma, porque ConfirmMerge se detiene en ella:
// la ascendencia contra cada destino; si ninguno la confirma, el árbol equivalente contra cada
// destino (si git lo admite; si no, la capa se omite con un aviso); y, si tampoco, el `gates.yaml` del
// cambio, salvo para la rama de archivado, que no admite la capa 3. Usa solo referencias locales.
func (d *doneRun) mergeEvidence(main string, planned branch.BranchPlan, repo branch.RepoPlan) (targets []string, evidence branch.MergeEvidence, err error) {
	type reference struct{ shown, full string }
	candidates := []reference{
		{originRemote + "/" + repo.IntegrationBranch, "refs/remotes/" + originRemote + "/" + repo.IntegrationBranch},
		{repo.IntegrationBranch, "refs/heads/" + repo.IntegrationBranch},
	}
	// Una rama de rol se confirma contra la de seguimiento, que es local.
	if planned.Kind == branch.KindRole {
		candidates = []reference{{planned.From, "refs/heads/" + planned.From}}
	}
	var existing []string
	for _, candidate := range candidates {
		if candidate.shown == "" {
			continue
		}
		_, found, err := d.engine.git.ResolveCommit(d.ctx, main, candidate.full)
		if err != nil {
			return nil, evidence, err
		}
		if !found {
			continue
		}
		targets = append(targets, candidate.shown)
		existing = append(existing, candidate.full)
	}
	branchRef := "refs/heads/" + planned.Name

	// Capa 1: ascendencia.
	for _, target := range existing {
		if evidence.Ancestry, err = d.engine.git.IsAncestor(d.ctx, main, branchRef, target); err != nil {
			return nil, evidence, err
		}
		if evidence.Ancestry {
			return targets, evidence, nil
		}
	}

	// Capa 2: árbol equivalente. Sin soporte se omite sin error y con un aviso (Q1).
	for _, target := range existing {
		equivalent, supported, err := d.engine.git.MergeTreeEquivalent(d.ctx, main, target, branchRef)
		if err != nil {
			return nil, evidence, err
		}
		if !supported {
			d.warn("la capa de árbol equivalente (`git merge-tree --write-tree`) se omite porque git no la admite (requiere git 2.38 o superior): un merge por squash o rebase solo se confirma con evidencia `pr_merged` en gates.yaml o con --force")
			break
		}
		evidence.TreeLayerSupported = true
		if evidence.TreeEquivalent = equivalent; equivalent {
			return targets, evidence, nil
		}
	}

	// Capa 3: evidencia `pr_merged` del cambio. No aplica a la rama de archivado (REQ-25.16).
	if planned.Kind != branch.KindArchive {
		evidence.PRMerged = d.prMergedEvidence()
	}
	return targets, evidence, nil
}

// prMergedEvidence dice si el `gates.yaml` del cambio, activo o archivado (kickoff.ArchivedChangeRoot),
// tiene un registro aprobado de la compuerta `integration` con evidencia `pr_merged` ya verificada
// (REQ-25.16, capa 3). Se lee la primera vez que hace falta. Un `gates.yaml` que no se puede leer no
// es evidencia: se avisa y la capa no confirma.
func (d *doneRun) prMergedEvidence() bool {
	if d.prMerged != nil {
		return *d.prMerged
	}
	found := false
	location, change := d.facts.location, d.request.Change
	roots := []string{filepath.Join(location.PlanningHome, "changes", change)}
	if archived, ok := kickoff.ArchivedChangeRoot(location.WorkspaceRoot, change); ok {
		roots = append(roots, archived)
	}
	for _, root := range roots {
		ledger, err := kickoff.LoadGates(root)
		if err != nil {
			d.warn(fmt.Sprintf("no se pudo leer gates.yaml para la capa de PR fusionado (%v): no se toma como evidencia", err))
			continue
		}
		for _, record := range ledger.Records {
			if record.Gate == kickoff.GateIntegration && record.Decision == kickoff.DecisionApproved &&
				record.EvidenceKind == kickoff.EvidencePRMerged && record.Verified {
				found = true
			}
		}
	}
	d.prMerged = &found
	return found
}

// warnOfOpenLineages avisa de cada linaje RDD abierto en el worktree que se va a eliminar y nombra
// `axiom review abandon`, que libera su autoridad (REQ-25.17, A-4). Es información: ni un fallo de
// la sonda ni un linaje abierto impiden la operación. El aviso se compone con el checkout principal,
// porque el worktree dejará de existir. Sin sonda no consulta nada.
func (d *doneRun) warnOfOpenLineages(name, worktreeRoot, main string) {
	if d.engine.lineages == nil {
		return
	}
	lineages, err := d.engine.lineages.OpenLineages(d.ctx, worktreeRoot)
	if err != nil {
		d.warn(fmt.Sprintf("no se pudo comprobar si hay linajes RDD abiertos en %s (%v): la rama %s se elimina igualmente", worktreeRoot, err, name))
		return
	}
	for _, lineage := range lineages {
		d.warn(fmt.Sprintf("el worktree %s de la rama %s tiene un linaje RDD abierto (%s) y se elimina igualmente: libera su autoridad con `axiom review abandon --cwd %s --lineage %s` (ejecuta `axiom review abandon` sin banderas para ver la plantilla de la orden)",
			worktreeRoot, name, lineage, pathquote.Quote(main), lineage))
	}
}

// updateRegistry actualiza el registro local con lo que `done` hizo (D-09): borra de él las ramas
// que ya no existen y el fichero cuando no queda ninguna, salvo que alguna rama se haya conservado
// (el registro es lo que dice de dónde salió). present dice si el cambio sigue teniendo registro. Un
// fallo al escribirlo no deshace nada, porque el registro es estado derivado que `ensure`
// reconstruye: se devuelve como aviso.
func (e *Engine) updateRegistry(facts changeFacts, gone []goneBranches, anyKept bool) (present bool, warning string) {
	record := facts.record
	if record == nil {
		return false, ""
	}
	commonDir := facts.input.PlanningRepo.CommonDir
	updated := *record
	updated.Repositories = nil
	for _, registered := range record.Repositories {
		var names []string
		for _, repo := range gone {
			if pathidentity.SameDirectory(repo.commonDir, registered.CommonDir) {
				names = append(names, repo.names...)
			}
		}
		registered.Branches = slices.DeleteFunc(slices.Clone(registered.Branches), func(recorded branch.RecordBranch) bool {
			return slices.Contains(names, recorded.Name)
		})
		if len(registered.Branches) > 0 {
			updated.Repositories = append(updated.Repositories, registered)
		}
	}
	var err error
	switch {
	case len(updated.Repositories) == 0 && !anyKept:
		err = e.registry.Delete(commonDir, record.Lane, record.Change)
		present = err != nil
	case len(updated.Repositories) == 0:
		// Se conservó alguna rama que el registro no guarda: se deja como estaba.
		present = true
	default:
		err = e.registry.Save(commonDir, updated)
		present = true
	}
	if err != nil {
		return true, fmt.Sprintf("no se pudo actualizar el registro local de %s (%v): es estado local derivado y `axiom branch ensure --change %s` lo reconstruye", record.Change, err, record.Change)
	}
	return present, ""
}
