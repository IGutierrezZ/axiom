package sddpublish

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Matriz de pruebas de `Publisher.Run` (DD-09, REQ-27.5 a REQ-27.10, REQ-27.14) con git real, un
// remoto desnudo temporal local y el motor de ramas real, sin red. Cubre las filas aplicables de la
// matriz de amenazas de design §7: rutas que parecen documentación, selección del repositorio,
// estado del commit y estado del push.

const monorepoConfig = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: core\n    repositories:\n      - path: \".\"\n"

func publisher() *Publisher {
	return &Publisher{Git: NewGit(), Targets: engine.Default()}
}

func run(t *testing.T, cwd string, request Request) Result {
	t.Helper()
	request.CWD, request.Change = cwd, "x"
	return publisher().Run(context.Background(), request)
}

// newMonorepo crea un monorepo embebido con el cambio x confirmado en main, `origin` apuntando a un
// remoto desnudo que ya tiene main, y la rama del cambio `feat/x` desplegada en el checkout.
func newMonorepo(t *testing.T) (repo, bare string) {
	t.Helper()
	isolateGit(t)
	repo, bare = newRepo(t), newBare(t)
	writeFile(t, filepath.Join(repo, "axiom.yaml"), monorepoConfig)
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "diseño\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "cambio x")
	runGit(t, repo, "remote", "add", "origin", bare)
	runGit(t, repo, "push", "-q", "origin", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	return repo, bare
}

func editDesign(t *testing.T, repo, content string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), content)
}

func wantRefused(t *testing.T, result Result, code Code) *Refusal {
	t.Helper()
	if result.Status != StatusRefused || result.Refusal == nil || result.Refusal.Code != code {
		t.Fatalf("resultado = %s %+v, se esperaba la negativa %s", result.Status, result.Refusal, code)
	}
	if result.Refusal.Label == "" || result.Refusal.Message == "" || result.Refusal.Continuation == "" {
		t.Errorf("la negativa debe llevar etiqueta, mensaje y continuación: %+v", result.Refusal)
	}
	if result.ExitCode() != 1 {
		t.Errorf("una negativa sale con 1, sale con %d", result.ExitCode())
	}
	return result.Refusal
}

func wantPublished(t *testing.T, result Result) {
	t.Helper()
	if result.Status != StatusPublished || result.Refusal != nil || result.Push == nil {
		t.Fatalf("resultado = %s %+v push=%+v, se esperaba published", result.Status, result.Refusal, result.Push)
	}
	if !reflect.DeepEqual(result.Resume, []string{"git fetch origin", "axiom branch ensure --change x --json"}) {
		t.Errorf("resume = %v", result.Resume)
	}
}

func commitFiles(t *testing.T, repo, rev string) []string {
	t.Helper()
	files := strings.Fields(runGit(t, repo, "diff-tree", "-r", "--no-commit-id", "--name-only", "--root", rev))
	slices.Sort(files)
	return files
}

func pushes(seen [][]string) [][]string {
	var found [][]string
	for _, invocation := range seen {
		if len(invocation) > 1 && invocation[1] == "push" {
			found = append(found, invocation)
		}
	}
	return found
}

