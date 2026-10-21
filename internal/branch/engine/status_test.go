package engine

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración del
// usuario, con un autor fijo y sin red (tasks.md, regla 9), con el mismo patrón que
// internal/specbaseline, y fija el techo de descubrimiento de proyectos de las
// pruebas (workspace.DiscoveryCeilingOverride): `status` llama a workspace.Locate.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

// isolatedGitConfig es el gitconfig global de las pruebas: sin gc ni mantenimiento
// automático en segundo plano (que compite con la limpieza del directorio temporal) y
// sin fsmonitor, caché de ficheros sin seguimiento ni fsync, que solo encarecen cada
// proceso git en Windows.
const isolatedGitConfig = `[gc]
	auto = 0
	autoDetach = false
[maintenance]
	auto = false
[core]
	fsmonitor = false
	untrackedCache = false
	fsync = none
`

func runTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "engine-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "engine: crear el HOME aislado: %v\n", err)
		return 1
	}
	gitConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte(isolatedGitConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "engine: escribir el gitconfig aislado: %v\n", err)
		return 1
	}
	templateDir := filepath.Join(root, "git-template")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "engine: crear la plantilla vacía de git: %v\n", err)
		return 1
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"GIT_CONFIG_GLOBAL":   gitConfig,
		"GIT_TEMPLATE_DIR":    templateDir,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Autor de Prueba",
		"GIT_AUTHOR_EMAIL":    "autor@ejemplo.invalid",
		"GIT_COMMITTER_NAME":  "Autor de Prueba",
		"GIT_COMMITTER_EMAIL": "autor@ejemplo.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "engine: fijar %s: %v\n", key, err)
			return 1
		}
	}
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	return m.Run()
}

func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

// runGit ejecuta git directamente, sin pasar por gitcommon.Run, para que el oráculo de
// una prueba del motor no dependa del motor. Devuelve la salida sin el salto final.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

func commit(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", message)
}

func commits(t *testing.T, dir, prefix string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		commit(t, dir, fmt.Sprintf("%s %d", prefix, i))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepository crea un repositorio real en la rama main y con un commit.
func newRepository(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	initRepository(t, dir)
	return dir
}

func initRepository(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "-q", "-b", "main")
	commit(t, dir, "inicial")
}

// configWithRoles es un axiom.yaml de monorepo embebido. roles asocia cada rol con las
// rutas de sus repositorios; extra son bloques adicionales (`git:`, `repositories:`).
func configWithRoles(roles map[string][]string, extra string) string {
	return configFor("monorepo-embedded", ".", roles, extra)
}

// configFor es un axiom.yaml de la topología y del repositorio de specs dados.
func configFor(topology, specsRepository string, roles map[string][]string, extra string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "workspace:\n  name: ejemplo\n  topology: %s\n  specs_repository: %s\nroles:\n", topology, specsRepository)
	for _, name := range slices.Sorted(maps.Keys(roles)) {
		fmt.Fprintf(&builder, "  %s:\n    name: %s\n    repositories:\n", name, name)
		for _, path := range roles[name] {
			fmt.Fprintf(&builder, "      - path: %q\n", filepath.ToSlash(path))
		}
	}
	builder.WriteString(extra)
	return builder.String()
}

// newProject crea un monorepo embebido: un repositorio real con el axiom.yaml dado
// confirmado en main.
func newProject(t *testing.T, config string) string {
	t.Helper()
	dir := newRepository(t)
	writeFile(t, filepath.Join(dir, "axiom.yaml"), config)
	runGit(t, dir, "add", "axiom.yaml")
	commit(t, dir, "configuración")
	return dir
}

// sealChange sella el kickoff de un cambio con los roles dados, como lo hace
// `axiom sdd kickoff seal`.
func sealChange(t *testing.T, master, change string, roles ...string) {
	t.Helper()
	sealChangeAt(t, master, change, roles...)
}

