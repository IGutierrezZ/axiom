package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Preferencias personales de worktree y modo de trabajo efectivo de `ensure` (RF-BRA-08,
// RF-HUB-07, design D-08). Las preferencias salen del hub, nunca de `axiom.yaml`; el modo
// efectivo de un cambio se resuelve con la prioridad registro > `--mode` > hub > defecto.

// hubPrefs implementa el puerto branch.Prefs sobre `~/.axiom/workspaces.json`.
type hubPrefs struct{}

var _ branch.Prefs = hubPrefs{}

// WorktreePrefs devuelve el modo de trabajo de las preferencias del proyecto cuya carpeta
// maestra es projectMaster, la raíz de worktrees, de dónde sale el modo (`hub` si el hub lo
// guarda, `default` si no) y un aviso. Un hub que no se puede leer, o con un valor guardado
// inválido, no impide trabajar: devuelve los valores por defecto (modo `worktree` y la raíz
// de por defecto) con un aviso que lo explica (design D-08).
func (hubPrefs) WorktreePrefs(projectMaster string) (branch.Mode, string, string, string) {
	manager, err := hub.NewManager("")
	if err != nil {
		return defaultPrefs(projectMaster, err)
	}
	prefs, stored, err := manager.LookupWorktreePrefs(projectMaster)
	if err != nil {
		return defaultPrefs(projectMaster, err)
	}
	source := branch.ModeSourceDefault
	if stored {
		source = branch.ModeSourceHub
	}
	return branch.Mode(prefs.Mode), prefs.Root, source, ""
}

// defaultPrefs son los valores por defecto cuando el hub no se pudo leer: la causa va en un
// aviso.
func defaultPrefs(projectMaster string, cause error) (branch.Mode, string, string, string) {
	warning := fmt.Sprintf("no se pudieron leer las preferencias personales del hub (%v): se usan los valores por defecto (modo %s y la raíz de worktrees por defecto)", cause, branch.ModeWorktree)
	return branch.ModeWorktree, branch.DefaultWorktreeRoot(projectMaster), branch.ModeSourceDefault, warning
}

// workPrefs son las preferencias de worktree del proyecto que lee `gather`: la raíz de
// worktrees entra en el plan y el modo, con su origen, en la resolución de `ensure`.
type workPrefs struct {
	mode    branch.Mode
	root    string
	source  string
	warning string
}

// workPrefs lee las preferencias del proyecto. La identidad del proyecto es la carpeta maestra
// del checkout principal del repositorio del planning home, nunca la de un worktree enlazado
// (design D-08); sin puerto (las pruebas que construyen el motor a mano) rigen los valores
// por defecto sin leer el hub.
func (e *Engine) workPrefs(location workspace.Location, planning branch.RepoFacts) workPrefs {
	master := resolveDirectory(projectMaster(location, planning))
	if e.prefs == nil {
		return workPrefs{mode: branch.ModeWorktree, root: branch.DefaultWorktreeRoot(master), source: branch.ModeSourceDefault}
	}
	mode, root, source, warning := e.prefs.WorktreePrefs(master)
	return workPrefs{mode: mode, root: root, source: source, warning: warning}
}

// projectMaster es la carpeta maestra del proyecto vista desde su checkout principal: la
// que resolvió `Locate` si cwd ya está en él, y, si estaba en un worktree enlazado, la misma
// ruta relativa dentro del checkout principal (`gitcommon.MainWorktreeFromFiles` sobre el git
// common dir). Si el checkout principal no se puede deducir por ficheros, o la maestra queda
// fuera del repositorio del planning home (multirepo), es la que resolvió `Locate`.
func projectMaster(location workspace.Location, planning branch.RepoFacts) string {
	main, known := gitcommon.MainWorktreeFromFiles(planning.CommonDir)
	if !known || pathidentity.SameDirectory(main, planning.GitRoot) {
		return location.Master
	}
	relative, inside := relativeWithin(planning.GitRoot, location.Master)
	if !inside {
		return location.Master
	}
	return filepath.Join(main, filepath.FromSlash(relative))
}

// workMode es el modo de trabajo efectivo de un `ensure` y de dónde sale (branch.ModeSource*).
type workMode struct {
	mode   branch.Mode
	source string
}

// resolveMode resuelve el modo efectivo del cambio con la prioridad registro > `--mode` > hub
// > defecto (design D-08):
//
//   - un cambio con registro conserva su modo y su origen, y un `--mode` distinto es
//     `plan_conflict` (uno igual es idempotente);
//   - sin registro, `--mode` fija el modo: su origen es `answer` si el hub pregunta (es la
//     respuesta a `ask`) y `flag` si no;
//   - sin `--mode`, rige el hub (`hub` si guarda un modo, `default` si no) y, si es `ask`,
//     `decision_required` sin crear nada ni leer la entrada estándar.
//
// El motor nunca escribe la preferencia del hub: la respuesta solo se registra por cambio.
func resolveMode(operation string, r EnsureRequest, facts changeFacts) (workMode, *branch.Refusal) {
	if previous := facts.record; previous != nil {
		if r.Mode != "" && r.Mode != previous.Mode {
			return workMode{}, branch.ModeConflict(operation, r.Change, r.CWD, r.Lane, previous.Mode, r.Mode)
		}
		return workMode{mode: previous.Mode, source: previous.ModeSource}, nil
	}
	prefs := facts.prefs
	if r.Mode != "" {
		source := branch.ModeSourceFlag
		if prefs.mode == branch.ModeAsk {
			source = branch.ModeSourceAnswer
		}
		return workMode{mode: r.Mode, source: source}, nil
	}
	if prefs.mode == branch.ModeAsk {
		return workMode{}, branch.DecisionRequired(ensureArgs(operation, r))
	}
	return workMode{mode: prefs.mode, source: prefs.source}, nil
}

// ensureArgs son los argumentos de la invocación de `ensure` que repiten las continuaciones de
// sus rechazos: el cambio, las excepciones, los `--repo` y los `--promotion` dados. Con
// `--archive` solo cuentan las excepciones: el plan de archivado no admite `--repo` ni
// `--promotion`.
func ensureArgs(operation string, r EnsureRequest) branch.EnsureArgs {
	args := branch.EnsureArgs{Operation: operation, Change: r.Change, CWD: r.CWD, Lane: r.Lane, Mode: r.Mode, Exceptions: r.Exceptions}
	if !r.Archive {
		args.Repos, args.Promotions = r.Repos, r.Promotions
	}
	return args
}

// promotionRequest es lo que condiciona la resolución de la promoción de una operación:
// interactive dice que un modo `ask` sin respuesta se pregunta (`ensure`; `status`, `done` y
// la instantánea no preguntan), requested son los `--promotion` dados y args, la invocación que
// repiten las continuaciones.
type promotionRequest struct {
	interactive bool
	requested   []branch.PromotionRequest
	args        branch.EnsureArgs
}

// planWithPromotion planifica en dos pasos (design D5): un plan provisional sin modos, que da los
// repositorios, sus nombres y sus papeles; la resolución del modo de promoción de cada
// repositorio de dos etapas (branch.ResolvePromotion: registro > instantánea > `--promotion` >
// política > defecto); y el plan definitivo con ese mapa en la entrada, que calcula la base
// efectiva. Sin ningún repositorio de dos etapas devuelve el plan provisional tal cual, de modo
// que sin `main_branch` el plan es idéntico al de siempre.
//
// Rechaza sin escribir nada: un modo pedido que no es una respuesta (`config_invalid`), un
// `--promotion <repo>=…` cuyo repositorio no es del plan (`invalid_name`), el primer repositorio
// cuyo modo materializado contradice el pedido (`plan_conflict`) y, solo en modo interactivo, el
// primer repositorio del plan con el modo `ask` sin respuesta (`decision_required`). Los avisos
// son los de las peticiones sin efecto. facts.input.Promotions queda con el mapa resuelto.
func (e *Engine) planWithPromotion(ctx context.Context, facts *changeFacts, req promotionRequest) (branch.ChangePlan, []string, *branch.Refusal) {
	for _, request := range req.requested {
		if request.Mode != branch.PromotionBranch && request.Mode != branch.PromotionRelease {
			return branch.ChangePlan{}, nil, branch.InvalidPromotion(req.args, string(request.Mode))
		}
	}
	facts.input.Promotions = nil
	provisional, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return branch.ChangePlan{}, nil, refusal
	}

	names := make([]string, 0, len(provisional.Repositories))
	withBranches := 0
	for _, repo := range provisional.Repositories {
		names = append(names, repo.Name)
		if len(repo.Branches) > 0 {
			withBranches++
		}
	}
	for _, request := range req.requested {
		if request.Repository != "" && !slices.Contains(names, request.Repository) {
			return branch.ChangePlan{}, nil, branch.UnknownPromotionRepository(req.args, request.Repository, names)
		}
	}

	known := plannedRepositoryFacts(facts.input)
	resolved := map[string]branch.ResolvedPromotion{}
	var pendingAsk *branch.RepoPlan
	effective := false
	for index, repo := range provisional.Repositories {
		planned := known[repo.GitRoot]
		policy := branch.EffectivePolicy(facts.input.Policy, planned)
		exempt := repo.Role == branch.RepoSpecsInPlace
		promotionFacts := branch.PromotionFacts{
			Exempt:         exempt,
			TwoStage:       repo.TwoStage(),
			Recorded:       recordedPromotion(facts.record, repo),
			Snapshot:       branch.SnapshotPromotionFor(facts.input, planned),
			Requested:      requestedFor(req.requested, repo.Name),
			Declared:       policy.Promotion,
			DeclaredSource: policy.PromotionSource,
		}
		if snapshot := promotionFacts.Snapshot; snapshot != nil && promotionFacts.TwoStage && !exempt && snapshot.Mode == "" && !snapshot.SealedWithMain {
			exists, err := e.git.BranchExists(ctx, repo.GitRoot, changeBranchName(repo))
			if err != nil {
				return branch.ChangePlan{}, nil, gitRefusal(req.args.Operation, req.args.CWD, "comprobar si existe la rama del cambio de "+repo.Name, err)
			}
			promotionFacts.ChangeBranchExists = exists
		}

		resolution := branch.ResolvePromotion(promotionFacts)
		if resolution.Materialized != "" {
			return branch.ChangePlan{}, nil, branch.PromotionConflict(req.args, repo, resolution.Materialized, promotionFacts.Requested, resolution.Resolved.Source)
		}
		if resolution.Resolved.Mode == "" {
			continue
		}
		effective = true
		resolved[repo.GitRoot] = resolution.Resolved
		if resolution.Resolved.Mode == branch.PromotionAsk && pendingAsk == nil {
			pendingAsk = &provisional.Repositories[index]
		}
	}
	warnings := ineffectivePromotions(provisional, req.requested, effective)

	if req.interactive && pendingAsk != nil {
		return branch.ChangePlan{}, nil, branch.PromotionDecisionRequired(req.args, *pendingAsk, withBranches >= 2)
	}
	if len(resolved) == 0 {
		return provisional, warnings, nil
	}
	facts.input.Promotions = resolved
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return branch.ChangePlan{}, nil, refusal
	}
	return plan, warnings, nil
}

