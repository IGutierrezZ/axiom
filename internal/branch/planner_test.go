package branch

import (
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas del planificador son tablas sin git ni disco (REQ-25.4, REQ-25.5,
// REQ-25.11, REQ-25.13): `Plan` es una función pura sobre los hechos que reúne el
// motor, así que cada escenario se escribe como una entrada y una salida esperada.

var (
	testMaster       = filepath.FromSlash("/proyectos/axiom")
	testWorktreeRoot = DefaultWorktreeRoot(testMaster)
)

// planningFacts son los hechos del repositorio del planning home: la propia carpeta
// maestra. roles son los roles de axiom.yaml cuyas rutas resuelven a esa raíz git.
func planningFacts(roles ...string) RepoFacts {
	return RepoFacts{GitRoot: testMaster, CommonDir: filepath.Join(testMaster, ".git"), RelPath: ".", DeclaredPath: ".", Roles: roles}
}

// codeFacts son los hechos de un repositorio declarado, anidado en la carpeta maestra.
func codeFacts(declared string, roles ...string) RepoFacts {
	root := filepath.Join(testMaster, filepath.FromSlash(declared))
	return RepoFacts{GitRoot: root, CommonDir: filepath.Join(root, ".git"), RelPath: path.Clean(declared), DeclaredPath: declared, Roles: roles}
}

// embeddedInput es la entrada de un monorepo embebido. Con roles sellados el cambio
// tiene kickoff; sin ellos, no (antes del sellado).
func embeddedInput(change string, sealed ...string) PlanInput {
	return PlanInput{
		Operation:     "ensure",
		Lane:          LaneSDD,
		Change:        change,
		Topology:      workspace.TopologyMonorepoEmbedded,
		Master:        testMaster,
		WorkspaceRoot: testMaster,
		WorkspaceRel:  ".",
		PlanningRepo:  planningFacts(),
		HasKickoff:    len(sealed) > 0,
		SealedRoles:   sealed,
		Policy:        ResolvePolicy(nil),
		WorktreeRoot:  testWorktreeRoot,
	}
}

func mustPlan(t *testing.T, in PlanInput) ChangePlan {
	t.Helper()
	plan, refusal := Plan(in)
	if refusal != nil {
		t.Fatalf("Plan rechazó la entrada: %s: %s", refusal.Code, refusal.Message)
	}
	return plan
}

func mustRefuse(t *testing.T, in PlanInput, want Code) *Refusal {
	t.Helper()
	plan, refusal := Plan(in)
	if refusal == nil {
		t.Fatalf("Plan debía rechazar con %s y devolvió %+v", want, plan)
	}
	if refusal.Code != want {
		t.Fatalf("código = %s (%s), se esperaba %s", refusal.Code, refusal.Message, want)
	}
	if !reflect.DeepEqual(plan, ChangePlan{}) {
		t.Errorf("un rechazo no devuelve plan, devolvió %+v", plan)
	}
	if refusal.Message == "" || refusal.Continuation == "" {
		t.Errorf("todo rechazo lleva mensaje y continuación: %+v", refusal)
	}
	return refusal
}

func branchNames(repo RepoPlan) []string {
	var names []string
	for _, branch := range repo.Branches {
		names = append(names, branch.Name)
	}
	return names
}

func wantNames(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, se esperaba %v", what, got, want)
	}
}

func repositoryNames(plan ChangePlan) []string {
	var names []string
	for _, repo := range plan.Repositories {
		names = append(names, repo.Name)
	}
	return names
}

// REQ-25.4 (escenario «Un rol, rama única») y REQ-25.11 (ruta sin sufijo de rol): con
// un rol solo existe `<prefijo>/<cambio>`, con PR hacia la integración y su worktree
// en `<raíz>/<cambio>/<repo>`; `fullstack` también cuenta como un único rol.
func TestPlanEmbeddedWithOneRoleHasASingleChangeBranch(t *testing.T) {
	for name, tc := range map[string]struct {
		sealed   []string
		declared []string
	}{
		"un rol declarado": {sealed: []string{"core"}, declared: []string{"core"}},
		"fullstack":        {sealed: []string{"fullstack"}},
	} {
		t.Run(name, func(t *testing.T) {
			in := embeddedInput("x", tc.sealed...)
			in.PlanningRepo = planningFacts(tc.declared...)
			plan := mustPlan(t, in)

			if plan.Lane != LaneSDD || plan.Change != "x" {
				t.Errorf("lane %q, change %q; se esperaba sdd y x", plan.Lane, plan.Change)
			}
			wantNames(t, "repositorios", repositoryNames(plan), "axiom")
			repo := plan.Repositories[0]
			if repo.Role != RepoChangeBranch || repo.IntegrationBranch != "main" {
				t.Errorf("role %q, integración %q; se esperaba change_branch y main", repo.Role, repo.IntegrationBranch)
			}
			if !reflect.DeepEqual(repo.Branches, []BranchPlan{{
				Name:     "feat/x",
				Kind:     KindChange,
				PRTarget: "main",
				Worktree: filepath.Join(testWorktreeRoot, "x", "axiom"),
			}}) {
				t.Errorf("ramas = %+v, se esperaba solo feat/x con su worktree sin sufijo de rol", repo.Branches)
			}
			if len(plan.Warnings) != 0 {
				t.Errorf("un plan de una sola rama no avisa de nada: %v", plan.Warnings)
			}
		})
	}
}

// REQ-25.4 (escenario «Dos roles sobre el mismo repositorio crean seguimiento y
// ramas de rol») y REQ-25.11 (ruta con sufijo de rol): la de seguimiento más una
// rama por rol, en orden alfabético, con el PR de los roles hacia la de seguimiento.
func TestPlanEmbeddedWithTwoRolesCreatesTrackingAndRoleBranches(t *testing.T) {
	in := embeddedInput("x", "qa", "core")
	in.PlanningRepo = planningFacts("core", "qa")
	plan := mustPlan(t, in)

	repo := plan.Repositories[0]
	want := []BranchPlan{
		{Name: "feat/x", Kind: KindChange, PRTarget: "main", Worktree: filepath.Join(testWorktreeRoot, "x", "axiom")},
		{Name: "feat/x-core", Kind: KindRole, RoleID: "core", From: "feat/x", PRTarget: "feat/x", Worktree: filepath.Join(testWorktreeRoot, "x-core", "axiom")},
		{Name: "feat/x-qa", Kind: KindRole, RoleID: "qa", From: "feat/x", PRTarget: "feat/x", Worktree: filepath.Join(testWorktreeRoot, "x-qa", "axiom")},
	}
	if !reflect.DeepEqual(repo.Branches, want) {
		t.Errorf("ramas = %+v\nse esperaba %+v", repo.Branches, want)
	}
	wantNames(t, "roles del repositorio", repo.Roles, "core", "qa")
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "Publica la rama de seguimiento feat/x") || !strings.Contains(plan.Warnings[0], "antes del primer PR de rol") {
		t.Errorf("avisos = %v, se esperaba el de publicar la rama de seguimiento antes del primer PR de rol", plan.Warnings)
	}
}

