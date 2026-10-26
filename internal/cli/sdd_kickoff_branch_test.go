package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de `axiom sdd kickoff seal` con la guarda A (REQ-25.38, REQ-25.37 escenario 2, design
// D-11 y D-12): sin `--infer` consulta el veredicto de rama del repositorio del planning home
// antes de escribir y se niega con `wrong_branch` y la continuación de `ensure`. Con git real,
// entorno aislado y sin red; reutilizan los ayudantes de sdd_integrate_test.go.

const sealChange = "x"

var sealBaseArgs = []string{"--execution-style", "continuous", "--handoff-policy", "none"}

// captureSealWarnings recoge los avisos de la guarda para que no ensucien la salida de las
// pruebas y se puedan comprobar.
func captureSealWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var warnings bytes.Buffer
	previous := sealBranchWarnings
	sealBranchWarnings = &warnings
	t.Cleanup(func() { sealBranchWarnings = previous })
	return &warnings
}

// embeddedSealProject es un monorepo embebido con git, en main y con la carpeta del cambio `x`
// confirmada: la forma de un proyecto en el que aún no se ha sellado nada.
func embeddedSealProject(t *testing.T, config string) workspacetest.Workspace {
	t.Helper()
	isolateIntegrateGit(t)
	ws := workspacetest.Embedded(t, ".")
	if config != "" {
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), config)
	}
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", sealChange, "proposal.md"), "# x\n")
	integrateInitRepo(t, ws.Master)
	integrateCommit(t, ws.Master, "base")
	return ws
}

func runSeal(t *testing.T, cwd string, extra ...string) (string, error) {
	t.Helper()
	args := append([]string{"seal", "--cwd", cwd, "--change", sealChange}, extra...)
	var out bytes.Buffer
	err := RunSDDKickoff(args, &out)
	return out.String(), err
}

func wantKickoffAbsent(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(ws.Home, "changes", sealChange, "kickoff.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("el rechazo no debe escribir kickoff.yaml (%v)", err)
	}
}

func wantKickoffWritten(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(ws.Home, "changes", sealChange, "kickoff.yaml")); err != nil {
		t.Errorf("el sellado debe escribir kickoff.yaml: %v", err)
	}
}

// wantWrongBranch comprueba el rechazo tipado de la guarda: el código, la rama actual y la
// esperada, y la orden exacta de `ensure` con `--cwd` apuntando a master.
func wantWrongBranch(t *testing.T, err error, current, expected, master string) {
	t.Helper()
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("error = %v, se esperaba el rechazo tipado wrong_branch", err)
	}
	for _, want := range []string{"wrong_branch", "«" + current + "»", "«" + expected + "»", "axiom branch ensure --cwd ", `" --change ` + sealChange} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe contener %q", err.Error(), want)
		}
	}
	match := regexp.MustCompile(`--cwd "([^"]*)"`).FindStringSubmatch(err.Error())
	if match == nil || !pathidentity.SameDirectory(match[1], master) {
		t.Errorf("la continuación de %q debe citar --cwd con la carpeta maestra %q", err.Error(), master)
	}
}

// REQ-25.38, escenario «Sellar en la rama de integración de un monorepo se rechaza»: rama
// actual `main`, esperada `feat/x`, la continuación exacta de `ensure`, sin kickoff.yaml y
// con error (código de salida distinto de cero).
func TestKickoffSealRefusesTheIntegrationBranchOfAMonorepo(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, "")

	out, err := runSeal(t, ws.Master, sealBaseArgs...)

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	wantKickoffAbsent(t, ws)
	if out != "" {
		t.Errorf("un rechazo no imprime el resumen del sellado: %q", out)
	}
}

// REQ-25.38, escenario «Sellar en la rama del cambio procede»; y la política: el prefijo
// configurado cambia la rama esperada.
func TestKickoffSealProceedsOnTheChangeBranch(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, "")
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")

	out, err := runSeal(t, ws.Master, sealBaseArgs...)

	if err != nil {
		t.Fatalf("sellar en feat/x: %v", err)
	}
	if !strings.Contains(out, "Kickoff sellado") {
		t.Errorf("la salida no resume el sellado:\n%s", out)
	}
	wantKickoffWritten(t, ws)
}

