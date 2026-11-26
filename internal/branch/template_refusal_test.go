package branch

import (
	"errors"
	"strings"
	"testing"
)

// D-11: cada rechazo de plantilla lleva su código tipado, una continuación no vacía y dice que no
// se ha creado nada.
func TestTemplateRefusals(t *testing.T) {
	cause := errors.New("yaml roto")
	tests := []struct {
		name    string
		refusal *Refusal
		code    Code
		wants   []string
		detail  map[string]string
	}{
		{"unresolved_ref", TemplateUnresolved("x", "/p", "git.branch_template", "ref", "", LaneSDD), CodeBranchTemplateUnresolved,
			[]string{"{ref}", "axiom change start --from <id|url>", "--branch <rama>"}, map[string]string{"variable": "ref", "key": "git.branch_template"}},
		{"unresolved_ref_odd", TemplateUnresolved("x", "/p", "git.branch_template", "ref", "", LaneODD), CodeBranchTemplateUnresolved,
			[]string{"--lane odd"}, map[string]string{"variable": "ref"}},
		{"unresolved_task_ref", TemplateUnresolved("x", "/p", "roles.backend.branch_template", "task_ref", "backend", LaneSDD), CodeBranchTemplateUnresolved,
			[]string{"axiom integrations tasks bind --change x --role backend=<id>", "axiom integrations tasks create --change x", "axiom branch ensure --change x --role backend"}, map[string]string{"role": "backend", "variable": "task_ref"}},
		{"not_applicable_shared", TemplateNotApplicable("x", "/p", "git.branch_template", TemplateReasonSharedBranch, LaneSDD), CodeBranchTemplateNotApplicable,
			[]string{"axiom workspace validate"}, map[string]string{"reason": "shared_branch"}},
		{"not_applicable_openspec", TemplateNotApplicable("x", "/p", "git.branch_template", TemplateReasonOpenspecBranch, LaneSDD), CodeBranchTemplateNotApplicable,
			[]string{"axiom workspace validate"}, map[string]string{"reason": "openspec_branch"}},
		{"not_applicable_odd", TemplateNotApplicable("x", "/p", "git.branch_template", TemplateReasonODDLane, LaneODD), CodeBranchTemplateNotApplicable,
			[]string{"axiom workspace validate", "axiom branch ensure --change x --lane odd --branch <rama>"}, map[string]string{"reason": "odd_lane"}},
		{"invalid_name", TemplateInvalidName("x", "/p", "git.branch_template", "ref", "a b", cause), CodeInvalidName,
			[]string{"yaml roto", "a b"}, map[string]string{"field": "ref", "value": "a b"}},
		{"collision", TemplateCollision("x", "/p", "repo", "feature/x", []string{"cambio", "backend"}), CodeInvalidName,
			[]string{"cambio, backend", "feature/x"}, map[string]string{"repository": "repo", "roles": "cambio,backend"}},
		{"reserved", TemplateReservedName("x", "/p", "repo", "main", "main_branch"), CodeInvalidName,
			[]string{"main_branch", "main"}, map[string]string{"reserved": "main_branch"}},
		{"candidate_collision", TemplateCandidateCollision("x", "/p", "repo", "feature/T-1", []string{"backend", "web"}), CodeInvalidName,
			[]string{"backend, web", "feature/T-1"}, map[string]string{"roles": "backend,web"}},
		{"links_unreadable", IntegrationsLinksUnreadable("ensure", "x", "/p", cause), CodeIntegrationsLinksInvalid,
			[]string{"yaml roto", "axiom integrations sync --change x"}, map[string]string{"operation": "ensure"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.refusal.Code != tt.code {
				t.Fatalf("código = %s, se esperaba %s", tt.refusal.Code, tt.code)
			}
			if strings.TrimSpace(tt.refusal.Continuation) == "" {
				t.Fatalf("la continuación no puede estar vacía")
			}
			text := tt.refusal.Message + " " + tt.refusal.Continuation
			for _, want := range tt.wants {
				if !strings.Contains(text, want) {
					t.Errorf("falta %q en %q", want, text)
				}
			}
			if !strings.Contains(tt.refusal.Message, "no se ha creado ninguna rama ni worktree") {
				t.Errorf("debe decir que no hay efectos: %q", tt.refusal.Message)
			}
			for k, v := range tt.detail {
				if tt.refusal.Details[k] != v {
					t.Errorf("details[%s] = %q, se esperaba %q", k, tt.refusal.Details[k], v)
				}
			}
		})
	}
}

func TestTemplateCodesHaveSpanishLabels(t *testing.T) {
	for token, want := range map[string]string{
		"branch_template_unresolved":     "variable de plantilla sin valor",
		"branch_template_not_applicable": "plantilla no aplicable",
		PendingReasonAwaitingRole:        "pendiente de la entrada del rol",
	} {
		if got := Label(token); got != want {
			t.Errorf("Label(%q) = %q, se esperaba %q", token, got, want)
		}
	}
	if PendingReasonAwaitingRole != "awaiting_role" {
		t.Errorf("PendingReasonAwaitingRole = %q", PendingReasonAwaitingRole)
	}
}
