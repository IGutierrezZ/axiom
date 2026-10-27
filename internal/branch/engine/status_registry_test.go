package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `status` con git real sobre el registro local de solo lectura (REQ-25.14,
// REQ-25.15), los multirepos (REQ-25.5, REQ-25.6) y la colocación de los worktrees
// (REQ-25.11): lo que el planificador puro decide con tablas, aquí se comprueba con los
// hechos que reúne el motor.

// canonicalDirectory devuelve dir con la grafía que usa git (en Windows, los temporales
// salen con nombres cortos 8.3): las rutas que el motor compara con las de git salen de
// ella.
func canonicalDirectory(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// plannedWorktree es la ruta del worktree del cambio x que planifica el motor para un
// repositorio: `<raíz de worktrees por defecto>/<cambio>/<repo>`.
func plannedWorktree(t *testing.T, master, change, repo string) string {
	t.Helper()
	return filepath.Join(branch.DefaultWorktreeRoot(canonicalDirectory(t, master)), change, repo)
}

// commonDir es el git common dir de un repositorio normal: su carpeta `.git`.
func commonDir(repo string) string { return filepath.Join(repo, ".git") }

// recordFile es la ruta del registro de un cambio, escrita como la fija el diseño (§4.4).
func recordFile(commonDir, lane, change string) string {
	return filepath.Join(commonDir, "axiom", "branch", "v1", lane, change+".json")
}

// registeredRecord es el registro de un cambio materializado en modo worktree desde el
// hub, con un repositorio y las ramas dadas, con rutas absolutas.
func registeredRecord(repo string, branches ...branch.RecordBranch) branch.Record {
	return branch.Record{
		Schema:       branch.RegistrySchema,
		Lane:         branch.LaneSDD,
		Change:       "x",
		Mode:         branch.ModeWorktree,
		ModeSource:   branch.ModeSourceHub,
		PlanningRepo: repo,
		Repositories: []branch.RecordRepo{{
			Name:      filepath.Base(repo),
			GitRoot:   repo,
			CommonDir: commonDir(repo),
			Role:      branch.RepoChangeBranch,
			Branches:  branches,
		}},
	}
}

func writeRegistry(t *testing.T, commonDir string, record branch.Record) string {
	t.Helper()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := recordFile(commonDir, string(record.Lane), record.Change)
	writeFile(t, path, string(data)+"\n")
	return path
}

func wantSameFile(t *testing.T, what, got, want string) {
	t.Helper()
	gotInfo, gotErr := os.Stat(got)
	wantInfo, wantErr := os.Stat(want)
	if gotErr != nil || wantErr != nil || !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("%s = %q, se esperaba el fichero %q (%v, %v)", what, got, want, gotErr, wantErr)
	}
}

// newEngine monta un motor con las reglas de rutas de un sistema operativo dado.
func newEngine(goos string) *Engine {
	return &Engine{git: gitcli.New(), writer: gitcli.New(), registry: registry.New(), goos: goos}
}

// REQ-25.14 (escenario «Registro ausente informado por `status`»): un cambio sin
// registro no falla, `registry.present` es falso, el modo no se conoce y leerlo no crea
// nada en el repositorio.
func TestStatusReportsAnAbsentRegistryWithoutCreatingAnything(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))

	result := status(t, repo, "x", branch.LaneSDD)

	wantReached(t, result)
	if result.Registry == nil || result.Registry.Present {
		t.Errorf("registry = %+v, se esperaba present=false", result.Registry)
	}
	if result.Mode != "" || result.ModeLabel != "" || result.ModeSource != "" || result.ModeSourceLabel != "" {
		t.Errorf("sin registro no se conoce el modo: %q %q %q %q", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
	}
	if _, err := os.Stat(filepath.Join(commonDir(repo), "axiom")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("`status` es de solo lectura y no debe crear el registro (%v)", err)
	}
}

