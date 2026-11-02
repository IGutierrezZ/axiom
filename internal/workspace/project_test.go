package workspace

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func assertAxiomProject(t *testing.T, start, wantRoot string) {
	t.Helper()
	root, ok := IsAxiomProject(start)
	if !ok || root != wantRoot {
		t.Fatalf("IsAxiomProject(%q) = (%q, %v), se esperaba (%q, true)", start, root, ok, wantRoot)
	}
}

func assertNotAxiomProject(t *testing.T, start string) {
	t.Helper()
	if root, ok := IsAxiomProject(start); ok {
		t.Fatalf("IsAxiomProject(%q) = (%q, true), se esperaba que no fuera un proyecto Axiom", start, root)
	}
}

func TestIsAxiomProjectAcceptsAxiomYAMLInDirAndAncestors(t *testing.T) {
	ws := workspacetest.Embedded(t, "openspec")
	sub := filepath.Join(ws.Master, "internal", "pkg")
	workspacetest.MkdirAll(t, sub)

	assertAxiomProject(t, ws.Master, ws.Master)
	assertAxiomProject(t, sub, ws.Master)
}

func TestIsAxiomProjectAcceptsTheMultirepoPointer(t *testing.T) {
	ws := workspacetest.Canonical(t)

	// El puntero vive en la carpeta maestra; los repositorios de código y el de
	// especificaciones cuelgan de ella. Desde el de especificaciones, el
	// axiom.yaml propio es el marcador más cercano.
	assertAxiomProject(t, ws.Master, ws.Master)
	assertAxiomProject(t, filepath.Join(ws.Backend, "src"), ws.Master)
	assertAxiomProject(t, ws.Specs, ws.Specs)
}

func TestIsAxiomProjectIgnoresABrokenPointer(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())

	assertNotAxiomProject(t, ws.Master)
	assertNotAxiomProject(t, ws.Backend)
}

func TestIsAxiomProjectRejectsMarkersOfOtherTools(t *testing.T) {
	// openspec/ es el marcador de OpenSpec, no de Axiom: Locate lo acepta como
	// OriginOpenSpec, pero un repositorio ajeno que lo use no es un proyecto Axiom.
	root := evalTempDir(t)
	workspacetest.MkdirAll(t, filepath.Join(root, "openspec", "changes", "algo"))
	workspacetest.MkdirAll(t, filepath.Join(root, ".git"))
	workspacetest.MkdirAll(t, filepath.Join(root, ".atl"))
	workspacetest.MkdirAll(t, filepath.Join(root, "skills"))
	workspacetest.MkdirAll(t, filepath.Join(root, ".claude", "skills"))
	workspacetest.WriteFile(t, filepath.Join(root, "AGENTS.md"), "# Notas\n")

	assertNotAxiomProject(t, root)
	if loc := mustLocate(t, root); loc.Origin != OriginOpenSpec {
		t.Fatalf("Locate debe seguir aceptando openspec/ (REQ-24.2), origen %q", loc.Origin)
	}
}

func TestIsAxiomProjectKeepsAscendingPastABareOpenSpec(t *testing.T) {
	ws := workspacetest.Embedded(t, "openspec")
	nested := filepath.Join(ws.Master, "vendor", "otro")
	workspacetest.MkdirAll(t, filepath.Join(nested, "openspec"))

	assertAxiomProject(t, nested, ws.Master)
}

func TestIsAxiomProjectRejectsPlainAndInvalidStarts(t *testing.T) {
	plain := evalTempDir(t)
	assertNotAxiomProject(t, plain)
	assertNotAxiomProject(t, filepath.Join(plain, "no-existe"))

	file := filepath.Join(plain, "fichero.txt")
	workspacetest.WriteFile(t, file, "x")
	assertNotAxiomProject(t, file)

	assertNotAxiomProject(t, filepath.VolumeName(plain)+string(filepath.Separator))
}

func TestIsAxiomProjectInALinkedWorktree(t *testing.T) {
	t.Run("con axiom.yaml propio", func(t *testing.T) {
		main := workspacetest.Embedded(t, "openspec")
		worktree := filepath.Join(evalTempDir(t), "x", "repo")
		workspacetest.WriteFile(t, filepath.Join(worktree, "axiom.yaml"), embeddedConfigOf(t, main.Master))
		workspacetest.MkdirAll(t, filepath.Join(worktree, "internal"))
		linkWorktreeOfCheckout(t, main.Master, worktree, "x")

		assertAxiomProject(t, filepath.Join(worktree, "internal"), worktree)
	})

	t.Run("sin marcador propio resuelve desde el checkout principal", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		outside := evalTempDir(t)
		worktree := filepath.Join(outside, "x", "repo-backend")
		workspacetest.MkdirAll(t, filepath.Join(worktree, "src"))
		linkWorktreeOfCheckout(t, ws.Backend, worktree, "x")
		workspacetest.MkdirAll(t, filepath.Join(ws.Backend, ".git"))

		assertAxiomProject(t, filepath.Join(worktree, "src"), ws.Master)
	})

	t.Run("un marcador fuera del worktree no cuenta", func(t *testing.T) {
		main := evalTempDir(t)
		outside := evalTempDir(t)
		workspacetest.WriteFile(t, filepath.Join(outside, "axiom.yaml"), "workspace:\n  name: ajeno\n")
		worktree := filepath.Join(outside, "wt")
		workspacetest.MkdirAll(t, worktree)
		linkWorktreeOfCheckout(t, main, worktree, "x")

		assertNotAxiomProject(t, worktree)
	})
}