// sealChangeAt sella el kickoff del cambio bajo el workspaceRoot dado: la carpeta
// maestra en un monorepo embebido y el repositorio de specs en un multirepo.
func sealChangeAt(t *testing.T, workspaceRoot, change string, roles ...string) {
	t.Helper()
	changeRoot := filepath.Join(workspaceRoot, "openspec", "changes", change)
	if err := os.MkdirAll(changeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	var sealed []kickoff.KickoffRole
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"})
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema:    kickoff.KickoffSchemaV1,
		Change:    change,
		SealedBy:  "prueba",
		Config:    kickoff.FlowConfig{FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit", HandoffPolicy: kickoff.HandoffNone, Roles: sealed},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("sellar el kickoff de %s: %v (publicado: %v)", change, err, ok)
	}
}

// observe instala un observador de las invocaciones de git y lo retira al terminar.
func observe(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &seen
}

func status(t *testing.T, cwd, change string, lane branch.Lane) branch.Result {
	t.Helper()
	return Default().Status(context.Background(), cwd, change, lane)
}

func wantSameDirectory(t *testing.T, what, got, want string) {
	t.Helper()
	if !filepath.IsAbs(got) || !pathidentity.SameDirectory(got, want) {
		t.Errorf("%s = %q, se esperaba el directorio absoluto %q", what, got, want)
	}
}

func wantReady(t *testing.T, result branch.Result) {
	t.Helper()
	if result.Status != branch.StatusReady || result.StatusLabel != "listo" || result.ExitCode() != branch.ExitReady {
		t.Fatalf("status %q / %q, salida %d, código %q: %s", result.Status, result.StatusLabel, result.ExitCode(), result.Code, result.Message)
	}
}

// wantReached comprueba que `status` alcanzó el estado pedido (código de salida 0) sin
// exigir que las ramas existan: un cambio sin ramas creadas es `not_started`.
func wantReached(t *testing.T, result branch.Result) {
	t.Helper()
	if result.ExitCode() != branch.ExitReady || result.Code != "" {
		t.Fatalf("status %q, salida %d, código %q: %s", result.Status, result.ExitCode(), result.Code, result.Message)
	}
}

func wantRefusal(t *testing.T, result branch.Result, code branch.Code, wantStatus branch.Status) {
	t.Helper()
	if result.Code != code || result.Status != wantStatus || result.ExitCode() != branch.ExitRefused {
		t.Fatalf("status %q, código %q (%s), salida %d; se esperaba %s con %s y salida 1", result.Status, result.Code, result.Message, result.ExitCode(), wantStatus, code)
	}
	if result.Message == "" || result.Continuation == "" {
		t.Errorf("todo rechazo lleva mensaje y continuación: %+v", result)
	}
	if len(result.Repositories) != 0 || result.Change != "" {
		t.Errorf("un rechazo no lleva datos del cambio: %+v", result)
	}
}

func onlyRepository(t *testing.T, result branch.Result) branch.RepositoryResult {
	t.Helper()
	if len(result.Repositories) != 1 {
		t.Fatalf("se esperaba un único repositorio, hay %d: %+v", len(result.Repositories), result.Repositories)
	}
	return result.Repositories[0]
}

func branchByName(t *testing.T, repository branch.RepositoryResult, name string) branch.BranchResult {
	t.Helper()
	for _, candidate := range repository.Branches {
		if candidate.Name == name {
			return candidate
		}
	}
	t.Fatalf("el repositorio %s no tiene la rama %s: %+v", repository.Name, name, repository.Branches)
	return branch.BranchResult{}
}

func intValue(t *testing.T, what string, value *int) int {
	t.Helper()
	if value == nil {
		t.Fatalf("%s no se calculó", what)
	}
	return *value
}

// remoteOperation dice si una invocación de git es de las que `status` no puede
// lanzar (REQ-25.1): fetch, pull, push o ls-remote.
func remoteOperation(subcommand string) bool {
	switch subcommand {
	case "fetch", "pull", "push", "ls-remote":
		return true
	}
	return false
}

// REQ-25.15, escenarios «Estado de un cambio con rama y worktree» y «Atraso calculado
// solo con referencias locales»: por repositorio, el papel; por rama, su worktree, su
// limpieza, su adelanto y su atraso respecto de la base, y el destino del PR. El
// observador no ve ninguna operación remota, ni ninguna fuera de la lista cerrada de
// lecturas.
func TestStatusOfAChangeWithABranchAndAWorktree(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	worktree := filepath.Join(t.TempDir(), "x", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree, "main")
	commit(t, worktree, "en feat/x")
	commits(t, repo, "en main", 3)
	mainSHA := runGit(t, repo, "rev-parse", "main")
	seen := observe(t)

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	if result.Operation != "status" || result.Change != "x" || result.Lane != branch.LaneSDD || result.Topology != workspace.TopologyMonorepoEmbedded {
		t.Errorf("operación %q, cambio %q, carril %q, topología %q", result.Operation, result.Change, result.Lane, result.Topology)
	}
	repository := onlyRepository(t, result)
	if repository.Name != filepath.Base(repo) || repository.Role != branch.RepoChangeBranch || repository.RoleLabel != "repositorio con rama de cambio" {
		t.Errorf("repositorio %q, role %q / %q", repository.Name, repository.Role, repository.RoleLabel)
	}
	wantSameDirectory(t, "gitRoot", repository.GitRoot, repo)
	wantSameDirectory(t, "commonDir", repository.CommonDir, filepath.Join(repo, ".git"))
	if repository.IntegrationBranch != "main" || repository.CurrentBranch != "main" || repository.Detached {
		t.Errorf("integración %q, rama actual %q, desacoplado %v", repository.IntegrationBranch, repository.CurrentBranch, repository.Detached)
	}
	if repository.Base == nil || *repository.Base != (branch.BaseResult{Ref: "main", SHA: mainSHA, Source: "local", Fetched: false}) {
		t.Errorf("base = %+v, se esperaba la rama local main en %s", repository.Base, mainSHA)
	}

	feature := branchByName(t, repository, "feat/x")
	if !feature.Exists || feature.Kind != branch.KindChange || feature.KindLabel != "rama del cambio" || feature.PRTarget != "main" {
		t.Errorf("rama del cambio = %+v", feature)
	}
	wantSameDirectory(t, "worktree", feature.Worktree, worktree)
	if feature.Dirty {
		t.Error("el worktree está limpio")
	}
	if feature.ComparedTo != "main" || intValue(t, "adelanto", feature.Ahead) != 1 || intValue(t, "atraso", feature.Behind) != 3 {
		t.Errorf("comparada con %q: adelanto %v, atraso %v; se esperaba main, 1 y 3", feature.ComparedTo, feature.Ahead, feature.Behind)
	}

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("status lanzó la operación remota git %s: %v", invocation[1], invocation)
		}
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list", "ls-tree":
		default:
			t.Errorf("status lanzó %q, fuera de la lista cerrada de lecturas: %v", invocation[1], invocation)
		}
	}

	t.Run("un fichero sin seguimiento ensucia el worktree", func(t *testing.T) {
		writeFile(t, filepath.Join(worktree, "pendiente.txt"), "x\n")
		if dirty := branchByName(t, onlyRepository(t, status(t, repo, "x", branch.LaneSDD)), "feat/x").Dirty; !dirty {
			t.Error("el worktree tiene un fichero sin seguimiento y debía figurar sucio")
		}
	})
	t.Run("el resultado es el mismo desde una subcarpeta y con el mismo estado", func(t *testing.T) {
		sub := filepath.Join(repo, "app", "web")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if got, want := status(t, sub, "x", branch.LaneSDD), status(t, repo, "x", branch.LaneSDD); !reflect.DeepEqual(got, want) {
			t.Errorf("desde una subcarpeta: %+v\ndesde la raíz: %+v", got, want)
		}
	})
}

