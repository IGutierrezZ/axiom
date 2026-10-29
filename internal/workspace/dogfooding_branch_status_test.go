package workspace_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// REQ-25.24 (escenario «Un cambio preexistente ya fusionado figura como pendiente de
// archivar»): `axiom branch status --json` sobre este repositorio lista como
// pendientes de archivar los cambios que ya están en el árbol de la rama de
// integración y siguen sin archivar, sin bloquear nada (salida 0). Es una lectura:
// no escribe nada ni toca la red.
//
// El hecho depende del estado de las referencias del clon, así que la prueba lo
// deriva de ellas en lugar de dar por sentado un nombre: `inc-23-knowledge-profile`
// debe figurar si y solo si su carpeta está en el árbol de la integración, y se salta
// si el clon no tiene ninguna referencia de integración (una copia superficial).
func TestDocsDogfoodingListsPreexistingChangesAsPendingArchive(t *testing.T) {
	if testing.Short() {
		t.Skip("las pruebas con git real se saltan con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	tree := ""
	for _, ref := range []string{"origin/main", "main"} {
		out, err := exec.Command("git", "-C", root, "ls-tree", "--name-only", ref, "openspec/changes/").Output()
		if err == nil {
			tree = string(out)
			break
		}
	}
	if tree == "" {
		t.Skip("el clon no tiene una referencia de integración con openspec/changes/")
	}
	const preexisting = "inc-23-knowledge-profile"
	inIntegration := strings.Contains(tree, "openspec/changes/"+preexisting)

	// Un HOME vacío: la lectura no depende de las preferencias del hub de quien la ejecuta.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	var stdout, stderr bytes.Buffer
	if code := cli.RunBranch([]string{"status", "--json", "--cwd", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("axiom branch status --json salió con %d (debe ser 0, solo informa):\n%s\n%s", code, stdout.String(), stderr.String())
	}
	var result struct {
		Schema         string `json:"schema"`
		PendingArchive []struct {
			Change       string `json:"change"`
			Continuation string `json:"continuation"`
		} `json:"pendingArchive"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, stdout.String())
	}
	if result.Schema != "axiom.branch-result/v1" {
		t.Fatalf("schema = %q, want axiom.branch-result/v1", result.Schema)
	}

	listed := false
	for _, pending := range result.PendingArchive {
		if pending.Change != preexisting {
			continue
		}
		listed = true
		if !strings.Contains(pending.Continuation, "axiom branch ensure --change "+preexisting+" --archive") {
			t.Errorf("la continuación de %s no es la orden de archivado: %q", preexisting, pending.Continuation)
		}
	}
	if inIntegration && !listed {
		t.Errorf("%s está en la rama de integración sin archivar y no figura como pendiente de archivar:\n%s", preexisting, stdout.String())
	}
	if !inIntegration && listed {
		t.Errorf("%s figura como pendiente de archivar pero no está en la rama de integración", preexisting)
	}
}