// REQ-25.14 y REQ-25.15: con registro, `status` informa del modo materializado y su
// origen, y del worktree que el registro guarda para cada rama, aparte del que git conoce.
func TestStatusReadsTheModeAndTheRegisteredWorktreesFromTheRegistry(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree, "main")

	t.Run("el registro coincide con git", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, Worktree: worktree}))

		result := status(t, repo, "x", branch.LaneSDD)

		wantReady(t, result)
		if result.Registry == nil || !result.Registry.Present {
			t.Fatalf("registry = %+v, se esperaba present=true", result.Registry)
		}
		if result.Mode != branch.ModeWorktree || result.ModeLabel != "un worktree por rama" || result.ModeSource != branch.ModeSourceHub || result.ModeSourceLabel != "preferencias personales del hub" {
			t.Errorf("modo %q / %q, origen %q / %q", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
		}
		feature := branchByName(t, onlyRepository(t, result), "feat/x")
		if !feature.Exists || feature.Worktree != worktree || feature.RegisteredWorktree != worktree {
			t.Errorf("rama = %+v, se esperaba el worktree %s tanto el de git como el registrado", feature, worktree)
		}
	})
	t.Run("la ruta registrada se limpia como las de git", func(t *testing.T) {
		unclean := worktree + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(worktree)
		writeRegistry(t, commonDir(repo), registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, Worktree: unclean}))

		feature := branchByName(t, onlyRepository(t, status(t, repo, "x", branch.LaneSDD)), "feat/x")
		if feature.RegisteredWorktree != worktree || feature.Worktree != worktree {
			t.Errorf("rama = %+v, se esperaba la ruta limpia %s en ambos campos", feature, worktree)
		}
	})
	t.Run("el registro guarda otro worktree que el de git", func(t *testing.T) {
		stale := filepath.Join(t.TempDir(), "antiguo")
		writeRegistry(t, commonDir(repo), registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, Worktree: stale}))

		feature := branchByName(t, onlyRepository(t, status(t, repo, "x", branch.LaneSDD)), "feat/x")
		if feature.Worktree != worktree || feature.RegisteredWorktree != stale {
			t.Errorf("rama = %+v, se esperaba el de git %s y el registrado %s", feature, worktree, stale)
		}
	})
	t.Run("el modo in_place registrado se informa tal cual", func(t *testing.T) {
		record := registeredRecord(repo)
		record.Mode, record.ModeSource = branch.ModeInPlace, branch.ModeSourceAnswer
		writeRegistry(t, commonDir(repo), record)

		result := status(t, repo, "x", branch.LaneSDD)
		if result.Mode != branch.ModeInPlace || result.ModeLabel != "en la carpeta del proyecto" || result.ModeSourceLabel != "respuesta registrada" {
			t.Errorf("modo %q / %q, origen %q / %q", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
		}
		if feature := branchByName(t, onlyRepository(t, result), "feat/x"); feature.RegisteredWorktree != "" {
			t.Errorf("sin worktree en el registro la rama no lleva worktree registrado: %+v", feature)
		}
	})
	t.Run("el registro de otro carril o de otro cambio no cuenta", func(t *testing.T) {
		other := registeredRecord(repo)
		other.Change = "otro"
		writeRegistry(t, commonDir(repo), other)

		if result := status(t, repo, "y", branch.LaneSDD); result.Registry == nil || result.Registry.Present {
			t.Errorf("registry = %+v, el cambio y no tiene registro", result.Registry)
		}
	})
}

// REQ-25.14, escenario «El registro es visible desde un worktree enlazado»: `status`
// ejecutado desde un worktree enlazado resuelve el registro con el git common dir, que es
// el del checkout principal.
func TestStatusSeesTheRegistryFromALinkedWorktree(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, repo, "worktree", "add", "-q", "-b", "inc/otra", linked, "main")
	writeRegistry(t, commonDir(repo), registeredRecord(repo))

	result := status(t, linked, "x", branch.LaneSDD)

	wantReached(t, result)
	if result.Registry == nil || !result.Registry.Present || result.Mode != branch.ModeWorktree {
		t.Errorf("registry %+v, modo %q; el registro del checkout principal debe verse desde el worktree enlazado", result.Registry, result.Mode)
	}
	if repository := onlyRepository(t, result); !pathidentity.SameDirectory(repository.CommonDir, commonDir(repo)) {
		t.Errorf("commonDir = %q, se esperaba el del checkout principal %q", repository.CommonDir, commonDir(repo))
	}
}

