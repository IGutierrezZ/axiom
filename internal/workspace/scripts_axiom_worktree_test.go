package workspace

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Arnés en proceso del envoltorio local `scripts/axiom-worktree.{sh,ps1}` (S12.19).
//
// Un `axiom` compilado en el PATH no es viable en este entorno (el EDR borra los
// binarios sueltos), así que cada prueba sustituye `axiom` y `gh` por un resguardo
// que registra sus argumentos y devuelve un JSON fijo con la forma real de
// `axiom.branch-result/v1`. Se comprueba qué órdenes lanza el envoltorio y qué hace
// con lo que devuelven, sobre un repositorio temporal con un remoto desnudo local.
// Ninguna prueba toca el repositorio real ni la red, ni crea worktrees fuera de
// t.TempDir().

type stubBranch struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Exists   bool   `json:"exists"`
	Worktree string `json:"worktree,omitempty"`
	Dirty    bool   `json:"dirty"`
	PRTarget string `json:"prTarget"`
}

type stubRepository struct {
	Name     string       `json:"name"`
	Role     string       `json:"role"`
	Branches []stubBranch `json:"branches"`
}

type stubResult struct {
	Schema       string           `json:"schema"`
	Operation    string           `json:"operation"`
	Status       string           `json:"status"`
	Change       string           `json:"change"`
	Repositories []stubRepository `json:"repositories"`
}

type wrapperFixture struct {
	root, proj, remote, bin, home, log string
	env                                []string
}