// REQ-25.4: `fullstack` es el único rol que toca la unión de los repositorios
// declarados y el del planning home (S-3); como cuenta como un solo rol, ningún
// repositorio genera ramas de rol. Los repositorios salen en orden estable: primero
// el del planning home y después por nombre, sin depender del orden de entrada.
func TestPlanFullstackTouchesEveryDeclaredRepositoryAndThePlanningHome(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	in.PlanningRepo = planningFacts("core", "qa")
	in.CodeRepos = []RepoFacts{codeFacts("repo-frontend", "web"), codeFacts("repo-backend", "api")}
	plan := mustPlan(t, in)

	wantNames(t, "repositorios", repositoryNames(plan), "axiom", "repo-backend", "repo-frontend")
	for _, repo := range plan.Repositories {
		wantNames(t, repo.Name+": roles", repo.Roles, "fullstack")
		wantNames(t, repo.Name+": ramas", branchNames(repo), "feat/x")
		if repo.Role != RepoChangeBranch {
			t.Errorf("%s: role %q, se esperaba change_branch (monorepo embebido)", repo.Name, repo.Role)
		}
	}
	if len(plan.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno", plan.Warnings)
	}
}

// REQ-25.4 (escenario «Un rol sellado que no toca ningún repositorio declarado no
// crea ramas»): un rol desconocido no aporta nada y genera un aviso que lo nombra.
func TestPlanAnUnknownRoleContributesNothingAndWarns(t *testing.T) {
	in := embeddedInput("x", "core", "fantasma")
	in.PlanningRepo = planningFacts("core")
	plan := mustPlan(t, in)

	repo := plan.Repositories[0]
	wantNames(t, "ramas", branchNames(repo), "feat/x")
	wantNames(t, "roles", repo.Roles, "core")
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "«fantasma»") {
		t.Errorf("avisos = %v, se esperaba uno que nombre el rol fantasma", plan.Warnings)
	}
}

