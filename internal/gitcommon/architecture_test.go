package gitcommon

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// execPointLabel es el único fichero de producción que puede lanzar procesos
// (REQ-25.1, G-7): los demás de `internal/branch/**` y de este paquete ejecutan git
// a través de gitcommon.Run.
const execPointLabel = "internal/gitcommon/exec.go"

// forbiddenImports son los paquetes con los que un fichero lanzaría procesos o
// abriría conexiones de red por su cuenta.
var forbiddenImports = map[string]bool{
	"os/exec":  true,
	"net":      true,
	"net/http": true,
}

// architectureSource es un fichero de producción con su ruta relativa a la raíz del
// repositorio, con barras.
type architectureSource struct {
	label string
	path  string
}

// collectSources recorre un directorio y devuelve sus ficheros Go de producción
// (sin `_test.go`). Un directorio que aún no existe no es un error: `internal/branch`
// llega con las rebanadas siguientes y la prueba debe tolerar su ausencia.
func collectSources(t *testing.T, dir, labelPrefix string) []architectureSource {
	t.Helper()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var sources []architectureSource
	err := filepath.WalkDir(dir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(dir, current)
		if err != nil {
			return err
		}
		sources = append(sources, architectureSource{label: path.Join(labelPrefix, filepath.ToSlash(relative)), path: current})
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", dir, err)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].label < sources[j].label })
	return sources
}

// architectureViolations analiza un fichero y nombra cada forma en que lanzaría un
// proceso o abriría la red: importar os/exec, net o net/http (con o sin alias) o
// llamar a os.StartProcess. El punto de ejecución de git (exec.go) queda exento.
func architectureViolations(label string, source []byte) ([]string, error) {
	if label == execPointLabel {
		return nil, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), label, source, 0)
	if err != nil {
		return nil, err
	}
	var violations []string
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if forbiddenImports[importPath] {
			violations = append(violations, label+" importa "+strconv.Quote(importPath))
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, isIdent := selector.X.(*ast.Ident); isIdent && pkg.Name == "os" && selector.Sel.Name == "StartProcess" {
			violations = append(violations, label+" llama a os.StartProcess")
		}
		return true
	})
	return violations, nil
}

// REQ-25.1, escenario «Una prueba de arquitectura fija el único punto de ejecución
// de git»: ningún fichero de `internal/branch/**` ni de este paquete lanza procesos
// ni abre la red, salvo exec.go.
func TestArchitectureOnlyExecGoLaunchesProcesses(t *testing.T) {
	sources := collectSources(t, ".", "internal/gitcommon")
	sources = append(sources, collectSources(t, filepath.Join("..", "branch"), "internal/branch")...)

	labels := map[string]bool{}
	for _, source := range sources {
		labels[source.label] = true
		content, err := os.ReadFile(source.path)
		if err != nil {
			t.Fatal(err)
		}
		violations, err := architectureViolations(source.label, content)
		if err != nil {
			t.Fatalf("analizar %s: %v", source.label, err)
		}
		for _, violation := range violations {
			t.Errorf("%s: solo %s puede lanzar procesos o abrir la red; ejecuta git con gitcommon.Run", violation, execPointLabel)
		}
	}

	// El recorrido no puede ser vacuo: ve el punto de ejecución y las lecturas de
	// ficheros, y todo `internal/branch/**`, también sus subpaquetes (el motor, las
	// lecturas de git y el registro local: tasks.md S2.30). El punto de ejecución sí
	// importa os/exec (si no, la excepción estaría caducada).
	for _, want := range []string{
		execPointLabel, "internal/gitcommon/files.go",
		"internal/branch/planner.go", "internal/branch/placement.go", "internal/branch/result.go",
		"internal/branch/engine/engine.go", "internal/branch/engine/status.go", "internal/branch/engine/pending.go",
		"internal/branch/gitcli/ops.go",
		"internal/branch/registry/registry.go",
	} {
		if !labels[want] {
			t.Errorf("el recorrido no vio %s; ve %v", want, labels)
		}
	}
	execSource, err := os.ReadFile("exec.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(execSource), `"os/exec"`) {
		t.Errorf("%s ya no importa os/exec: ¿dónde se ejecuta git ahora? Actualiza esta prueba y design D-01", execPointLabel)
	}
}

