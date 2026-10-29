package cli

import (
	"context"
	"fmt"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// Sonda de linajes RDD abiertos de `axiom branch done` (INC-25, S6, REQ-25.17, A-4): antes de
// eliminar un worktree, `done` avisa de los linajes de revisión que siguen abiertos en él. El
// adaptador vive aquí y no en internal/branch/** porque el almacén de `internal/reviewtransaction`
// exige git (resuelve la raíz del repositorio y su git common dir con sus propias invocaciones de
// git, que no pasan por gitcommon.Run), de modo que escapa al observador y a la prueba de
// arquitectura del motor (REQ-25.1). Es una exención local y documentada, solo de lectura: no crea
// directorios, no toma bloqueos y no hace ninguna operación remota. El motor solo conoce el puerto
// branch.LineageProbe.

// newBranchLineageProbe monta la sonda real de `done`. Es una variable para que las pruebas del
// verbo la sustituyan por un doble.
var newBranchLineageProbe = newLineageProbe

// reviewLineageProbe implementa branch.LineageProbe sobre el inventario de autoridad de
// internal/reviewtransaction. Las dos funciones son las reales salvo en las pruebas.
type reviewLineageProbe struct {
	inventory func(ctx context.Context, repo string) (reviewtransaction.AuthorityStatusReport, error)
	foreign   func(ctx context.Context, repo, lineageID string) (bool, error)
}

var _ branch.LineageProbe = reviewLineageProbe{}

func newLineageProbe() branch.LineageProbe {
	return reviewLineageProbe{
		inventory: reviewtransaction.InventoryAuthority,
		foreign:   reviewtransaction.ExactReviewLineageForeignWorktree,
	}
}

// OpenLineages devuelve, ordenados, los identificadores de los linajes de revisión compactos que
// siguen abiertos (estado `active`: ni aprobados, ni escalados, ni invalidados) y que se congelaron
// desde el worktree worktreeRoot, que debe ser la raíz de ese worktree. El almacén es compartido por
// todos los worktrees del repositorio, de modo que descarta los que congeló otro worktree. Un
// linaje anterior a la vinculación con el worktree no se puede comparar y cuenta como de este, como
// hace el propio almacén. Los linajes del formato anterior (legacy-v1) no se consultan. Un fallo al
// leer el inventario o al comprobar a quién pertenece un linaje se devuelve como error.
func (p reviewLineageProbe) OpenLineages(ctx context.Context, worktreeRoot string) ([]string, error) {
	report, err := p.inventory(ctx, worktreeRoot)
	if err != nil {
		return nil, fmt.Errorf("leer el inventario de revisiones de %s: %w", worktreeRoot, err)
	}
	var open []string
	for _, entry := range report.Entries {
		if entry.Version != reviewtransaction.AuthorityVersionCompact || entry.Status != reviewtransaction.AuthorityStatusActive || entry.LineageID == "" {
			continue
		}
		foreign, err := p.foreign(ctx, worktreeRoot, entry.LineageID)
		if err != nil {
			return nil, fmt.Errorf("comprobar a qué worktree pertenece el linaje %s: %w", entry.LineageID, err)
		}
		if !foreign {
			open = append(open, entry.LineageID)
		}
	}
	slices.Sort(open)
	return open, nil
}
