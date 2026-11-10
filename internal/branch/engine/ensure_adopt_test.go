package engine

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Adopción de la rama publicada (REQ-27.15, DD-13): si la rama del cambio, de rol o de
// archivado no existe en local pero `refs/remotes/origin/<rama>` ya está traída, `ensure` la
// crea desde ella con seguimiento y sin red. Los remotos son repositorios desnudos locales.

// publishBranch publica en el remoto desnudo, desde otro clon, la rama name con un commit propio
// que parte de main, y devuelve su commit. repo no lo ve hasta que haga `git fetch origin`.
func publishBranch(t *testing.T, remote, name string) string {
	t.Helper()
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	runGit(t, clone, "switch", "-q", "-c", name)
	writeFile(t, filepath.Join(clone, "publicado.txt"), name+"\n")
	runGit(t, clone, "add", "-A")
	commit(t, clone, "trabajo publicado en "+name)
	runGit(t, clone, "push", "-q", "origin", name)
	return runGit(t, clone, "rev-parse", "HEAD")
}

func wantAdopted(t *testing.T, repo string, described branch.BranchResult, published string) {
	t.Helper()
	name := described.Name
	if described.CreatedFrom != branch.CreatedFromRemote || described.CreatedFromRef != "refs/remotes/origin/"+name {
		t.Errorf("%s: createdFrom=%q createdFromRef=%q, se esperaba remote y refs/remotes/origin/%s", name, described.CreatedFrom, described.CreatedFromRef, name)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/"+name); got != published {
		t.Errorf("%s apunta a %s, se esperaba el commit publicado %s", name, got, published)
	}
	if got := runGit(t, repo, "config", "branch."+name+".remote"); got != "origin" {
		t.Errorf("el upstream de %s es %q, se esperaba origin", name, got)
	}
	if got := runGit(t, repo, "config", "branch."+name+".merge"); got != "refs/heads/"+name {
		t.Errorf("la rama de seguimiento de %s es %q, se esperaba refs/heads/%s", name, got, name)
	}
}

func wantNoRemoteOperations(t *testing.T, seen [][]string) {
	t.Helper()
	for _, invocation := range seen {
		switch invocation[1] {
		case "fetch", "pull", "ls-remote", "push", "clone":
			t.Errorf("sin --fetch no hay operación remota: %v", invocation)
		}
	}
}

func wantNoBaseWarning(t *testing.T, result branch.Result) {
	t.Helper()
	var text, errText bytes.Buffer
	result.WriteText(&text, &errText)
	if strings.Contains(text.String(), "creada desde la base") || len(result.Warnings) != 0 {
		t.Errorf("una rama adoptada no lleva el aviso de rama creada desde la base ni avisos en el JSON: %v\n%s", result.Warnings, text.String())
	}
}

func TestEnsureAdoptsThePublishedChangeBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	published := publishBranch(t, remote, "feat/x")
	runGit(t, repo, "fetch", "-q", "origin")
	integration := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
	seen := observe(t)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	repository := onlyRepository(t, result)
	wantAdopted(t, repo, branchByName(t, repository, "feat/x"), published)
	if repository.Base == nil || repository.Base.Source != "remote-tracking" || repository.Base.Ref != "origin/main" || repository.Base.SHA != integration || repository.Base.Fetched {
		t.Errorf("base = %+v; la adopción no cambia la semántica de base.source", repository.Base)
	}
	wantNoRemoteOperations(t, *seen)
	wantNoBaseWarning(t, result)
	var text, errText bytes.Buffer
	result.WriteText(&text, &errText)
	if !strings.Contains(text.String(), "adoptada desde origin/feat/x (seguimiento configurado)") {
		t.Errorf("el texto no informa de la adopción:\n%s", text.String())
	}

	state := repositoryState(t, repo)
	again := ensureOf(t, repo, "x")
	wantReady(t, again)
	if branchByName(t, onlyRepository(t, again), "feat/x").CreatedFrom != "" || repositoryState(t, repo) != state {
		t.Errorf("repetir `ensure` es idempotente y no informa createdFrom: %+v", onlyRepository(t, again).Branches)
	}
}

func TestEnsureWithoutAPublishedBranchCreatesItFromTheBaseAndWarns(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	described := branchByName(t, onlyRepository(t, result), "feat/x")
	if described.CreatedFrom != branch.CreatedFromBase || described.CreatedFromRef != "origin/main" {
		t.Errorf("createdFrom=%q createdFromRef=%q, se esperaba base y origin/main", described.CreatedFrom, described.CreatedFromRef)
	}
	var text, errText bytes.Buffer
	result.WriteText(&text, &errText)
	if !strings.Contains(text.String(), "Rama feat/x creada desde la base") || !strings.Contains(text.String(), "git fetch origin") {
		t.Errorf("falta la línea informativa de rama creada desde la base:\n%s", text.String())
	}
	if len(result.Warnings) != 0 {
		t.Errorf("el JSON no lleva avisos nuevos: %v", result.Warnings)
	}
	wantNoRemoteOperations(t, *seen)
}