func installHook(t *testing.T, gitDir, name string) {
	t.Helper()
	hook := filepath.Join(gitDir, "hooks", name)
	writeFile(t, hook, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
}

// Matriz «Estado del commit»: el código preparado sigue preparado tras publicar, y no entra en el commit.
func TestPublishKeepsStagedCodeStaged(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	writeFile(t, filepath.Join(repo, "src", "a.go"), "package a\n")
	runGit(t, repo, "add", "src/a.go")

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if result.Commit == nil || !reflect.DeepEqual(commitFiles(t, repo, "HEAD"), []string{"openspec/changes/x/design.md"}) {
		t.Fatalf("el commit debe contener solo el documento, contiene %v", commitFiles(t, repo, "HEAD"))
	}
	if got := runGit(t, repo, "diff", "--cached", "--name-only"); got != "src/a.go" {
		t.Errorf("el código preparado sigue preparado, el índice tiene %q", got)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x"); got != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
	}
	if want := "docs(sdd): publicar los documentos de x"; result.Commit.Message != want || runGit(t, repo, "log", "-1", "--format=%s") != want {
		t.Errorf("mensaje = %q, se esperaba %q", result.Commit.Message, want)
	}
}

// Matriz «Estado del commit»: lo que no está preparado nunca se confirma, ni lo modificado ni lo
// que no tiene seguimiento (nunca `commit -a`).
func TestPublishNeverCommitsUnstagedCode(t *testing.T) {
	repo, _ := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	writeFile(t, filepath.Join(repo, "README.md"), "modificado sin preparar\n")
	writeFile(t, filepath.Join(repo, "src", "b.go"), "package b\n")

	wantPublished(t, run(t, repo, Request{}))

	if !reflect.DeepEqual(commitFiles(t, repo, "HEAD"), []string{"openspec/changes/x/design.md"}) {
		t.Errorf("el commit contiene %v", commitFiles(t, repo, "HEAD"))
	}
	if got := runGit(t, repo, "status", "--porcelain", "--untracked-files=all"); !strings.Contains(got, " M README.md") || !strings.Contains(got, "?? src/b.go") {
		t.Errorf("el código sin preparar debe seguir igual:\n%s", got)
	}
}

// Matriz «Estado del commit»: sin cambios de documentación no hay commit vacío ni push.
func TestPublishNoDocChangesMakesNoCommit(t *testing.T) {
	repo, bare := newMonorepo(t)
	head := runGit(t, repo, "rev-parse", "HEAD")
	seen := observe(t)

	result := run(t, repo, Request{})

	if result.Status != StatusUpToDate || result.Commit != nil || result.Push != nil || result.Refusal != nil || result.ExitCode() != 0 {
		t.Fatalf("resultado = %+v, se esperaba up_to_date sin commit ni push", result)
	}
	if runGit(t, repo, "rev-parse", "HEAD") != head {
		t.Error("no debe crear ningún commit")
	}
	if len(pushes(*seen)) != 0 {
		t.Errorf("no debe subir nada: %v", pushes(*seen))
	}
	if out, err := tryGit(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Errorf("el remoto no debía recibir la rama, tiene %s", out)
	}
}

// Matriz «Estado del commit»: `.gates.lock`, `.axiom-instance` y los temporales de la raíz de la
// carpeta del cambio nunca se versionan.
func TestPublishExcludesLocalStateFiles(t *testing.T) {
	repo, _ := newMonorepo(t)
	folder := filepath.Join(repo, "openspec", "changes", "x")
	writeFile(t, filepath.Join(folder, "gates.yaml"), "schema: axiom.sdd-gate-ledger/v1\n")
	writeFile(t, filepath.Join(folder, ".gates.lock"), "")
	writeFile(t, filepath.Join(folder, ".axiom-instance"), "id\n")
	writeFile(t, filepath.Join(folder, ".gates-123.yaml"), "temporal\n")

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if want := []string{"openspec/changes/x/gates.yaml"}; !reflect.DeepEqual(commitFiles(t, repo, "HEAD"), want) || !reflect.DeepEqual(result.Commit.Paths, want) {
		t.Errorf("el commit contiene %v (resultado %v), se esperaba %v", commitFiles(t, repo, "HEAD"), result.Commit.Paths, want)
	}
	status := runGit(t, repo, "status", "--porcelain", "--untracked-files=all")
	for _, local := range []string{".gates.lock", ".axiom-instance", ".gates-123.yaml"} {
		if !strings.Contains(status, "?? openspec/changes/x/"+local) {
			t.Errorf("%s debe seguir sin seguimiento:\n%s", local, status)
		}
	}
}

// REQ-27.14: el archivado publica la carpeta archivada, los borrados de `changes/<x>/`, las specs y el
// índice, con el mensaje de archivado y solo con `--archive` en la rama de archivado.
func TestPublishCommitsArchiveDeletions(t *testing.T) {
	repo, bare := newMonorepo(t)
	writeFile(t, filepath.Join(repo, "openspec", "specs", "a", "spec.md"), "spec\n")
	writeFile(t, filepath.Join(repo, "openspec", "INDEX.md"), "índice\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "specs")
	runGit(t, repo, "push", "-q", "origin", "main")
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	changes := filepath.Join(repo, "openspec", "changes")
	if err := os.Rename(filepath.Join(changes, "x"), filepath.Join(changes, "archive-tmp")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(changes, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(changes, "archive-tmp"), filepath.Join(changes, "archive", "2026-11-09-x")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "openspec", "specs", "a", "spec.md"), "spec actualizada\n")
	writeFile(t, filepath.Join(repo, "openspec", "INDEX.md"), "índice actualizado\n")
	writeFile(t, filepath.Join(repo, "src", "otro.go"), "package otro\n")
	// Un cambio ajeno archivado no es de este cambio.
	writeFile(t, filepath.Join(changes, "archive", "2026-11-09-foo-x", "proposal.md"), "ajeno\n")

	t.Run("sin --archive la rama de archivado es wrong_branch", func(t *testing.T) {
		refusal := wantRefused(t, run(t, repo, Request{}), "wrong_branch")
		if !strings.Contains(refusal.Continuation, "--archive") {
			t.Errorf("la continuación %q debe nombrar `--archive`", refusal.Continuation)
		}
	})

	result := run(t, repo, Request{Archive: true})

	wantPublished(t, result)
	if result.Context != ContextArchive {
		t.Errorf("context = %q, se esperaba archive", result.Context)
	}
	want := []string{
		"openspec/INDEX.md",
		"openspec/changes/archive/2026-11-09-x/design.md",
		"openspec/changes/archive/2026-11-09-x/proposal.md",
		"openspec/changes/x/design.md",
		"openspec/changes/x/proposal.md",
		"openspec/specs/a/spec.md",
	}
	if got := commitFiles(t, repo, "HEAD"); !reflect.DeepEqual(got, want) {
		t.Errorf("el commit de archivado contiene:\n got %v\nwant %v", got, want)
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != "docs(sdd): publicar el archivado de x" {
		t.Errorf("mensaje = %q", got)
	}
	status := runGit(t, repo, "status", "--porcelain", "--untracked-files=all")
	if !strings.Contains(status, "?? openspec/changes/archive/2026-11-09-foo-x/proposal.md") || !strings.Contains(status, "?? src/otro.go") {
		t.Errorf("el archivado de otro cambio y el código quedan sin seguimiento:\n%s", status)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x-archive"); got != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s", got)
	}
}

// El cambio ya movido a `changes/archive/` y sin carpeta archivada en disco se reconoce por los
// borrados que git aún ve; sin nada sucio ni archivado es `change_not_found`.
func TestPublishArchiveFindsTheChangeThroughItsDeletions(t *testing.T) {
	repo, _ := newMonorepo(t)
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	if err := os.RemoveAll(filepath.Join(repo, "openspec", "changes", "x")); err != nil {
		t.Fatal(err)
	}

	result := run(t, repo, Request{Archive: true})

	wantPublished(t, result)
	if got := commitFiles(t, repo, "HEAD"); !reflect.DeepEqual(got, []string{"openspec/changes/x/design.md", "openspec/changes/x/proposal.md"}) {
		t.Errorf("el commit contiene %v", got)
	}
	// Ahora no queda nada: ni carpeta ni borrados.
	wantRefused(t, run(t, repo, Request{Archive: true}), CodeChangeNotFound)
}

// Matriz «Estado del push»: el primer push crea la rama remota con `--set-upstream`.
func TestPublishFirstPushCreatesRemoteBranch(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	seen := observe(t)

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if !result.Push.Created || result.Push.Refspec != "refs/heads/feat/x:refs/heads/feat/x" {
		t.Errorf("push = %+v, se esperaba la rama creada con la refspec explícita", result.Push)
	}
	if head := runGit(t, repo, "rev-parse", "HEAD"); result.Push.SHA != head || runGit(t, bare, "rev-parse", "refs/heads/feat/x") != head {
		t.Errorf("push.sha = %s, remoto = %s, se esperaba %s", result.Push.SHA, runGit(t, bare, "rev-parse", "refs/heads/feat/x"), head)
	}
	if got := runGit(t, repo, "config", "branch.feat/x.remote"); got != "origin" {
		t.Errorf("el primer push configura el seguimiento, remote = %q", got)
	}
	found := pushes(*seen)
	if len(found) != 1 || !slices.Contains(found[0], "--set-upstream") {
		t.Errorf("pushes = %v, se esperaba uno con --set-upstream", found)
	}
	if result.Repository == nil || result.Repository.Name == "" || result.Branch != "feat/x" || result.InPlace {
		t.Errorf("repositorio/rama = %+v %q inPlace=%v", result.Repository, result.Branch, result.InPlace)
	}
	if !reflect.DeepEqual(result.UnpublishedCommits, []string{result.Commit.SHA}) {
		t.Errorf("unpublishedCommits = %v, se esperaba solo el commit de documentación", result.UnpublishedCommits)
	}
}

// Matriz «Estado del push»: con la rama ya publicada, el siguiente push es un avance rápido y no crea la rama.
func TestPublishTrackedBranchFastForward(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "primera\n")
	wantPublished(t, run(t, repo, Request{}))
	editDesign(t, repo, "segunda\n")

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if result.Push.Created {
		t.Error("la rama remota ya existía: created debe ser falso")
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x"); got != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
	}
}

// Matriz «Estado del push»: si el remoto avanzó, la negativa es tipada, el commit local se conserva y
// no hay reintento, ni `fetch`, ni fuerza.
func TestPublishNonFastForwardRefusedWithoutRetry(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "primera\n")
	wantPublished(t, run(t, repo, Request{}))
	other := t.TempDir()
	runGit(t, other, "clone", "-q", bare, "clon")
	clone := filepath.Join(other, "clon")
	runGit(t, clone, "switch", "-q", "feat/x")
	writeFile(t, filepath.Join(clone, "ajeno.txt"), "ajeno\n")
	runGit(t, clone, "add", "-A")
	runGit(t, clone, "commit", "-q", "-m", "ajeno")
	runGit(t, clone, "push", "-q", "origin", "feat/x")
	remoteHead := runGit(t, bare, "rev-parse", "refs/heads/feat/x")
	editDesign(t, repo, "segunda\n")
	seen := observe(t)

	result := run(t, repo, Request{})

	refusal := wantRefused(t, result, CodePushNonFastForward)
	if strings.Contains(refusal.Continuation, "--force") {
		t.Errorf("la continuación no debe proponer forzar: %q", refusal.Continuation)
	}
	if result.Commit == nil || runGit(t, repo, "log", "-1", "--format=%s") != result.Commit.Message {
		t.Errorf("el commit local se conserva y se informa: %+v", result.Commit)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x"); got != remoteHead {
		t.Error("el remoto no debe cambiar")
	}
	if len(pushes(*seen)) != 1 {
		t.Errorf("sin reintento: se esperaba un solo push, hay %v", pushes(*seen))
	}
	for _, invocation := range *seen {
		if len(invocation) > 1 && slices.Contains([]string{"fetch", "pull", "ls-remote"}, invocation[1]) {
			t.Errorf("no debe traer nada del remoto: %v", invocation)
		}
	}
}

