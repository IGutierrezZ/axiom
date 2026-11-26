package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqtrace"
)

// C-03-d: `axiom trace <ID>` sobre repositorios git reales. Los nombres empiezan por
// TestAxiomTrace para entrar en el mismo patrón de `-run`. Los ayudantes traceRepo, traceGit,
// traceIntegrate y traceDiverge son los de sdd_trace_flow_test.go.

const axTraceIDsYAML = "schema: axiom.requirement-ids/v1\npattern: REQ-<n>.<m>\nassignments:\n" +
	"  - provisional: prov:trace:1\n    definitive: REQ-30.1\n    capability: pagos\n    requirement: Cobro\n"

func axTraceSpec(section, title string) string {
	return "## " + section + " Requirements\n\n### Requirement: " + title + "\n\nEl sistema MUST algo.\n\n#### Scenario: x\n\n- **DADO** a\n"
}

func axTraceODDDoc(feature, section, title string) string {
	return "# " + feature + "\n\n## Objetivo\n\nx\n\n## Spec viva\n\n### Capacidad: pagos\n\n```openspec\n" + axTraceSpec(section, title) + "```\n"
}

// axTraceBranch crea la rama con un commit por mapa de ficheros, añade un commit ajeno a main
// y la integra con la estrategia mode. Devuelve el commit de main tras integrar.
func axTraceBranch(t *testing.T, repo, branch, mode string, commits ...map[string]string) string {
	t.Helper()
	traceGit(t, repo, "checkout", "-q", "main")
	traceGit(t, repo, "checkout", "-q", "-b", branch)
	for i, files := range commits {
		for path, content := range files {
			writeCLIFixture(t, filepath.Join(repo, filepath.FromSlash(path)), content)
		}
		traceGit(t, repo, "add", "-A")
		traceGit(t, repo, "commit", "-q", "-m", branch+" "+string(rune('a'+i)))
	}
	traceDiverge(t, repo, strings.ReplaceAll(branch, "/", "-")+"-ajeno.md")
	return traceIntegrate(t, repo, branch, mode)
}

// axTraceArchive archiva changes/<change> como changes/archive/<folder> en una rama aparte.
func axTraceArchive(t *testing.T, repo, mode, change, folder string, extra map[string]string) string {
	t.Helper()
	traceGit(t, repo, "checkout", "-q", "main")
	branch := "inc/" + change + "-archivo"
	traceGit(t, repo, "checkout", "-q", "-b", branch)
	if err := os.MkdirAll(filepath.Join(repo, "openspec", "changes", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	traceGit(t, repo, "mv", "openspec/changes/"+change, "openspec/changes/archive/"+folder)
	for path, content := range extra {
		writeCLIFixture(t, filepath.Join(repo, filepath.FromSlash(path)), content)
	}
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "archivar "+change)
	traceDiverge(t, repo, change+"-archivo-ajeno.md")
	return traceIntegrate(t, repo, branch, mode)
}

// newAxTraceFixture monta un proyecto con un cambio SDD que añade REQ-30.1, un bug SDD que lo
// cubre y dos documentos ODD (uno que lo modifica y otro que no parsea), integrados con mode.
func newAxTraceFixture(t *testing.T, mode string) string {
	t.Helper()
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "")
	writeCLIFixture(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")
	writeCLIFixture(t, filepath.Join(repo, "odd", "tasks", ".gitkeep"), "")
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "base")

	axTraceBranch(t, repo, "inc/alta", mode,
		map[string]string{
			"openspec/changes/alta/proposal.md":         "# P\n",
			"openspec/changes/alta/specs/pagos/spec.md": axTraceSpec("ADDED", "Cobro (REQ-30.1)"),
			"openspec/changes/alta/tasks.md":            "- [x] 1.1 trabajo\n",
		},
		map[string]string{"openspec/changes/alta/verify-report.md": "# V\n"})
	axTraceArchive(t, repo, mode, "alta", "2026-11-12-alta", map[string]string{"openspec/changes/archive/2026-11-12-alta/ids.yaml": axTraceIDsYAML})

	axTraceBranch(t, repo, "inc/arreglo", mode,
		map[string]string{
			"openspec/changes/arreglo/proposal.md": "# P\n\n## Informe de bug\n\nObservado: x\nEsperado: y\n",
			"openspec/changes/arreglo/tasks.md":    "- [x] 1.1 trabajo\n  Cubre: REQ-30.1\n",
		},
		map[string]string{"openspec/changes/arreglo/verify-report.md": "# V\n"})
	axTraceArchive(t, repo, mode, "arreglo", "2026-11-20-arreglo", nil)

	axTraceBranch(t, repo, "inc/odd", mode, map[string]string{
		"odd/tasks/cobro.md": axTraceODDDoc("cobro", "MODIFIED", "Cobro (REQ-30.1)"),
		"odd/tasks/roto.md":  "# roto\n\n## Spec viva\n\nuno\n\n## Spec viva\n\ndos\n\n- [x] T1\n  Cubre: REQ-30.1\n",
	})
	traceGit(t, repo, "checkout", "-q", "main")
	return repo
}

