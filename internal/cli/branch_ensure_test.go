package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `axiom branch ensure` (INC-25, S3.3): banderas, códigos de salida y salida
// en JSON y en texto. El comportamiento del motor se prueba en internal/branch/engine; aquí
// se comprueba el contrato de la línea de órdenes sobre repositorios reales.

// branchGitOutput ejecuta git en dir y devuelve su salida sin el salto final.
func branchGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// plannedBranchWorktree es el worktree del cambio x que planifica `ensure` para un
// repositorio sin axiom.yaml: `<padre>/<nombre>-wt/<cambio>/<nombre>`, con la grafía de git.
func plannedBranchWorktree(t *testing.T, repo, change string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(filepath.Dir(resolved), filepath.Base(resolved)+"-wt", change, filepath.Base(resolved))
}

// REQ-25.7, REQ-25.8 y REQ-25.2: `ensure --change x --json --cwd <ruta>` crea la rama y el
// worktree, imprime el estado final como `axiom.branch-result/v1` y sale con 0.
func TestBranchEnsureCreatesTheBranchAndTheWorktreeAndExitsWithZero(t *testing.T) {
	repo := newBranchRepository(t)
	head := branchGitOutput(t, repo, "rev-parse", "HEAD")

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Schema != "axiom.branch-result/v1" || result.Operation != "ensure" || result.Status != "ready" || result.StatusLabel != "listo" {
		t.Errorf("schema %q, operation %q, status %q / %q", result.Schema, result.Operation, result.Status, result.StatusLabel)
	}
	if result.Code != "" || result.Message != "" || result.Continuation != "" {
		t.Errorf("un resultado listo no lleva código, mensaje ni continuación: %+v", result)
	}
	if result.Change != "x" || result.Lane != "sdd" || result.Mode != "worktree" || result.ModeLabel == "" || result.ModeSource != "default" || result.ModeSourceLabel == "" {
		t.Errorf("cambio %q, carril %q, modo %q (%q), origen %q (%q)", result.Change, result.Lane, result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
	}
	worktree := plannedBranchWorktree(t, repo, "x")
	wantBranchDirectory(t, "worktreeRoot", result.WorktreeRoot, filepath.Dir(filepath.Dir(worktree)))
	repository := onlyBranchRepository(t, result)
	if repository.Base == nil || repository.Base.Ref != "main" || repository.Base.SHA != head || repository.Base.Source != "local" || repository.Base.Fetched {
		t.Errorf("base = %+v, se esperaba la rama local main en %s sin fetch", repository.Base, head)
	}
	if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/x" || !repository.Branches[0].Exists {
		t.Fatalf("ramas = %+v, se esperaba feat/x creada", repository.Branches)
	}
	wantBranchDirectory(t, "worktree", repository.Branches[0].Worktree, worktree)
	if got := branchGitOutput(t, worktree, "symbolic-ref", "--short", "HEAD"); got != "feat/x" {
		t.Errorf("el worktree tiene %s, se esperaba feat/x", got)
	}
	if got := branchGitOutput(t, repo, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("el checkout principal pasó a %s", got)
	}
}

// REQ-25.7: dos ejecuciones dan el mismo JSON byte a byte y el mismo
// `git worktree list --porcelain`; el resultado de texto también es estable.
func TestBranchEnsureIsIdempotent(t *testing.T) {
	repo := newBranchRepository(t)

	_, first, _ := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)
	worktrees := branchGitOutput(t, repo, "worktree", "list", "--porcelain")
	branches := branchGitOutput(t, repo, "branch", "--list")
	code, second, _ := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)

	if code != 0 || first == "" || first != second {
		t.Errorf("código %d; la salida cambió entre ejecuciones:\n%s\n---\n%s", code, first, second)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain"); got != worktrees {
		t.Errorf("git worktree list cambió:\n%s\n---\n%s", worktrees, got)
	}
	if got := branchGitOutput(t, repo, "branch", "--list"); got != branches {
		t.Errorf("git branch --list cambió:\n%s\n---\n%s", branches, got)
	}
	_, text, _ := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo)
	_, textAgain, _ := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo)
	if text == "" || text != textAgain {
		t.Errorf("la salida de texto debe ser estable:\n%s\n---\n%s", text, textAgain)
	}
}

