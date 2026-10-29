package engine

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// El aviso de la guarda D (REQ-25.21 escenarios 3 a 8, design D-18 paso 4): con el evento `post` y
// un fichero que no es un documento de Axiom, avisa sin bloquear —la escritura ya ocurrió— cuando
// el checkout que lo contiene no está en la rama de ninguno de los cambios activos registrados por
// `ensure` en ese repositorio. Como el bloqueo, decide solo con lecturas de ficheros: no lanza un
// proceso, no consulta `VerdictFor` (el consultante `guard` evalúa un repositorio de código como
// `not_applicable`; el aviso no pregunta por el papel del repositorio sino por el registro local y
// por `HEAD`) y falla en abierto.

// guardWarn compone el aviso de las rutas de un evento `post`. Cada checkout se avisa una sola vez,
// aunque el parche escriba varios ficheros en él; sin nada que avisar devuelve `none`.
func (e *Engine) guardWarn(ctx context.Context, payload GuardPayload) GuardDecision {
	var roots, messages []string
	for _, path := range payload.Paths {
		if ctx.Err() != nil {
			return guardNone
		}
		root, message, ok := e.warnFile(path)
		if !ok || containsDirectory(roots, root) {
			continue
		}
		roots, messages = append(roots, root), append(messages, message)
	}
	if len(messages) == 0 {
		return guardNone
	}
	return GuardDecision{Action: GuardActionWarn, Message: strings.Join(messages, "\n")}
}

// warnFile evalúa un fichero escrito y devuelve la raíz git del checkout que lo contiene y el
// aviso. ok es falso cuando no hay nada que avisar: el fichero es un documento de Axiom (su
// responsabilidad es la del bloqueo, evento `pre`), queda fuera de un proyecto Axiom o es del
// repositorio de specs de un `multirepo` (exento); el repositorio no tiene ningún cambio activo
// registrado; el checkout está en la rama de uno de ellos; o cualquier lectura falla.
func (e *Engine) warnFile(path string) (root, message string, ok bool) {
	start := existingDirectory(path)
	root, inGit := gitcommon.GitRootFromFiles(start)
	if !inGit {
		return "", "", false
	}
	loc, err := workspace.Locate(start)
	if err != nil || loc.Origin == workspace.OriginStart || branch.ConfigRefusal(loc) != nil {
		return "", "", false
	}
	if document, isDocument := classifyAxiomDocument(path, runtime.GOOS == "windows"); isDocument && containsDocument(loc, document) {
		return "", "", false
	}

	planningRoot, inGit := gitcommon.GitRootFromFiles(loc.WorkspaceRoot)
	if !inGit {
		return "", "", false
	}
	isPlanning := pathidentity.SameDirectory(root, planningRoot)
	if isPlanning && topologyOf(loc) == workspace.TopologyMultirepo {
		return "", "", false
	}
	planningCommon, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		return "", "", false
	}
	repoCommon := planningCommon
	if !isPlanning {
		if repoCommon, err = gitcommon.CommonDirFromFiles(root); err != nil {
			return "", "", false
		}
	}
	active, ok := e.activeChangesOf(planningCommon, repoCommon)
	if !ok || len(active) == 0 {
		return "", "", false
	}

	current, detached, err := gitcommon.ReadHead(root)
	if err != nil {
		return "", "", false
	}
	for _, change := range active {
		for _, registered := range change.branches {
			if !detached && registered.Name == current {
				return "", "", false
			}
		}
	}
	return root, warnMessage(path, current, detached, active), true
}

// activeChange es un cambio activo registrado en un repositorio: su carril, su nombre, el modo de
// trabajo y las ramas que el registro guarda para ese repositorio.
type activeChange struct {
	lane     branch.Lane
	name     string
	mode     branch.Mode
	branches []branch.RecordBranch
}

// activeChangesOf lee los registros activos de los dos carriles (SDD y ODD) del git common dir del
// planning home y devuelve los cambios que tienen una entrada para el repositorio cuyo common dir
// es repoCommon, en orden (primero SDD, por nombre de cambio, y luego ODD). Un registro cuyo
// repositorio no tiene ramas no es un cambio al que avisar. ok es falso si no se puede leer el
// registro: la guarda no avisa con lo que no puede comprobar.
func (e *Engine) activeChangesOf(planningCommon, repoCommon string) (active []activeChange, ok bool) {
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		records, err := e.registry.List(planningCommon, lane)
		if err != nil {
			return nil, false
		}
		for _, record := range records {
			for _, repository := range record.Repositories {
				if len(repository.Branches) > 0 && pathidentity.SameDirectory(repository.CommonDir, repoCommon) {
					active = append(active, activeChange{lane: record.Lane, name: record.Change, mode: record.Mode, branches: repository.Branches})
					break
				}
			}
		}
	}
	return active, true
}

// warnMessage es el aviso que ve el agente: dice dónde acaba de escribir y nombra, por cada cambio
// activo, el worktree correcto o, en `in_place`, la rama esperada.
func warnMessage(path, current string, detached bool, active []activeChange) string {
	where := fmt.Sprintf("la rama «%s»", current)
	if detached {
		where = "un HEAD desacoplado"
	}
	places := make([]string, 0, len(active))
	for _, change := range active {
		places = append(places, change.describe())
	}
	return fmt.Sprintf("axiom branch guard: aviso: has escrito %s en %s, que no es la rama de ningún cambio activo de este repositorio. El código se escribe donde corresponde: %s.",
		path, where, strings.Join(places, "; "))
}

// describe nombra el sitio del cambio: el worktree de cada una de sus ramas o, si la rama se
// despliega en el propio repositorio (`in_place`), la rama esperada. Las ramas de archivado solo
// cuentan si el cambio no tiene otras: no son donde se escribe el código.
func (c activeChange) describe() string {
	chosen := make([]branch.RecordBranch, 0, len(c.branches))
	for _, registered := range c.branches {
		if registered.Kind != branch.KindArchive {
			chosen = append(chosen, registered)
		}
	}
	if len(chosen) == 0 {
		chosen = c.branches
	}
	label := fmt.Sprintf("cambio «%s»", c.name)
	if c.lane == branch.LaneODD {
		label = fmt.Sprintf("cambio ODD «%s»", c.name)
	}
	sites := make([]string, 0, len(chosen))
	for _, registered := range chosen {
		if registered.Worktree != "" {
			sites = append(sites, fmt.Sprintf("worktree %s (rama «%s»)", pathquote.Quote(registered.Worktree), registered.Name))
			continue
		}
		sites = append(sites, fmt.Sprintf("rama esperada «%s» (modo in_place)", registered.Name))
	}
	return fmt.Sprintf("%s: %s", label, strings.Join(sites, ", "))
}

// containsDirectory dice si dirs tiene dir, por identidad de directorio y no por texto.
func containsDirectory(dirs []string, dir string) bool {
	for _, known := range dirs {
		if pathidentity.SameDirectory(known, dir) {
			return true
		}
	}
	return false
}