// REQ-25.12 y REQ-25.14: las excepciones que el registro guarda (`--prefix`, `--branch`,
// `--base`) fijan los nombres de rama y la base con los que `status` mide el cambio, de
// modo que describe lo que `ensure` materializó.
func TestStatusAppliesTheExceptionsOfTheRegistry(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	runGit(t, repo, "branch", "release/2", "main~1")
	runGit(t, repo, "branch", "inc/x", "main")
	runGit(t, repo, "branch", "fix/x", "main")
	releaseSHA := runGit(t, repo, "rev-parse", "release/2")

	withExceptions := func(exceptions branch.Exceptions) branch.Record {
		record := registeredRecord(repo)
		record.Exceptions = exceptions
		return record
	}

	t.Run("--branch fija el nombre de la rama del cambio", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), withExceptions(branch.Exceptions{Branch: "inc/x"}))

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if len(repository.Branches) != 1 || repository.Branches[0].Name != "inc/x" || !repository.Branches[0].Exists {
			t.Errorf("ramas = %+v, se esperaba inc/x existente y no feat/x", repository.Branches)
		}
	})
	t.Run("--prefix fija el prefijo", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), withExceptions(branch.Exceptions{Prefix: "fix"}))

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if len(repository.Branches) != 1 || repository.Branches[0].Name != "fix/x" || !repository.Branches[0].Exists {
			t.Errorf("ramas = %+v, se esperaba fix/x existente", repository.Branches)
		}
	})
	t.Run("--base fija la base con la que se mide el adelanto", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), withExceptions(branch.Exceptions{Branch: "inc/x", Base: "release/2"}))

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if repository.Base == nil || repository.Base.Ref != "release/2" || repository.Base.Source != "local" || repository.Base.SHA != releaseSHA || repository.Base.Fetched {
			t.Fatalf("base = %+v, se esperaba release/2 local con su commit y sin fetch", repository.Base)
		}
		inc := branchByName(t, repository, "inc/x")
		if inc.ComparedTo != "release/2" || intValue(t, "adelanto", inc.Ahead) != 1 || intValue(t, "atraso", inc.Behind) != 0 {
			t.Errorf("rama = %+v, se esperaba medida contra release/2: adelanto 1 y atraso 0", inc)
		}
	})
	t.Run("--base con una referencia remota ya traída a local", func(t *testing.T) {
		runGit(t, repo, "update-ref", "refs/remotes/origin/release", releaseSHA)
		writeRegistry(t, commonDir(repo), withExceptions(branch.Exceptions{Branch: "inc/x", Base: "origin/release"}))

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if repository.Base == nil || repository.Base.Ref != "origin/release" || repository.Base.Source != "remote-tracking" || repository.Base.SHA != releaseSHA {
			t.Errorf("base = %+v, se esperaba origin/release remote-tracking", repository.Base)
		}
	})
	t.Run("--base que no existe avisa y no mide", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), withExceptions(branch.Exceptions{Branch: "inc/x", Base: "no/existe"}))

		result := status(t, repo, "x", branch.LaneSDD)

		wantReady(t, result)
		repository := onlyRepository(t, result)
		if repository.Base != nil || repository.Branches[0].Ahead != nil {
			t.Errorf("base %+v, rama %+v; sin la base fijada no se mide", repository.Base, repository.Branches[0])
		}
		if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "«no/existe»") {
			t.Errorf("avisos = %v, se esperaba uno que nombre la base no/existe", result.Warnings)
		}
	})
}

// REQ-25.14 y design D-07: un fichero corrupto es `config_invalid` con el fichero y la
// continuación de borrarlo y reconstruirlo con `ensure`.
func TestStatusRefusesACorruptRegistryNamingTheFile(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	path := recordFile(commonDir(repo), "sdd", "x")
	writeFile(t, path, "esto no es JSON {")

	result := status(t, repo, "x", branch.LaneSDD)

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	wantSameFile(t, "details.path", result.Details["path"], path)
	if !strings.Contains(result.Message, "registro local") || !strings.Contains(result.Message, "JSON") {
		t.Errorf("el mensaje %q debe nombrar el registro y la causa", result.Message)
	}
	for _, want := range []string{"borra", pathquote.Quote(result.Details["path"]), "axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	if result.Registry != nil {
		t.Errorf("un rechazo no lleva datos del cambio: %+v", result.Registry)
	}
}

