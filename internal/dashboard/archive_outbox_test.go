package dashboard

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// RF-ARC-03 (MODIFIED): la cola de integraciones `.integrations-outbox.yaml` de la raíz de una
// carpeta archivada es estado local, no contenido del incremento: el listado de incrementos es el
// mismo con y sin ella.
func TestArchiveOutboxIsNotIncrementContent(t *testing.T) {
	t.Parallel()
	listingOf := func(withOutbox bool) []byte {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")
		if withOutbox {
			workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-viejo", ".integrations-outbox.yaml"), "schema: axiom.integrations-outbox/v1\nchange: viejo\n")
		}
		listing := getListing(t, NewService(ws.Master))
		raw, err := json.Marshal(listing.list)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}

	without, with := listingOf(false), listingOf(true)
	if string(with) != string(without) {
		t.Fatalf("el listado de incrementos no cambia por la cola archivada:\n%s\n---\n%s", with, without)
	}
	if strings.Contains(string(with), "integrations-outbox") {
		t.Fatalf("ningún incremento menciona la cola: %s", with)
	}
}
