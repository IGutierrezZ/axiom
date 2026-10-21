package engine

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// `ensure --archive` (REQ-25.45, design D-25): crea, desde la rama de integración y con el cambio
// ya fusionado, la rama de archivado `<prefijo>/<cambio>-archive` del repositorio del planning
// home y, según el modo de trabajo del cambio, su worktree o su checkout. Es un plan aparte con
// las mismas garantías que el del cambio (design D-05): comprobación sin efectos antes de mutar
// nada, solo lo que falta, registro que solo se reescribe si cambian sus bytes e idempotencia byte
// a byte. Nunca toca las ramas del cambio ni las de rol y nunca publica la rama.
//
// La secuencia es la de Engine.Ensure; lo propio del archivado está aquí:
//
//   - aplicabilidad y banderas (`archive_branch_not_applicable`, `plan_conflict`): useArchivePlan,
//     antes de resolver el modo, de modo que una petición que no se puede atender no pregunta;
//   - nombre y colisión con el rol `archive` (S-15): el planificador (branch.Plan con Archive);
//   - base: la de integración de D-06, con `--fetch` opcional, como la del cambio;
//   - comprobación de integración (`change_not_integrated`): checkIntegrated;
//   - modo: el registrado del cambio, también la respuesta a `ask` (REQ-25.8), y, sin registro, el
//     personal; `in_place` hace `git switch` y rechaza el árbol sucio (REQ-25.9);
//   - registro de la rama con `kind: archive` y del prefijo de archivado: newArchiveRecord.

// archiveCommand es la orden de `axiom branch` que repiten las continuaciones del archivado.
const archiveCommand = "ensure --archive"

// useArchivePlan sustituye, en los hechos de un cambio, la entrada del plan ordinario por la del
// archivado y rechaza la petición que no se puede atender: el carril o la topología sin rama de
// archivado y las banderas que no se admiten con `--archive` (branch.CheckArchiveRequest).
func useArchivePlan(facts *changeFacts, r EnsureRequest) *branch.Refusal {
	input := archivePlanInput(*facts, r.Exceptions)
	if refusal := branch.CheckArchiveRequest(input); refusal != nil {
		return refusal
	}
	facts.input = input
	return nil
}

// checkIntegrated es la comprobación de integración de D-25 (paso 5): la base evaluada debe
// contener la carpeta `changes/<cambio>/` del planning home, que se lee del árbol del commit de la
// base con `git ls-tree`, sin cambiar de rama ni usar la red. Si no la contiene, es
// `change_not_integrated`, sin crear nada, nombrando la base y sugiriendo `--fetch`; si en su
// lugar contiene `changes/archive/<fecha>-<cambio>/`, el cambio ya está archivado y el mensaje
// nombra esa entrada y dice que no hay nada que archivar (G-13). fetched dice si esta ejecución
// ya trajo el remoto.
func (e *Engine) checkIntegrated(ctx context.Context, command, cwd string, repo branch.RepoPlan, facts changeFacts, base baseReference, fetched bool) *branch.Refusal {
	planningHome := facts.location.PlanningHome
	relative, inside := relativeWithin(repo.GitRoot, planningHome)
	if !inside {
		return branch.PlanningHomeOutsideRepository(facts.input.Master, planningHome, repo)
	}
	change := facts.input.Change
	changesPath := path.Join(relative, "changes")
	changePath := path.Join(changesPath, change)

	// Se lee el commit ya resuelto, no el nombre de la rama: es la misma referencia aunque el
	// nombre sea ambiguo.
	_, integrated, err := e.git.TreeDirs(ctx, repo.GitRoot, base.sha, changePath)
	if err != nil {
		return gitRefusal(command, cwd, fmt.Sprintf("leer %s en %s", changePath, base.ref), err)
	}
	if integrated {
		return nil
	}
	archivePath := path.Join(changesPath, "archive")
	archived, _, err := e.git.TreeDirs(ctx, repo.GitRoot, base.sha, archivePath)
	if err != nil {
		return gitRefusal(command, cwd, fmt.Sprintf("leer %s en %s", archivePath, base.ref), err)
	}
	// Con varias entradas del mismo cambio, la última (la de fecha más reciente) es la que se nombra.
	entry := ""
	matches := archivedEntry(change)
	for i := len(archived) - 1; i >= 0 && entry == ""; i-- {
		if matches(archived[i]) {
			entry = archivePath + "/" + archived[i] + "/"
		}
	}
	return branch.ChangeNotIntegrated(change, facts.input.Master, facts.input.Exceptions.Prefix, repo, base.ref, changePath+"/", entry, fetched)
}

// archivedEntry reconoce la carpeta con la que `archive` mueve un cambio a `changes/archive/`:
// `<fecha>-<cambio>`, con la fecha `AAAA-MM-DD`.
func archivedEntry(change string) func(name string) bool {
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-` + regexp.QuoteMeta(change) + `$`)
	return pattern.MatchString
}

// newArchiveRecord compone el registro de un `ensure --archive` con el estado final de git: el
// registro previo del cambio, que no se toca salvo por la rama de archivado (tipo `archive`) y el
// prefijo de archivado, o, si el cambio no tenía registro, uno nuevo con el modo resuelto y solo
// la rama de archivado del repositorio del planning home. `ensure` ordinario lo completa después
// con las ramas del cambio y conserva la de archivado. Sin marcas de tiempo: el mismo estado da los
// mismos bytes.
func newArchiveRecord(facts changeFacts, plan branch.ChangePlan, mode workMode) branch.Record {
	archived := recordRepository(facts, plan.Repositories[0], mode)
	prefix := facts.input.Exceptions.Prefix
	previous := facts.record
	if previous == nil {
		return branch.Record{
			Schema:        branch.RegistrySchema,
			Lane:          plan.Lane,
			Change:        plan.Change,
			Mode:          mode.mode,
			ModeSource:    mode.source,
			ArchivePrefix: prefix,
			PlanningRepo:  facts.input.PlanningRepo.GitRoot,
			Repositories:  []branch.RecordRepo{archived},
		}
	}

	record := *previous
	if prefix != "" {
		record.ArchivePrefix = prefix
	}
	record.Repositories = make([]branch.RecordRepo, len(previous.Repositories))
	merged := false
	for i, registered := range previous.Repositories {
		registered.Branches = slices.Clone(registered.Branches)
		if pathidentity.SameDirectory(registered.CommonDir, archived.CommonDir) {
			merged = true
			registered.Branches = slices.DeleteFunc(registered.Branches, func(recorded branch.RecordBranch) bool {
				return recorded.Kind == branch.KindArchive
			})
			registered.Branches = append(registered.Branches, archived.Branches...)
		}
		record.Repositories[i] = registered
	}
	if !merged {
		record.Repositories = append(record.Repositories, archived)
	}
	return record
}
