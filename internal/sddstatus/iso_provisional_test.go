package sddstatus

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	provDeltaOK = "## ADDED Requirements\n\n### Requirement: Cobro con IVA (prov:30:1)\n\nEl sistema cobra con IVA.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n- **ENTONCES** incluye IVA\n"
	provTasks   = "- [ ] 1.1 Trabajo\n  Cubre: prov:30:1\n"
)

func isoDefaultProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n")
	return ws
}

// REQ-29.3 y REQ-29.4 (C-03-b): un ADDED con provisional, citado por `Cubre:`, no bloquea.
func TestISOAcceptsAProvisionalInAnAddedAndInCubre(t *testing.T) {
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": provDeltaOK}, provTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status)
	if status.NextRecommended != "apply" {
		t.Errorf("nextRecommended = %q, se esperaba apply: %v", status.NextRecommended, status.BlockedReasons)
	}
}

// REQ-29.3: un provisional fuera de un ADDED bloquea con iso_requirement_id.
func TestISORejectsAProvisionalOutsideAdded(t *testing.T) {
	ws := isoDefaultProject(t)
	delta := strings.Replace(provDeltaOK, "ADDED", "MODIFIED", 1)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": delta}, provTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	if tokens := isoTokens(status.BlockedReasons); len(tokens) == 0 || tokens[0] != "iso_requirement_id" {
		t.Fatalf("tokens = %v, se esperaba iso_requirement_id: %v", tokens, status.BlockedReasons)
	}
	if joined := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(joined, "prov:30:1") || !strings.Contains(joined, "MODIFIED") {
		t.Errorf("el motivo debe nombrar la sección y el provisional: %v", status.BlockedReasons)
	}
}

// REQ-29.4: un provisional citado que ningún ADDED declara bloquea con iso_cubre_unknown, y
// el ADDED sin cita bloquea por cobertura.
func TestISOCubreOfAnUndeclaredProvisional(t *testing.T) {
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": provDeltaOK}, "- [ ] 1.1 Trabajo\n  Cubre: prov:30:1, prov:30:9\n", "")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status, "iso_cubre_unknown")
	if joined := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(joined, "prov:30:9") {
		t.Errorf("el motivo debe nombrar el provisional citado: %v", status.BlockedReasons)
	}

	other := isoDefaultProject(t)
	seedISOChange(t, other, "demo", "# Propuesta\n", map[string]string{"pagos": provDeltaOK}, "- [ ] 1.1 Trabajo\n", "")
	wantTokens(t, resolveFrom(t, other.Master, "demo"), "iso_coverage")
}

// REQ-29.3: dos ADDED con el mismo provisional, en dominios distintos, bloquean.
func TestISORejectsADuplicatedProvisional(t *testing.T) {
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": provDeltaOK, "envios": provDeltaOK}, provTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	if tokens := isoTokens(status.BlockedReasons); len(tokens) == 0 || tokens[0] != "iso_requirement_id" {
		t.Fatalf("tokens = %v: %v", tokens, status.BlockedReasons)
	}
	if joined := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(joined, "se repite") {
		t.Errorf("el motivo debe decir que el provisional se repite: %v", status.BlockedReasons)
	}
}
