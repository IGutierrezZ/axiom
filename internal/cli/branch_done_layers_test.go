package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Arnés en proceso de S6.10 y cableado de la sonda de linajes RDD en `axiom branch done` (INC-25,
// S6-c2, REQ-25.16, REQ-25.17, A-4). El arnés llama a `RunBranch`, el mismo punto de entrada que
// `case "branch"` de cmd/axiom/main.go, sobre un repositorio temporal, sin compilar ningún ejecutable.

var branchGitVersion = regexp.MustCompile(`(\d+)\.(\d+)`)

// requireBranchMergeTree omite la prueba con un git anterior a la 2.38 y devuelve la versión
// (`git version`) con la que se ejecuta.
func requireBranchMergeTree(t *testing.T) string {
	t.Helper()
	isolateBranchGit(t)
	output, err := exec.Command("git", "version").Output()
	if err != nil {
		t.Skipf("no se pudo leer la versión de git: %v", err)
	}
	numbers := branchGitVersion.FindStringSubmatch(string(output))
	if numbers == nil {
		t.Skipf("no se entiende la versión de git %q", output)
	}
	major, _ := strconv.Atoi(numbers[1])
	minor, _ := strconv.Atoi(numbers[2])
	if major < 2 || (major == 2 && minor < 38) {
		t.Skipf("la prueba real de la capa del árbol equivalente exige git 2.38 o superior, hay %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}

func branchCommitFile(t *testing.T, dir, name, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(message+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	branchTestGit(t, dir, "add", name)
	branchTestGit(t, dir, "commit", "-q", "-m", message)
}

// Arnés de S6.10: `done --change x --json` tras un squash con `main` avanzado, en un repositorio
// temporal con git 2.38 o superior, confirma la rama por `equivalent_tree`, retira el worktree,
// borra la rama local y el registro, sale con 0 y no hace nada remoto. Registra la versión de git y
// el JSON exacto con t.Log.
func TestBranchDoneHarnessConfirmsASquashByEquivalentTree(t *testing.T) {
	version := requireBranchMergeTree(t)
	repo := newBranchRepository(t)
	if code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo); code != 0 {
		t.Fatalf("ensure: código %d, stderr %q\n%s", code, stderr, stdout)
	}
	worktree := plannedBranchWorktree(t, repo, "x")
	branchCommitFile(t, worktree, "cambio.txt", "cambio de x")
	branchTestGit(t, repo, "merge", "--squash", "feat/x")
	branchTestGit(t, repo, "commit", "-q", "-m", "squash de x")
	branchCommitFile(t, repo, "posterior.txt", "main avanza")
	if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", "feat/x", "main").Run() == nil {
		t.Fatal("la preparación debía dejar a feat/x sin ser ancestro de main")
	}

	code, stdout, stderr := runBranchCLI(t, "done", "--change", "x", "--json", "--cwd", repo)

	t.Logf("%s", version)
	t.Logf("axiom branch done --change x --json (código %d):\n%s", code, stdout)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío con --json", code, stderr)
	}
	result := decodeDoneResult(t, stdout)
	described := onlyDoneBranch(t, result)
	if result.Status != "ready" || described.Outcome != "removed" || described.ConfirmedBy != "equivalent_tree" || described.ConfirmedByLabel != "árbol equivalente" || described.TipSha != "" {
		t.Errorf("estado %q, rama = %+v; se esperaba removed por equivalent_tree", result.Status, described)
	}
	if _, err := os.Stat(worktree); !os.IsNotExist(err) || branchGitOutput(t, repo, "branch", "--list", "feat/x") != "" {
		t.Errorf("el worktree (%v) y la rama local debían eliminarse", err)
	}
	if result.Registry == nil || result.Registry.Present {
		t.Errorf("registry = %+v, sin ramas registradas el registro se borra", result.Registry)
	}
}

// A-4 y REQ-25.17: `done` monta la sonda de linajes RDD (aquí un doble, por la variable que las
// pruebas sustituyen): con un linaje abierto en el worktree de una rama fusionada, el JSON lleva un
// aviso que nombra el linaje y `axiom review abandon`, la eliminación se realiza y sale con 0.
func TestBranchDoneMountsTheLineageProbeAndWarnsOfAnOpenLineage(t *testing.T) {
	repo := newBranchRepository(t)
	if code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--mode", "in_place", "--cwd", repo); code != 0 {
		t.Fatalf("ensure: código %d, stderr %q\n%s", code, stderr, stdout)
	}
	branchTestGit(t, repo, "branch", "-f", "main", "feat/x")
	var asked []string
	previous := newBranchLineageProbe
	newBranchLineageProbe = func() branch.LineageProbe {
		return fixedLineageProbe{lineages: []string{"linaje-abierto"}, asked: &asked}
	}
	t.Cleanup(func() { newBranchLineageProbe = previous })

	code, stdout, stderr := runBranchCLI(t, "done", "--change", "x", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	result := decodeDoneResult(t, stdout)
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "linaje-abierto") || !strings.Contains(result.Warnings[0], "axiom review abandon") {
		t.Errorf("avisos = %v, se esperaba uno que nombre el linaje y `axiom review abandon`", result.Warnings)
	}
	if len(asked) != 1 || branchGitOutput(t, repo, "branch", "--list", "feat/x") != "" {
		t.Errorf("la sonda se consulta una vez (%v) y la rama se elimina", asked)
	}
}

type fixedLineageProbe struct {
	lineages []string
	asked    *[]string
}

func (f fixedLineageProbe) OpenLineages(_ context.Context, worktreeRoot string) ([]string, error) {
	*f.asked = append(*f.asked, worktreeRoot)
	return f.lineages, nil
}