func axTraceRun(t *testing.T, args ...string) reqtrace.Result {
	t.Helper()
	out, err := axiomTrace(append(args, "--json")...)
	if err != nil {
		t.Fatalf("RunTrace(%v) error = %v\n%s", args, err, out)
	}
	var result reqtrace.Result
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("la salida no es un único JSON: %v\n%s", err, out)
	}
	return result
}

type axRow struct {
	Lane   reqtrace.Lane
	Name   string
	Kind   reqtrace.Kind
	Role   reqtrace.Role
	Anchor reqtrace.Anchor
}

func axRows(result reqtrace.Result) []axRow {
	var rows []axRow
	for _, inc := range result.Increments {
		rows = append(rows, axRow{inc.Lane, inc.Name, inc.Type, inc.Role, inc.Integration.Anchor})
	}
	return rows
}

func axFirstParent(t *testing.T, repo, ref string) map[string]bool {
	t.Helper()
	line := map[string]bool{}
	for _, sha := range strings.Fields(traceGit(t, repo, "rev-list", "--first-parent", ref)) {
		line[sha] = true
	}
	return line
}

func axEqualRows(t *testing.T, got, want []axRow) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("incrementos = %+v\nquería %+v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("incrementos = %+v\nquería %+v", got, want)
		}
	}
}

