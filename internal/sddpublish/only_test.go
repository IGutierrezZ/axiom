package sddpublish

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
)

// Pruebas del modo acotado (`Only`) y de la reproducción única (`Replay`): git real, remotos
// desnudos locales y el motor de ramas real, sin red.

const marksRel = "openspec/changes/x/pruebas-estado.yaml"

const testDigest = "2fe1cfd9a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e"

// marksFile devuelve un fichero de marcas con una entrada por caso dado.
func marksFile(t *testing.T, base []byte, cases ...string) []byte {
	t.Helper()
	data := base
	for i, c := range cases {
		var err error
		data, err = marks.Append(data, marks.Entry{
			Case: c, Environment: "pre", Result: marks.ResultOK, By: "Ana <ana@example.org>",
			At: "2026-11-26T10:00:0" + string(rune('0'+i)) + ".000Z", PlanDigest: testDigest,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return data
}

// newMultirepoSpecs crea un multirepo con el cambio x confirmado y publicado en la integración
// (main) del repositorio de specs, que se trabaja in situ.
func newMultirepoSpecs(t *testing.T) (master, specs, bare string) {
	t.Helper()
	isolateGit(t)
	master = t.TempDir()
	specs, backend := filepath.Join(master, "repo-specs"), filepath.Join(master, "repo-backend")
	for _, dir := range []string{specs, backend} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, dir, "init", "-q", "-b", "main")
		runGit(t, dir, "config", "core.autocrlf", "false")
		writeFile(t, filepath.Join(dir, "README.md"), "inicial\n")
		runGit(t, dir, "add", "-A")
		runGit(t, dir, "commit", "-q", "-m", "inicial")
	}
	writeFile(t, filepath.Join(master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: backend\n    repositories:\n      - path: \"repo-backend\"\n")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "diseño\n")
	runGit(t, specs, "add", "-A")
	runGit(t, specs, "commit", "-q", "-m", "cambio x")
	bare = newBare(t)
	runGit(t, specs, "remote", "add", "origin", filepath.ToSlash(bare))
	runGit(t, specs, "push", "-q", "origin", "main")
	return master, specs, bare
}

func onlyRequest() Request { return Request{Only: []string{"pruebas-estado.yaml"}} }

func writeMarks(t *testing.T, specs string, content []byte) {
	t.Helper()
	writeFile(t, filepath.Join(specs, filepath.FromSlash(marksRel)), string(content))
}

func gitInvocations(seen [][]string, names ...string) [][]string {
	var found [][]string
	for _, invocation := range seen {
		if len(invocation) > 1 && slices.Contains(names, invocation[1]) {
			found = append(found, invocation)
		}
	}
	return found
}

// pushFromClone sube a la rama main del remoto un commit de otra persona que escribe marksRel.
func pushFromClone(t *testing.T, bare string, content []byte) {
	t.Helper()
	other := t.TempDir()
	runGit(t, other, "clone", "-q", filepath.ToSlash(bare), "clon")
	clone := filepath.Join(other, "clon")
	writeFile(t, filepath.Join(clone, filepath.FromSlash(marksRel)), string(content))
	runGit(t, clone, "add", "-A")
	runGit(t, clone, "commit", "-q", "-m", "marcas ajenas")
	runGit(t, clone, "push", "-q", "origin", "main")
}

// Matriz «Rutas tipo documentación»: ninguna ruta distinta de pruebas-estado.yaml llega a git.
func TestOnlyRejectsEveryPathClassWithoutGit(t *testing.T) {
	master, _, _ := newMultirepoSpecs(t)
	for _, name := range []string{"design.md", "internal/x.go", "../pruebas-estado.yaml", "/etc/passwd", `C:\x\pruebas-estado.yaml`, "sub/pruebas-estado.yaml", ""} {
		seen := observe(t)
		result := run(t, master, Request{Only: []string{name}})
		wantRefused(t, result, CodeOnlyPathRejected)
		if len(*seen) != 0 {
			t.Errorf("%q: la negativa no debe ejecutar git, ejecutó %v", name, *seen)
		}
		// Una ruta permitida junto a una prohibida también se rechaza entera.
		result = run(t, master, Request{Only: []string{"pruebas-estado.yaml", name}})
		wantRefused(t, result, CodeOnlyPathRejected)
	}
}

func TestOnlyReplayWithoutOnlyAndArchiveWithOnlyAreRejected(t *testing.T) {
	master, _, _ := newMultirepoSpecs(t)
	seen := observe(t)
	merge := func(l, r []byte) ([]byte, error) { return r, nil }
	wantRefused(t, run(t, master, Request{Replay: merge}), CodeOnlyPathRejected)
	wantRefused(t, run(t, master, Request{Only: []string{"pruebas-estado.yaml"}, Archive: true}), CodeOnlyPathRejected)
	if len(*seen) != 0 {
		t.Errorf("sin git: %v", *seen)
	}
}

func TestOnlyRequiresInPlaceDestination(t *testing.T) {
	t.Parallel()
	repo, _ := newMonorepo(t)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(marksRel)), "x\n")
	head := runGit(t, repo, "rev-parse", "HEAD")
	result := run(t, repo, onlyRequest())
	wantRefused(t, result, CodeOnlyRequiresInPlace)
	if runGit(t, repo, "rev-parse", "HEAD") != head || result.Commit != nil {
		t.Error("la negativa no crea commits")
	}
}

