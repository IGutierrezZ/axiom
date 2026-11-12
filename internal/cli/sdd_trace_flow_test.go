package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// REQ-29.9: la traza de un cambio ISO recorrido de principio a fin sobre
// repositorios git reales. Los nombres empiezan por TestSDDTrace.

// traceIntegrate integra branch en main con la estrategia de fusión indicada, como
// lo haría una PR: merge (commit de fusión), squash (un commit) o rebase (los
// commits de la rama sobre main). Devuelve el SHA de main tras integrar.
func traceIntegrate(t *testing.T, repo, branch, mode string) string {
	t.Helper()
	switch mode {
	case "merge":
		traceGit(t, repo, "checkout", "-q", "main")
		traceGit(t, repo, "merge", "-q", "--no-ff", branch, "-m", "merge "+branch)
	case "squash":
		traceGit(t, repo, "checkout", "-q", "main")
		traceGit(t, repo, "merge", "-q", "--squash", branch)
		traceGit(t, repo, "commit", "-q", "-m", "squash "+branch)
	case "rebase":
		traceGit(t, repo, "checkout", "-q", branch)
		traceGit(t, repo, "rebase", "-q", "main")
		traceGit(t, repo, "checkout", "-q", "main")
		traceGit(t, repo, "merge", "-q", "--ff-only", branch)
	default:
		t.Fatalf("estrategia de fusión %q desconocida", mode)
	}
	return traceGit(t, repo, "rev-parse", "HEAD")
}

// traceDiverge añade a main un commit ajeno al cambio, para que merge y rebase no
// sean avances rápidos triviales.
func traceDiverge(t *testing.T, repo, name string) {
	t.Helper()
	traceGit(t, repo, "checkout", "-q", "main")
	writeCLIFixture(t, filepath.Join(repo, name), name+"\n")
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "ajeno "+name)
}

const (
	traceDelta  = "## ADDED Requirements\n\n### Requirement: REQ-30.1 Cobro con IVA\n\nEl sistema cobra con IVA.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n- **ENTONCES** incluye IVA\n"
	traceTasks  = "# Tareas\n\n- [x] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
	traceVerify = "# Informe\n\n## Matriz de verificación\n\n" + archiveCheckMatrixHeader +
		"| REQ-30.1 | Cobro general | unitario | internal/pagos/cobro_test.go:TestCobraIVA | go test ./internal/pagos | 0 | | | | pass | |\n"
)

// traceFlow es un cambio ISO recorrido sobre un repositorio real: la PR del cambio
// (con sus tasks y su verify-report) y, aparte, la PR documental de archivado.
type traceFlow struct {
	repo        string
	codeCommit  string // main tras integrar el cambio
	archiveSHA  string // main tras integrar el archivado
	archiveName string
}

// newTraceFlow monta el repositorio hasta la integración del código, o hasta el
// archivado también si archive es verdadero.
func newTraceFlow(t *testing.T, mode string, archive bool) traceFlow {
	t.Helper()
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "sdd:\n  iso: true\n")
	writeCLIFixture(t, filepath.Join(repo, "openspec", "specs", "pagos", "spec.md"), archiveCheckBaseline51)
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "base")

	flow := traceFlow{repo: repo, archiveName: "2026-11-12-demo"}
	changeRoot := filepath.Join(repo, "openspec", "changes", "demo")

	// PR del cambio: dos commits, el segundo con verify-report.md.
	traceGit(t, repo, "checkout", "-q", "-b", "inc/demo")
	writeCLIFixture(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
	writeCLIFixture(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), traceDelta)
	writeCLIFixture(t, filepath.Join(changeRoot, "tasks.md"), traceTasks)
	sealTestChange(t, changeRoot, "demo", true)
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "plan")
	writeCLIFixture(t, filepath.Join(changeRoot, "verify-report.md"), traceVerify)
	writeCLIFixture(t, filepath.Join(repo, "internal", "pagos", "cobro.go"), "package pagos\n")
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "código y verify")
	traceDiverge(t, repo, "ajeno-1.md")
	flow.codeCommit = traceIntegrate(t, repo, "inc/demo", mode)
	if !archive {
		return flow
	}

	// La compuerta integration registra el commit de la integración (evidencia
	// pr_merged) y la PR documental archiva la carpeta.
	traceGit(t, repo, "checkout", "-q", "-b", "inc/demo-archive")
	if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
		Gate: kickoff.GateIntegration, Decision: kickoff.DecisionApproved, Reason: "integrado",
		EvidenceKind: kickoff.EvidencePRMerged, EvidenceRef: flow.codeCommit, Verified: true, Actor: "prueba",
	}); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(repo, "openspec", "changes", "archive", flow.archiveName)
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(changeRoot, archived); err != nil {
		t.Fatal(err)
	}
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "archivar demo")
	traceDiverge(t, repo, "ajeno-2.md")
	flow.archiveSHA = traceIntegrate(t, repo, "inc/demo-archive", mode)
	return flow
}

