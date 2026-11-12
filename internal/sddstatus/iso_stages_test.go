package sddstatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	// Requisitos de una delta ADDED: uno correcto, uno sin identificador y uno sin
	// escenarios.
	deltaAddedOK          = "## ADDED Requirements\n\n### Requirement: REQ-30.1 Cobro con IVA\n\nEl sistema cobra con IVA.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n- **ENTONCES** incluye IVA\n"
	deltaAddedNoID        = "## ADDED Requirements\n\n### Requirement: Cobro con IVA\n\nEl sistema cobra con IVA.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n"
	deltaAddedNoScene     = "## ADDED Requirements\n\n### Requirement: REQ-30.1 Cobro con IVA\n\nEl sistema cobra con IVA.\n"
	deltaModifiedNoID     = "## MODIFIED Requirements\n\n### Requirement: Cobro antiguo\n\nTexto.\n\n#### Scenario: Cobro\n\n- **CUANDO** x\n"
	tasksPlain            = "- [ ] 1.1 Trabajo\n"
	tasksCoverRequirement = "- [ ] 1.1 Trabajo\n  Cubre: REQ-30.1\n"
	baselineReq51         = "# pagos\n\n## Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n"
	baselineReq51NoScen   = "# pagos\n\n## Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo.\n"
)

// isoProject monta un proyecto embebido con `sdd: {iso: true}` y sin ningún
// cambio.
func isoProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n")
	return ws
}

