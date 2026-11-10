package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas de `axiom sdd publish` (DD-10, REQ-27.5, REQ-27.2b, REQ-27.7, REQ-27.9) en proceso, con git
// real, un remoto desnudo temporal local y sin red. Usan el punto de entrada RunSDDPublish.

// newPublishProject crea un monorepo embebido con el cambio x confirmado en main, `origin` apuntando a
// un remoto desnudo que ya tiene main y la rama del cambio `feat/x` desplegada. Devuelve el
// repositorio y el remoto.
func newPublishProject(t *testing.T) (repo, bare string) {
	t.Helper()
	repo = newBranchRepository(t)
	writeBranchConfig(t, repo, "")
	for name, content := range map[string]string{"proposal.md": "propuesta\n", "design.md": "diseño\n"} {
		writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", name), content)
	}
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "cambio x")
	bare = t.TempDir()
	branchTestGit(t, bare, "init", "-q", "--bare", "-b", "main")
	branchTestGit(t, repo, "remote", "add", "origin", bare)
	branchTestGit(t, repo, "push", "-q", "origin", "main")
	branchTestGit(t, repo, "switch", "-q", "-c", "feat/x")
	return repo, bare
}

func writePublishFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func publishGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, err := publishGitOutputIn(dir, args...)
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return output
}

func runPublish(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	var out bytes.Buffer
	err = RunSDDPublish(args, &out)
	return out.String(), err
}

// publishJSON es lo que las pruebas leen del `axiom.sdd-publish/v1`.
type publishJSON struct {
	Schema     string `json:"schema"`
	Change     string `json:"change"`
	Context    string `json:"context"`
	Status     string `json:"status"`
	Repository *struct {
		Name    string `json:"name"`
		GitRoot string `json:"gitRoot"`
	} `json:"repository"`
	Remote  string `json:"remote"`
	Branch  string `json:"branch"`
	InPlace bool   `json:"inPlace"`
	Commit  *struct {
		SHA   string   `json:"sha"`
		Paths []string `json:"paths"`
	} `json:"commit"`
	Push *struct {
		Refspec string `json:"refspec"`
		Created bool   `json:"created"`
	} `json:"push"`
	Resume  []string `json:"resume"`
	Refusal *struct {
		Code         string `json:"code"`
		Label        string `json:"label"`
		Message      string `json:"message"`
		Continuation string `json:"continuation"`
	} `json:"refusal"`
}

func decodePublish(t *testing.T, stdout string) publishJSON {
	t.Helper()
	var result publishJSON
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("la salida no es un axiom.sdd-publish/v1: %v\n%s", err, stdout)
	}
	return result
}

// Los errores de uso salen con error y sin efectos: nada se confirma ni se sube.
func TestSDDPublishUsageErrorsHaveNoEffects(t *testing.T) {
	repo, bare := newPublishProject(t)
	branchTestGit(t, repo, "status")
	writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "editado\n")
	head := publishGitOutput(t, repo, "rev-parse", "HEAD")

	cases := map[string][]string{
		"sin --change":        {"--cwd", repo},
		"--change vacío":      {"--cwd", repo, "--change", ""},
		"--change en blanco":  {"--cwd", repo, "--change", "  "},
		"--change con barra":  {"--cwd", repo, "--change", "a/b"},
		"--change con punto":  {"--cwd", repo, "--change", ".."},
		"bandera desconocida": {"--cwd", repo, "--change", "x", "--force"},
		"argumento suelto":    {"--cwd", repo, "--change", "x", "otro"},
		"--branch en blanco":  {"--cwd", repo, "--change", "x", "--branch", ""},
		"--cwd sin valor":     {"--change", "x", "--cwd"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			stdout, err := runPublish(t, args...)
			if err == nil || errors.Is(err, errPublishRefused) {
				t.Fatalf("se esperaba un error de uso, hay %v", err)
			}
			if stdout != "" {
				t.Errorf("un error de uso no escribe nada en la salida: %q", stdout)
			}
		})
	}
	if publishGitOutput(t, repo, "rev-parse", "HEAD") != head {
		t.Error("los errores de uso no deben confirmar nada")
	}
	if _, err := publishGitOutputIn(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Error("los errores de uso no deben subir nada")
	}
}

func TestSDDPublishHelpDocumentsTheVerb(t *testing.T) {
	stdout, err := runPublish(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"axiom sdd publish", "--cwd", "--change", "--archive", "--branch", "--json", "No sube código"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda no menciona %q:\n%s", want, stdout)
		}
	}
}

// published: salida 0, texto en castellano con commit, push y la continuación de quien siga.
func TestSDDPublishPublishesAndPrintsTheNextStepForWhoFollows(t *testing.T) {
	repo, bare := newPublishProject(t)
	writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "editado\n")

	stdout, err := runPublish(t, "--cwd", repo, "--change", "x")

	if err != nil {
		t.Fatalf("RunSDDPublish: %v\n%s", err, stdout)
	}
	for _, want := range []string{
		"Publicación de x en origin/feat/x (repositorio ",
		"Commit ", "docs(sdd): publicar los documentos de x (1 rutas)",
		"Push: refs/heads/feat/x:refs/heads/feat/x -> origin (rama creada)",
		"Quien siga: `git fetch origin` y `axiom branch ensure --change x --json`.",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, stdout)
		}
	}
	if got := publishGitOutput(t, bare, "rev-parse", "refs/heads/feat/x"); got != publishGitOutput(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
	}

	// up_to_date: salida 0 y sin commit.
	again, err := runPublish(t, "--cwd", repo, "--change", "x")
	if err != nil || !strings.Contains(again, "nada que publicar") || !strings.Contains(again, "Quien siga:") {
		t.Errorf("segunda publicación = %v\n%s", err, again)
	}
}

