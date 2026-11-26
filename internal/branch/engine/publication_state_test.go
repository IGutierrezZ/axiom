package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
)

// Pruebas de `Engine.PublicationState` (RF-PUB-11): con git real, entorno aislado y remotos
// desnudos locales, sin red. Fijan que los hechos son los que `axiom sdd publish` vería (la misma
// lista cerrada de rutas y el mismo rango de exclusiones) y que calcularlos solo lee.

func publicationState(t *testing.T, request branch.PublicationRequest) (branch.PublicationState, *branch.Refusal) {
	t.Helper()
	return Default().PublicationState(context.Background(), request)
}

func wantPublicationState(t *testing.T, request branch.PublicationRequest) branch.PublicationState {
	t.Helper()
	state, refusal := publicationState(t, request)
	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	return state
}

// stateWorktree prepara un monorepo con el cambio x, un remoto desnudo con main y el worktree de
// la rama del cambio, que es desde donde se pide el estado.
func stateWorktree(t *testing.T) (repo, worktree, remote string) {
	t.Helper()
	repo = handOffMonorepo(t, "core")
	remote = remoteWithMain(t, repo)
	return repo, plannedWorktree(t, repo, "x", filepath.Base(repo)), remote
}

func changeFile(worktree, name string) string {
	return filepath.Join(worktree, "openspec", "changes", "x", name)
}

// Matriz de amenazas «Estado del commit»: lo preparado, lo borrado y lo sin seguimiento cuentan; el
// estado local con punto no; una carpeta sin seguimiento se mira fichero a fichero.
func TestPublicationStateDirtyDocsFollowTheClosedListAndIgnoreLocalState(t *testing.T) {
	t.Parallel()
	_, worktree, _ := stateWorktree(t)
	request := branch.PublicationRequest{CWD: worktree, Change: "x"}

	if state := wantPublicationState(t, request); len(state.DirtyDocs) != 0 || state.Unpushed != 0 || state.RemoteMissing || state.RangeUnknown {
		t.Fatalf("estado inicial = %+v, se esperaba sin nada pendiente", state)
	}

	writeFile(t, changeFile(worktree, ".gates.lock"), "")
	writeFile(t, changeFile(worktree, ".axiom-instance"), "")
	if state := wantPublicationState(t, request); len(state.DirtyDocs) != 0 {
		t.Errorf("solo estado local: DirtyDocs = %v, el estado local nunca cuenta", state.DirtyDocs)
	}

	writeFile(t, changeFile(worktree, filepath.Join("specs", "nueva", "spec.md")), "spec\n")
	want := []string{"openspec/changes/x/specs/nueva/spec.md"}
	if state := wantPublicationState(t, request); !slices.Equal(state.DirtyDocs, want) {
		t.Errorf("carpeta sin seguimiento: DirtyDocs = %v, se esperaba %v", state.DirtyDocs, want)
	}

	writeFile(t, changeFile(worktree, "proposal.md"), "propuesta\n")
	runGit(t, worktree, "add", "openspec/changes/x/proposal.md")
	if err := os.Remove(changeFile(worktree, "kickoff.yaml")); err != nil {
		t.Fatal(err)
	}
	want = []string{
		"openspec/changes/x/kickoff.yaml",
		"openspec/changes/x/proposal.md",
		"openspec/changes/x/specs/nueva/spec.md",
	}
	if state := wantPublicationState(t, request); !slices.Equal(state.DirtyDocs, want) {
		t.Errorf("preparado, borrado y sin seguimiento: DirtyDocs = %v, se esperaba %v", state.DirtyDocs, want)
	}

	writeFile(t, filepath.Join(worktree, "requirements.txt"), "x\n")
	writeFile(t, filepath.Join(worktree, "docs", "x.mdx"), "x\n")
	for _, rel := range wantPublicationState(t, request).DirtyDocs {
		if rel == "requirements.txt" || rel == "docs/x.mdx" {
			t.Errorf("DirtyDocs incluye %s, que es código", rel)
		}
	}
}

