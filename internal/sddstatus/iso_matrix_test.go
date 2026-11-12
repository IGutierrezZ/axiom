package sddstatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	matrixTableHeader = "| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |\n|---|---|---|---|---|---|---|---|---|---|---|\n"
	rowPass           = "| REQ-30.1 | Cobro general | unitario | internal/pagos/cobro_test.go:TestCobraIVA | go test ./internal/pagos -run TestCobraIVA -count=1 | 0 | | | | pass | |\n"
	rowFail           = "| REQ-30.1 | Cobro general | unitario | internal/pagos/cobro_test.go:TestCobraIVA | go test ./internal/pagos -run TestCobraIVA -count=1 | 1 | | | | fail | |\n"
	rowManual         = "| REQ-30.1 | Cobro general | manual | acta-12 | | | Ana Pérez | 2026-11-12 | La impresora fiscal no tiene emulador | pass | |\n"
	rowShortcut       = "| REQ-30.1 | Cobro general | unitario | TestCobraIVA | go test ./internal/pagos -run TestCobraIVA -count=1 | 0 | | | | pass | |\n"
	rowUnresolved     = "| REQ-30.1 | Cobro general | unitario | internal/pagos/no_existe_test.go:TestX | go test ./... | 0 | | | | pass | |\n"
	rowExtra          = "| REQ-30.1 | Escenario inexistente | manual | acta-13 | | | Ana Pérez | 2026-11-12 | No se automatiza | pass | |\n"
	tasksCoveredDone  = "- [x] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
	tasksCoveredOpen  = "- [ ] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
)

func verifyReportWith(rows ...string) string {
	return "# Informe de verificación\n\n## Cumplimiento técnico\n\nTodo conforme.\n\n## Matriz de verificación\n\n" + matrixTableHeader + strings.Join(rows, "")
}

// seedPagosTest crea la prueba a la que apuntan las referencias de la matriz.
func seedPagosTest(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	write(t, filepath.Join(ws.Master, "internal", "pagos", "cobro_test.go"),
		"package pagos\n\nimport \"testing\"\n\nfunc TestCobraIVA(t *testing.T) {}\n")
}

func archiveTokens(findings []isoevidence.Finding) []string {
	tokens := make([]string, 0, len(findings))
	for _, finding := range findings {
		tokens = append(tokens, string(finding.Code))
	}
	return tokens
}