// REQ-25.15: con la rama de integración remota ya traída a local, la base es esa
// referencia (`remote-tracking`) y el adelanto y el atraso se miden contra ella.
func TestStatusUsesTheLocalRemoteTrackingReferenceAsTheBase(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	remote := filepath.Join(t.TempDir(), "remoto.git")
	runGit(t, repo, "init", "-q", "--bare", remote)
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "main")
	remoteSHA := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
	runGit(t, repo, "branch", "feat/x")
	commits(t, repo, "en main", 2)

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if repository.Base == nil || *repository.Base != (branch.BaseResult{Ref: "origin/main", SHA: remoteSHA, Source: "remote-tracking"}) {
		t.Errorf("base = %+v, se esperaba origin/main en %s como referencia remota", repository.Base, remoteSHA)
	}
	feature := branchByName(t, repository, "feat/x")
	if feature.ComparedTo != "origin/main" || intValue(t, "adelanto", feature.Ahead) != 0 || intValue(t, "atraso", feature.Behind) != 0 {
		t.Errorf("comparada con %q: adelanto %v, atraso %v; se esperaba origin/main, 0 y 0 (main local avanzó, pero la base es la remota)", feature.ComparedTo, feature.Ahead, feature.Behind)
	}
	if feature.Worktree != "" || !feature.Exists {
		t.Errorf("la rama existe pero no está desplegada en ningún worktree: %+v", feature)
	}
}

