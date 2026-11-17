package sddstatus

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Identidad del documento que proyecta ResolveWorkspaceTechnicalContext.
const (
	WorkspaceContextSchemaName    = "axiom.workspace-context"
	WorkspaceContextSchemaVersion = 1
)

// WorkspaceTechnicalContext es el contexto técnico declarado de un workspace,
// proyectado sin seleccionar ningún cambio SDD. TechnicalContext es nil (null en
// JSON, sin omitempty) sin declaración o con una declaración inválida; en ese
// último caso Invalid lleva el motivo (REQ-30.30: falla cerrado, sin proyección
// parcial).
type WorkspaceTechnicalContext struct {
	SchemaName       string                      `json:"schemaName"`
	SchemaVersion    int                         `json:"schemaVersion"`
	WorkspaceRoot    string                      `json:"workspaceRoot"`
	TechnicalContext *TechnicalContextProjection `json:"technicalContext"`
	Invalid          string                      `json:"invalid,omitempty"`
}

// ResolveWorkspaceTechnicalContext resuelve el workspace de cwd igual que Resolve
// sin raíz explícita (workspace.Locate) y proyecta su contexto técnico con
// projectTechnicalContext. Un axiom.yaml ilegible o un bloque inválido no es un
// error de la función: se informa en Invalid. Solo falla si el workspace no se
// puede localizar.
func ResolveWorkspaceTechnicalContext(cwd string) (WorkspaceTechnicalContext, error) {
	location, err := workspace.Locate(cwd)
	if err != nil {
		return WorkspaceTechnicalContext{}, err
	}
	result := WorkspaceTechnicalContext{
		SchemaName:       WorkspaceContextSchemaName,
		SchemaVersion:    WorkspaceContextSchemaVersion,
		WorkspaceRoot:    location.WorkspaceRoot,
		TechnicalContext: projectTechnicalContext(location),
	}
	var reasons []string
	if location.Declarations.ParseErr != nil {
		reasons = append(reasons, errorLines(location.Declarations.ParseErr)...)
	}
	if location.Declarations.TechnicalContextErr != nil {
		reasons = append(reasons, errorLines(location.Declarations.TechnicalContextErr)...)
	}
	result.Invalid = strings.Join(reasons, "; ")
	return result, nil
}