// seedISOChange crea un cambio ISO sellado en continuous con las deltas dadas
// (dominio a texto), el plan de tareas y, si no es nil, el informe de verify. Sin
// delta, proposal.md es el de un bug (proposal) y no hay carpeta specs/.
func seedISOChange(t *testing.T, ws workspacetest.Workspace, name, proposal string, deltas map[string]string, tasks, verify string) string {
	t.Helper()
	changeRoot := filepath.Join(ws.Master, "openspec", "changes", name)
	write(t, filepath.Join(changeRoot, "proposal.md"), proposal)
	write(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
	for domain, text := range deltas {
		write(t, filepath.Join(changeRoot, "specs", domain, "spec.md"), text)
	}
	if tasks != "" {
		write(t, filepath.Join(changeRoot, "tasks.md"), tasks)
	}
	if verify != "" {
		write(t, filepath.Join(changeRoot, "verify-report.md"), verify)
	}
	sealISOChange(t, changeRoot, name, &kickoff.SDDSnapshot{ISO: true})
	return changeRoot
}

func writeBaseline(t *testing.T, ws workspacetest.Workspace, domain, text string) {
	t.Helper()
	write(t, filepath.Join(ws.Master, "openspec", "specs", domain, "spec.md"), text)
}

// isoTokens devuelve los tokens `blocked(<token>)` de los motivos, en orden.
func isoTokens(reasons []string) []string {
	var tokens []string
	for _, reason := range reasons {
		if strings.HasPrefix(reason, "blocked(") {
			if end := strings.Index(reason, ")"); end > 0 {
				tokens = append(tokens, reason[len("blocked("):end])
			}
		}
	}
	return tokens
}

func wantTokens(t *testing.T, status Status, want ...string) {
	t.Helper()
	var got []string
	for _, token := range isoTokens(status.BlockedReasons) {
		if strings.HasPrefix(token, "iso_") {
			got = append(got, token)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("motivos iso_* = %v, se esperaba %v (todos: %v)", got, want, status.BlockedReasons)
	}
}

// REQ-29.3, REQ-29.10 y REQ-29.6 (design D6): los hallazgos de la etapa `specs`
// van a blockedReasons, estrechan `specs` y `coreReady`, y devuelven la ruta a
// `spec`; fuera de ISO no aparece ningún token iso_*.
func TestISOSpecsStage(t *testing.T) {
	tests := []struct {
		name       string
		deltas     map[string]string
		baseline   string
		tasks      string
		isoChange  bool
		wantTokens []string
		wantDetail []string
	}{
		{name: "delta correcta", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: tasksCoverRequirement, isoChange: true},
		{
			name: "ADDED sin identificador", deltas: map[string]string{"pagos": deltaAddedNoID}, tasks: tasksPlain, isoChange: true,
			wantTokens: []string{"iso_requirement_id"}, wantDetail: []string{"Cobro con IVA", "identificador REQ-<n>.<m>"},
		},
		{
			name: "ADDED sin escenarios", deltas: map[string]string{"pagos": deltaAddedNoScene}, tasks: tasksCoverRequirement, isoChange: true,
			wantTokens: []string{"iso_requirement_scenarios"}, wantDetail: []string{"pagos", "ADDED", "REQ-30.1"},
		},
		{
			name: "MODIFIED sin identificador propone el RENAMED", deltas: map[string]string{"pagos": deltaModifiedNoID}, tasks: tasksPlain, isoChange: true,
			wantTokens: []string{"iso_requirement_id"}, wantDetail: []string{"RENAMED"},
		},
		{
			name: "ADDED con un identificador que ya existe en la baseline", deltas: map[string]string{"cobros": strings.Replace(deltaAddedOK, "REQ-30.1 Cobro con IVA", "REQ-5.1 Otro recibo", 1)},
			baseline: baselineReq51, tasks: "- [ ] 1.1 Trabajo\n  Cubre: REQ-5.1\n", isoChange: true,
			wantTokens: []string{"iso_requirement_id"}, wantDetail: []string{"REQ-5.1", "baseline"},
		},
		{
			name: "varios bloqueos concurrentes se proyectan todos", deltas: map[string]string{"pagos": deltaAddedNoID, "cobros": deltaAddedNoScene}, tasks: tasksCoverRequirement, isoChange: true,
			wantTokens: []string{"iso_requirement_id", "iso_requirement_scenarios"},
		},
		{name: "fuera de ISO no se proyecta nada", deltas: map[string]string{"pagos": deltaAddedNoID}, tasks: tasksCoverRequirement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := isoProject(t)
			if tt.baseline != "" {
				writeBaseline(t, ws, "pagos", tt.baseline)
			}
			changeRoot := seedISOChange(t, ws, "demo", "# Propuesta\n", tt.deltas, tt.tasks, "")
			if !tt.isoChange {
				if err := os.Remove(filepath.Join(changeRoot, kickoff.KickoffFileName)); err != nil {
					t.Fatal(err)
				}
				sealISOChange(t, changeRoot, "demo", nil)
			}
			status := resolveFrom(t, ws.Master, "demo")

			wantTokens(t, status, tt.wantTokens...)
			if len(tt.wantTokens) == 0 {
				if status.NextRecommended != "apply" || status.Dependencies.Specs != DependencyAllDone {
					t.Errorf("ruta %q, specs %q, se esperaba apply y all_done", status.NextRecommended, status.Dependencies.Specs)
				}
				return
			}
			if status.NextRecommended != "spec" {
				t.Errorf("nextRecommended = %q, se esperaba spec", status.NextRecommended)
			}
			if status.Dependencies.Specs != DependencyBlocked || status.ApplyState != ApplyBlocked {
				t.Errorf("specs %q y applyState %q, se esperaba blocked y blocked", status.Dependencies.Specs, status.ApplyState)
			}
			if status.Artifacts["specs"] != ArtifactDone {
				t.Errorf("artifacts.specs = %q: los motivos no mienten sobre los ficheros", status.Artifacts["specs"])
			}
			joined := strings.Join(status.BlockedReasons, "\n")
			for _, fragment := range tt.wantDetail {
				if !strings.Contains(joined, fragment) {
					t.Errorf("ningún motivo nombra %q: %v", fragment, status.BlockedReasons)
				}
			}
		})
	}
}

// REQ-29.3 enmendado (D17): un archivado interrumpido tras componer un ADDED con
// el compositor real no proyecta iso_requirement_id ni devuelve la ruta a `spec`;
// con otro bloque o otro nombre sigue bloqueando.
func TestISOInterruptedArchive(t *testing.T) {
	canonical := baselineReq51
	delta := deltaAddedOK
	composed, err := ComposeOpenSpecCanonicalSpec(canonical, delta)
	if err != nil {
		t.Fatalf("ComposeOpenSpecCanonicalSpec: %v", err)
	}

	tests := []struct {
		name       string
		baseline   string
		wantTokens []string
	}{
		{name: "antes de componer", baseline: canonical},
		{name: "tras componer con el compositor real", baseline: composed},
		{
			name:       "con el mismo identificador y otro bloque",
			baseline:   strings.Replace(composed, "El sistema cobra con IVA.", "El sistema cobra sin IVA.", 1),
			wantTokens: []string{"iso_requirement_id"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := isoProject(t)
			writeBaseline(t, ws, "pagos", tt.baseline)
			seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": delta}, "- [ ] 1.1 Trabajo\n  Cubre: REQ-30.1\n", "")
			status := resolveFrom(t, ws.Master, "demo")
			wantTokens(t, status, tt.wantTokens...)
			wantRoute := "apply"
			if len(tt.wantTokens) > 0 {
				wantRoute = "spec"
			}
			if status.NextRecommended != wantRoute {
				t.Errorf("nextRecommended = %q, se esperaba %q", status.NextRecommended, wantRoute)
			}
		})
	}
}

// REQ-29.4, REQ-29.5, REQ-29.19 y REQ-29.16: los hallazgos de la etapa `tasks`
// (cobertura, bug con requisito existente y bug sin impacto) van a
// blockedReasons, estrechan `tasks` y devuelven la ruta a `tasks`, un motivo por
// hallazgo.
func TestISOTasksStage(t *testing.T) {
	tests := []struct {
		name       string
		proposal   string
		deltas     map[string]string
		baseline   string
		tasks      string
		wantTokens []string
		wantDetail []string
	}{
		{name: "delta cubierta", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: "- [ ] 1.1 T\n  Cubre: REQ-30.1\n"},
		{
			name: "requisito sin cobertura", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK}, tasks: "- [ ] 1.1 T\n",
			wantTokens: []string{"iso_coverage"}, wantDetail: []string{"REQ-30.1", "Cubre: REQ-30.1"},
		},
		{
			name: "Cubre a un requisito inexistente", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK},
			tasks:      "- [ ] 1.1 T\n  Cubre: REQ-30.1, REQ-99.9\n",
			wantTokens: []string{"iso_cubre_unknown"}, wantDetail: []string{"tasks.md:2", "REQ-99.9"},
		},
		{
			name: "cobertura y desconocido a la vez", proposal: "# P\n", deltas: map[string]string{"pagos": deltaAddedOK},
			tasks:      "- [ ] 1.1 T\n  Cubre: REQ-99.9\n",
			wantTokens: []string{"iso_coverage", "iso_cubre_unknown"},
		},
		{name: "bug con requisito existente citado", proposal: bugReportOK, baseline: baselineReq51, tasks: "- [ ] 1.1 T\n  Cubre: REQ-5.1\n"},
		{
			name: "bug con requisito existente sin citar", proposal: bugReportOK, baseline: baselineReq51, tasks: "- [ ] 1.1 T\n",
			wantTokens: []string{"iso_bug_requirement"}, wantDetail: []string{"ningún requisito"},
		},
		{
			name: "bug con un requisito existente sin escenarios", proposal: bugReportOK, baseline: baselineReq51NoScen, tasks: "- [ ] 1.1 T\n  Cubre: REQ-5.1\n",
			wantTokens: []string{"iso_bug_requirement"}, wantDetail: []string{"REQ-5.1", "cambio normal con delta"},
		},
		{
			name: "bug con un requisito inexistente", proposal: bugReportOK, baseline: baselineReq51, tasks: "- [ ] 1.1 T\n  Cubre: REQ-9.9\n",
			wantTokens: []string{"iso_cubre_unknown"},
		},
		{name: "bug sin impacto y sin Cubre", proposal: bugReportOK + bugImpactOK, tasks: "- [ ] 1.1 T\n"},
		{
			name: "bug sin impacto con Cubre", proposal: bugReportOK + bugImpactOK, baseline: baselineReq51, tasks: "- [ ] 1.1 T\n  Cubre: REQ-5.1\n",
			wantTokens: []string{"iso_bug_contradiction"}, wantDetail: []string{"tasks.md:2", "REQ-5.1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := isoProject(t)
			if tt.baseline != "" {
				writeBaseline(t, ws, "pagos", tt.baseline)
			}
			seedISOChange(t, ws, "demo", tt.proposal, tt.deltas, tt.tasks, "")
			status := resolveFrom(t, ws.Master, "demo")

			wantTokens(t, status, tt.wantTokens...)
			if len(tt.wantTokens) == 0 {
				if status.NextRecommended != "apply" || status.Dependencies.Tasks != DependencyAllDone {
					t.Errorf("ruta %q, tasks %q, se esperaba apply y all_done", status.NextRecommended, status.Dependencies.Tasks)
				}
				return
			}
			if status.NextRecommended != "tasks" {
				t.Errorf("nextRecommended = %q, se esperaba tasks", status.NextRecommended)
			}
			if status.Dependencies.Tasks != DependencyBlocked || status.ApplyState != ApplyBlocked {
				t.Errorf("tasks %q y applyState %q, se esperaba blocked y blocked", status.Dependencies.Tasks, status.ApplyState)
			}
			joined := strings.Join(status.BlockedReasons, "\n")
			for _, fragment := range tt.wantDetail {
				if !strings.Contains(joined, fragment) {
					t.Errorf("ningún motivo nombra %q: %v", fragment, status.BlockedReasons)
				}
			}
		})
	}

	t.Run("sin el fichero de tareas no se evalúa la cobertura", func(t *testing.T) {
		ws := isoProject(t)
		seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedOK}, "", "")
		status := resolveFrom(t, ws.Master, "demo")
		wantTokens(t, status)
		if status.NextRecommended != "tasks" {
			t.Errorf("nextRecommended = %q, se esperaba tasks (falta el fichero)", status.NextRecommended)
		}
	})

	t.Run("la cobertura es la unión de los ficheros de tareas de cada rol", func(t *testing.T) {
		ws := isoProject(t)
		changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
		write(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
		write(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
		two := deltaAddedOK + "\n### Requirement: REQ-30.2 Recibo\n\nEmite recibo.\n\n#### Scenario: Recibo\n\n- **CUANDO** x\n"
		write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), two)
		write(t, filepath.Join(changeRoot, "tasks.backend.md"), "- [ ] 1.1 API\n  Cubre: REQ-30.1\n")
		sealISORoles(t, changeRoot, "demo", "backend", "frontend")

		// Falta el plan del segundo rol: la cobertura aún no se evalúa.
		status := resolveFrom(t, ws.Master, "demo")
		wantTokens(t, status)

		write(t, filepath.Join(changeRoot, "tasks.frontend.md"), "- [ ] 1.1 Vista\n")
		status = resolveFrom(t, ws.Master, "demo")
		wantTokens(t, status, "iso_coverage")
		if !strings.Contains(strings.Join(status.BlockedReasons, "\n"), "REQ-30.2") {
			t.Errorf("el motivo debe nombrar REQ-30.2: %v", status.BlockedReasons)
		}

		write(t, filepath.Join(changeRoot, "tasks.frontend.md"), "- [ ] 1.1 Vista\n  Cubre: REQ-30.2\n")
		status = resolveFrom(t, ws.Master, "demo")
		wantTokens(t, status)
	})
}