// REQ-25.15, escenario «Cambio no iniciado»: un cambio sin ramas es `not_started`
// (etiqueta «no iniciado», salida 0), no un error, y el plan sigue visible.
func TestStatusOfAChangeWithoutBranchesIsNotStarted(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))

	result := status(t, repo, "nuevo", branch.LaneSDD)

	if result.Status != branch.StatusNotStarted || result.StatusLabel != "no iniciado" || result.ExitCode() != 0 || result.Code != "" {
		t.Fatalf("status %q / %q, salida %d, código %q; se esperaba not_started / no iniciado / 0 sin código", result.Status, result.StatusLabel, result.ExitCode(), result.Code)
	}
	repository := onlyRepository(t, result)
	feature := branchByName(t, repository, "feat/nuevo")
	if feature.Exists || feature.Worktree != "" || feature.Ahead != nil || feature.Behind != nil || feature.PRTarget != "main" {
		t.Errorf("la rama planificada no existe y se propone su PR hacia main: %+v", feature)
	}
	if result.Change != "nuevo" || result.Warnings != nil {
		t.Errorf("cambio %q, avisos %v", result.Change, result.Warnings)
	}
}

// REQ-25.4 (escenarios «Dos roles sobre el mismo repositorio…» y «La rama de
// archivado no pertenece al modelo de roles») y REQ-25.15 (escenario «Rama destino
// propuesta para el PR»): con dos roles sellados, el destino de cada rama de rol es la
// de seguimiento y el de esta, la integración; el adelanto de una rama de rol se mide
// contra la de seguimiento.
func TestStatusProposesThePRTargetOfEachBranch(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	sealChange(t, repo, "x", "core", "qa")
	runGit(t, repo, "branch", "feat/x")
	core := filepath.Join(t.TempDir(), "x-core", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x-core", core, "feat/x")
	commits(t, core, "en el rol core", 2)

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if !reflect.DeepEqual(repository.Roles, []string{"core", "qa"}) {
		t.Errorf("roles = %v, se esperaba [core qa]", repository.Roles)
	}
	var names []string
	for _, candidate := range repository.Branches {
		names = append(names, candidate.Name)
	}
	if want := []string{"feat/x", "feat/x-core", "feat/x-qa"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("ramas = %v, se esperaba %v", names, want)
	}
	tracking, coreBranch, qaBranch := branchByName(t, repository, "feat/x"), branchByName(t, repository, "feat/x-core"), branchByName(t, repository, "feat/x-qa")
	if tracking.PRTarget != "main" || coreBranch.PRTarget != "feat/x" || qaBranch.PRTarget != "feat/x" {
		t.Errorf("destinos del PR: seguimiento %q, core %q, qa %q; se esperaba main, feat/x y feat/x", tracking.PRTarget, coreBranch.PRTarget, qaBranch.PRTarget)
	}
	if coreBranch.Kind != branch.KindRole || coreBranch.KindLabel != "rama de rol" || coreBranch.Role != "core" {
		t.Errorf("rama de rol = %+v", coreBranch)
	}
	wantSameDirectory(t, "worktree de core", coreBranch.Worktree, core)
	if coreBranch.ComparedTo != "feat/x" || intValue(t, "adelanto", coreBranch.Ahead) != 2 || intValue(t, "atraso", coreBranch.Behind) != 0 {
		t.Errorf("core comparada con %q: adelanto %v, atraso %v; se esperaba feat/x, 2 y 0", coreBranch.ComparedTo, coreBranch.Ahead, coreBranch.Behind)
	}
	if !tracking.Exists || !coreBranch.Exists || qaBranch.Exists || qaBranch.Ahead != nil {
		t.Errorf("existen la de seguimiento y la de core, pero no la de qa: %+v / %+v / %+v", tracking, coreBranch, qaBranch)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Publica la rama de seguimiento feat/x") {
		t.Errorf("avisos = %v, se esperaba el de publicar la rama de seguimiento", result.Warnings)
	}
}

// REQ-25.4 (S-1): antes del sellado el plan solo tiene la rama del cambio, aunque
// axiom.yaml declare roles.
func TestStatusBeforeSealingListsOnlyTheChangeBranch(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))

	repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

	if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/x" || len(repository.Roles) != 0 {
		t.Errorf("ramas %+v, roles %v; antes del sellado solo existe feat/x", repository.Branches, repository.Roles)
	}
}

