package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch done` (INC-25, S6, REQ-25.16, REQ-25.17, REQ-25.1): la ayuda, el uso de las
// banderas, el resultado `axiom.branch-result/v1` por rama y la eliminación del worktree y de la
// rama local de un cambio fusionado y el rechazo de uno sin fusionar. El comportamiento del motor
// se prueba en internal/branch/engine.

// Lo que estas pruebas leen del `axiom.branch-result/v1` de `done`, aparte del de `status`: el
// resultado de cada rama (design §4.3).
type doneBranchJSON struct {
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	Exists           bool     `json:"exists"`
	Worktree         string   `json:"worktree"`
	Outcome          string   `json:"outcome"`
	OutcomeLabel     string   `json:"outcomeLabel"`
	ConfirmedBy      string   `json:"confirmedBy"`
	ConfirmedByLabel string   `json:"confirmedByLabel"`
	Discarded        []string `json:"discarded"`
	TipSha           string   `json:"tipSha"`
	Code             string   `json:"code"`
	Message          string   `json:"message"`
	Continuation     string   `json:"continuation"`
}

type doneResultJSON struct {
	Schema       string   `json:"schema"`
	Operation    string   `json:"operation"`
	Status       string   `json:"status"`
	StatusLabel  string   `json:"statusLabel"`
	Code         string   `json:"code"`
	Message      string   `json:"message"`
	Continuation string   `json:"continuation"`
	Change       string   `json:"change"`
	Warnings     []string `json:"warnings"`
	Registry     *struct {
		Present bool `json:"present"`
	} `json:"registry"`
	Repositories []struct {
		Name     string           `json:"name"`
		Branches []doneBranchJSON `json:"branches"`
	} `json:"repositories"`
}

func decodeDoneResult(t *testing.T, stdout string) doneResultJSON {
	t.Helper()
	var result doneResultJSON
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("la salida no es un axiom.branch-result/v1 reconocible: %v\n%s", err, stdout)
	}
	return result
}

func onlyDoneBranch(t *testing.T, result doneResultJSON) doneBranchJSON {
	t.Helper()
	if len(result.Repositories) != 1 || len(result.Repositories[0].Branches) != 1 {
		t.Fatalf("se esperaba un único repositorio con una única rama: %+v", result.Repositories)
	}
	return result.Repositories[0].Branches[0]
}

func TestBranchDoneHelpDocumentsTheSafetyRules(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "done", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{
		"[--force]", "[--fetch]", "  --force ", "ancestry", "confirmedBy", "merge_not_confirmed", "dirty_worktree", "tipSha",
		"in_place", "--cwd", "checkout principal", "nunca toca una rama remota", "partial", "idempotente",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de done debe documentar %q:\n%s", want, stdout)
		}
	}
}

// REQ-25.1 y REQ-25.2: un error de uso sale con 2, no escribe en stdout y nombra la continuación.
func TestBranchDoneUsageErrorsExitWithCode2(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"sin --change":               {[]string{"done"}, "falta --change"},
		"--change en blanco":         {[]string{"done", "--change", "  "}, "falta --change"},
		"carril desconocido":         {[]string{"done", "--change", "x", "--lane", "otro"}, "--lane 'otro'"},
		"argumento inesperado":       {[]string{"done", "--change", "x", "sobra"}, "argumento inesperado 'sobra'"},
		"bandera desconocida":        {[]string{"done", "--change", "x", "--push"}, "-push"},
		"--repo no es de este verbo": {[]string{"done", "--change", "x", "--repo", "."}, "-repo"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)

			if code != 2 || stdout != "" {
				t.Fatalf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			for _, want := range []string{tc.want, "axiom branch done --help"} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr %q debe contener %q", stderr, want)
				}
			}
		})
	}
}

// Harness de S6.5: `axiom branch done --change x --json` tras una fusión real elimina el worktree
// y la rama local, sale con 0 y confirma la rama por ascendencia; sin fusionar, sale con 1, el
// estado es `refused` con `merge_not_confirmed`, la rama y su worktree quedan intactos y la
// continuación es una orden ejecutable que ofrece `--force`; con `--force` se elimina y el JSON
// lleva la capa `forced` y la punta. En texto, la salida describe cada rama y el error va a stderr.
func TestBranchDoneRemovesAMergedChangeAndRefusesAnUnmergedOne(t *testing.T) {
	repo := newBranchRepository(t)
	for _, change := range []string{"x", "y"} {
		if code, stdout, stderr := runBranchCLI(t, "ensure", "--change", change, "--cwd", repo); code != 0 {
			t.Fatalf("ensure %s: código %d, stderr %q\n%s", change, code, stderr, stdout)
		}
	}
	mergedWorktree, unmergedWorktree := plannedBranchWorktree(t, repo, "x"), plannedBranchWorktree(t, repo, "y")
	branchTestGit(t, mergedWorktree, "commit", "-q", "--allow-empty", "-m", "trabajo de x")
	branchTestGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusiona x", "feat/x")
	// Con cambios reales: un commit vacío tendría el mismo árbol que `main` y la capa de árbol
	// equivalente lo daría por fusionado.
	if err := os.WriteFile(filepath.Join(unmergedWorktree, "y.txt"), []byte("trabajo de y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	branchTestGit(t, unmergedWorktree, "add", "-A")
	branchTestGit(t, unmergedWorktree, "commit", "-q", "-m", "trabajo de y, sin fusionar")

	code, stdout, stderr := runBranchCLI(t, "done", "--change", "x", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	merged := decodeDoneResult(t, stdout)
	if merged.Schema != "axiom.branch-result/v1" || merged.Operation != "done" || merged.Status != "ready" || merged.StatusLabel != "listo" || merged.Change != "x" || merged.Code != "" {
		t.Errorf("esquema %q, operación %q, status %q (%q), cambio %q, código %q", merged.Schema, merged.Operation, merged.Status, merged.StatusLabel, merged.Change, merged.Code)
	}
	described := onlyDoneBranch(t, merged)
	if described.Name != "feat/x" || described.Kind != "change" || described.Exists || described.Outcome != "removed" || described.OutcomeLabel != "eliminada" ||
		described.ConfirmedBy != "ancestry" || described.ConfirmedByLabel != "ascendencia" {
		t.Errorf("rama = %+v", described)
	}
	// El worktree ya no existe: se compara la ruta, no la identidad del directorio.
	if !filepath.IsAbs(described.Worktree) || !strings.EqualFold(filepath.Clean(described.Worktree), filepath.Clean(mergedWorktree)) {
		t.Errorf("worktree retirado = %q, se esperaba la ruta absoluta %q", described.Worktree, mergedWorktree)
	}
	if merged.Registry == nil || merged.Registry.Present {
		t.Errorf("registry = %+v, sin ramas registradas el registro se borra", merged.Registry)
	}
	if _, err := os.Stat(mergedWorktree); !os.IsNotExist(err) {
		t.Errorf("el worktree de x debía desaparecer del disco: %v", err)
	}
	if branchGitOutput(t, repo, "branch", "--list", "feat/x") != "" {
		t.Error("la rama local feat/x debía eliminarse")
	}
	if !strings.Contains(branchGitOutput(t, repo, "worktree", "list", "--porcelain"), "feat/y") || branchGitOutput(t, repo, "branch", "--list", "feat/y") == "" {
		t.Error("done --change x no debe tocar el cambio y")
	}

	state := branchGitOutput(t, repo, "worktree", "list", "--porcelain") + branchGitOutput(t, repo, "branch", "--list")
	code, stdout, stderr = runBranchCLI(t, "done", "--change", "y", "--json", "--cwd", repo)

	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	unmerged := decodeDoneResult(t, stdout)
	if unmerged.Status != "refused" || unmerged.StatusLabel != "rechazado" || unmerged.Code != "merge_not_confirmed" || unmerged.Message == "" {
		t.Errorf("status %q (%q), código %q, mensaje %q", unmerged.Status, unmerged.StatusLabel, unmerged.Code, unmerged.Message)
	}
	kept := onlyDoneBranch(t, unmerged)
	if kept.Outcome != "kept" || kept.Code != "merge_not_confirmed" || kept.ConfirmedBy != "" || !kept.Exists {
		t.Errorf("rama = %+v", kept)
	}
	if want := "axiom branch done --change y --cwd " + pathquote.Quote(repo) + " --force"; !strings.Contains(unmerged.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q", unmerged.Continuation, want)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain") + branchGitOutput(t, repo, "branch", "--list"); got != state {
		t.Errorf("el rechazo no debe cambiar ramas ni worktrees:\n%s\n---\n%s", state, got)
	}

	code, text, errText := runBranchCLI(t, "done", "--change", "y", "--cwd", repo)

	if code != 1 || !strings.Contains(text, "feat/y (rama del cambio): conservada") || !strings.Contains(errText, "Error [merge_not_confirmed]") || !strings.Contains(errText, "Continuación: ") {
		t.Errorf("código %d; la salida de texto debe describir la rama en stdout y el error en stderr:\n%s\n---\n%s", code, text, errText)
	}

	tip := branchGitOutput(t, repo, "rev-parse", "refs/heads/feat/y")
	code, stdout, stderr = runBranchCLI(t, "done", "--change", "y", "--force", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("--force: código %d, stderr %q\n%s", code, stderr, stdout)
	}
	forced := onlyDoneBranch(t, decodeDoneResult(t, stdout))
	if forced.Outcome != "removed" || forced.ConfirmedBy != "forced" || forced.ConfirmedByLabel != "forzado" || forced.TipSha != tip {
		t.Errorf("rama = %+v, se esperaba la capa forced y la punta %s", forced, tip)
	}
	if _, err := os.Stat(unmergedWorktree); !os.IsNotExist(err) || branchGitOutput(t, repo, "branch", "--list", "feat/y") != "" {
		t.Errorf("con --force se eliminan el worktree (%v) y la rama de y", err)
	}
}

// REQ-25.17 (rechazos previos): un nombre de cambio inválido y un directorio que no es un
// repositorio salen con 1 y un código tipado, sin datos de ninguna rama.
func TestBranchDoneRefusesAnInvalidChangeNameAndAnUnusableDirectory(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "done", "--change", "../x", "--json", "--cwd", repo)
	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q\n%s", code, stderr, stdout)
	}
	if invalid := decodeDoneResult(t, stdout); invalid.Operation != "done" || invalid.Code != "invalid_name" || len(invalid.Repositories) != 0 {
		t.Errorf("operación %q, código %q, repositorios %v", invalid.Operation, invalid.Code, invalid.Repositories)
	}

	code, stdout, stderr = runBranchCLI(t, "done", "--change", "x", "--cwd", branchOutsideGit(t))
	if code != 1 || stdout != "" || !strings.Contains(stderr, "Error [not_a_git_repository]") {
		t.Errorf("código %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