func TestKickoffSealTakesTheExpectedBranchFromThePolicyPrefix(t *testing.T) {
	captureSealWarnings(t)
	config := "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: \".\"\ngit:\n  integration_branch: main\n  branch_prefix: inc\nroles:\n  core:\n    name: core\n    repositories:\n      - path: \".\"\n"
	ws := embeddedSealProject(t, config)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")

	_, err := runSeal(t, ws.Master, sealBaseArgs...)
	wantWrongBranch(t, err, "feat/x", "inc/x", ws.Master)
	wantKickoffAbsent(t, ws)

	integrateGit(t, ws.Master, "switch", "-q", "-c", "inc/x")
	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("sellar en inc/x: %v", err)
	}
	wantKickoffWritten(t, ws)
}

// REQ-25.38 y REQ-25.18: con los roles que se van a sellar, una rama de rol del cambio es
// correcta; una rama ajena no.
func TestKickoffSealAcceptsTheRoleBranchesOfTheRolesBeingSealed(t *testing.T) {
	captureSealWarnings(t)
	config := "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: \".\"\nroles:\n  core:\n    name: core\n    repositories:\n      - path: \".\"\n  qa:\n    name: qa\n    repositories:\n      - path: \".\"\n"
	ws := embeddedSealProject(t, config)
	roles := append([]string{"--role", "core", "--role", "qa"}, sealBaseArgs...)

	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x-otro")
	_, err := runSeal(t, ws.Master, roles...)
	wantWrongBranch(t, err, "feat/x-otro", "feat/x", ws.Master)
	wantKickoffAbsent(t, ws)

	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x-qa")
	if _, err := runSeal(t, ws.Master, roles...); err != nil {
		t.Fatalf("sellar en la rama de rol feat/x-qa: %v", err)
	}
	wantKickoffWritten(t, ws)
}

// REQ-25.37, escenario «`seal --infer` funciona desde cualquier rama y no escribe
// instantánea»: desde `main` se completa sin exigir rama (y sin consultar el veredicto).
func TestKickoffSealInferWorksFromAnyBranch(t *testing.T) {
	warnings := captureSealWarnings(t)
	ws := embeddedSealProject(t, "")

	out, err := runSeal(t, ws.Master, "--infer")

	if err != nil {
		t.Fatalf("seal --infer desde main: %v", err)
	}
	if !strings.Contains(out, "fullstack") {
		t.Errorf("la salida no refleja el retro-sellado:\n%s", out)
	}
	wantKickoffWritten(t, ws)
	if warnings.Len() != 0 {
		t.Errorf("--infer no consulta la rama y no avisa: %q", warnings.String())
	}
}

// REQ-25.37: un cambio cuyo kickoff.yaml ya existe sin instantánea queda exento: volver a
// sellarlo desde `main` no se rechaza y devuelve al ganador sin reescribirlo.
func TestKickoffSealOfAnAlreadySealedChangeWithoutSnapshotIsExempt(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, "")
	// El retro-sellado no escribe instantánea: el cambio queda exento (R15).
	if _, err := runSeal(t, ws.Master, "--infer"); err != nil {
		t.Fatalf("primer sellado: %v", err)
	}
	kickoffPath := filepath.Join(ws.Home, "changes", sealChange, "kickoff.yaml")
	before, err := os.ReadFile(kickoffPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runSeal(t, ws.Master, "--execution-style", "checkpointed", "--handoff-policy", "none")

	if err != nil {
		t.Fatalf("un cambio sellado sin instantánea queda exento: %v", err)
	}
	if after, _ := os.ReadFile(kickoffPath); !bytes.Equal(before, after) {
		t.Errorf("el segundo sellado reescribió kickoff.yaml:\n%s\n---\n%s", before, after)
	}
	if !strings.Contains(out, "Kickoff ya estaba sellado") {
		t.Errorf("la salida debe indicar que ya estaba sellado:\n%s", out)
	}
	if loaded, err := kickoff.Load(filepath.Join(ws.Home, "changes", sealChange)); err != nil || loaded == nil || loaded.Git != nil {
		t.Errorf("el cambio retro-sellado sigue sin instantánea: %+v, %v", loaded, err)
	}
}

// twoRoleConfig declara los roles core y qa sobre el repositorio del proyecto, con la política
// git `main` y `feat`.
const twoRoleConfig = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: \".\"\ngit:\n  integration_branch: main\n  branch_prefix: feat\nroles:\n  core:\n    name: core\n    repositories:\n      - path: \".\"\n  qa:\n    name: qa\n    repositories:\n      - path: \".\"\n"

func loadSealed(t *testing.T, ws workspacetest.Workspace) kickoff.Kickoff {
	t.Helper()
	loaded, err := kickoff.Load(filepath.Join(ws.Home, "changes", sealChange))
	if err != nil || loaded == nil {
		t.Fatalf("kickoff.yaml debía existir: %+v, %v", loaded, err)
	}
	return *loaded
}