// S9 y matriz «Estado del push»: la rama de integración de un repositorio de código no se publica, y la
// negativa no crea ningún commit.
func TestPublishRefusesIntegrationBranchInCodeRepo(t *testing.T) {
	repo, _ := newMonorepo(t)
	runGit(t, repo, "switch", "-q", "main")
	editDesign(t, repo, "diseño nuevo\n")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := run(t, repo, Request{})

	wantRefused(t, result, "integration_branch")
	if result.Commit != nil || runGit(t, repo, "rev-parse", "HEAD") != head || runGit(t, repo, "diff", "--cached", "--name-only") != "" {
		t.Error("una negativa de destino no escribe nada")
	}
}

// S9: el repositorio de specs de un multirepo se trabaja in situ y sube su rama de integración, sin
// configurar el seguimiento.
func TestPublishMultirepoSpecsPushesIntegration(t *testing.T) {
	isolateGit(t)
	master := t.TempDir()
	specs, backend := filepath.Join(master, "repo-specs"), filepath.Join(master, "repo-backend")
	for _, dir := range []string{specs, backend} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, dir, "init", "-q", "-b", "main")
		writeFile(t, filepath.Join(dir, "README.md"), "inicial\n")
		runGit(t, dir, "add", "-A")
		runGit(t, dir, "commit", "-q", "-m", "inicial")
	}
	writeFile(t, filepath.Join(master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: backend\n    repositories:\n      - path: \"repo-backend\"\n")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
	runGit(t, specs, "add", "-A")
	runGit(t, specs, "commit", "-q", "-m", "cambio x")
	bare := newBare(t)
	runGit(t, specs, "remote", "add", "origin", bare)
	runGit(t, specs, "push", "-q", "origin", "main")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "diseño\n")
	seen := observe(t)

	result := run(t, master, Request{})

	wantPublished(t, result)
	if !result.InPlace || result.Branch != "main" {
		t.Errorf("rama = %q inPlace=%v, se esperaba main in situ", result.Branch, result.InPlace)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/main"); got != runGit(t, specs, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD de specs", got)
	}
	if found := pushes(*seen); len(found) != 1 || slices.Contains(found[0], "--set-upstream") {
		t.Errorf("pushes = %v, se esperaba uno sin --set-upstream", found)
	}
	if got := runGit(t, backend, "log", "--oneline"); strings.Contains(got, "publicar") {
		t.Error("el repositorio de código no se toca")
	}
}