func TestEnsureRoleBranchesAdoptTheirPublishedBranch(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	remote := remoteWithMain(t, repo)
	sealChange(t, repo, "x", "core", "qa")
	publishedQA := publishBranch(t, remote, "feat/x-qa")
	runGit(t, repo, "fetch", "-q", "origin")

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	repository := onlyRepository(t, result)
	wantAdopted(t, repo, branchByName(t, repository, "feat/x-qa"), publishedQA)
	if core := branchByName(t, repository, "feat/x-core"); core.CreatedFrom != branch.CreatedFromChangeBranch || core.CreatedFromRef != "refs/heads/feat/x" {
		t.Errorf("feat/x-core: createdFrom=%q createdFromRef=%q, se esperaba change_branch y refs/heads/feat/x", core.CreatedFrom, core.CreatedFromRef)
	}
	if tracking := branchByName(t, repository, "feat/x"); tracking.CreatedFrom != branch.CreatedFromBase {
		t.Errorf("feat/x: createdFrom=%q, se esperaba base", tracking.CreatedFrom)
	}
}

func TestEnsureArchiveAdoptsThePublishedArchiveBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	remote := remoteWithMain(t, repo)
	published := publishBranch(t, remote, "feat/x-archive")
	runGit(t, repo, "fetch", "-q", "origin")
	seen := observe(t)

	result := ensureArchive(t, archiveRequest(repo))

	wantReady(t, result)
	wantAdopted(t, repo, wantOnlyArchiveBranch(t, result, "feat/x-archive"), published)
	wantNoRemoteOperations(t, *seen)
}

func TestEnsureArchiveStillRefusesWhenTheBaseLacksTheChangeEvenIfTheBranchIsPublished(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	publishBranch(t, remote, "feat/x-archive")
	runGit(t, repo, "fetch", "-q", "origin")
	state := repositoryState(t, repo)

	result := ensureArchive(t, archiveRequest(repo))

	if result.Code != branch.CodeChangeNotIntegrated {
		t.Errorf("código %q, se esperaba %q: una rama publicada no es una rama integrada", result.Code, branch.CodeChangeNotIntegrated)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("el rechazo no escribe nada:\n%s\n---\n%s", state, got)
	}
}

func TestEnsureNeverMovesAnExistingLocalBranchWhenAPublishedOneExists(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	local := runGit(t, repo, "rev-parse", "refs/heads/main")
	runGit(t, repo, "branch", "feat/x", local)
	publishBranch(t, remote, "feat/x")
	runGit(t, repo, "fetch", "-q", "origin")

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != local {
		t.Errorf("feat/x se movió de %s a %s", local, got)
	}
	if described := branchByName(t, onlyRepository(t, result), "feat/x"); described.CreatedFrom != "" {
		t.Errorf("una rama que ya existía no informa createdFrom: %q", described.CreatedFrom)
	}
}

func TestEnsureFetchAdoptsThePublishedBranchInASingleRun(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	published := publishBranch(t, remote, "feat/x")
	seen := observe(t)

	result := ensureWithFetch(t, repo, "x")

	wantReady(t, result)
	wantAdopted(t, repo, branchByName(t, onlyRepository(t, result), "feat/x"), published)
	if fetches := fetchInvocations(*seen); len(fetches) != 1 {
		t.Errorf("se esperaba un único fetch: %v", fetches)
	}
	wantNoBaseWarning(t, result)
}

func TestEnsureInPlaceAdoptsThePublishedBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	published := publishBranch(t, remote, "feat/x")
	runGit(t, repo, "fetch", "-q", "origin")
	seen := observe(t)

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	wantAdopted(t, repo, branchByName(t, onlyRepository(t, result), "feat/x"), published)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Errorf("rama actual %q, se esperaba feat/x desplegada en el propio repositorio", got)
	}
	wantNoRemoteOperations(t, *seen)
}

// REQ-27.15: createdFrom y createdFromRef solo salen en la ejecución que crea la rama; la
// repetición describe el estado final sin ellos.
func TestEnsureReportsCreatedFromOnlyOnTheRunThatCreatesTheBranch(t *testing.T) {
	repo := embeddedCoreProject(t)

	first := rawResultJSON(t, ensureOf(t, repo, "x"))
	second := rawResultJSON(t, ensureOf(t, repo, "x"))

	if !strings.Contains(first, `"createdFrom": "base"`) || !strings.Contains(first, `"createdFromRef": "main"`) {
		t.Errorf("la ejecución que crea la rama informa createdFrom y createdFromRef:\n%s", first)
	}
	if strings.Contains(second, "createdFrom") {
		t.Errorf("la repetición no informa createdFrom:\n%s", second)
	}
}
