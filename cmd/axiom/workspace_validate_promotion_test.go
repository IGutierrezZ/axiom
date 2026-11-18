package main

import (
	"strings"
	"testing"
)

// REQ-25.26: `promotion` sin `main_branch` no tiene efecto; `workspace validate` lo
// avisa sin invalidar la configuración (código 0 y COMPLIANT).
func TestWorkspaceValidateWarnsAboutPromotionWithoutMainBranch(t *testing.T) {
	master := validateWorkspace(t, "git:\n  promotion: branch\n", "artifact_store: hybrid\n")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("un aviso no puede invalidar la configuración, salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Advertencias", "'git.promotion' (branch)", "git.main_branch")
	if strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("la salida no debe ser NON-COMPLIANT:\n%s", output)
	}
}

// Con dos etapas coherentes no hay aviso de promoción.
func TestWorkspaceValidateAcceptsPromotionWithMainBranchWithoutWarnings(t *testing.T) {
	master := validateWorkspace(t, "git:\n  integration_branch: develop\n  main_branch: master\n  promotion: ask\n", "artifact_store: hybrid\n")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT")
	if strings.Contains(output, "Advertencias") {
		t.Errorf("la configuración coherente no avisa:\n%s", output)
	}
}
