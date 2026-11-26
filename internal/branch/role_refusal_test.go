package branch

import (
	"strings"
	"testing"
)

// RF-BRA-33: cada rechazo de `--role` lleva su código tipado, nombra lo que pasó y no sugiere
// nada que cree ramas.
func TestRoleRefusals(t *testing.T) {
	tests := []struct {
		name    string
		refusal *Refusal
		code    Code
		wants   []string
		detail  map[string]string
	}{
		{"archive", RoleArchiveConflict("x", "/p"), CodePlanConflict, []string{"--role", "--archive", "sin --role"}, map[string]string{"change": "x"}},
		{"odd", RoleNotApplicable("x", "backend", "/p"), CodeRoleNotApplicable, []string{"backend", "odd", "--lane odd"}, map[string]string{"role": "backend", "lane": "odd"}},
		{"kickoff", KickoffNotSealed("x", "backend", "/p"), CodeKickoffNotSealed, []string{"kickoff", "axiom sdd kickoff seal"}, map[string]string{"role": "backend"}},
		{"roster", RoleNotInRoster("x", "ops", []string{"backend", "web"}), CodeRoleNotInRoster, []string{"ops", "backend, web"}, map[string]string{"roster": "backend,web", "role": "ops"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.refusal.Code != tt.code {
				t.Fatalf("código = %s, se esperaba %s", tt.refusal.Code, tt.code)
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

func TestRoleCodesHaveSpanishLabels(t *testing.T) {
	for token, want := range map[string]string{
		"role_not_applicable": "rol no aplicable",
		"kickoff_not_sealed":  "kickoff sin sellar",
		"role_not_in_roster":  "rol fuera del roster",
	} {
		if got := Label(token); got != want {
			t.Errorf("Label(%q) = %q, se esperaba %q", token, got, want)
		}
	}
}
