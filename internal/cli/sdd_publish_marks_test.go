package cli

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
)

// Pruebas del publicador de marcas del dashboard (C-06-d): git real, un remoto desnudo local y el
// motor de ramas real, sin red.

const marksPublishRel = "openspec/changes/x/pruebas-estado.yaml"

// newMarksPublishWorkspace monta un multirepo con el repositorio de specs en main, el cambio x
// publicado y un remoto desnudo. Devuelve la carpeta maestra, el repo de specs y el remoto.
func newMarksPublishWorkspace(t *testing.T) (master, specs, bare string) {
	t.Helper()
	isolateBranchGit(t)
	master = t.TempDir()
	specs, backend := filepath.Join(master, "repo-specs"), filepath.Join(master, "repo-backend")
	for _, dir := range []string{specs, backend} {
		writePublishFile(t, filepath.Join(dir, "README.md"), "inicial\n")
		branchTestGit(t, dir, "init", "-q", "-b", "main")
		branchTestGit(t, dir, "add", "-A")
		branchTestGit(t, dir, "commit", "-q", "-m", "inicial")
	}
	writePublishFile(t, filepath.Join(master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: backend\n    repositories:\n      - path: \"repo-backend\"\n")
	writePublishFile(t, filepath.Join(specs, "openspec", "changes", "x", "proposal.md"), "propuesta\n")
	writePublishFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "diseño\n")
	branchTestGit(t, specs, "add", "-A")
	branchTestGit(t, specs, "commit", "-q", "-m", "cambio x")
	bare = t.TempDir()
	branchTestGit(t, bare, "init", "-q", "--bare", "-b", "main")
	branchTestGit(t, specs, "remote", "add", "origin", filepath.ToSlash(bare))
	branchTestGit(t, specs, "push", "-q", "origin", "main")
	return master, specs, bare
}

func writeMarksPublishFile(t *testing.T, specs string) {
	t.Helper()
	data, err := marks.Append(nil, marks.Entry{
		Case: "P-01", Environment: "pre", Result: marks.ResultOK, By: "Ana <ana@example.org>",
		At: "2026-11-26T10:00:00.000Z", PlanDigest: strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(specs, filepath.FromSlash(marksPublishRel)), string(data))
}

func TestMarksPublisherPublishesOnlyTheMarksFile(t *testing.T) {
	master, specs, bare := newMarksPublishWorkspace(t)
	writeMarksPublishFile(t, specs)
	writePublishFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "diseño sin confirmar\n")

	got := NewMarksPublisher().PublishMarks(context.Background(), MarksPublishRequest{CWD: master, Change: "x", Branch: "main"})

	if got.Refusal != nil || got.Status != "published" || got.Remote != "origin" || got.Branch != "main" {
		t.Fatalf("resultado = %+v", got)
	}
	if got.Commit == "" || got.PushedSHA != got.Commit || !sameDir(got.GitRoot, specs) {
		t.Errorf("commit/push/raíz = %+v", got)
	}
	if !slices.Equal(got.UncommittedDocs, []string{"openspec/changes/x/design.md"}) {
		t.Errorf("UncommittedDocs = %v", got.UncommittedDocs)
	}
	if published := publishGitOutput(t, bare, "rev-parse", "refs/heads/main"); strings.TrimSpace(published) != strings.TrimSpace(publishGitOutput(t, specs, "rev-parse", "HEAD")) {
		t.Errorf("el remoto no tiene HEAD: %s", published)
	}
	if files := strings.Fields(publishGitOutput(t, specs, "show", "--name-only", "--format=", "HEAD")); !slices.Equal(files, []string{marksPublishRel}) {
		t.Errorf("el commit contiene %v", files)
	}
	// Con todo publicado, repetir no crea nada.
	again := NewMarksPublisher().PublishMarks(context.Background(), MarksPublishRequest{CWD: master, Change: "x"})
	if again.Refusal != nil || again.Status != "up_to_date" {
		t.Errorf("segunda publicación = %+v", again)
	}
}

func TestMarksPublisherRefusalsAreTypedWithTheirContinuation(t *testing.T) {
	master, specs, _ := newMarksPublishWorkspace(t)
	writeMarksPublishFile(t, specs)

	wrong := NewMarksPublisher().PublishMarks(context.Background(), MarksPublishRequest{CWD: master, Change: "x", Branch: "otra"})
	if wrong.Status != "refused" || wrong.Refusal == nil || wrong.Refusal.Code != "wrong_branch" || wrong.Refusal.Message == "" || wrong.Refusal.Continuation == "" || wrong.Refusal.Label == "" {
		t.Fatalf("rama distinta = %+v", wrong)
	}

	missing := NewMarksPublisher().PublishMarks(context.Background(), MarksPublishRequest{CWD: master, Change: "no-existe"})
	if missing.Refusal == nil || missing.Refusal.Code != "change_not_found" || missing.Refusal.Continuation == "" {
		t.Errorf("cambio inexistente = %+v", missing)
	}

	// Un commit ajeno sin subir no se sube a escondidas: negativa con la continuación concreta.
	writePublishFile(t, filepath.Join(specs, "openspec", "changes", "x", "design.md"), "otro diseño\n")
	branchTestGit(t, specs, "add", "-A", "--", "openspec/changes/x/design.md")
	branchTestGit(t, specs, "commit", "-q", "-m", "diseño ajeno")
	other := NewMarksPublisher().PublishMarks(context.Background(), MarksPublishRequest{CWD: master, Change: "x"})
	if other.Refusal == nil || other.Refusal.Code != "other_commits_unpublished" {
		t.Fatalf("commits ajenos = %+v", other)
	}
	if !strings.Contains(other.Refusal.Continuation, "log origin/main..HEAD --stat") || !strings.Contains(other.Refusal.Continuation, "axiom sdd publish --change x") {
		t.Errorf("continuación = %q", other.Refusal.Continuation)
	}
	if last := strings.TrimSpace(publishGitOutput(t, specs, "log", "-1", "--format=%s")); last != "diseño ajeno" {
		t.Errorf("no debe crearse un commit de marcas: %q", last)
	}
}

func TestMarksPublisherIdentity(t *testing.T) {
	original := resolveActorIdentity
	t.Cleanup(func() { resolveActorIdentity = original })
	dir := t.TempDir()
	var asked string
	resolveActorIdentity = func(_ context.Context, got string) gitidentity.Identity {
		asked = got
		return gitidentity.New("Ana Pérez", "ana@example.org")
	}
	actor, missing := NewMarksPublisher().MarksIdentity(context.Background(), dir)
	if actor != "Ana Pérez <ana@example.org>" || len(missing) != 0 || asked != dir {
		t.Fatalf("identidad completa = %q %v (dir %q)", actor, missing, asked)
	}
	resolveActorIdentity = func(context.Context, string) gitidentity.Identity {
		return gitidentity.New("Ana Pérez", "  ")
	}
	actor, missing = NewMarksPublisher().MarksIdentity(context.Background(), dir)
	if actor != "" || !slices.Equal(missing, []string{"user.email"}) {
		t.Fatalf("sin correo = %q %v", actor, missing)
	}
}

// sameDir compara directorios por identidad: en Windows el mismo directorio puede llegar con
// nombre corto o largo.
func sameDir(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}
