package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `axiom branch status` sobre el registro local de solo lectura, los
// multirepos y la colocación de los worktrees (REQ-25.5, REQ-25.11, REQ-25.14,
// REQ-25.15): el contrato visible de la CLI, con git real y sin red.

// writeBranchRegistry escribe a mano el registro de un cambio en el git common dir de
// un repositorio normal (`.git`), como lo haría `ensure`, y devuelve su ruta.
func writeBranchRegistry(t *testing.T, repo string, record branch.Record) string {
	t.Helper()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, ".git", "axiom", "branch", "v1", string(record.Lane), record.Change+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func branchRecord(repo string) branch.Record {
	return branch.Record{
		Schema:       branch.RegistrySchema,
		Lane:         branch.LaneSDD,
		Change:       "x",
		Mode:         branch.ModeWorktree,
		ModeSource:   branch.ModeSourceAnswer,
		PlanningRepo: repo,
		Repositories: []branch.RecordRepo{{
			Name:      filepath.Base(repo),
			GitRoot:   repo,
			CommonDir: filepath.Join(repo, ".git"),
			Role:      branch.RepoChangeBranch,
			Branches:  []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange, Worktree: filepath.Join(filepath.Dir(repo), "x", filepath.Base(repo))}},
		}},
	}
}

// REQ-25.14, escenario «Registro ausente informado por `status`»: el JSON indica que el
// cambio no tiene registro y el comando no falla.
func TestBranchStatusReportsAnAbsentRegistry(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Registry == nil || result.Registry.Present || result.Mode != "" || result.ModeSource != "" {
		t.Errorf("registry %+v, modo %q, origen %q; se esperaba present=false y modo desconocido", result.Registry, result.Mode, result.ModeSource)
	}
	if !strings.Contains(stdout, `"present": false`) {
		t.Errorf("`registry.present: false` debe imprimirse explícitamente:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "axiom")); !os.IsNotExist(err) {
		t.Errorf("`status` es de solo lectura y no debe crear el registro (%v)", err)
	}
}

// REQ-25.14 y REQ-25.15: con un registro escrito a mano, `status` informa del modo
// materializado y su origen (token y etiqueta) y del worktree registrado, tanto en JSON
// como en el texto en castellano.
func TestBranchStatusReadsTheRegistry(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "branch", "feat/x")
	record := branchRecord(repo)
	writeBranchRegistry(t, repo, record)

	t.Run("JSON", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)

		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
		}
		result := decodeBranchResult(t, stdout)
		if result.Registry == nil || !result.Registry.Present {
			t.Fatalf("registry = %+v, se esperaba present=true", result.Registry)
		}
		if result.Mode != "worktree" || result.ModeLabel != "un worktree por rama" || result.ModeSource != "answer" || result.ModeSourceLabel != "respuesta registrada" {
			t.Errorf("modo %q / %q, origen %q / %q", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
		}
		feature := onlyBranchRepository(t, result).Branches[0]
		if feature.RegisteredWorktree != record.Repositories[0].Branches[0].Worktree {
			t.Errorf("registeredWorktree = %q, se esperaba el del registro %q", feature.RegisteredWorktree, record.Repositories[0].Branches[0].Worktree)
		}
	})
	t.Run("texto", func(t *testing.T) {
		code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--cwd", repo)

		if code != 0 {
			t.Fatalf("código %d; se esperaba 0\n%s", code, stdout)
		}
		for _, want := range []string{"Modo de trabajo: un worktree por rama (respuesta registrada)", "Registro local: presente", "worktree registrado " + record.Repositories[0].Branches[0].Worktree} {
			if !strings.Contains(stdout, want) {
				t.Errorf("el texto debe contener %q:\n%s", want, stdout)
			}
		}
	})
}