// plannedRepositoryFacts son los hechos de cada repositorio de la entrada del planificador, por
// raíz git: el planning home, los de código y los incluidos con `--repo`.
func plannedRepositoryFacts(input branch.PlanInput) map[string]branch.RepoFacts {
	known := map[string]branch.RepoFacts{input.PlanningRepo.GitRoot: input.PlanningRepo}
	for _, group := range [][]branch.RepoFacts{input.CodeRepos, input.IncludedRepos} {
		for _, repo := range group {
			known[repo.GitRoot] = repo
		}
	}
	return known
}

// recordedPromotion es lo que el registro local del cambio sabe de la promoción del repositorio
// del plan (se reconoce por su git common dir), o nil si el registro no lo lista.
func recordedPromotion(record *branch.Record, repo branch.RepoPlan) *branch.RecordedPromotion {
	if record == nil {
		return nil
	}
	for _, registered := range record.Repositories {
		if !pathidentity.SameDirectory(registered.CommonDir, repo.CommonDir) {
			continue
		}
		hasChange := slices.ContainsFunc(registered.Branches, func(recorded branch.RecordBranch) bool {
			return recorded.Kind == branch.KindChange
		})
		return &branch.RecordedPromotion{Mode: registered.Promotion, Source: registered.PromotionSource, HasChangeBranch: hasChange}
	}
	return nil
}