// Un rol cuya ruta se omitió (no es un repositorio git) ya tiene su mensaje en
// `skipped`: no recibe además el aviso de rol desconocido (REQ-25.6).
func TestPlanARoleWithASkippedPathDoesNotWarnAsUnknown(t *testing.T) {
	skipped := SkippedPath{Path: filepath.Join(testMaster, "scripts"), Role: "tooling", Code: CodeNotAGitRepository, Message: "no es un repositorio"}
	in := embeddedInput("x", "tooling")
	in.Skipped = []SkippedPath{skipped}
	plan := mustPlan(t, in)

	if len(plan.Warnings) != 0 {
		t.Errorf("avisos = %v, el rol ya figura en skipped", plan.Warnings)
	}
	if !reflect.DeepEqual(plan.Skipped, []SkippedPath{skipped}) {
		t.Errorf("skipped = %+v, se esperaba la ruta omitida tal cual", plan.Skipped)
	}
}

// REQ-25.4 / design D-04 (S-1): antes del sellado el plan contiene solo la rama del
// cambio en el planning home; los repositorios de código que declaran los roles aún
// no se conocen como tocados.
func TestPlanBeforeSealingHasOnlyTheChangeBranchInThePlanningHome(t *testing.T) {
	in := embeddedInput("x")
	in.PlanningRepo = planningFacts("core", "qa")
	in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "api")}
	plan := mustPlan(t, in)

	wantNames(t, "repositorios", repositoryNames(plan), "axiom")
	wantNames(t, "ramas", branchNames(plan.Repositories[0]), "feat/x")
	if len(plan.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno antes del sellado", plan.Warnings)
	}
}

// El modelo se evalúa por repositorio y el plan se completa de forma incremental tras
// el sellado: un repositorio puede tener rama de rol y otro no, y un repositorio que
// ningún rol sellado toca no entra.
func TestPlanAfterSealingEvaluatesTheModelPerRepository(t *testing.T) {
	t.Run("cada repositorio con los roles que lo tocan", func(t *testing.T) {
		in := embeddedInput("x", "core", "qa")
		in.PlanningRepo = planningFacts("core")
		in.CodeRepos = []RepoFacts{codeFacts("tests", "qa"), codeFacts("docs", "escritor")}
		plan := mustPlan(t, in)

		wantNames(t, "repositorios", repositoryNames(plan), "axiom", "tests")
		wantNames(t, "ramas de axiom", branchNames(plan.Repositories[0]), "feat/x")
		wantNames(t, "ramas de tests", branchNames(plan.Repositories[1]), "feat/x")
	})
	t.Run("un repositorio compartido por los dos roles genera ramas de rol y el otro no", func(t *testing.T) {
		in := embeddedInput("x", "core", "qa")
		in.PlanningRepo = planningFacts("core", "qa")
		in.CodeRepos = []RepoFacts{codeFacts("tests", "qa")}
		plan := mustPlan(t, in)

		wantNames(t, "ramas de axiom", branchNames(plan.Repositories[0]), "feat/x", "feat/x-core", "feat/x-qa")
		wantNames(t, "ramas de tests", branchNames(plan.Repositories[1]), "feat/x")
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "axiom") {
			t.Errorf("avisos = %v, se esperaba un único aviso, el del repositorio con ramas de rol", plan.Warnings)
		}
	})
}

// REQ-25.4 / design D-04 (S-3): los roles sin coincidencia de mayúsculas se
// reconocen igual, y la entrada no se altera (ni se reordena) al planificar.
func TestPlanMatchesRolesIgnoringCaseAndDoesNotMutateItsInput(t *testing.T) {
	in := embeddedInput("x", "QA", "Core")
	in.PlanningRepo = planningFacts("core", "qa")
	in.CodeRepos = []RepoFacts{codeFacts("b", "z"), codeFacts("a", "z")}
	before := PlanInput{SealedRoles: append([]string(nil), in.SealedRoles...), CodeRepos: append([]RepoFacts(nil), in.CodeRepos...)}

	plan := mustPlan(t, in)

	wantNames(t, "ramas", branchNames(plan.Repositories[0]), "feat/x", "feat/x-Core", "feat/x-QA")
	if !reflect.DeepEqual(in.SealedRoles, before.SealedRoles) || !reflect.DeepEqual(in.CodeRepos, before.CodeRepos) {
		t.Errorf("Plan alteró su entrada: roles %v, repositorios %v", in.SealedRoles, in.CodeRepos)
	}
}

// El plan es determinista: la misma entrada da el mismo plan, sea cual sea el orden
// de los roles y de los repositorios que se le entregan.
func TestPlanIsDeterministic(t *testing.T) {
	first := embeddedInput("x", "fullstack")
	first.PlanningRepo = planningFacts("core")
	first.CodeRepos = []RepoFacts{codeFacts("b", "x"), codeFacts("a", "y"), codeFacts("c", "z")}
	second := first
	second.CodeRepos = []RepoFacts{codeFacts("c", "z"), codeFacts("a", "y"), codeFacts("b", "x")}

	wantPlan := mustPlan(t, first)
	for i := 0; i < 3; i++ {
		if got := mustPlan(t, first); !reflect.DeepEqual(got, wantPlan) {
			t.Fatalf("la misma entrada dio dos planes distintos:\n%+v\n%+v", got, wantPlan)
		}
	}
	if got := mustPlan(t, second); !reflect.DeepEqual(got, wantPlan) {
		t.Errorf("el orden de entrada de los repositorios cambió el plan:\n%+v\n%+v", got, wantPlan)
	}
}