// Matriz «Estado del commit»: solo el fichero de marcas entra en el commit; otro documento
// preparado y otro sucio sin preparar quedan intactos y se avisan.
func TestOnlyCommitsOnlyTheMarksFileAndWarnsAboutOtherDocs(t *testing.T) {
	master, specs, bare := newMultirepoSpecs(t)
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "diseño preparado\n")
	runGit(t, specs, "add", "openspec/changes/x/design.md")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "proposal.md"), "propuesta sucia\n")
	writeFile(t, filepath.Join(specs, "src", "a.go"), "package a\n")

	result := run(t, master, onlyRequest())

	wantPublished(t, result)
	if !result.InPlace || result.Branch != "main" {
		t.Errorf("destino = %q inPlace=%v", result.Branch, result.InPlace)
	}
	if got := commitFiles(t, specs, "HEAD"); !slices.Equal(got, []string{marksRel}) {
		t.Fatalf("el commit contiene %v", got)
	}
	if got := runGit(t, specs, "diff", "--cached", "--name-only"); got != "openspec/changes/x/design.md" {
		t.Errorf("el documento preparado sigue preparado, el índice tiene %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(specs, "openspec", "changes", "x", "proposal.md")); string(got) != "propuesta sucia\n" {
		t.Errorf("el documento sucio debe quedar intacto: %q", got)
	}
	want := []string{"openspec/changes/x/design.md", "openspec/changes/x/proposal.md"}
	if !slices.Equal(result.UncommittedDocs, want) {
		t.Errorf("UncommittedDocs = %v, se esperaba %v", result.UncommittedDocs, want)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/main"); got != runGit(t, specs, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s", got)
	}
	if want := "docs(sdd): publicar las marcas de pruebas de x"; result.Commit == nil || result.Commit.Message != want {
		t.Errorf("commit = %+v", result.Commit)
	}
}

func TestOnlyWrongBranchIdentityAndRemoteMissingWriteNothing(t *testing.T) {
	master, specs, _ := newMultirepoSpecs(t)
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	head := runGit(t, specs, "rev-parse", "HEAD")

	req := onlyRequest()
	req.Branch = "otra"
	wantRefused(t, run(t, master, req), "wrong_branch")

	runGit(t, specs, "remote", "rename", "origin", "otro")
	wantRefused(t, run(t, master, onlyRequest()), CodeRemoteMissing)
	runGit(t, specs, "remote", "rename", "otro", "origin")

	for _, name := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	runGit(t, specs, "config", "user.useConfigOnly", "true")
	wantRefused(t, run(t, master, onlyRequest()), CodeIdentityMissing)

	if runGit(t, specs, "rev-parse", "HEAD") != head {
		t.Error("ninguna negativa debe crear commits")
	}
}