// REQ-25.14: un registro corrupto es `config_invalid` con código de salida 1, el
// fichero nombrado y la continuación de borrarlo y reconstruirlo con `ensure`.
func TestBranchStatusRefusesACorruptRegistry(t *testing.T) {
	repo := newBranchRepository(t)
	path := writeBranchRegistry(t, repo, branchRecord(repo))
	if err := os.WriteFile(path, []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)

	if code != 1 {
		t.Fatalf("código de salida = %d, se esperaba 1\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "refused" || result.StatusLabel != "rechazado" || result.Code != "config_invalid" {
		t.Errorf("status %q / %q, código %q; se esperaba refused / rechazado / config_invalid", result.Status, result.StatusLabel, result.Code)
	}
	if got := result.Details["path"]; got == "" || !strings.HasSuffix(filepath.ToSlash(got), "axiom/branch/v1/sdd/x.json") {
		t.Errorf("details.path = %q, se esperaba el fichero del registro", got)
	}
	for _, want := range []string{"borra", pathquote.Quote(result.Details["path"]), "axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	if result.Registry != nil || len(result.Repositories) != 0 {
		t.Errorf("un rechazo no lleva datos del cambio: %+v", result)
	}

	t.Run("texto", func(t *testing.T) {
		code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--cwd", repo)
		if code != 1 || stdout != "" || !strings.Contains(stderr, "Error [config_invalid]") || !strings.Contains(stderr, "Continuación:") {
			t.Errorf("código %d, stdout %q, stderr %q; se esperaba el rechazo en stderr", code, stdout, stderr)
		}
	})
}

// REQ-25.5 y REQ-25.15 (multirepo): el repositorio de specs se informa in situ y sin
// ramas, y cada repositorio de código con su rama del cambio; el JSON lleva el papel con
// su etiqueta.
func TestBranchStatusMultirepoReportsTheSpecsRepositoryInPlace(t *testing.T) {
	isolateBranchGit(t)
	master := t.TempDir()
	for _, dir := range []string{"repo-specs", "repo-backend", "repo-frontend"} {
		if err := os.MkdirAll(filepath.Join(master, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		initBranchRepository(t, filepath.Join(master, dir))
	}
	config := `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
  frontend:
    name: Frontend
    repositories:
      - path: repo-frontend
`
	if err := os.WriteFile(filepath.Join(master, "axiom.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	sealBranchChange(t, filepath.Join(master, "repo-specs"), "x", "backend", "frontend")

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", master)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 (cambio no iniciado)\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "not_started" || result.Topology != "multirepo" || len(result.Repositories) != 3 {
		t.Fatalf("status %q, topología %q, repositorios %d; se esperaba not_started en multirepo con tres repositorios", result.Status, result.Topology, len(result.Repositories))
	}
	specs := result.Repositories[0]
	if specs.Name != "repo-specs" || specs.Role != "specs_in_place" || specs.RoleLabel != "repositorio de specs in situ" || len(specs.Branches) != 0 || specs.CurrentBranch != "main" {
		t.Errorf("repositorio de specs = %+v, se esperaba in situ, sin ramas y en main", specs)
	}
	for _, codeRepository := range result.Repositories[1:] {
		if codeRepository.Role != "code" || codeRepository.RoleLabel != "repositorio de código" || len(codeRepository.Branches) != 1 || codeRepository.Branches[0].Name != "feat/x" {
			t.Errorf("repositorio de código = %+v, se esperaba code con feat/x", codeRepository)
		}
	}
	wantBranchDirectory(t, "workspaceRoot", result.WorkspaceRoot, filepath.Join(master, "repo-specs"))
}

// REQ-25.11 y design D-04: sin proyecto, la carpeta maestra es la subcarpeta de partida
// y su raíz de worktrees (`<carpeta>-wt`) quedaría dentro del repositorio: `plan_conflict`
// con código de salida 1 y una continuación ejecutable con la carpeta citada.
func TestBranchStatusRefusesAWorktreeRootInsideTheRepository(t *testing.T) {
	repo := newBranchRepository(t)
	sub := filepath.Join(repo, "app", "web")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", sub)

	if code != 1 {
		t.Fatalf("código de salida = %d, se esperaba 1\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Code != "plan_conflict" || result.StatusLabel != "rechazado" || !strings.HasSuffix(result.Details["worktreeRoot"], "web-wt") {
		t.Errorf("código %q, etiqueta %q, details %v; se esperaba plan_conflict con la raíz web-wt", result.Code, result.StatusLabel, result.Details)
	}
	for _, want := range []string{"axiom branch status --change x --cwd", pathquote.Quote(sub)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// REQ-25.11 (límite real de 220 caracteres): solo tiene sentido en Windows; el resto de
// sistemas no tiene límite propio y la regla se prueba en el planificador y en el motor
// con las reglas de Windows inyectadas. El carril ODD no lee el kickoff, de modo que el
// nombre largo no obliga a la prueba a crear ficheros de más de 260 caracteres.
func TestBranchStatusRefusesAWorktreePathOverTheWindowsLimit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("el límite real de 220 caracteres solo rige en Windows")
	}
	repo := newBranchRepository(t)
	change := strings.Repeat("c", 200)

	code, stdout, _ := runBranchCLI(t, "status", "--change", change, "--lane", "odd", "--json", "--cwd", repo)

	if code != 1 {
		t.Fatalf("código de salida = %d, se esperaba 1\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Code != "path_too_long" || result.Details["max"] != "220" || len(result.Details["path"]) <= 220 {
		t.Errorf("código %q, details %v; se esperaba path_too_long con una ruta de más de 220 caracteres y máximo 220", result.Code, result.Details)
	}
}
