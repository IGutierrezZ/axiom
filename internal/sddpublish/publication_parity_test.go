package sddpublish

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
)

// Paridad entre `publish` y el estado de publicación del status (RF-PUB-11): los dos componen la
// misma lista cerrada y miden el mismo rango, de modo que lo que el status anuncia como pendiente es
// exactamente lo que `publish` haría. El motor se importa solo desde la prueba: en producción solo
// `internal/cli/sdd_publish.go` importa este paquete.

func publicationStateOf(t *testing.T, repo string) branch.PublicationState {
	t.Helper()
	state, refusal := engine.Default().PublicationState(context.Background(), branch.PublicationRequest{CWD: repo, Change: "x"})
	if refusal != nil {
		t.Fatalf("PublicationState rechazó %q: %s", refusal.Code, refusal.Message)
	}
	return state
}

func TestPublicationStateAgreesWithPublish(t *testing.T) {
	repo, _ := newMonorepo(t)

	// Con la rama del cambio sin publicar y nada sucio, no hay documentación pendiente.
	if state := publicationStateOf(t, repo); len(state.DirtyDocs) != 0 || state.Unpushed != 0 || state.RemoteBranch {
		t.Fatalf("estado inicial = %+v, se esperaba sin nada pendiente", state)
	}

	// Un documento editado: el estado lo ve y `publish` lo confirma, con las mismas rutas.
	editDesign(t, repo, "diseño nuevo\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", ".gates.lock"), "")
	state := publicationStateOf(t, repo)
	result := run(t, repo, Request{})
	wantPublished(t, result)
	if !slices.Equal(state.DirtyDocs, result.Commit.Paths) {
		t.Errorf("DirtyDocs = %v, publish confirmó %v", state.DirtyDocs, result.Commit.Paths)
	}

	// Tras un `publish` correcto, el estado no ve nada.
	after := publicationStateOf(t, repo)
	if len(after.DirtyDocs) != 0 || after.Unpushed != 0 || len(after.CodeCommits) != 0 || !after.RemoteBranch {
		t.Errorf("tras publicar, estado = %+v; se esperaba todo vacío y con rama remota", after)
	}
}

// Un commit de código sin subir lo ven los dos: el estado lo lista y `publish` se niega con el mismo.
func TestPublicationStateSeesTheCodeCommitsPublishRefuses(t *testing.T) {
	repo, _ := newMonorepo(t)
	writeFile(t, filepath.Join(repo, "README.sh"), "echo\n")
	runGit(t, repo, "add", "README.sh")
	runGit(t, repo, "commit", "-q", "-m", "código")
	short := runGit(t, repo, "rev-parse", "--short=7", "HEAD")

	state := publicationStateOf(t, repo)
	refusal := wantRefused(t, run(t, repo, Request{}), CodeCodeCommitsUnpublished)
	if !slices.Equal(state.CodeCommits, []string{short}) {
		t.Errorf("CodeCommits = %v, se esperaba [%s]", state.CodeCommits, short)
	}
	if got, _ := refusal.Details["commits"].([]string); !slices.Equal(got, state.CodeCommits) {
		t.Errorf("publish nombra %v y el estado %v", refusal.Details["commits"], state.CodeCommits)
	}
}

func TestPublicationRangeUnknownCodeMatchesPublish(t *testing.T) {
	if string(branch.PublicationCodeRangeUnknown) != string(CodeUnpublishedRangeUnknown) {
		t.Errorf("branch.PublicationCodeRangeUnknown = %q, sddpublish.CodeUnpublishedRangeUnknown = %q", branch.PublicationCodeRangeUnknown, CodeUnpublishedRangeUnknown)
	}
	if branch.PublicationMaxUnpushed != maxUnpublishedCommits {
		t.Errorf("branch.PublicationMaxUnpushed = %d, sddpublish.maxUnpublishedCommits = %d", branch.PublicationMaxUnpushed, maxUnpublishedCommits)
	}
}
