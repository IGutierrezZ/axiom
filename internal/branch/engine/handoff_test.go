package engine

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de `Engine.HandOff` (REQ-26.4, REQ-26.5, REQ-27.12 y DD-12): con git real, entorno
// aislado y sin red. Cubren que la cesión no comprueba lo confirmado, el destino de la publicación
// por topología, las entregas de código y que el cálculo es de solo lectura.

// handOffMonorepo crea un monorepo embebido con el kickoff de x sellado y confirmado en main, y
// ejecuta `ensure` para tener la rama del cambio (y las de rol, con dos o más) desplegadas. Devuelve
// el checkout principal.
func handOffMonorepo(t *testing.T, roles ...string) string {
	t.Helper()
	declared := map[string][]string{}
	for _, role := range roles {
		declared[role] = []string{"."}
	}
	repo := newProject(t, configWithRoles(declared, ""))
	sealChange(t, repo, "x", roles...)
	runGit(t, repo, "add", "openspec")
	commit(t, repo, "kickoff de x")
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	return repo
}

// handOffRequest es la petición de la salida 2 de la puerta dada, desde el worktree dado.
func handOffRequest(worktree, gate, role, handedOffTo string) branch.HandOffRequest {
	return branch.HandOffRequest{
		CWD:         worktree,
		Change:      "x",
		ChangeRoot:  filepath.Join(worktree, "openspec", "changes", "x"),
		Role:        role,
		Gate:        gate,
		HandedOffTo: handedOffTo,
	}
}

func handOff(t *testing.T, request branch.HandOffRequest) (branch.HandOffPlan, *branch.Refusal) {
	t.Helper()
	return Default().HandOff(context.Background(), request)
}

// REQ-26.4 y DD-12: la cesión no inspecciona lo confirmado. Una carpeta del cambio sucia, un fichero
// preparado, una carpeta entera sin seguimiento y el trabajo del rol sin confirmar ya no la rechazan:
// lo informa después `axiom sdd publish`.
func TestHandOffDoesNotCheckWhatIsCommitted(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	folder := filepath.Join(worktree, "openspec", "changes", "x")

	cases := map[string]struct {
		gate, role, handedOffTo string
		dirty                   func(t *testing.T)
	}{
		"worktree limpio": {gate: "spec", handedOffTo: "design", dirty: func(t *testing.T) {}},
		"fichero preparado sin confirmar": {gate: "spec", handedOffTo: "design", dirty: func(t *testing.T) {
			writeFile(t, filepath.Join(folder, "proposal.md"), "propuesta\n")
			runGit(t, worktree, "add", "openspec/changes/x/proposal.md")
		}},
		"fichero sin seguimiento": {gate: "spec", handedOffTo: "design", dirty: func(t *testing.T) {
			writeFile(t, filepath.Join(folder, "design.md"), "diseño\n")
		}},
		"gates.yaml y estado local sucios": {gate: "spec", handedOffTo: "design", dirty: func(t *testing.T) {
			writeFile(t, filepath.Join(folder, "gates.yaml"), "schema: axiom.sdd-gate-ledger/v1\n")
			writeFile(t, filepath.Join(folder, ".gates.lock"), "")
		}},
		"lo sucio fuera de la carpeta del cambio": {gate: "design", handedOffTo: "tasks", dirty: func(t *testing.T) {
			writeFile(t, filepath.Join(worktree, "otro.txt"), "x\n")
		}},
		"role-apply con código sin confirmar": {gate: "role-apply:core", role: "core", handedOffTo: "verify", dirty: func(t *testing.T) {
			writeFile(t, filepath.Join(worktree, "internal", "core.go"), "package core\n")
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.dirty(t)
			t.Cleanup(func() { resetTree(t, worktree) })
			if _, refusal := handOff(t, handOffRequest(worktree, tc.gate, tc.role, tc.handedOffTo)); refusal != nil {
				t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
			}
		})
	}
}

// A-4 y DD-12: en un monorepo la rama del cambio publica hacia la integración; una rama de rol, hacia
// la rama de seguimiento. La invocación de publicación va atada a la rama con --branch y la
// reanudación es `git fetch origin` y `axiom branch ensure`.
func TestHandOffPublishesTheChangeBranchToTheIntegrationAndARoleBranchToTheTracking(t *testing.T) {
	repo := handOffMonorepo(t, "core", "web")
	base := filepath.Base(repo)

	t.Run("rama del cambio", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x", base)
		plan, refusal := handOff(t, handOffRequest(worktree, "design", "", "tasks"))
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		if plan.Topology != workspace.TopologyMonorepoEmbedded || len(plan.Deliveries) != 0 {
			t.Fatalf("plan = %+v, sin entregas de código con la rama del cambio", plan)
		}
		want := branch.HandOffPublish{
			Remote:     "origin",
			Branch:     "feat/x",
			PRTarget:   "main",
			Invocation: branch.PublishInvocation(worktree, "x", "feat/x"),
		}
		if plan.Publish != want {
			t.Errorf("publish = %+v, se esperaba %+v", plan.Publish, want)
		}
		if want := []string{"git fetch origin", "axiom branch ensure --change x --json"}; !reflect.DeepEqual(plan.Resume, want) {
			t.Errorf("resume = %q, se esperaba %q", plan.Resume, want)
		}
	})

	t.Run("rama de rol", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x-core", base)
		plan, refusal := handOff(t, handOffRequest(worktree, "role-apply:core", "core", "apply:web"))
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		if plan.Publish.Branch != "feat/x-core" || plan.Publish.PRTarget != "feat/x" || plan.Publish.InPlace {
			t.Errorf("publish = %+v, se esperaba la rama de rol hacia la de seguimiento", plan.Publish)
		}
		if !strings.Contains(plan.Publish.Invocation, "--branch feat/x-core") {
			t.Errorf("la invocación %q debe atar la rama de rol con --branch", plan.Publish.Invocation)
		}
	})
}