// sealISORoles sella un cambio ISO continuo con un plan por rol.
func sealISORoles(t *testing.T, changeRoot, name string, roles ...string) {
	t.Helper()
	var assigned []kickoff.KickoffRole
	for _, role := range roles {
		assigned = append(assigned, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks." + role + ".md", VerifyFile: "verify-report.md"})
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone, Roles: assigned,
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       &kickoff.SDDSnapshot{ISO: true},
	})
	if err != nil || !ok {
		t.Fatalf("sealISORoles: kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// isoBranchMonorepo monta un monorepo embebido con un cambio ISO sellado con la
// instantánea de git, con la delta sin identificador (un hallazgo de la etapa
// specs) y el HEAD en la rama dada.
func isoBranchMonorepo(t *testing.T, head string) string {
	t.Helper()
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n")
	switchTo(t, ws.Master, head)
	changeRoot := filepath.Join(ws.Master, "openspec", "changes", branchChange)
	write(t, filepath.Join(changeRoot, "proposal.md"), "# P\n")
	write(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
	write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), deltaAddedNoID)
	write(t, filepath.Join(changeRoot, "tasks.md"), tasksPlain)
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: branchChange, SealedBy: "prueba",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git:       monorepoSnapshot(filepath.Base(ws.Master)),
		SDD:       &kickoff.SDDSnapshot{ISO: true},
	})
	if err != nil || !ok {
		t.Fatalf("sellar: %v (publicado: %v)", err, ok)
	}
	return ws.Master
}