// Nada sucio y un commit de marcas sin subir: el push se hace sin crear un commit nuevo.
func TestOnlyPushesAnUnpushedMarksCommitWithoutNewCommit(t *testing.T) {
	master, specs, bare := newMultirepoSpecs(t)
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	runGit(t, specs, "add", marksRel)
	runGit(t, specs, "commit", "-q", "-m", "marcas locales")
	head := runGit(t, specs, "rev-parse", "HEAD")
	seen := observe(t)

	result := run(t, master, onlyRequest())

	wantPublished(t, result)
	if result.Commit != nil || runGit(t, specs, "rev-parse", "HEAD") != head {
		t.Errorf("no debe crear un commit nuevo: %+v", result.Commit)
	}
	if len(gitInvocations(*seen, "commit")) != 0 || len(gitInvocations(*seen, "push")) != 1 {
		t.Errorf("git = %v", *seen)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/main"); got != head {
		t.Errorf("el remoto tiene %s", got)
	}
}

func TestOnlyOtherCommitsUnpublishedCreatesNoCommit(t *testing.T) {
	master, specs, bare := newMultirepoSpecs(t)
	editDesignAt(t, specs, "diseño nuevo\n")
	runGit(t, specs, "add", "-A")
	runGit(t, specs, "commit", "-q", "-m", "otro documento")
	head := runGit(t, specs, "rev-parse", "HEAD")
	remoteHead := runGit(t, bare, "rev-parse", "refs/heads/main")
	writeMarks(t, specs, marksFile(t, nil, "P-01"))

	result := run(t, master, onlyRequest())

	refusal := wantRefused(t, result, CodeOtherCommitsUnpublished)
	for _, want := range []string{"git -C", "log origin/main..HEAD --stat", "axiom sdd publish --change x"} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q no contiene %q", refusal.Continuation, want)
		}
	}
	if result.Commit != nil || runGit(t, specs, "rev-parse", "HEAD") != head || runGit(t, bare, "rev-parse", "refs/heads/main") != remoteHead {
		t.Error("no debe crear commits ni subir nada")
	}
	if runGit(t, specs, "status", "--porcelain", "--", marksRel) == "" {
		t.Error("las marcas siguen sin confirmar")
	}
}

func editDesignAt(t *testing.T, specs, content string) {
	t.Helper()
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), content)
}

func TestOnlyPrePushHookRefusalAndNoFetch(t *testing.T) {
	master, specs, _ := newMultirepoSpecs(t)
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	installHook(t, filepath.Join(specs, ".git"), "pre-push")
	seen := observe(t)

	result := run(t, master, Request{Only: []string{"pruebas-estado.yaml"}, Replay: marks.Merge})

	wantRefused(t, result, CodePushHookRejected)
	if got := gitInvocations(*seen, "fetch", "pull", "ls-remote"); len(got) != 0 {
		t.Errorf("sin non-fast-forward no hay fetch: %v", got)
	}
	if result.Commit == nil {
		t.Error("el commit local se conserva y se informa")
	}
}

