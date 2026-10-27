package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// isolateGateGitEnv aísla los procesos git que lanza la prueba de la
// configuración del desarrollador: HOME propio, sin configuración global ni de
// sistema, y autor y committer fijos. Usa t.Setenv, así que la prueba que la
// invoca no puede ser paralela.
func isolateGateGitEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Prueba de compuerta")
	t.Setenv("GIT_AUTHOR_EMAIL", "compuerta@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Prueba de compuerta")
	t.Setenv("GIT_COMMITTER_EMAIL", "compuerta@example.com")
}

// commitInGateTestRepo inicializa dir como repositorio git en la rama main, crea
// un fichero con contenido inventado y lo commitea, y devuelve el SHA del commit.
func commitInGateTestRepo(t *testing.T, dir, message string) string {
	t.Helper()
	runGateTestGit(t, dir, "init", "-q", "-b", "main")
	runGateTestGit(t, dir, "add", "-A")
	runGateTestGit(t, dir, "commit", "-q", "-m", message)
	return strings.TrimSpace(runGateTestGit(t, dir, "rev-parse", "HEAD"))
}

// gitHasCommit informa de si dir conoce el commit sha en su grafo.
func gitHasCommit(dir, sha string) bool {
	return exec.Command("git", "-C", dir, "cat-file", "-e", sha+"^{commit}").Run() == nil
}

// readGatesYAMLOrNil devuelve los bytes de gates.yaml, o nil si todavía no existe.
func readGatesYAMLOrNil(t *testing.T, changeRoot string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(changeRoot, "gates.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("leer gates.yaml: %v", err)
	}
	return data
}

// REQ-24.12 y REQ-21.16: en el espacio canónico, la evidencia pr_merged de la
// compuerta integration se verifica contra el repositorio de especificaciones y
// no contra los repositorios de código, aunque el verbo se lance desde la
// carpeta maestra. S está fusionado en la rama principal de repo-specs; C solo
// existe en repo-backend.
func TestSDDGateRecordPRMergedVerifiesAgainstSpecsRepositoryInCanonicalWorkspace(t *testing.T) {
	isolateGateGitEnv(t)
	ws := workspacetest.Canonical(t)
	changeRoot := filepath.Join(ws.Home, "changes", planningHomeChange)

	// El cambio queda sellado antes del commit S: el kickoff.yaml forma parte
	// del contenido de repo-specs, como en un cambio real.
	sealForGateTest(t, ws.Master, planningHomeChange)
	specsCommit := commitInGateTestRepo(t, ws.Specs, "cambio inc-99-ejemplo sellado")

	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "src", "servicio.txt"), "servicio de ejemplo\n")
	codeCommit := commitInGateTestRepo(t, ws.Backend, "servicio de ejemplo")

	if specsCommit == codeCommit {
		t.Fatalf("S y C deben ser commits distintos, ambos son %s", specsCommit)
	}
	if gitHasCommit(ws.Specs, codeCommit) {
		t.Fatalf("precondición rota: C (%s) no debe existir en repo-specs", codeCommit)
	}
	if gitHasCommit(ws.Backend, specsCommit) {
		t.Fatalf("precondición rota: S (%s) no debe existir en repo-backend", specsCommit)
	}

	record := func(commit string) error {
		return RunSDDGate([]string{
			"record", "--cwd", ws.Master, "--change", planningHomeChange,
			"--gate", "integration", "--decision", "approved", "--reason", "PR fusionado en la rama principal",
			"--evidence-kind", "pr_merged", "--commit", commit,
		}, &bytes.Buffer{})
	}

	t.Run("C ausente del repositorio de especificaciones se rechaza sin anexar nada", func(t *testing.T) {
		before := readGatesYAMLOrNil(t, changeRoot)

		err := record(codeCommit)
		if err == nil {
			t.Fatalf("gate record --commit %s = nil, se esperaba el rechazo: C no está en el grafo de repo-specs", codeCommit)
		}
		if !strings.Contains(err.Error(), codeCommit) || !strings.Contains(err.Error(), "ancestro") || !strings.Contains(err.Error(), "main") {
			t.Fatalf("el rechazo debe nombrar el commit %s, la rama main y la causa (ancestro): %v", codeCommit, err)
		}
		if after := readGatesYAMLOrNil(t, changeRoot); !bytes.Equal(before, after) {
			t.Fatalf("gates.yaml cambió pese al rechazo:\nantes:\n%s\ndespués:\n%s", before, after)
		}
		ledger, loadErr := kickoff.LoadGates(changeRoot)
		if loadErr != nil {
			t.Fatalf("kickoff.LoadGates() error = %v", loadErr)
		}
		if len(ledger.Records) != 0 {
			t.Fatalf("gates.yaml tiene %d registros tras el rechazo, se esperaban 0", len(ledger.Records))
		}
	})

	t.Run("S fusionado en la rama principal de repo-specs se acepta y se anexa", func(t *testing.T) {
		before := readGatesYAMLOrNil(t, changeRoot)

		if err := record(specsCommit); err != nil {
			t.Fatalf("gate record --commit %s: %v", specsCommit, err)
		}
		if after := readGatesYAMLOrNil(t, changeRoot); bytes.Equal(before, after) {
			t.Fatal("gates.yaml no cambió tras aceptar la evidencia: no se anexó el registro")
		}
		ledger, loadErr := kickoff.LoadGates(changeRoot)
		if loadErr != nil {
			t.Fatalf("kickoff.LoadGates() error = %v", loadErr)
		}
		if len(ledger.Records) != 1 {
			t.Fatalf("gates.yaml tiene %d registros, se esperaba exactamente 1", len(ledger.Records))
		}
		rec := ledger.Records[0]
		if rec.Gate != kickoff.GateIntegration || rec.Decision != kickoff.DecisionApproved ||
			!rec.Verified || rec.EvidenceRef != specsCommit || rec.EvidenceBaseRef != "main" {
			t.Fatalf("registro = %#v, se esperaba integration aprobada con Verified=true, EvidenceRef=%s y EvidenceBaseRef=main", rec, specsCommit)
		}
	})
}

