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
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// El plugin de OpenCode (v1 y v2) lee en proceso la topología del proyecto para saber qué grupos del
// pre-vuelo exige (RF-PRE-13), porque lanzar un subproceso por cada llamada de herramienta sería un modo
// de fallo nuevo. Eso solo es seguro mientras el bloque de TypeScript y el criterio del binario no
// discrepen: este test ejecuta el bloque real con Node sobre un conjunto de directorios y lo compara con
// la topología que resuelve el hook de Go (workspace.Locate y Location.GoverningTopology, con
// monorepo-embedded ante un error o ante un proyecto sin topología que aplicar). Un fallo aquí significa
// que el criterio cambió en un lado: cámbialo en los dos.
//
// Divergencia conocida y aceptada: el lector de líneas del plugin no interpreta el estilo de flujo de YAML
// (`workspace: {topology: multirepo}`); lo trata como topología ilegible y rige monorepo-embedded, que es el
// lado seguro (se pregunta de más). Los axiom.yaml que genera Axiom usan estilo de bloque. Tampoco replica
// el techo de pruebas, el límite de un worktree enlazado ni la comprobación de que exista el repositorio de
// especificaciones de un multirepo (con una configuración multirepo cuyo repositorio de especificaciones
// falta, Locate falla y el binario gobierna monorepo-embedded; el plugin lee multirepo). Los fixtures de
// este test crean siempre ese repositorio.
const topologyReaderModulePrelude = `import { readFile, stat } from "node:fs/promises"
import { dirname, isAbsolute, join, resolve } from "node:path"
`

const topologyReaderHarness = `
import { readFileSync } from "node:fs"
const starts: string[] = JSON.parse(readFileSync(process.argv[2], "utf8"))
const results: string[] = []
for (const start of starts) results.push(await readAxiomTopology(start))
process.stdout.write(JSON.stringify(results))
`

// delimitedBlock devuelve el texto de source entre las marcas begin y end (incluidas).
func delimitedBlock(t *testing.T, source, begin, end, what string) string {
	t.Helper()
	b := strings.Index(source, begin)
	e := strings.Index(source, end)
	if b < 0 || e < b || strings.Count(source, begin) != 1 || strings.Count(source, end) != 1 {
		t.Fatalf("el plugin ya no delimita %s con una única pareja %s / %s", what, begin, end)
	}
	return source[b : e+len(end)]
}

// goGoverningTopology es el criterio del hook de Go (sddPreflightHookTopology en internal/cli).
func goGoverningTopology(start string) string {
	loc, err := workspace.Locate(start)
	if err != nil {
		return string(workspace.TopologyMonorepoEmbedded)
	}
	if topology, ok := loc.GoverningTopology(); ok {
		return string(topology)
	}
	return string(workspace.TopologyMonorepoEmbedded)
}

// multirepoFixture crea <root>/<name> con un puntero a <name>/repo-specs/axiom.yaml, que lleva config, y
// devuelve la carpeta maestra.
func multirepoFixture(t *testing.T, root, name, config string) string {
	t.Helper()
	master := filepath.Join(root, name)
	writeFixtureFile(t, filepath.Join(master, ".axiom-workspace"), "config: repo-specs/axiom.yaml\n")
	writeFixtureFile(t, filepath.Join(master, "repo-specs", "axiom.yaml"), config)
	return master
}

const parityRoles = "roles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"