// Un registro que existe pero no se puede leer (aquí, un directorio en el lugar del
// fichero) también es `config_invalid` y nombra el fichero.
func TestStatusRefusesAnUnreadableRegistryAsConfigInvalid(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	path := recordFile(commonDir(repo), "sdd", "x")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	result := status(t, repo, "x", branch.LaneSDD)

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	wantSameFile(t, "details.path", result.Details["path"], path)
}

// El carril lo valida la CLI; si llegara uno que no puede nombrar un registro, el motor
// no lo lee ni sale de la carpeta del registro: `config_invalid` sin fichero que nombrar.
func TestStatusRefusesALaneThatCannotNameARegistry(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))

	result := status(t, repo, "x", branch.Lane("../v1"))

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	if !strings.Contains(result.Message, "carril") || len(result.Details) != 0 {
		t.Errorf("mensaje %q, details %v; se esperaba nombrar el carril y ningún fichero", result.Message, result.Details)
	}
	for _, want := range []string{"axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// La continuación que reconstruye el registro cita el nombre del cambio como una palabra del
// intérprete: un nombre con espacios no se parte en dos argumentos.
func TestRegistryRefusalContinuationQuotesAChangeNameWithSpaces(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()

	refusal := registryRefusal(errors.New("el registro no se puede leer"), cwd, "phase one")

	for _, want := range []string{"axiom branch ensure --change 'phase one' --cwd", pathquote.Quote(cwd)} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	}
	if plain := registryRefusal(errors.New("x"), cwd, "x"); !strings.Contains(plain.Continuation, "--change x --cwd") {
		t.Errorf("un nombre seguro va sin comillas: %q", plain.Continuation)
	}
}

// newMultirepo crea un multirepo: una carpeta maestra con su axiom.yaml y, dentro, el
// repositorio de specs `repo-specs` y los repositorios de código dados, cada uno un
// repositorio real con su commit en main.
func newMultirepo(t *testing.T, roles map[string][]string, codeRepositories ...string) (master, specs string) {
	t.Helper()
	requireGit(t)
	master = t.TempDir()
	specs = filepath.Join(master, "repo-specs")
	for _, dir := range append([]string{"repo-specs"}, codeRepositories...) {
		if err := os.MkdirAll(filepath.Join(master, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		initRepository(t, filepath.Join(master, dir))
	}
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", roles, ""))
	return master, specs
}

// REQ-25.5 y REQ-25.4 (multirepo): el repositorio de specs se trabaja in situ, sin ramas
// ni worktrees ni checkout; cada repositorio de código recibe la misma rama del cambio
// desde su propia integración.
func TestStatusMultirepoWorksTheSpecsRepositoryInPlace(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	sealChangeAt(t, specs, "x", "backend", "frontend")

	t.Run("sin ramas creadas el cambio no está iniciado", func(t *testing.T) {
		result := status(t, master, "x", branch.LaneSDD)

		if result.Status != branch.StatusNotStarted || result.ExitCode() != branch.ExitReady {
			t.Fatalf("status %q, salida %d; se esperaba not_started con salida 0", result.Status, result.ExitCode())
		}
		var names, roles []string
		for _, repository := range result.Repositories {
			names, roles = append(names, repository.Name), append(roles, string(repository.Role))
		}
		if want := []string{"repo-specs", "repo-backend", "repo-frontend"}; !reflect.DeepEqual(names, want) {
			t.Fatalf("repositorios = %v, se esperaba %v (primero el de specs)", names, want)
		}
		if want := []string{"specs_in_place", "code", "code"}; !reflect.DeepEqual(roles, want) {
			t.Errorf("papeles = %v, se esperaba %v", roles, want)
		}
		specsResult := result.Repositories[0]
		if specsResult.RoleLabel != "repositorio de specs in situ" || len(specsResult.Branches) != 0 || specsResult.CurrentBranch != "main" {
			t.Errorf("repositorio de specs = %+v, se esperaba in situ, sin ramas y en main", specsResult)
		}
		wantSameDirectory(t, "gitRoot del repositorio de specs", specsResult.GitRoot, specs)
		wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, specs)
		wantSameDirectory(t, "planningHome", filepath.Dir(result.PlanningHome), specs)
		for _, repository := range result.Repositories[1:] {
			if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/x" || repository.Branches[0].PRTarget != "main" || repository.Branches[0].Exists {
				t.Errorf("%s: ramas %+v, se esperaba feat/x hacia main sin crear", repository.Name, repository.Branches)
			}
		}
	})
	t.Run("con el cambio iniciado en un repositorio de código", func(t *testing.T) {
		backend := filepath.Join(master, "repo-backend")
		worktree := plannedWorktree(t, master, "x", "repo-backend")
		runGit(t, backend, "worktree", "add", "-q", "-b", "feat/x", worktree, "main")

		result := status(t, master, "x", branch.LaneSDD)

		wantReady(t, result)
		created := branchByName(t, result.Repositories[1], "feat/x")
		if !created.Exists || !filepath.IsAbs(created.Worktree) || !pathidentity.SameDirectory(created.Worktree, worktree) {
			t.Errorf("rama de repo-backend = %+v, se esperaba existente con su worktree %s", created, worktree)
		}
		if pending := branchByName(t, result.Repositories[2], "feat/x"); pending.Exists {
			t.Errorf("repo-frontend no se ha tocado: %+v", pending)
		}
		if len(result.Repositories[0].Branches) != 0 {
			t.Errorf("el repositorio de specs sigue sin ramas: %+v", result.Repositories[0])
		}
	})
}

