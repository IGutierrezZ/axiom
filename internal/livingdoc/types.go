package livingdoc

import (
	"fmt"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

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
	// ExternalBaseline solo aparece con una baseline del proyecto declarada
	// (REQ-24.25): lista los dominios y las rutas declarados, que no se indexan.
	ExternalBaseline *ExternalBaseline `json:"external_baseline,omitempty"`
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

// ExternalBaseline informa de que la baseline de especificaciones es del
// proyecto (REQ-24.25): los documentos declarados no se indexan, y un catálogo
// vacío no debe interpretarse como ausencia de especificación.
type ExternalBaseline struct {
	Documents []ExternalBaselineDocument `json:"documents"`
}

// ExternalBaselineDocument es un documento declarado de la baseline del proyecto.
type ExternalBaselineDocument struct {
	Domain string `json:"domain"`
	Path   string `json:"path"` // absoluta
}

// ExternalBaselineError lo devuelve GetSpecDetail para un dominio de la
// baseline del proyecto: su contenido no se indexa ni se muestra aquí.
type ExternalBaselineError struct {
	Domain     string
	Path       string // documento declarado, absoluto
	ConfigPath string // axiom.yaml que lo declara
}

// Error nombra el dominio, su ruta y el axiom.yaml que lo declara.
func (e *ExternalBaselineError) Error() string {
	return fmt.Sprintf("el dominio %q pertenece a la baseline externa del proyecto (declarada en %s) y no se indexa: su documento está en %s",
		e.Domain, pathquote.Quote(e.ConfigPath), pathquote.Quote(e.Path))
}
