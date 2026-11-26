package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
)

// Diferido por rol en la CLI (C-07-f, RF-BRA-33 modificado): `ensure --role` pasa el rol al motor,
// que crea la rama pendiente de ese rol; `status` y `ensure` la listan como `pendingBranches`
// (aditivo) con etiquetas en castellano. El doble se instala con integrationstest.Install y no se
// mezcla con el de Azure DevOps.

// taskTemplateProject monta un multirepo con el rol core en repo-backend, cuya rama se nombra con
// `{task_ref}`, y el cambio x sellado con su instantánea. Devuelve la carpeta maestra, el
// repositorio de specs y el de código.
func taskTemplateProject(t *testing.T) (master, specs, backend string) {
	t.Helper()
	isolateBranchGit(t)
	integrationstest.Install(t, integrationstest.NewFake())
	quiet := writerBranchWarnings
	writerBranchWarnings = discardWriter{}
	t.Cleanup(func() { writerBranchWarnings = quiet })
	master = t.TempDir()
	for _, dir := range []string{"repo-specs", "repo-backend"} {
		if err := os.MkdirAll(filepath.Join(master, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		initBranchRepository(t, filepath.Join(master, dir))
	}
	config := "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n" +
		"git:\n  integration_branch: main\n" +
		"roles:\n  core:\n    name: Core\n    branch_template: \"feature/TASK-{task_ref}\"\n    repositories:\n      - path: repo-backend\n" +
		"integrations:\n  fake:\n    project: p\n"
	if err := os.WriteFile(filepath.Join(master, "axiom.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, backend = filepath.Join(master, "repo-specs"), filepath.Join(master, "repo-backend")
	sealBranchChangeWithSnapshot(t, specs, "x", "core")
	return master, specs, backend
}

func bindCoreTask(t *testing.T, specs, id string) {
	t.Helper()
	body := "schema: axiom.integrations/v1\nchange: x\nlinks:\n" +
		"  - {plugin: fake, kind: task, id: \"" + id + "\", role: core, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n"
	if err := os.WriteFile(filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func featureBranches(t *testing.T, repo string) string {
	t.Helper()
	output, err := exec.Command("git", "-C", repo, "branch", "--list", "feature/*").CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v\n%s", err, output)
	}
	return strings.TrimSpace(string(output))
}

func codeRepository(t *testing.T, result branchResultJSON, name string) branchRepositoryJSON {
	t.Helper()
	for _, repository := range result.Repositories {
		if repository.Name == name {
			return repository
		}
	}
	t.Fatalf("no hay repositorio %s: %+v", name, result.Repositories)
	return branchRepositoryJSON{}
}

func TestBranchEnsureRoleCreatesThePendingBranchOfThatRole(t *testing.T) {
	master, specs, backend := taskTemplateProject(t)
	bindCoreTask(t, specs, "1235")

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", master)
	if code != 0 || stderr != "" {
		t.Fatalf("sin rol: %d %q\n%s", code, stderr, stdout)
	}
	pending := codeRepository(t, decodeBranchResult(t, stdout), "repo-backend").PendingBranches
	if len(pending) != 1 || pending[0].Role != "core" || pending[0].Template != "feature/TASK-{task_ref}" || pending[0].Name != "feature/TASK-1235" || pending[0].Reason != "awaiting_role" || pending[0].ReasonLabel == "" {
		t.Fatalf("pendientes = %+v", pending)
	}
	if !strings.Contains(pending[0].Continuation, "branch ensure --change x --role core") {
		t.Errorf("continuación = %q", pending[0].Continuation)
	}
	if listed := featureBranches(t, backend); listed != "" {
		t.Fatalf("sin --role no se crea la rama pendiente: %q", listed)
	}

	code, stdout, stderr = runBranchCLI(t, "ensure", "--change", "x", "--json", "--role", "core", "--cwd", master)
	if code != 0 || stderr != "" {
		t.Fatalf("con rol: %d %q\n%s", code, stderr, stdout)
	}
	created := codeRepository(t, decodeBranchResult(t, stdout), "repo-backend")
	if len(created.PendingBranches) != 0 || len(created.Branches) != 1 || created.Branches[0].Name != "feature/TASK-1235" {
		t.Fatalf("la rama del rol debe crearse: %+v", created)
	}
	if listed := featureBranches(t, backend); !strings.Contains(listed, "feature/TASK-1235") {
		t.Fatalf("la rama no existe: %q", listed)
	}
}

func TestBranchStatusTextListsPendingBranchesInSpanish(t *testing.T) {
	master, _, _ := taskTemplateProject(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--cwd", master)

	if code != 0 || stderr != "" {
		t.Fatalf("status: %d %q\n%s", code, stderr, stdout)
	}
	for _, want := range []string{"Rama pendiente", "del rol core", "plantilla feature/TASK-{task_ref}", "sin nombre todavía", "pendiente de la entrada del rol", "branch ensure --change x --role core"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("falta %q en la salida:\n%s", want, stdout)
		}
	}
}

// Sin la Task vinculada, `ensure --role` con plantilla no crea rama ni worktree ni registro (el
// doble no declara items.create, así que es la plantilla la que rechaza, antes de crear nada).
func TestBranchEnsureRoleWithoutTaskCreatesNothing(t *testing.T) {
	master, specs, backend := taskTemplateProject(t)

	before := refsAndWorktrees(t, specs) + refsAndWorktrees(t, backend)
	code, stdout, _ := runBranchCLI(t, "ensure", "--change", "x", "--json", "--role", "core", "--cwd", master)

	result := decodeBranchResult(t, stdout)
	if code != 1 || result.Code != "branch_template_unresolved" {
		t.Fatalf("código %d, %q\n%s", code, result.Code, stdout)
	}
	if after := refsAndWorktrees(t, specs) + refsAndWorktrees(t, backend); after != before {
		t.Errorf("el rechazo no puede crear ramas ni worktrees:\n%s\n---\n%s", before, after)
	}
}

// refsAndWorktrees resume las ramas y los worktrees de un repositorio.
func refsAndWorktrees(t *testing.T, repo string) string {
	t.Helper()
	return branchGitOutput(t, repo, "branch", "--list") + "|" + branchGitOutput(t, repo, "worktree", "list", "--porcelain")
}