// REQ-29.10, REQ-29.11, REQ-29.6 a REQ-29.8 y REQ-29.19: la regla del archivado
// ISO, compartida por el status, archive-check y archive-compose.
func TestISOArchiveRefusal(t *testing.T) {
	tests := []struct {
		name     string
		proposal string
		deltas   map[string]string
		baseline string
		tasks    string
		verify   string
		want     []string
		detail   string
	}{
		{name: "matriz válida con fila automática", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone, verify: verifyReportWith(rowPass)},
		{name: "atajo TestX resuelve en el árbol", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone, verify: verifyReportWith(rowShortcut)},
		{name: "matriz válida con fila manual", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone, verify: verifyReportWith(rowManual)},
		{
			name: "matriz ausente", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			want: []string{"iso_matrix_missing"},
		},
		{
			name: "informe libre sin matriz", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			verify: "# Verificación\n\nTodo bien.\n", want: []string{"iso_matrix_missing"},
		},
		{
			name: "escenario sin fila", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			verify: verifyReportWith(), want: []string{"iso_matrix_invalid"}, detail: "Cobro general",
		},
		{
			name: "fila de un escenario que no figura", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			verify: verifyReportWith(rowPass, rowExtra), want: []string{"iso_matrix_invalid"}, detail: "Escenario inexistente",
		},
		{
			name: "referencia que no resuelve", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			verify: verifyReportWith(rowUnresolved), want: []string{"iso_matrix_invalid"}, detail: "no_existe_test.go",
		},
		{
			name: "fila fail", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoveredDone,
			verify: verifyReportWith(rowFail), want: []string{"iso_matrix_failed"},
		},
		{
			name: "ADDED sin escenarios", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedNoScene}, tasks: tasksCoveredDone,
			verify: verifyReportWith(), want: []string{"iso_requirement_scenarios"}, detail: "pagos",
		},
		{
			name: "no evalúa REQ-29.3", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedNoID}, tasks: tasksPlain,
			verify: verifyReportWith(), want: nil,
		},
		{name: "bug sin impacto sin matriz", proposal: bugReportOK + bugImpactOK, tasks: tasksPlain},
		{
			name: "bug sin impacto con fila", proposal: bugReportOK + bugImpactOK, tasks: tasksPlain, verify: verifyReportWith(rowManual),
			want: []string{"iso_matrix_invalid"},
		},
		{
			name: "bug con requisito existente sin matriz", proposal: bugReportOK, baseline: baselineReq51, tasks: "- [x] 1.1 T\n  Cubre: REQ-5.1\n",
			want: []string{"iso_matrix_missing"},
		},
		{
			name: "bug con requisito existente y su fila", proposal: bugReportOK, baseline: baselineReq51, tasks: "- [x] 1.1 T\n  Cubre: REQ-5.1\n",
			verify: verifyReportWith("| REQ-5.1 | Recibo emitido | manual | acta-1 | | | Ana Pérez | 2026-11-12 | Impresora | pass | |\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := isoProject(t)
			seedPagosTest(t, ws)
			if tt.baseline != "" {
				writeBaseline(t, ws, "pagos", tt.baseline)
			}
			changeRoot := seedISOChange(t, ws, "demo", tt.proposal, tt.deltas, tt.tasks, tt.verify)

			findings := ISOArchiveRefusal(ws.Master, changeRoot)
			if got := strings.Join(archiveTokens(findings), ","); got != strings.Join(tt.want, ",") {
				t.Fatalf("hallazgos = %v, se esperaba %v", got, tt.want)
			}
			if tt.detail != "" {
				var all []string
				for _, finding := range findings {
					all = append(all, finding.Reason())
				}
				if !strings.Contains(strings.Join(all, "\n"), tt.detail) {
					t.Errorf("ningún motivo nombra %q: %v", tt.detail, all)
				}
			}
		})
	}

	t.Run("un cambio no ISO no tiene hallazgos", func(t *testing.T) {
		ws := isoProject(t)
		changeRoot := seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedNoScene}, tasksCoveredDone, "")
		sealRewrite(t, changeRoot, "demo")
		if findings := ISOArchiveRefusal(ws.Master, changeRoot); len(findings) != 0 {
			t.Fatalf("hallazgos = %v, un cambio no ISO no tiene ninguno", archiveTokens(findings))
		}
	})

	t.Run("sin kickoff no hay hallazgos", func(t *testing.T) {
		ws := isoProject(t)
		changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
		write(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
		if findings := ISOArchiveRefusal(ws.Master, changeRoot); len(findings) != 0 {
			t.Fatalf("hallazgos = %v", archiveTokens(findings))
		}
	})

	t.Run("una declaración inválida se suma a la matriz", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: quizás\n")
		changeRoot := seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedOK}, tasksCoveredDone, "")
		findings := ISOArchiveRefusal(ws.Master, changeRoot)
		if got := strings.Join(archiveTokens(findings), ","); got != "iso_declaration_invalid,iso_matrix_missing" {
			t.Fatalf("hallazgos = %v", got)
		}
	})
}

// sealRewrite vuelve a sellar el cambio como no ISO.
func sealRewrite(t *testing.T, changeRoot, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(changeRoot, kickoff.KickoffFileName)); err != nil {
		t.Fatal(err)
	}
	sealISOChange(t, changeRoot, name, nil)
}

// REQ-29.11 y D12: con tareas pendientes el archivado explícito de trabajo
// incompleto sigue exigiendo la matriz: `archive` queda bloqueado, el motivo va
// solo a notes y la ruta sigue siendo apply.
func TestISOArchiveWithPendingTasks(t *testing.T) {
	ws := isoProject(t)
	seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedOK}, tasksCoveredOpen, "")
	status := resolveFrom(t, ws.Master, "demo")

	if status.NextRecommended != "apply" || status.ApplyState != ApplyReady {
		t.Fatalf("ruta %q y applyState %q, se esperaba apply y ready (D12)", status.NextRecommended, status.ApplyState)
	}
	if status.Dependencies.Archive != DependencyBlocked {
		t.Errorf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
	}
	if len(status.BlockedReasons) != 0 {
		t.Errorf("blockedReasons = %v, apply no admite motivos", status.BlockedReasons)
	}
	if len(reasonsWith(status.Notes, "blocked(iso_matrix_missing):")) != 1 {
		t.Errorf("notes = %v, se esperaba el motivo de la matriz ausente", status.Notes)
	}
	if got := isoOf(t, status); got == nil || got.Matrix != nil {
		t.Errorf("iso = %+v, la matriz solo se proyecta con apply terminado", got)
	}
}