// Criterio de terminado (prov:trace:15) con merge, squash y rebase: el SDD precede al ODD, el
// bug SDD aparece como bug/covered, y provisional y definitivo dan el mismo resultado.
func TestAxiomTraceSDDAndODDIntegratedWithMergeSquashAndRebase(t *testing.T) {
	want := []axRow{
		{reqtrace.LaneSDD, "alta", reqtrace.KindIncrement, reqtrace.RoleCreated, reqtrace.AnchorCode},
		{reqtrace.LaneSDD, "arreglo", reqtrace.KindBug, reqtrace.RoleCovered, reqtrace.AnchorCode},
		{reqtrace.LaneODD, "cobro", reqtrace.KindIncrement, reqtrace.RoleModified, reqtrace.AnchorDocument},
		{reqtrace.LaneODD, "roto", reqtrace.KindIncrement, reqtrace.RoleCovered, reqtrace.AnchorDocument},
	}
	for _, mode := range []string{"merge", "squash", "rebase"} {
		t.Run(mode, func(t *testing.T) {
			repo := newAxTraceFixture(t, mode)
			// Un documento presente en el árbol pero no integrado se omite con una advertencia.
			writeCLIFixture(t, filepath.Join(repo, "odd", "tasks", "pendiente.md"), axTraceODDDoc("pendiente", "MODIFIED", "Cobro (REQ-30.1)"))

			result := axTraceRun(t, "REQ-30.1", "--cwd", repo)
			if result.Schema != reqtrace.Schema || result.ID != "REQ-30.1" || result.IntegrationBranch != "main" {
				t.Fatalf("cabecera = %+v", result)
			}
			axEqualRows(t, axRows(result), want)
			onLine := axFirstParent(t, repo, "main")
			for _, inc := range result.Increments {
				if !inc.Integration.Found || !onLine[inc.Integration.Commit] {
					t.Errorf("%s %s: el commit %+v no está en la línea --first-parent", inc.Lane, inc.Name, inc.Integration)
				}
				if inc.Lane == reqtrace.LaneSDD && (inc.Archive == nil || !inc.Archive.Found || !onLine[inc.Archive.Commit]) {
					t.Errorf("%s: archiveCommit = %+v", inc.Name, inc.Archive)
				}
			}
			if !strings.HasPrefix(result.Increments[0].Path, "openspec/changes/archive/2026-11-12-alta") ||
				result.Increments[2].Path != "odd/tasks/cobro.md" {
				t.Errorf("rutas = %q y %q", result.Increments[0].Path, result.Increments[2].Path)
			}
			if tasks := result.Increments[1].Tasks; len(tasks) != 1 || tasks[0].Line != 2 ||
				tasks[0].File != "openspec/changes/archive/2026-11-20-arreglo/tasks.md" {
				t.Errorf("tareas del bug = %+v", tasks)
			}
			var codes []string
			for _, w := range result.Warnings {
				codes = append(codes, w.Code+"@"+w.Path)
			}
			if got := strings.Join(codes, ","); got != "odd_not_integrated@odd/tasks/pendiente.md,odd_unparseable@odd/tasks/roto.md" {
				t.Errorf("advertencias = %s", got)
			}

			// Consultar por el provisional da el mismo resultado y la asignación es visible.
			byProv := axTraceRun(t, "prov:trace:1", "--cwd", repo)
			axEqualRows(t, axRows(byProv), want)
			if strings.Join(byProv.ResolvedIDs, ",") != "REQ-30.1,prov:trace:1" || strings.Join(result.ResolvedIDs, ",") != "REQ-30.1,prov:trace:1" {
				t.Errorf("resueltos: %v y %v", result.ResolvedIDs, byProv.ResolvedIDs)
			}
			if a := result.Increments[0].Assignments; len(a) != 1 || a[0].Provisional != "prov:trace:1" || a[0].Definitive != "REQ-30.1" {
				t.Errorf("asignación = %+v", a)
			}

			// Un ID válido sin apariciones: listas vacías y salida 0.
			empty := axTraceRun(t, "REQ-99.9", "--cwd", repo)
			if len(empty.Increments) != 0 || len(empty.Warnings) != 0 {
				t.Errorf("sin apariciones = %+v", empty)
			}
		})
	}
}

func TestAxiomTraceBranchExplicitMissingAndDeterministicOrder(t *testing.T) {
	repo := newAxTraceFixture(t, "merge")
	traceGit(t, repo, "branch", "previo", "main~1")

	t.Run("rama explícita prevalece", func(t *testing.T) {
		result := axTraceRun(t, "REQ-30.1", "--cwd", repo, "--integration-branch", "previo")
		if result.IntegrationBranch != "previo" {
			t.Fatalf("integrationBranch = %q", result.IntegrationBranch)
		}
		for _, inc := range result.Increments {
			if inc.Lane == reqtrace.LaneODD {
				t.Fatalf("los documentos integrados después de previo no deben aparecer: %+v", inc)
			}
		}
		if len(result.Increments) != 2 {
			t.Fatalf("incrementos = %+v", result.Increments)
		}
	})

	t.Run("rama inexistente sin fetch", func(t *testing.T) {
		calls := observeGit(t)
		_, err := axiomTrace("REQ-30.1", "--cwd", repo, "--integration-branch", "no-existe")
		if err == nil || !strings.Contains(err.Error(), "no-existe") || !strings.Contains(err.Error(), "--integration-branch") {
			t.Fatalf("error = %v", err)
		}
		for _, call := range *calls {
			if len(call) > 1 && call[1] == "fetch" {
				t.Fatalf("trace ejecutó fetch: %v", call)
			}
		}
	})

	t.Run("dos ejecuciones idénticas byte a byte", func(t *testing.T) {
		first, err := axiomTrace("REQ-30.1", "--cwd", repo, "--json")
		if err != nil {
			t.Fatal(err)
		}
		second, err := axiomTrace("REQ-30.1", "--cwd", repo, "--json")
		if err != nil || first != second {
			t.Fatalf("las salidas difieren:\n%s\n%s", first, second)
		}
	})
}

