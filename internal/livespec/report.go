package livespec

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
)

// ReportSchema es el esquema del informe de `axiom archive integrate --odd --json`.
const ReportSchema = "axiom.odd-integration/v1"

// Report es el informe de una integración ODD: lo que se integró, lo que se escribió y,
// cuando el proyecto declara contexto técnico, la comprobación estructural de RF-OLS-06.
type Report struct {
	Schema  string `json:"schema"`
	Feature string `json:"feature"`
	Kind    Kind   `json:"kind"`
	// Impact es delta, no_impact o absent (ImpactDelta, ImpactNone, ImpactAbsent).
	Impact string `json:"impact"`
	// Reason es el motivo del marcador cuando Impact es no_impact.
	Reason       string             `json:"reason,omitempty"`
	Capabilities []CapabilityReport `json:"capabilities"`
	// Context solo está si el proyecto declara `technical_context`.
	Context *ContextReport `json:"context,omitempty"`
	// Written son los ficheros que la orden escribió, relativos a la raíz del workspace y con `/`.
	Written            []string `json:"written"`
	NothingToIntegrate bool     `json:"nothingToIntegrate"`
	// Check es verdadero con `--check`: la integración se planificó sin escribir (RF-OLS-10).
	Check bool `json:"check,omitempty"`
	// Requirements solo está con `--check`: lo que las deltas tocan y crean.
	Requirements *openspecdoc.Declaration `json:"requirements,omitempty"`
	// Assignments es el mapa de provisional a definitivo, con la capacidad y el estado de
	// cada entrada (RF-RID-14); no está si el documento no tiene provisionales.
	Assignments []AssignmentReport `json:"assignments,omitempty"`
	// Duplicates son los identificadores repetidos en la spec viva resultante (RF-RID-13).
	Duplicates []Duplicate `json:"duplicates,omitempty"`
}

// CapabilityReport es el resultado de una capacidad en el informe.
type CapabilityReport struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	Outcome Outcome `json:"outcome"`
}

// AssignmentReport es una entrada del mapa de provisional a definitivo en un informe.
type AssignmentReport struct {
	Provisional string   `json:"provisional"`
	Definitive  string   `json:"definitive"`
	Capability  string   `json:"capability"`
	Requirement string   `json:"requirement"`
	State       string   `json:"state"`
	Previous    []string `json:"previous,omitempty"`
}

// AssignmentReports proyecta las entradas planificadas al formato de los informes.
func AssignmentReports(entries []PlannedEntry) []AssignmentReport {
	var out []AssignmentReport
	for _, e := range entries {
		out = append(out, AssignmentReport{
			Provisional: e.Provisional, Definitive: e.Definitive, Capability: e.Capability,
			Requirement: e.Requirement, State: e.State, Previous: e.Previous,
		})
	}
	return out
}

// ContextReport es la comprobación estructural del contexto técnico de la rama (RF-OLS-06).
type ContextReport struct {
	Declared          bool     `json:"declared"`
	MaintainedTouched []string `json:"maintainedTouched"`
	ReferenceTouched  []string `json:"referenceTouched"`
}

// Report construye el informe de la integración; written son las rutas absolutas que
// Apply devolvió (nil si no se aplicó nada).
func (i Integration) Report(written []string) Report {
	report := Report{
		Schema:             ReportSchema,
		Feature:            i.Feature,
		Kind:               i.Kind,
		Impact:             i.Impact,
		Reason:             i.Reason,
		Capabilities:       []CapabilityReport{},
		Context:            i.Context,
		Written:            []string{},
		NothingToIntegrate: i.NothingToIntegrate(),
		Assignments:        AssignmentReports(i.Assignments),
		Duplicates:         i.Duplicates,
	}
	if i.Check {
		requirements := i.Requirements
		if requirements.Modifies == nil {
			requirements.Modifies = []openspecdoc.DeclaredRequirement{}
		}
		if requirements.Creates == nil {
			requirements.Creates = []openspecdoc.DeclaredRequirement{}
		}
		report.Check, report.Requirements = true, &requirements
	}
	for _, c := range i.Capabilities {
		report.Capabilities = append(report.Capabilities, CapabilityReport{Name: c.Name, Path: i.relative(c.Path), Outcome: c.Outcome})
	}
	for _, path := range written {
		report.Written = append(report.Written, i.relative(path))
	}
	return report
}

// relative da la ruta relativa a la raíz del workspace con `/`.
func (i Integration) relative(path string) string {
	if i.loc.WorkspaceRoot == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(i.loc.WorkspaceRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
