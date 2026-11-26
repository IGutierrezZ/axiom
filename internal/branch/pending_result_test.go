package branch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// La rama pendiente en el resultado (C-07-f, D-15): JSON aditivo y línea de texto en castellano.

func TestNewPendingBranchResultAwaitingRoleCarriesTheEnsureOrder(t *testing.T) {
	t.Parallel()
	got := NewPendingBranchResult(PendingBranch{Kind: KindChange, RoleID: "Backend", Template: "feature/TASK-{task_ref}"}, "x", "/maestra")

	if got.Reason != PendingReasonAwaitingRole || got.ReasonLabel != "pendiente de la entrada del rol" || got.Name != "" || got.Code != "" {
		t.Fatalf("pendiente = %+v", got)
	}
	if want := "axiom branch ensure --change x --role backend --cwd"; !strings.Contains(got.Continuation, want) {
		t.Errorf("continuación = %q, se esperaba %q", got.Continuation, want)
	}
	if line := got.textLine(); !strings.Contains(line, "sin nombre todavía") || !strings.Contains(line, "del rol Backend") {
		t.Errorf("línea = %q", line)
	}
}

func TestNewPendingBranchResultFromARefusalReportsItsCode(t *testing.T) {
	t.Parallel()
	refusal := &Refusal{Code: CodeIntegrationsLinksInvalid, Message: "ilegible", Continuation: "revisa integrations.yaml"}

	got := NewPendingBranchResult(PendingBranch{Kind: KindChange, Template: "t", Candidate: "feature/TASK-7", Refusal: refusal}, "x", "/m")

	if got.Reason != string(CodeIntegrationsLinksInvalid) || got.Code != CodeIntegrationsLinksInvalid || got.Message != "ilegible" || got.Continuation != "revisa integrations.yaml" || got.Name != "feature/TASK-7" {
		t.Fatalf("pendiente = %+v", got)
	}
	if line := got.textLine(); !strings.Contains(line, "nombre feature/TASK-7") || !strings.Contains(line, "[integrations_links_invalid]") {
		t.Errorf("línea = %q", line)
	}
}

// Sin ramas pendientes el JSON y el texto de un repositorio no cambian.
func TestRepositoryResultWithoutPendingBranchesIsUnchanged(t *testing.T) {
	t.Parallel()
	repository := RepositoryResult{Name: "r", GitRoot: "/r", CommonDir: "/r/.git", Role: RepoCode, RoleLabel: "repositorio de código", IntegrationBranch: "main"}

	encoded, err := json.Marshal(repository)
	if err != nil {
		t.Fatal(err)
	}
	var text bytes.Buffer
	repository.writeText(&text)

	if strings.Contains(string(encoded), "pendingBranches") || strings.Contains(text.String(), "pendiente") {
		t.Errorf("sin pendientes no hay rastro: %s / %s", encoded, text.String())
	}
}
