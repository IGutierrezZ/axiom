package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// REQ-29.10, REQ-29.11 y REQ-29.3: `axiom sdd archive-check` aplica la regla del
// archivado ISO sin escribir nada. Las pruebas se llaman TestSDDArchiveCheck*
// para entrar en el mismo patrón de `-run`.

const (
	archiveCheckDeltaOK      = "## ADDED Requirements\n\n### Requirement: REQ-30.1 Cobro con IVA\n\nEl sistema cobra con IVA.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n- **ENTONCES** incluye IVA\n"
	archiveCheckDeltaNoScene = "## ADDED Requirements\n\n### Requirement: REQ-30.1 Cobro con IVA\n\nEl sistema cobra con IVA.\n"
	archiveCheckTasksDone    = "- [x] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
	archiveCheckTasksOpen    = "- [ ] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
	archiveCheckBugReport    = "# Bug\n\n## Informe de bug\n\nObservado: no emite recibo\nEsperado: emite recibo\n"
	archiveCheckBaseline51   = "# pagos\n\n## Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n"
	archiveCheckMatrixHeader = "| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |\n|---|---|---|---|---|---|---|---|---|---|---|\n"
	archiveCheckRowPass      = "| REQ-30.1 | Cobro general | unitario | internal/pagos/cobro_test.go:TestCobraIVA | go test ./internal/pagos -run TestCobraIVA -count=1 | 0 | | | | pass | |\n"
)

func archiveCheckVerify(rows ...string) string {
	return "# Verificación\n\n## Matriz de verificación\n\n" + archiveCheckMatrixHeader + strings.Join(rows, "")
}

// archiveCheckChange describe el cambio que monta archiveCheckWorkspace.
type archiveCheckChange struct {
	name     string
	proposal string
	deltas   map[string]string
	tasks    string
	verify   string
	// iso sella el cambio con la instantánea ISO.
	iso bool
}

// archiveCheckWorkspace monta un proyecto embebido con `sdd: {iso: true}` y un
// cambio sellado. Devuelve la raíz del workspace y la del cambio.
func archiveCheckWorkspace(t *testing.T, change archiveCheckChange) (root, changeRoot string) {
	t.Helper()
	root = t.TempDir()
	writeSDDAxiomYAML(t, root, "sdd:\n  iso: true\n")
	writeCLIFixture(t, filepath.Join(root, "internal", "pagos", "cobro_test.go"),
		"package pagos\n\nimport \"testing\"\n\nfunc TestCobraIVA(t *testing.T) {}\n")
	changeRoot = filepath.Join(root, "openspec", "changes", change.name)
	writeCLIFixture(t, filepath.Join(changeRoot, "proposal.md"), change.proposal)
	writeCLIFixture(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
	for domain, text := range change.deltas {
		writeCLIFixture(t, filepath.Join(changeRoot, "specs", domain, "spec.md"), text)
	}
	if change.tasks != "" {
		writeCLIFixture(t, filepath.Join(changeRoot, "tasks.md"), change.tasks)
	}
	if change.verify != "" {
		writeCLIFixture(t, filepath.Join(changeRoot, "verify-report.md"), change.verify)
	}
	snapshot := &kickoff.SDDSnapshot{ISO: change.iso}
	if !change.iso {
		snapshot = nil
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: change.name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
	return root, changeRoot
}

// treeSnapshot devuelve el contenido de todos los ficheros bajo root.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		snapshot[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func runArchiveCheck(root, change string, extra ...string) (string, error) {
	var out bytes.Buffer
	err := RunSDDArchiveCheck(append([]string{"--cwd", root, "--change", change}, extra...), &out)
	return out.String(), err
}

func TestSDDArchiveCheck(t *testing.T) {
	tests := []struct {
		name    string
		change  archiveCheckChange
		baseln  string
		wantErr bool
		tokens  []string
	}{
		{
			name:   "cambio no ISO sale 0 aunque no cumpla nada",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone},
		},
		{
			name:   "ISO sin hallazgos sale 0",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(archiveCheckRowPass), iso: true},
		},
		{
			name:    "matriz ausente",
			change:  archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, iso: true},
			wantErr: true, tokens: []string{"iso_matrix_missing"},
		},
		{
			name:    "matriz ausente también con tareas pendientes",
			change:  archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksOpen, iso: true},
			wantErr: true, tokens: []string{"iso_matrix_missing"},
		},
		{
			name:    "bug sin delta con requisito existente y sin matriz",
			change:  archiveCheckChange{name: "demo", proposal: archiveCheckBugReport, tasks: "- [x] 1.1 T\n  Cubre: REQ-5.1\n", iso: true},
			baseln:  archiveCheckBaseline51,
			wantErr: true, tokens: []string{"iso_matrix_missing"},
		},
		{
			name:    "capacidad nueva sin escenarios, sin composición posible",
			change:  archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"nueva": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true},
			wantErr: true, tokens: []string{"iso_requirement_scenarios"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := archiveCheckWorkspace(t, tt.change)
			if tt.baseln != "" {
				writeCLIFixture(t, filepath.Join(root, "openspec", "specs", "pagos", "spec.md"), tt.baseln)
			}
			before := treeSnapshot(t, root)

			out, err := runArchiveCheck(root, tt.change.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, se esperaba error=%v\nsalida:\n%s", err, tt.wantErr, out)
			}
			for _, token := range tt.tokens {
				if !strings.Contains(out, "blocked("+token+")") {
					t.Errorf("la salida no incluye blocked(%s):\n%s", token, out)
				}
			}
			after := treeSnapshot(t, root)
			if len(before) != len(after) {
				t.Errorf("archive-check escribió ficheros: antes %d, después %d", len(before), len(after))
			}
			for path, text := range before {
				if after[path] != text {
					t.Errorf("archive-check modificó %s", path)
				}
			}
		})
	}
}