// sealISOCheckpointed sella un cambio ISO en checkpointed con el rol fullstack.
func sealISOCheckpointed(t *testing.T, changeRoot, name string) {
	t.Helper()
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       &kickoff.SDDSnapshot{ISO: true},
	})
	if err != nil || !ok {
		t.Fatalf("sealISOCheckpointed: kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// REQ-29.11 y D19: con apply terminado y la matriz ausente o inválida la ruta es
// `verify`; con filas `fail`, resolve-blockers; con una matriz de éxito, archive.
// `iso.matrix` lo proyecta en cada caso.
func TestISOVerifyRoute(t *testing.T) {
	tests := []struct {
		name        string
		proposal    string
		deltas      map[string]string
		verify      string
		wantRoute   string
		wantToken   string
		wantMatrix  ISOMatrixProjection
		wantArchive DependencyState
	}{
		{
			name: "matriz ausente", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK},
			wantRoute: "verify", wantToken: "iso_matrix_missing", wantMatrix: ISOMatrixProjection{State: "missing", Findings: 1}, wantArchive: DependencyBlocked,
		},
		{
			name: "matriz inválida", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, verify: verifyReportWith(),
			wantRoute: "verify", wantToken: "iso_matrix_invalid", wantMatrix: ISOMatrixProjection{State: "invalid", Findings: 1}, wantArchive: DependencyBlocked,
		},
		{
			name: "filas fail", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, verify: verifyReportWith(rowFail),
			wantRoute: "resolve-blockers", wantToken: "iso_matrix_failed", wantMatrix: ISOMatrixProjection{State: "valid", Findings: 1}, wantArchive: DependencyBlocked,
		},
		{
			name: "matriz de éxito", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, verify: verifyReportWith(rowPass),
			wantRoute: "archive", wantMatrix: ISOMatrixProjection{State: "valid", Successful: true}, wantArchive: DependencyReady,
		},
		{
			name: "bug sin impacto sin matriz", proposal: bugReportOK + bugImpactOK,
			wantRoute: "archive", wantMatrix: ISOMatrixProjection{State: "not_required", Successful: true}, wantArchive: DependencyReady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := isoProject(t)
			seedPagosTest(t, ws)
			tasks := tasksCoveredDone
			if tt.deltas == nil {
				tasks = "- [x] 1.1 Trabajo\n"
			}
			seedISOChange(t, ws, "demo", tt.proposal, tt.deltas, tasks, tt.verify)
			status := resolveFrom(t, ws.Master, "demo")

			if status.ApplyState != ApplyAllDone {
				t.Fatalf("applyState = %q, se esperaba all_done", status.ApplyState)
			}
			if status.NextRecommended != tt.wantRoute {
				t.Errorf("nextRecommended = %q, se esperaba %q (motivos %v)", status.NextRecommended, tt.wantRoute, status.BlockedReasons)
			}
			if status.Dependencies.Archive != tt.wantArchive {
				t.Errorf("dependencies.archive = %q, se esperaba %q", status.Dependencies.Archive, tt.wantArchive)
			}
			if tt.wantToken == "" {
				wantTokens(t, status)
			} else {
				wantTokens(t, status, tt.wantToken)
				if len(reasonsWith(status.Notes, "blocked("+tt.wantToken+"):")) != 1 {
					t.Errorf("notes = %v, se esperaba el motivo también en notes", status.Notes)
				}
			}
			got := isoOf(t, status)
			if got == nil || got.Matrix == nil || *got.Matrix != tt.wantMatrix {
				t.Fatalf("iso.matrix = %+v, se esperaba %+v", got, tt.wantMatrix)
			}
		})
	}
}

