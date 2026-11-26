package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// C-07-f: en monorepo-embedded `roles.<id>.branch_template` no tiene efecto;
// `workspace validate` lo avisa sin invalidar la configuración.
func TestWorkspaceValidateWarnsAboutRoleBranchTemplateInMonorepo(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	base := strings.Replace(validateEmbeddedBase, "    name: Fullstack\n", "    name: Fullstack\n    branch_template: \"feature/{role}-{task_ref}\"\n", 1)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), base)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: hybrid\n")

	output, code := runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if code != 0 {
		t.Fatalf("un aviso no puede invalidar la configuración, salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Advertencias", "roles.fullstack.branch_template", "monorepo-embedded")
}

// La regla de la rama con `openspec/` invalida la configuración y nombra la clave.
func TestWorkspaceValidateRejectsTaskRefInTheOpenspecBranchTemplate(t *testing.T) {
	master := validateWorkspace(t, "git:\n  branch_template: \"feature/{task_ref}\"\n", "artifact_store: hybrid\n")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code == 0 {
		t.Fatalf("la plantilla con {task_ref} en la rama con openspec debía invalidar la configuración:\n%s", output)
	}
	wantOutput(t, output, "git.branch_template", "task_ref")
}
