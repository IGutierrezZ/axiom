package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// RF-RID-08: con el patrón inaccesible `sdd archive-check` y `sdd trace` terminan con error y
// la continuación `axiom workspace validate --path <proyecto>`.
func TestSDDArchiveCheckInaccessiblePatternFailsClosedWithContinuation(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone,
		verify: archiveCheckVerify(archiveCheckRowPass), iso: true,
	})
	writeSDDAxiomYAML(t, root, "sdd:\n  iso: true\n  id_pattern: \"<m><m>\"\n")
	out, err := runArchiveCheck(root, "demo")
	if err == nil {
		t.Fatalf("un patrón inválido debe terminar con error:\n%s", out)
	}
	all := err.Error() + "\n" + out
	for _, want := range []string{"axiom workspace validate --path", "id_pattern", "iso_declaration_invalid"} {
		if !strings.Contains(all, want) {
			t.Errorf("la salida y el error deben contener %q:\n%s", want, all)
		}
	}
}

func TestSDDTraceInaccessiblePatternFailsClosedWithContinuation(t *testing.T) {
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "sdd:\n  id_pattern: \"<m><m>\"\n")
	var out bytes.Buffer
	err := RunSDDTrace([]string{"--cwd", repo, "--change", "demo"}, &out)
	if err == nil {
		t.Fatalf("un patrón inválido debe terminar con error:\n%s", out.String())
	}
	for _, want := range []string{"axiom workspace validate --path", "id_pattern"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
}

// RF-ARC-11 y RF-ISO-09: `alreadyComposed` trata un REMOVED por identificador con el título
// obsoleto como «no compuesto» mientras el requisito sigue en la canónica y como «compuesto»
// cuando ya no se resuelve.
func TestSDDArchiveCheckAlreadyComposedRemovedByID(t *testing.T) {
	const canonical = "# pagos\n\n## Requirements\n\n" +
		"### Requirement: Recibo (REQ-5.1)\n\nEl sistema emite recibo.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n\n" +
		"### Requirement: Sobra (REQ-5.3)\n\nSe retira.\n\n#### Scenario: Sobra\n\n- **CUANDO** y\n"
	const delta = "## REMOVED Requirements\n\n### Requirement: Sobra antigua (REQ-5.3)\n\n(Reason: ya no aplica)\n"

	for _, supersede := range []bool{false, true} {
		name := "REMOVED borrado"
		if supersede {
			name = "REMOVED superado"
		}
		t.Run(name, func(t *testing.T) {
			root, _ := archiveCheckWorkspace(t, archiveCheckChange{
				name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": delta}, tasks: "- [x] 1.1 T\n", verify: archiveCheckVerify(), iso: true,
			})
			writeCLIFixture(t, filepath.Join(root, "openspec", "specs", "pagos", "spec.md"), canonical)
			composedNow := func() bool {
				t.Helper()
				out, _ := runArchiveCheck(root, "demo", "--json")
				var report struct {
					Domains []struct {
						Domain          string `json:"domain"`
						AlreadyComposed bool   `json:"alreadyComposed"`
					} `json:"domains"`
				}
				if err := json.Unmarshal([]byte(out), &report); err != nil || len(report.Domains) != 1 {
					t.Fatalf("salida JSON no válida (%v):\n%s", err, out)
				}
				return report.Domains[0].AlreadyComposed
			}
			if composedNow() {
				t.Fatal("antes de componer el requisito sigue resolviéndose por identificador: no compuesto")
			}
			composed, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonical, delta, sddstatus.ComposeOptions{SupersedeRemoved: supersede})
			if err != nil {
				t.Fatalf("compose: %v", err)
			}
			writeCLIFixture(t, filepath.Join(root, "openspec", "specs", "pagos", "spec.md"), composed)
			if !composedNow() {
				t.Fatalf("tras componer el requisito ya no se resuelve: compuesto\n%s", composed)
			}
		})
	}
}

// RF-RID-07 y RF-ISO-09: con `RF-<DOM>-<NN>` `sdd trace` encadena RF-USU-01.
func TestReqidRFTraceChainsAnIdentifier(t *testing.T) {
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	changeRoot := filepath.Join(repo, "openspec", "changes", "demo")
	writeCLIFixture(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
	writeCLIFixture(t, filepath.Join(changeRoot, "specs", "usuarios", "spec.md"),
		"## ADDED Requirements\n\n### Requirement: Alta de usuario (RF-USU-01)\n\nEl sistema da de alta usuarios.\n\n#### Scenario: Alta correcta\n\n- **CUANDO** se da de alta\n- **ENTONCES** queda registrado\n")
	writeCLIFixture(t, filepath.Join(changeRoot, "tasks.md"), "# Tareas\n\n- [x] 1.1 Trabajo\n  Cubre: RF-USU-01\n")
	writeCLIFixture(t, filepath.Join(changeRoot, "verify-report.md"), "# Informe\n\n## Matriz de verificación\n\n"+archiveCheckMatrixHeader+
		"| RF-USU-01 | Alta correcta | unitario | internal/usuarios/alta_test.go:TestAlta | go test ./internal/usuarios | 0 | | | | pass | |\n")
	sealTestChange(t, changeRoot, "demo", true)
	traceGit(t, repo, "add", "-A")
	traceGit(t, repo, "commit", "-q", "-m", "plan")

	report := runTrace(t, repo)
	if len(report.Chain) != 1 || report.Chain[0].Requirement != "RF-USU-01" {
		t.Fatalf("cadena = %+v, se esperaba RF-USU-01", report.Chain)
	}
	if len(report.Chain[0].Cubre) != 1 || len(report.Chain[0].Rows) != 1 {
		t.Errorf("la cadena debe llevar la línea Cubre y la fila de la matriz: %+v", report.Chain[0])
	}
}