// REQ-25.13 y design D-04: el carril ODD no tiene roles ni archivado, así que solo
// recibe la rama de la funcionalidad en el planning home, y el documento ODD es
// `<workspaceRoot del plan>/odd/tasks/<feature>.md`.
func TestPlanOddLaneHasNoRolesAndNamesTheOddDocument(t *testing.T) {
	in := embeddedInput("mi-feature", "core", "qa")
	in.Lane = LaneODD
	in.PlanningRepo = planningFacts("core", "qa")
	in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "core")}
	plan := mustPlan(t, in)

	if plan.Lane != LaneODD {
		t.Errorf("lane = %q, se esperaba odd", plan.Lane)
	}
	wantNames(t, "repositorios", repositoryNames(plan), "axiom")
	repo := plan.Repositories[0]
	wantNames(t, "ramas", branchNames(repo), "feat/mi-feature")
	if len(repo.Roles) != 0 || len(plan.Warnings) != 0 {
		t.Errorf("roles %v, avisos %v; el carril ODD no divide en roles", repo.Roles, plan.Warnings)
	}
	wantWorkspace := filepath.Join(testWorktreeRoot, "mi-feature", "axiom")
	if want := filepath.Join(wantWorkspace, "odd", "tasks", "mi-feature.md"); plan.OddDocument != want {
		t.Errorf("oddDocument = %q, se esperaba %q", plan.OddDocument, want)
	}

	t.Run("el carril SDD no tiene documento ODD", func(t *testing.T) {
		if got := mustPlan(t, embeddedInput("x", "core")).OddDocument; got != "" {
			t.Errorf("oddDocument = %q, se esperaba vacío", got)
		}
	})
}

// El workspaceRoot y el planning home del plan están en el worktree de la rama del
// cambio, con la misma ruta relativa a la raíz git que tienen en el checkout (una
// carpeta maestra puede ser una subcarpeta del repositorio).
func TestPlanPlacesTheWorkspaceInsideTheChangeWorktree(t *testing.T) {
	for name, tc := range map[string]struct {
		rel  string
		want []string
	}{
		"la carpeta maestra es la raíz git":    {rel: ".", want: nil},
		"la carpeta maestra es una subcarpeta": {rel: "app/web", want: []string{"app", "web"}},
	} {
		t.Run(name, func(t *testing.T) {
			in := embeddedInput("x", "core")
			in.WorkspaceRel = tc.rel
			plan := mustPlan(t, in)

			wantRoot := filepath.Join(append([]string{testWorktreeRoot, "x", "axiom"}, tc.want...)...)
			if plan.WorkspaceRoot != wantRoot {
				t.Errorf("workspaceRoot = %q, se esperaba %q", plan.WorkspaceRoot, wantRoot)
			}
			if want := filepath.Join(wantRoot, "openspec"); plan.PlanningHome != want {
				t.Errorf("planningHome = %q, se esperaba %q", plan.PlanningHome, want)
			}
			if plan.WorktreeRoot != testWorktreeRoot {
				t.Errorf("worktreeRoot = %q, se esperaba %q", plan.WorktreeRoot, testWorktreeRoot)
			}
		})
	}
}

// REQ-25.11 (S-17): `<repo>` es el último segmento de la ruta declarada y, para `.`, el
// nombre del directorio de la raíz git.
func TestPlanNamesEachRepositoryByTheLastSegmentOfItsDeclaredPath(t *testing.T) {
	tests := []struct {
		name     string
		declared string
		want     string
	}{
		{"subcarpeta declarada", "services/repo-backend", "repo-backend"},
		{"ruta con barra invertida", `services\api`, "api"},
		{"ruta con barra final", "repo-front/", "repo-front"},
		{"un solo segmento", "web", "web"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := embeddedInput("x", "core")
			facts := codeFacts("services/anidado", "core")
			facts.DeclaredPath = tc.declared
			in.CodeRepos = []RepoFacts{facts}
			plan := mustPlan(t, in)

			repo := plan.Repositories[1]
			if repo.Name != tc.want {
				t.Errorf("nombre = %q, se esperaba %q", repo.Name, tc.want)
			}
			if got, want := repo.Branches[0].Worktree, filepath.Join(testWorktreeRoot, "x", tc.want); got != want {
				t.Errorf("worktree = %q, se esperaba %q", got, want)
			}
		})
	}

	for _, declared := range []string{".", "./", "", ".."} {
		t.Run("la ruta «"+declared+"» toma el nombre del directorio de la raíz git", func(t *testing.T) {
			in := embeddedInput("x")
			in.PlanningRepo.DeclaredPath = declared
			if got := mustPlan(t, in).Repositories[0].Name; got != "axiom" {
				t.Errorf("nombre = %q, se esperaba axiom (el directorio de la raíz git)", got)
			}
		})
	}
}