// `--json` escribe axiom.sdd-publish/v1, también en una negativa, y la negativa devuelve el error que
// hace salir con 1.
func TestSDDPublishJSONIsWrittenAlsoOnARefusal(t *testing.T) {
	repo, _ := newPublishProject(t)
	writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "editado\n")

	stdout, err := runPublish(t, "--cwd", repo, "--change", "x", "--json")
	if err != nil {
		t.Fatalf("RunSDDPublish: %v\n%s", err, stdout)
	}
	published := decodePublish(t, stdout)
	if published.Schema != "axiom.sdd-publish/v1" || published.Status != "published" || published.Context != "change" ||
		published.Remote != "origin" || published.Branch != "feat/x" || published.InPlace || published.Refusal != nil ||
		published.Commit == nil || published.Push == nil || !published.Push.Created || published.Repository == nil {
		t.Errorf("publicación = %+v\n%s", published, stdout)
	}

	// Una negativa: la rama atada no es la actual.
	stdout, err = runPublish(t, "--cwd", repo, "--change", "x", "--branch", "feat/otra", "--json")
	if !errors.Is(err, errPublishRefused) {
		t.Fatalf("error = %v, se esperaba errPublishRefused", err)
	}
	if !strings.Contains(err.Error(), "wrong_branch") {
		t.Errorf("el error %q debe llevar la línea resumen con el código", err)
	}
	refused := decodePublish(t, stdout)
	if refused.Status != "refused" || refused.Refusal == nil || refused.Refusal.Code != "wrong_branch" || refused.Refusal.Label == "" || refused.Refusal.Continuation == "" {
		t.Errorf("negativa = %+v\n%s", refused.Refusal, stdout)
	}
}

// Sin `--json`, una negativa escribe su etiqueta, su mensaje y su continuación en la salida, y el commit
// local que se conserva.
func TestSDDPublishRefusalTextNamesTheKeptCommit(t *testing.T) {
	repo, _ := newPublishProject(t)
	writePublishFile(t, filepath.Join(repo, "README.sh"), "#!/bin/sh\n")
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "feat: código")
	writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "editado\n")

	stdout, err := runPublish(t, "--cwd", repo, "--change", "x")

	if !errors.Is(err, errPublishRefused) {
		t.Fatalf("error = %v, se esperaba errPublishRefused", err)
	}
	for _, want := range []string{"rechazada", "code_commits_unpublished", "commits de código sin subir", "README.sh", "Continuación:", "git -C", "se conserva en local"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, stdout)
		}
	}
}

// Matriz «Selección del repositorio git»: `--cwd` relativo se resuelve con filepath.Abs y Locate, también
// desde una subcarpeta del proyecto.
func TestPublishRelativeCwdResolvesToWorkspace(t *testing.T) {
	repo, bare := newPublishProject(t)
	writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "design.md"), "editado\n")
	t.Chdir(filepath.Join(repo, "openspec", "changes", "x"))

	stdout, err := runPublish(t, "--cwd", ".", "--change", "x", "--json")

	if err != nil {
		t.Fatalf("RunSDDPublish: %v\n%s", err, stdout)
	}
	published := decodePublish(t, stdout)
	if published.Status != "published" || published.Repository == nil || !pathidentity.SameDirectory(published.Repository.GitRoot, repo) {
		t.Fatalf("publicación = %+v, se esperaba published en la raíz git %s", published, repo)
	}
	if got := publishGitOutput(t, bare, "rev-parse", "refs/heads/feat/x"); got != publishGitOutput(t, repo, "rev-parse", "HEAD") {
		t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
	}
}

// REQ-27.14: tras archivar, la rama de archivado solo se publica con `--archive`, y sin él la negativa
// nombra el flag que falta.
func TestSDDPublishArchiveBranchRequiresTheArchiveFlag(t *testing.T) {
	repo, _ := newPublishProject(t)
	branchTestGit(t, repo, "switch", "-q", "main")
	branchTestGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	changes := filepath.Join(repo, "openspec", "changes")
	if err := os.MkdirAll(filepath.Join(changes, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(changes, "x"), filepath.Join(changes, "archive", "2026-11-09-x")); err != nil {
		t.Fatal(err)
	}

	stdout, err := runPublish(t, "--cwd", repo, "--change", "x", "--json")
	if err == nil {
		t.Fatalf("sin --archive la rama de archivado no se publica:\n%s", stdout)
	}
	if message := err.Error() + stdout; !strings.Contains(message, "--archive") {
		t.Errorf("la negativa debe nombrar `--archive`:\nerror: %v\nsalida: %s", err, stdout)
	}

	stdout, err = runPublish(t, "--cwd", repo, "--change", "x", "--archive", "--json")
	if err != nil {
		t.Fatalf("RunSDDPublish --archive: %v\n%s", err, stdout)
	}
	if published := decodePublish(t, stdout); published.Status != "published" || published.Context != "archive" || published.Branch != "feat/x-archive" {
		t.Errorf("publicación = %+v", published)
	}
}

// publishGitOutputIn ejecuta git directamente en dir, sin pasar por el producto, y devuelve su salida
// sin el salto final.
func publishGitOutputIn(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	return strings.TrimRight(string(output), "\r\n"), err
}
