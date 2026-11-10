package branch

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de los constructores de la entrega de la salida 2 (REQ-26.4, REQ-26.5, REQ-27.12, DD-12).
// Matriz de amenazas: «Selección de repositorio git», «Estado del push» y «Órdenes de PR».

const spacedRoot = `C:\Users\dev name\axiom-wt\x\axiom`

func TestHandOffPushStepQuotesTheGitRootAndUsesItInsteadOfTheWorkingDirectory(t *testing.T) {
	for _, step := range []string{HandOffPushStep(spacedRoot, "feat/x", true), HandOffPushStep(spacedRoot, "main", false)} {
		if !strings.HasPrefix(step, "git -C "+pathquote.Quote(spacedRoot)+" ") {
			t.Errorf("el paso %q debe empezar por git -C con la raíz absoluta citada", step)
		}
	}
}

func TestHandOffPushNamesTheConcreteBranchAndNeverHEAD(t *testing.T) {
	if got, want := HandOffPushStep(`C:\x`, "inc/paradas", true), `git -C "C:\x" push -u origin inc/paradas`; got != want {
		t.Errorf("push = %q, se esperaba %q", got, want)
	}
	if got, want := HandOffPushStep(`C:\x`, "main", false), `git -C "C:\x" push origin main`; got != want {
		t.Errorf("push in situ = %q, se esperaba %q", got, want)
	}
	if strings.Contains(HandOffPushStep(`C:\x`, "inc/x", true), "HEAD") {
		t.Error("el push nombra la rama concreta, nunca HEAD")
	}
}

// REQ-27.7 y REQ-27.12: la invocación de publicación va atada a la rama con --branch y cita la
// raíz del espacio de trabajo; sin `&&` ni órdenes de plataforma.
func TestPublishInvocationIsBoundToTheBranchAndQuotesTheWorkspaceRoot(t *testing.T) {
	got := PublishInvocation(spacedRoot, "x", "inc/x")
	if want := "axiom sdd publish --cwd " + pathquote.Quote(spacedRoot) + " --change x --branch inc/x"; got != want {
		t.Errorf("invocación = %q, se esperaba %q", got, want)
	}
	for _, forbidden := range []string{"&&", "gh ", "glab ", "--force", "--no-verify"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("la invocación %q no puede contener %q", got, forbidden)
		}
	}
}

func TestHandOffResumeIsFetchAndEnsureWithoutAPath(t *testing.T) {
	want := []string{"git fetch origin", "axiom branch ensure --change x --json"}
	got := HandOffResume("x")
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("resume = %q, se esperaba %q", got, want)
	}
	for _, step := range got {
		if strings.Contains(step, " -C ") || strings.Contains(step, "&&") {
			t.Errorf("el paso %q no puede llevar -C ni ser compuesto", step)
		}
	}
}

func TestHandOffWrongBranchRefusalCarriesAnExecutableContinuation(t *testing.T) {
	refusal := HandOffWrongBranch("x", `C:\master`, RepoPlan{Name: "axiom", GitRoot: `C:\x`}, "otra", []string{"feat/x"})
	if refusal.Code != CodeWrongBranch || refusal.Continuation == "" || !strings.Contains(refusal.Continuation, "axiom branch ensure --change x") {
		t.Fatalf("rechazo = %+v", refusal)
	}
	detached := HandOffWrongBranch("x", `C:\master`, RepoPlan{Name: "axiom", GitRoot: `C:\x`}, "", []string{"feat/x"})
	if !strings.Contains(detached.Message, "HEAD desacoplado") {
		t.Errorf("mensaje = %q", detached.Message)
	}
}
