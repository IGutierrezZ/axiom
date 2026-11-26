package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

// La barrera task_not_bound de `ensure --role` y el registro de started_at (C-07-e, U10): el escenario
// es un repositorio git con el cambio x sellado, el work item vinculado y el doble de Azure DevOps.

// linksProbe es el plugin de Azure DevOps con la entrega observada: guarda los vínculos que ve cada
// evento, para comprobar que role.start ya lleva started_at.
type linksProbe struct {
	*azuredevops.Plugin
	seen map[string][]integrations.Link
}

func (p *linksProbe) Deliver(ctx context.Context, d integrations.Delivery) (integrations.Outcome, error) {
	p.seen[d.Event.IdempotencyKey] = append([]integrations.Link(nil), d.Links...)
	return p.Plugin.Deliver(ctx, d)
}

type roleTrackerEnv struct {
	repo       string
	changeRoot string
	srv        *adotest.Server
	folder     integrations.Folder
	probe      *linksProbe
}

// newRoleTrackerEnv monta el repositorio con el bloque de integraciones dado. workItem vincula el work
// item 100 al cambio; taskRoles vincula una Task por cada rol dado.
func newRoleTrackerEnv(t *testing.T, block string, workItem bool, taskRoles []string, roles ...string) *roleTrackerEnv {
	t.Helper()
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, block)
	sealBranchChange(t, repo, "x", roles...)
	changeRoot := filepath.Join(repo, "openspec", "changes", "x")
	if err := os.WriteFile(filepath.Join(changeRoot, "proposal.md"), []byte("propuesta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	quiet := writerBranchWarnings
	writerBranchWarnings = discardWriter{}
	t.Cleanup(func() { writerBranchWarnings = quiet })
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	t.Setenv(adoPATEnv, adoPAT)

	srv := adotest.NewServer(t, nil)
	adotest.Install(t, srv)
	probe := &linksProbe{Plugin: azuredevops.New(srv.Options()), seen: map[string][]integrations.Link{}}
	integrations.SetDefaultRegistryForTest(t, probe)
	srv.SeedStates("Task", adotest.State{Name: "To Do", Category: "Proposed"}, adotest.State{Name: "Active", Category: "InProgress"})

	e := &roleTrackerEnv{repo: repo, changeRoot: changeRoot, srv: srv, probe: probe, folder: integrations.Folder{WorkspaceRoot: repo, Root: changeRoot}}
	now := time.Now().UTC()
	if workItem {
		link := integrations.Link{Plugin: "azure-devops", Kind: "work_item", ID: "100", URL: srv.URL() + "/contoso/_apis/wit/workItems/100", Source: integrations.WorkItemLinkSource}
		if _, err := integrations.LinkWorkItem(e.folder, "azure-devops", link, now); err != nil {
			t.Fatalf("vincular el work item: %v", err)
		}
	}
	for i, role := range taskRoles {
		id := 200 + i
		link := integrations.Link{Plugin: "azure-devops", Kind: integrations.TaskLinkKind, ID: itoaCLI(id), URL: srv.URL() + "/contoso/_apis/wit/workItems/" + itoaCLI(id), Role: role, Source: integrations.TasksBindSource}
		if _, err := integrations.BindTask(e.folder, "azure-devops", link, now); err != nil {
			t.Fatalf("vincular la Task de %s: %v", role, err)
		}
		srv.SeedWorkItem(id, 1, map[string]any{"System.WorkItemType": "Task", "System.State": "To Do", "System.Title": "x / " + role})
	}
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "cambio x")
	return e
}

// ensure ejecuta `ensure` y, si el resultado trae el planning home (el del worktree cuando lo hay),
// pasa a leer y escribir los vínculos de esa carpeta, que es donde se registra started_at.
func (e *roleTrackerEnv) ensure(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	code, stdout, stderr := runBranchCLI(t, append([]string{"ensure", "--change", "x", "--json", "--cwd", e.repo}, args...)...)
	if stdout != "" {
		if result := decodeBranchResult(t, stdout); result.Status == "ready" && result.PlanningHome != "" {
			e.follow(result.WorkspaceRoot, result.PlanningHome)
		}
	}
	return code, stdout, stderr
}

func (e *roleTrackerEnv) follow(workspaceRoot, planningHome string) {
	e.changeRoot = filepath.Join(planningHome, "changes", "x")
	e.folder = integrations.Folder{WorkspaceRoot: workspaceRoot, Root: e.changeRoot}
}

