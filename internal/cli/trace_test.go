package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqtrace"
)

// C-03-d: `axiom trace <ID>` es de solo lectura. Los nombres de las pruebas empiezan por
// TestAxiomTrace para entrar en el mismo patrón de `-run`.

// axiomTrace ejecuta `axiom trace` en proceso y devuelve su salida y su error.
func axiomTrace(args ...string) (string, error) {
	var out bytes.Buffer
	err := RunTrace(args, &out)
	return out.String(), err
}

// axiomTraceProject es un repositorio git con un proyecto Axiom mínimo (sin cambios archivados).
func axiomTraceProject(t *testing.T, blocks string) string {
	t.Helper()
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, blocks)
	writeCLIFixture(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")
	return repo
}

func TestAxiomTraceRejectsBadArgumentsWithoutGit(t *testing.T) {
	repo := axiomTraceProject(t, "")
	notDir := filepath.Join(repo, "README.md")
	outside := t.TempDir()
	calls := observeGit(t)

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"sin ID":                       {nil, "requiere un ID"},
		"solo opciones":                {[]string{"--json"}, "requiere un ID"},
		"varios posicionales":          {[]string{"REQ-1.1", "REQ-1.2"}, "argumento inesperado"},
		"posicional tras opciones":     {[]string{"--json", "REQ-1.1", "extra"}, "argumento inesperado"},
		"opción desconocida":           {[]string{"REQ-1.1", "--nope"}, "axiom trace --help"},
		"rama sin valor":               {[]string{"REQ-1.1", "--integration-branch"}, "axiom trace --help"},
		"rama vacía":                   {[]string{"REQ-1.1", "--integration-branch", ""}, "--integration-branch"},
		"rama con ..":                  {[]string{"REQ-1.1", "--integration-branch", "../x"}, "--integration-branch"},
		"rama con a..b":                {[]string{"REQ-1.1", "--integration-branch", "a..b"}, "--integration-branch"},
		"rama con guion":               {[]string{"REQ-1.1", "--integration-branch", "-x"}, "--integration-branch"},
		"rama con ^":                   {[]string{"REQ-1.1", "--integration-branch", "x^"}, "--integration-branch"},
		"rama con :":                   {[]string{"REQ-1.1", "--integration-branch", "x:y"}, "--integration-branch"},
		"--cwd inexistente":            {[]string{"REQ-1.1", "--cwd", filepath.Join(repo, "no-existe")}, "--cwd"},
		"--cwd es un fichero":          {[]string{"REQ-1.1", "--cwd", notDir}, "--cwd"},
		"directorio fuera de proyecto": {[]string{"REQ-1.1", "--cwd", outside}, "proyecto Axiom"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := axiomTrace(tc.args...)
			if err == nil {
				t.Fatalf("debió rechazarse; salida:\n%s", out)
			}
			if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "`axiom ") {
				t.Fatalf("error = %q; debe contener %q y una continuación `axiom …`", err, tc.want)
			}
			if out != "" {
				t.Fatalf("un rechazo no escribe en stdout: %q", out)
			}
		})
	}
	if len(*calls) != 0 {
		t.Fatalf("se llamó a git antes de rechazar la entrada: %v", *calls)
	}
}

func TestAxiomTraceInvalidIDNamesPatternAndContinuationWithoutGit(t *testing.T) {
	repo := axiomTraceProject(t, "")
	calls := observeGit(t)

	_, err := axiomTrace("RF-USU-01", "--cwd", repo)
	if err == nil {
		t.Fatal("un ID que no cumple el patrón debe rechazarse")
	}
	for _, want := range []string{"RF-USU-01", "REQ-<n>.<m>", "<serie>", "axiom trace --help"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if _, err := axiomTrace("--cwd", repo, "REQ-01.1x"); err == nil {
		t.Fatal("un ID con sufijo debe rechazarse")
	}
	if len(*calls) != 0 {
		t.Fatalf("se llamó a git con un ID inválido: %v", *calls)
	}
}

func TestAxiomTraceProjectPatternDecidesTheID(t *testing.T) {
	repo := axiomTraceProject(t, "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	if _, err := axiomTrace("RF-USU-01", "--cwd", repo); err != nil {
		t.Fatalf("el ID del patrón del proyecto debe aceptarse: %v", err)
	}
	if _, err := axiomTrace("REQ-1.1", "--cwd", repo); err == nil || !strings.Contains(err.Error(), "RF-<DOM>-<NN>") {
		t.Fatalf("el patrón por defecto no vale en este proyecto: %v", err)
	}
}

func TestAxiomTraceHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"REQ-1.1", "--help"}} {
		out, err := axiomTrace(args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		for _, want := range []string{"Uso: axiom trace <ID>", "--json", "--integration-branch", "--cwd", reqtrace.Schema, "Spec viva"} {
			if !strings.Contains(out, want) {
				t.Errorf("%v: la ayuda no contiene %q:\n%s", args, want, out)
			}
		}
	}
}

