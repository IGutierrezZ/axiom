package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `axiom branch status` sin `--change` y de los cambios pendientes de archivar
// (REQ-25.15, REQ-25.20): el contrato visible de la CLI, con git real y sin red.

type branchRegisteredBranchJSON struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	KindLabel  string `json:"kindLabel"`
	Role       string `json:"role"`
	Worktree   string `json:"worktree"`
}

type branchRegisteredJSON struct {
	Change          string                       `json:"change"`
	Lane            string                       `json:"lane"`
	Mode            string                       `json:"mode"`
	ModeLabel       string                       `json:"modeLabel"`
	ModeSource      string                       `json:"modeSource"`
	ModeSourceLabel string                       `json:"modeSourceLabel"`
	Branches        []branchRegisteredBranchJSON `json:"branches"`
}

type branchPendingJSON struct {
	Change       string `json:"change"`
	Continuation string `json:"continuation"`
}

// addBranchChanges confirma en la rama actual de repo las carpetas de cambio dadas bajo
// `openspec/changes/`: es como si el PR del cambio ya se hubiera fusionado en main.
func addBranchChanges(t *testing.T, repo string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(repo, "openspec", "changes", name, "proposal.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("propuesta\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "cambios "+strings.Join(names, ", "))
}

func pendingEntries(t *testing.T, result branchResultJSON) []branchPendingJSON {
	t.Helper()
	var entries []branchPendingJSON
	if err := json.Unmarshal(result.PendingArchive, &entries); err != nil || entries == nil {
		t.Fatalf("pendingArchive = %s, se esperaba una lista (%v)", result.PendingArchive, err)
	}
	return entries
}

// REQ-25.15, escenario «`status` sin `--change` lista registrados y pendientes», y
// REQ-25.20, escenario «Un cambio fusionado y sin archivar figura como pendiente»: `x`
// registrado y `y` fusionado y sin archivar; con la continuación para archivarlo y salida 0.
func TestBranchStatusWithoutChangeListsRegisteredAndPendingAsJSON(t *testing.T) {
	repo := newBranchRepository(t)
	addBranchChanges(t, repo, "y")
	record := branchRecord(repo)
	writeBranchRegistry(t, repo, record)

	code, stdout, stderr := runBranchCLI(t, "status", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Schema != "axiom.branch-result/v1" || result.Operation != "status" || result.Status != "ready" || result.StatusLabel != "listo" {
		t.Errorf("schema %q, operación %q, estado %q / %q", result.Schema, result.Operation, result.Status, result.StatusLabel)
	}
	if result.Change != "" || result.Lane != "sdd" || result.Code != "" || result.Continuation != "" || len(result.Repositories) != 0 {
		t.Errorf("cambio %q, carril %q, código %q, continuación %q, repositorios %v; sin --change no hay plan ni negativa", result.Change, result.Lane, result.Code, result.Continuation, result.Repositories)
	}
	wantBranchDirectory(t, "workspaceRoot", result.WorkspaceRoot, repo)
	wantBranchDirectory(t, "planningHome", result.PlanningHome, filepath.Join(repo, "openspec"))

	if len(result.Registered) != 1 {
		t.Fatalf("registered = %+v, se esperaba x", result.Registered)
	}
	registered := result.Registered[0]
	if registered.Change != "x" || registered.Lane != "sdd" || registered.Mode != "worktree" || registered.ModeLabel != "un worktree por rama" ||
		registered.ModeSource != "answer" || registered.ModeSourceLabel != "respuesta registrada" {
		t.Errorf("registrado = %+v", registered)
	}
	if len(registered.Branches) != 1 || registered.Branches[0].Name != "feat/x" || registered.Branches[0].Kind != "change" || registered.Branches[0].KindLabel != "rama del cambio" ||
		registered.Branches[0].Repository != filepath.Base(repo) || registered.Branches[0].Worktree != record.Repositories[0].Branches[0].Worktree {
		t.Errorf("ramas registradas = %+v", registered.Branches)
	}

	pending := pendingEntries(t, result)
	if len(pending) != 1 || pending[0].Change != "y" || pending[0].Continuation != "axiom branch ensure --change y --archive" {
		t.Errorf("pendingArchive = %+v, se esperaba y con la orden para archivarlo", pending)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno", result.Warnings)
	}
}

// REQ-25.15 y REQ-25.20, escenario «`status --change` indica que el cambio solicitado
// está pendiente»: `pendingArchive` es un booleano con `--change`, y el pendiente lleva la
// orden para archivarlo; un cambio que no lo está, no.
func TestBranchStatusWithChangeReportsPendingArchiveAsABoolean(t *testing.T) {
	repo := newBranchRepository(t)
	addBranchChanges(t, repo, "x")

	t.Run("pendiente", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)

		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
		}
		result := decodeBranchResult(t, stdout)
		if string(result.PendingArchive) != "true" || result.PendingArchiveContinuation != "axiom branch ensure --change x --archive" {
			t.Errorf("pendingArchive = %s, continuación %q; se esperaba true con la orden para archivarlo", result.PendingArchive, result.PendingArchiveContinuation)
		}
		if result.Registered != nil {
			t.Errorf("con --change no se listan los registrados: %+v", result.Registered)
		}
		if !strings.Contains(stdout, `"pendingArchive": true`) {
			t.Errorf("`pendingArchive: true` debe imprimirse como booleano:\n%s", stdout)
		}
	})
	t.Run("no pendiente", func(t *testing.T) {
		code, stdout, _ := runBranchCLI(t, "status", "--change", "otro", "--json", "--cwd", repo)

		if code != 0 {
			t.Fatalf("código %d, se esperaba 0\n%s", code, stdout)
		}
		result := decodeBranchResult(t, stdout)
		if string(result.PendingArchive) != "false" || result.PendingArchiveContinuation != "" {
			t.Errorf("pendingArchive = %s, continuación %q; se esperaba false sin continuación", result.PendingArchive, result.PendingArchiveContinuation)
		}
		if !strings.Contains(stdout, `"pendingArchive": false`) {
			t.Errorf("`pendingArchive: false` debe imprimirse explícitamente:\n%s", stdout)
		}
	})
}