// REQ-25.3 (escenario «Un repositorio con ajustes propios prevalece sobre `git:`») y
// REQ-25.6: `fullstack` toca el repositorio del planning home y los declarados; un
// repositorio anidado toma su política propia (aquí, su rama de integración).
func TestStatusPlansEveryRepositoryWithItsOwnPolicy(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(
		map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}},
		"git:\n  integration_branch: main\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n"))
	// Los repositorios anidados se crean en su sitio, con su propia rama principal.
	for _, name := range []string{"repo-backend", "repo-frontend"} {
		dir := filepath.Join(repo, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		initRepository(t, dir)
	}
	runGit(t, filepath.Join(repo, "repo-backend"), "branch", "release")
	sealChange(t, repo, "x", "fullstack")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReached(t, result)
	if len(result.Repositories) != 3 {
		t.Fatalf("repositorios = %+v, se esperaban tres (planning home y dos anidados)", result.Repositories)
	}
	var names []string
	for _, repository := range result.Repositories {
		names = append(names, repository.Name)
	}
	if want := []string{filepath.Base(repo), "repo-backend", "repo-frontend"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("repositorios = %v, se esperaba %v (primero el del planning home, después por nombre)", names, want)
	}
	backend, frontend := result.Repositories[1], result.Repositories[2]
	if backend.IntegrationBranch != "release" || backend.Base == nil || backend.Base.Ref != "release" || backend.Branches[0].PRTarget != "release" {
		t.Errorf("repo-backend: integración %q, base %+v, PR hacia %q; se esperaba release", backend.IntegrationBranch, backend.Base, backend.Branches[0].PRTarget)
	}
	if frontend.IntegrationBranch != "main" || frontend.Base == nil || frontend.Base.Ref != "main" || frontend.Branches[0].PRTarget != "main" {
		t.Errorf("repo-frontend: integración %q, base %+v, PR hacia %q; se esperaba main", frontend.IntegrationBranch, frontend.Base, frontend.Branches[0].PRTarget)
	}
	wantSameDirectory(t, "gitRoot de repo-backend", backend.GitRoot, filepath.Join(repo, "repo-backend"))
	for _, repository := range result.Repositories {
		if !reflect.DeepEqual(repository.Roles, []string{"fullstack"}) || repository.Branches[0].Name != "feat/x" {
			t.Errorf("%s: roles %v, ramas %+v; fullstack toca todos y solo existe feat/x", repository.Name, repository.Roles, repository.Branches)
		}
	}
}

// REQ-25.6: una ruta declarada que no existe o que no es un repositorio git se omite
// sin abortar el plan, con su código y un mensaje que nombra la ruta y el rol.
func TestStatusSkipsDeclaredPathsThatAreNotGitRepositories(t *testing.T) {
	t.Parallel()
	requireGit(t)
	outside := t.TempDir()
	for current := outside; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "ausente": {"no-existe"}, "externo": {outside}}, ""))
	sealChange(t, repo, "x", "core", "ausente", "externo")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReached(t, result)
	if len(result.Skipped) != 2 {
		t.Fatalf("omitidas = %+v, se esperaban dos", result.Skipped)
	}
	byRole := map[string]branch.SkippedPath{}
	for _, skipped := range result.Skipped {
		byRole[skipped.Role] = skipped
		if skipped.Code != branch.CodeNotAGitRepository || !filepath.IsAbs(skipped.Path) {
			t.Errorf("omitida = %+v, se esperaba not_a_git_repository y una ruta absoluta", skipped)
		}
	}
	if missing := byRole["ausente"]; !strings.Contains(missing.Message, "no existe") || !strings.Contains(missing.Message, "ausente") || !strings.Contains(missing.Message, "no-existe") {
		t.Errorf("mensaje de la ruta inexistente = %q, debe nombrar la ruta, el rol y que no existe", missing.Message)
	}
	if external := byRole["externo"]; !strings.Contains(external.Message, "ningún repositorio git") || !strings.Contains(external.Message, "externo") || !strings.Contains(external.Message, outside) {
		t.Errorf("mensaje de la ruta que no es git = %q, debe nombrar la ruta, el rol y que no es un repositorio", external.Message)
	}
	if len(onlyRepository(t, result).Branches) != 1 {
		t.Errorf("el plan sigue con el repositorio del planning home: %+v", result.Repositories)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("los roles con rutas omitidas no reciben además el aviso de rol desconocido: %v", result.Warnings)
	}
}