// REQ-27.7 regla 4: `--branch` atado a otra rama es `wrong_branch` y no escribe nada.
func TestPublishBranchFlagMismatchIsWrongBranch(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := run(t, repo, Request{Branch: "feat/otra"})

	wantRefused(t, result, "wrong_branch")
	if runGit(t, repo, "rev-parse", "HEAD") != head || runGit(t, repo, "status", "--porcelain") == "" {
		t.Error("la negativa no debe confirmar nada")
	}
	if _, err := tryGit(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Error("la negativa no debe subir nada")
	}
	// Con la rama autorizada, publica.
	wantPublished(t, run(t, repo, Request{Branch: "feat/x"}))
}

// Matriz «Estado del push» (observador): ninguna publicación hace `fetch`, `pull` ni `ls-remote`, ni
// usa `+`, `--force*` o `--no-verify`, ni `-a` en el commit.
func TestPublishNeverRunsFetchOrForce(t *testing.T) {
	repo, _ := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	seen := observe(t)

	wantPublished(t, run(t, repo, Request{}))

	for _, invocation := range *seen {
		args := invocation[1:]
		if len(args) == 0 {
			continue
		}
		if slices.Contains([]string{"fetch", "pull", "ls-remote", "clone"}, args[0]) {
			t.Errorf("no debe traer nada del remoto: %v", invocation)
		}
		for _, arg := range args {
			if arg == "--no-verify" || arg == "-n" || arg == "-a" || arg == "--amend" || arg == "-f" || strings.HasPrefix(arg, "--force") || strings.HasPrefix(arg, "+") || strings.Contains(arg, ":+") {
				t.Errorf("argumento prohibido %q en %v", arg, invocation)
			}
		}
		if args[0] == "push" && args[len(args)-1] != "refs/heads/feat/x:refs/heads/feat/x" {
			t.Errorf("el push debe llevar la refspec explícita, lleva %v", args)
		}
	}
	if len(pushes(*seen)) != 1 {
		t.Errorf("se esperaba un único push, hay %v", pushes(*seen))
	}
}

