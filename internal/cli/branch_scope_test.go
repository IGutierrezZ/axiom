package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de `axiom branch scope` (C-02-b, REQ-25.1, REQ-25.2, REQ-25.51): banderas, códigos de
// salida y salida en JSON y en texto. El cálculo del alcance se prueba en internal/branch/engine;
// aquí solo el contrato de la línea de órdenes, en proceso con RunBranch.

// sealBranchChangeWithSnapshot sella el kickoff de un cambio con la instantánea de política git
// que escribe `axiom sdd kickoff seal` desde una rama del cambio.
func sealBranchChangeWithSnapshot(t *testing.T, repo, change string, roles ...string) {
	t.Helper()
	location, err := workspace.Locate(repo)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := engine.Default().Snapshot(context.Background(), location, change, roles)
	if err != nil || snapshot == nil {
		t.Fatalf("instantánea de %s: %v (%v)", change, snapshot, err)
	}
	changeRoot := filepath.Join(repo, "openspec", "changes", change)
	var sealed []kickoff.KickoffRole
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"})
	}
	if err := os.MkdirAll(changeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeRoot, "proposal.md"), []byte("# propuesta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema:    kickoff.KickoffSchemaV1,
		Change:    change,
		SealedBy:  "prueba",
		Config:    kickoff.FlowConfig{FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit", HandoffPolicy: kickoff.HandoffNone, Roles: sealed},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("sellar %s con instantánea: %v (publicado: %v)", change, err, ok)
	}
}

// REQ-25.51: `--fetch`, una bandera desconocida, `--change` ausente o en blanco y un nombre
// inválido son errores de uso (2) antes de leer ningún repositorio.
func TestBranchScopeRejectsMisuseWithCode2BeforeReadingAnything(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-existe")
	cases := []struct {
		name string
		args []string
	}{
		{"--fetch", []string{"scope", "--change", "x", "--fetch"}},
		{"bandera desconocida", []string{"scope", "--change", "x", "--lane", "sdd"}},
		{"sin --change", []string{"scope", "--cwd", missing}},
		{"--change en blanco", []string{"scope", "--change", "  ", "--cwd", missing}},
		{"nombre inválido", []string{"scope", "--change", "../x", "--cwd", missing}},
		{"argumento suelto", []string{"scope", "--change", "x", "sobra"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)
			if code != 2 || stdout != "" || !strings.Contains(stderr, "axiom branch scope --help") {
				t.Errorf("código %d, stdout %q, stderr %q; se esperaba 2, stdout vacío y la ayuda citada", code, stdout, stderr)
			}
		})
	}
}

// La ayuda de `scope` documenta sus tres banderas y no ofrece `--fetch` ni `--lane`.
func TestBranchScopeHelpDocumentsOnlyItsFlags(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "scope", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"--change <nombre>", "--cwd <ruta>", "--json", "axiom.branch-scope/v1", "solo con lecturas", "no ejecuta fetch"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de scope debe contener %q:\n%s", want, stdout)
		}
	}
	for _, absent := range []string{"  --fetch", "  --lane"} {
		if strings.Contains(stdout, absent) {
			t.Errorf("la ayuda de scope no puede ofrecer %q:\n%s", absent, stdout)
		}
	}
}

// REQ-25.51: sin kickoff ni documento ODD el cambio no está sellado; sale con 1.
func TestBranchScopeRefusesAChangeWithoutKickoff(t *testing.T) {
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "")

	code, stdout, stderr := runBranchCLI(t, "scope", "--change", "x", "--cwd", repo)

	if code != 1 || stdout != "" || !strings.Contains(stderr, "kickoff_not_sealed") {
		t.Errorf("código %d, stdout %q, stderr %q; se esperaba 1 con kickoff_not_sealed en stderr", code, stdout, stderr)
	}
}

// REQ-25.51: un cambio con documento ODD no es aplicable; sale con 1.
func TestBranchScopeIsNotApplicableToAnOddChange(t *testing.T) {
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "")
	if err := os.MkdirAll(filepath.Join(repo, "odd", "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "odd", "tasks", "x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runBranchCLI(t, "scope", "--change", "x", "--cwd", repo, "--json")

	var result branch.ScopeResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, stdout)
	}
	if code != 1 || result.Schema != branch.ScopeSchema || result.Code != branch.CodeScopeNotApplicable || result.Status != branch.StatusRefused {
		t.Errorf("código %d, schema %q, código %q, estado %q; se esperaba 1, axiom.branch-scope/v1 y scope_not_applicable", code, result.Schema, result.Code, result.Status)
	}
}

// REQ-25.2 y REQ-25.51: con kickoff e instantánea, `--json` imprime `axiom.branch-scope/v1` y
// sale con 0 aunque el repositorio sea `unverifiable` (la rama del cambio no existe); la `--cwd`
// relativa se resuelve a rutas absolutas y `pending` es una lista.
func TestBranchScopeJSONAndRelativeCwd(t *testing.T) {
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "git:\n  integration_branch: main\n  branch_prefix: feat\n")
	sealBranchChangeWithSnapshot(t, repo, "x", "core")
	t.Chdir(repo)

	code, stdout, stderr := runBranchCLI(t, "scope", "--change", "x", "--cwd", ".", "--json")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío\n%s", code, stderr, stdout)
	}
	var result branch.ScopeResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, stdout)
	}
	if result.Schema != "axiom.branch-scope/v1" || result.Operation != "scope" || result.Status != branch.StatusReady || result.Change != "x" {
		t.Errorf("schema %q, operation %q, estado %q, cambio %q", result.Schema, result.Operation, result.Status, result.Change)
	}
	if len(result.Repositories) != 1 || result.Repositories[0].Outcome != branch.ScopeOutcomeUnverifiable || !filepath.IsAbs(result.Repositories[0].GitRoot) {
		t.Fatalf("se esperaba un repositorio unverifiable con ruta absoluta: %+v", result.Repositories)
	}
	if result.Summary == nil || result.Summary.Offer != branch.ScopeOfferUnverifiable || !strings.Contains(stdout, `"pending": [`) {
		t.Errorf("resumen %+v; se esperaba la oferta unverifiable y pending como lista", result.Summary)
	}

	code, stdout, stderr = runBranchCLI(t, "scope", "--change", "x", "--cwd", ".")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "Cambio: x") {
		t.Errorf("texto: código %d, stderr %q, stdout %q; se esperaba 0 y el resumen en castellano", code, stderr, stdout)
	}
}