// REQ-25.5 y design D-04: los perfiles `knowledge` y `spec-only` de `axiom init` generan
// un multirepo con el rol `knowledge` sobre `.`, el propio repositorio de specs: no hay
// ramas ni worktrees que crear.
func TestStatusOfTheKnowledgeAndSpecOnlyProfilesHasNoBranches(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configFor("multirepo", "openspec", map[string][]string{"knowledge": {"."}}, ""))
	sealChangeAt(t, repo, "x", "knowledge")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if repository.Role != branch.RepoSpecsInPlace || len(repository.Branches) != 0 || !reflect.DeepEqual(repository.Roles, []string{"knowledge"}) {
		t.Errorf("repositorio = %+v, se esperaba specs_in_place con el rol knowledge y sin ramas", repository)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno", result.Warnings)
	}
	wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, repo)
}

// REQ-25.6 (escenarios «Una subcarpeta resuelve a la raíz git del repositorio que la
// contiene» y «Dos roles sobre rutas del mismo repositorio se deduplican»): las rutas
// de los roles se resuelven con las consultas de git; el plan tiene un único repositorio
// por raíz y los dos roles cuentan como roles del cambio sobre él.
func TestStatusResolvesDeclaredSubfoldersToTheirGitRootAndDeduplicates(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"api": {"services/api"}, "web": {"services/web"}}, ""))
	for _, dir := range []string{"services/api", "services/web"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sealChange(t, repo, "x", "api", "web")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReached(t, result)
	repository := onlyRepository(t, result)
	wantSameDirectory(t, "gitRoot", repository.GitRoot, repo)
	if !reflect.DeepEqual(repository.Roles, []string{"api", "web"}) {
		t.Errorf("roles = %v, se esperaban api y web sobre el mismo repositorio", repository.Roles)
	}
	var names []string
	for _, planned := range repository.Branches {
		names = append(names, planned.Name)
	}
	if want := []string{"feat/x", "feat/x-api", "feat/x-web"}; !reflect.DeepEqual(names, want) {
		t.Errorf("ramas = %v, se esperaba %v (seguimiento y una rama por rol)", names, want)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("las subcarpetas del repositorio no se omiten: %+v", result.Skipped)
	}
}

// REQ-25.4 y REQ-25.6: un repositorio de código compartido por dos roles se planifica
// una sola vez, con una rama de rol por cada uno.
func TestStatusAnEmbeddedRepositorySharedByTwoRolesIsPlannedOnce(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"api": {"repo-backend"}, "jobs": {"repo-backend"}}, ""))
	dir := filepath.Join(repo, "repo-backend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepository(t, dir)
	sealChange(t, repo, "x", "api", "jobs")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReached(t, result)
	if len(result.Repositories) != 2 {
		t.Fatalf("repositorios = %d, se esperaban el planning home y repo-backend una sola vez", len(result.Repositories))
	}
	backend := result.Repositories[1]
	var names []string
	for _, planned := range backend.Branches {
		names = append(names, planned.Name)
	}
	if want := []string{"feat/x", "feat/x-api", "feat/x-jobs"}; backend.Name != "repo-backend" || !reflect.DeepEqual(names, want) || !reflect.DeepEqual(backend.Roles, []string{"api", "jobs"}) {
		t.Errorf("repo-backend: nombre %q, ramas %v, roles %v; se esperaba una sola vez con %v", backend.Name, names, backend.Roles, want)
	}
}