func TestSDDArchiveCheckJSON(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, iso: true,
	})
	out, err := runArchiveCheck(root, "demo", "--json")
	if err == nil {
		t.Fatal("se esperaba un error con la matriz ausente")
	}
	var report struct {
		Schema   string `json:"schema"`
		Change   string `json:"change"`
		ISO      bool   `json:"iso"`
		OK       bool   `json:"ok"`
		Findings []struct {
			Token  string `json:"token"`
			Reason string `json:"reason"`
		} `json:"findings"`
		Domains []struct {
			Domain          string `json:"domain"`
			AlreadyComposed bool   `json:"alreadyComposed"`
		} `json:"domains"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("salida JSON no válida: %v\n%s", err, out)
	}
	if report.Schema != "axiom.sdd-archive-check/v1" || report.Change != "demo" || !report.ISO || report.OK {
		t.Errorf("informe = %+v", report)
	}
	if len(report.Findings) != 1 || report.Findings[0].Token != "iso_matrix_missing" || !strings.HasPrefix(report.Findings[0].Reason, "blocked(iso_matrix_missing):") {
		t.Errorf("hallazgos = %+v", report.Findings)
	}
	if len(report.Domains) != 1 || report.Domains[0].Domain != "pagos" || report.Domains[0].AlreadyComposed {
		t.Errorf("dominios = %+v", report.Domains)
	}
}

// El estado de un cambio no ISO no se vuelve ISO por leer axiom.yaml en vivo: la
// instantánea manda, y un cambio no ISO sale sin informar de dominios.
func TestSDDArchiveCheckNonISOReportsNothingElse(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{name: "demo", proposal: "# P\n", tasks: archiveCheckTasksDone})
	out, err := runArchiveCheck(root, "demo", "--json")
	if err != nil {
		t.Fatalf("un cambio no ISO sale sin error: %v", err)
	}
	if !strings.Contains(out, `"iso": false`) || !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"domains": []`) {
		t.Errorf("salida inesperada:\n%s", out)
	}
}