// D-22 (REQ-25.23, escenario 5): sin --base-ref, la evidencia pr_merged se comprueba contra la
// rama de integración resuelta del repositorio del planning home (`git.integration_branch`), no
// contra un `main` fijo. El repositorio solo tiene la rama `develop`: con el literal `main` la
// comprobación de ancestro fallaría porque esa referencia no existe.
func TestSDDGateRecordCompletesTheBaseRefWithTheResolvedIntegration(t *testing.T) {
	isolateGateGitEnv(t)
	root, changeRoot := newGovernanceWorkspace(t, planningHomeChange)
	writeCLIFixture(t, filepath.Join(root, "axiom.yaml"),
		"workspace:\n  name: integracion-develop\n  topology: monorepo-embedded\ngit: {integration_branch: develop}\n"+
			"roles:\n  core:\n    name: core\n    repositories:\n      - path: .\n")
	writeCLIFixture(t, filepath.Join(root, "README.md"), "raíz\n")
	runGateTestGit(t, root, "init", "-q", "-b", "develop")
	runGateTestGit(t, root, "add", "-A")
	runGateTestGit(t, root, "commit", "-q", "-m", "raíz")
	commit := strings.TrimSpace(runGateTestGit(t, root, "rev-parse", "HEAD"))
	retroSealForGuard(t, root, planningHomeChange)

	var out bytes.Buffer
	err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", planningHomeChange,
		"--gate", "integration", "--decision", "approved", "--reason", "PR fusionado en develop",
		"--evidence-kind", "pr_merged", "--commit", commit,
	}, &out)
	if err != nil {
		t.Fatalf("gate record sin --base-ref con integración develop: %v", err)
	}
	ledger, loadErr := kickoff.LoadGates(changeRoot)
	if loadErr != nil {
		t.Fatalf("kickoff.LoadGates() error = %v", loadErr)
	}
	if len(ledger.Records) != 1 {
		t.Fatalf("gates.yaml tiene %d registros, se esperaba exactamente 1", len(ledger.Records))
	}
	if rec := ledger.Records[0]; !rec.Verified || rec.EvidenceBaseRef != "develop" || rec.EvidenceRef != commit {
		t.Fatalf("registro = %#v, se esperaba Verified=true, EvidenceBaseRef=develop y EvidenceRef=%s", rec, commit)
	}
}