// D19: la ruta `verify` sustituye a await-gate solo cuando la compuerta abierta
// es `integration`; las demás compuertas conservan su precedencia.
func TestISOVerifyRouteGatePrecedence(t *testing.T) {
	seed := func(t *testing.T, verify string, approved ...kickoff.GateKey) (workspacetest.Workspace, string) {
		t.Helper()
		ws := isoProject(t)
		seedPagosTest(t, ws)
		changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
		write(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
		write(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
		write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), deltaAddedOK)
		write(t, filepath.Join(changeRoot, "tasks.md"), tasksCoveredDone)
		if verify != "" {
			write(t, filepath.Join(changeRoot, "verify-report.md"), verify)
		}
		sealISOCheckpointed(t, changeRoot, "demo")
		approveGates(t, changeRoot, approved...)
		return ws, changeRoot
	}
	allButIntegration := []kickoff.GateKey{kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks, kickoff.RoleApplyGate("fullstack")}

	t.Run("integration abierta con la matriz ausente rutea a verify", func(t *testing.T) {
		ws, _ := seed(t, "", allButIntegration...)
		status := resolveFrom(t, ws.Master, "demo")
		if status.NextRecommended != "verify" {
			t.Fatalf("nextRecommended = %q, se esperaba verify (motivos %v)", status.NextRecommended, status.BlockedReasons)
		}
		if status.GateQuestion != nil {
			t.Errorf("sin await-gate no hay pregunta de compuerta: %+v", status.GateQuestion)
		}
		wantTokens(t, status, "iso_matrix_missing")
	})

	t.Run("integration abierta con la matriz de éxito sigue en await-gate", func(t *testing.T) {
		ws, _ := seed(t, verifyReportWith(rowPass), allButIntegration...)
		status := resolveFrom(t, ws.Master, "demo")
		if status.NextRecommended != "await-gate" {
			t.Fatalf("nextRecommended = %q, se esperaba await-gate", status.NextRecommended)
		}
		wantTokens(t, status)
	})

	t.Run("integration abierta con filas fail rutea a resolve-blockers", func(t *testing.T) {
		ws, _ := seed(t, verifyReportWith(rowFail), allButIntegration...)
		status := resolveFrom(t, ws.Master, "demo")
		if status.NextRecommended != "resolve-blockers" {
			t.Fatalf("nextRecommended = %q, se esperaba resolve-blockers", status.NextRecommended)
		}
	})

	t.Run("otra compuerta abierta conserva su precedencia", func(t *testing.T) {
		ws, _ := seed(t, "", kickoff.GateSpec, kickoff.GateTasks, kickoff.RoleApplyGate("fullstack"))
		status := resolveFrom(t, ws.Master, "demo")
		if status.NextRecommended != "await-gate" {
			t.Fatalf("nextRecommended = %q, la compuerta design conserva su precedencia", status.NextRecommended)
		}
		if gate := gateStateOf(t, status, kickoff.GateDesign); gate.Status != "pending" {
			t.Fatalf("la compuerta design = %+v", gate)
		}
	})

	t.Run("con integration aprobada la matriz ausente sigue rutando a verify", func(t *testing.T) {
		ws, _ := seed(t, "", append(allButIntegration, kickoff.GateIntegration)...)
		status := resolveFrom(t, ws.Master, "demo")
		if status.NextRecommended != "verify" {
			t.Fatalf("nextRecommended = %q, se esperaba verify", status.NextRecommended)
		}
	})
}

// REQ-29.16 y REQ-29.2: iso.matrix se proyecta igual en v1 y v2, y solo en un
// cambio ISO con apply terminado.
func TestISOMatrixProjection(t *testing.T) {
	t.Run("v1 y v2 coinciden", func(t *testing.T) {
		ws := isoProject(t)
		seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedOK}, tasksCoveredDone, "")
		status := resolveFrom(t, ws.Master, "demo")
		if status.ISO == nil || status.ISO.Matrix == nil || status.ISO.Matrix.State != "missing" {
			t.Fatalf("status.ISO = %+v", status.ISO)
		}
		if got := isoOf(t, status); got == nil || got.Matrix == nil || *got.Matrix != *status.ISO.Matrix {
			t.Fatalf("v2 = %+v, v1 = %+v", got, status.ISO)
		}
	})

	t.Run("un cambio no ISO no proyecta matriz ni la evalúa", func(t *testing.T) {
		ws := isoProject(t)
		changeRoot := seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedOK}, tasksCoveredDone, "# Verificación libre\n")
		sealRewrite(t, changeRoot, "demo")
		status := resolveFrom(t, ws.Master, "demo")
		if got := isoOf(t, status); got == nil || got.Matrix != nil {
			t.Fatalf("iso = %+v, un cambio no ISO no proyecta matriz", got)
		}
		if status.NextRecommended != "archive" || status.Dependencies.Archive != DependencyReady {
			t.Errorf("ruta %q y archive %q: un cambio no ISO archiva con un verify libre", status.NextRecommended, status.Dependencies.Archive)
		}
		for _, reason := range append(append([]string{}, status.BlockedReasons...), status.Notes...) {
			if strings.Contains(reason, "iso_") {
				t.Errorf("fuera de ISO no debe aparecer ningún token iso_*: %q", reason)
			}
		}
	})
}
