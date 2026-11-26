package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
)

// roleRepository monta un repositorio con axiom.yaml (con el doble configurado), el cambio x con su
// kickoff sellado con los roles dados, todo confirmado en main, y el doble instalado.
func roleRepository(t *testing.T, block string, roles ...string) (string, *integrationstest.Fake) {
	t.Helper()
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, block)
	if len(roles) > 0 {
		sealBranchChange(t, repo, "x", roles...)
	} else if err := os.MkdirAll(filepath.Join(repo, "openspec", "changes", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "openspec", "changes", "x", "proposal.md"), []byte("propuesta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "cambio x")
	fake := integrationstest.NewFake()
	integrationstest.Install(t, fake)
	quiet := writerBranchWarnings
	writerBranchWarnings = discardWriter{}
	t.Cleanup(func() { writerBranchWarnings = quiet })
	return repo, fake
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// gitState resume ramas, worktrees y el registro local del repositorio: lo que un rechazo no puede cambiar.
func gitState(t *testing.T, repo string) string {
	t.Helper()
	registry := "sin registro"
	if _, err := os.Stat(filepath.Join(repo, ".git", "axiom")); err == nil {
		registry = "con registro"
	}
	return branchGitOutput(t, repo, "branch", "--list") + "|" + branchGitOutput(t, repo, "worktree", "list", "--porcelain") + "|" + registry
}

func TestBranchEnsureRoleBlankIsAUsageError(t *testing.T) {
	repo, _ := roleRepository(t, fakeBlock, "backend")
	for _, value := range []string{"", "   "} {
		code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--role="+value, "--cwd", repo)
		if code != 2 || !strings.Contains(stderr, "--role no puede estar vacío") {
			t.Fatalf("--role=%q: código %d, stderr %q", value, code, stderr)
		}
	}
}

func TestBranchEnsureRoleRejectionsAreTypedAndHaveNoEffects(t *testing.T) {
	sealed, _ := roleRepository(t, fakeBlock, "backend", "web")
	unsealed, _ := roleRepository(t, fakeBlock)
	tests := []struct {
		name  string
		repo  string
		args  []string
		code  string
		wants []string
	}{
		{"con --archive", sealed, []string{"--role", "backend", "--archive"}, "plan_conflict", []string{"--archive", "sin --role"}},
		{"con --lane odd", sealed, []string{"--role", "backend", "--lane", "odd"}, "role_not_applicable", []string{"odd"}},
		{"sin kickoff sellado", unsealed, []string{"--role", "backend"}, "kickoff_not_sealed", []string{"axiom sdd kickoff seal"}},
		{"rol fuera del roster", sealed, []string{"--role", "ops"}, "role_not_in_roster", []string{"ops", "backend, web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := gitState(t, tt.repo)
			code, stdout, stderr := runBranchCLI(t, append([]string{"ensure", "--change", "x", "--json", "--cwd", tt.repo}, tt.args...)...)
			if code != 1 || stderr != "" {
				t.Fatalf("código %d, stderr %q\n%s", code, stderr, stdout)
			}
			result := decodeBranchResult(t, stdout)
			if result.Status != "refused" || result.Code != tt.code {
				t.Fatalf("estado %q, código %q; se esperaba refused y %q\n%s", result.Status, result.Code, tt.code, stdout)
			}
			for _, want := range tt.wants {
				if !strings.Contains(result.Message+" "+result.Continuation, want) {
					t.Errorf("falta %q en el mensaje o la continuación:\n%s", want, stdout)
				}
			}
			if after := gitState(t, tt.repo); after != before {
				t.Fatalf("el rechazo no puede crear ramas, worktrees ni registro:\nantes   %s\ndespués %s", before, after)
			}
		})
	}
}

func TestBranchEnsureRoleMatchesRosterCaseInsensitively(t *testing.T) {
	repo, _ := roleRepository(t, fakeBlock, "backend")
	code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--role", "BackEnd", "--cwd", repo)
	if code != 0 {
		t.Fatalf("el rol se compara en la forma canónica en minúsculas: %d %q", code, stderr)
	}
}

func TestBranchEnsureRoleJSONIsIdenticalWithAndWithoutTheFlagAndEmitsRoleStart(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")

	code, withRole, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--role", "backend", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("con rol: %d %q\n%s", code, stderr, withRole)
	}
	if keys := fake.DeliveredKeys(); len(keys) != 1 || keys[0] != "role.start/x/backend" {
		t.Fatalf("ensure --role emite role.start con su clave: %v", keys)
	}
	code, again, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--role", "backend", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("con rol, repetido: %d %q", code, stderr)
	}
	code, without, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("sin rol: %d %q", code, stderr)
	}
	if again != without {
		t.Fatalf("el JSON de ensure es idéntico con y sin la bandera:\n%s\n---\n%s", again, without)
	}
	if strings.Contains(withRole, "role.start") || strings.Contains(withRole, "backend\"") && strings.Contains(withRole, "\"role\"") {
		t.Fatalf("el JSON no lleva nada del rol: %s", withRole)
	}
	if fake.Calls() != 2 {
		t.Fatalf("sin --role no se emite nada (llamadas %d)", fake.Calls())
	}

	// Repetir ensure --role no duplica la operación: el doble, idempotente por clave, no registra otra.
	runBranchCLI(t, "ensure", "--change", "x", "--role", "backend", "--cwd", repo)
	if keys := fake.DeliveredKeys(); len(keys) != 1 {
		t.Fatalf("la clave se repite sin duplicar: %v", keys)
	}
}

func TestBranchEnsureRoleStartFailureKeepsEnsureOutputAndExitCode(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")
	fake.FailAlways(os.ErrDeadlineExceeded)
	warnings := captureIntegrationWarnings(t)

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--json", "--role", "backend", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("ensure sale con su resultado de siempre: %d %q", code, stderr)
	}
	if result := decodeBranchResult(t, stdout); result.Status != "ready" {
		t.Fatalf("estado %q", result.Status)
	}
	if !strings.Contains(warnings.String(), "Aviso:") || !strings.Contains(warnings.String(), "axiom integrations sync") {
		t.Fatalf("el fallo avisa por el canal propio: %q", warnings)
	}
}

func TestBranchEnsureWithoutRoleChangesNothing(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")
	code, _, stderr := runBranchCLI(t, "ensure", "--change", "x", "--cwd", repo)
	if code != 0 || stderr != "" || fake.Calls() != 0 {
		t.Fatalf("sin --role: %d %q llamadas %d", code, stderr, fake.Calls())
	}
}

func TestEmitEnsureRoleStartOnlyForAReadyResult(t *testing.T) {
	repo, fake := roleRepository(t, fakeBlock, "backend")
	for _, status := range []branch.Status{branch.StatusDecisionRequired, branch.StatusRefused, branch.StatusError, branch.StatusPartial} {
		emitEnsureRoleStart(repo, branch.Result{Status: status, PlanningHome: filepath.Join(repo, "openspec")}, "x", "backend")
	}
	if fake.Calls() != 0 {
		t.Fatalf("solo un resultado ready emite (llamadas %d)", fake.Calls())
	}
}

// RF-BRA-33 modificado (C-07-f): el rol sí llega al motor, que lo usa para crear las ramas
// pendientes de ese rol; sin plantilla no tiene ningún efecto en el plan.
func TestEnsureRequestCarriesTheEnteringRole(t *testing.T) {
	if _, has := reflect.TypeOf(engine.EnsureRequest{}).FieldByName("Role"); !has {
		t.Fatal("EnsureRequest debe llevar el rol que entra (Role)")
	}
}
