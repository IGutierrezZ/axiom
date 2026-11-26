package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Lectura de la versión publicada: sin referencia remota, con referencia sin el fichero y con el
// fichero; siempre sin red y sin escribir nada.
func TestPublishedChangeFile(t *testing.T) {
	t.Parallel()
	_, worktree, _ := stateWorktree(t)
	request := branch.PublicationRequest{CWD: worktree, Change: "x"}
	read := func() (branch.PublishedChangeFile, *branch.Refusal) {
		return Default().PublishedChangeFile(context.Background(), request, "pruebas-estado.yaml")
	}

	got, refusal := read()
	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if got.RemoteRefExists || got.Exists || len(got.Content) != 0 || got.RemoteRef != "refs/remotes/origin/feat/x" {
		t.Fatalf("sin referencia remota: %+v", got)
	}

	// Con referencia pero sin el fichero en ella.
	runGit(t, worktree, "push", "-q", "origin", "feat/x")
	if got, _ = read(); !got.RemoteRefExists || got.Exists || len(got.Content) != 0 {
		t.Fatalf("referencia sin fichero: %+v", got)
	}

	// El fichero sin confirmar o sin subir no es la versión publicada.
	writeFile(t, changeFile(worktree, "pruebas-estado.yaml"), "- case: P-01\n")
	runGit(t, worktree, "add", "--", "openspec/changes/x/pruebas-estado.yaml")
	commit(t, worktree, "marcas")
	if got, _ = read(); got.Exists {
		t.Fatalf("un commit sin subir no es publicado: %+v", got)
	}

	// Tras subir, es el contenido de la referencia, aunque el local cambie después.
	runGit(t, worktree, "push", "-q", "origin", "feat/x")
	writeFile(t, changeFile(worktree, "pruebas-estado.yaml"), "- case: P-01\n- case: P-02\n")
	got, refusal = read()
	if refusal != nil || !got.Exists || string(got.Content) != "- case: P-01\n" {
		t.Fatalf("publicado = %+v, %v", got, refusal)
	}
	wantSameDirectory(t, "GitRoot", got.Target.GitRoot, worktree)
	if status := runGit(t, worktree, "status", "--porcelain", "--", filepath.ToSlash("openspec/changes/x/pruebas-estado.yaml")); status == "" {
		t.Error("la lectura no debe haber tocado el fichero local modificado")
	}
}

func TestPublishedChangeFileRefusesWrongBranch(t *testing.T) {
	t.Parallel()
	_, worktree, _ := stateWorktree(t)
	_, refusal := Default().PublishedChangeFile(context.Background(),
		branch.PublicationRequest{CWD: worktree, Change: "x", Branch: "otra"}, "pruebas-estado.yaml")
	if refusal == nil || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("rechazo = %+v, se esperaba wrong_branch", refusal)
	}
}