// Matriz «Rutas que parecen documentación»: un commit de código sin subir que toca `README.sh` en la
// raíz es código y nunca se sube; el mismo nombre bajo la carpeta del cambio es documentación.
func TestPublishRefusesUnpushedCommitTouchingReadmeSh(t *testing.T) {
	repo, bare := newMonorepo(t)
	writeFile(t, filepath.Join(repo, "README.sh"), "#!/bin/sh\necho código\n")
	writeFile(t, filepath.Join(repo, "requirements.txt"), "x==1\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "feat: código")
	codeCommit := runGit(t, repo, "rev-parse", "--short=7", "HEAD")
	editDesign(t, repo, "diseño nuevo\n")

	result := run(t, repo, Request{})

	refusal := wantRefused(t, result, CodeCodeCommitsUnpublished)
	if result.Commit == nil {
		t.Fatal("el commit de documentación se conserva y se informa")
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != result.Commit.Message {
		t.Errorf("el commit local debe existir: %q", got)
	}
	if !reflect.DeepEqual(refusal.Details["commits"], []string{codeCommit}) {
		t.Errorf("details.commits = %v, se esperaba [%s]", refusal.Details["commits"], codeCommit)
	}
	paths, _ := refusal.Details["paths"].([]string)
	if !slices.Contains(paths, "README.sh") || !slices.Contains(paths, "requirements.txt") {
		t.Errorf("details.paths = %v, debe nombrar README.sh y requirements.txt", paths)
	}
	if !strings.Contains(refusal.Continuation, "push origin feat/x:feat/x") || !strings.Contains(refusal.Continuation, "git -C") {
		t.Errorf("la continuación debe ser el push de la persona: %q", refusal.Continuation)
	}
	if _, err := tryGit(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Error("no se sube nada cuando hay código sin subir")
	}
}

func TestPublishTreatsReadmeShUnderTheChangeFolderAsDocumentation(t *testing.T) {
	repo, _ := newMonorepo(t)
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "README.sh"), "#!/bin/sh\necho documento\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "docs: ya confirmado")
	editDesign(t, repo, "diseño nuevo\n")

	wantPublished(t, run(t, repo, Request{}))
}

// Lo que ya está en la integración publicada no es trabajo del cambio: una fusión de `main` en la rama
// no da una negativa falsa por los commits de código de main.
func TestPublishIgnoresPublishedIntegrationCommits(t *testing.T) {
	repo, bare := newMonorepo(t)
	editDesign(t, repo, "primera\n")
	wantPublished(t, run(t, repo, Request{}))
	other := t.TempDir()
	runGit(t, other, "clone", "-q", bare, "clon")
	clone := filepath.Join(other, "clon")
	writeFile(t, filepath.Join(clone, "src", "main.go"), "package main\n")
	runGit(t, clone, "add", "-A")
	runGit(t, clone, "commit", "-q", "-m", "feat: código de main")
	runGit(t, clone, "push", "-q", "origin", "main")
	runGit(t, repo, "fetch", "-q", "origin")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusiona main", "origin/main")
	editDesign(t, repo, "segunda\n")

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x"); got != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
	}
}