func TestHandOffQuotesAWorkspaceRootWithSpaces(t *testing.T) {
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	spaced := filepath.Join(t.TempDir(), "carpeta con espacios")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", spaced, "main")
	sealChangeAt(t, spaced, "x", "core")
	runGit(t, spaced, "add", "openspec")
	commit(t, spaced, "kickoff")

	plan, refusal := handOff(t, handOffRequest(spaced, "spec", "", "design"))

	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if !strings.Contains(plan.Publish.Invocation, pathquote.Quote(spaced)) {
		t.Errorf("la invocación %q debe citar la raíz con espacios", plan.Publish.Invocation)
	}
}

// Matriz «Selección de repositorio git» y «Estado del push»: el cálculo no ejecuta ninguna
// operación remota, ni `commit`, ni `add`.
func TestHandOffNeverRunsRemoteOperationsOrCommit(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	seen := observe(t)

	if _, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design")); refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) || invocation[1] == "commit" || invocation[1] == "add" || invocation[1] == "push" {
			t.Errorf("HandOff lanzó %v", invocation)
		}
	}
}

// REQ-26.5, segundo escenario: el repositorio de specs de un multirepo se trabaja in situ; el
// destino de la publicación es la rama de integración, sin rama de cambio, sin `prTarget` ni PR.
func TestHandOffOfTheSpecsRepositoryOfAMultirepoIsInPlaceWithoutAPRTarget(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	runGit(t, specs, "add", "-A")
	commit(t, specs, "kickoff de x")
	request := branch.HandOffRequest{
		CWD:         master,
		Change:      "x",
		ChangeRoot:  filepath.Join(specs, "openspec", "changes", "x"),
		Gate:        "spec",
		HandedOffTo: "design",
	}

	plan, refusal := handOff(t, request)

	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if plan.Topology != workspace.TopologyMultirepo || len(plan.Deliveries) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
	if !plan.Publish.InPlace || plan.Publish.Branch != "main" || plan.Publish.PRTarget != "" {
		t.Errorf("publish = %+v, se esperaba in situ en main, sin destino de PR", plan.Publish)
	}
	if want := branch.PublishInvocation(master, "x", "main"); plan.Publish.Invocation != want {
		t.Errorf("invocación = %q, se esperaba %q", plan.Publish.Invocation, want)
	}

	t.Run("un fichero de la carpeta sin confirmar ya no se rechaza", func(t *testing.T) {
		writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "spec.md"), "spec\n")
		t.Cleanup(func() { resetTree(t, specs) })
		if _, refusal := handOff(t, request); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
	t.Run("fuera de la integración no se presenta la publicación de la rama equivocada", func(t *testing.T) {
		runGit(t, specs, "switch", "-q", "-c", "otra")
		t.Cleanup(func() { runGit(t, specs, "switch", "-q", "main") })
		_, refusal := handOff(t, request)
		if refusal == nil || refusal.Code != branch.CodeWrongBranch || refusal.Continuation == "" {
			t.Fatalf("rechazo = %+v, se esperaba wrong_branch con continuación", refusal)
		}
	})
}

// DD-12, role-apply en un multirepo: la publicación de los documentos de specs y, solo de los
// repositorios de código del rol, su push con su propia raíz git y su propio destino. El trabajo del
// rol sin confirmar en el repositorio de código ya no lo impide.
func TestHandOffOfARoleApplyGateInAMultirepoListsOnlyTheCodeRepositories(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	runGit(t, specs, "add", "-A")
	commit(t, specs, "kickoff de x")
	wantReady(t, ensureInPlace(t, master, "x"))
	backend := filepath.Join(master, "repo-backend")
	request := branch.HandOffRequest{
		CWD:         master,
		Change:      "x",
		ChangeRoot:  filepath.Join(specs, "openspec", "changes", "x"),
		Role:        "backend",
		Gate:        "role-apply:backend",
		HandedOffTo: "verify",
	}

	writeFile(t, filepath.Join(backend, "api.go"), "package api\n")
	t.Cleanup(func() { resetTree(t, backend) })

	plan, refusal := handOff(t, request)
	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if !plan.Publish.InPlace || plan.Publish.Branch != "main" {
		t.Errorf("publish = %+v, se esperaba la integración in situ de specs", plan.Publish)
	}
	if len(plan.Deliveries) != 1 {
		t.Fatalf("entregas = %+v, solo repo-backend (los documentos los sube publish)", plan.Deliveries)
	}
	code := plan.Deliveries[0]
	wantSameDirectory(t, "gitRoot de repo-backend", code.GitRoot, backend)
	if code.Repository == "repo-specs" || code.Branch != "feat/x" || code.PRTarget != "main" {
		t.Errorf("entrega de código = %+v", code)
	}
	if want := []string{"git -C " + pathquote.Quote(code.GitRoot) + " push -u origin feat/x"}; !reflect.DeepEqual(code.Steps, want) {
		t.Errorf("pasos del código = %q, se esperaba %q", code.Steps, want)
	}
	wantNoCompoundOrPlatformCommands(t, code.Steps)
}

func wantNoCompoundOrPlatformCommands(t *testing.T, steps []string) {
	t.Helper()
	for _, step := range steps {
		if strings.Contains(step, "&&") || strings.Contains(step, "gh ") || strings.Contains(step, "glab ") {
			t.Errorf("el paso %q no puede ser compuesto ni llamar a la plataforma", step)
		}
		if strings.Contains(step, " -a ") || strings.Contains(step, " HEAD") {
			t.Errorf("el paso %q no puede usar `-a` ni `HEAD`", step)
		}
	}
}
