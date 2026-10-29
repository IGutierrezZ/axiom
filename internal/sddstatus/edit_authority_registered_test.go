package sddstatus

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// S9-c3 (REQ-25.43, REQ-24.9, design D-14): las raíces que `axiom branch ensure` registra entran en
// `allowedEditRoots`, las tareas se resuelven contra el worktree que corresponde y una ruta del
// checkout principal de un repositorio registrado se reubica en su worktree. Los registros de estas
// pruebas se escriben con registry.Store y los repositorios se montan solo con ficheros.

// registeredRepo es la entrada del registro de un repositorio con las ramas dadas.
func registeredRepo(t *testing.T, gitRoot string, role branch.RepoRole, branches ...branch.RecordBranch) branch.RecordRepo {
	t.Helper()
	common, err := gitcommon.CommonDirFromFiles(gitRoot)
	if err != nil {
		t.Fatalf("common dir de %s: %v", gitRoot, err)
	}
	return branch.RecordRepo{Name: filepath.Base(gitRoot), GitRoot: gitRoot, CommonDir: common, Role: role, Branches: branches}
}

// saveRegistry escribe el registro local del cambio x en el git common dir del planning home.
func saveRegistry(t *testing.T, planningRoot string, mode branch.Mode, repos ...branch.RecordRepo) {
	t.Helper()
	common, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		t.Fatalf("common dir de %s: %v", planningRoot, err)
	}
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: branchChange,
		Mode: mode, ModeSource: branch.ModeSourceDefault, PlanningRepo: planningRoot, Repositories: repos,
	}
	if err := registry.New().Save(common, record); err != nil {
		t.Fatalf("guardar el registro: %v", err)
	}
}

// linkedWorktreeDir crea la carpeta de un worktree enlazado, fuera de cualquier otro repositorio y
// solo con ficheros: basta el `.git` para que la raíz git del detector sea el propio worktree.
func linkedWorktreeDir(t *testing.T) string {
	t.Helper()
	dir := realPath(t, t.TempDir())
	write(t, filepath.Join(dir, ".git"), "gitdir: "+filepath.Join(dir, ".gitdir")+"\n")
	return dir
}

func changeBranch(name, worktree string) branch.RecordBranch {
	return branch.RecordBranch{Name: name, Kind: branch.KindChange, Worktree: worktree}
}

func allowedRoots(status Status) []string { return status.ActionContext.AllowedEditRoots }

func hasRoot(roots []string, want string) bool {
	for _, root := range roots {
		if root == want {
			return true
		}
	}
	return false
}

// D-14: con `rebase` identidad y una única fuente `tasks.md`, los dos detectores reproducen el
// comportamiento de antes del incremento, y la identidad y nil son lo mismo.
func TestEditAuthorityDetectorsKeepTodayWithIdentityRebase(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	serviceA := filepath.Join(workspace, "service-a")
	serviceB := filepath.Join(workspace, "service-b")
	for _, repo := range []string{planning, serviceA, serviceB} {
		write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	}
	tasks := strings.Join([]string{
		"- [ ] 1.1 Update `../service-a/internal/api/handler.go`",
		"- [ ] 2.1 Update `../service-b/internal/worker/consume.go`",
		"",
	}, "\n")
	wantA, wantB := realPath(t, serviceA), realPath(t, serviceB)

	for name, rebase := range map[string]func(string) string{"identidad": identityRebase, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			all := detectUnauthorizedEditRoots(tasks, planning, []string{planning}, rebase)
			if !reflect.DeepEqual(all, []string{wantA, wantB}) {
				t.Errorf("detectUnauthorizedEditRoots() = %v, se esperaba [%s %s]", all, wantA, wantB)
			}
			blocking, future := detectUnauthorizedEditRootsForCurrentWorkUnit(tasks, planning, []string{planning}, rebase)
			if !reflect.DeepEqual(blocking, []string{wantA}) || !reflect.DeepEqual(future, []string{wantB}) {
				t.Errorf("ForCurrentWorkUnit() = (%v, %v), se esperaba ([%s], [%s])", blocking, future, wantA, wantB)
			}
		})
	}
}