// Matriz «Selección del repositorio git»: la publicación actúa en la raíz git del destino, aunque se
// pida desde un worktree enlazado, y no toca el checkout principal.
func TestPublishRunsInTargetRootFromLinkedWorktree(t *testing.T) {
	repo, bare := newMonorepo(t)
	runGit(t, repo, "switch", "-q", "main")
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, repo, "worktree", "add", "-q", linked, "feat/x")
	editDesign(t, linked, "diseño nuevo en el worktree\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "nota.md"), "sucio en el principal\n")
	mainHead := runGit(t, repo, "rev-parse", "HEAD")
	seen := observe(t)

	result := run(t, linked, Request{})

	wantPublished(t, result)
	if !pathidentity.SameDirectory(result.Repository.GitRoot, linked) {
		t.Errorf("gitRoot = %s, se esperaba el worktree %s", result.Repository.GitRoot, linked)
	}
	for _, invocation := range *seen {
		if len(invocation) > 1 && slices.Contains([]string{"add", "commit", "push"}, invocation[1]) && !pathidentity.SameDirectory(invocation[0], linked) {
			t.Errorf("%v se ejecutó fuera del worktree %s", invocation, linked)
		}
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/feat/x"); got != runGit(t, linked, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s", got)
	}
	if runGit(t, repo, "rev-parse", "HEAD") != mainHead || !strings.Contains(runGit(t, repo, "status", "--porcelain", "--untracked-files=all"), "?? openspec/changes/x/nota.md") {
		t.Error("el checkout principal no debe tocarse")
	}
}

// REQ-27.10: con un almacén sin ficheros no hay documentos que publicar.
func TestPublishRefusesStoresWithoutFiles(t *testing.T) {
	for _, store := range []string{"engram", "none"} {
		t.Run(store, func(t *testing.T) {
			repo, _ := newMonorepo(t)
			writeFile(t, filepath.Join(repo, "openspec", "config.yaml"), "artifact_store: "+store+"\n")
			runGit(t, repo, "add", "-A")
			runGit(t, repo, "commit", "-q", "-m", "almacén")
			editDesign(t, repo, "diseño nuevo\n")
			head := runGit(t, repo, "rev-parse", "HEAD")

			wantRefused(t, run(t, repo, Request{}), CodeStoreWithoutFiles)

			if runGit(t, repo, "rev-parse", "HEAD") != head {
				t.Error("no debe confirmar nada")
			}
		})
	}
	t.Run("hybrid sí se publica", func(t *testing.T) {
		repo, _ := newMonorepo(t)
		writeFile(t, filepath.Join(repo, "openspec", "config.yaml"), "artifact_store: hybrid\n")
		editDesign(t, repo, "diseño nuevo\n")
		wantPublished(t, run(t, repo, Request{}))
	})
}

