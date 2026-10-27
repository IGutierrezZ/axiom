package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
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
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("primer sellado: %v", err)
	}
	integrateGit(t, ws.Master, "switch", "-q", "main")
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