// prov:trace:10: solo comandos de lectura y ni el árbol, ni el índice ni la lista de ficheros
// cambian, con y sin --json; con otro repositorio como directorio del proceso, git solo se
// ejecuta en el del proyecto.
func TestAxiomTraceIsReadOnlyAndUsesTheProjectRepository(t *testing.T) {
	repo := newAxTraceFixture(t, "merge")
	writeCLIFixture(t, filepath.Join(repo, "odd", "tasks", "pendiente.md"), axTraceODDDoc("pendiente", "MODIFIED", "Cobro (REQ-30.1)"))
	state := func() [4]string {
		return [4]string{
			traceGit(t, repo, "status", "--porcelain"),
			traceGit(t, repo, "ls-files", "-s"),
			traceGit(t, repo, "rev-parse", "HEAD"),
			strings.Join(listDir(t, filepath.Join(repo, "odd", "tasks")), ","),
		}
	}
	beforeTree, beforeState := treeSnapshotWithoutGit(t, repo), state()

	foreign := traceRepo(t)
	t.Chdir(foreign)
	calls := observeGit(t)
	for _, args := range [][]string{{"REQ-30.1", "--cwd", repo}, {"REQ-30.1", "--cwd", repo, "--json"}} {
		if _, err := axiomTrace(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if len(*calls) == 0 {
		t.Fatal("trace no llamó a git")
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"rev-parse": true, "log": true, "ls-tree": true, "merge-base": true}
	for _, call := range *calls {
		if !allowed[call[1]] {
			t.Errorf("comando de git no permitido: %v", call)
		}
		dir, evalErr := filepath.EvalSymlinks(call[0])
		relative, relErr := filepath.Rel(want, dir)
		if evalErr != nil || relErr != nil || !filepath.IsAbs(call[0]) || strings.HasPrefix(relative, "..") {
			t.Errorf("git se ejecutó en %q (git %v), se esperaba el repositorio del proyecto %q", call[0], call[1:], want)
		}
	}
	if afterState := state(); afterState != beforeState {
		t.Fatalf("el estado de git cambió:\nantes %q\ndespués %q", beforeState, afterState)
	}
	afterTree := treeSnapshotWithoutGit(t, repo)
	if len(afterTree) != len(beforeTree) {
		t.Fatalf("el árbol cambió: %d ficheros antes y %d después", len(beforeTree), len(afterTree))
	}
	for path, content := range beforeTree {
		if afterTree[path] != content {
			t.Fatalf("el fichero %s cambió", path)
		}
	}
}

// Los nombres fuera del juego seguro se omiten con una advertencia y sin llamar a git con ellos.
func TestAxiomTraceUnsafeNamesAreSkippedWithoutGit(t *testing.T) {
	repo := axiomTraceProject(t, "")
	writeCLIFixture(t, filepath.Join(repo, "openspec", "changes", "archive", "2026-01-01-a[1]", "specs", "x", "spec.md"), axTraceSpec("ADDED", "Cobro (REQ-1.1)"))
	for _, name := range []string{"-x.md", "a[1].md", "a b.md"} {
		writeCLIFixture(t, filepath.Join(repo, "odd", "tasks", name), axTraceODDDoc("x", "MODIFIED", "Cobro (REQ-1.1)"))
	}
	calls := observeGit(t)
	result := axTraceRun(t, "REQ-1.1", "--cwd", repo)
	if len(result.Increments) != 0 || len(result.Warnings) != 4 {
		t.Fatalf("resultado = %+v", result)
	}
	for _, w := range result.Warnings {
		if w.Code != reqtrace.WarnUnsafeName {
			t.Errorf("advertencia = %+v", w)
		}
	}
	for _, call := range *calls {
		for _, arg := range call[1:] {
			if strings.Contains(arg, "a[1]") || strings.Contains(arg, "-x.md") || strings.Contains(arg, "a b") {
				t.Errorf("git recibió una ruta no segura: %v", call)
			}
		}
	}
}

// axTraceCommit escribe los ficheros en dir (un repositorio) y los confirma en main.
func axTraceCommit(t *testing.T, dir, message string, files map[string]string) string {
	t.Helper()
	for path, content := range files {
		writeCLIFixture(t, filepath.Join(dir, filepath.FromSlash(path)), content)
	}
	integrateCommit(t, dir, message)
	return integrateGit(t, dir, "rev-parse", "HEAD")
}

// axTraceSpecsHistory escribe un cambio SDD y su archivado en el repositorio specs (con prefix
// de planning home) y devuelve el commit de código.
func axTraceSpecsHistory(t *testing.T, specs, prefix string) string {
	t.Helper()
	axTraceCommit(t, specs, "plan", map[string]string{
		prefix + "changes/demo/specs/pagos/spec.md": axTraceSpec("ADDED", "Cobro (REQ-30.1)"),
		prefix + "changes/demo/tasks.md":            "- [x] 1.1 trabajo\n",
	})
	code := axTraceCommit(t, specs, "verify", map[string]string{prefix + "changes/demo/verify-report.md": "# V\n"})
	if err := os.MkdirAll(filepath.Join(specs, filepath.FromSlash(prefix+"changes/archive")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(specs, filepath.FromSlash(prefix+"changes/demo")), filepath.Join(specs, filepath.FromSlash(prefix+"changes/archive/2026-11-12-demo"))); err != nil {
		t.Fatal(err)
	}
	integrateCommit(t, specs, "archivar")
	return code
}

// prov:trace:14: en un multirepo, specs y odd/tasks están en el repositorio de specs, las rutas
// son relativas a su raíz y git solo se ejecuta en ese repositorio.
func TestAxiomTraceMultirepoAnchorsInTheSpecsRepository(t *testing.T) {
	ws := multirepoSealProject(t)
	code := axTraceSpecsHistory(t, ws.Specs, "openspec/")
	odd := axTraceCommit(t, ws.Specs, "odd", map[string]string{"odd/tasks/cobro.md": axTraceODDDoc("cobro", "MODIFIED", "Cobro (REQ-30.1)")})

	calls := observeGit(t)
	result := axTraceRun(t, "REQ-30.1", "--cwd", ws.Master)
	axEqualRows(t, axRows(result), []axRow{
		{reqtrace.LaneSDD, "demo", reqtrace.KindIncrement, reqtrace.RoleCreated, reqtrace.AnchorCode},
		{reqtrace.LaneODD, "cobro", reqtrace.KindIncrement, reqtrace.RoleModified, reqtrace.AnchorDocument},
	})
	if got := result.Increments[0]; got.Integration.Commit != code || got.Path != "openspec/changes/archive/2026-11-12-demo" {
		t.Errorf("SDD = %+v; commit esperado %s", got, code)
	}
	if got := result.Increments[1]; got.Integration.Commit != odd || got.Path != "odd/tasks/cobro.md" {
		t.Errorf("ODD = %+v; commit esperado %s", got, odd)
	}
	specsRoot, err := filepath.EvalSymlinks(ws.Specs)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range *calls {
		dir, evalErr := filepath.EvalSymlinks(call[0])
		relative, relErr := filepath.Rel(specsRoot, dir)
		if evalErr != nil || relErr != nil || strings.HasPrefix(relative, "..") {
			t.Errorf("git se ejecutó fuera del repositorio de specs: %v", call)
		}
	}
}

// axTraceOpenspecMaster monta una carpeta maestra con `specs_repository: openspec`: el planning
// home es su propio repositorio y odd/tasks cuelga de la carpeta maestra.
func axTraceOpenspecMaster(t *testing.T) (master, specs string) {
	t.Helper()
	isolateIntegrateGit(t)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeCLIFixture(t, filepath.Join(dir, ".axiom-workspace"), "config: axiom.yaml\n")
	writeCLIFixture(t, filepath.Join(dir, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: openspec}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
	specs = filepath.Join(dir, "openspec")
	if err := os.MkdirAll(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	integrateInitRepo(t, specs)
	integrateGit(t, specs, "commit", "-q", "--allow-empty", "-m", "base")
	return dir, specs
}

func TestAxiomTraceMultirepoWithOpenspecRepositoryAnchorsEachSourceInItsOwnHistory(t *testing.T) {
	t.Run("sin git en la carpeta maestra se advierte y se omite ODD", func(t *testing.T) {
		master, specs := axTraceOpenspecMaster(t)
		code := axTraceSpecsHistory(t, specs, "")
		writeCLIFixture(t, filepath.Join(master, "odd", "tasks", "cobro.md"), axTraceODDDoc("cobro", "MODIFIED", "Cobro (REQ-30.1)"))

		result := axTraceRun(t, "REQ-30.1", "--cwd", master)
		axEqualRows(t, axRows(result), []axRow{{reqtrace.LaneSDD, "demo", reqtrace.KindIncrement, reqtrace.RoleCreated, reqtrace.AnchorCode}})
		if got := result.Increments[0]; got.Integration.Commit != code || got.Path != "changes/archive/2026-11-12-demo" {
			t.Errorf("SDD = %+v; commit esperado %s", got, code)
		}
		if len(result.Warnings) != 1 || result.Warnings[0].Code != reqtrace.WarnODDRepoUnavailable {
			t.Errorf("advertencias = %+v", result.Warnings)
		}
	})

	t.Run("con git en la carpeta maestra cada incremento toma el commit de su historia", func(t *testing.T) {
		master, specs := axTraceOpenspecMaster(t)
		code := axTraceSpecsHistory(t, specs, "")
		integrateInitRepo(t, master)
		odd := axTraceCommit(t, master, "odd", map[string]string{
			".gitignore":         "openspec/\n",
			"odd/tasks/cobro.md": axTraceODDDoc("cobro", "MODIFIED", "Cobro (REQ-30.1)"),
		})

		result := axTraceRun(t, "REQ-30.1", "--cwd", master)
		if len(result.Increments) != 2 || len(result.Warnings) != 0 {
			t.Fatalf("resultado = %+v", result)
		}
		var sdd, oddInc reqtrace.Increment
		for _, inc := range result.Increments {
			if inc.Lane == reqtrace.LaneSDD {
				sdd = inc
			} else {
				oddInc = inc
			}
		}
		if sdd.Integration.Commit != code || sdd.Path != "changes/archive/2026-11-12-demo" {
			t.Errorf("SDD = %+v; commit esperado %s", sdd, code)
		}
		if oddInc.Integration.Commit != odd || oddInc.Path != "odd/tasks/cobro.md" {
			t.Errorf("ODD = %+v; commit esperado %s", oddInc, odd)
		}
	})
}
