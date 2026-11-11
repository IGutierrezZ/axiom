package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

func runProjectMasterGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

// ProjectMaster es la clave del hub: un worktree enlazado debe resolver la carpeta maestra
// del checkout principal, y fuera de un proyecto Axiom no hay maestra.
func TestProjectMasterIsSharedByLinkedWorktrees(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(root, "proyecto")
	if err := os.MkdirAll(filepath.Join(main, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	runProjectMasterGit(t, main, "init", "-q")
	runProjectMasterGit(t, main, "config", "user.email", "test@example.com")
	runProjectMasterGit(t, main, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(main, DefaultConfigFilename), []byte("workspace:\n  name: p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runProjectMasterGit(t, main, "add", DefaultConfigFilename)
	runProjectMasterGit(t, main, "commit", "-qm", "base")
	linked := filepath.Join(root, "proyecto-wt")
	runProjectMasterGit(t, main, "worktree", "add", "-q", "-b", "rama", linked)

	for name, start := range map[string]string{
		"checkout principal": main,
		"subcarpeta":         filepath.Join(main, "sub"),
		"worktree enlazado":  linked,
	} {
		master, ok := ProjectMaster(start)
		if !ok || !pathidentity.SameDirectory(master, main) {
			t.Errorf("%s: ProjectMaster(%s) = %q, %v; se esperaba la carpeta maestra %s", name, start, master, ok, main)
		}
	}
}

func TestProjectMasterOutsideAnAxiomProjectIsNotAProject(t *testing.T) {
	dir := t.TempDir()
	runProjectMasterGit(t, dir, "init", "-q")
	if master, ok := ProjectMaster(dir); ok {
		t.Fatalf("ProjectMaster(%s) = %q, true; un repositorio sin axiom.yaml no es un proyecto", dir, master)
	}
}