func (e *roleTrackerEnv) linksFile(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.changeRoot, integrations.LinksFile))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func (e *roleTrackerEnv) taskLink(t *testing.T, role string) integrations.Link {
	t.Helper()
	links, err := integrations.TaskLinks(e.folder)
	if err != nil {
		t.Fatalf("TaskLinks: %v", err)
	}
	for _, l := range links {
		if l.Role == role {
			return l
		}
	}
	t.Fatalf("no hay vínculo de Task del rol %s", role)
	return integrations.Link{}
}

func stubTrackerNow(t *testing.T, at time.Time) {
	t.Helper()
	previous := trackerNow
	trackerNow = func() time.Time { return at }
	t.Cleanup(func() { trackerNow = previous })
}

func TestEnsureRoleBarrierRefusesRoleWithoutTaskAndMutatesNothing(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, nil, "fullstack")
	before, links := gitState(t, e.repo), e.linksFile(t)

	code, stdout, stderr := e.ensure(t, "--role", "fullstack")
	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q\n%s", code, stderr, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "refused" || result.Code != "task_not_bound" {
		t.Fatalf("estado %q, código %q\n%s", result.Status, result.Code, stdout)
	}
	text := result.Message + " " + result.Continuation
	for _, want := range []string{"axiom integrations tasks create --change x", "axiom integrations tasks bind --change x --role fullstack=<id>"} {
		if !strings.Contains(text, want) {
			t.Errorf("falta %q en la resolución: %s", want, text)
		}
	}
	if gitState(t, e.repo) != before || e.linksFile(t) != links {
		t.Fatal("el rechazo no puede crear ramas, worktrees ni registro ni tocar integrations.yaml")
	}
	if n := len(e.srv.Requests()); n != 0 || len(e.probe.seen) != 0 {
		t.Fatalf("la barrera no usa la red ni emite eventos: %d peticiones, %d entregas", n, len(e.probe.seen))
	}
}

func TestEnsureRoleBarrierIsLiftedWhenTheRoleHasItsTask(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	code, stdout, stderr := e.ensure(t, "--role", "fullstack")
	if code != 0 {
		t.Fatalf("con la Task vinculada ensure termina listo: %d %q\n%s", code, stderr, stdout)
	}
	if result := decodeBranchResult(t, stdout); result.Status != "ready" {
		t.Fatalf("estado %q", result.Status)
	}
}

func TestEnsureRoleRosterRefusalPrecedesTheBarrier(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, nil, "backend")
	code, stdout, _ := e.ensure(t, "--role", "frontend")
	if result := decodeBranchResult(t, stdout); code != 1 || result.Code != "role_not_in_roster" {
		t.Fatalf("código %d, %q: el roster va antes que la barrera\n%s", code, result.Code, stdout)
	}
}

func TestEnsureRoleBarrierIgnoresChangesWithoutWorkItemOrWithoutAnActivePlugin(t *testing.T) {
	tests := []struct {
		name     string
		block    string
		workItem bool
	}{
		{"sin vínculo de work item", adoBlock, false},
		{"plugin desactivado", adoBlock + "    enabled: false\n", true},
		{"sin bloque de integraciones", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newRoleTrackerEnv(t, tt.block, tt.workItem, nil, "fullstack")
			captureIntegrationWarnings(t)
			links := e.linksFile(t)
			code, stdout, stderr := e.ensure(t, "--role", "fullstack")
			if code != 0 || stderr != "" {
				t.Fatalf("no hay barrera: %d %q\n%s", code, stderr, stdout)
			}
			if result := decodeBranchResult(t, stdout); result.Status != "ready" {
				t.Fatalf("estado %q\n%s", result.Status, stdout)
			}
			if e.linksFile(t) != links {
				t.Fatal("integrations.yaml no puede cambiar")
			}
		})
	}
}