// requestedFor es el `--promotion` que se aplica al repositorio de ese nombre: el cualificado
// prevalece sobre el que no lo está. Vacío si ninguno lo alcanza.
func requestedFor(requests []branch.PromotionRequest, name string) branch.Promotion {
	var unqualified branch.Promotion
	for _, request := range requests {
		switch request.Repository {
		case name:
			return request.Mode
		case "":
			unqualified = request.Mode
		}
	}
	return unqualified
}

// changeBranchName es el nombre de la rama del cambio de un repositorio del plan.
func changeBranchName(repo branch.RepoPlan) string {
	for _, planned := range repo.Branches {
		if planned.Kind == branch.KindChange {
			return planned.Name
		}
	}
	return ""
}

// ineffectivePromotions son los avisos de los `--promotion` que no tienen efecto: el de un
// repositorio de una sola etapa o de specs (`<repo>=<modo>`) y el sin cualificar cuando ningún
// repositorio del plan tiene dos etapas (effective es falso).
func ineffectivePromotions(plan branch.ChangePlan, requests []branch.PromotionRequest, effective bool) []string {
	var warnings []string
	for _, request := range requests {
		if request.Repository == "" {
			if !effective {
				warnings = append(warnings, fmt.Sprintf("--promotion %s no tiene efecto: ningún repositorio del plan tiene git.main_branch distinta de su rama de integración", request.Mode))
			}
			continue
		}
		for _, repo := range plan.Repositories {
			if repo.Name != request.Repository {
				continue
			}
			switch {
			case repo.Role == branch.RepoSpecsInPlace:
				warnings = append(warnings, fmt.Sprintf("--promotion %s=%s no tiene efecto: el repositorio de specs se trabaja in situ y no tiene modo de promoción", request.Repository, request.Mode))
			case !repo.TwoStage():
				warnings = append(warnings, fmt.Sprintf("--promotion %s=%s no tiene efecto: %s tiene una sola etapa (sin git.main_branch distinta de su rama de integración)", request.Repository, request.Mode, request.Repository))
			}
		}
	}
	return warnings
}