// REQ-25.11, escenario «Raíz de worktrees por defecto fuera de cualquier
// repositorio»: `<P>/axiom` da `<P>/axiom-wt`.
func TestDefaultWorktreeRootIsASiblingOfTheMasterFolder(t *testing.T) {
	parent := filepath.FromSlash("/trabajo/proyectos")
	master := filepath.Join(parent, "axiom")
	if got, want := DefaultWorktreeRoot(master), filepath.Join(parent, "axiom-wt"); got != want {
		t.Errorf("DefaultWorktreeRoot(%q) = %q, se esperaba %q", master, got, want)
	}
	if got, want := DefaultWorktreeRoot(master+string(filepath.Separator)), filepath.Join(parent, "axiom-wt"); got != want {
		t.Errorf("con separador final: %q, se esperaba %q", got, want)
	}
	if rel, err := filepath.Rel(master, DefaultWorktreeRoot(master)); err != nil || !strings.HasPrefix(rel, "..") {
		t.Errorf("la raíz por defecto debe quedar fuera de la carpeta maestra, ruta relativa %q (%v)", rel, err)
	}
}

// REQ-25.11 (escenario «Nombre con carácter inválido en Windows») y design D-04:
// `<cambio>` y `<rol>` inválidos dan `invalid_name` antes de planificar nada.
func TestPlanRejectsInvalidChangeAndRoleNamesWithInvalidName(t *testing.T) {
	roleInput := func(roles ...string) PlanInput {
		in := embeddedInput("x", roles...)
		in.PlanningRepo = planningFacts(roles...)
		return in
	}
	tests := []struct {
		name      string
		in        PlanInput
		wantField string
		wantValue string
		wantText  []string
	}{
		{"cambio con espacio y dos puntos", embeddedInput("a b:c", "core"), "--change", "a b:c", []string{"«a b:c»", "cambio"}},
		{"cambio con dos puntos", embeddedInput("a:b"), "--change", "a:b", nil},
		{"cambio con una barra", embeddedInput("a/b"), "--change", "a/b", nil},
		{"cambio reservado de Windows", embeddedInput("CON"), "--change", "CON", []string{"reservado de Windows"}},
		{"cambio vacío", embeddedInput(""), "--change", "", nil},
		{"cambio que empieza por punto", embeddedInput(".oculto"), "--change", ".oculto", nil},
		{"rol inválido", roleInput("core", "qa:x"), "role", "qa:x", []string{"«qa:x»", "rol"}},
		{"rol que se llama archive", roleInput("core", "archive"), "role", "archive", []string{"«archive»", "feat/x-archive"}},
		{"rol Archive con otras mayúsculas", roleInput("core", "Archive"), "role", "Archive", []string{"feat/x-archive"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refusal := mustRefuse(t, tc.in, CodeInvalidName)
			if refusal.Details["field"] != tc.wantField || refusal.Details["value"] != tc.wantValue {
				t.Errorf("details = %v, se esperaba field %q y value %q", refusal.Details, tc.wantField, tc.wantValue)
			}
			for _, want := range tc.wantText {
				if !strings.Contains(refusal.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
				}
			}
			for _, want := range []string{"axiom branch ensure", pathquote.Quote(testMaster)} {
				if !strings.Contains(refusal.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
				}
			}
		})
	}

	t.Run("un rol inválido que no genera rama de rol no se rechaza", func(t *testing.T) {
		in := embeddedInput("x", "qa:x")
		in.PlanningRepo = planningFacts("qa:x")
		wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "feat/x")
	})
}

// ValidateChangeName es la comprobación previa del motor, antes de componer ninguna
// ruta con el nombre: rechaza con `invalid_name` y la continuación cita la orden y
// el directorio de partida.
func TestValidateChangeName(t *testing.T) {
	if refusal := ValidateChangeName("status", testMaster, "inc-25"); refusal != nil {
		t.Errorf("un nombre válido no se rechaza: %+v", refusal)
	}
	for _, invalid := range []string{"", "a b:c", "../fuera", `a\b`, "NUL"} {
		refusal := ValidateChangeName("status", testMaster, invalid)
		if refusal == nil || refusal.Code != CodeInvalidName {
			t.Errorf("ValidateChangeName(%q) = %+v, se esperaba invalid_name", invalid, refusal)
			continue
		}
		for _, want := range []string{"axiom branch status --change <cambio> --cwd", pathquote.Quote(testMaster)} {
			if !strings.Contains(refusal.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
			}
		}
	}
}