// REQ-29.1: una declaración `sdd:` inválida en un cambio ISO falla cerrada.
func TestSDDArchiveCheckInvalidDeclaration(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone,
		verify: archiveCheckVerify(archiveCheckRowPass), iso: true,
	})
	writeSDDAxiomYAML(t, root, "sdd:\n  iso: quizás\n")
	out, err := runArchiveCheck(root, "demo")
	if err == nil || !strings.Contains(out, "blocked(iso_declaration_invalid)") {
		t.Fatalf("error = %v, salida:\n%s", err, out)
	}
}

// Un kickoff ilegible nunca puede dar salida 0: sddstatus.ISOArchiveRefusal lo
// trata como ausente, así que el verbo lo carga por su cuenta y falla cerrado.
func TestSDDArchiveCheckUnreadableKickoffFailsClosed(t *testing.T) {
	root, changeRoot := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone,
		verify: archiveCheckVerify(archiveCheckRowPass), iso: true,
	})
	if err := os.WriteFile(filepath.Join(changeRoot, kickoff.KickoffFileName), []byte("schema: [no es un mapa\n  - ::"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, loadErr := kickoff.Load(changeRoot); loadErr == nil {
		t.Fatal("la prueba necesita un kickoff ilegible")
	}
	for _, extra := range [][]string{nil, {"--json"}} {
		out, err := runArchiveCheck(root, "demo", extra...)
		if err == nil {
			t.Fatalf("un kickoff ilegible dio salida 0 (args %v):\n%s", extra, out)
		}
		if !strings.Contains(err.Error(), "kickoff") {
			t.Errorf("el error no nombra el kickoff: %v", err)
		}
	}
}

func TestSDDArchiveCheckRejectsBadArguments(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{name: "demo", proposal: "# P\n"})
	for _, change := range []string{"", "../x", "archive", "a:b", "no-existe"} {
		if _, err := runArchiveCheck(root, change); err == nil {
			t.Errorf("--change %q debió rechazarse", change)
		}
	}
	var out bytes.Buffer
	if err := RunSDDArchiveCheck([]string{"--help"}, &out); err != nil || !strings.Contains(out.String(), "archive-check") {
		t.Errorf("--help: err=%v salida=%q", err, out.String())
	}
	if err := RunSDDArchiveCheck([]string{"--cwd", root, "--change", "demo", "sobra"}, &out); err == nil {
		t.Error("un argumento posicional debió rechazarse")
	}
}

