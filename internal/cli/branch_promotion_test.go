package cli

import (
	"strings"
	"testing"
)

// `axiom branch ensure --promotion` (C-02-a, REQ-25.47, REQ-25.45): la ayuda documenta la bandera
// y cada valor que no es una respuesta es un error de uso con código 2 y sin leer nada del
// repositorio. El comportamiento del motor se prueba en internal/branch/engine.

func TestBranchPromotionHelpDocumentsTheFlag(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "ensure", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{
		"--promotion branch|release|<repo>=branch|release", "repetible", "git.main_branch", "decision_required", "plan_conflict",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
		}
	}
}

func TestBranchPromotionUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"valor vacío", []string{"ensure", "--change", "x", "--promotion", ""}, "--promotion"},
		{"valor en blanco", []string{"ensure", "--change", "x", "--promotion", " "}, "--promotion"},
		{"ask no es una respuesta", []string{"ensure", "--change", "x", "--promotion", "ask"}, "ask"},
		{"modo desconocido", []string{"ensure", "--change", "x", "--promotion", "Branch"}, "Branch"},
		{"repositorio sin modo", []string{"ensure", "--change", "x", "--promotion", "x="}, "x="},
		{"modo sin repositorio", []string{"ensure", "--change", "x", "--promotion", "=branch"}, "=branch"},
		{"dos valores sin cualificar", []string{"ensure", "--change", "x", "--promotion", "branch", "--promotion", "release"}, "sin <repo>"},
		{"repositorio repetido", []string{"ensure", "--change", "x", "--promotion", "a=branch", "--promotion", "a=release"}, "repite el repositorio"},
		{"junto a --archive", []string{"ensure", "--change", "x", "--archive", "--promotion", "branch"}, "--archive"},
		{"--promotion sin valor", []string{"ensure", "--change", "x", "--promotion"}, "promotion"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)

			if code != 2 || stdout != "" {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			for _, want := range []string{"axiom branch ensure --help", tc.want} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr %q debe contener %q", stderr, want)
				}
			}
		})
	}
}