func (f *wrapperFixture) git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = f.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newWrapperFixture monta un repositorio `proj` con un commit y un remoto desnudo
// local, copia los scripts reales dentro de `proj/scripts` y prepara el directorio
// de resguardos. Una prueba pasa por aquí solo si git está disponible.
func newWrapperFixture(t *testing.T) *wrapperFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("las pruebas con git real se saltan con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}

	root := t.TempDir()
	f := &wrapperFixture{
		root:   root,
		proj:   filepath.Join(root, "proj"),
		remote: filepath.Join(root, "origin.git"),
		bin:    filepath.Join(root, "bin"),
		home:   filepath.Join(root, "home"),
		log:    filepath.Join(root, "axiom-stub.log"),
	}
	for _, dir := range []string{f.proj, f.bin, f.home, filepath.Join(f.proj, "scripts")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gitConfig := filepath.Join(f.home, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte("[user]\n\tname = t\n\temail = t@example.com\n[init]\n\tdefaultBranch = main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.env = append(os.Environ(),
		"PATH="+f.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+f.home,
		"USERPROFILE="+f.home,
		"GIT_CONFIG_GLOBAL="+gitConfig,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"AXIOM_STUB_LOG="+f.log,
		"AXIOM_STUB_ENSURE_JSON="+filepath.Join(root, "ensure.json"),
		"AXIOM_STUB_STATUS_JSON="+filepath.Join(root, "status.json"),
		"AXIOM_STUB_HELP_EXIT=0",
	)

	f.git(t, f.proj, "init", "-q", "-b", "main", ".")
	if err := os.WriteFile(filepath.Join(f.proj, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.git(t, f.proj, "add", "a.txt")
	f.git(t, f.proj, "commit", "-q", "-m", "init")
	f.git(t, root, "init", "-q", "--bare", "-b", "main", f.remote)
	f.git(t, f.proj, "remote", "add", "origin", f.remote)
	f.git(t, f.proj, "push", "-q", "origin", "main")

	for _, name := range []string{"axiom-worktree.sh", "axiom-worktree.ps1"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", name))
		if err != nil {
			t.Fatalf("no se pudo leer scripts/%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(f.proj, "scripts", name), raw, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *wrapperFixture) writeJSON(t *testing.T, name string, value stubResult) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, name), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// installBashStubs instala `axiom` y `gh` como guiones de bash.
func (f *wrapperFixture) installBashStubs(t *testing.T) {
	t.Helper()
	axiom := `#!/usr/bin/env bash
echo "axiom $*" >> "$AXIOM_STUB_LOG"
if [ "$1 $2" = "branch --help" ]; then exit "${AXIOM_STUB_HELP_EXIT:-0}"; fi
case "$1 $2" in
  "branch ensure") cat "$AXIOM_STUB_ENSURE_JSON" ;;
  "branch status") cat "$AXIOM_STUB_STATUS_JSON" ;;
  "branch done") echo "done ok" ;;
esac
`
	gh := `#!/usr/bin/env bash
echo "gh $*" >> "$AXIOM_STUB_LOG"
exit 0
`
	for name, body := range map[string]string{"axiom": axiom, "gh": gh} {
		if err := os.WriteFile(filepath.Join(f.bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// installPowerShellStubs instala `axiom` y `gh` como guiones .ps1.
func (f *wrapperFixture) installPowerShellStubs(t *testing.T) {
	t.Helper()
	axiom := `Add-Content -LiteralPath $env:AXIOM_STUB_LOG -Value ('axiom ' + ($args -join ' '))
if ($args[0] -eq 'branch' -and $args[1] -eq '--help') { exit [int]$env:AXIOM_STUB_HELP_EXIT }
switch ("$($args[0]) $($args[1])") {
    'branch ensure' { Get-Content -Raw -LiteralPath $env:AXIOM_STUB_ENSURE_JSON }
    'branch status' { Get-Content -Raw -LiteralPath $env:AXIOM_STUB_STATUS_JSON }
    'branch done'   { 'done ok' }
}
exit 0
`
	gh := `Add-Content -LiteralPath $env:AXIOM_STUB_LOG -Value ('gh ' + ($args -join ' '))
exit 0
`
	for name, body := range map[string]string{"axiom.ps1": axiom, "gh.ps1": gh} {
		if err := os.WriteFile(filepath.Join(f.bin, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *wrapperFixture) logText(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(f.log)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

// worktreePath es la ruta que `axiom branch ensure` devolvería para el cambio.
func (f *wrapperFixture) worktreePath(change string) string {
	return filepath.Join(f.root, "proj-wt", change, "proj")
}

func (f *wrapperFixture) run(t *testing.T, name string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = f.proj
	cmd.Env = f.env
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	err := cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		return out.String(), errBuf.String(), exit.ExitCode()
	}
	if err != nil {
		t.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
	return out.String(), errBuf.String(), 0
}

// bashPath devuelve el bash que ejecuta el envoltorio. En Windows se evita el
// `bash.exe` de WSL que resuelve el PATH y se usa el de Git for Windows.
func bashPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if base == "" {
				continue
			}
			for _, candidate := range []string{
				filepath.Join(base, "Git", "bin", "bash.exe"),
				filepath.Join(base, "Programs", "Git", "bin", "bash.exe"),
			} {
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
		t.Skip("no hay Git Bash en este equipo")
	}
	path, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash no está disponible")
	}
	return path
}

func demoChange(f *wrapperFixture, branch, prTarget string, withWorktree bool) stubResult {
	b := stubBranch{Name: branch, Kind: "change", Exists: true, PRTarget: prTarget}
	if withWorktree {
		b.Worktree = f.worktreePath("demo")
	}
	return stubResult{
		Schema: "axiom.branch-result/v1", Status: "ready", Change: "demo",
		Repositories: []stubRepository{{Name: "proj", Role: "change_branch", Branches: []stubBranch{b}}},
	}
}

func TestScriptsAxiomWorktreeShellNewRunsAxiomBranchEnsure(t *testing.T) {
	f := newWrapperFixture(t)
	f.installBashStubs(t)
	bash := bashPath(t)
	f.writeJSON(t, "ensure.json", demoChange(f, "inc/demo", "main", true))

	t.Run("con prefijo", func(t *testing.T) {
		stdout, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "new", "demo", "inc")
		if code != 0 {
			t.Fatalf("new salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, f.worktreePath("demo")) {
			t.Errorf("new no muestra la ruta del worktree %q:\n%s", f.worktreePath("demo"), stdout)
		}
		want := regexp.MustCompile(`(?m)^axiom branch ensure --cwd \S*proj --change demo --mode worktree --prefix inc --json$`)
		if !want.MatchString(f.logText(t)) {
			t.Errorf("new no lanzó `axiom branch ensure --cwd <principal> --change demo --mode worktree --prefix inc --json`:\n%s", f.logText(t))
		}
	})

	t.Run("sin prefijo rige el de axiom.yaml", func(t *testing.T) {
		if err := os.Remove(f.log); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if _, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "new", "demo"); code != 0 {
			t.Fatalf("new salió con %d\nstderr: %s", code, stderr)
		}
		if strings.Contains(f.logText(t), "--prefix") {
			t.Errorf("new sin prefijo no debe pasar --prefix:\n%s", f.logText(t))
		}
	})

	t.Run("no crea ni toca nada por su cuenta", func(t *testing.T) {
		if got := f.git(t, f.proj, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
			t.Errorf("el envoltorio creó un worktree por su cuenta:\n%s", got)
		}
		if got := f.git(t, f.proj, "branch", "--list"); strings.Contains(got, "demo") {
			t.Errorf("el envoltorio creó una rama por su cuenta:\n%s", got)
		}
	})
}

func TestScriptsAxiomWorktreeShellPrTakesTheBaseFromPrTarget(t *testing.T) {
	f := newWrapperFixture(t)
	f.installBashStubs(t)
	bash := bashPath(t)

	worktree := f.worktreePath("demo")
	if err := os.MkdirAll(filepath.Dir(worktree), 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(t, f.proj, "worktree", "add", "-q", "-b", "inc/demo", worktree, "main")
	f.writeJSON(t, "status.json", demoChange(f, "inc/demo", "release", true))

	t.Run("worktree con cambios sin confirmar", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(worktree, "dirty.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "pr", "demo")
		if code == 0 || !strings.Contains(stderr, "cambios sin commitear") {
			t.Fatalf("pr debía rechazar un worktree sucio, salió con %d:\n%s", code, stderr)
		}
		if strings.Contains(f.logText(t), "gh pr create") {
			t.Error("pr abrió un PR con el worktree sucio")
		}
		if err := os.Remove(filepath.Join(worktree, "dirty.txt")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sube la rama y abre el PR hacia prTarget", func(t *testing.T) {
		stdout, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "pr", "demo")
		if code != 0 {
			t.Fatalf("pr salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		log := f.logText(t)
		if !strings.Contains(log, "axiom branch status --cwd") || !strings.Contains(log, "--change demo --json") {
			t.Errorf("pr no consultó `axiom branch status --cwd ... --change demo --json`:\n%s", log)
		}
		if !strings.Contains(log, "--base release --head inc/demo") {
			t.Errorf("pr no tomó la base de prTarget (release):\n%s", log)
		}
		if got := f.git(t, f.remote, "branch", "--list", "inc/demo"); !strings.Contains(got, "inc/demo") {
			t.Errorf("pr no publicó la rama en origin:\n%s", got)
		}
	})
}

func TestScriptsAxiomWorktreeShellDoneRunsAxiomBranchDone(t *testing.T) {
	f := newWrapperFixture(t)
	f.installBashStubs(t)
	bash := bashPath(t)
	f.writeJSON(t, "status.json", demoChange(f, "inc/demo", "main", true))

	stdout, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "done", "demo")
	if code != 0 {
		t.Fatalf("done salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	log := f.logText(t)
	if !regexp.MustCompile(`(?m)^axiom branch done --cwd \S*proj --change demo$`).MatchString(log) {
		t.Errorf("done no lanzó `axiom branch done --cwd <principal> --change demo`:\n%s", log)
	}
	if strings.Contains(log, "--force") {
		t.Errorf("done sin -Force no debe pasar --force:\n%s", log)
	}
	if !strings.Contains(stdout, "desmantelado") {
		t.Errorf("done no confirma el desmontaje:\n%s", stdout)
	}

	if _, _, code := f.run(t, bash, "scripts/axiom-worktree.sh", "done", "demo", "--force"); code != 0 {
		t.Fatalf("done --force salió con %d", code)
	}
	if !strings.Contains(f.logText(t), "--change demo --force") {
		t.Errorf("done --force no pasó --force a axiom:\n%s", f.logText(t))
	}
}

// Una rama creada fuera de `ensure` no está en el registro: `axiom branch status` no ve
// ninguna rama del cambio y `done` no retira nada. El envoltorio no debe darlo por
// desmantelado.
func TestScriptsAxiomWorktreeShellDoneWarnsWhenNoBranchExists(t *testing.T) {
	f := newWrapperFixture(t)
	f.installBashStubs(t)
	bash := bashPath(t)
	absent := demoChange(f, "inc/demo", "main", false)
	absent.Repositories[0].Branches[0].Exists = false
	f.writeJSON(t, "status.json", absent)

	stdout, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", "done", "demo", "fix")
	if code != 0 {
		t.Fatalf("done salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(f.logText(t), "axiom branch done --cwd") {
		t.Errorf("done no lanzó `axiom branch done`:\n%s", f.logText(t))
	}
	if strings.Contains(stdout, "desmantelado") {
		t.Errorf("done anunció el desmontaje sin ninguna rama del cambio:\n%s", stdout)
	}
	for _, want := range []string{"nada que desmantelar", "a mano con git", "no se aplica a done"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("done no avisa %q:\n%s", want, stdout)
		}
	}
}

func TestScriptsAxiomWorktreeShellStopsWithoutAxiomBranch(t *testing.T) {
	f := newWrapperFixture(t)
	f.installBashStubs(t)
	bash := bashPath(t)
	f.env = append(f.env, "AXIOM_STUB_HELP_EXIT=1")

	for _, verb := range []string{"new", "pr", "done"} {
		_, stderr, code := f.run(t, bash, "scripts/axiom-worktree.sh", verb, "demo")
		if code == 0 {
			t.Errorf("%s debía terminar con error sin `axiom branch`", verb)
		}
		if !strings.Contains(stderr, "instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`)") {
			t.Errorf("%s no explica cómo instalar `axiom branch`:\n%s", verb, stderr)
		}
	}
	for _, line := range strings.Split(f.logText(t), "\n") {
		if strings.HasPrefix(line, "axiom branch ensure") || strings.HasPrefix(line, "axiom branch done") {
			t.Errorf("el envoltorio lanzó %q sin comprobar `axiom branch --help`", line)
		}
	}
}

func powerShellPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("el envoltorio de PowerShell se prueba solo en Windows")
	}
	path, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("powershell.exe no está disponible")
	}
	return path
}

func powerShellArgs(script string, args ...string) []string {
	return append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script}, args...)
}

func TestScriptsAxiomWorktreePowerShellDrivesAxiomBranch(t *testing.T) {
	f := newWrapperFixture(t)
	f.installPowerShellStubs(t)
	powershell := powerShellPath(t)
	script := filepath.Join("scripts", "axiom-worktree.ps1")

	worktree := f.worktreePath("demo")
	if err := os.MkdirAll(filepath.Dir(worktree), 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(t, f.proj, "worktree", "add", "-q", "-b", "inc/demo", worktree, "main")
	f.writeJSON(t, "ensure.json", demoChange(f, "inc/demo", "main", true))
	f.writeJSON(t, "status.json", demoChange(f, "inc/demo", "release", true))

	t.Run("new", func(t *testing.T) {
		stdout, stderr, code := f.run(t, powershell, powerShellArgs(script, "new", "demo", "-Prefix", "inc")...)
		if code != 0 {
			t.Fatalf("new salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, worktree) {
			t.Errorf("new no muestra la ruta del worktree %q:\n%s", worktree, stdout)
		}
		want := regexp.MustCompile(`(?m)^axiom branch ensure --cwd \S*proj --change demo --mode worktree --prefix inc --json\s*$`)
		if !want.MatchString(f.logText(t)) {
			t.Errorf("new no lanzó `axiom branch ensure --cwd <principal> --change demo --mode worktree --prefix inc --json`:\n%s", f.logText(t))
		}
	})

	t.Run("pr", func(t *testing.T) {
		stdout, stderr, code := f.run(t, powershell, powerShellArgs(script, "pr", "demo")...)
		if code != 0 {
			t.Fatalf("pr salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(f.logText(t), "--base release --head inc/demo") {
			t.Errorf("pr no tomó la base de prTarget (release):\n%s", f.logText(t))
		}
		if got := f.git(t, f.remote, "branch", "--list", "inc/demo"); !strings.Contains(got, "inc/demo") {
			t.Errorf("pr no publicó la rama en origin:\n%s", got)
		}
	})

	t.Run("done", func(t *testing.T) {
		stdout, stderr, code := f.run(t, powershell, powerShellArgs(script, "done", "demo", "-Force")...)
		if code != 0 {
			t.Fatalf("done salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !regexp.MustCompile(`(?m)^axiom branch done --cwd \S*proj --change demo --force\s*$`).MatchString(f.logText(t)) {
			t.Errorf("done no lanzó `axiom branch done --cwd <principal> --change demo --force`:\n%s", f.logText(t))
		}
		if !strings.Contains(stdout, "desmantelado") {
			t.Errorf("done no confirma el desmontaje:\n%s", stdout)
		}
	})

	t.Run("done sin ninguna rama del cambio", func(t *testing.T) {
		absent := demoChange(f, "inc/demo", "main", false)
		absent.Repositories[0].Branches[0].Exists = false
		f.writeJSON(t, "status.json", absent)
		defer f.writeJSON(t, "status.json", demoChange(f, "inc/demo", "release", true))

		stdout, stderr, code := f.run(t, powershell, powerShellArgs(script, "done", "demo", "-Prefix", "fix")...)
		if code != 0 {
			t.Fatalf("done salió con %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if strings.Contains(stdout, "desmantelado") {
			t.Errorf("done anunció el desmontaje sin ninguna rama del cambio:\n%s", stdout)
		}
		// Con tildes a propósito: el script lleva BOM y PowerShell 5.1 debe mostrarlas bien.
		for _, want := range []string{"no había nada que desmantelar", "a mano con git", "-Prefix no se aplica a done", "escribió 'ensure'"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("done no avisa %q:\n%s", want, stdout)
			}
		}
	})

	t.Run("sin axiom branch", func(t *testing.T) {
		f.env = append(f.env, "AXIOM_STUB_HELP_EXIT=1")
		stdout, stderr, code := f.run(t, powershell, powerShellArgs(script, "new", "demo")...)
		if code == 0 {
			t.Fatal("new debía terminar con error sin `axiom branch`")
		}
		if !strings.Contains(stdout+stderr, "instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`)") {
			t.Errorf("new no explica cómo instalar `axiom branch`:\n%s%s", stdout, stderr)
		}
	})
}