// REQ-25.13: el carril ODD no lee el kickoff, no divide en roles y su resultado nombra
// el documento `odd/tasks/<feature>.md`.
func TestStatusOddLaneNamesTheOddDocument(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	sealChange(t, repo, "mi-feature", "core", "qa")

	result := status(t, repo, "mi-feature", branch.LaneODD)

	wantReached(t, result)
	if result.Lane != branch.LaneODD {
		t.Errorf("carril = %q, se esperaba odd", result.Lane)
	}
	repository := onlyRepository(t, result)
	if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/mi-feature" || len(repository.Roles) != 0 {
		t.Errorf("ramas %+v, roles %v; el carril ODD solo tiene la rama de la funcionalidad", repository.Branches, repository.Roles)
	}
	wantSuffix := filepath.Join("mi-feature", filepath.Base(repo), "odd", "tasks", "mi-feature.md")
	if !filepath.IsAbs(result.OddDocument) || !strings.HasSuffix(result.OddDocument, wantSuffix) {
		t.Errorf("oddDocument = %q, se esperaba una ruta absoluta que termine en %q", result.OddDocument, wantSuffix)
	}
}

// Un HEAD desacoplado se informa sin rama actual; uno que no se puede leer sin git
// deja la rama vacía y un aviso, sin fallar.
func TestStatusReportsADetachedAndAnUnreadableHead(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))

	runGit(t, repo, "checkout", "-q", "--detach")
	repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))
	if repository.CurrentBranch != "" || !repository.Detached {
		t.Errorf("rama actual %q, desacoplado %v; se esperaba sin rama y HEAD desacoplado", repository.CurrentBranch, repository.Detached)
	}
	runGit(t, repo, "checkout", "-q", "main")

	writeFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/.invalid\n")
	result := status(t, repo, "x", branch.LaneSDD)
	if repository := onlyRepository(t, result); repository.CurrentBranch != "" || repository.Detached {
		t.Errorf("rama actual %q, desacoplado %v; se esperaba sin rama y sin HEAD desacoplado", repository.CurrentBranch, repository.Detached)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "rama actual") {
		t.Errorf("avisos = %v, se esperaba uno sobre la rama actual", result.Warnings)
	}
}

// Sin la rama de integración, ni local ni como referencia remota local, `status` no
// falla (el rechazo `base_not_found` es de `ensure`): avisa y no calcula adelanto ni
// atraso.
func TestStatusWithoutTheIntegrationBranchWarnsAndSkipsTheComparison(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n"))
	runGit(t, repo, "branch", "feat/x")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	feature := branchByName(t, repository, "feat/x")
	if repository.Base != nil || feature.Ahead != nil || feature.Behind != nil || feature.ComparedTo != "" || !feature.Exists {
		t.Errorf("base %+v, rama %+v; sin rama de integración no hay base ni comparación", repository.Base, feature)
	}
	// Dos avisos con la misma causa y distinto efecto: no se calculan adelanto ni atraso, y
	// la lista de pendientes de archivar no se evalúa (REQ-25.20).
	if len(result.Warnings) != 2 || !strings.Contains(result.Warnings[0], "«develop»") || !strings.Contains(result.Warnings[0], "adelanto") ||
		!strings.Contains(result.Warnings[1], "«develop»") || !strings.Contains(result.Warnings[1], "pendientes de archivar") {
		t.Errorf("avisos = %v, se esperaban dos que nombren la rama de integración develop: el de la comparación y el de los pendientes", result.Warnings)
	}
}

// Un worktree que git aún lista pero cuya carpeta ya no existe no hace fallar
// `status`: se informa con un aviso y la rama figura sin worktree.
func TestStatusToleratesAWorktreeWhoseDirectoryIsGone(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	worktree := filepath.Join(t.TempDir(), "x", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree, "main")
	if err := os.RemoveAll(worktree); err != nil {
		t.Fatal(err)
	}

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	feature := branchByName(t, onlyRepository(t, result), "feat/x")
	if !feature.Exists || feature.Worktree != "" || feature.Dirty {
		t.Errorf("rama = %+v, se esperaba existente y sin worktree", feature)
	}
	// La orden cita la raíz git tal como la devuelve git (la grafía larga de Windows).
	wantCommand := "git -C " + pathquote.Quote(onlyRepository(t, result).GitRoot) + " worktree prune"
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], wantCommand) {
		t.Errorf("avisos = %v, se esperaba uno con la orden `%s`", result.Warnings, wantCommand)
	}
}