// REQ-25.36, escenario «El sellado escribe la instantánea»: desde la rama del cambio, en la misma
// única escritura, `kickoff.yaml` lleva la integración, el prefijo, la base y los nombres
// `feat/x`, `feat/x-core`, `feat/x-qa` y `feat/x-archive`, sin SHAs ni rutas absolutas.
func TestKickoffSealWritesTheGitSnapshotOnce(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, twoRoleConfig)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")

	if _, err := runSeal(t, ws.Master, append([]string{"--role", "core", "--role", "qa"}, sealBaseArgs...)...); err != nil {
		t.Fatalf("sellar en feat/x: %v", err)
	}

	sealed := loadSealed(t, ws)
	want := &kickoff.GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "feat",
		ChangeBranch:      "feat/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path: ".", Name: filepath.Base(ws.Master), Role: "change_branch", IntegrationBranch: "main", Base: "main",
			Branches: []kickoff.GitSnapshotBranch{
				{Name: "feat/x", Kind: "change"},
				{Name: "feat/x-core", Kind: "role", Role: "core"},
				{Name: "feat/x-qa", Kind: "role", Role: "qa"},
				{Name: "feat/x-archive", Kind: "archive"},
			},
		}},
	}
	if !reflect.DeepEqual(sealed.Git, want) {
		t.Errorf("instantánea = %+v, se esperaba %+v", sealed.Git, want)
	}
	data, err := os.ReadFile(filepath.Join(ws.Home, "changes", sealChange, "kickoff.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`\b[0-9a-f]{40}\b`).Match(data) || strings.Contains(string(data), ws.Master) || strings.Contains(string(data), filepath.ToSlash(ws.Master)) {
		t.Errorf("kickoff.yaml no debe contener SHAs ni rutas absolutas:\n%s", data)
	}
	// Con -v se ve el bloque tal como queda escrito (el arnés en proceso, sin binario propio).
	if _, block, found := strings.Cut(string(data), "\ngit:"); found {
		t.Logf("bloque git: de kickoff.yaml:\ngit:%s", block)
	}
}

// REQ-25.36, escenario «La instantánea no se reescribe»: un segundo sellado desde la rama
// correcta devuelve al ganador y no modifica kickoff.yaml; desde `main` la instantánea ya no
// exime y el segundo sellado se rechaza como `wrong_branch` sin tocar nada.
func TestKickoffSealDoesNotRewriteTheSnapshotOnASecondSeal(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, twoRoleConfig)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("primer sellado: %v", err)
	}
	path := filepath.Join(ws.Home, "changes", sealChange, "kickoff.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runSeal(t, ws.Master, "--execution-style", "checkpointed", "--handoff-policy", "none", "--role", "core", "--role", "qa")
	if err != nil {
		t.Fatalf("segundo sellado en la rama correcta: %v", err)
	}
	if !strings.Contains(out, "Kickoff ya estaba sellado") || !strings.Contains(out, "continuous") {
		t.Errorf("la salida debe mostrar al ganador, sellado con continuous:\n%s", out)
	}

	integrateGit(t, ws.Master, "switch", "-q", "main")
	_, err = runSeal(t, ws.Master, sealBaseArgs...)
	wantWrongBranch(t, err, "main", "feat/x", ws.Master)

	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Errorf("kickoff.yaml cambió:\n%s\n---\n%s", before, after)
	}
}

// REQ-25.37: `seal --infer` no escribe instantánea aunque la rama sea la del cambio.
func TestKickoffSealInferWritesNoSnapshot(t *testing.T) {
	captureSealWarnings(t)
	ws := embeddedSealProject(t, twoRoleConfig)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")

	if _, err := runSeal(t, ws.Master, "--infer"); err != nil {
		t.Fatalf("seal --infer: %v", err)
	}

	if sealed := loadSealed(t, ws); sealed.Git != nil {
		t.Errorf("--infer no escribe instantánea: %+v", sealed.Git)
	}
}

// REQ-25.42 (fuera de git): sin repositorio no hay política que instantaneizar; se sella igual,
// con aviso y sin bloque `git:`.
func TestKickoffSealOutsideGitWritesNoSnapshot(t *testing.T) {
	warnings := captureSealWarnings(t)
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", sealChange, "proposal.md"), "# x\n")

	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("sellar fuera de git: %v", err)
	}

	if sealed := loadSealed(t, ws); sealed.Git != nil {
		t.Errorf("fuera de git no hay instantánea: %+v", sealed.Git)
	}
	if !strings.Contains(warnings.String(), "git") {
		t.Errorf("fuera de git se avisa: %q", warnings.String())
	}
}