// D-14: la reubicación se aplica a la ruta ya resuelta y antes de reducirla a su raíz git, en los dos
// detectores; lo que no reubica sigue exigiendo autorización.
func TestEditAuthorityDetectorsApplyTheRebaseToEveryResolvedPath(t *testing.T) {
	workspace := realPath(t, t.TempDir())
	planning := filepath.Join(workspace, "planning")
	main := filepath.Join(workspace, "service-a")
	other := filepath.Join(workspace, "service-b")
	worktree := linkedWorktreeDir(t)
	for _, repo := range []string{planning, main, other} {
		write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	}
	tasks := "- [ ] 1.1 Update `../service-a/internal/api/handler.go` and `../service-b/x.go`\n"
	rebase := func(path string) string {
		if suffix, ok := strings.CutPrefix(path, main); ok {
			return worktree + suffix
		}
		return path
	}

	want := []string{realPath(t, other)}
	if got := detectUnauthorizedEditRoots(tasks, planning, []string{planning, worktree}, rebase); !reflect.DeepEqual(got, want) {
		t.Errorf("detectUnauthorizedEditRoots() = %v, se esperaba %v", got, want)
	}
	if blocking, _ := detectUnauthorizedEditRootsForCurrentWorkUnit(tasks, planning, []string{planning, worktree}, rebase); !reflect.DeepEqual(blocking, want) {
		t.Errorf("ForCurrentWorkUnit() blocking = %v, se esperaba %v", blocking, want)
	}
}

// REQ-25.43, escenario «Las tareas de un cambio en worktree no exigen autorización»: un monorepo con
// el cambio en un worktree registrado. Sin registro, un worktree fuera del workspace exige concesión;
// con el registro entra en `allowedEditRoots` y deja de bloquear.
func TestEditAuthorityTasksOfAChangeInARegisteredWorktreeNeedNoGrant(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "feat/x")
	worktree := linkedWorktreeDir(t)
	seedReadyChange(t, ws.Master, branchChange, strings.Join([]string{
		"- [ ] 1.1 Editar `internal/api/handler.go`",
		"- [ ] 1.2 Editar `" + filepath.Join(worktree, "internal", "api", "router.go") + "`",
		"",
	}, "\n"))

	before := resolveFrom(t, ws.Master, branchChange)
	if !containsReason(before.BlockedReasons, "blocked(edit_authority_missing)") || !containsReason(before.BlockedReasons, worktree) {
		t.Fatalf("sin registro el worktree exige concesión; motivos: %v", before.BlockedReasons)
	}
	if want := []string{ws.Master}; !reflect.DeepEqual(allowedRoots(before), want) {
		t.Fatalf("sin registro allowedEditRoots = %v, se esperaba %v", allowedRoots(before), want)
	}

	saveRegistry(t, ws.Master, branch.ModeWorktree, registeredRepo(t, ws.Master, branch.RepoChangeBranch, changeBranch("feat/x", worktree)))
	after := resolveFrom(t, ws.Master, branchChange)

	if containsReason(after.BlockedReasons, "blocked(edit_authority_missing)") || after.ApplyState != ApplyReady {
		t.Errorf("con el worktree registrado applyState %q y motivos %v, se esperaba ready sin edit_authority_missing", after.ApplyState, after.BlockedReasons)
	}
	if want := []string{ws.Master, worktree}; !reflect.DeepEqual(allowedRoots(after), want) {
		t.Errorf("allowedEditRoots = %v, se esperaba %v", allowedRoots(after), want)
	}
	if after.Consent != nil {
		t.Errorf("sin raíces sin autorizar no hay sobre de consentimiento: %+v", after.Consent)
	}
}

// REQ-25.43, escenario «La tarea de un rol se resuelve contra el worktree del rol»: `tasks.core.md`
// se resuelve contra el worktree de la rama de rol; un rol sin worktree propio y `tasks.md` contra el
// del cambio. La raíz del worktree del rol figura en `allowedEditRoots`.
func TestEditAuthorityRoleTasksResolveAgainstTheRoleWorktree(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "feat/x")
	changeRoot := seedMultiRoleChange(t, ws.Master, branchChange, kickoff.ExecutionContinuous, map[string]string{
		"core": "- [ ] 1.1 Editar `internal/core/a.go`\n",
		"qa":   "- [ ] 1.1 Editar `internal/qa/b_test.go`\n",
	}, "core", "qa")
	worktree, coreWorktree := linkedWorktreeDir(t), linkedWorktreeDir(t)
	saveRegistry(t, ws.Master, branch.ModeWorktree, registeredRepo(t, ws.Master, branch.RepoChangeBranch,
		changeBranch("feat/x", worktree),
		branch.RecordBranch{Name: "feat/x-core", Kind: branch.KindRole, RoleID: "core", Worktree: coreWorktree},
	))

	bctx := resolveBranchContext(branchQuery{
		workspaceRoot: ws.Master, changeRoot: changeRoot, change: branchChange,
		plans: []planText{{name: "tasks.md"}, {name: "tasks.core.md"}, {name: "tasks.qa.md"}},
	})
	wantBases := map[string]string{"tasks.md": worktree, "tasks.core.md": coreWorktree, "tasks.qa.md": worktree}
	if len(bctx.plans) != len(wantBases) {
		t.Fatalf("plans = %+v, se esperaban %d fuentes", bctx.plans, len(wantBases))
	}
	for _, plan := range bctx.plans {
		if plan.base != wantBases[plan.name] {
			t.Errorf("la base de %s es %q, se esperaba %q", plan.name, plan.base, wantBases[plan.name])
		}
	}

	status := resolveFrom(t, ws.Master, branchChange)
	if containsReason(status.BlockedReasons, "blocked(edit_authority_missing)") {
		t.Errorf("las tareas de los roles caen en sus worktrees registrados: %v", status.BlockedReasons)
	}
	for _, want := range []string{worktree, coreWorktree} {
		if !hasRoot(allowedRoots(status), want) {
			t.Errorf("allowedEditRoots = %v debe incluir %s", allowedRoots(status), want)
		}
	}
}