// REQ-29.16: los motivos ISO se añaden después de los existentes y respetan la
// precedencia; con la rama equivocada `wrong_branch` va el primero y el motivo ISO
// se proyecta además.
func TestISOReasonsOrdering(t *testing.T) {
	t.Run("wrong_branch va delante del motivo ISO", func(t *testing.T) {
		root := isoBranchMonorepo(t, "main")
		status := resolveFrom(t, root, branchChange)

		if len(status.BlockedReasons) < 2 || !strings.HasPrefix(status.BlockedReasons[0], wrongBranchPrefix) {
			t.Fatalf("blockedReasons = %v, se esperaba blocked(wrong_branch) en primera posición", status.BlockedReasons)
		}
		wantTokens(t, status, "iso_requirement_id")
		if status.NextRecommended != "resolve-blockers" {
			t.Errorf("nextRecommended = %q, con la rama equivocada la ruta es resolve-blockers como hoy", status.NextRecommended)
		}
		last := status.BlockedReasons[len(status.BlockedReasons)-1]
		if !strings.HasPrefix(last, "blocked(iso_requirement_id)") {
			t.Errorf("el motivo ISO debe ir tras los existentes: %v", status.BlockedReasons)
		}
	})

	t.Run("en la rama del cambio solo queda el motivo ISO", func(t *testing.T) {
		root := isoBranchMonorepo(t, "feat/x")
		status := resolveFrom(t, root, branchChange)
		wantNotWrongBranch(t, status)
		wantTokens(t, status, "iso_requirement_id")
		if status.NextRecommended != "spec" {
			t.Errorf("nextRecommended = %q, se esperaba spec", status.NextRecommended)
		}
	})

	t.Run("varios hallazgos de etapas distintas se proyectan todos en orden", func(t *testing.T) {
		ws := isoProject(t)
		seedISOChange(t, ws, "demo", "# P\n", map[string]string{"pagos": deltaAddedNoID + "\n### Requirement: REQ-30.3 Sin escenarios\n\nTexto.\n"}, "- [ ] 1.1 T\n  Cubre: REQ-99.9\n", "")
		status := resolveFrom(t, ws.Master, "demo")
		wantTokens(t, status, "iso_requirement_id", "iso_requirement_scenarios", "iso_coverage", "iso_cubre_unknown")
		if status.NextRecommended != "spec" {
			t.Errorf("nextRecommended = %q, con hallazgos de specs y de tasks la ruta vuelve a la fase más temprana", status.NextRecommended)
		}
	})
}
