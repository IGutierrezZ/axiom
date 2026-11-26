package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
)

// Diferido por rol (C-07-f, design D-07, D-08 y D-15): `ensure --role` crea las ramas pendientes de
// ese rol, `status` las describe sin mutar y `--branch` sobre la rama del cambio pendiente la crea.
// Las que instalan el doble de integraciones tocan estado de paquete y no llaman a t.Parallel().

func ensureRole(t *testing.T, master, role string, mutate func(*EnsureRequest)) branch.Result {
	t.Helper()
	request := EnsureRequest{CWD: master, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeWorktree, Role: role}
	if mutate != nil {
		mutate(&request)
	}
	engine := Default()
	engine.prefs = hubWorktree
	return engine.Ensure(context.Background(), request)
}

func statusOf(t *testing.T, master string) branch.Result {
	t.Helper()
	engine := Default()
	engine.prefs = hubWorktree
	return engine.Status(context.Background(), master, "x", branch.LaneSDD)
}

// twoRoleProject monta un multirepo con un repositorio por rol, ambos con `{task_ref}` en la
// plantilla de su rol.
func twoRoleProject(t *testing.T) (master, specs string) {
	t.Helper()
	integrationstest.Install(t, integrationstest.NewFake())
	roles := map[string][]string{"api": {"repo-api"}, "jobs": {"repo-jobs"}}
	master, specs = newMultirepo(t, roles, "repo-api", "repo-jobs")
	writeFile(t, filepath.Join(master, "axiom.yaml"), "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n"+
		"git:\n  integration_branch: main\n  branch_prefix: inc\n"+
		"roles:\n"+
		"  api:\n    name: api\n    branch_template: \"feat/TASK-{task_ref}\"\n    repositories:\n      - path: repo-api\n"+
		"  jobs:\n    name: jobs\n    branch_template: \"feat/TASK-{task_ref}\"\n    repositories:\n      - path: repo-jobs\n"+
		"integrations:\n  fake:\n    project: p\n")
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "api", "jobs"), "api", "jobs")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"),
		"schema: axiom.integrations/v1\nchange: x\nlinks:\n"+
			"  - {plugin: fake, kind: task, id: \"1\", role: api, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n"+
			"  - {plugin: fake, kind: task, id: \"2\", role: jobs, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n")
	return master, specs
}

func localBranches(t *testing.T, repo string) string {
	t.Helper()
	return strings.TrimSpace(runGit(t, repo, "branch", "--list", "feature/*", "feat/*"))
}

func templatedRepo(t *testing.T, result branch.Result, name string) branch.RepositoryResult {
	t.Helper()
	for _, repository := range result.Repositories {
		if repository.Name == name {
			return repository
		}
	}
	t.Fatalf("no hay repositorio %s en %+v", name, result.Repositories)
	return branch.RepositoryResult{}
}

// Matriz de amenazas: sin `--role` o con el de otro rol, la rama pendiente no se crea aunque su
// candidato ya esté traído como `origin/<candidato>`; con su rol, sí, y no toca a los demás.
func TestEnsureRoleCreatesOnlyTheBranchOfTheEnteringRole(t *testing.T) {
	master, _ := twoRoleProject(t)
	api, jobs := filepath.Join(master, "repo-api"), filepath.Join(master, "repo-jobs")
	fakeRemoteRef(t, api, "feat/TASK-1")

	wantReady(t, ensureRole(t, master, "", nil))
	wantReady(t, ensureRole(t, master, "jobs", nil))

	if got := localBranches(t, api); got != "" {
		t.Fatalf("api no entra: no debe crearse ni adoptarse su rama: %q", got)
	}
	if got := localBranches(t, jobs); !strings.Contains(got, "feat/TASK-2") {
		t.Fatalf("jobs entra: su rama debe crearse: %q", got)
	}

	wantReady(t, ensureRole(t, master, "api", nil))

	if got := localBranches(t, api); !strings.Contains(got, "feat/TASK-1") {
		t.Fatalf("api entra: su rama debe crearse: %q", got)
	}
	if got := localBranches(t, jobs); !strings.Contains(got, "feat/TASK-2") || strings.Contains(got, "TASK-1") {
		t.Errorf("la entrada de api no toca a jobs: %q", got)
	}
}