// Sin worktree del cambio (`in_place`) la base de las fuentes sigue siendo el workspace.
func TestEditAuthorityPlanBasesFallBackToTheWorkspace(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "feat/x")
	changeRoot := seedReadyChange(t, ws.Master, branchChange, branchPlan)
	saveRegistry(t, ws.Master, branch.ModeInPlace, registeredRepo(t, ws.Master, branch.RepoChangeBranch,
		branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, InPlace: true}))

	bctx := resolveBranchContext(branchQuery{
		workspaceRoot: ws.Master, changeRoot: changeRoot, change: branchChange, plans: []planText{{name: "tasks.md"}},
	})

	if len(bctx.plans) != 1 || bctx.plans[0].base != "" {
		t.Errorf("plans = %+v, se esperaba una fuente sin base propia (el workspace)", bctx.plans)
	}
}

// REQ-25.43 y REQ-24.9, escenarios «Un repositorio de código registrado por `ensure` entra en las
// raíces», «Una ruta del checkout principal de un repositorio registrado se reescribe» y «Un
// repositorio no registrado exige concesión».
func TestEditAuthorityRegisteredCodeRepositoryRebasesAndUnregisteredStillNeedsAGrant(t *testing.T) {
	ws := workspacetest.Canonical(t)
	for _, repo := range []string{ws.Specs, ws.Backend, ws.Frontend} {
		switchTo(t, repo, "main")
	}
	plan := strings.Join([]string{
		"- [ ] 1.1 Editar `../repo-backend/src/a.go`",
		"- [ ] 1.2 Editar `../repo-frontend/ui/b.go`",
		"",
	}, "\n")
	changeRoot := seedReadyChange(t, ws.Specs, branchChange, plan)
	backendWorktree := linkedWorktreeDir(t)

	before := resolveFrom(t, ws.Specs, branchChange)
	if !containsReason(before.BlockedReasons, ws.Backend) || !containsReason(before.BlockedReasons, ws.Frontend) {
		t.Fatalf("sin registro ambos repositorios exigen concesión; motivos: %v", before.BlockedReasons)
	}

	saveRegistry(t, ws.Specs, branch.ModeWorktree,
		registeredRepo(t, ws.Specs, branch.RepoSpecsInPlace),
		registeredRepo(t, ws.Backend, branch.RepoCode, changeBranch("feat/x", backendWorktree)),
	)
	after := resolveFrom(t, ws.Specs, branchChange)

	if !containsReason(after.BlockedReasons, "blocked(edit_authority_missing)") || !containsReason(after.BlockedReasons, ws.Frontend) {
		t.Errorf("el repositorio no registrado sigue exigiendo concesión: %v", after.BlockedReasons)
	}
	if containsReason(after.BlockedReasons, ws.Backend) {
		t.Errorf("el repositorio registrado no debe bloquear: %v", after.BlockedReasons)
	}
	if !hasRoot(allowedRoots(after), backendWorktree) || hasRoot(allowedRoots(after), ws.Backend) || hasRoot(allowedRoots(after), ws.Frontend) {
		t.Errorf("allowedEditRoots = %v, se esperaba el worktree de repo-backend y ni su checkout principal ni repo-frontend", allowedRoots(after))
	}

	bctx := resolveBranchContext(branchQuery{workspaceRoot: ws.Specs, changeRoot: changeRoot, change: branchChange, plans: []planText{{name: "tasks.md", text: plan}}})
	for from, want := range map[string]string{
		filepath.Join(ws.Backend, "src", "a.go"):         filepath.Join(backendWorktree, "src", "a.go"),
		filepath.Join(ws.Backend, "nuevo", "deep", "x"):  filepath.Join(backendWorktree, "nuevo", "deep", "x"),
		filepath.Join(ws.Frontend, "ui", "b.go"):         filepath.Join(ws.Frontend, "ui", "b.go"),
		filepath.Join(backendWorktree, "src", "a.go"):    filepath.Join(backendWorktree, "src", "a.go"),
		filepath.Join(ws.Specs, "openspec", "design.md"): filepath.Join(ws.Specs, "openspec", "design.md"),
	} {
		if got := bctx.rebase(from); got != want {
			t.Errorf("rebase(%s) = %s, se esperaba %s", from, got, want)
		}
	}
}

