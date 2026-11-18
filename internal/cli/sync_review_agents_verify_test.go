package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
	"github.com/IGutierrezZ/axiom/v3/internal/verify"
)

func claudeSDDSelection() model.Selection {
	return model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode},
		Components: []model.ComponentID{model.ComponentPersona, model.ComponentSDD},
		Persona:    model.PersonaNeutral,
	}
}

// reviewAgentChecks filtra los checks de un informe que tocan agentes review-*.
func reviewAgentChecks(report verify.Report) []verify.CheckResult {
	var out []verify.CheckResult
	for _, check := range report.Checks {
		if strings.Contains(check.ID, filepath.Join(".claude", "agents", "review-")) {
			out = append(out, check)
		}
	}
	return out
}

// Regresión: con RDD suspendido, sync retira los agentes review-* y la
// verificación posterior debe comprobar su ausencia, no su existencia.
func TestWorkspaceSyncVerifyChecksReviewAgentsAbsentWhileSuspended(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	t.Cleanup(rddharness.SetForTesting(false))
	if rddharness.Enabled() {
		t.Skip("binario compilado con rdd_harness: el apagado no aplica")
	}
	home, workspace, _ := footprintEnv(t)
	selection := claudeSDDSelection()

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync: %v", err)
	}
	report := runPostSyncVerificationScoped(home, workspace, ScopeWorkspace, selection)
	if report.Failed != 0 || !report.Ready {
		t.Fatalf("la verificación tiene %d fallos: %+v", report.Failed, report.Checks)
	}
	checks := reviewAgentChecks(report)
	if len(checks) == 0 {
		t.Fatal("la verificación no cubre los agentes review-*")
	}
	for _, check := range checks {
		if check.Description != "retired managed file removed" {
			t.Errorf("%s: descripción %q; quiere comprobación de ausencia", check.ID, check.Description)
		}
	}
}

// Con RDD activo los agentes review-* se instalan y se exige que existan.
func TestWorkspaceSyncVerifyRequiresReviewAgentsWhileEnabled(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	t.Cleanup(rddharness.SetForTesting(true))
	home, workspace, _ := footprintEnv(t)
	selection := claudeSDDSelection()

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync: %v", err)
	}
	report := runPostSyncVerificationScoped(home, workspace, ScopeWorkspace, selection)
	checks := reviewAgentChecks(report)
	if len(checks) == 0 {
		t.Fatal("la verificación no cubre los agentes review-*")
	}
	for _, check := range checks {
		if check.Description != "synced file exists" {
			t.Errorf("%s: descripción %q; quiere comprobación de existencia", check.ID, check.Description)
		}
	}
	if err := os.Remove(filepath.Join(workspace, ".claude", "agents", "review-risk.md")); err != nil {
		t.Fatalf("borrar agente: %v", err)
	}
	if report := runPostSyncVerificationScoped(home, workspace, ScopeWorkspace, selection); report.Failed == 0 {
		t.Error("con RDD activo, falta de review-risk.md debe fallar la verificación")
	}
}