// REQ-25.36 y REQ-25.38: en un multirepo la instantánea lista el repositorio de specs in situ,
// sin ramas, y los de código con la rama del cambio, con rutas relativas a la carpeta maestra.
func TestKickoffSealOfAMultirepoWritesTheSnapshotOfEveryRepository(t *testing.T) {
	captureSealWarnings(t)
	ws := multirepoSealProject(t)
	change := planningHomeChange

	var out bytes.Buffer
	args := append([]string{"seal", "--cwd", ws.Master, "--change", change, "--role", "backend", "--role", "frontend"}, sealBaseArgs...)
	if err := RunSDDKickoff(args, &out); err != nil {
		t.Fatalf("sellar el multirepo: %v", err)
	}

	sealed, err := kickoff.Load(filepath.Join(ws.Home, "changes", change))
	if err != nil || sealed == nil || sealed.Git == nil {
		t.Fatalf("kickoff.yaml debía llevar instantánea: %+v, %v", sealed, err)
	}
	snapshot := sealed.Git
	if snapshot.ChangeBranch != "feat/"+change || len(snapshot.Repositories) != 3 {
		t.Fatalf("rama del cambio %q con %d repositorios, se esperaba feat/%s con tres", snapshot.ChangeBranch, len(snapshot.Repositories), change)
	}
	specs, backend, frontend := snapshot.Repositories[0], snapshot.Repositories[1], snapshot.Repositories[2]
	if specs.Path != "repo-specs" || specs.Role != "specs_in_place" || len(specs.Branches) != 0 {
		t.Errorf("repositorio de specs = %+v, se esperaba in situ y sin ramas", specs)
	}
	for _, code := range []kickoff.GitSnapshotRepo{backend, frontend} {
		if code.Role != "code" || len(code.Branches) != 1 || code.Branches[0].Name != "feat/"+change || code.Branches[0].Kind != "change" {
			t.Errorf("repositorio de código = %+v, se esperaba solo la rama del cambio", code)
		}
	}
	if backend.Path != "repo-backend" || frontend.Path != "repo-frontend" {
		t.Errorf("rutas %q y %q, se esperaban relativas a la carpeta maestra", backend.Path, frontend.Path)
	}
}

// multirepoSealProject es el espacio canónico (multirepo) con git en el repositorio de specs y en
// los de código, todos en main.
func multirepoSealProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	isolateIntegrateGit(t)
	ws := workspacetest.Canonical(t)
	for _, dir := range []string{ws.Specs, ws.Backend, ws.Frontend} {
		integrateInitRepo(t, dir)
		// repo-frontend no tiene ficheros: el commit base puede ser vacío.
		integrateGit(t, dir, "add", "-A")
		integrateGit(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	}
	return ws
}

// REQ-25.38, escenarios «Sellar en un multirepo exige la integración del repositorio de specs»
// y «...con el repositorio de specs en su integración procede»: solo se comprueba el
// repositorio de specs; la rama de `repo-backend` no interviene.
func TestKickoffSealOfAMultirepoRequiresTheSpecsIntegrationBranchAndIgnoresTheCodeRepositories(t *testing.T) {
	captureSealWarnings(t)
	ws := multirepoSealProject(t)
	change := planningHomeChange
	seal := func() error {
		var out bytes.Buffer
		return RunSDDKickoff(append([]string{"seal", "--cwd", ws.Master, "--change", change}, sealBaseArgs...), &out)
	}

	integrateGit(t, ws.Specs, "switch", "-q", "-c", "feat/"+change+"-core")
	err := seal()
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("specs en una rama de rol: error = %v, se esperaba wrong_branch", err)
	}
	if !strings.Contains(err.Error(), "«main»") || !strings.Contains(err.Error(), "repo-specs") {
		t.Errorf("el rechazo %q debe nombrar el repositorio de specs y su integración «main»", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(ws.Home, "changes", change, "kickoff.yaml")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("el rechazo no debe escribir kickoff.yaml (%v)", statErr)
	}

	// Con specs en su integración procede aunque repo-backend esté en una rama cualquiera.
	integrateGit(t, ws.Specs, "switch", "-q", "main")
	integrateGit(t, ws.Backend, "switch", "-q", "-c", "ajena")
	if err := seal(); err != nil {
		t.Fatalf("specs en main: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws.Home, "changes", change, "kickoff.yaml")); statErr != nil {
		t.Errorf("el sellado debe escribir kickoff.yaml: %v", statErr)
	}
}
