package branch

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de los constructores de la entrega de la salida 2 (REQ-26.4, REQ-26.5, DD-07, DD-08).
// Matriz de amenazas: «Selección de repositorio git», «Estado del commit», «Estado del push» y
// «Órdenes de PR».

const spacedRoot = `C:\Users\dev name\axiom-wt\x\axiom`

func TestHandOffStepsQuoteTheGitRootAndUseItInsteadOfTheWorkingDirectory(t *testing.T) {
	steps := []string{
		HandOffAddStep(spacedRoot, "openspec/changes/x/gates.yaml"),
		HandOffCommitStep(spacedRoot, "mensaje", "openspec/changes/x/gates.yaml"),
		HandOffPushStep(spacedRoot, "feat/x", true),
		HandOffPushStep(spacedRoot, "main", false),
	}
	for _, step := range steps {
		if !strings.HasPrefix(step, "git -C "+pathquote.Quote(spacedRoot)+" ") {
			t.Errorf("el paso %q debe empezar por git -C con la raíz absoluta citada", step)
		}
	}
}

func TestHandOffAddAndCommitUseAnExplicitPathspecAndNeverCommitAll(t *testing.T) {
	root := `C:\repos\x`
	add := HandOffAddStep(root, "openspec/changes/x/gates.yaml")
	if want := `git -C "C:\repos\x" add -- openspec/changes/x/gates.yaml`; add != want {
		t.Errorf("add = %q, se esperaba %q", add, want)
	}
	commit := HandOffCommitStep(root, "chore(sdd): aprobar spec de x y ceder design", "openspec/changes/x/gates.yaml")
	if want := `git -C "C:\repos\x" commit -m "chore(sdd): aprobar spec de x y ceder design" -- openspec/changes/x/gates.yaml`; commit != want {
		t.Errorf("commit = %q, se esperaba %q", commit, want)
	}
	for _, step := range []string{add, commit} {
		for _, forbidden := range []string{" -a ", " -A", " --all", "&&", "gh ", "glab "} {
			if strings.Contains(step, forbidden) {
				t.Errorf("el paso %q no puede contener %q", step, forbidden)
			}
		}
	}
}

func TestHandOffPathspecWithSpacesIsQuoted(t *testing.T) {
	step := HandOffAddStep(`C:\x`, "openspec/changes/con espacios/gates.yaml")
	if want := `git -C "C:\x" add -- "openspec/changes/con espacios/gates.yaml"`; step != want {
		t.Errorf("add = %q, se esperaba %q", step, want)
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

func TestHandOffCommitMessageIsConventionalAndInSpanish(t *testing.T) {
	if got, want := HandOffCommitMessage("spec", "x", "design"), "chore(sdd): aprobar spec de x y ceder design"; got != want {
		t.Errorf("mensaje = %q, se esperaba %q", got, want)
	}
}

func TestMergeNoticeNamesTheTargetAndTheReasonOfEnsure(t *testing.T) {
	notice := MergeNotice("main")
	for _, fragment := range []string{"fusionada en main", "axiom branch ensure", "no recupera trabajo sin fusionar"} {
		if !strings.Contains(notice, fragment) {
			t.Errorf("el aviso %q debe contener %q", notice, fragment)
		}
	}
	if inPlace := InPlaceNotice("main"); !strings.Contains(inPlace, "main") || strings.Contains(inPlace, "PR esté") {
		t.Errorf("el aviso in situ = %q: nombra la integración y no habla de fusionar una PR", inPlace)
	}
}

func TestUncommittedArtifactsRefusalNamesThePathsAndAnExecutableCommit(t *testing.T) {
	repo := RepoPlan{Name: "axiom", GitRoot: spacedRoot}
	refusal := UncommittedArtifacts("x", repo, []string{"openspec/changes/x/proposal.md", "openspec/changes/x/con espacios.md"})

	if refusal.Code != CodeUncommittedArtifacts || Label(string(refusal.Code)) != "artefactos sin confirmar" {
		t.Fatalf("código %q / %q", refusal.Code, Label(string(refusal.Code)))
	}
	for _, path := range []string{"openspec/changes/x/proposal.md", "openspec/changes/x/con espacios.md"} {
		if !strings.Contains(refusal.Message, path) {
			t.Errorf("el mensaje %q debe nombrar %q", refusal.Message, path)
		}
	}
	root := pathquote.Quote(spacedRoot)
	for _, fragment := range []string{
		"git -C " + root + " add -- openspec/changes/x/proposal.md " + `"openspec/changes/x/con espacios.md"`,
		"git -C " + root + " commit -m ",
	} {
		if !strings.Contains(refusal.Continuation, fragment) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, fragment)
		}
	}
	if refusal.Continuation == "" || strings.Contains(refusal.Continuation, "&&") || strings.Contains(refusal.Continuation, " -a ") {
		t.Errorf("continuación no ejecutable o con -a/&&: %q", refusal.Continuation)
	}
	if !strings.Contains(refusal.Message, "no se ha escrito nada") {
		t.Errorf("el rechazo debe decir que no se ha escrito nada: %q", refusal.Message)
	}
}

func TestUncommittedArtifactsRefusalSummarisesALongList(t *testing.T) {
	var paths []string
	for i := 0; i < 14; i++ {
		paths = append(paths, "openspec/changes/x/f"+string(rune('a'+i))+".md")
	}
	refusal := UncommittedArtifacts("x", RepoPlan{Name: "axiom", GitRoot: `C:\x`}, paths)
	if !strings.Contains(refusal.Message, "y 4 más") {
		t.Errorf("el mensaje debe resumir las rutas que sobran: %q", refusal.Message)
	}
	if got := strings.Count(refusal.Details["paths"], ",") + 1; got != 14 {
		t.Errorf("details.paths lleva %d rutas, se esperaban todas (14)", got)
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
