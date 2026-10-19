package engine

import (
	"errors"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// El registro local en `ensure` (design D-05 fases 5 y 6, D-07, REQ-25.14): `ensure` lo
// compone siempre desde el plan y el estado final de git, y `registry.Store.Save` solo
// escribe si cambian sus bytes. Por eso no hay un camino de «reconstrucción» aparte: un
// registro que falta, o que se quedó corto porque el sellado añadió ramas, se compone con
// las mismas reglas que el primero, a partir de las ramas planificadas que existen y de los
// worktrees que git conoce. Borrar el fichero no daña el repositorio y la siguiente
// ejecución lo recompone idéntico.

// newRecord compone el registro del cambio con el estado final de git: las ramas del plan
// (tras aplicarlo existen todas y cada una está desplegada) con el worktree en el que git
// las tiene, que puede ser otro que el planificado si ya estaba registrado. El modo y su
// origen son los que resolvió `resolveMode` (prioridad: registro > --mode > hub > defecto,
// D-08): con un registro previo, los de este. Las excepciones y el prefijo de archivado de un
// registro previo se conservan. Sin marcas de tiempo: el mismo estado da los mismos bytes.
func newRecord(facts changeFacts, plan branch.ChangePlan, mode workMode) branch.Record {
	record := branch.Record{
		Schema:       branch.RegistrySchema,
		Lane:         plan.Lane,
		Change:       plan.Change,
		Mode:         mode.mode,
		ModeSource:   mode.source,
		Exceptions:   facts.input.Exceptions,
		PlanningRepo: facts.input.PlanningRepo.GitRoot,
		Repositories: make([]branch.RecordRepo, 0, len(plan.Repositories)),
	}
	if previous := facts.record; previous != nil {
		record.ArchivePrefix = previous.ArchivePrefix
	}
	for _, repo := range plan.Repositories {
		// Las vistas de candidato de RDD son worktrees desacoplados: no cuentan (D-07).
		deployed := map[string]string{}
		for _, entry := range facts.worktrees[repo.GitRoot] {
			if entry.Branch != "" && !entry.Detached {
				deployed[entry.Branch] = entry.Path
			}
		}
		// En `in_place` la rama no tiene worktree: está desplegada en el propio repositorio,
		// que es el primer worktree de git (el checkout principal), o no lo está. Una rama de
		// rol que otra herramienta tenga en un worktree ajeno no es de este registro.
		var main string
		if listed := facts.worktrees[repo.GitRoot]; len(listed) > 0 {
			main = listed[0].Path
		}
		registered := branch.RecordRepo{
			Name:      repo.Name,
			GitRoot:   repo.GitRoot,
			CommonDir: repo.CommonDir,
			Role:      repo.Role,
			Branches:  make([]branch.RecordBranch, 0, len(repo.Branches)),
		}
		for _, planned := range repo.Branches {
			recorded := branch.RecordBranch{Name: planned.Name, Kind: planned.Kind, RoleID: planned.RoleID}
			if mode.mode == branch.ModeInPlace {
				recorded.InPlace = deployed[planned.Name] != "" && pathidentity.SameDirectory(deployed[planned.Name], main)
			} else {
				recorded.Worktree = deployed[planned.Name]
			}
			registered.Branches = append(registered.Branches, recorded)
		}
		record.Repositories = append(record.Repositories, registered)
	}
	return record
}

// lockRefusal traduce el fallo de bloquear el cambio: un bloqueo ocupado es `plan_conflict`
// (design D-05); cualquier otro fallo es del registro local.
func lockRefusal(err error, operation, change, cwd string) *branch.Refusal {
	if errors.Is(err, registry.ErrBusy) {
		return branch.ChangeLocked(operation, change, cwd)
	}
	return writeRefusal(err, operation, change, cwd)
}

// writeRefusal traduce un fallo del registro local al bloquear o al guardar: un
// *registry.WriteError nombra el fichero o la carpeta que falló.
func writeRefusal(err error, operation, change, cwd string) *branch.Refusal {
	var failure *registry.WriteError
	path := ""
	if errors.As(err, &failure) {
		path = failure.Path
	}
	if path == "" {
		path = cwd
	}
	return branch.RegistryWriteFailed(operation, change, cwd, path, err)
}