// REQ-29.3 y D17: tras componer un dominio con el compositor real, archive-check
// informa alreadyComposed: true para él y false para el resto.
func TestSDDArchiveCheckAlreadyComposed(t *testing.T) {
	const canonicalPagos = "# pagos\n\n## Requirements\n\n" +
		"### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n\n" +
		"### Requirement: REQ-5.2 Antiguo\n\nTexto antiguo.\n\n#### Scenario: Antiguo\n\n- **CUANDO** x\n\n" +
		"### Requirement: REQ-5.3 Sobra\n\nSe retira.\n\n#### Scenario: Sobra\n\n- **CUANDO** y\n"
	const deltaPagos = "## ADDED Requirements\n\n### Requirement: REQ-5.4 Factura\n\nEl sistema factura.\n\n#### Scenario: Factura emitida\n\n- **CUANDO** se cobra\n\n" +
		"## MODIFIED Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo digital.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n\n" +
		"## REMOVED Requirements\n\n### Requirement: REQ-5.3 Sobra\n\n(Reason: ya no aplica)\n\n" +
		"## RENAMED Requirements\n\n### Requirement: REQ-5.2 Antiguo → REQ-5.2 Nuevo\n\n(Reason: aclarar el nombre)\n"
	const canonicalEnvios = "# envios\n\n## Requirements\n\n### Requirement: REQ-6.1 Etiqueta\n\nEl sistema imprime etiqueta.\n\n#### Scenario: Etiqueta impresa\n\n- **CUANDO** se envía\n"
	const deltaEnvios = "## ADDED Requirements\n\n### Requirement: REQ-6.2 Seguimiento\n\nEl sistema da seguimiento.\n\n#### Scenario: Seguimiento activo\n\n- **CUANDO** se envía\n"

	for _, supersede := range []bool{false, true} {
		name := "REMOVED borrado"
		if supersede {
			name = "REMOVED superado (compuesto con --supersede)"
		}
		t.Run(name, func(t *testing.T) {
			root, _ := archiveCheckWorkspace(t, archiveCheckChange{
				name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": deltaPagos, "envios": deltaEnvios},
				tasks: "- [x] 1.1 T\n", verify: archiveCheckVerify(), iso: true,
			})
			specs := filepath.Join(root, "openspec", "specs")
			writeCLIFixture(t, filepath.Join(specs, "pagos", "spec.md"), canonicalPagos)
			writeCLIFixture(t, filepath.Join(specs, "envios", "spec.md"), canonicalEnvios)

			domains := func() map[string]bool {
				t.Helper()
				out, _ := runArchiveCheck(root, "demo", "--json")
				var report struct {
					Domains []struct {
						Domain          string `json:"domain"`
						AlreadyComposed bool   `json:"alreadyComposed"`
					} `json:"domains"`
				}
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatalf("salida JSON no válida: %v\n%s", err, out)
				}
				got := map[string]bool{}
				for _, domain := range report.Domains {
					got[domain.Domain] = domain.AlreadyComposed
				}
				return got
			}

			if got := domains(); got["pagos"] || got["envios"] || len(got) != 2 {
				t.Fatalf("antes de componer = %v, ningún dominio está compuesto", got)
			}

			composed, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonicalPagos, deltaPagos, sddstatus.ComposeOptions{SupersedeRemoved: supersede})
			if err != nil {
				t.Fatalf("ComposeOpenSpecCanonicalSpecWithOptions() error = %v", err)
			}
			writeCLIFixture(t, filepath.Join(specs, "pagos", "spec.md"), composed)

			got := domains()
			if !got["pagos"] || got["envios"] {
				t.Fatalf("tras componer pagos = %v, se esperaba pagos=true y envios=false", got)
			}
		})
	}
}

// Cada resultado negativo de la tolerancia: nombre, bloque o dominio distinto.
func TestSDDArchiveCheckAlreadyComposedNegatives(t *testing.T) {
	const delta = "## ADDED Requirements\n\n### Requirement: REQ-5.4 Factura\n\nEl sistema factura.\n\n#### Scenario: Factura emitida\n\n- **CUANDO** se cobra\n"
	const canonical = "# pagos\n\n## Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nTexto.\n\n#### Scenario: S\n\n- **CUANDO** x\n\n"
	tests := []struct {
		name      string
		canonical string
	}{
		{"un byte distinto en el bloque", canonical + strings.Replace(delta, "factura.", "factura!", 1)[len("## ADDED Requirements\n\n"):]},
		{"otro nombre", canonical + strings.Replace(delta, "Factura", "Facturas", 1)[len("## ADDED Requirements\n\n"):]},
		{"ausente de la canónica", canonical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := archiveCheckWorkspace(t, archiveCheckChange{
				name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": delta}, tasks: "- [x] 1.1 T\n", verify: archiveCheckVerify(), iso: true,
			})
			writeCLIFixture(t, filepath.Join(root, "openspec", "specs", "pagos", "spec.md"), tt.canonical)
			out, _ := runArchiveCheck(root, "demo", "--json")
			if !strings.Contains(out, `"alreadyComposed": false`) {
				t.Errorf("se esperaba alreadyComposed=false:\n%s", out)
			}
		})
	}
}