// El escenario exige que la prueba falle si se añade una ejecución de git fuera del
// punto único: se comprueba con fuentes sintéticas.
func TestArchitectureFlagsAnyLaunchOutsideTheExecPoint(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		source     string
		violations int
	}{
		{"os/exec en gitcli", "internal/branch/gitcli/ops.go", "package gitcli\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n", 1},
		{"os/exec con alias", "internal/branch/engine/engine.go", "package engine\n\nimport run \"os/exec\"\n\nvar _ = run.Command\n", 1},
		{"net/http en el registro", "internal/branch/registry/registry.go", "package registry\n\nimport \"net/http\"\n\nvar _ = http.Get\n", 1},
		{"net en otro fichero de gitcommon", "internal/gitcommon/files.go", "package gitcommon\n\nimport \"net\"\n\nvar _ = net.Dial\n", 1},
		{"os.StartProcess", "internal/branch/planner.go", "package branch\n\nimport \"os\"\n\nvar _ = os.StartProcess\n", 1},
		{"varias a la vez", "internal/branch/x.go", "package branch\n\nimport (\n\t\"net\"\n\t\"os/exec\"\n)\n\nvar _, _ = net.Dial, exec.Command\n", 2},
		{"un fichero limpio", "internal/branch/planner.go", "package branch\n\nimport (\n\t\"os\"\n\t\"strings\"\n)\n\nvar _, _ = os.Getenv, strings.ToUpper\n", 0},
		{"exec.go está exento", execPointLabel, "package gitcommon\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := architectureViolations(tc.label, []byte(tc.source))
			if err != nil {
				t.Fatalf("architectureViolations: %v", err)
			}
			if len(violations) != tc.violations {
				t.Errorf("%s: %d violaciones %v, se esperaban %d", tc.label, len(violations), violations, tc.violations)
			}
		})
	}
}

// Un directorio que aún no existe no es un error del recorrido: `internal/branch/`
// llega con las rebanadas siguientes.
func TestArchitectureWalkToleratesAMissingBranchPackage(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no", "existe")
	if sources := collectSources(t, missing, "internal/branch"); len(sources) != 0 {
		t.Errorf("un directorio inexistente no tiene fuentes, obtenidas %v", sources)
	}
}

// El recorrido ignora las pruebas: files_test.go e exec_test.go importan os/exec
// para montar repositorios reales sin pasar por Run.
func TestArchitectureWalkSkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ops.go"), "package gitcli\n")
	writeFile(t, filepath.Join(dir, "ops_test.go"), "package gitcli\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n")
	writeFile(t, filepath.Join(dir, "sub", "otro.go"), "package sub\n")
	writeFile(t, filepath.Join(dir, "notas.txt"), "no es Go\n")

	var labels []string
	for _, source := range collectSources(t, dir, "internal/branch") {
		labels = append(labels, source.label)
	}
	want := []string{"internal/branch/ops.go", "internal/branch/sub/otro.go"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Errorf("fuentes recorridas %v, se esperaban %v", labels, want)
	}
}

// remoteSubcommand dice si una invocación de git es una de las operaciones
// remotas que `status` no puede lanzar: fetch, pull, push o ls-remote. Salta las
// opciones previas al subcomando (`-c clave=valor`, `-C dir`).
func remoteSubcommand(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-c" || arg == "-C" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		switch arg {
		case "fetch", "pull", "push", "ls-remote":
			return arg, true
		}
		return arg, false
	}
	return "", false
}

func TestArchitectureRemoteSubcommandDetection(t *testing.T) {
	tests := []struct {
		args   []string
		remote bool
	}{
		{[]string{"fetch", "origin"}, true},
		{[]string{"pull"}, true},
		{[]string{"push", "origin", "main"}, true},
		{[]string{"ls-remote", "origin"}, true},
		{[]string{"-c", "core.fsmonitor=false", "fetch"}, true},
		{[]string{"-C", "/ruta", "push"}, true},
		{[]string{"--no-pager", "pull"}, true},
		{[]string{"rev-parse", "--show-toplevel"}, false},
		{[]string{"worktree", "list", "--porcelain"}, false},
		{[]string{"rev-parse", "--verify", "refs/heads/fetch"}, false},
		{nil, false},
	}
	for _, tc := range tests {
		if _, remote := remoteSubcommand(tc.args); remote != tc.remote {
			t.Errorf("remoteSubcommand(%v) = %v, se esperaba %v", tc.args, remote, tc.remote)
		}
	}
}

// REQ-25.1: con un remoto configurado, las lecturas que componen `status` pasan
// por el observador y ninguna es una operación remota.
func TestArchitectureReadsLaunchNoRemoteOperation(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	seen := observe(t)

	if _, err := TopLevel(ctx, repo); err != nil {
		t.Fatalf("TopLevel: %v", err)
	}
	if _, err := CommonDir(ctx, repo); err != nil {
		t.Fatalf("CommonDir: %v", err)
	}
	if _, _, err := ReadHead(repo); err != nil {
		t.Fatalf("ReadHead: %v", err)
	}

	if len(*seen) != 2 {
		t.Fatalf("TopLevel y CommonDir lanzan dos invocaciones de git y ReadHead ninguna; el observador vio %v", *seen)
	}
	for _, invocation := range *seen {
		if subcommand, remote := remoteSubcommand(invocation[1:]); remote {
			t.Errorf("una lectura lanzó la operación remota git %s: %v", subcommand, invocation)
		}
	}
}