// Matriz «Estado del push» y «Rutas que parecen documentación»: sin `origin/<rama>`, un commit de
// documentación está sin subir y uno que toca código o un `docs/` sin declarar, además, es de
// código; tras subir al remoto desnudo no queda nada.
func TestPublicationStateUnpushedCommitsAndCodeCommits(t *testing.T) {
	t.Parallel()
	_, worktree, _ := stateWorktree(t)
	request := branch.PublicationRequest{CWD: worktree, Change: "x"}
	commitFile := func(rel, message string) string {
		writeFile(t, filepath.Join(worktree, filepath.FromSlash(rel)), message+"\n")
		runGit(t, worktree, "add", "--", rel)
		commit(t, worktree, message)
		return runGit(t, worktree, "rev-parse", "--short=7", "HEAD")
	}

	commitFile("openspec/changes/x/design.md", "documento")
	state := wantPublicationState(t, request)
	if state.Unpushed != 1 || state.RemoteBranch || len(state.CodeCommits) != 0 || len(state.DirtyDocs) != 0 {
		t.Fatalf("estado = %+v, se esperaba 1 commit de documentación sin subir y sin rama remota", state)
	}
	if state.Target.Branch != "feat/x" || state.Target.PRTarget != "main" || state.Target.PlanningRel != "openspec" {
		t.Errorf("destino = %+v, se esperaba feat/x hacia main", state.Target)
	}
	wantSameDirectory(t, "GitRoot", state.Target.GitRoot, worktree)

	first := commitFile("requirements.txt", "dependencias")
	second := commitFile("docs/x.mdx", "documento sin declarar")
	commitFile("openspec/changes/x/tasks.md", "otro documento")
	state = wantPublicationState(t, request)
	if state.Unpushed != 4 || !slices.Equal(state.CodeCommits, []string{first, second}) {
		t.Errorf("Unpushed = %d, CodeCommits = %v; se esperaban 4 y %v (del más antiguo al más reciente)", state.Unpushed, state.CodeCommits, []string{first, second})
	}

	runGit(t, worktree, "push", "-q", "origin", "feat/x")
	state = wantPublicationState(t, request)
	if state.Unpushed != 0 || !state.RemoteBranch || len(state.CodeCommits) != 0 || len(state.DirtyDocs) != 0 {
		t.Errorf("tras subir, estado = %+v; se esperaba todo vacío y con rama remota", state)
	}
	commitFile("openspec/changes/x/verify-report.md", "tras subir")
	if state := wantPublicationState(t, request); state.Unpushed != 1 || !state.RemoteBranch {
		t.Errorf("con rama remota, estado = %+v; se esperaba 1 commit sin subir", state)
	}
}

func TestPublicationStateWithoutOriginIsRemoteMissing(t *testing.T) {
	t.Parallel()
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	state := wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
	if !state.RemoteMissing || state.Unpushed != 0 || len(state.DirtyDocs) != 0 {
		t.Errorf("estado = %+v, se esperaba RemoteMissing y nada más", state)
	}
}

// Una fusión limpia de `origin/main` en la rama no es trabajo del cambio: lo que ya está en la
// integración remota se excluye y el commit de fusión no lleva rutas.
func TestPublicationStateACleanMergeOfTheIntegrationIsNotCode(t *testing.T) {
	t.Parallel()
	repo, worktree, _ := stateWorktree(t)
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n")
	runGit(t, repo, "add", "main.go")
	commit(t, repo, "código en main")
	runGit(t, repo, "push", "-q", "origin", "main")
	commit(t, worktree, "avance de la rama")
	runGit(t, worktree, "merge", "-q", "--no-ff", "-m", "fusiona main", "origin/main")

	state := wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
	if len(state.CodeCommits) != 0 || state.Unpushed != 2 {
		t.Errorf("estado = %+v, se esperaban 2 commits sin subir (avance y fusión) y ninguno de código", state)
	}
}

// El repositorio de specs de un multirepo se trabaja in situ y se mide contra `origin/<integración>`;
// la raíz git del destino es absoluta, no la del proceso.
func TestPublicationStateMultirepoSpecsInPlaceMeasuresAgainstTheRemoteIntegration(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	remoteWithMain(t, specs)
	request := branch.PublicationRequest{CWD: master, Change: "x"}

	state := wantPublicationState(t, request)
	if !state.Target.InPlace || state.Target.PRTarget != "" || state.Target.Branch != "main" || !state.RemoteBranch {
		t.Fatalf("destino = %+v, se esperaba main in situ con rama remota", state.Target)
	}
	wantSameDirectory(t, "GitRoot", state.Target.GitRoot, specs)
	if want := []string{"openspec/changes/x/kickoff.yaml"}; !slices.Equal(state.DirtyDocs, want) {
		t.Errorf("DirtyDocs = %v, se esperaba %v", state.DirtyDocs, want)
	}

	runGit(t, specs, "add", "-A")
	runGit(t, specs, "commit", "-q", "-m", "docs: kickoff")
	state = wantPublicationState(t, request)
	if len(state.DirtyDocs) != 0 || state.Unpushed != 1 || len(state.CodeCommits) != 0 {
		t.Errorf("estado = %+v, se esperaba 1 commit de documentación sin subir", state)
	}
	runGit(t, specs, "push", "-q", "origin", "main")
	if state := wantPublicationState(t, request); state.Unpushed != 0 {
		t.Errorf("tras subir, Unpushed = %d", state.Unpushed)
	}
}

// RF-ARC-08: con `Archive`, el destino es la rama de archivado y el estado mira su carpeta archivada.
func TestPublicationStateArchiveBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	remoteWithMain(t, repo)
	wantReady(t, ensureArchive(t, archiveRequest(repo)))
	worktree := archiveWorktree(t, repo)
	archived := filepath.Join(worktree, "openspec", "changes", "archive", "2026-01-02-x", "proposal.md")
	writeFile(t, archived, "archivado\n")

	state := wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x", Archive: true})
	if want := []string{"openspec/changes/archive/2026-01-02-x/proposal.md"}; !slices.Equal(state.DirtyDocs, want) {
		t.Errorf("DirtyDocs = %v, se esperaba %v", state.DirtyDocs, want)
	}
	if state.Target.Branch != "feat/x-archive" {
		t.Errorf("Branch = %q, se esperaba la rama de archivado", state.Target.Branch)
	}
	_, refusal := publicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
	wantPublicationRefusal(t, refusal, branch.CodeWrongBranch)
}