// policyConfig construye la configuración que deja la carga de axiom.yaml.
func policyConfig(integration, prefix string, repositories ...workspace.RepositoryPolicy) *workspace.WorkspaceConfig {
	return &workspace.WorkspaceConfig{
		Git:          &workspace.GitConfig{IntegrationBranch: integration, BranchPrefix: prefix},
		Repositories: repositories,
	}
}

// REQ-25.3: sin configuración rigen `main` y `feat`; el bloque `git:` los sustituye;
// un repositorio con ajustes propios prevalece sobre `git:` solo en su repositorio.
func TestPlanResolvesTheBranchPolicyPerRepository(t *testing.T) {
	t.Run("sin configuración rigen main y feat", func(t *testing.T) {
		repo := mustPlan(t, embeddedInput("x")).Repositories[0]
		if repo.IntegrationBranch != "main" || repo.Branches[0].Name != "feat/x" || repo.Branches[0].PRTarget != "main" {
			t.Errorf("integración %q, rama %+v; se esperaba main y feat/x hacia main", repo.IntegrationBranch, repo.Branches[0])
		}
	})
	t.Run("el bloque git sustituye los valores por defecto", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = ResolvePolicy(policyConfig("develop", "inc"))
		repo := mustPlan(t, in).Repositories[0]
		if repo.IntegrationBranch != "develop" || repo.Branches[0].Name != "inc/x" || repo.Branches[0].PRTarget != "develop" {
			t.Errorf("integración %q, rama %+v; se esperaba develop e inc/x hacia develop", repo.IntegrationBranch, repo.Branches[0])
		}
	})
	t.Run("un repositorio con ajustes propios prevalece sobre git", func(t *testing.T) {
		in := embeddedInput("x", "api", "web")
		in.PlanningRepo = planningFacts()
		in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "api"), codeFacts("repo-frontend", "web")}
		in.Policy = ResolvePolicy(policyConfig("main", "feat",
			workspace.RepositoryPolicy{Path: "repo-backend", IntegrationBranch: "release"},
			workspace.RepositoryPolicy{Path: "repo-frontend", BranchPrefix: "ui"}))
		plan := mustPlan(t, in)

		byName := map[string]RepoPlan{}
		for _, repo := range plan.Repositories {
			byName[repo.Name] = repo
		}
		backend, frontend := byName["repo-backend"], byName["repo-frontend"]
		if backend.IntegrationBranch != "release" || backend.Branches[0].PRTarget != "release" || backend.Branches[0].Name != "feat/x" {
			t.Errorf("repo-backend: integración %q, rama %+v; se esperaba release y feat/x hacia release", backend.IntegrationBranch, backend.Branches[0])
		}
		if frontend.IntegrationBranch != "main" || frontend.Branches[0].PRTarget != "main" || frontend.Branches[0].Name != "ui/x" {
			t.Errorf("repo-frontend: integración %q, rama %+v; se esperaba main y ui/x hacia main", frontend.IntegrationBranch, frontend.Branches[0])
		}
		if byName["axiom"].IntegrationBranch != "main" {
			t.Errorf("el repositorio sin entrada hereda git: integración %q", byName["axiom"].IntegrationBranch)
		}
	})
	t.Run("un repositorio fuera de la carpeta maestra toma la política del workspace", func(t *testing.T) {
		in := embeddedInput("x")
		in.PlanningRepo.RelPath = ""
		in.Policy = ResolvePolicy(policyConfig("develop", "feat", workspace.RepositoryPolicy{Path: ".", IntegrationBranch: "release"}))
		if got := mustPlan(t, in).Repositories[0].IntegrationBranch; got != "develop" {
			t.Errorf("integración = %q; la entrada «.» es de la carpeta maestra, no de un repositorio que la contiene (se esperaba develop)", got)
		}
	})
}