func TestPublishChangeNotFound(t *testing.T) {
	repo, _ := newMonorepo(t)
	result := publisher().Run(context.Background(), Request{CWD: repo, Change: "no-existe"})
	wantRefused(t, result, CodeChangeNotFound)
}

// Sin `origin` no se crea ningún commit: la negativa va antes de escribir.
func TestPublishRemoteMissingMakesNoCommit(t *testing.T) {
	repo, _ := newMonorepo(t)
	runGit(t, repo, "remote", "remove", "origin")
	editDesign(t, repo, "diseño nuevo\n")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := run(t, repo, Request{})

	refusal := wantRefused(t, result, CodeRemoteMissing)
	if !strings.Contains(refusal.Continuation, "remote add origin") {
		t.Errorf("la continuación %q debe explicar cómo añadir origin", refusal.Continuation)
	}
	if result.Commit != nil || runGit(t, repo, "rev-parse", "HEAD") != head || runGit(t, repo, "diff", "--cached", "--name-only") != "" {
		t.Error("sin origin no se escribe nada, ni siquiera el índice")
	}
}

func TestPublishUnpublishedRangeUnknown(t *testing.T) {
	isolateGit(t)
	repo, bare := newRepo(t), newBare(t)
	writeFile(t, filepath.Join(repo, "axiom.yaml"), monorepoConfig)
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "cambio x")
	runGit(t, repo, "remote", "add", "origin", bare)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	runGit(t, repo, "branch", "-q", "-D", "main")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "diseño\n")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := run(t, repo, Request{})

	wantRefused(t, result, CodeUnpublishedRangeUnknown)
	if runGit(t, repo, "rev-parse", "HEAD") != head {
		t.Error("sin referencia para calcular el rango no se confirma nada")
	}
}

func TestPublishIdentityMissingWritesNothing(t *testing.T) {
	repo, _ := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	for _, name := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	runGit(t, repo, "config", "user.useConfigOnly", "true")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := run(t, repo, Request{})

	wantRefused(t, result, CodeIdentityMissing)
	if runGit(t, repo, "rev-parse", "HEAD") != head || runGit(t, repo, "diff", "--cached", "--name-only") != "" {
		t.Error("sin identidad no se escribe nada")
	}
}

func TestPublishDetachedHeadWritesNothing(t *testing.T) {
	repo, _ := newMonorepo(t)
	runGit(t, repo, "switch", "-q", "--detach")
	editDesign(t, repo, "diseño nuevo\n")
	head := runGit(t, repo, "rev-parse", "HEAD")

	wantRefused(t, run(t, repo, Request{}), "detached_head")

	if runGit(t, repo, "rev-parse", "HEAD") != head {
		t.Error("con HEAD desacoplado no se escribe nada")
	}
}

// Los hooks nunca se saltan: un `pre-commit` que rechaza es `commit_hook_rejected`, un `pre-push`
// local y un `pre-receive` remoto son `push_hook_rejected`, y tras el commit el local se conserva.
func TestPublishHooksRejectionsAreTyped(t *testing.T) {
	t.Run("pre-commit", func(t *testing.T) {
		repo, _ := newMonorepo(t)
		installHook(t, filepath.Join(repo, ".git"), "pre-commit")
		editDesign(t, repo, "diseño nuevo\n")
		head := runGit(t, repo, "rev-parse", "HEAD")

		result := run(t, repo, Request{})

		wantRefused(t, result, CodeCommitHookRejected)
		if result.Commit != nil || runGit(t, repo, "rev-parse", "HEAD") != head {
			t.Error("el hook rechazó el commit: no hay commit")
		}
	})
	t.Run("pre-push local", func(t *testing.T) {
		repo, bare := newMonorepo(t)
		installHook(t, filepath.Join(repo, ".git"), "pre-push")
		editDesign(t, repo, "diseño nuevo\n")

		result := run(t, repo, Request{})

		wantRefused(t, result, CodePushHookRejected)
		if result.Commit == nil || runGit(t, repo, "log", "-1", "--format=%s") != result.Commit.Message {
			t.Error("el commit local se conserva y se informa")
		}
		if _, err := tryGit(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
			t.Error("el hook impidió el push")
		}
	})
	t.Run("pre-receive remoto", func(t *testing.T) {
		repo, bare := newMonorepo(t)
		installHook(t, bare, "pre-receive")
		editDesign(t, repo, "diseño nuevo\n")

		result := run(t, repo, Request{})

		wantRefused(t, result, CodePushHookRejected)
		if result.Commit == nil {
			t.Error("el commit local se conserva y se informa")
		}
	})
}

