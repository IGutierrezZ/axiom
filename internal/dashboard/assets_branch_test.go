package dashboard

import (
	"regexp"
	"strings"
	"testing"
)

// La SPA no tiene infraestructura de pruebas de navegador: sus piezas de INC-25 S10-c4 se fijan por
// marcadores, como el resto de los assets embebidos, y su contrato con el servidor lo cubren las pruebas
// HTTP de S10-c1 a S10-c3 (`decision_required` tipado, `branchPolicy`, `pendingArchive` y `X-Axiom-Notes`).

func readAsset(t *testing.T, name string) string {
	t.Helper()
	raw, err := AssetsFS.ReadFile("assets/" + name)
	if err != nil {
		t.Fatalf("leer assets/%s: %v", name, err)
	}
	return string(raw)
}

// REQ-25.34: la tarjeta de un cambio muestra la insignia de pendiente de archivar y la rama de su
// worktree, y los avisos de `X-Axiom-Notes` llegan a un contenedor visible del listado.
func TestSPAShowsPendingArchiveBranchAndNotes(t *testing.T) {
	html, js := readAsset(t, "index.html"), readAsset(t, "app.js")

	if !strings.Contains(html, `id="increments-notes"`) {
		t.Error("index.html debe tener el contenedor `increments-notes` de los avisos del listado")
	}
	for _, marker := range []string{"inc.pendingArchive", "Pendiente de archivar", "inc.worktreePath", "inc.branch", "X-Axiom-Notes", "increments-notes"} {
		if !strings.Contains(js, marker) {
			t.Errorf("app.js debe contener %q", marker)
		}
	}
	// El aviso es un array JSON: se decodifica y no se interpreta como HTML.
	notes := regexp.MustCompile(`(?s)function renderIncrementNotes.*?\n  }\n`).FindString(js)
	if !strings.Contains(notes, "JSON.parse") || !strings.Contains(notes, "textContent") || strings.Contains(notes, "innerHTML = `") {
		t.Errorf("renderIncrementNotes debe decodificar el array JSON y pintarlo con textContent:\n%s", notes)
	}
}

// REQ-25.32: ante un 409 `decision_required` la SPA presenta la decisión y repite la creación con `mode`.
func TestSPARetriesTheCreationWithModeAfterADecisionRequired(t *testing.T) {
	html, js := readAsset(t, "index.html"), readAsset(t, "app.js")

	for _, id := range []string{"modal-increment-decision", "increment-decision-question", "increment-decision-choices", "btn-cancel-increment-decision"} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Errorf("index.html debe contener el diálogo de decisión (%s)", id)
		}
	}
	create := regexp.MustCompile(`(?s)async function createIncrement.*?\n  }\n`).FindString(js)
	if create == "" {
		t.Fatal("app.js debe tener createIncrement")
	}
	for _, marker := range []string{"res.status === 409", "decision_required", "askIncrementDecision", "mode: answer", "data.continuation"} {
		if !strings.Contains(create, marker) {
			t.Errorf("createIncrement debe contener %q:\n%s", marker, create)
		}
	}
	// La decisión se presenta con las respuestas del servidor, sin interpretarlas como HTML.
	ask := regexp.MustCompile(`(?s)function askIncrementDecision.*?\n  }\n`).FindString(js)
	for _, marker := range []string{"decision.question", "decision.choices", "choice.answer", "choice.label", "choice.effect", "textContent"} {
		if !strings.Contains(ask, marker) {
			t.Errorf("askIncrementDecision debe contener %q:\n%s", marker, ask)
		}
	}
	if strings.Contains(ask, "innerHTML = `") {
		t.Errorf("askIncrementDecision no debe pintar la decisión como HTML:\n%s", ask)
	}
	// El alta del modal usa la creación con reintento.
	if submit := regexp.MustCompile(`(?s)btnSubmitNewIncrement\.addEventListener.*?\n    }\);\n`).FindString(js); !strings.Contains(submit, "createIncrement(") {
		t.Errorf("el botón de crear incremento debe usar createIncrement:\n%s", submit)
	}
}

// REQ-1.1 y REQ-25.33: la política de ramas y los ajustes por repositorio se muestran en solo lectura.
func TestSPAShowsTheBranchPolicyReadOnly(t *testing.T) {
	html, js := readAsset(t, "index.html"), readAsset(t, "app.js")

	for _, id := range []string{"top-ws-integration-branch", "top-ws-branch-prefix", "topology-repo-policy-tbody"} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Errorf("index.html debe mostrar la política de ramas (%s)", id)
		}
	}
	for _, marker := range []string{"workspaceData.branchPolicy", "integrationBranch", "branchPrefix", "workspaceData.repositories", "top-ws-integration-branch", "topology-repo-policy-tbody"} {
		if !strings.Contains(js, marker) {
			t.Errorf("app.js debe contener %q", marker)
		}
	}
	// Solo lectura: ni controles de edición de `git:` ni una escritura de la política.
	if strings.Contains(html, "integration_branch") || strings.Contains(html, "branch_prefix") {
		t.Error("index.html no debe ofrecer controles de edición de `git:` (REQ-25.33)")
	}
	policyInputs := regexp.MustCompile(`(?s)<(input|select|textarea)[^>]*id="[^"]*(integration|branch-prefix)[^"]*"`).FindString(html)
	if policyInputs != "" {
		t.Errorf("la política de ramas es de solo lectura, no debe tener controles: %s", policyInputs)
	}
}