// REQ-25.3 (escenario «La excepción por cambio prevalece sobre todo lo demás») y
// REQ-25.12: las excepciones de la invocación van encima de la entrada del
// repositorio, del bloque `git:` y de los valores por defecto, en todos los
// repositorios del plan.
func TestPlanExceptionsPrevailOverTheWholePolicy(t *testing.T) {
	twoRepositories := func() PlanInput {
		in := embeddedInput("x", "api", "web")
		in.PlanningRepo = planningFacts()
		in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "api"), codeFacts("repo-frontend", "web")}
		in.Policy = ResolvePolicy(policyConfig("develop", "inc",
			workspace.RepositoryPolicy{Path: "repo-backend", IntegrationBranch: "release", BranchPrefix: "be"},
			workspace.RepositoryPolicy{Path: "repo-frontend", BranchPrefix: "ui"}))
		return in
	}

	t.Run("--prefix fija el prefijo en todos los repositorios", func(t *testing.T) {
		in := twoRepositories()
		in.Exceptions = Exceptions{Prefix: "fix"}
		plan := mustPlan(t, in)

		if len(plan.Repositories) != 3 {
			t.Fatalf("repositorios = %v, se esperaban tres", repositoryNames(plan))
		}
		for _, repo := range plan.Repositories {
			wantNames(t, repo.Name+": ramas", branchNames(repo), "fix/x")
		}
		// El prefijo es lo único que cambia: la integración sigue siendo la de la política.
		for name, want := range map[string]string{"axiom": "develop", "repo-backend": "release", "repo-frontend": "develop"} {
			for _, repo := range plan.Repositories {
				if repo.Name == name && repo.IntegrationBranch != want {
					t.Errorf("%s: integración %q, se esperaba %q", name, repo.IntegrationBranch, want)
				}
			}
		}
	})
	t.Run("--branch fija el nombre de la rama del cambio y deriva el de las ramas de rol", func(t *testing.T) {
		in := embeddedInput("x", "core", "qa")
		in.PlanningRepo = planningFacts("core", "qa")
		in.Exceptions = Exceptions{Branch: "inc/x"}
		repo := mustPlan(t, in).Repositories[0]

		wantNames(t, "ramas", branchNames(repo), "inc/x", "inc/x-core", "inc/x-qa")
		if repo.Branches[1].From != "inc/x" || repo.Branches[1].PRTarget != "inc/x" || repo.Branches[0].PRTarget != "main" {
			t.Errorf("las ramas de rol salen de la de seguimiento y su PR va hacia ella: %+v", repo.Branches)
		}
	})
	t.Run("--branch gana a --prefix", func(t *testing.T) {
		in := embeddedInput("x")
		in.Exceptions = Exceptions{Prefix: "fix", Branch: "inc/x"}
		wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "inc/x")
	})
	t.Run("--base fija la base de todos los repositorios", func(t *testing.T) {
		in := twoRepositories()
		in.Exceptions = Exceptions{Base: "origin/release"}
		for _, repo := range mustPlan(t, in).Repositories {
			if repo.BaseRef != "origin/release" {
				t.Errorf("%s: base %q, se esperaba origin/release", repo.Name, repo.BaseRef)
			}
		}
	})
	t.Run("sin --base la base no está fijada", func(t *testing.T) {
		if got := mustPlan(t, twoRepositories()).Repositories[0].BaseRef; got != "" {
			t.Errorf("base = %q, se esperaba vacío (la resuelve el motor desde la integración)", got)
		}
	})
}

// REQ-25.3 (escenarios «Un prefijo inválido de la invocación se rechaza sin efectos»
// y «Un prefijo inválido de `axiom.yaml` se rechaza como configuración»): el nombre
// inválido de la invocación es `invalid_name`; el de axiom.yaml, `config_invalid`.
func TestPlanDistinguishesAnInvalidInvocationNameFromAnInvalidConfiguration(t *testing.T) {
	t.Run("de la invocación: invalid_name", func(t *testing.T) {
		tests := []struct {
			name       string
			exceptions Exceptions
			wantField  string
		}{
			{"prefijo con espacio y barras", Exceptions{Prefix: "mi prefijo//"}, "--prefix"},
			{"prefijo con dos puntos", Exceptions{Prefix: "a b:c"}, "--prefix"},
			{"prefijo con barra final", Exceptions{Prefix: "inc/"}, "--prefix"},
			{"rama con dos puntos seguidos", Exceptions{Branch: "a..b"}, "--branch"},
			{"rama con espacio", Exceptions{Branch: "inc/mi cambio"}, "--branch"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				in := embeddedInput("x")
				in.Exceptions = tc.exceptions
				refusal := mustRefuse(t, in, CodeInvalidName)
				if refusal.Details["field"] != tc.wantField {
					t.Errorf("details = %v, se esperaba field %q", refusal.Details, tc.wantField)
				}
				for _, want := range []string{"axiom branch ensure --change x " + tc.wantField, pathquote.Quote(testMaster)} {
					if !strings.Contains(refusal.Continuation, want) {
						t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
					}
				}
			})
		}
	})

	t.Run("de axiom.yaml: config_invalid", func(t *testing.T) {
		tests := []struct {
			name      string
			policy    Policy
			wantField string
			wantRepo  string
		}{
			{"prefijo del workspace", Policy{IntegrationBranch: "main", BranchPrefix: "inc//"}, "branch_prefix", "axiom"},
			{"integración del workspace", Policy{IntegrationBranch: "a b", BranchPrefix: "feat"}, "integration_branch", "axiom"},
			{"política vacía", Policy{}, "integration_branch", "axiom"},
			{"prefijo propio de un repositorio", Policy{
				IntegrationBranch: "main", BranchPrefix: "feat",
				PerRepository: map[string]RepoPolicy{".": {BranchPrefix: "mi prefijo//"}},
			}, "branch_prefix", "axiom"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				in := embeddedInput("x")
				in.Policy = tc.policy
				refusal := mustRefuse(t, in, CodeConfigInvalid)
				if refusal.Details["field"] != tc.wantField || refusal.Details["repository"] != tc.wantRepo {
					t.Errorf("details = %v, se esperaba field %q y repository %q", refusal.Details, tc.wantField, tc.wantRepo)
				}
				for _, want := range []string{"axiom workspace validate --path", pathquote.Quote(testMaster)} {
					if !strings.Contains(refusal.Continuation, want) {
						t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
					}
				}
			})
		}
	})

	t.Run("la excepción válida sustituye a un prefijo de configuración que no se usa", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = Policy{IntegrationBranch: "main", BranchPrefix: "inc//"}
		in.Exceptions = Exceptions{Prefix: "fix"}
		wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "fix/x")
	})
}