// traceJSON es el informe de `sdd trace --json` tal como lo decodifican las pruebas.
type traceJSON struct {
	Schema            string `json:"schema"`
	Change            string `json:"change"`
	IntegrationBranch string `json:"integrationBranch"`
	CodeCommit        struct {
		Commit string `json:"commit"`
		Found  bool   `json:"found"`
	} `json:"codeCommit"`
	ArchiveCommit struct {
		Commit string `json:"commit"`
		Found  bool   `json:"found"`
	} `json:"archiveCommit"`
	ArchiveFolder string `json:"archiveFolder"`
	Discrepancy   string `json:"discrepancy"`
	ChainSource   string `json:"chainSource"`
	Chain         []struct {
		Requirement string `json:"requirement"`
		Cubre       []struct {
			File string `json:"file"`
			Line int    `json:"line"`
		} `json:"cubre"`
		Rows []struct {
			Scenario string `json:"scenario"`
			Result   string `json:"result"`
		} `json:"rows"`
	} `json:"chain"`
}

func runTrace(t *testing.T, repo string, extra ...string) traceJSON {
	t.Helper()
	var out bytes.Buffer
	if err := RunSDDTrace(append([]string{"--cwd", repo, "--change", "demo", "--json"}, extra...), &out); err != nil {
		t.Fatalf("RunSDDTrace() error = %v\n%s", err, out.String())
	}
	var report traceJSON
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("salida JSON no válida: %v\n%s", err, out.String())
	}
	return report
}

// treeSnapshotWithoutGit es treeSnapshot sin el directorio .git.
func treeSnapshotWithoutGit(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := treeSnapshot(t, root)
	for path := range snapshot {
		if relative, err := filepath.Rel(root, path); err == nil && strings.HasPrefix(filepath.ToSlash(relative), ".git/") {
			delete(snapshot, path)
		}
	}
	return snapshot
}

// REQ-29.9: la misma cadena requisito, `Cubre:`, fila, commit de código y commit
// de archivado con los tres modos de fusión, sin escribir nada en el árbol.
func TestSDDTraceSameChainWithSquashMergeAndRebase(t *testing.T) {
	for _, mode := range []string{"squash", "merge", "rebase"} {
		t.Run(mode, func(t *testing.T) {
			flow := newTraceFlow(t, mode, true)
			before := treeSnapshotWithoutGit(t, flow.repo)
			status := traceGit(t, flow.repo, "status", "--porcelain")

			report := runTrace(t, flow.repo)

			if report.Schema != "axiom.sdd-trace/v1" || report.Change != "demo" || report.IntegrationBranch != "main" {
				t.Errorf("cabecera = %+v", report)
			}
			if !report.CodeCommit.Found || report.CodeCommit.Commit != flow.codeCommit {
				t.Errorf("commit de código = %+v, se esperaba %s", report.CodeCommit, flow.codeCommit)
			}
			if !report.ArchiveCommit.Found || report.ArchiveCommit.Commit != flow.archiveSHA {
				t.Errorf("commit de archivado = %+v, se esperaba %s", report.ArchiveCommit, flow.archiveSHA)
			}
			if report.ArchiveFolder != flow.archiveName {
				t.Errorf("carpeta archivada = %q, se esperaba %q", report.ArchiveFolder, flow.archiveName)
			}
			if report.Discrepancy != "" {
				t.Errorf("discrepancia = %q, la evidencia pr_merged coincide", report.Discrepancy)
			}
			if report.ChainSource != "archived" || len(report.Chain) != 1 {
				t.Fatalf("cadena = %+v (%s)", report.Chain, report.ChainSource)
			}
			entry := report.Chain[0]
			if entry.Requirement != "REQ-30.1" || len(entry.Cubre) != 1 || entry.Cubre[0].File != "tasks.md" || entry.Cubre[0].Line != 4 ||
				len(entry.Rows) != 1 || entry.Rows[0].Scenario != "Cobro general" || entry.Rows[0].Result != "pass" {
				t.Errorf("cadena = %+v", entry)
			}

			after := treeSnapshotWithoutGit(t, flow.repo)
			if len(after) != len(before) {
				t.Errorf("la traza cambió el árbol: antes %d ficheros, después %d", len(before), len(after))
			}
			for path, text := range before {
				if after[path] != text {
					t.Errorf("la traza modificó %s", path)
				}
			}
			if got := traceGit(t, flow.repo, "status", "--porcelain"); got != status {
				t.Errorf("git status cambió:\nantes:\n%s\ndespués:\n%s", status, got)
			}
		})
	}
}

