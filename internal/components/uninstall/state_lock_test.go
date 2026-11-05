package uninstall

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/statecoord"
)

// El leer-modificar-escribir de updateStateAfterUninstall corre dentro del
// cerrojo canónico del estado: el RDDMode de un escritor concurrente sobrevive a
// la retirada del agente y, con el cerrojo ocupado, falla rápido sin escribir.
func TestUpdateStateAfterUninstallReReadsLatestStateAfterLockContention(t *testing.T) {
	home := t.TempDir()
	lockPath, err := statecoord.LockPath(home)
	if err != nil {
		t.Fatalf("install state lock path: %v", err)
	}
	held, err := reviewtransaction.AcquireAuthorityFileLock(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{string(model.AgentOpenCode), "claude-code"},
		RDDMode:         string(reviewtransaction.RDDModeOn),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := updateStateAfterUninstall(home, []model.AgentID{model.AgentOpenCode}); err == nil || !strings.Contains(err.Error(), "acquire install state lock") {
		t.Fatalf("error con cerrojo ocupado = %v", err)
	}
	contended, err := state.Read(home)
	if err != nil || len(contended.InstalledAgents) != 2 || contended.RDDMode != string(reviewtransaction.RDDModeOn) {
		t.Fatalf("estado tras la ejecución con cerrojo ocupado = %#v, err = %v", contended, err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	removed, err := updateStateAfterUninstall(home, []model.AgentID{model.AgentOpenCode})
	if err != nil {
		t.Fatalf("updateStateAfterUninstall() error = %v", err)
	}
	if len(removed) != 1 || removed[0] != model.AgentOpenCode {
		t.Fatalf("removed = %v, want [opencode]", removed)
	}
	got, err := state.Read(home)
	if err != nil {
		t.Fatalf("releer estado: %v", err)
	}
	if len(got.InstalledAgents) != 1 || got.InstalledAgents[0] != "claude-code" {
		t.Fatalf("InstalledAgents = %v, want [claude-code]", got.InstalledAgents)
	}
	if got.RDDMode != string(reviewtransaction.RDDModeOn) {
		t.Fatalf("RDDMode pisado: %q, want %q", got.RDDMode, reviewtransaction.RDDModeOn)
	}
}
