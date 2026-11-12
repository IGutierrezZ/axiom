package livespec

import "path/filepath"

// ReportSchema es el esquema del informe de `axiom archive integrate --odd --json`.
const ReportSchema = "axiom.odd-integration/v1"

// Report es el informe de una integración ODD: lo que se integró, lo que se escribió y,
// cuando el proyecto declara contexto técnico, la comprobación estructural de REQ-30.6.
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
}

// CapabilityReport es el resultado de una capacidad en el informe.
type CapabilityReport struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	Outcome Outcome `json:"outcome"`
}

// ContextReport es la comprobación estructural del contexto técnico de la rama (REQ-30.6).
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
