package workspace

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func assertRoot(t *testing.T, start, home, wantRoot string, wantSource RootSource) {
	t.Helper()
	root, source := ResolveRoot(start, home)
	if root != wantRoot || source != wantSource {
		t.Fatalf("ResolveRoot(%q, %q) = (%q, %q), se esperaba (%q, %q)", start, home, root, source, wantRoot, wantSource)
	}
}

func TestResolveRootPrefersTheAxiomProjectOverTheGitRepository(t *testing.T) {
	ws := workspacetest.Embedded(t, "openspec")
	workspacetest.MkdirAll(t, filepath.Join(ws.Master, ".git"))
	sub := filepath.Join(ws.Master, "internal", "pkg")
	workspacetest.MkdirAll(t, sub)

	assertRoot(t, sub, "", ws.Master, RootSourceAxiomProject)
	assertRoot(t, ws.Master, "", ws.Master, RootSourceAxiomProject)
}

func TestResolveRootUsesTheAxiomProjectAboveANestedGitRepository(t *testing.T) {
	ws := workspacetest.Embedded(t, "openspec")
	nested := filepath.Join(ws.Master, "vendor", "otro")
	workspacetest.MkdirAll(t, filepath.Join(nested, ".git"))

	assertRoot(t, nested, "", ws.Master, RootSourceAxiomProject)
}

func TestResolveRootFollowsTheMultirepoPointer(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.MkdirAll(t, filepath.Join(ws.Backend, ".git"))

	assertRoot(t, filepath.Join(ws.Backend, "src"), "", ws.Master, RootSourceAxiomProject)
}

func TestResolveRootFallsBackToTheGitRepositoryRoot(t *testing.T) {
	root := evalTempDir(t)
	workspacetest.MkdirAll(t, filepath.Join(root, ".git"))
	workspacetest.MkdirAll(t, filepath.Join(root, "openspec", "changes", "ajeno"))
	sub := filepath.Join(root, "src", "app")
	workspacetest.MkdirAll(t, sub)

	assertRoot(t, sub, "", root, RootSourceGitRepository)
}

func TestResolveRootKeepsTheStartOutsideAnyProject(t *testing.T) {
	plain := evalTempDir(t)
	sub := filepath.Join(plain, "a", "b")
	workspacetest.MkdirAll(t, sub)

	assertRoot(t, sub, "", sub, RootSourceCurrentDir)
}

func TestResolveRootNeverTakesTheHomeDirectoryAsGitRoot(t *testing.T) {
	home := evalTempDir(t)
	// Un repositorio de dotfiles: el home entero es un repositorio git.
	workspacetest.MkdirAll(t, filepath.Join(home, ".git"))
	sub := filepath.Join(home, "proyectos", "ajeno")
	workspacetest.MkdirAll(t, sub)

	assertRoot(t, sub, home, sub, RootSourceCurrentDir)
	// Un repositorio propio dentro del home sí es raíz.
	repo := filepath.Join(home, "proyectos", "propio")
	workspacetest.MkdirAll(t, filepath.Join(repo, ".git"))
	workspacetest.MkdirAll(t, filepath.Join(repo, "src"))
	assertRoot(t, filepath.Join(repo, "src"), home, repo, RootSourceGitRepository)
}

func TestResolveRootAcceptsAnAxiomYAMLInTheHomeDirectory(t *testing.T) {
	ws := workspacetest.Embedded(t, "openspec")
	sub := filepath.Join(ws.Master, "docs")
	workspacetest.MkdirAll(t, sub)

	assertRoot(t, sub, ws.Master, ws.Master, RootSourceAxiomProject)
}

func TestResolveRootInALinkedWorktreeStaysInTheWorktree(t *testing.T) {
	t.Run("con axiom.yaml propio", func(t *testing.T) {
		main := workspacetest.Embedded(t, "openspec")
		worktree := filepath.Join(evalTempDir(t), "x", "repo")
		workspacetest.WriteFile(t, filepath.Join(worktree, "axiom.yaml"), embeddedConfigOf(t, main.Master))
		workspacetest.MkdirAll(t, filepath.Join(worktree, "internal"))
		linkWorktreeOfCheckout(t, main.Master, worktree, "x")

		assertRoot(t, filepath.Join(worktree, "internal"), "", worktree, RootSourceAxiomProject)
	})

	t.Run("sin marcador propio no sube al checkout principal", func(t *testing.T) {
		main := workspacetest.Embedded(t, "openspec")
		worktree := filepath.Join(evalTempDir(t), "x", "repo")
		workspacetest.MkdirAll(t, filepath.Join(worktree, "internal"))
		linkWorktreeOfCheckout(t, main.Master, worktree, "x")

		if _, ok := IsAxiomProject(worktree); !ok {
			t.Fatal("precondición: IsAxiomProject resuelve desde el checkout principal")
		}
		assertRoot(t, filepath.Join(worktree, "internal"), "", worktree, RootSourceGitRepository)
	})
}

func TestResolveRootReturnsInvalidStartsUnchanged(t *testing.T) {
	plain := evalTempDir(t)
	missing := filepath.Join(plain, "no-existe")
	assertRoot(t, missing, "", missing, RootSourceCurrentDir)

	system := filepath.VolumeName(plain) + string(filepath.Separator)
	assertRoot(t, system, "", system, RootSourceCurrentDir)
}
