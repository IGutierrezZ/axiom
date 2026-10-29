package branch

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `dirty_worktree` de `ensure` en modo `in_place` y el plan sin worktrees (REQ-25.8, REQ-25.9,
// design D-05): el rechazo nombra el repositorio, los ficheros y una continuación ejecutable,
// y el planificador no coloca ni mide worktrees cuando las ramas se despliegan en el propio
// repositorio. Pruebas puras, sin git.

var dirtyTestRepo = RepoPlan{
	Name:              "axiom",
	GitRoot:           modeTestCWD,
	IntegrationBranch: "main",
	Branches:          []BranchPlan{{Name: "feat/x", Kind: KindChange}},
}

func TestDirtyWorktreeNamesTheFilesAndAnExecutableContinuation(t *testing.T) {
	refusal := DirtyWorktree("ensure", "x", modeTestCWD, LaneSDD, dirtyTestRepo, modeTestCWD, []string{"datos.txt", "carpeta/"}, false)

	if refusal.Code != CodeDirtyWorktree || refusal.Decision != nil {
		t.Fatalf("rechazo = %+v, se esperaba dirty_worktree sin decisión", refusal)
	}
	for _, want := range []string{"axiom", modeTestCWD, "feat/x", "datos.txt", "carpeta/", "no se ha modificado nada"} {
		if !strings.Contains(refusal.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
		}
	}
	for _, want := range []string{
		"git -C " + pathquote.Quote(modeTestCWD) + " stash push --include-untracked",
		"axiom branch ensure --change x --mode in_place --cwd " + pathquote.Quote(modeTestCWD),
		"axiom branch ensure --change x --mode worktree --cwd " + pathquote.Quote(modeTestCWD),
	} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	}
	if refusal.Details["repository"] != modeTestCWD || refusal.Details["branch"] != "feat/x" || refusal.Details["paths"] != "datos.txt\ncarpeta/" {
		t.Errorf("details = %v", refusal.Details)
	}
}

// Un cambio ya materializado en `in_place` no puede cambiar de modo (plan_conflict): la
// continuación no ofrece `--mode worktree`, solo confirmar o guardar y repetir.
func TestDirtyWorktreeOffersNoWorktreeModeForAnAlreadyMaterializedChange(t *testing.T) {
	refusal := DirtyWorktree("ensure", "x", modeTestCWD, LaneODD, dirtyTestRepo, modeTestCWD, []string{"datos.txt"}, true)

	if strings.Contains(refusal.Continuation, "--mode worktree") {
		t.Errorf("la continuación %q no debe ofrecer un modo que el registro rechazaría", refusal.Continuation)
	}
	if want := "axiom branch ensure --change x --lane odd --mode in_place --cwd " + pathquote.Quote(modeTestCWD); !strings.Contains(refusal.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q (con el carril odd)", refusal.Continuation, want)
	}
}

// Con muchos ficheros, el mensaje nombra los primeros y resume el resto; Details los lleva todos.
func TestDirtyWorktreeSummarisesALongListOfFiles(t *testing.T) {
	var paths []string
	for i := 0; i < maxDirtyPathsListed+3; i++ {
		paths = append(paths, fmt.Sprintf("fichero-%02d.txt", i))
	}

	refusal := DirtyWorktree("ensure", "x", modeTestCWD, LaneSDD, dirtyTestRepo, modeTestCWD, paths, false)

	if !strings.Contains(refusal.Message, "fichero-09.txt") || strings.Contains(refusal.Message, "fichero-10.txt") || !strings.Contains(refusal.Message, "y 3 más") {
		t.Errorf("el mensaje %q debe nombrar los %d primeros y resumir «y 3 más»", refusal.Message, maxDirtyPathsListed)
	}
	if got := strings.Count(refusal.Details["paths"], "\n") + 1; got != len(paths) {
		t.Errorf("details.paths lleva %d ficheros, se esperaban %d", got, len(paths))
	}
}

// design D-05 y REQ-25.8: en `in_place` las ramas se planifican igual, pero sin ruta de
// worktree: ni se coloca ni se mide en Windows, de modo que ni una raíz de worktrees dentro del
// repositorio ni una demasiado larga impiden el plan, y el workspace es el propio checkout.
func TestPlanInPlaceHasNoWorktreesAndIgnoresThePlacementRules(t *testing.T) {
	in := embeddedInput("x", "core", "qa")
	in.PlanningRepo = planningFacts("core", "qa")
	in.InPlace = true
	// Esta raíz estaría dentro del repositorio y superaría el límite de Windows con worktrees.
	in.WorktreeRoot = filepath.Join(testMaster, strings.Repeat("r", 300))
	in.PathRules = PathRules{GOOS: "windows"}

	plan := mustPlan(t, in)

	repo := plan.Repositories[0]
	wantNames(t, "ramas", branchNames(repo), "feat/x", "feat/x-core", "feat/x-qa")
	for _, planned := range repo.Branches {
		if planned.Worktree != "" {
			t.Errorf("la rama %s lleva el worktree %q: en in_place no hay worktrees", planned.Name, planned.Worktree)
		}
	}
	if plan.WorkspaceRoot != testMaster {
		t.Errorf("workspaceRoot = %q, se esperaba el propio checkout %q", plan.WorkspaceRoot, testMaster)
	}

	t.Run("el mismo plan en modo worktree sí se rechaza", func(t *testing.T) {
		in.InPlace = false
		mustRefuse(t, in, CodePlanConflict)
	})
}