// Reproducción: otra persona subió marcas antes; el reintento único fusiona y sube.
func TestReplayMergesRemoteMarksAndPushes(t *testing.T) {
	master, specs, bare := newMultirepoSpecs(t)
	pushFromClone(t, bare, marksFile(t, nil, "P-09"))
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	seen := observe(t)

	result := run(t, master, Request{Only: []string{"pruebas-estado.yaml"}, Replay: marks.Merge})

	wantPublished(t, result)
	if !result.Replayed || result.Commit == nil {
		t.Fatalf("resultado = %+v", result)
	}
	remote, err := marks.Parse([]byte(runGit(t, bare, "show", "main:"+marksRel) + "\n"))
	if err != nil || len(remote.Entries) != 2 || remote.Entries[0].Case != "P-09" || remote.Entries[1].Case != "P-01" {
		t.Fatalf("remoto = %+v %v", remote, err)
	}
	if got := runGit(t, bare, "rev-parse", "refs/heads/main"); got != runGit(t, specs, "rev-parse", "HEAD") {
		t.Error("el remoto debe tener HEAD")
	}
	if got := gitInvocations(*seen, "fetch"); len(got) != 1 {
		t.Errorf("un único fetch, hay %v", got)
	}
	if got := gitInvocations(*seen, "push"); len(got) != 2 {
		t.Errorf("dos pushes (original y reintento), hay %v", got)
	}
	for _, invocation := range *seen {
		joined := strings.Join(invocation[1:], " ")
		if strings.Contains(joined, "rebase") || strings.Contains(joined, "--force") || strings.Contains(joined, "--no-verify") || strings.Contains(joined, " +") {
			t.Errorf("operación prohibida: %v", invocation)
		}
	}
}

// Todo estaba ya publicado (un push que el cliente creyó fallido): up_to_date sin commit nuevo.
func TestReplayUpToDateWhenEverythingWasAlreadyPublished(t *testing.T) {
	master, specs, bare := newMultirepoSpecs(t)
	local := marksFile(t, nil, "P-01")
	writeMarks(t, specs, local)
	runGit(t, specs, "add", marksRel)
	runGit(t, specs, "commit", "-q", "-m", "marcas locales")
	pushFromClone(t, bare, marksFile(t, local, "P-02")) // el remoto ya tiene P-01 y además P-02

	result := run(t, master, Request{Only: []string{"pruebas-estado.yaml"}, Replay: marks.Merge})

	if result.Status != StatusUpToDate || !result.Replayed || result.Commit != nil || result.Refusal != nil {
		t.Fatalf("resultado = %s %+v", result.Status, result)
	}
	if got := runGit(t, specs, "rev-parse", "HEAD"); got != runGit(t, bare, "rev-parse", "refs/heads/main") {
		t.Error("la rama local debe quedar en el remoto")
	}
	if runGit(t, specs, "status", "--porcelain") != "" {
		t.Error("árbol limpio")
	}
}

// failingGit sustituye operaciones del adaptador real para fijar los fallos de R1 a R6.
type failingGit struct {
	*GitCLI
	nonFastForward bool // todo push se rechaza como non-fast-forward
	fetch          *PushOutcome
	resetErr       error
	pushes         int
}

func (g *failingGit) Push(ctx context.Context, dir, remote, branch string, up bool) (PushOutcome, error) {
	g.pushes++
	if g.nonFastForward {
		return PushOutcome{ExitCode: 1, Stdout: "!\trefs/heads/main:refs/heads/main\t[rejected] (non-fast-forward)\n"}, nil
	}
	return g.GitCLI.Push(ctx, dir, remote, branch, up)
}

func (g *failingGit) Fetch(ctx context.Context, dir, remote, branch string) (PushOutcome, error) {
	if g.fetch != nil {
		return *g.fetch, nil
	}
	return g.GitCLI.Fetch(ctx, dir, remote, branch)
}

func (g *failingGit) ResetKeep(ctx context.Context, dir, rev string) error {
	if g.resetErr != nil {
		return g.resetErr
	}
	return g.GitCLI.ResetKeep(ctx, dir, rev)
}

func runWith(t *testing.T, git Git, cwd string, req Request) Result {
	t.Helper()
	req.CWD, req.Change = cwd, "x"
	return (&Publisher{Git: git, Targets: engine.Default()}).Run(context.Background(), req)
}