func TestOpenCodeTopologyReaderParity(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH; skipping the topology reader parity test")
	}

	v1 := assets.MustRead("opencode/plugins/sdd-task-result-artifacts.ts")
	v2 := assets.MustRead("opencode/plugins-v2/sdd-task-result-artifacts.ts")
	const gateBegin, gateEnd = "// axiom-project-gate:begin", "// axiom-project-gate:end"
	const topologyBegin, topologyEnd = "// axiom-topology:begin", "// axiom-topology:end"
	gate := delimitedBlock(t, v1, gateBegin, gateEnd, "la guarda de proyecto Axiom")
	topology := delimitedBlock(t, v1, topologyBegin, topologyEnd, "el lector de topología")
	if got := delimitedBlock(t, v2, topologyBegin, topologyEnd, "el lector de topología"); got != topology {
		t.Fatal("el lector de topología difiere entre plugins/ y plugins-v2/; deben llevar el mismo bloque")
	}

	root := t.TempDir()
	fixture := func(name string) string { return filepath.Join(root, name) }

	// Estilo de bloque, comillas, comentarios y finales de línea.
	multirepoBlock := multirepoFixture(t, root, "block-multirepo", "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "block-embedded", "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "quoted-double", "workspace:\n  name: ejemplo\n  topology: \"multirepo\"\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "quoted-single", "workspace:\n  name: ejemplo\n  topology: 'multirepo'\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "comments", "# cabecera\nworkspace: # bloque del workspace\n  # el nombre\n  name: ejemplo\n\n  topology: multirepo # trailing\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "crlf", strings.ReplaceAll("workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n"+parityRoles, "\n", "\r\n"))
	multirepoFixture(t, root, "topology-first", "workspace:\n  topology: multirepo\n  name: ejemplo\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "other-section-first", "other:\n  topology: multirepo\nworkspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "nested-topology", "workspace:\n  name: ejemplo\n  meta:\n    topology: multirepo\n  topology: monorepo-embedded\n  specs_repository: repo-specs\n"+parityRoles)

	// Valor desconocido, sin topología y fichero no cargable: monorepo-embedded.
	multirepoFixture(t, root, "unknown-value", "workspace:\n  name: ejemplo\n  topology: federated\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "no-topology", "workspace:\n  name: ejemplo\n  specs_repository: repo-specs\n"+parityRoles)
	multirepoFixture(t, root, "unreadable", "workspace: [sin cerrar\n")
	multirepoFixture(t, root, "empty", "")
	writeFixtureFile(t, fixture("direct-unknown/axiom.yaml"), "workspace:\n  name: x\n  topology: retirada\n")
	writeFixtureFile(t, fixture("direct-no-topology/axiom.yaml"), "name: fixture\n")

	// Puntero a otro axiom.yaml por `specs`, y puntero absoluto.
	writeFixtureFile(t, fixture("pointer-specs/.axiom-workspace"), "specs: repo-specs\n")
	writeFixtureFile(t, fixture("pointer-specs/repo-specs/axiom.yaml"), "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n"+parityRoles)
	writeFixtureFile(t, fixture("pointer-absolute/.axiom-workspace"), "config: '"+filepath.Join(multirepoBlock, "repo-specs", "axiom.yaml")+"'\n")
	if err := os.MkdirAll(fixture("pointer-absolute/repo-specs"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Un axiom.yaml directo y la resolución desde un subdirectorio.
	embedded := workspacetest.Embedded(t, ".")
	canonical := workspacetest.Canonical(t)

	// Sin marcador, solo openspec/ y rutas inexistentes.
	if err := os.MkdirAll(fixture("plain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture("openspec-only/openspec/changes/x"), 0o755); err != nil {
		t.Fatal(err)
	}
	// openspec/ más cercano que un axiom.yaml multirepo ascendente: no hay topología que aplicar.
	writeFixtureFile(t, fixture("openspec-nearer/axiom.yaml"), "workspace:\n  name: x\n  topology: multirepo\n  specs_repository: repo-specs\n"+parityRoles)
	if err := os.MkdirAll(fixture("openspec-nearer/repo-specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture("openspec-nearer/inner/openspec"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Un puntero roto, también bajo un axiom.yaml multirepo: el puntero más cercano manda y la resolución
	// falla (monorepo-embedded); la guarda de proyecto seguiría ascendiendo.
	writeFixtureFile(t, fixture("broken-pointer/.axiom-workspace"), "config: no-existe/axiom.yaml\n")
	writeFixtureFile(t, fixture("broken-pointer-specs/.axiom-workspace"), "specs: no-existe\n")
	writeFixtureFile(t, fixture("broken-pointer-no-keys/.axiom-workspace"), "otra: clave\n")
	writeFixtureFile(t, fixture("broken-under-multirepo/axiom.yaml"), "workspace:\n  name: x\n  topology: multirepo\n  specs_repository: repo-specs\n"+parityRoles)
	if err := os.MkdirAll(fixture("broken-under-multirepo/repo-specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture("broken-under-multirepo/inner/.axiom-workspace"), "config: no-existe/axiom.yaml\n")

	starts := []string{
		multirepoBlock,
		filepath.Join(multirepoBlock, "repo-specs"),
		fixture("block-embedded"),
		fixture("quoted-double"),
		fixture("quoted-single"),
		fixture("comments"),
		fixture("crlf"),
		fixture("topology-first"),
		fixture("other-section-first"),
		fixture("nested-topology"),
		fixture("unknown-value"),
		fixture("no-topology"),
		fixture("unreadable"),
		fixture("empty"),
		fixture("direct-unknown"),
		fixture("direct-no-topology"),
		fixture("pointer-specs"),
		fixture("pointer-absolute"),
		embedded.Master,
		filepath.Join(embedded.Master, "openspec", "changes"),
		canonical.Master,
		canonical.Backend,
		filepath.Join(canonical.Backend, "src"),
		canonical.Specs,
		fixture("plain"),
		fixture("openspec-only"),
		fixture("openspec-only/openspec/changes/x"),
		fixture("openspec-nearer/inner"),
		fixture("broken-pointer"),
		fixture("broken-pointer-specs"),
		fixture("broken-pointer-no-keys"),
		fixture("broken-under-multirepo/inner"),
		fixture("does-not-exist"),
		filepath.Join(fixture("block-embedded"), ".axiom-workspace"),
	}

	dir := t.TempDir()
	module := filepath.Join(dir, "harness.mts")
	writeFixtureFile(t, module, topologyReaderModulePrelude+gate+"\n"+topology+topologyReaderHarness)

	run := func(cwd string, starts []string) []string {
		t.Helper()
		startsJSON, err := json.Marshal(starts)
		if err != nil {
			t.Fatal(err)
		}
		startsPath := filepath.Join(dir, "starts.json")
		writeFixtureFile(t, startsPath, string(startsJSON))
		cmd := exec.Command(node, "--experimental-strip-types", module, startsPath)
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(), "TMPDIR="+dir, "TEMP="+dir, "TMP="+dir)
		output, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr = exitErr.Stderr
			}
			t.Fatalf("node harness failed: %v\n%s", err, stderr)
		}
		var got []string
		if err := json.Unmarshal(output, &got); err != nil {
			t.Fatalf("decode harness output: %v\nraw output:\n%s", err, output)
		}
		return got
	}

	want := make([]string, len(starts))
	var multirepoCount, monorepoCount int
	for i, start := range starts {
		want[i] = goGoverningTopology(start)
		if want[i] == string(workspace.TopologyMultirepo) {
			multirepoCount++
		} else {
			monorepoCount++
		}
	}
	// Salvaguarda contra un test vacuo: tiene que haber respuestas de las dos clases.
	if multirepoCount < 8 || monorepoCount < 15 {
		t.Fatalf("el conjunto de directorios ya no cubre las dos clases de respuesta (multirepo=%d, monorepo-embedded=%d)", multirepoCount, monorepoCount)
	}
	got := run(dir, starts)
	if len(got) != len(want) {
		t.Fatalf("el lector respondió %d veces, se esperaban %d", len(got), len(want))
	}
	for i, start := range starts {
		if got[i] != want[i] {
			t.Errorf("lector TypeScript = %q, resolución de Go = %q para %q", got[i], want[i], start)
		}
	}

	// Un directorio de partida relativo se resuelve contra el directorio del proceso, como hace Locate.
	relative := []string{".", "repo-specs", "no-existe"}
	t.Chdir(multirepoBlock)
	wantRelative := make([]string, len(relative))
	for i, start := range relative {
		wantRelative[i] = goGoverningTopology(start)
	}
	if wantRelative[0] != string(workspace.TopologyMultirepo) {
		t.Fatalf("la resolución de Go para %q = %q, se esperaba multirepo", relative[0], wantRelative[0])
	}
	gotRelative := run(multirepoBlock, relative)
	for i, start := range relative {
		if gotRelative[i] != wantRelative[i] {
			t.Errorf("lector TypeScript = %q, resolución de Go = %q para el directorio relativo %q", gotRelative[i], wantRelative[i], start)
		}
	}
}