// REQ-25.11 y design D-04: `plan_conflict` cuando la raíz de worktrees (la carpeta
// hermana `<carpeta maestra>-wt`) queda dentro del repositorio del plan, como ocurre
// cuando la carpeta maestra es una subcarpeta del repositorio.
func TestStatusRefusesAWorktreeRootInsideTheRepository(t *testing.T) {
	t.Parallel()
	repo := newRepository(t)
	master := filepath.Join(repo, "app")
	writeFile(t, filepath.Join(master, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, ""))

	result := status(t, master, "x", branch.LaneSDD)

	wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
	wantRoot := branch.DefaultWorktreeRoot(canonicalDirectory(t, master))
	if result.Details["worktreeRoot"] != wantRoot {
		t.Errorf("details = %v, se esperaba la raíz de worktrees %s", result.Details, wantRoot)
	}
	wantSameDirectory(t, "details.repository", result.Details["repository"], repo)
	for _, want := range []string{"axiom branch status --change x --cwd", pathquote.Quote(master)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// REQ-25.11 y design D-04: `plan_conflict` cuando una ruta planificada queda dentro de
// un worktree existente distinto del suyo, por ejemplo uno antiguo de los scripts
// locales en `<raíz>/<cambio>`; los worktrees de otros cambios no estorban.
func TestStatusRefusesAPlannedPathInsideAnOldWorktree(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	root := branch.DefaultWorktreeRoot(canonicalDirectory(t, repo))

	t.Run("un worktree de otro cambio no estorba", func(t *testing.T) {
		runGit(t, repo, "worktree", "add", "-q", "-b", "antiguo-otro", filepath.Join(root, "otro"), "main")
		wantReached(t, status(t, repo, "x", branch.LaneSDD))
	})
	t.Run("un worktree antiguo en la ruta del cambio contiene la ruta planificada", func(t *testing.T) {
		old := filepath.Join(root, "x")
		runGit(t, repo, "worktree", "add", "-q", "-b", "antiguo-x", old, "main")

		result := status(t, repo, "x", branch.LaneSDD)

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		wantSameDirectory(t, "details.existing", result.Details["existing"], old)
		if want := filepath.Join(old, filepath.Base(repo)); result.Details["worktree"] != want {
			t.Errorf("details = %v, se esperaba la ruta planificada %s", result.Details, want)
		}
		for _, want := range []string{"worktree remove", pathquote.Quote(result.Details["existing"]), "axiom branch status --change <cambio> --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
}

// REQ-25.11 y design D-04 (A-1): el motor entrega al planificador las reglas de rutas de
// su sistema operativo. Con las de Windows, una ruta de worktree de más de 220 caracteres
// da `path_too_long`, sea cual sea el sistema en que corra la prueba; con las de otro
// sistema no hay límite propio. El carril ODD no lee el kickoff del cambio, de modo que la
// ruta del kickoff, más larga que la del worktree, no interviene.
func TestStatusAppliesTheWindowsPathLimitOfItsOperatingSystem(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	change := strings.Repeat("c", 200)

	t.Run("con las reglas de Windows", func(t *testing.T) {
		result := newEngine("windows").Status(context.Background(), repo, change, branch.LaneODD)

		wantRefusal(t, result, branch.CodePathTooLong, branch.StatusRefused)
		if result.Details["max"] != "220" || !strings.Contains(result.Message, "220") || !strings.Contains(result.Message, change) {
			t.Errorf("details %v, mensaje %q; se esperaba la ruta, su longitud y el máximo 220", result.Details, result.Message)
		}
		for _, want := range []string{"axiom branch status --change <cambio> --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
	t.Run("con las reglas de otro sistema no hay límite propio", func(t *testing.T) {
		wantReached(t, newEngine("linux").Status(context.Background(), repo, change, branch.LaneODD))
	})
	t.Run("Default usa el sistema operativo en el que corre", func(t *testing.T) {
		if got := Default().goos; got == "" {
			t.Error("Default debe fijar las reglas de rutas del sistema operativo en el que corre")
		}
	})
}