func TestReplayDoubleNonFastForwardHasNoThirdAttemptAndKeepsTheCommit(t *testing.T) {
	master, specs, _ := newMultirepoSpecs(t)
	writeMarks(t, specs, marksFile(t, nil, "P-01"))
	git := &failingGit{GitCLI: NewGit(), nonFastForward: true}

	result := runWith(t, git, master, Request{Only: []string{"pruebas-estado.yaml"}, Replay: marks.Merge})

	wantRefused(t, result, CodePushNonFastForward)
	if git.pushes != 2 {
		t.Errorf("pushes = %d, se esperaban 2 (sin tercer intento)", git.pushes)
	}
	if result.Commit == nil || runGit(t, specs, "log", "-1", "--format=%s") != result.Commit.Message || !result.Replayed {
		t.Errorf("el commit nuevo se conserva: %+v", result.Commit)
	}
}

// Fallos de R1 a R4: nada cambia en el árbol ni en la rama y el commit local se conserva.
func TestReplayFailuresKeepTheLocalState(t *testing.T) {
	unavailable := &PushOutcome{ExitCode: 128, Stderr: "fatal: unable to access 'x': Could not resolve host: y"}
	otherFailure := &PushOutcome{ExitCode: 1, Stderr: "fatal: algo raro"}
	cases := []struct {
		name  string
		git   func() *failingGit
		merge func(l, r []byte) ([]byte, error)
		code  Code
	}{
		{"R1 sin red", func() *failingGit { return &failingGit{GitCLI: NewGit(), nonFastForward: true, fetch: unavailable} }, marks.Merge, CodeRemoteUnavailable},
		{"R1 otro fallo", func() *failingGit { return &failingGit{GitCLI: NewGit(), nonFastForward: true, fetch: otherFailure} }, marks.Merge, CodeReplayFailed},
		{"R1 tiempo agotado", func() *failingGit {
			return &failingGit{GitCLI: NewGit(), nonFastForward: true, fetch: &PushOutcome{TimedOut: true}}
		}, marks.Merge, CodeRemoteUnavailable},
		{"R3 fusión", func() *failingGit { return &failingGit{GitCLI: NewGit(), nonFastForward: true} },
			func(l, r []byte) ([]byte, error) { return nil, errors.New("YAML remoto ilegible") }, CodeReplayFailed},
		{"R4 reset", func() *failingGit {
			return &failingGit{GitCLI: NewGit(), nonFastForward: true, resetErr: errors.New("Entry would be overwritten")}
		}, marks.Merge, CodeReplayFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			master, specs, _ := newMultirepoSpecs(t)
			writeMarks(t, specs, marksFile(t, nil, "P-01"))
			git := tc.git()

			result := runWith(t, git, master, Request{Only: []string{"pruebas-estado.yaml"}, Replay: tc.merge})

			refusal := wantRefused(t, result, tc.code)
			if tc.code == CodeReplayFailed && !strings.Contains(refusal.Message, "reproducción interrumpida") {
				t.Errorf("mensaje = %q", refusal.Message)
			}
			if git.pushes != 1 {
				t.Errorf("pushes = %d, solo el original", git.pushes)
			}
			if result.Commit == nil || runGit(t, specs, "log", "-1", "--format=%s") != result.Commit.Message {
				t.Error("el commit local se conserva")
			}
			if runGit(t, specs, "status", "--porcelain") != "" {
				t.Errorf("el árbol no cambia: %s", runGit(t, specs, "status", "--porcelain"))
			}
		})
	}
}

// El JSON de `axiom sdd publish` no cambia: los campos nuevos son omitempty.
func TestOnlyFieldsAreOmittedFromTheOrdinaryResult(t *testing.T) {
	repo, _ := newMonorepo(t)
	editDesign(t, repo, "diseño nuevo\n")
	result := run(t, repo, Request{})
	wantPublished(t, result)
	if result.UncommittedDocs != nil || result.Replayed {
		t.Errorf("la publicación ordinaria no rellena los campos acotados: %+v", result)
	}
}
