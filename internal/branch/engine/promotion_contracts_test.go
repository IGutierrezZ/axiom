package engine

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Contratos que la promoción no cambia (C-02-a, REQ-25.50 y REQ-25.45, W1 del informe de
// verificación): con `main_branch` declarado y modo `branch`, la publicación, el relevo y el
// archivado siguen refiriéndose a la rama de integración. Con git real y sin red.

// REQ-25.50 (escenario «La publicación y el relevo siguen apuntando a la integración»): con
// `feat/x` ya fusionada en `develop`, `status` presenta la siguiente PR hacia `master`, pero el
// destino de `publish` y el del relevo siguen siendo los de la primera etapa.
func TestPublicationAndHandOffKeepTargetingTheIntegrationInBranchMode(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	wantReady(t, ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"}))
	commitFile(t, repo, "cambio.txt", "cambio de x")
	runGit(t, repo, "switch", "-q", "develop")
	mergeInto(t, repo, "feat/x")
	runGit(t, repo, "switch", "-q", "feat/x")
	if got := statusPRTarget(t, repo, "feat/x"); got != "master" {
		t.Fatalf("la preparación debía dejar la siguiente PR de status hacia master, es %q", got)
	}

	target, refusal := publicationTarget(t, branch.PublicationRequest{CWD: repo, Change: "x"})
	if refusal != nil {
		t.Fatalf("publish: rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if target.Branch != "feat/x" || target.IntegrationBranch != "develop" || target.PRTarget != "develop" {
		t.Errorf("publish = %+v, se esperaba feat/x hacia develop", target)
	}

	plan, refusal := handOff(t, handOffRequest(repo, "spec", "", "design"))
	if refusal != nil {
		t.Fatalf("relevo: rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if plan.Publish.Branch != "feat/x" || plan.Publish.PRTarget != "develop" {
		t.Errorf("relevo = %+v, se esperaba feat/x hacia develop", plan.Publish)
	}
}

// REQ-25.45 y REQ-25.50 (escenarios «Con dos etapas en modo `branch` el archivado espera solo a la
// integración» y «El archivado se rechaza si el cambio no está en la integración aunque esté en la
// rama principal»): `ensure --archive` evalúa `develop`, nunca `master`.
func TestEnsureArchiveWaitsOnlyForTheIntegrationInBranchMode(t *testing.T) {
	archive := func(repo string) branch.Result {
		return ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Archive: true})
	}
	publishChangeFolder := func(repo, target string) {
		runGit(t, repo, "switch", "-q", target)
		writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
		runGit(t, repo, "add", "openspec")
		commit(t, repo, "cambio x en "+target)
		runGit(t, repo, "switch", "-q", "develop")
	}

	t.Run("fusionado solo en develop se archiva desde develop", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		publishChangeFolder(repo, "develop")
		develop := runGit(t, repo, "rev-parse", "develop")

		result := archive(repo)

		wantReady(t, result)
		repository := onlyRepository(t, result)
		if repository.Base == nil || repository.Base.Ref != "develop" || repository.Base.SHA != develop {
			t.Errorf("base = %+v, se esperaba develop en %s", repository.Base, develop)
		}
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != develop {
			t.Errorf("feat/x-archive apunta a %s, se esperaba develop (%s)", got, develop)
		}
	})

	t.Run("fusionado solo en master se rechaza", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		publishChangeFolder(repo, "master")

		result := archive(repo)

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if result.Details["base"] != "develop" {
			t.Errorf("detalles = %v, se esperaba la base develop", result.Details)
		}
		if branchExists(t, repo, "feat/x-archive") {
			t.Error("el rechazo no crea la rama de archivado")
		}
	})
}
