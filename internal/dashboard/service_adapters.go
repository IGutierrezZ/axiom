package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// installAdapters instala adaptadores en la raíz de un proyecto. Es una variable
// para que las pruebas no ejecuten la instalación real.
var installAdapters = installAdaptersInWorkspace

// AdapterDTO describe un adaptador (agente) elegible al dar de alta un proyecto.
// Workspace es si se puede instalar en el modo workspace, el único que usa el alta;
// los que no, llevan en Reason el motivo.
type AdapterDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Workspace bool   `json:"workspace"`
	Reason    string `json:"reason,omitempty"`
}

// AdaptersOutcome es el resultado de instalar los adaptadores pedidos en el alta.
// Error no vacío es un fallo parcial: el proyecto ya está creado y no se deshace.
type AdaptersOutcome struct {
	Requested []string `json:"requested"`
	Installed []string `json:"installed"`
	Error     string   `json:"error,omitempty"`
}

// ProjectInitResponse es la respuesta de POST /api/projects/init: el resultado del
// alta (con su forma de siempre) y, si se pidieron adaptadores, lo que pasó con ellos.
type ProjectInitResponse struct {
	*hub.InitResult
	Adapters *AdaptersOutcome `json:"adapters,omitempty"`
	// ReviewModeError es un fallo parcial al guardar la preferencia de revisión pedida: el
	// proyecto ya está creado y registrado y no se deshace.
	ReviewModeError string `json:"review_mode_error,omitempty"`
}

// ListAdapters devuelve el catálogo de adaptadores seleccionables, en el orden del catálogo.
func (s *Service) ListAdapters() []AdapterDTO {
	agents := catalog.SelectableAgents()
	out := make([]AdapterDTO, 0, len(agents))
	for _, agent := range agents {
		out = append(out, AdapterDTO{
			ID:        string(agent.ID),
			Name:      agent.Name,
			Workspace: cli.AgentSupportsWorkspace(agent.ID),
			Reason:    cli.WorkspaceUnsupportedReason(agent.ID),
		})
	}
	return out
}

// adapterViolations valida los identificadores pedidos: deben existir, ser
// seleccionables y admitir el modo workspace. Los repetidos se cuentan una vez.
func adapterViolations(ids []string) []string {
	var violations []string
	seen := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if seen[id] {
			continue
		}
		seen[id] = true
		agent := model.AgentID(id)
		switch {
		case !catalog.IsSupportedAgent(agent):
			violations = append(violations, fmt.Sprintf("el adaptador %q no existe", id))
		case !catalog.IsSelectableAgent(agent):
			violations = append(violations, fmt.Sprintf("el adaptador %q no está disponible de momento", id))
		case !cli.AgentSupportsWorkspace(agent):
			violations = append(violations, fmt.Sprintf("el adaptador %q no se puede instalar en el proyecto: %s", id, cli.WorkspaceUnsupportedReason(agent)))
		}
	}
	return violations
}

// uniqueAdapterIDs devuelve los identificadores sin repetir ni vacíos, en orden.
func uniqueAdapterIDs(ids []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// checkRequest rechaza, antes de escribir nada, un alta con adaptadores o modo de
// revisión inválidos. Reúne con ellos las violaciones de la configuración del proyecto
// para que el usuario vea todos los errores a la vez.
func checkRequest(ini *hub.Initializer, opts hub.InitOptions, ids []string, reviewMode string) error {
	violations := adapterViolations(ids)
	if _, err := normalizeReviewModeInput(reviewMode); err != nil {
		violations = append(violations, err.Error())
	}
	if len(violations) == 0 {
		return nil
	}
	var all []string
	var invalid *hub.InitValidationError
	if err := ini.ValidateInit(opts); errors.As(err, &invalid) {
		all = append(all, invalid.Violations...)
	}
	return &hub.InitValidationError{Violations: append(all, violations...)}
}

// installAdaptersInWorkspace instala los adaptadores en la raíz del proyecto con el
// mismo núcleo que `axiom install --scope workspace`, sin depender del directorio
// actual del proceso.
func installAdaptersInWorkspace(projectDir string, ids []string) ([]string, error) {
	detection, err := system.Detect(context.Background())
	if err != nil {
		detection = system.DetectionResult{}
	}
	result, err := cli.InstallInWorkspace(projectDir, []string{"--agent", strings.Join(ids, ",")}, detection)
	if err != nil {
		return nil, err
	}
	installed := make([]string, 0, len(result.Selection.Agents))
	for _, agent := range result.Selection.Agents {
		installed = append(installed, string(agent))
	}
	return installed, nil
}