func TestEnsureRoleBarrierFailsClosedOnUnreadableLinks(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, nil, "fullstack")
	if err := os.WriteFile(filepath.Join(e.changeRoot, integrations.LinksFile), []byte("links: [esto no es\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := gitState(t, e.repo)
	code, stdout, _ := e.ensure(t, "--role", "fullstack")
	result := decodeBranchResult(t, stdout)
	if code != 1 || result.Code != "integrations_links_invalid" {
		t.Fatalf("código %d, %q\n%s", code, result.Code, stdout)
	}
	if !strings.Contains(result.Continuation, "axiom integrations sync --change x") {
		t.Errorf("la resolución nombra la orden: %s", result.Continuation)
	}
	if gitState(t, e.repo) != before {
		t.Fatal("el rechazo no crea nada")
	}
}

func TestEnsureRoleRecordsStartedAtOnReadyAndKeepsItOnReentry(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	first := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	stubTrackerNow(t, first)
	if code, stdout, stderr := e.ensure(t, "--role", "fullstack"); code != 0 {
		t.Fatalf("primera entrada: %d %q\n%s", code, stderr, stdout)
	}
	started := e.taskLink(t, "fullstack").StartedAt
	if started == nil || !started.Equal(first) {
		t.Fatalf("started_at = %v, se esperaba %v", started, first)
	}

	stubTrackerNow(t, first.Add(48*time.Hour))
	if code, stdout, stderr := e.ensure(t, "--role", "fullstack"); code != 0 {
		t.Fatalf("reentrada: %d %q\n%s", code, stderr, stdout)
	}
	if again := e.taskLink(t, "fullstack").StartedAt; again == nil || !again.Equal(first) {
		t.Fatalf("la reentrada no sobrescribe started_at: %v", again)
	}
}

func TestEnsureRoleStartDeliverySeesTheLinkWithStartedAt(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	first := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	stubTrackerNow(t, first)
	if code, stdout, stderr := e.ensure(t, "--role", "fullstack"); code != 0 {
		t.Fatalf("ensure: %d %q\n%s", code, stderr, stdout)
	}
	links, ok := e.probe.seen["role.start/x/fullstack"]
	if !ok {
		t.Fatalf("role.start no se entregó: %v", e.probe.seen)
	}
	var seen *integrations.Link
	for i := range links {
		if links[i].Kind == integrations.TaskLinkKind && links[i].Role == "fullstack" {
			seen = &links[i]
		}
	}
	if seen == nil || seen.StartedAt == nil || !seen.StartedAt.Equal(first) {
		t.Fatalf("role.start debe ver el vínculo con started_at: %+v", seen)
	}
}

func TestEnsureRoleStartedAtWriteFailureWarnsAndKeepsOutputAndExitCode(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	warnings := captureIntegrationWarnings(t)
	original := integrationWriterGuard
	integrationWriterGuard = func(_, _, _ string, _ branch.Context) error { return errors.New("rama equivocada") }
	t.Cleanup(func() { integrationWriterGuard = original })

	code, stdout, stderr := e.ensure(t, "--role", "fullstack")
	if code != 0 || stderr != "" {
		t.Fatalf("ensure sale con su resultado de siempre: %d %q", code, stderr)
	}
	if result := decodeBranchResult(t, stdout); result.Status != "ready" {
		t.Fatalf("estado %q", result.Status)
	}
	if !strings.Contains(warnings.String(), "started_at_not_recorded") {
		t.Fatalf("falta el aviso tipado: %q", warnings)
	}
	if e.taskLink(t, "fullstack").StartedAt != nil {
		t.Fatal("no se registró nada")
	}
}

func TestEnsureRoleDoesNotRecordStartedAtOutsideReady(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	links := e.linksFile(t)
	for _, status := range []branch.Status{branch.StatusDecisionRequired, branch.StatusRefused, branch.StatusError, branch.StatusPartial} {
		recordEnsureRoleStarted(e.repo, branch.Result{Status: status, PlanningHome: filepath.Join(e.repo, "openspec")}, "x", "fullstack")
	}
	if e.linksFile(t) != links {
		t.Fatal("solo un resultado ready registra started_at")
	}
}

func TestEnsureWithoutRoleLeavesIntegrationsFileUntouched(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	links := e.linksFile(t)
	if code, stdout, stderr := e.ensure(t); code != 0 {
		t.Fatalf("ensure sin --role: %d %q\n%s", code, stderr, stdout)
	}
	if e.linksFile(t) != links {
		t.Fatal("sin --role integrations.yaml queda byte a byte igual")
	}
	if len(e.probe.seen) != 0 {
		t.Fatalf("sin --role no se emite nada: %v", e.probe.seen)
	}
}

func TestDashboardEnsurePathLeavesIntegrationsFileUntouched(t *testing.T) {
	e := newRoleTrackerEnv(t, adoBlock, true, []string{"fullstack"}, "fullstack")
	links := e.linksFile(t)
	result := engine.Default().Ensure(context.Background(), engine.EnsureRequest{CWD: e.repo, Change: "x", Lane: branch.LaneSDD})
	if result.Status != branch.StatusReady {
		t.Fatalf("el motor termina listo: %s (%s)", result.Status, result.Message)
	}
	e.follow(result.WorkspaceRoot, result.PlanningHome)
	if e.linksFile(t) != links {
		t.Fatal("el alta desde el dashboard llama al motor y no registra started_at")
	}
}

func TestEnsureRoleBarrierSkipsAnUnlocatableWorkspace(t *testing.T) {
	outside := branchOutsideGit(t)
	if refusal := ensureRoleTaskBarrier(outside, "x", "fullstack", outside); refusal != nil {
		t.Fatalf("sin workspace no hay barrera: %+v", refusal)
	}
}