// `ensure --role` es idempotente: repetirla da el mismo resultado byte a byte.
func TestEnsureRoleIsIdempotent(t *testing.T) {
	master, _ := twoRoleProject(t)
	wantReady(t, ensureRole(t, master, "api", nil))
	first := ensureRole(t, master, "api", nil)
	wantReady(t, first)

	second := ensureRole(t, master, "api", nil)

	wantReady(t, second)
	one, _ := json.Marshal(first)
	two, _ := json.Marshal(second)
	if string(one) != string(two) {
		t.Errorf("la repetición difiere:\n%s\n%s", one, two)
	}
}

// Matriz de amenazas: un `{task_ref}` no seguro (`-D`) rechaza la entrada del rol sin crear rama,
// worktree ni registro.
func TestEnsureRoleWithAnUnsafeTaskRefCreatesNothing(t *testing.T) {
	master, specs := twoRoleProject(t)
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"),
		"schema: axiom.integrations/v1\nchange: x\nlinks:\n"+
			"  - {plugin: fake, kind: task, id: \"-D\", role: api, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n")

	result := ensureRole(t, master, "api", nil)

	if result.Code != branch.CodeInvalidName {
		t.Fatalf("código = %q (%s), se esperaba invalid_name", result.Code, result.Message)
	}
	if got := localBranches(t, filepath.Join(master, "repo-api")); got != "" {
		t.Errorf("no debe crearse ninguna rama: %q", got)
	}
	wantNoRegistryFile(t, specs, "x")
}

// S2: segundo clon, sin registro y con la rama ya publicada (`origin/<candidato>`): `ensure --role`
// la crea con seguimiento, sin red.
func TestEnsureRoleAdoptsThePublishedBranchWithoutRegistry(t *testing.T) {
	master, _ := twoRoleProject(t)
	api := filepath.Join(master, "repo-api")
	fakeRemoteRef(t, api, "feat/TASK-1")

	result := ensureRole(t, master, "api", nil)

	wantReady(t, result)
	if got := localBranches(t, api); !strings.Contains(got, "feat/TASK-1") {
		t.Fatalf("la rama publicada debe adoptarse: %q", got)
	}
	created := branchByName(t, templatedRepo(t, result, "repo-api"), "feat/TASK-1")
	if created.CreatedFrom != branch.CreatedFromRemote {
		t.Errorf("createdFrom = %q, se esperaba remote", created.CreatedFrom)
	}
}

// S4: `--branch` sobre la rama del cambio pendiente la crea ya.
func TestEnsureBranchFlagCreatesAPendingChangeBranch(t *testing.T) {
	master, _, backend := sealedTaskProject(t)

	result := ensureRole(t, master, "", func(r *EnsureRequest) { r.Exceptions = branch.Exceptions{Branch: "feature/manual"} })

	wantReady(t, result)
	if got := strings.TrimSpace(runGit(t, backend, "branch", "--list", "feature/*")); !strings.Contains(got, "feature/manual") {
		t.Fatalf("`--branch` debe crear la rama: %q", got)
	}
	if pending := templatedRepo(t, result, "repo-backend").PendingBranches; len(pending) != 0 {
		t.Errorf("no debe quedar pendiente: %+v", pending)
	}
}

// D-15: `status` lista la rama pendiente con su plantilla, su rol y la orden que la crea, sin nombre
// mientras la Task no exista y con él cuando ya existe; no crea nada.
func TestStatusListsPendingBranchesWithoutMutating(t *testing.T) {
	master, specs, backend := sealedTaskProject(t)

	result := statusOf(t, master)

	wantReached(t, result)
	pending := templatedRepo(t, result, "repo-backend").PendingBranches
	if len(pending) != 1 {
		t.Fatalf("pendientes = %+v", pending)
	}
	got := pending[0]
	if got.Role != "backend" || got.Template != "feature/TASK-{task_ref}" || got.Name != "" || got.Reason != branch.PendingReasonAwaitingRole || got.ReasonLabel == "" {
		t.Errorf("pendiente = %+v", got)
	}
	if !strings.Contains(got.Continuation, "branch ensure --change x --role backend") {
		t.Errorf("continuación = %q", got.Continuation)
	}

	bindTask(t, specs, "backend", "1235")
	named := templatedRepo(t, statusOf(t, master), "repo-backend").PendingBranches
	if len(named) != 1 || named[0].Name != "feature/TASK-1235" {
		t.Errorf("con Task el nombre candidato debe constar: %+v", named)
	}
	if listed := runGit(t, backend, "branch", "--list", "feature/*"); listed != "" {
		t.Errorf("status no crea nada: %q", listed)
	}
	wantNoRegistryFile(t, specs, "x")
}

