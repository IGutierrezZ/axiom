package engine

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de `Engine.PublicationTarget` (REQ-27.7, REQ-27.14, REQ-25.19, DD-07): con git real, entorno
// aislado y sin red. Cubren las reglas de rama por topología, la rama de archivado, la atadura con
// `--branch` y que el cálculo es una lectura que no publica ni trae nada.

func publicationTarget(t *testing.T, request branch.PublicationRequest) (branch.PublicationTarget, *branch.Refusal) {
	t.Helper()
	return Default().PublicationTarget(context.Background(), request)
}

func wantPublicationRefusal(t *testing.T, refusal *branch.Refusal, code branch.Code) *branch.Refusal {
	t.Helper()
	if refusal == nil {
		t.Fatalf("se esperaba el rechazo %q y se calculó el destino", code)
	}
	if refusal.Code != code {
		t.Fatalf("código %q (%s), se esperaba %q", refusal.Code, refusal.Message, code)
	}
	if refusal.Continuation == "" {
		t.Errorf("el rechazo %q no ofrece una continuación", code)
	}
	return refusal
}

// statusPRTarget es el `prTarget` que `axiom branch status` da para una rama del plan.
func statusPRTarget(t *testing.T, cwd, name string) string {
	t.Helper()
	result := Default().Status(context.Background(), cwd, "x", branch.LaneSDD)
	for _, repository := range result.Repositories {
		for _, planned := range repository.Branches {
			if planned.Name == name {
				return planned.PRTarget
			}
		}
	}
	t.Fatalf("`status` no lista la rama %s: %+v", name, result.Repositories)
	return ""
}

// REQ-27.7 y matriz «Estado del push»: en un monorepo embebido la rama del cambio y las de rol son
// publicables, cada una con su `prTarget` (el mismo que `status`), y el destino lleva la raíz git del
// worktree donde se está, no la del checkout principal.
func TestPublicationTargetMonorepoChangeAndRoleBranches(t *testing.T) {
	repo := handOffMonorepo(t, "core", "web")
	base := filepath.Base(repo)

	t.Run("rama del cambio", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x", base)
		target, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		wantSameDirectory(t, "gitRoot", target.GitRoot, worktree)
		if target.Branch != "feat/x" || target.IntegrationBranch != "main" || target.InPlace || target.PRTarget != "main" {
			t.Errorf("destino = %+v, se esperaba feat/x hacia main", target)
		}
		if target.Topology != workspace.TopologyMonorepoEmbedded || target.PlanningRel != "openspec" || target.Repository != base {
			t.Errorf("destino = %+v, se esperaba monorepo-embedded con planning home `openspec` en %s", target, base)
		}
		if want := statusPRTarget(t, repo, "feat/x"); target.PRTarget != want {
			t.Errorf("prTarget %q distinto del de `status` (%q)", target.PRTarget, want)
		}
	})
	t.Run("rama de rol", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x-core", base)
		target, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		wantSameDirectory(t, "gitRoot", target.GitRoot, worktree)
		if target.Branch != "feat/x-core" || target.PRTarget != "feat/x" {
			t.Errorf("destino = %+v, se esperaba feat/x-core hacia feat/x", target)
		}
		if want := statusPRTarget(t, repo, "feat/x-core"); target.PRTarget != want {
			t.Errorf("prTarget %q distinto del de `status` (%q)", target.PRTarget, want)
		}
	})
	t.Run("--branch atado a la rama actual", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x", base)
		if _, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x", Branch: "feat/x"}); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
}

