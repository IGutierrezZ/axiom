package sddtaskresult

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// El plugin de OpenCode (v1 y v2) evalúa en proceso si la sesión está en un proyecto Axiom, porque
// lanzar un subproceso por cada llamada de herramienta sería un modo de fallo nuevo. Eso solo es
// seguro mientras el bloque de TypeScript y workspace.IsAxiomProject no discrepen, así que este
// test ejecuta el bloque real con Node sobre un conjunto de directorios y lo compara con el criterio
// de Go. Un fallo aquí significa que el criterio cambió en un lado: cámbialo en los dos.
//
// Divergencia conocida y aceptada: el bloque no replica las cotas de Locate (techo de pruebas y
// worktree enlazado sin marcador propio, que se resuelve desde su checkout principal). Ante la duda,
// el plugin queda inerte, que es el lado seguro.
const projectGateModulePrelude = `import { readFile, stat } from "node:fs/promises"
import { dirname, isAbsolute, join, resolve } from "node:path"
`

const projectGateHarness = `
export { isAxiomProject }
import { readFileSync } from "node:fs"
const dirs: string[] = JSON.parse(readFileSync(process.argv[2], "utf8"))
const results: boolean[] = []
for (const dir of dirs) results.push(await isAxiomProject(dir))
process.stdout.write(JSON.stringify(results))
`

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeProjectGateMatchesTheGoCriterion(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH; skipping the project gate parity test")
	}

	source := assets.MustRead("opencode/plugins/sdd-task-result-artifacts.ts")
	begin := strings.Index(source, "// axiom-project-gate:begin")
	end := strings.Index(source, "// axiom-project-gate:end")
	if begin < 0 || end < begin {
		t.Fatal("el plugin ya no delimita la guarda de proyecto Axiom con axiom-project-gate:begin/end")
	}
	gate := source[begin:end]

	root := t.TempDir()
	fixture := func(name string) string { return filepath.Join(root, name) }

	// Directorios con respuesta distinta de «no es proyecto».
	writeFixtureFile(t, fixture("axiom-root/axiom.yaml"), "name: fixture\n")
	if err := os.MkdirAll(fixture("axiom-root/sub/deeper"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture("pointer-relative/.axiom-workspace"), "config: repo-specs/axiom.yaml\n")
	writeFixtureFile(t, fixture("pointer-relative/repo-specs/axiom.yaml"), "name: specs\n")
	writeFixtureFile(t, fixture("pointer-quoted/.axiom-workspace"), "config: \"repo-specs/axiom.yaml\" # puntero\r\nspecs: otro\r\n")
	writeFixtureFile(t, fixture("pointer-quoted/repo-specs/axiom.yaml"), "name: specs\n")
	writeFixtureFile(t, fixture("pointer-absolute/.axiom-workspace"), "config: '"+fixture("pointer-relative/repo-specs/axiom.yaml")+"'\n")
	writeFixtureFile(t, fixture("pointer-specs/.axiom-workspace"), "specs: repo-specs\n")
	writeFixtureFile(t, fixture("pointer-specs/repo-specs/axiom.yaml"), "name: specs\n")
	writeFixtureFile(t, fixture("pointer-empty-config/.axiom-workspace"), "config: \"\"\nspecs: repo-specs\n")
	writeFixtureFile(t, fixture("pointer-empty-config/repo-specs/axiom.yaml"), "name: specs\n")
	writeFixtureFile(t, fixture("broken-pointer-under-axiom/axiom.yaml"), "name: fixture\n")
	writeFixtureFile(t, fixture("broken-pointer-under-axiom/inner/.axiom-workspace"), "config: no-existe/axiom.yaml\n")

	// Directorios que no son un proyecto Axiom.
	if err := os.MkdirAll(fixture("plain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture("openspec-only/openspec/changes/x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture("git-only/.git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture("pointer-missing/.axiom-workspace"), "config: no-existe/axiom.yaml\n")
	writeFixtureFile(t, fixture("pointer-specs-missing/.axiom-workspace"), "specs: no-existe\n")
	writeFixtureFile(t, fixture("pointer-no-keys/.axiom-workspace"), "otra: clave\n")
	writeFixtureFile(t, fixture("pointer-config-wins/.axiom-workspace"), "config: no-existe/axiom.yaml\nspecs: repo-specs\n")
	writeFixtureFile(t, fixture("pointer-config-wins/repo-specs/axiom.yaml"), "name: specs\n")
	if err := os.MkdirAll(fixture("axiom-yaml-is-a-directory/axiom.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture("a-file/axiom.yaml"), "name: fixture\n")

	starts := []string{
		fixture("axiom-root"),
		fixture("axiom-root/sub/deeper"),
		fixture("pointer-relative"),
		fixture("pointer-quoted"),
		fixture("pointer-absolute"),
		fixture("pointer-specs"),
		fixture("pointer-empty-config"),
		fixture("broken-pointer-under-axiom/inner"),
		fixture("plain"),
		fixture("openspec-only"),
		fixture("openspec-only/openspec/changes/x"),
		fixture("git-only"),
		fixture("pointer-missing"),
		fixture("pointer-specs-missing"),
		fixture("pointer-no-keys"),
		fixture("pointer-config-wins"),
		fixture("axiom-yaml-is-a-directory"),
		fixture("a-file/axiom.yaml"),
		fixture("does-not-exist"),
	}
	want := make([]bool, len(starts))
	for i, start := range starts {
		_, want[i] = workspace.IsAxiomProject(start)
	}
	// Salvaguarda contra un test vacuo: tiene que haber respuestas de las dos clases.
	var yes, no int
	for _, ok := range want {
		if ok {
			yes++
		} else {
			no++
		}
	}
	if yes < 6 || no < 8 {
		t.Fatalf("el conjunto de directorios ya no cubre las dos clases de respuesta (sí=%d, no=%d)", yes, no)
	}

	dir := t.TempDir()
	module := filepath.Join(dir, "harness.mts")
	writeFixtureFile(t, module, projectGateModulePrelude+gate+projectGateHarness)
	startsJSON, err := json.Marshal(starts)
	if err != nil {
		t.Fatal(err)
	}
	startsPath := filepath.Join(dir, "starts.json")
	writeFixtureFile(t, startsPath, string(startsJSON))

	cmd := exec.Command(node, "--experimental-strip-types", module, startsPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TMPDIR="+dir, "TEMP="+dir, "TMP="+dir)
	output, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		}
		t.Fatalf("node harness failed: %v\n%s", err, stderr)
	}
	var got []bool
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode harness output: %v\nraw output:\n%s", err, output)
	}
	if len(got) != len(want) {
		t.Fatalf("la guarda respondió %d veces, se esperaban %d", len(got), len(want))
	}
	for i, start := range starts {
		if got[i] != want[i] {
			t.Errorf("guarda TypeScript = %v, workspace.IsAxiomProject = %v para %q", got[i], want[i], start)
		}
	}
}
