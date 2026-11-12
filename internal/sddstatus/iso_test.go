package sddstatus

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	bugReportOK  = "## Informe de bug\n\nObservado: el total omite el IVA reducido.\nEsperado: el total aplica el IVA de cada línea.\n"
	bugImpactOK  = "\n## Impacto en la spec\n\nSin impacto en la spec\nMotivo: el defecto estaba en el script de despliegue.\n"
	bugTasksOpen = "- [ ] 1.1 Corregir el cálculo\n"
)

// seedBugChange crea un cambio sin ningún fichero de specs: el bug sin delta
// de REQ-29.15 declara su informe en proposal.md.
func seedBugChange(t *testing.T, root, name, proposal, tasks string) string {
	t.Helper()
	changeRoot := filepath.Join(root, "openspec", "changes", name)
	write(t, filepath.Join(changeRoot, "proposal.md"), proposal)
	write(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
	write(t, filepath.Join(changeRoot, "tasks.md"), tasks)
	return changeRoot
}

func reasonsWith(reasons []string, needle string) []string {
	var found []string
	for _, reason := range reasons {
		if strings.Contains(reason, needle) {
			found = append(found, reason)
		}
	}
	return found
}

// REQ-29.15 (design D9): el artefacto de specs se cumple sin delta con un bug
// declarado; una declaración incompleta lo deja `missing` con un motivo por cada
// falta; con delta las secciones no se analizan; vale en un proyecto no ISO.
func TestISOBugProfile(t *testing.T) {
	tests := []struct {
		name          string
		proposal      string
		withDelta     bool
		wantSpecs     ArtifactState
		wantRoute     string
		wantTokens    []string // un motivo por cada token, en este orden
		wantInReasons []string // fragmentos que nombra algún motivo
	}{
		{name: "sin secciones el estado no cambia", proposal: "# Propuesta\n", wantSpecs: ArtifactMissing, wantRoute: "spec"},
		{name: "bug con requisito existente", proposal: "# Propuesta\n\n" + bugReportOK, wantSpecs: ArtifactDone, wantRoute: "apply"},
		{name: "bug sin impacto", proposal: "# Propuesta\n\n" + bugReportOK + bugImpactOK, wantSpecs: ArtifactDone, wantRoute: "apply"},
		{
			name:          "informe sin Esperado",
			proposal:      "## Informe de bug\n\nObservado: algo falla.\n",
			wantSpecs:     ArtifactMissing,
			wantRoute:     "spec",
			wantTokens:    []string{"bug_report_incomplete"},
			wantInReasons: []string{"`Esperado:`"},
		},
		{
			name:          "marca sin Motivo",
			proposal:      bugReportOK + "\n## Impacto en la spec\n\nSin impacto en la spec\n",
			wantSpecs:     ArtifactMissing,
			wantRoute:     "spec",
			wantTokens:    []string{"bug_no_impact_incomplete"},
			wantInReasons: []string{"`Motivo:`"},
		},
		{
			name:          "impacto sin informe",
			proposal:      bugImpactOK,
			wantSpecs:     ArtifactMissing,
			wantRoute:     "spec",
			wantTokens:    []string{"bug_report_incomplete"},
			wantInReasons: []string{"## Informe de bug"},
		},
		{
			name:       "faltan cosas de las dos secciones",
			proposal:   "## Informe de bug\n\nObservado: x\n\n## Impacto en la spec\n\nImpacto: ninguno\n",
			wantSpecs:  ArtifactMissing,
			wantRoute:  "spec",
			wantTokens: []string{"bug_report_incomplete", "bug_no_impact_incomplete", "bug_no_impact_incomplete"},
		},
		{name: "con delta las secciones se ignoran", proposal: bugReportOK + bugImpactOK, withDelta: true, wantSpecs: ArtifactDone, wantRoute: "apply"},
		{name: "con delta un informe incompleto no genera motivos", proposal: "## Informe de bug\n\nObservado: x\n", withDelta: true, wantSpecs: ArtifactDone, wantRoute: "apply"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			changeRoot := seedBugChange(t, root, "bug", tt.proposal, bugTasksOpen)
			if tt.withDelta {
				write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), "### Requirement: REQ-1.1 Cobro\n#### Scenario: Cobro\n")
			}
			status := resolveFrom(t, root, "bug")

			if got := status.Artifacts["specs"]; got != tt.wantSpecs {
				t.Errorf("artifacts.specs = %q, se esperaba %q", got, tt.wantSpecs)
			}
			if status.NextRecommended != tt.wantRoute {
				t.Errorf("nextRecommended = %q, se esperaba %q (motivos %v)", status.NextRecommended, tt.wantRoute, status.BlockedReasons)
			}
			var typed []string
			for _, reason := range status.BlockedReasons {
				if strings.HasPrefix(reason, "blocked(bug_") {
					typed = append(typed, reason[len("blocked("):strings.Index(reason, ")")])
				}
			}
			if strings.Join(typed, ",") != strings.Join(tt.wantTokens, ",") {
				t.Errorf("motivos tipados = %v, se esperaba %v (todos: %v)", typed, tt.wantTokens, status.BlockedReasons)
			}
			for _, fragment := range tt.wantInReasons {
				if len(reasonsWith(status.BlockedReasons, fragment)) == 0 {
					t.Errorf("ningún motivo nombra %q: %v", fragment, status.BlockedReasons)
				}
			}
			if tt.wantSpecs == ArtifactDone && status.Dependencies.Specs != DependencyAllDone {
				t.Errorf("dependencies.specs = %q, se esperaba all_done", status.Dependencies.Specs)
			}
			for _, reason := range status.BlockedReasons {
				if strings.Contains(reason, "blocked(iso_") {
					t.Errorf("fuera de ISO no debe aparecer ningún token iso_*: %q", reason)
				}
			}
		})
	}
}
