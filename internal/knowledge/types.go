package knowledge

import "time"

// AmbiguityItem representa una duda, inconsistencia o laguna técnica detectada durante el barrido.
type AmbiguityItem struct {
	Category    string `json:"category"` // "architecture", "dependency", "module", "orphan"
	Severity    string `json:"severity"` // "low", "medium", "high"
	Path        string `json:"path"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
}

// ModuleSummary describe un módulo funcional detectado.
type ModuleSummary struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Components  []string `json:"components,omitempty"`
}

// SweepOptions define los parámetros para el comando axiom knowledge sweep.
type SweepOptions struct {
	WorkspaceRoot string
	SpecsRoot     string
	Headless      bool
	Format        string // "text", "json"
}

// SweepResult contiene los resultados del barrido técnico y funcional.
type SweepResult struct {
	PrimaryLanguage string          `json:"primary_language"`
	Frameworks      []string        `json:"frameworks"`
	Entrypoints     []string        `json:"entrypoints"`
	Modules         []ModuleSummary `json:"modules"`
	Ambiguities     []AmbiguityItem `json:"ambiguities"`
	CreatedSpecs    []string        `json:"created_specs"`
	Duration        time.Duration   `json:"duration"`
}

// QueryType define la lente de análisis para la consulta.
type QueryType string

const (
	QueryTypeAuto       QueryType = "auto"
	QueryTypeTechnical  QueryType = "technical"
	QueryTypeFunctional QueryType = "functional"
)

// EvidenceItem detalla una evidencia de código que respalda la respuesta.
type EvidenceItem struct {
	Source  string `json:"source"` // "spec", "serena", "codegraph", "code"
	File    string `json:"file"`
	Lines   string `json:"lines,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
	Context string `json:"context,omitempty"`
}

// QueryOptions define los parámetros para el comando axiom knowledge query.
type QueryOptions struct {
	WorkspaceRoot string
	SpecsRoot     string
	Question      string
	Type          QueryType
	ForceDeep     bool
	Enrich        bool
}

// QueryResult detalla la respuesta estructurada y los efectos secundarios de auto-enriquecimiento.
type QueryResult struct {
	DirectAnswer     string         `json:"direct_answer"`
	Evidences        []EvidenceItem `json:"evidences"`
	ResolvedFromSpec bool           `json:"resolved_from_spec"`
	SpecUpdated      bool           `json:"spec_updated"`
	TargetSpecPath   string         `json:"target_spec_path,omitempty"`
}
