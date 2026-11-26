package cli

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// azureDevOpsMCPOrganization devuelve la organización del plugin azure-devops
// cuando está activo en el axiom.yaml de workspaceDir, para que el componente
// MCP inyecte también el servidor MCP oficial de Azure DevOps (C-07-g). Devuelve
// "" fuera del ámbito workspace, sin axiom.yaml, con el plugin ausente o
// desactivado, o con una configuración inválida: en todos esos casos nada
// cambia, y `axiom workspace validate` es quien informa de la configuración.
func azureDevOpsMCPOrganization(scope InstallScope, workspaceDir string) string {
	if scope != ScopeWorkspace || strings.TrimSpace(workspaceDir) == "" {
		return ""
	}
	loc, err := workspace.LocateAt(workspaceDir)
	if err != nil || loc.ConfigErr != nil || loc.Declarations.IntegrationsErr != nil {
		return ""
	}
	for _, plugin := range loc.Declarations.Integrations.Active() {
		if plugin.ID == manifest.AzureDevOpsID {
			organization, _ := plugin.Values["organization"].(string)
			return strings.TrimSpace(organization)
		}
	}
	return ""
}
