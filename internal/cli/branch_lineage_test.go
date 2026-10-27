package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// Sonda de linajes RDD abiertos de `done` (INC-25, S6-c2, REQ-25.17, A-4): el adaptador real sobre el
// almacén de internal/reviewtransaction. Las pruebas con dobles fijan qué linajes cuenta; una con el
// almacén real fija que lee un repositorio sin revisiones sin error.

func inventoryOf(entries ...reviewtransaction.AuthorityInventoryEntry) func(context.Context, string) (reviewtransaction.AuthorityStatusReport, error) {
	return func(context.Context, string) (reviewtransaction.AuthorityStatusReport, error) {
		return reviewtransaction.AuthorityStatusReport{Entries: entries}, nil
	}
}

func entryOf(version reviewtransaction.AuthorityVersion, lineage string, status reviewtransaction.AuthorityStatus) reviewtransaction.AuthorityInventoryEntry {
	return reviewtransaction.AuthorityInventoryEntry{Version: version, LineageID: lineage, Status: status}
}

// Solo cuentan los linajes compactos abiertos (`active`) que congeló este worktree, ordenados: los
// aprobados, escalados, invalidados e inválidos, los del formato anterior y los de otro worktree del
// mismo repositorio no avisan.
func TestBranchLineageProbeReportsOnlyTheOpenCompactLineagesOfThisWorktree(t *testing.T) {
	compact := reviewtransaction.AuthorityVersionCompact
	var asked []string
	probe := reviewLineageProbe{
		inventory: inventoryOf(
			entryOf(compact, "linaje-b", reviewtransaction.AuthorityStatusActive),
			entryOf(compact, "linaje-a", reviewtransaction.AuthorityStatusActive),
			entryOf(compact, "linaje-ajeno", reviewtransaction.AuthorityStatusActive),
			entryOf(compact, "linaje-aprobado", reviewtransaction.AuthorityStatusApproved),
			entryOf(compact, "linaje-escalado", reviewtransaction.AuthorityStatusEscalated),
			entryOf(compact, "linaje-invalidado", reviewtransaction.AuthorityStatusInvalidated),
			entryOf(compact, "linaje-invalido", reviewtransaction.AuthorityStatusInvalid),
			entryOf(reviewtransaction.AuthorityVersionLegacy, "linaje-anterior", reviewtransaction.AuthorityStatusActive),
			entryOf(compact, "", reviewtransaction.AuthorityStatusActive),
		),
		foreign: func(_ context.Context, repo, lineage string) (bool, error) {
			asked = append(asked, repo+"|"+lineage)
			return lineage == "linaje-ajeno", nil
		},
	}

	got, err := probe.OpenLineages(context.Background(), "/worktree")

	if err != nil || !slices.Equal(got, []string{"linaje-a", "linaje-b"}) {
		t.Errorf("OpenLineages = %v, %v; se esperaban linaje-a y linaje-b", got, err)
	}
	if want := []string{"/worktree|linaje-b", "/worktree|linaje-a", "/worktree|linaje-ajeno"}; !slices.Equal(asked, want) {
		t.Errorf("la pertenencia se comprueba solo para los compactos abiertos y con el worktree: %v, se esperaba %v", asked, want)
	}

	if none, err := (reviewLineageProbe{inventory: inventoryOf()}).OpenLineages(context.Background(), "/worktree"); err != nil || len(none) != 0 {
		t.Errorf("sin entradas no hay linajes: %v, %v", none, err)
	}
}

// Un fallo del inventario o de la comprobación de pertenencia es un error que nombra la causa: el
// motor lo convierte en un aviso y no impide eliminar.
func TestBranchLineageProbeFailsWhenTheInventoryOrTheOwnershipCheckFails(t *testing.T) {
	failing := reviewLineageProbe{inventory: func(context.Context, string) (reviewtransaction.AuthorityStatusReport, error) {
		return reviewtransaction.AuthorityStatusReport{}, errors.New("almacén ilegible")
	}}
	if _, err := failing.OpenLineages(context.Background(), "/worktree"); err == nil || !strings.Contains(err.Error(), "almacén ilegible") {
		t.Errorf("el error del inventario debe llegar: %v", err)
	}

	unowned := reviewLineageProbe{
		inventory: inventoryOf(entryOf(reviewtransaction.AuthorityVersionCompact, "linaje-a", reviewtransaction.AuthorityStatusActive)),
		foreign:   func(context.Context, string, string) (bool, error) { return false, errors.New("identidad ilegible") },
	}
	if _, err := unowned.OpenLineages(context.Background(), "/worktree"); err == nil || !strings.Contains(err.Error(), "linaje-a") || !strings.Contains(err.Error(), "identidad ilegible") {
		t.Errorf("el error de pertenencia debe nombrar el linaje y la causa: %v", err)
	}
}

// A-4: el almacén real de internal/reviewtransaction se lee, sin error, en un repositorio sin
// revisiones, con el adaptador que monta `done`. La API del almacén lanza git por su cuenta (no por
// gitcommon.Run), de modo que el observador del motor no ve esas invocaciones: es la exención local
// de A-4 que documenta spike-guard-runtimes.md. Lo que sí fija esta prueba es que la sonda no pasa por
// el único punto de ejecución de git ni hace ninguna operación remota.
func TestBranchLineageProbeReadsTheRealStoreOfARepositoryWithoutReviews(t *testing.T) {
	repo := newBranchRepository(t)
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) { seen = append(seen, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })
	var probe branch.LineageProbe = newLineageProbe()

	got, err := probe.OpenLineages(context.Background(), repo)

	if err != nil || len(got) != 0 {
		t.Fatalf("OpenLineages = %v, %v; un repositorio sin revisiones no tiene linajes abiertos", got, err)
	}
	for _, invocation := range seen {
		if branchRemoteOperationName(invocation[1]) {
			t.Errorf("la sonda no hace operaciones remotas: %v", invocation)
		}
	}
}

func branchRemoteOperationName(subcommand string) bool {
	switch subcommand {
	case "fetch", "pull", "push", "ls-remote":
		return true
	}
	return false
}
