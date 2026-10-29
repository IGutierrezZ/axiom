package cli

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch ensure --prefix --branch --base --repo` (INC-25, S4.12, REQ-25.12, REQ-25.13): la
// ayuda, los errores de uso con código 2, y las excepciones por cambio (registro, idempotencia,
// `plan_conflict` e `invalid_name`) con código de salida, JSON y texto, sobre repositorios reales.
// El comportamiento del motor se prueba en internal/branch/engine.

func TestBranchEnsureHelpDocumentsTheExceptionFlagsTheMultirepoAndTheOddLane(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "ensure", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{
		"--prefix <prefijo>", "--branch <rama>", "--base <ref>", "--repo <ruta>", "repetible",
		"invalid_name", "plan_conflict", "specs_repo_not_on_integration", "--lane odd", "odd/tasks/<cambio>.md", "kickoff",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
		}
	}
}

func TestBranchEnsureExceptionFlagsUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"prefijo vacío", []string{"ensure", "--change", "x", "--prefix", ""}, "--prefix"},
		{"rama vacía", []string{"ensure", "--change", "x", "--branch", " "}, "--branch"},
		{"base vacía", []string{"ensure", "--change", "x", "--base", ""}, "--base"},
		{"repositorio vacío", []string{"ensure", "--change", "x", "--repo", ""}, "--repo"},
		{"un repositorio vacío entre varios", []string{"ensure", "--change", "x", "--repo", "a", "--repo", ""}, "--repo"},
		{"--prefix sin valor", []string{"ensure", "--change", "x", "--prefix"}, "prefix"},
		{"--repo sin valor", []string{"ensure", "--change", "x", "--repo"}, "repo"},
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

// REQ-25.12 (harness S4.13): `--branch` fija el nombre de la rama del cambio y se registra; repetirlo
// es idempotente (código 0 y el mismo JSON); una excepción posterior contradictoria sale con 1 y
// `plan_conflict`, con la continuación que repite la excepción registrada y sin cambiar nada; un
// nombre inválido sale con 1 e `invalid_name` sin efectos.
func TestBranchEnsureExceptionBranchIsIdempotentAndAContradictionExitsWithCode1(t *testing.T) {
	repo := newBranchRepository(t)

	code, first, stderr := runBranchCLI(t, "ensure", "--change", "x", "--branch", "inc/x", "--mode", "in_place", "--json", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, first)
	}
	ready := decodeBranchResult(t, first)
	if ready.Status != "ready" || len(ready.Repositories) != 1 || len(ready.Repositories[0].Branches) != 1 || ready.Repositories[0].Branches[0].Name != "inc/x" {
		t.Errorf("resultado = %+v, se esperaba la rama inc/x", ready)
	}
	if got := branchGitOutput(t, repo, "branch", "--show-current"); got != "inc/x" {
		t.Errorf("el repositorio está en %q, se esperaba inc/x", got)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/x"); got != "" {
		t.Errorf("no debía crearse feat/x, hay %q", got)
	}

	code, again, _ := runBranchCLI(t, "ensure", "--change", "x", "--branch", "inc/x", "--json", "--cwd", repo)
	if code != 0 || again != first {
		t.Errorf("repetir la excepción debe dar el mismo JSON byte a byte: código %d\n%s\n---\n%s", code, first, again)
	}

	before := branchGitOutput(t, repo, "for-each-ref", "refs/heads") + branchGitOutput(t, repo, "rev-parse", "HEAD")
	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--branch", "feat/x", "--json", "--cwd", repo)
	if code != 1 || stderr != "" {
		t.Fatalf("contradicción: código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	refused := decodeBranchResult(t, stdout)
	if refused.Status != "refused" || refused.Code != "plan_conflict" || !strings.Contains(refused.Message, "inc/x") || !strings.Contains(refused.Message, "feat/x") {
		t.Errorf("status %q, code %q, mensaje %q", refused.Status, refused.Code, refused.Message)
	}
	if want := "axiom branch ensure --change x --branch inc/x --cwd " + pathquote.Quote(repo); !strings.Contains(refused.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q", refused.Continuation, want)
	}
	if after := branchGitOutput(t, repo, "for-each-ref", "refs/heads") + branchGitOutput(t, repo, "rev-parse", "HEAD"); after != before {
		t.Errorf("la contradicción no debía cambiar el repositorio:\n%s\n---\n%s", before, after)
	}

	code, stdout, stderr = runBranchCLI(t, "ensure", "--change", "x", "--branch", "feat/x", "--cwd", repo)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "plan_conflict") || !strings.Contains(stderr, "Continuación:") {
		t.Errorf("en texto: código %d, stdout %q, stderr %q; se esperaba 1, stdout vacío y el código con la continuación por stderr", code, stdout, stderr)
	}

	code, stdout, _ = runBranchCLI(t, "ensure", "--change", "y", "--prefix", "mi prefijo", "--json", "--cwd", repo)
	invalid := decodeBranchResult(t, stdout)
	if code != 1 || invalid.Code != "invalid_name" || invalid.Details["field"] != "--prefix" {
		t.Errorf("prefijo inválido: código %d, code %q, details %v; se esperaba 1 e invalid_name sobre --prefix", code, invalid.Code, invalid.Details)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "mi*"); got != "" {
		t.Errorf("un nombre inválido no crea nada, hay %q", got)
	}
}

// REQ-25.13 (harness S4.13): `--lane odd --repo <ruta>` incluye los repositorios indicados; una
// ruta que no existe se omite con `not_a_git_repository` sin impedir el resto, y el documento ODD
// y la ausencia de kickoff se mantienen.
func TestBranchEnsureOddLaneRepoFlagSkipsAPathThatIsNotARepository(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "mi-feature", "--lane", "odd", "--repo", "no-existe", "--mode", "in_place", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Lane != "odd" || len(result.Repositories) != 1 || len(result.Skipped) != 1 || result.Skipped[0].Code != "not_a_git_repository" || !strings.Contains(result.Skipped[0].Message, "no-existe") {
		t.Errorf("carril %q, repositorios %+v, omitidos %+v; se esperaba el repositorio del proyecto y no-existe omitido", result.Lane, result.Repositories, result.Skipped)
	}
	if got := branchGitOutput(t, repo, "branch", "--show-current"); got != "feat/mi-feature" {
		t.Errorf("el repositorio está en %q, se esperaba feat/mi-feature", got)
	}
}
