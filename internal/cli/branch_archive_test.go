package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch ensure --archive` (INC-25, S5, REQ-25.45, REQ-25.12): la ayuda, el uso de las
// banderas, los rechazos tipados con código de salida 1 y la creación de la rama de archivado de
// un cambio fusionado, con y sin `--fetch` contra un remoto desnudo local. El comportamiento del
// motor se prueba en internal/branch/engine.

// archiveWriteChange escribe y confirma en la rama actual de dir la carpeta del cambio.
func archiveWriteChange(t *testing.T, dir, change string) {
	t.Helper()
	changeDir := filepath.Join(dir, "openspec", "changes", change)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("propuesta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	branchTestGit(t, dir, "add", "-A")
	branchTestGit(t, dir, "commit", "-q", "-m", "cambio "+change)
}

func TestBranchEnsureHelpDocumentsArchive(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "ensure", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{
		"[--archive]", "  --archive ", "<cambio>-archive", "change_not_integrated", "archive_branch_not_applicable",
		"plan_conflict", "--fetch", "nunca la publica", "solo admite --prefix",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
		}
	}
}

// REQ-25.12: `--repo` no se admite con `--archive` (error de uso, código 2, que nombra las dos
// banderas y la orden que sí vale); `--branch` y `--base` llegan al motor y son `plan_conflict`
// (código 1); el carril ODD no tiene rama de archivado (código 1). Ningún caso crea nada.
func TestBranchEnsureArchiveFlagsAreValidated(t *testing.T) {
	repo := newBranchRepository(t)
	branches := branchGitOutput(t, repo, "branch", "--list")

	t.Run("--repo es un error de uso", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--archive", "--repo", ".", "--cwd", repo)

		if code != 2 || stdout != "" {
			t.Fatalf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
		}
		for _, want := range []string{"--repo", "--archive", "axiom branch ensure --change <nombre> --archive"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q debe contener %q", stderr, want)
			}
		}
	})
	t.Run("--branch y --base son plan_conflict", func(t *testing.T) {
		for flag, value := range map[string]string{"--branch": "doc/x", "--base": "release/2"} {
			code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--archive", flag, value, "--json", "--cwd", repo)

			if code != 1 || stderr != "" {
				t.Fatalf("%s: código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", flag, code, stderr, stdout)
			}
			result := decodeBranchResult(t, stdout)
			if result.Operation != "ensure" || result.Status != "refused" || result.Code != "plan_conflict" || !strings.Contains(result.Message, flag) {
				t.Errorf("%s: operación %q, status %q, código %q, mensaje %q", flag, result.Operation, result.Status, result.Code, result.Message)
			}
			if want := "axiom branch ensure --archive --change x --cwd " + pathquote.Quote(repo); !strings.Contains(result.Continuation, want) {
				t.Errorf("%s: la continuación %q debe contener %q", flag, result.Continuation, want)
			}
		}
	})
	t.Run("el carril ODD no tiene rama de archivado", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "mi-feature", "--lane", "odd", "--archive", "--cwd", repo)

		if code != 1 || stdout != "" {
			t.Fatalf("código %d, stdout %q; se esperaba 1 y stdout vacío en texto", code, stdout)
		}
		for _, want := range []string{"Error [archive_branch_not_applicable]", "ODD", "Continuación: "} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q debe contener %q", stderr, want)
			}
		}
	})
	if got := branchGitOutput(t, repo, "branch", "--list"); got != branches {
		t.Errorf("los rechazos no crean ramas:\n%s\n---\n%s", branches, got)
	}
}

// REQ-25.45 (harness S5.6): tras fusionar `x` solo en el remoto desnudo, `ensure --archive` sin
// `--fetch` sale con 1 y `change_not_integrated`, nombra `origin/main` y sugiere `--fetch` con una
// orden ejecutable, sin crear nada; con `--fetch` sale con 0, crea `feat/x-archive` desde el commit
// traído con su worktree y `fetched` verdadero; repetirlo da el mismo JSON byte a byte, el mismo
// estado de ramas y worktrees y no publica la rama; en texto, la salida nombra la rama de archivado.
func TestBranchEnsureArchiveRefusesWithoutFetchAndCreatesTheBranchWithIt(t *testing.T) {
	repo := newBranchRepository(t)
	remote := t.TempDir()
	branchTestGit(t, remote, "init", "-q", "--bare", "-b", "main")
	branchTestGit(t, repo, "remote", "add", "origin", remote)
	branchTestGit(t, repo, "push", "-q", "origin", "main")
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	branchTestGit(t, scratch, "clone", "-q", remote, clone)
	archiveWriteChange(t, clone, "x")
	branchTestGit(t, clone, "push", "-q", "origin", "main")
	advanced := branchGitOutput(t, clone, "rev-parse", "HEAD")
	branches := branchGitOutput(t, repo, "branch", "--list")
	worktrees := branchGitOutput(t, repo, "worktree", "list", "--porcelain")

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--archive", "--json", "--cwd", repo)

	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	refused := decodeBranchResult(t, stdout)
	if refused.Status != "refused" || refused.Code != "change_not_integrated" || !strings.Contains(refused.Message, "origin/main") {
		t.Errorf("status %q, código %q, mensaje %q", refused.Status, refused.Code, refused.Message)
	}
	if want := "axiom branch ensure --cwd " + pathquote.Quote(repo) + " --change x --archive --fetch"; !strings.Contains(refused.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q", refused.Continuation, want)
	}
	if branchGitOutput(t, repo, "branch", "--list") != branches || branchGitOutput(t, repo, "worktree", "list", "--porcelain") != worktrees {
		t.Error("el rechazo no debe crear ramas ni worktrees")
	}

	code, first, stderr := runBranchCLI(t, "ensure", "--change", "x", "--archive", "--fetch", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío\n%s", code, stderr, first)
	}
	result := decodeBranchResult(t, first)
	if result.Operation != "ensure" || result.Status != "ready" || result.Change != "x" || result.Code != "" {
		t.Errorf("operación %q, status %q, cambio %q, código %q", result.Operation, result.Status, result.Change, result.Code)
	}
	repository := onlyBranchRepository(t, result)
	if repository.Base == nil || repository.Base.Ref != "origin/main" || repository.Base.SHA != advanced || !repository.Base.Fetched {
		t.Errorf("base = %+v, se esperaba origin/main en %s con fetched=true", repository.Base, advanced)
	}
	if len(repository.Branches) != 1 {
		t.Fatalf("ramas = %+v, se esperaba solo la de archivado", repository.Branches)
	}
	archive := repository.Branches[0]
	if archive.Name != "feat/x-archive" || archive.Kind != "archive" || archive.KindLabel != "rama de archivado" || !archive.Exists || archive.PRTarget != "main" {
		t.Errorf("rama = %+v", archive)
	}
	wantBranchDirectory(t, "worktree", archive.Worktree, plannedBranchWorktree(t, repo, "x-archive"))
	if got := branchGitOutput(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != advanced {
		t.Errorf("feat/x-archive apunta a %s, se esperaba el commit traído %s", got, advanced)
	}
	if branchGitOutput(t, repo, "branch", "--list", "feat/x") != "" {
		t.Error("`--archive` no crea la rama del cambio")
	}

	state := branchGitOutput(t, repo, "worktree", "list", "--porcelain") + branchGitOutput(t, repo, "branch", "--list")
	code, second, _ := runBranchCLI(t, "ensure", "--change", "x", "--archive", "--fetch", "--json", "--cwd", repo)
	if code != 0 || second != first {
		t.Errorf("código %d; la repetición cambió la salida:\n%s\n---\n%s", code, first, second)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain") + branchGitOutput(t, repo, "branch", "--list"); got != state {
		t.Errorf("la repetición cambió ramas o worktrees:\n%s\n---\n%s", state, got)
	}
	if listed := branchGitOutput(t, remote, "branch", "--list"); strings.Contains(listed, "archive") {
		t.Errorf("la rama de archivado nunca se publica:\n%s", listed)
	}

	code, text, stderr := runBranchCLI(t, "ensure", "--change", "x", "--archive", "--fetch", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"Cambio: x (carril sdd)", "feat/x-archive (rama de archivado)", "PR hacia main"} {
		if !strings.Contains(text, want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, text)
		}
	}

	// `status` incluye la rama de archivado (REQ-25.15) y la continuación de los pendientes
	// existe como orden de `ensure` (REQ-25.20).
	code, listed, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
	if code != 0 {
		t.Fatalf("status: código %d\n%s", code, listed)
	}
	statusResult := decodeBranchResult(t, listed)
	var names []string
	for _, described := range onlyBranchRepository(t, statusResult).Branches {
		names = append(names, described.Name+":"+described.Kind+":"+described.PRTarget)
	}
	if want := []string{"feat/x:change:main", "feat/x-archive:archive:main"}; strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("status = %v, se esperaba %v", names, want)
	}
}