// S9, REQ-27.7: la rama de integración de un repositorio que no es el de specs de un multirepo no se
// publica, y HEAD desacoplado no tiene rama. Un rechazo no escribe nada.
func TestPublicationTargetRefusesIntegrationBranchAndDetachedHead(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))

	t.Run("integración en un monorepo", func(t *testing.T) {
		refusal := wantPublicationRefusal(t, mustRefuse(publicationTarget(t, branch.PublicationRequest{CWD: repo, Change: "x"})), branch.CodeIntegrationBranch)
		if refusal.Details["integration"] != "main" {
			t.Errorf("detalles = %v, se esperaba la rama main", refusal.Details)
		}
	})
	t.Run("HEAD desacoplado", func(t *testing.T) {
		runGit(t, worktree, "switch", "-q", "--detach")
		t.Cleanup(func() { runGit(t, worktree, "switch", "-q", "feat/x") })
		wantPublicationRefusal(t, mustRefuse(publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"})), branch.CodeDetachedHead)
	})
	t.Run("rama ajena", func(t *testing.T) {
		runGit(t, worktree, "switch", "-q", "-c", "feat/otro")
		t.Cleanup(func() {
			runGit(t, worktree, "switch", "-q", "feat/x")
			runGit(t, worktree, "branch", "-q", "-D", "feat/otro")
		})
		refusal := wantPublicationRefusal(t, mustRefuse(publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"})), branch.CodeWrongBranch)
		if refusal.Details["current"] != "feat/otro" || !strings.Contains(refusal.Details["expected"], "feat/x") {
			t.Errorf("detalles = %v, se esperaba la rama actual y la esperada", refusal.Details)
		}
	})
}

// mustRefuse deja pasar el rechazo y descarta el destino, para componer las comprobaciones.
func mustRefuse(_ branch.PublicationTarget, refusal *branch.Refusal) *branch.Refusal { return refusal }

// REQ-27.7 regla 4: `--branch` distinto de la rama permitida es `wrong_branch`.
func TestPublicationTargetBranchFlagMismatchIsWrongBranch(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))

	_, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x", Branch: "feat/x-otra"})

	wantPublicationRefusal(t, refusal, branch.CodeWrongBranch)
	if !strings.Contains(refusal.Message, "feat/x-otra") {
		t.Errorf("el mensaje %q debe nombrar la rama autorizada", refusal.Message)
	}
}

// REQ-27.14 y REQ-25.19: la rama de archivado solo se publica con `--archive`, y una rama del cambio
// con `--archive` es `wrong_branch`; la continuación nombra el flag que falta o sobra.
func TestPublicationTargetArchiveBranchNeedsTheArchiveFlag(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	remoteWithMain(t, repo)
	wantReady(t, ensureArchive(t, archiveRequest(repo)))
	worktree := archiveWorktree(t, repo)

	t.Run("con --archive", func(t *testing.T) {
		target, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x", Archive: true})
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		wantSameDirectory(t, "gitRoot", target.GitRoot, worktree)
		if target.Branch != "feat/x-archive" || target.PRTarget != "main" || target.InPlace || target.PlanningRel != "openspec" {
			t.Errorf("destino = %+v, se esperaba feat/x-archive hacia main", target)
		}
	})
	t.Run("sin --archive", func(t *testing.T) {
		_, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
		wantPublicationRefusal(t, refusal, branch.CodeWrongBranch)
		if !strings.Contains(refusal.Continuation, "--archive") {
			t.Errorf("la continuación %q debe nombrar `--archive`", refusal.Continuation)
		}
	})
	t.Run("una rama del cambio con --archive", func(t *testing.T) {
		runGit(t, worktree, "switch", "-q", "-c", "feat/x")
		t.Cleanup(func() {
			runGit(t, worktree, "switch", "-q", "feat/x-archive")
			runGit(t, worktree, "branch", "-q", "-D", "feat/x")
		})
		_, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x", Archive: true})
		wantPublicationRefusal(t, refusal, branch.CodeWrongBranch)
		if !strings.Contains(refusal.Continuation, "quita `--archive`") {
			t.Errorf("la continuación %q debe pedir quitar `--archive`", refusal.Continuation)
		}
		// Sin --archive, esa misma rama es la del cambio y es publicable.
		if _, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"}); refusal != nil {
			t.Errorf("la rama del cambio sin --archive es publicable: %q %s", refusal.Code, refusal.Message)
		}
	})
}

// S9, REQ-27.7 regla 2: el repositorio de specs de un multirepo se trabaja in situ y solo se publica
// su rama de integración, sin `prTarget`; con `--archive` el destino es el mismo, porque no hay rama
// de archivado.
func TestPublicationTargetMultirepoSpecsPublishesTheIntegrationBranchInPlace(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")

	for _, archive := range []bool{false, true} {
		target, refusal := publicationTarget(t, branch.PublicationRequest{CWD: master, Change: "x", Archive: archive})
		if refusal != nil {
			t.Fatalf("archive=%v: rechazo %q: %s", archive, refusal.Code, refusal.Message)
		}
		wantSameDirectory(t, "gitRoot", target.GitRoot, specs)
		if target.Branch != "main" || target.IntegrationBranch != "main" || !target.InPlace || target.PRTarget != "" || target.Topology != workspace.TopologyMultirepo {
			t.Errorf("archive=%v: destino = %+v, se esperaba main in situ sin prTarget", archive, target)
		}
	}

	t.Run("fuera de la integración", func(t *testing.T) {
		runGit(t, specs, "switch", "-q", "-c", "feat/otro")
		t.Cleanup(func() { runGit(t, specs, "switch", "-q", "main"); runGit(t, specs, "branch", "-q", "-D", "feat/otro") })
		_, refusal := publicationTarget(t, branch.PublicationRequest{CWD: master, Change: "x"})
		wantPublicationRefusal(t, refusal, branch.CodeWrongBranch)
	})
	t.Run("HEAD desacoplado", func(t *testing.T) {
		runGit(t, specs, "switch", "-q", "--detach")
		t.Cleanup(func() { runGit(t, specs, "switch", "-q", "main") })
		_, refusal := publicationTarget(t, branch.PublicationRequest{CWD: master, Change: "x"})
		wantPublicationRefusal(t, refusal, branch.CodeDetachedHead)
	})
}

// REQ-25.1, REQ-27.11: calcular el destino es una lectura. No lanza `push`, `commit`, `fetch`, `pull`
// ni `ls-remote`, y no deja nada nuevo en el repositorio.
func TestPublicationTargetOnlyReads(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	remoteWithMain(t, repo)
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	state := repositoryState(t, repo)
	seen := observe(t)

	if _, refusal := publicationTarget(t, branch.PublicationRequest{CWD: worktree, Change: "x"}); refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if _, refusal := publicationTarget(t, branch.PublicationRequest{CWD: repo, Change: "x"}); refusal == nil {
		t.Fatal("la integración debía rechazarse")
	}

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación")
	}
	for _, invocation := range *seen {
		if len(invocation) > 1 && slices.Contains([]string{"push", "commit", "fetch", "pull", "ls-remote", "add"}, invocation[1]) {
			t.Errorf("el cálculo del destino lanzó %v", invocation)
		}
		if isWriteInvocation(invocation) {
			t.Errorf("el cálculo del destino escribió: %v", invocation)
		}
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("el repositorio cambió:\n%s\n---\n%s", state, got)
	}
}