// D-15: con `integrations.yaml` ilegible, `status` no falla: la entrada pendiente informa el código.
func TestStatusWithUnreadableIntegrationsReportsTheCodeOnTheEntry(t *testing.T) {
	master, specs, _ := sealedTaskProject(t)
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"), "esto: [no es válido")

	result := statusOf(t, master)

	wantReached(t, result)
	pending := templatedRepo(t, result, "repo-backend").PendingBranches
	if len(pending) != 1 || pending[0].Code != branch.CodeIntegrationsLinksInvalid || pending[0].Reason != string(branch.CodeIntegrationsLinksInvalid) {
		t.Fatalf("pendiente = %+v", pending)
	}
}

// Sin plantilla, el JSON de `ensure` no lleva `pendingBranches`.
func TestWithoutTemplateThereAreNoPendingBranches(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "pendingBranches") {
		t.Errorf("sin plantilla el JSON no cambia: %s", encoded)
	}
}

// fakeRemoteRef simula una rama ya publicada y traída: configura `origin` y deja
// `refs/remotes/origin/<rama>` en la punta de HEAD, sin red.
func fakeRemoteRef(t *testing.T, repo, name string) {
	t.Helper()
	runGit(t, repo, "config", "remote.origin.url", repo)
	runGit(t, repo, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	runGit(t, repo, "update-ref", "refs/remotes/origin/"+name, "HEAD")
}

// D-07: la pregunta de promoción `ask` no se hace en el `ensure` que deja la rama pendiente, sino en
// el `ensure --role` que la crea; sin respuesta, `decision_required` sin crear nada.
func TestEnsureRoleAsksThePromotionWhenItCreatesTheBranch(t *testing.T) {
	master, _ := twoRoleProject(t)
	config := filepath.Join(master, "axiom.yaml")
	text := readText(t, config)
	writeFile(t, config, strings.Replace(text, "git:\n  integration_branch: main\n", "git:\n  integration_branch: main\n  main_branch: master\n  promotion: ask\n", 1))
	api := filepath.Join(master, "repo-api")

	wantReady(t, ensureRole(t, master, "", nil))

	asked := ensureRole(t, master, "api", nil)
	if asked.Status != branch.StatusDecisionRequired {
		t.Fatalf("status = %q (%s), se esperaba decision_required", asked.Status, asked.Message)
	}
	if got := localBranches(t, api); got != "" {
		t.Fatalf("sin respuesta no se crea nada: %q", got)
	}

	answered := ensureRole(t, master, "api", func(r *EnsureRequest) {
		r.Promotions = []branch.PromotionRequest{{Mode: branch.PromotionRelease}}
	})
	wantReady(t, answered)
	if got := localBranches(t, api); !strings.Contains(got, "feat/TASK-1") {
		t.Errorf("con la respuesta se crea la rama: %q", got)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Pregunta abierta del diseño: `axiom branch scope` con un repositorio solo pendiente es
// `unverifiable` con `branch_missing`, sin código nuevo ni `fetch` que no lo resolvería.
func TestScopeOfAPendingOnlyRepositoryIsBranchMissing(t *testing.T) {
	master, _, _ := sealedTaskProject(t)

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, "repo-backend")
	if entry.Outcome != branch.ScopeOutcomeUnverifiable || entry.Reason != branch.ScopeReasonBranchMissing {
		t.Fatalf("entrada = %+v", entry)
	}
	if entry.Continuation != "" || !strings.Contains(entry.Message, "--role") {
		t.Errorf("no hay fetch que ofrecer y el mensaje nombra --role: %+v", entry)
	}
	if result.Summary == nil || result.Summary.Offer != branch.ScopeOfferUnverifiable {
		t.Errorf("resumen = %+v", result.Summary)
	}
}
