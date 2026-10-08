package livingdoc

import "time"

// RequirementMeta describe un requerimiento canónico extraído de una spec viva.
type RequirementMeta struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Scenarios  []string `json:"scenarios"`
	LineNumber int      `json:"line_number"`
}

// LivingSpecEntry representa una especificación viva consolidada en <planning home>/specs/.
type LivingSpecEntry struct {
	Domain         string            `json:"domain"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	FilePath       string            `json:"file_path"`
	Requirements   []RequirementMeta `json:"requirements"`
	TotalScenarios int               `json:"total_scenarios"`
	LastModified   time.Time         `json:"last_modified"`
}

// LivingCatalog agrupa todas las especificaciones vivas consolidadas en el workspace.
type LivingCatalog struct {
	Specs             []LivingSpecEntry `json:"specs"`
	TotalRequirements int               `json:"total_requirements"`
	TotalScenarios    int               `json:"total_scenarios"`
	LastSync          time.Time         `json:"last_sync"`
	// ProjectName es el nombre del proyecto que lo contiene: workspace.name o, sin
	// él, el nombre de la carpeta maestra. Lo rellena el servicio y lo usa la
	// cabecera de INDEX.md.
	ProjectName string `json:"project_name,omitempty"`
	// Warnings reúne los avisos de lectura del catálogo, entre ellos el de la
	// ubicación antigua divergente (Q14, REQ-24.10).
	Warnings []string `json:"warnings,omitempty"`
}

// SyncReport resume el resultado determinista de una sincronización de índice de especificaciones vivas.
type SyncReport struct {
	SpecsCount        int       `json:"specs_count"`
	RequirementsCount int       `json:"requirements_count"`
	ScenariosCount    int       `json:"scenarios_count"`
	IndexPath         string    `json:"index_path"`
	SyncedAt          time.Time `json:"synced_at"`
	Warnings          []string  `json:"warnings,omitempty"`
}