// Los rechazos del motor: cada uno con su código tipado, su estado y una continuación
// que es una orden ejecutable con las rutas citadas.
func TestStatusRefusals(t *testing.T) {
	t.Run("fuera de un repositorio git", func(t *testing.T) {
		requireGit(t)
		outside := t.TempDir()
		for current := outside; ; {
			if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
				t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
		result := status(t, outside, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodeNotAGitRepository, branch.StatusRefused)
		for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(outside)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
	t.Run("un directorio que no existe", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no", "existe")
		result := status(t, missing, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodeNotAGitRepository, branch.StatusRefused)
		if !strings.Contains(result.Message, "no existe") || !strings.Contains(result.Message, missing) {
			t.Errorf("el mensaje %q debe decir que %s no existe", result.Message, missing)
		}
	})
	t.Run("un nombre de cambio inválido, antes de lanzar git ni leer el disco", func(t *testing.T) {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
		seen := observe(t)
		// El nombre vacío ya no es inválido: sin cambio, `status` lista (pending_test.go).
		for _, invalid := range []string{"a b:c", "../fuera", "CON", "   "} {
			result := status(t, repo, invalid, branch.LaneSDD)

			wantRefusal(t, result, branch.CodeInvalidName, branch.StatusRefused)
			if result.Details["field"] != "--change" || result.Details["value"] != invalid {
				t.Errorf("details = %v, se esperaba el campo --change y el valor %q", result.Details, invalid)
			}
			for _, want := range []string{"axiom branch status --change <cambio> --cwd", pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
		}
		if len(*seen) != 0 {
			t.Errorf("un nombre inválido se rechaza sin lanzar git, pero el observador vio %v", *seen)
		}
	})
	t.Run("una configuración inválida", func(t *testing.T) {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  branch_prefix: \"mi prefijo//\"\n"))
		result := status(t, repo, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
		if !strings.Contains(result.Message, "branch_prefix") {
			t.Errorf("el mensaje %q debe nombrar la causa", result.Message)
		}
		for _, want := range []string{"axiom workspace validate --path", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
	t.Run("un kickoff corrupto no se confunde con un cambio sin sellar", func(t *testing.T) {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
		kickoffPath := filepath.Join(repo, "openspec", "changes", "x", "kickoff.yaml")
		writeFile(t, kickoffPath, "esto: [no es un kickoff\n")
		result := status(t, repo, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
		if result.Details["path"] != kickoffPath || !strings.Contains(result.Message, "kickoff") {
			t.Errorf("details %v, mensaje %q; se esperaba nombrar %s", result.Details, result.Message, kickoffPath)
		}
		for _, want := range []string{"axiom sdd status x --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
	t.Run("git no se puede ejecutar", func(t *testing.T) {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
		t.Setenv("PATH", t.TempDir())
		result := status(t, repo, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
		if result.Details["operation"] == "" || !strings.Contains(result.Message, result.Details["operation"]) || result.Details["summary"] == "" {
			t.Errorf("git_failed debe nombrar la operación y resumir el error: mensaje %q, details %v", result.Message, result.Details)
		}
		for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
}

// relativeWithin asciende por identidad de ruta: git devuelve la grafía larga de
// Windows y la carpeta maestra puede llevar el nombre corto 8.3 (tasks.md, regla 10).
func TestRelativeWithin(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nested := filepath.Join(root, "app", "web")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()

	tests := []struct {
		name                 string
		ancestor, descendant string
		want                 string
		wantInside           bool
	}{
		{"el propio directorio", root, root, ".", true},
		{"un descendiente", root, nested, "app/web", true},
		{"un ancestro no está dentro", nested, root, "", false},
		{"otro árbol", root, outside, "", false},
		{"con barras finales y rutas sin limpiar", root + string(filepath.Separator), filepath.Join(nested, "..", "web"), "app/web", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, inside := relativeWithin(tc.ancestor, tc.descendant)
			if got != tc.want || inside != tc.wantInside {
				t.Errorf("relativeWithin(%q, %q) = %q, %v; se esperaba %q, %v", tc.ancestor, tc.descendant, got, inside, tc.want, tc.wantInside)
			}
		})
	}
}
