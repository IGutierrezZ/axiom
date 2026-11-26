package dashboard

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// errNoActiveProject es el rechazo de una instalación sin proyecto activo.
var errNoActiveProject = errors.New("no hay un proyecto de Axiom activo: abre o da de alta un proyecto antes de instalar agentes")

// detectAgentsFn devuelve los agentes que se detectan en el equipo: los que tienen
// su carpeta de configuración global en homeDir. Es una comprobación de ficheros,
// sin subprocesos. Es una variable para que las pruebas fijen el resultado sin
// crear carpetas de agentes en el HOME de pruebas.
var detectAgentsFn = func(homeDir string) map[model.AgentID]bool {
	detected := map[model.AgentID]bool{}
	reg, err := agents.NewDefaultRegistry()
	if err != nil || homeDir == "" {
		return detected
	}
	for _, agent := range agents.DiscoverInstalled(reg, homeDir) {
		detected[agent.ID] = true
	}
	return detected
}

// ListProjectAgents devuelve los agentes del catálogo, en su orden, con tres marcas
// (admitido, detectado en el equipo e instalado en el proyecto activo) y si se
// pueden instalar en el proyecto. Es de solo lectura.
//
// Un agente solo figura como instalado cuando el proyecto ha registrado su lista
// (Store.AgentsRecorded): la semilla global no aporta agentes (state.ProjectView),
// así que la lista global nunca cuenta como la del proyecto.
func (s *Service) ListProjectAgents() ProjectAgentsDTO {
	root := s.getRootPath()
	home, _ := os.UserHomeDir()
	view := ProjectAgentsDTO{ProjectRoot: root, Project: isAxiomProject(root), Agents: []ProjectAgentDTO{}}

	installed := map[string]bool{}
	if root != "" {
		store := state.ProjectStore(home, root)
		current, err := store.Read()
		switch {
		case err == nil:
			if store.AgentsRecorded(current) {
				view.Recorded = true
				for _, id := range current.InstalledAgents {
					installed[id] = true
				}
			}
		case errors.Is(err, os.ErrNotExist):
			// Ni el proyecto ni la semilla global tienen estado: nada registrado.
		default:
			view.StateError = fmt.Sprintf("no se pudo leer el estado del proyecto (%s): %v", store.Path(), err)
		}
	}

	detected := detectAgentsFn(home)
	for _, agent := range catalog.AllAgents() {
		view.Agents = append(view.Agents, ProjectAgentDTO{
			ID:        string(agent.ID),
			Name:      agent.Name,
			Supported: catalog.IsSelectableAgent(agent.ID),
			Detected:  detected[agent.ID],
			Installed: installed[string(agent.ID)],
			Workspace: cli.AgentSupportsWorkspace(agent.ID),
			Reason:    projectAgentReason(agent.ID),
		})
	}
	return view
}

// projectAgentReason explica por qué el agente no se puede instalar en el proyecto,
// o devuelve "" si se puede.
func projectAgentReason(id model.AgentID) string {
	if !catalog.IsSelectableAgent(id) {
		return "no está disponible de momento en Axiom"
	}
	return cli.WorkspaceUnsupportedReason(id)
}

// agentName devuelve el nombre del agente en el catálogo, o su identificador.
func agentName(id string) string {
	for _, agent := range catalog.AllAgents() {
		if string(agent.ID) == id {
			return agent.Name
		}
	}
	return id
}

// AgentInstallError es el rechazo de una instalación antes de escribir nada:
// Status es el código HTTP con el que se responde (400 o 409).
type AgentInstallError struct {
	Status int
	Err    error
}

func (e *AgentInstallError) Error() string { return e.Err.Error() }

// CheckProjectAgentInstall valida, sin escribir nada, la instalación de un agente
// en el proyecto activo: el agente debe existir, ser seleccionable y admitir el
// ámbito workspace, y tiene que haber un proyecto activo.
func (s *Service) CheckProjectAgentInstall(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return &AgentInstallError{Status: 400, Err: errors.New("falta el agente que instalar")}
	}
	if violations := adapterViolations([]string{id}); len(violations) > 0 {
		return &AgentInstallError{Status: 400, Err: errors.New(strings.Join(violations, "; "))}
	}
	if !isAxiomProject(s.getRootPath()) {
		return &AgentInstallError{Status: 409, Err: errNoActiveProject}
	}
	return nil
}

// isAxiomProject dice si root es la raíz de un proyecto de Axiom: tiene en sí misma
// un marcador de workspace (puntero, axiom.yaml u openspec/). Sin él, instalar
// escribiría `.axiom/` y la configuración de agentes en una carpeta cualquiera.
func isAxiomProject(root string) bool {
	if root == "" {
		return false
	}
	loc, err := workspace.LocateAt(root)
	return err == nil && loc.Origin != workspace.OriginStart
}

// InstallProjectAgent instala un agente en el proyecto activo con el núcleo de
// `axiom install --scope workspace --agent <id>` (installAdapters), que lo añade a
// installed_agents del estado del proyecto sin quitar los ya registrados. Las
// instalaciones se ejecutan de una en una. Un rechazo previo devuelve
// *AgentInstallError y no escribe nada.
func (s *Service) InstallProjectAgent(id string) (*AgentInstallResponse, error) {
	if err := s.CheckProjectAgentInstall(id); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	root := s.getRootPath()

	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	name := agentName(id)
	installed, err := installAdapters(root, []string{id})
	if err != nil {
		return &AgentInstallResponse{
			Success: false,
			Agent:   id,
			Message: fmt.Sprintf("No se pudo instalar %s en el proyecto", name),
			Error:   err.Error(),
		}, err
	}
	return &AgentInstallResponse{
		Success: true,
		Agent:   id,
		Message: fmt.Sprintf("%s instalado en el proyecto %s", name, root),
		Agents:  installed,
	}, nil
}
