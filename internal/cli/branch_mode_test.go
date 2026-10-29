package cli

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch ensure --mode` y `decision_required` (INC-25, S4.4, REQ-25.8): la bandera, su
// validación, la ayuda y el contrato de salida de `ask` sobre repositorios reales. El
// comportamiento del motor se prueba en internal/branch/engine.

// askModeHub registra el proyecto en el hub aislado de la prueba (isolateBranchGit fija HOME y
// USERPROFILE) con el modo de trabajo `ask`, y devuelve el gestor.
func askModeHub(t *testing.T, project string) *hub.Manager {
	t.Helper()
	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Register(project, "ejemplo", ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Workspaces[0].WorktreeMode = "ask"
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestBranchEnsureModeFlagUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"ask no es una respuesta", []string{"ensure", "--change", "x", "--mode", "ask"}, "--mode 'ask'"},
		{"modo desconocido", []string{"ensure", "--change", "x", "--mode", "siempre"}, "--mode 'siempre'"},
		{"modo vacío", []string{"ensure", "--change", "x", "--mode", ""}, "--mode ''"},
		{"--mode sin valor", []string{"ensure", "--change", "x", "--mode"}, "mode"},
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
			if strings.Contains(tc.want, "--mode '") && !strings.Contains(stderr, "worktree o in_place") {
				t.Errorf("stderr %q debe listar los modos válidos (worktree o in_place)", stderr)
			}
		})
	}
}

func TestBranchEnsureHelpDocumentsTheModeFlagAndTheAskDecision(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "ensure", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"--mode worktree|in_place", "decision_required", "ask", "preferencias personales", "registro", "no espera entrada", "plan_conflict"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
		}
	}
}

// REQ-25.8, escenario «Modo `ask` devuelve `decision_required` sin crear nada»: con el hub en
// `ask` y sin respuesta, salida 1, resultado tipado con la decisión, sin rama, worktree ni
// registro; en texto, el código y las dos órdenes por stderr.
func TestBranchEnsureAskModeReturnsDecisionRequiredWithExitCode1(t *testing.T) {
	repo := newBranchRepository(t)
	askModeHub(t, repo)

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)

	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "decision_required" || result.StatusLabel != "decisión requerida" || result.Code != "decision_required" || result.Message == "" || result.Continuation == "" {
		t.Errorf("status %q (%q), code %q, mensaje %q, continuación %q", result.Status, result.StatusLabel, result.Code, result.Message, result.Continuation)
	}
	if result.Decision == nil || result.Decision.Question == "" || len(result.Decision.Choices) != 2 {
		t.Fatalf("decision = %+v, se esperaba la pregunta y dos opciones", result.Decision)
	}
	for i, answer := range []string{"worktree", "in_place"} {
		choice := result.Decision.Choices[i]
		want := "axiom branch ensure --change x --mode " + answer + " --cwd " + pathquote.Quote(repo)
		if choice.Answer != answer || choice.Label == "" || choice.Effect == "" || choice.Invocation != want {
			t.Errorf("opción %d = %+v, se esperaba %s con etiqueta, efecto e invocación %q", i, choice, answer, want)
		}
	}
	if len(result.Repositories) != 0 || result.Registry != nil {
		t.Errorf("decision_required no describe repositorios ni registro: %+v", result)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/x"); got != "" {
		t.Errorf("no debía crearse ninguna rama, hay %q", got)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
		t.Errorf("no debía crearse ningún worktree:\n%s", got)
	}

	code, stdout, stderr = runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "decision_required") || !strings.Contains(stderr, "Continuación:") ||
		!strings.Contains(stderr, "--mode worktree") || !strings.Contains(stderr, "--mode in_place") {
		t.Errorf("en texto: código %d, stdout %q, stderr %q; se esperaba 1, stdout vacío y el código con las dos órdenes por stderr", code, stdout, stderr)
	}
}

// REQ-25.8, escenario «La respuesta a `ask` se registra por cambio» (harness S4.5): con
// `--mode worktree` se crea y se registra con origen `answer`; después, sin `--mode`, el mismo
// cambio no vuelve a preguntar; la preferencia personal del hub no cambia.
func TestBranchEnsureAskAnswerWithTheModeFlagIsRecordedAndReused(t *testing.T) {
	repo := newBranchRepository(t)
	manager := askModeHub(t, repo)

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--mode", "worktree", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	answered := decodeBranchResult(t, stdout)
	if answered.Status != "ready" || answered.Mode != "worktree" || answered.ModeSource != "answer" || answered.ModeSourceLabel == "" || answered.Decision != nil {
		t.Errorf("status %q, modo %q, origen %q (%q), decisión %+v", answered.Status, answered.Mode, answered.ModeSource, answered.ModeSourceLabel, answered.Decision)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/x"); got == "" {
		t.Error("feat/x debía crearse con la respuesta")
	}

	code, stdout, _ = runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)
	if code != 0 {
		t.Fatalf("la repetición sin --mode salió con %d, se esperaba 0\n%s", code, stdout)
	}
	if again := decodeBranchResult(t, stdout); again.Mode != "worktree" || again.ModeSource != "answer" {
		t.Errorf("la repetición reutiliza la respuesta: modo %q, origen %q", again.Mode, again.ModeSource)
	}

	code, stdout, _ = runBranchCLI(t, "ensure", "--change", "y", "--json", "--cwd", repo)
	if code != 1 || decodeBranchResult(t, stdout).Status != "decision_required" {
		t.Errorf("otro cambio sin respuesta debe volver a preguntar: código %d\n%s", code, stdout)
	}

	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workspaces[0].WorktreeMode != "ask" {
		t.Errorf("la respuesta no modifica la preferencia del hub: %q", cfg.Workspaces[0].WorktreeMode)
	}
}