// Por encima del umbral, el rango es demasiado grande y no se examina uno a uno.
func TestPublicationStateTooManyCommitsIsRangeTooBig(t *testing.T) {
	t.Parallel()
	_, worktree, _ := stateWorktree(t)
	var script strings.Builder
	for i := 0; i <= branch.PublicationMaxUnpushed; i++ {
		message := fmt.Sprintf("c%d", i)
		fmt.Fprintf(&script, "commit refs/heads/feat/x\ncommitter A <a@ejemplo.invalid> %d +0000\ndata %d\n%s\n", 1700000000+i, len(message), message)
		if i == 0 {
			script.WriteString("from refs/heads/feat/x^0\n")
		}
		script.WriteString("\n")
	}
	importer := exec.Command("git", "-C", worktree, "fast-import", "--quiet")
	importer.Stdin = strings.NewReader(script.String())
	if output, err := importer.CombinedOutput(); err != nil {
		t.Fatalf("git fast-import: %v\n%s", err, output)
	}

	state := wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
	if !state.RangeTooBig || state.Unpushed != branch.PublicationMaxUnpushed+1 || len(state.CodeCommits) != 0 {
		t.Errorf("estado = Unpushed %d, RangeTooBig %v, CodeCommits %v; se esperaba el rango demasiado grande", state.Unpushed, state.RangeTooBig, state.CodeCommits)
	}
}

// Sin `origin/<integración>` ni la rama de integración local, no se puede calcular el rango.
func TestPublicationStateWithoutAnyIntegrationReferenceIsRangeUnknown(t *testing.T) {
	t.Parallel()
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	runGit(t, repo, "remote", "add", "origin", t.TempDir())
	runGit(t, repo, "update-ref", "-d", "refs/heads/main")

	state := wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})
	if !state.RangeUnknown || state.RemoteMissing || state.Unpushed != 0 {
		t.Errorf("estado = %+v, se esperaba RangeUnknown", state)
	}
}

// RF-BRA-01: calcular el estado es una lectura. No lanza `push`, `commit`, `fetch`, `pull`,
// `ls-remote` ni `add`, y no cambia nada del repositorio.
func TestPublicationStateReadOnlyNoNetwork(t *testing.T) {
	repo, worktree, _ := stateWorktree(t)
	writeFile(t, changeFile(worktree, "proposal.md"), "propuesta\n")
	commit(t, worktree, "avance")
	state := repositoryState(t, repo)
	index := runGit(t, worktree, "status", "--porcelain=v1", "--untracked-files=all")
	seen := observe(t)

	wantPublicationState(t, branch.PublicationRequest{CWD: worktree, Change: "x"})

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación")
	}
	for _, invocation := range *seen {
		if len(invocation) > 1 && slices.Contains([]string{"push", "commit", "fetch", "pull", "ls-remote", "add"}, invocation[1]) {
			t.Errorf("el cálculo del estado lanzó %v", invocation)
		}
		if isWriteInvocation(invocation) {
			t.Errorf("el cálculo del estado escribió: %v", invocation)
		}
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("el repositorio cambió:\n%s\n---\n%s", state, got)
	}
	if got := runGit(t, worktree, "status", "--porcelain=v1", "--untracked-files=all"); got != index {
		t.Errorf("el árbol de trabajo cambió:\n%s\n---\n%s", index, got)
	}
}

// El destino y las negativas son los de `PublicationTarget`, que no cambian.
func TestPublicationStateRefusesLikePublicationTarget(t *testing.T) {
	t.Parallel()
	repo := handOffMonorepo(t, "core")
	_, refusal := publicationState(t, branch.PublicationRequest{CWD: repo, Change: "x"})
	wantPublicationRefusal(t, refusal, branch.CodeIntegrationBranch)
}

// remoteConfigFailure es el lector real de git con la lectura de la configuración del remoto rota.
type remoteConfigFailure struct{ *gitcli.Git }

func (remoteConfigFailure) RemoteConfigured(context.Context, string, string) (bool, error) {
	return false, errors.New("config ilegible")
}

// Si falla la lectura del remoto, el estado no se inventa: es el rechazo `git_failed`, que el
// status proyecta como `unavailable`.
func TestPublicationStateFailedRemoteReadIsGitFailed(t *testing.T) {
	t.Parallel()
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	real := gitcli.New()
	failing := &Engine{git: remoteConfigFailure{real}, writer: real, registry: registry.New(), prefs: hubPrefs{}, goos: runtime.GOOS}

	_, refusal := failing.PublicationState(context.Background(), branch.PublicationRequest{CWD: worktree, Change: "x"})

	wantPublicationRefusal(t, refusal, branch.CodeGitFailed)
}