func TestAxiomTraceIDBeforeOrAfterOptionsAndEmptyResult(t *testing.T) {
	repo := axiomTraceProject(t, "")
	for _, args := range [][]string{
		{"REQ-1.1", "--json", "--cwd", repo},
		{"--json", "--cwd", repo, "REQ-1.1"},
		{"--cwd", repo, "REQ-1.1", "--json"},
	} {
		out, err := axiomTrace(args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%v: la salida no es un único JSON: %v\n%s", args, err, out)
		}
		if doc["schema"] != reqtrace.Schema || doc["id"] != "REQ-1.1" || doc["integrationBranch"] != "main" {
			t.Fatalf("%v: documento = %v", args, doc)
		}
		for _, key := range []string{"increments", "warnings", "resolvedIds"} {
			if list, ok := doc[key].([]any); !ok || (key != "resolvedIds" && len(list) != 0) {
				t.Fatalf("%v: %s = %#v", args, key, doc[key])
			}
		}
	}
	out, err := axiomTrace("REQ-1.1", "--cwd", repo)
	if err != nil || !strings.Contains(out, "Ningún cambio SDD archivado ni documento ODD integrado cita REQ-1.1.") {
		t.Fatalf("texto sin apariciones = %q (%v)", out, err)
	}
}

func TestAxiomTraceMissingBranchNamesBranchAndRemedyWithoutFetch(t *testing.T) {
	repo := axiomTraceProject(t, "")
	calls := observeGit(t)
	_, err := axiomTrace("REQ-1.1", "--cwd", repo, "--integration-branch", "no-existe")
	if err == nil {
		t.Fatal("una rama inexistente debe fallar")
	}
	for _, want := range []string{"no-existe", "--integration-branch", "fetch"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	for _, call := range *calls {
		if len(call) > 1 && (call[1] == "fetch" || call[1] == "pull") {
			t.Fatalf("trace no debe ejecutar %v", call)
		}
	}
}

func TestAxiomTraceRenderText(t *testing.T) {
	result := reqtrace.Result{
		Schema: reqtrace.Schema, ID: "REQ-30.1", IntegrationBranch: "main", ResolvedIDs: []string{"REQ-30.1", "prov:trace:1"},
		Increments: []reqtrace.Increment{
			{
				Lane: reqtrace.LaneSDD, Name: "demo", Path: "openspec/changes/archive/2026-11-12-demo", Type: reqtrace.KindIncrement,
				Role: reqtrace.RoleCreated, Capabilities: []string{"pagos"}, Tasks: []reqtrace.Task{},
				Integration: reqtrace.Commit{Commit: "1a2b3c4d5e6f7a8b9c0d", Found: true, Anchor: reqtrace.AnchorCode},
				Assignments: []reqtrace.Assignment{{Provisional: "prov:trace:1", Definitive: "REQ-30.1"}},
			},
			{
				Lane: reqtrace.LaneODD, Name: "cobro", Path: "odd/tasks/cobro.md", Type: reqtrace.KindBug, Role: reqtrace.RoleCovered,
				Tasks:       []reqtrace.Task{{File: "odd/tasks/cobro.md", Line: 12}},
				Integration: reqtrace.Commit{Commit: "9f8e7d6c5b4a3210", Found: true, Anchor: reqtrace.AnchorDocument},
			},
			{
				Lane: reqtrace.LaneSDD, Name: "viejo", Path: "openspec/changes/archive/2026-01-01-viejo", Type: reqtrace.KindIncrement,
				Role: reqtrace.RoleModified, Tasks: []reqtrace.Task{},
			},
		},
		Warnings: []reqtrace.Warning{
			{Code: reqtrace.WarnODDUnparseable, Path: "odd/tasks/viejo.md", Message: "no parsea"},
			{Code: reqtrace.WarnProvisionalAmbiguous, Message: "ambiguo"},
		},
	}
	var out bytes.Buffer
	if err := renderRequirementTrace(&out, result); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"Traza de REQ-30.1 en main",
		"IDs resueltos: REQ-30.1, prov:trace:1",
		"Incrementos (3):",
		"1. sdd demo [increment, created] 1a2b3c4d5e6f (código)",
		"     openspec/changes/archive/2026-11-12-demo",
		"Asignación: prov:trace:1 -> REQ-30.1",
		"2. odd cobro [bug, covered] 9f8e7d6c5b4a (documento)",
		"Cubre: odd/tasks/cobro.md:12",
		"3. sdd viejo [increment, modified] sin commit en la línea --first-parent de main",
		"  - [odd_unparseable] odd/tasks/viejo.md: no parsea",
		"  - [provisional_ambiguous] ambiguo",
		"Nota: en ODD el commit es el primero",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("el texto no contiene %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "1a2b3c4d5e6f7") {
		t.Errorf("el commit debe abreviarse a 12 caracteres:\n%s", text)
	}

	// Sin incrementos ODD no hay nota; sin incrementos hay un mensaje propio.
	result.Increments = result.Increments[:1]
	out.Reset()
	if err := renderRequirementTrace(&out, result); err != nil || strings.Contains(out.String(), "Nota:") {
		t.Fatalf("sin ODD no debe haber nota: %q (%v)", out.String(), err)
	}
	result.Increments = nil
	out.Reset()
	if err := renderRequirementTrace(&out, result); err != nil || !strings.Contains(out.String(), "Ningún cambio SDD archivado ni documento ODD integrado cita REQ-30.1.") {
		t.Fatalf("sin incrementos: %q (%v)", out.String(), err)
	}
}
