package knowledge

import (
	"errors"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Operaciones de knowledge que escriben <planning home>/specs o
// <planning home>/INDEX.md, tal como las nombra el rechazo de la guarda
// (REQ-24.23).
const (
	opSweep       = "la siembra de especificaciones y del catálogo de `axiom knowledge sweep`"
	opQueryEnrich = "el enriquecimiento de la especificación viva de `axiom knowledge query --enrich`"
	opRecordUnit  = "`axiom knowledge crawl --record-unit`"
	opFinalize    = "`axiom knowledge crawl --finalize`"
)

// defaultSpecsRoot resuelve la raíz de especificaciones por defecto de knowledge:
// el planning home que devuelve workspace.Locate desde la raíz de código que se
// analiza (REQ-24.10, D-07). La raíz de código y la de especificaciones son
// distintas: en el workspace canónico, analizar un repositorio de código lee las
// especificaciones del repositorio de especificaciones.
//
// Devuelve además los avisos no bloqueantes de la resolución (el de la ubicación
// antigua divergente, Q14), que los llamadores vuelcan en las notes de su
// resultado. Un fallo de Locate se propaga: nombra la causa y la continuación.
func defaultSpecsRoot(absWorkspace string) (specsRoot string, notes []string, err error) {
	loc, err := workspace.Locate(absWorkspace)
	if err != nil {
		return "", nil, err
	}
	if warning, ok := loc.LegacySpecsDivergence(); ok {
		notes = append(notes, warning)
	}
	return loc.PlanningHome, notes, nil
}

// specsTarget es lo que RunSweep y RunQuery necesitan de la resolución: la raíz
// de especificaciones, los avisos y si pueden escribir en ella.
type specsTarget struct {
	root  string   // raíz de especificaciones efectiva
	notes []string // avisos no bloqueantes (ubicación antigua divergente)
	// blocked es la causa por la que no se puede escribir en la raíz: un
	// *workspace.BaselineRetiredError con un `baseline:` residual, o
	// el *workspace.LocateError que impide saber qué baseline es la del proyecto
	// (REQ-24.22). Es nil si se puede escribir.
	blocked error
}

// resolveSpecsTarget resuelve la raíz de especificaciones y evalúa la guarda de
// escritura de op (D-06). Con una raíz explícita se usa tal cual, pero la guarda
// se evalúa igualmente desde la raíz de código: no se puede eludir pasando una
// ruta. Sin raíz explícita, un rechazo de Locate se propaga porque no hay dónde
// leer; con raíz explícita solo impide escribir.
func resolveSpecsTarget(absWorkspace, explicitRoot, op string) (specsTarget, error) {
	loc, err := workspace.Locate(absWorkspace)
	if err != nil {
		if explicitRoot == "" {
			return specsTarget{}, err
		}
		return specsTarget{root: explicitRoot, blocked: err}, nil
	}
	target := specsTarget{root: explicitRoot, blocked: loc.GuardSpecsWrite(op)}
	if target.root == "" {
		target.root = loc.PlanningHome
		if warning, ok := loc.LegacySpecsDivergence(); ok {
			target.notes = append(target.notes, warning)
		}
	}
	return target, nil
}

// skipNote explica que se omite una escritura y que el resto del resultado se
// completa: nombra el `baseline:` residual y su migración, o la causa por la que
// no se puede resolver el planning home.
func skipNote(blocked error, skipped string) string {
	var retired *workspace.BaselineRetiredError
	if errors.As(blocked, &retired) {
		return fmt.Sprintf("%v. Se omite %s y el resto del resultado se completa.", blocked, skipped)
	}
	return fmt.Sprintf("No se puede escribir en la especificación viva porque la resolución del planning home se rechaza: %v. Se omite %s y el resto del resultado se completa.", blocked, skipped)
}

// guardCrawlWrite resuelve el planning home de la raíz de código del job en el
// momento de escribir y rechaza la escritura de op con un `baseline:` residual
// (REQ-30.21). Un rechazo de Locate también impide escribir.
func guardCrawlWrite(job *CrawlJob, op string) error {
	loc, err := workspace.Locate(job.WorkspaceRoot)
	if err != nil {
		return err
	}
	return loc.GuardSpecsWrite(op)
}