// Antes del archivado hay commit de código y no de archivado, y la cadena sale de
// la carpeta activa.
func TestSDDTraceBeforeArchiving(t *testing.T) {
	flow := newTraceFlow(t, "squash", false)
	report := runTrace(t, flow.repo)
	if !report.CodeCommit.Found || report.CodeCommit.Commit != flow.codeCommit {
		t.Errorf("commit de código = %+v, se esperaba %s", report.CodeCommit, flow.codeCommit)
	}
	if report.ArchiveCommit.Found || report.ArchiveFolder != "" {
		t.Errorf("sin archivar: commit %+v, carpeta %q", report.ArchiveCommit, report.ArchiveFolder)
	}
	if report.ChainSource != "active" || len(report.Chain) != 1 {
		t.Errorf("cadena = %+v (%s)", report.Chain, report.ChainSource)
	}
}

// Un cambio que aún no se integró no es un error: la traza informa de la ausencia.
func TestSDDTraceChangeNotIntegratedYet(t *testing.T) {
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "sdd:\n  iso: true\n")
	writeCLIFixture(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "base")

	report := runTrace(t, repo)
	if report.CodeCommit.Found || report.ArchiveCommit.Found || report.ChainSource != "none" || len(report.Chain) != 0 {
		t.Errorf("informe = %+v, se esperaba la ausencia de todo", report)
	}

	var out bytes.Buffer
	if err := RunSDDTrace([]string{"--cwd", repo, "--change", "demo"}, &out); err != nil {
		t.Fatalf("RunSDDTrace() texto: %v", err)
	}
	if !strings.Contains(out.String(), "no hay") {
		t.Errorf("el texto no informa de la ausencia:\n%s", out.String())
	}
}

// Una evidencia pr_merged que no es el commit de código ni un ancestro suyo se
// informa como discrepancia, nunca como error.
func TestSDDTraceDiscrepancyWithEvidence(t *testing.T) {
	flow := newTraceFlow(t, "merge", true)
	// La evidencia apunta al commit inicial del repositorio: el commit de código
	// no es ese commit ni un ancestro suyo.
	first := traceGit(t, flow.repo, "rev-list", "--max-parents=0", "main")
	archived := filepath.Join(flow.repo, "openspec", "changes", "archive", flow.archiveName)
	if err := os.Remove(filepath.Join(archived, kickoff.GatesFileName)); err != nil {
		t.Fatal(err)
	}
	if err := kickoff.AppendGate(archived, kickoff.GateRecord{
		Gate: kickoff.GateIntegration, Decision: kickoff.DecisionApproved, Reason: "integrado",
		EvidenceKind: kickoff.EvidencePRMerged, EvidenceRef: first, Verified: true, Actor: "prueba",
	}); err != nil {
		t.Fatal(err)
	}

	report := runTrace(t, flow.repo)
	if !strings.Contains(report.Discrepancy, first) {
		t.Errorf("discrepancia = %q, se esperaba que nombrase la evidencia %s", report.Discrepancy, first)
	}
	if !report.CodeCommit.Found {
		t.Errorf("la discrepancia no debe impedir derivar el commit de código: %+v", report.CodeCommit)
	}
}

// Una rama de integración que no existe es un error que nombra la opción.
func TestSDDTraceUnknownIntegrationBranch(t *testing.T) {
	flow := newTraceFlow(t, "squash", false)
	var out bytes.Buffer
	err := RunSDDTrace([]string{"--cwd", flow.repo, "--change", "demo", "--integration-branch", "no-existe"}, &out)
	if err == nil || !strings.Contains(err.Error(), "--integration-branch") {
		t.Fatalf("error = %v, se esperaba que nombrase --integration-branch", err)
	}
}

// La bandera manda sobre la rama por defecto.
func TestSDDTraceIntegrationBranchFlag(t *testing.T) {
	flow := newTraceFlow(t, "squash", false)
	traceGit(t, flow.repo, "branch", "release", "main")
	if got := runTrace(t, flow.repo).IntegrationBranch; got != "main" {
		t.Errorf("sin bandera = %q, se esperaba main", got)
	}
	if got := runTrace(t, flow.repo, "--integration-branch", "release"); got.IntegrationBranch != "release" || !got.CodeCommit.Found {
		t.Errorf("con --integration-branch release = %+v", got)
	}
}