// REQ-25.43, escenario «Un repositorio de código registrado por `ensure` entra en las raíces»: entra
// aunque ninguna tarea lo nombre.
func TestEditAuthorityRegisteredCodeRepositoryEntersTheRootsWithoutTasksNamingIt(t *testing.T) {
	ws := workspacetest.Canonical(t)
	switchTo(t, ws.Specs, "main")
	switchTo(t, ws.Backend, "main")
	seedReadyChange(t, ws.Specs, branchChange, branchPlan)
	backendWorktree := linkedWorktreeDir(t)
	saveRegistry(t, ws.Specs, branch.ModeWorktree,
		registeredRepo(t, ws.Specs, branch.RepoSpecsInPlace),
		registeredRepo(t, ws.Backend, branch.RepoCode, changeBranch("feat/x", backendWorktree)),
	)

	status := resolveFrom(t, ws.Specs, branchChange)

	if want := []string{ws.Specs, backendWorktree}; !reflect.DeepEqual(allowedRoots(status), want) {
		t.Errorf("allowedEditRoots = %v, se esperaba %v", allowedRoots(status), want)
	}
}

// D-14: las raíces registradas incluyen el worktree de archivado y, en `in_place`, la raíz del propio
// repositorio de código; un worktree que ya no existe se omite.
func TestEditAuthorityRegisteredRootsCoverArchiveInPlaceAndSkipMissingWorktrees(t *testing.T) {
	ws := workspacetest.Canonical(t)
	switchTo(t, ws.Specs, "main")
	switchTo(t, ws.Backend, "main")
	seedReadyChange(t, ws.Specs, branchChange, "- [ ] 1.1 Editar `../repo-backend/src/a.go`\n")
	archiveWorktree := linkedWorktreeDir(t)
	missing := filepath.Join(realPath(t, t.TempDir()), "ya-no-existe")

	saveRegistry(t, ws.Specs, branch.ModeInPlace,
		registeredRepo(t, ws.Specs, branch.RepoSpecsInPlace),
		registeredRepo(t, ws.Backend, branch.RepoCode,
			branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, InPlace: true},
			branch.RecordBranch{Name: "feat/x-archive", Kind: branch.KindArchive, Worktree: archiveWorktree},
			branch.RecordBranch{Name: "feat/x-core", Kind: branch.KindRole, RoleID: "core", Worktree: missing},
		),
	)

	status := resolveFrom(t, ws.Specs, branchChange)

	if want := []string{ws.Specs, ws.Backend, archiveWorktree}; !reflect.DeepEqual(allowedRoots(status), want) {
		t.Errorf("allowedEditRoots = %v, se esperaba %v", allowedRoots(status), want)
	}
	if containsReason(status.BlockedReasons, "blocked(edit_authority_missing)") {
		t.Errorf("en in_place el repositorio de código es del cambio: %v", status.BlockedReasons)
	}
}

// REQ-25.43, escenario «Sin registro el comportamiento es el anterior»: sin registro, y con uno
// ilegible, `allowedEditRoots` es el workspace más las concesiones; el registro ilegible deja un
// aviso en `notes` y no bloquea.
func TestEditAuthorityWithoutARegistryKeepsTheRootsOfBeforeTheIncrement(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "feat/x")
	seedReadyChange(t, ws.Master, branchChange, "- [ ] 1.1 Editar `internal/api/handler.go`\n")
	want := []string{ws.Master}

	status := resolveFrom(t, ws.Master, branchChange)
	if !reflect.DeepEqual(allowedRoots(status), want) || status.ApplyState != ApplyReady {
		t.Fatalf("sin registro allowedEditRoots = %v y applyState %q, se esperaba %v y ready", allowedRoots(status), status.ApplyState, want)
	}

	common, err := gitcommon.CommonDirFromFiles(ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(common, "axiom", "branch", "v1", "sdd", branchChange+".json"), "{no es json")
	status = resolveFrom(t, ws.Master, branchChange)
	if !reflect.DeepEqual(allowedRoots(status), want) || status.ApplyState != ApplyReady {
		t.Errorf("con un registro ilegible allowedEditRoots = %v y applyState %q, se esperaba %v y ready", allowedRoots(status), status.ApplyState, want)
	}
	if !containsReason(status.Notes, "note(registered_roots_unchecked)") {
		t.Errorf("notes = %v, se esperaba el aviso note(registered_roots_unchecked)", status.Notes)
	}
	if containsReason(status.BlockedReasons, "registered_roots") {
		t.Errorf("el aviso no es un bloqueo: %v", status.BlockedReasons)
	}
}