// Sin nada registrado ni pendiente, ambas listas se imprimen vacías: «evaluada y sin
// elementos» no es lo mismo que «no evaluada».
func TestBranchStatusWithoutChangeOfAProjectWithNothingToShow(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	for _, want := range []string{`"registered": []`, `"pendingArchive": []`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "axiom")); !os.IsNotExist(err) {
		t.Errorf("`status` es de solo lectura y no debe crear el registro (%v)", err)
	}
}

// Sin la rama de integración la lista de pendientes se omite con un aviso y la salida
// sigue siendo 0 (REQ-25.20): el JSON no trae `pendingArchive`.
func TestBranchStatusWithoutChangeOmitsThePendingListWithoutAnIntegrationBranch(t *testing.T) {
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")

	code, stdout, stderr := runBranchCLI(t, "status", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.PendingArchive != nil {
		t.Errorf("pendingArchive = %s, se esperaba omitido", result.PendingArchive)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "«develop»") || !strings.Contains(result.Warnings[0], "pendientes de archivar") {
		t.Errorf("avisos = %v, se esperaba uno sobre los pendientes que nombre la rama develop", result.Warnings)
	}
}

// Sin --json, el texto en castellano lista los registrados y los pendientes con la orden
// para archivarlos; con `--change`, indica que está pendiente.
func TestBranchStatusWithoutChangePrintsSpanishText(t *testing.T) {
	repo := newBranchRepository(t)
	addBranchChanges(t, repo, "y")
	writeBranchRegistry(t, repo, branchRecord(repo))

	code, stdout, stderr := runBranchCLI(t, "status", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{
		"(carril sdd)", "Cambios registrados: 1", "  - x: un worktree por rama (respuesta registrada)", "feat/x (rama del cambio)",
		"Cambios pendientes de archivar: 1", "  - y: axiom branch ensure --change y --archive",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"schema"`) {
		t.Errorf("sin --json la salida no es JSON:\n%s", stdout)
	}

	_, stdout, _ = runBranchCLI(t, "status", "--change", "y", "--cwd", repo)
	if want := "Pendiente de archivar: sí (axiom branch ensure --change y --archive)"; !strings.Contains(stdout, want) {
		t.Errorf("con --change, la salida de texto debe contener %q:\n%s", want, stdout)
	}
}

// El carril elige qué registros se listan (`--lane odd`).
func TestBranchStatusWithoutChangeListsTheRegisteredChangesOfTheLane(t *testing.T) {
	repo := newBranchRepository(t)
	writeBranchRegistry(t, repo, branchRecord(repo))
	odd := branchRecord(repo)
	odd.Lane, odd.Change, odd.Repositories[0].Branches[0].Name = branch.LaneODD, "mi-feature", "feat/mi-feature"
	writeBranchRegistry(t, repo, odd)

	for lane, want := range map[string]string{"sdd": "x", "odd": "mi-feature"} {
		code, stdout, _ := runBranchCLI(t, "status", "--lane", lane, "--json", "--cwd", repo)
		if code != 0 {
			t.Fatalf("--lane %s: código %d\n%s", lane, code, stdout)
		}
		result := decodeBranchResult(t, stdout)
		if result.Lane != lane || len(result.Registered) != 1 || result.Registered[0].Change != want || result.Registered[0].Lane != lane {
			t.Errorf("--lane %s: carril %q, registrados %+v; se esperaba solo %s", lane, result.Lane, result.Registered, want)
		}
	}
}

// REQ-25.1 y REQ-25.20, escenario «La evaluación no usa la red»: con un remoto
// inalcanzable, `status` sin `--change` no lanza ninguna operación remota y solo la lista
// cerrada de lecturas, que incluye `ls-tree` para los pendientes.
func TestBranchStatusWithoutChangeLaunchesNoRemoteGitOperation(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	addBranchChanges(t, repo, "y")

	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })

	for _, args := range [][]string{
		{"status", "--json", "--cwd", repo},
		{"status", "--cwd", repo},
	} {
		if code, _, _ := runBranchCLI(t, args...); code != 0 {
			t.Fatalf("status %v terminó con el código %d", args, code)
		}
	}

	sawTree := false
	for _, invocation := range seen {
		if subcommand, remote := branchRemoteOperation(invocation[1:]); remote {
			t.Errorf("status lanzó la operación remota git %s: %v", subcommand, invocation)
		}
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list":
		case "ls-tree":
			sawTree = true
		default:
			t.Errorf("status lanzó %q, fuera de la lista cerrada de lecturas: %v", invocation[1], invocation)
		}
	}
	if !sawTree {
		t.Errorf("la lista de pendientes se lee con `git ls-tree`, y el observador vio %v", seen)
	}
}

// Los rechazos de `status` sin `--change` son los de siempre: fuera de git,
// `not_a_git_repository` con código 1 y la continuación; un fichero del registro corrupto,
// `config_invalid` que lo nombra.
func TestBranchStatusWithoutChangeRefusals(t *testing.T) {
	t.Run("fuera de un repositorio git", func(t *testing.T) {
		outside := branchOutsideGit(t)

		code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", outside)

		if code != 1 {
			t.Errorf("código de salida = %d, se esperaba 1", code)
		}
		result := decodeBranchResult(t, stdout)
		if result.Status != "refused" || result.Code != "not_a_git_repository" || !strings.Contains(result.Continuation, "axiom branch status --cwd") {
			t.Errorf("estado %q, código %q, continuación %q", result.Status, result.Code, result.Continuation)
		}
	})
	t.Run("un registro corrupto", func(t *testing.T) {
		repo := newBranchRepository(t)
		path := filepath.Join(repo, ".git", "axiom", "branch", "v1", "sdd", "roto.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{esto no es json"), 0o644); err != nil {
			t.Fatal(err)
		}

		code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)

		if code != 1 {
			t.Errorf("código de salida = %d, se esperaba 1", code)
		}
		result := decodeBranchResult(t, stdout)
		// Git devuelve la grafía larga de Windows y el temporal lleva el nombre corto 8.3
		// (tasks.md, regla 10): el fichero se compara por identidad, no por texto.
		named, namedErr := os.Stat(result.Details["path"])
		expected, expectedErr := os.Stat(path)
		if result.Code != "config_invalid" || namedErr != nil || expectedErr != nil || !os.SameFile(named, expected) {
			t.Errorf("código %q, details %v; se esperaba config_invalid que nombre %s", result.Code, result.Details, path)
		}
		for _, want := range []string{"borra", pathquote.Quote(result.Details["path"]), "axiom branch status --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
}

// Los errores de uso de `status` siguen terminando con 2 sin `--change`: el carril solo
// admite sdd u odd, y un `--change` explícito pero en blanco no se confunde con la
// ausencia de la bandera (listaría todo sin que el usuario lo pidiera).
func TestBranchStatusWithoutChangeKeepsTheUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"carril desconocido", []string{"status", "--lane", "xyz"}, "--lane 'xyz'"},
		{"--change en blanco", []string{"status", "--json", "--change", " "}, "--change"},
		{"argumento posicional", []string{"status", "sobra"}, "'sobra'"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)
			if code != 2 || stdout != "" {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			if !strings.Contains(stderr, "axiom branch status --help") || !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr %q debe nombrar %q y `axiom branch status --help`", stderr, tc.want)
			}
		})
	}
}

// La ayuda de `status` documenta que `--change` es opcional y qué lista sin él.
func TestBranchStatusHelpDocumentsTheOptionalChangeAndThePendingList(t *testing.T) {
	code, stdout, _ := runBranchCLI(t, "status", "--help")

	if code != 0 {
		t.Fatalf("código %d, se esperaba 0", code)
	}
	for _, want := range []string{"[--change <nombre>]", "(opcional)", "registrados", "pendientes de archivar", "axiom branch ensure --change <c> --archive"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de status debe contener %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "(obligatorio)") {
		t.Errorf("--change ya no es obligatorio:\n%s", stdout)
	}
}