// Sin `--json` la salida es texto en castellano con el estado final.
func TestBranchEnsurePrintsSpanishTextWithoutJSON(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"Cambio: x (carril sdd)", "Estado: listo", "Modo de trabajo:", "Rama actual: main", "feat/x (rama del cambio)", "worktree " + plannedBranchWorktree(t, repo, "x"), "PR hacia main"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"schema"`) {
		t.Errorf("sin --json la salida no es JSON:\n%s", stdout)
	}
}

// REQ-25.13: `--lane odd` se admite y el resultado nombra el documento de la feature.
func TestBranchEnsureAcceptsTheOddLane(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, _ := runBranchCLI(t, "ensure", "--change", "mi-feature", "--lane", "odd", "--json", "--cwd", repo)

	if code != 0 {
		t.Fatalf("código de salida = %d, se esperaba 0\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Lane != "odd" || !strings.HasSuffix(result.OddDocument, filepath.Join("odd", "tasks", "mi-feature.md")) {
		t.Errorf("carril %q, oddDocument %q", result.Lane, result.OddDocument)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/mi-feature"); got == "" {
		t.Error("feat/mi-feature debía crearse")
	}
}

// REQ-25.2: cada rechazo sale con código 1, un resultado tipado en JSON o un mensaje en
// texto por stderr, y una continuación con la orden de `ensure` y las rutas citadas.
func TestBranchEnsureRefusalsExitWithCode1AndNameTheContinuation(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (cwd string)
		change       string
		wantCode     string
		wantStatus   string
		wantContains []string
	}{
		{
			name:         "nombre de cambio inválido",
			setup:        newBranchRepository,
			change:       "a b:c",
			wantCode:     "invalid_name",
			wantStatus:   "refused",
			wantContains: []string{"axiom branch ensure --change <cambio> --cwd"},
		},
		{
			name: "fuera de un repositorio git",
			setup: func(t *testing.T) string {
				isolateBranchGit(t)
				return branchOutsideGit(t)
			},
			change:       "x",
			wantCode:     "not_a_git_repository",
			wantStatus:   "refused",
			wantContains: []string{"axiom branch ensure --cwd"},
		},
		{
			name: "rama de integración inexistente",
			setup: func(t *testing.T) string {
				repo := newBranchRepository(t)
				writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")
				return repo
			},
			change:       "x",
			wantCode:     "base_not_found",
			wantStatus:   "refused",
			wantContains: []string{"axiom branch ensure --change x --cwd"},
		},
		{
			name: "configuración inválida",
			setup: func(t *testing.T) string {
				repo := newBranchRepository(t)
				writeBranchConfig(t, repo, "git:\n  branch_prefix: \"mi prefijo//\"\n")
				return repo
			},
			change:       "x",
			wantCode:     "config_invalid",
			wantStatus:   "refused",
			wantContains: []string{"axiom workspace validate --path"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cwd := tc.setup(t)

			code, stdout, _ := runBranchCLI(t, "ensure", "--change", tc.change, "--json", "--cwd", cwd)

			if code != 1 {
				t.Errorf("código de salida = %d, se esperaba 1", code)
			}
			result := decodeBranchResult(t, stdout)
			if result.Status != tc.wantStatus || result.Code != tc.wantCode || result.Message == "" {
				t.Errorf("status %q, code %q, mensaje %q; se esperaba %s con %s y mensaje", result.Status, result.Code, result.Message, tc.wantStatus, tc.wantCode)
			}
			for _, want := range append([]string{pathquote.Quote(cwd)}, tc.wantContains...) {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}

			code, stdout, stderr := runBranchCLI(t, "ensure", "--change", tc.change, "--cwd", cwd)
			if code != 1 || stdout != "" || !strings.Contains(stderr, tc.wantCode) || !strings.Contains(stderr, "Continuación:") {
				t.Errorf("en texto: código %d, stdout %q, stderr %q; se esperaba 1, stdout vacío y el código y la continuación por stderr", code, stdout, stderr)
			}
		})
	}
}

// REQ-25.2 y REQ-25.7: un fallo de git es `git_failed` con código 1; aquí git no se puede
// ejecutar. La continuación repite `ensure` con el cambio.
func TestBranchEnsureReportsGitFailedWhenGitCannotRun(t *testing.T) {
	repo := newBranchRepository(t)
	t.Setenv("PATH", t.TempDir())

	code, stdout, _ := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)

	if code != 1 {
		t.Errorf("código de salida = %d, se esperaba 1", code)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "error" || result.Code != "git_failed" || result.Details["operation"] == "" || result.Details["summary"] == "" {
		t.Errorf("status %q, code %q, details %v; se esperaba error / git_failed con la operación y el resumen", result.Status, result.Code, result.Details)
	}
	for _, want := range []string{"axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// Los errores de uso de `ensure` terminan con 2 y nombran la continuación: el cambio es
// obligatorio y no puede estar en blanco, el carril solo admite sdd u odd, y no hay
// argumentos posicionales ni banderas desconocidas.
func TestBranchEnsureUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"sin --change", []string{"ensure"}, "--change"},
		{"--change vacío", []string{"ensure", "--change", ""}, "--change"},
		{"--change solo con espacios", []string{"ensure", "--change", "  "}, "--change"},
		{"--change sin valor", []string{"ensure", "--change"}, "change"},
		{"carril desconocido", []string{"ensure", "--change", "x", "--lane", "xyz"}, "--lane 'xyz'"},
		{"bandera desconocida", []string{"ensure", "--change", "x", "--no-existe"}, "no-existe"},
		{"argumento posicional", []string{"ensure", "--change", "x", "sobra"}, "'sobra'"},
		{"--cwd sin valor", []string{"ensure", "--change", "x", "--cwd"}, "cwd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)
			if code != 2 || stdout != "" {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			if !strings.Contains(stderr, "axiom branch ensure --help") || !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr %q debe nombrar %q y `axiom branch ensure --help`", stderr, tc.want)
			}
		})
	}

	t.Run("ensure --help describe las banderas y que no hay red", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "ensure", "--help")
		if code != 0 || stderr != "" {
			t.Errorf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
		}
		for _, want := range []string{"--change", "--lane", "--cwd", "--json", "--fetch", "worktree", "idempotente", "Ninguna orden hace red sin --fetch", "git fetch origin", "Nunca ejecuta push"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
			}
		}
		if strings.Contains(stdout, "No ejecuta fetch") {
			t.Errorf("la ayuda ya no puede decir que ensure no ejecuta fetch: con --fetch sí lo hace:\n%s", stdout)
		}
	})
}

// newBranchRemote crea un remoto desnudo local, lo registra como `origin` en repo y publica
// main; devuelve su ruta. Es preparación de la prueba: no hay red.
func newBranchRemote(t *testing.T, repo string) string {
	t.Helper()
	remote := t.TempDir()
	branchGitOutput(t, remote, "init", "-q", "--bare", "-b", "main")
	branchGitOutput(t, repo, "remote", "add", "origin", remote)
	branchGitOutput(t, repo, "push", "-q", "origin", "main")
	return remote
}

// REQ-25.10, escenario «`--fetch` es explícito»: `ensure --fetch` ejecuta un fetch antes de
// resolver la base, de modo que contra un remoto desnudo local que avanzó la base alcanza el
// commit nuevo y el JSON devuelve `fetched: true`; sin la bandera, no.
func TestBranchEnsureFetchReachesTheNewRemoteCommit(t *testing.T) {
	repo := newBranchRepository(t)
	remote := newBranchRemote(t, repo)
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	branchGitOutput(t, scratch, "clone", "-q", remote, clone)
	branchGitOutput(t, clone, "commit", "-q", "--allow-empty", "-m", "avance del remoto")
	branchGitOutput(t, clone, "push", "-q", "origin", "main")
	advanced := branchGitOutput(t, clone, "rev-parse", "HEAD")
	stale := branchGitOutput(t, repo, "rev-parse", "refs/remotes/origin/main")

	t.Run("sin --fetch", func(t *testing.T) {
		other := newBranchRepository(t)
		newBranchRemote(t, other)
		code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", other)
		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q", code, stderr)
		}
		if base := onlyBranchRepository(t, decodeBranchResult(t, stdout)).Base; base == nil || base.Fetched || base.Source != "remote-tracking" {
			t.Errorf("base = %+v, se esperaba la referencia remota local sin fetch", base)
		}
	})
	t.Run("con --fetch", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--fetch", "--json", "--cwd", repo)

		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
		}
		base := onlyBranchRepository(t, decodeBranchResult(t, stdout)).Base
		if base == nil || base.Ref != "origin/main" || base.SHA != advanced || base.SHA == stale || base.Source != "remote-tracking" || !base.Fetched {
			t.Errorf("base = %+v, se esperaba origin/main en %s (no en %s) con fetched=true", base, advanced, stale)
		}
		if got := branchGitOutput(t, repo, "rev-parse", "refs/heads/feat/x"); got != advanced {
			t.Errorf("feat/x apunta a %s, se esperaba %s", got, advanced)
		}
	})
}

// REQ-25.10, escenario «`--fetch` fallido no deja efectos»: contra un remoto que no existe,
// código de salida 1, `fetch_failed` en JSON y en texto, y ni rama, ni worktree, ni registro.
func TestBranchEnsureFetchFailedExitsWithCode1AndCreatesNothing(t *testing.T) {
	repo := newBranchRepository(t)
	branchGitOutput(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))

	code, stdout, _ := runBranchCLI(t, "ensure", "--change", "x", "--fetch", "--json", "--cwd", repo)

	if code != 1 {
		t.Errorf("código de salida = %d, se esperaba 1", code)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "refused" || result.Code != "fetch_failed" || result.Message == "" || result.Details["remote"] != "origin" {
		t.Errorf("status %q, code %q, mensaje %q, detalles %v; se esperaba refused / fetch_failed", result.Status, result.Code, result.Message, result.Details)
	}
	for _, want := range []string{"axiom branch ensure --change x --fetch --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	code, out, stderr := runBranchCLI(t, "ensure", "--change", "x", "--fetch", "--cwd", repo)
	if code != 1 || out != "" || !strings.Contains(stderr, "fetch_failed") || !strings.Contains(stderr, "Continuación:") {
		t.Errorf("en texto: código %d, stdout %q, stderr %q; se esperaba 1, stdout vacío y el código y la continuación por stderr", code, out, stderr)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/x"); got != "" {
		t.Errorf("no debía crearse la rama: %q", got)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
		t.Errorf("no debía crearse ningún worktree:\n%s", got)
	}
	record := filepath.Join(filepath.FromSlash(branchGitOutput(t, repo, "rev-parse", "--absolute-git-dir")), "axiom", "branch", "v1", "sdd", "x.json")
	if _, err := os.Stat(record); err == nil {
		t.Errorf("no debía escribirse el registro %s", record)
	}
}