// REQ-25.5 y design D-04: en `multirepo` el repositorio de specs se trabaja in situ
// (sin ramas ni worktrees) y cada repositorio de código recibe el modelo de REQ-25.4;
// un rol cuyo único repositorio es el de specs (perfiles `knowledge` y `spec-only`)
// no genera ramas.
func TestPlanMultirepoWorksTheSpecsRepositoryInPlace(t *testing.T) {
	multirepoInput := func(sealed ...string) PlanInput {
		in := embeddedInput("x", sealed...)
		in.Topology = workspace.TopologyMultirepo
		return in
	}

	t.Run("el repositorio de specs no recibe ramas y los de código sí", func(t *testing.T) {
		in := multirepoInput("backend", "frontend")
		in.CodeRepos = []RepoFacts{codeFacts("repo-frontend", "frontend"), codeFacts("repo-backend", "backend")}
		plan := mustPlan(t, in)

		wantNames(t, "repositorios", repositoryNames(plan), "axiom", "repo-backend", "repo-frontend")
		specs := plan.Repositories[0]
		if specs.Role != RepoSpecsInPlace || len(specs.Branches) != 0 {
			t.Errorf("repositorio de specs: role %q, ramas %v; se esperaba specs_in_place sin ramas", specs.Role, specs.Branches)
		}
		for _, repo := range plan.Repositories[1:] {
			if repo.Role != RepoCode {
				t.Errorf("%s: role %q, se esperaba code", repo.Name, repo.Role)
			}
			wantNames(t, repo.Name+": ramas", branchNames(repo), "feat/x")
		}
		if plan.WorkspaceRoot != testMaster {
			t.Errorf("workspaceRoot = %q; el repositorio de specs se trabaja in situ (se esperaba %q)", plan.WorkspaceRoot, testMaster)
		}
	})
	t.Run("dos roles que comparten un repositorio de código", func(t *testing.T) {
		in := multirepoInput("api", "jobs", "frontend")
		in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "api", "jobs"), codeFacts("repo-frontend", "frontend")}
		plan := mustPlan(t, in)

		wantNames(t, "ramas de repo-backend", branchNames(plan.Repositories[1]), "feat/x", "feat/x-api", "feat/x-jobs")
		wantNames(t, "ramas de repo-frontend", branchNames(plan.Repositories[2]), "feat/x")
	})
	t.Run("un rol que solo toca el repositorio de specs no genera ramas ni avisos", func(t *testing.T) {
		in := multirepoInput("knowledge")
		in.PlanningRepo = planningFacts("knowledge")
		plan := mustPlan(t, in)

		wantNames(t, "repositorios", repositoryNames(plan), "axiom")
		if len(plan.Repositories[0].Branches) != 0 || len(plan.Warnings) != 0 {
			t.Errorf("ramas %v, avisos %v; se esperaba ninguno", plan.Repositories[0].Branches, plan.Warnings)
		}
	})
	t.Run("el carril ODD trabaja in situ y su documento está en el repositorio de specs", func(t *testing.T) {
		in := multirepoInput()
		in.Lane = LaneODD
		in.Change = "mi-feature"
		plan := mustPlan(t, in)

		if want := filepath.Join(testMaster, "odd", "tasks", "mi-feature.md"); plan.OddDocument != want {
			t.Errorf("oddDocument = %q, se esperaba %q", plan.OddDocument, want)
		}
		if len(plan.Repositories[0].Branches) != 0 {
			t.Errorf("el repositorio de specs no recibe ramas: %v", plan.Repositories[0].Branches)
		}
	})
}