func TestPublishRemoteWithNonexistentURLIsUnavailable(t *testing.T) {
	repo, _ := newMonorepo(t)
	runGit(t, repo, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
	editDesign(t, repo, "diseño nuevo\n")

	result := run(t, repo, Request{})

	wantRefused(t, result, CodeRemoteUnavailable)
	if result.Commit == nil {
		t.Error("el commit local se conserva y se informa")
	}
}

func TestPublishUnpublishedRangeTooLarge(t *testing.T) {
	repo, bare := newMonorepo(t)
	previous := maxUnpublishedCommits
	maxUnpublishedCommits = 2
	t.Cleanup(func() { maxUnpublishedCommits = previous })
	for _, content := range []string{"uno\n", "dos\n"} {
		editDesign(t, repo, content)
		runGit(t, repo, "add", "-A")
		runGit(t, repo, "commit", "-q", "-m", "docs: "+strings.TrimSpace(content))
	}
	editDesign(t, repo, "tres\n")

	result := run(t, repo, Request{})

	wantRefused(t, result, CodeUnpublishedRangeTooBig)
	if result.Commit == nil {
		t.Error("el commit de documentación se conserva y se informa")
	}
	if _, err := tryGit(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Error("no se sube nada")
	}
}

// Idempotencia: publicar dos veces seguidas la segunda es `up_to_date`.
func TestPublishTwiceIsUpToDate(t *testing.T) {
	repo, _ := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	wantPublished(t, run(t, repo, Request{}))

	second := run(t, repo, Request{})

	if second.Status != StatusUpToDate || second.Commit != nil || second.Push != nil || second.ExitCode() != 0 {
		t.Fatalf("segunda publicación = %+v, se esperaba up_to_date", second)
	}
	if !reflect.DeepEqual(second.Resume, []string{"git fetch origin", "axiom branch ensure --change x --json"}) {
		t.Errorf("resume = %v", second.Resume)
	}
}

// Las rutas de `baseline` y `technical_context.maintained` declaradas en axiom.yaml se publican, y las de
// `reference` nunca, aunque estén sucias.
func TestPublishDeclaredBaselineAndMaintainedPathsAreDocumentation(t *testing.T) {
	repo, _ := newMonorepo(t)
	config := monorepoConfig +
		"baseline:\n  documents:\n    - domain: arquitectura\n      path: docs/arquitectura.md\n" +
		"technical_context:\n  maintained:\n    entry: docs/contexto/README.md\n    paths:\n      - docs/contexto\n  reference:\n    paths:\n      - docs/ref\n"
	// La configuración llega por la integración publicada: no es trabajo de la rama del cambio.
	runGit(t, repo, "switch", "-q", "main")
	writeFile(t, filepath.Join(repo, "axiom.yaml"), config)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "configuración")
	runGit(t, repo, "push", "-q", "origin", "main")
	runGit(t, repo, "switch", "-q", "feat/x")
	runGit(t, repo, "merge", "-q", "--ff-only", "main")
	writeFile(t, filepath.Join(repo, "docs", "arquitectura.md"), "arquitectura\n")
	writeFile(t, filepath.Join(repo, "docs", "contexto", "a.md"), "contexto\n")
	writeFile(t, filepath.Join(repo, "docs", "ref", "r.md"), "referencia\n")
	writeFile(t, filepath.Join(repo, "docs", "otro.md"), "otro\n")

	result := run(t, repo, Request{})

	wantPublished(t, result)
	if want := []string{"docs/arquitectura.md", "docs/contexto/a.md"}; !reflect.DeepEqual(commitFiles(t, repo, "HEAD"), want) {
		t.Errorf("el commit contiene %v, se esperaba %v", commitFiles(t, repo, "HEAD"), want)
	}
	status := runGit(t, repo, "status", "--porcelain", "--untracked-files=all")
	if !strings.Contains(status, "?? docs/ref/r.md") || !strings.Contains(status, "?? docs/otro.md") {
		t.Errorf("reference y lo no declarado quedan sin seguimiento:\n%s", status)
	}
}
