package dashboard

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/mcpexclusion"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Exclusiones personales de servidores MCP por proyecto (feature contexto-por-proyecto).
// Viven en el hub y se aplican a la configuración local del agente en la raíz del proyecto
// y en sus worktrees; ningún fichero versionado cambia.

// mcpNotRegisteredError dice que el proyecto abierto no está en el hub, así que no hay
// entrada donde guardar sus exclusiones. El servidor lo responde con 409.
type mcpNotRegisteredError struct{ path string }

func (e *mcpNotRegisteredError) Error() string {
	return fmt.Sprintf("el proyecto activo %s no está registrado en el Hub, así que no tiene exclusiones de servidores MCP: regístralo con `axiom project add %s`", pathquote.Quote(e.path), pathquote.Quote(e.path))
}

// MCPAgentDTO es un agente de la pantalla de exclusiones y si Axiom sabe aplicárselas.
type MCPAgentDTO struct {
	ID        string `json:"id"`
	Supported bool   `json:"supported"`
}

// MCPExclusionsDTO es la respuesta de `GET|POST /api/projects/active/mcp-exclusions`:
// el proyecto activo, sus exclusiones guardadas por agente, los nombres que se ofrecen
// para marcar, su uso en los últimos 30 días (Usage; UsageData es falso si Claude Code no
// guarda transcripciones del proyecto: «sin datos»), los directorios donde se aplican y,
// solo tras guardar, el informe.
type MCPExclusionsDTO struct {
	ID         string                     `json:"id"`
	Path       string                     `json:"path"`
	Exclusions map[string][]string        `json:"exclusions"`
	Candidates []string                   `json:"candidates"`
	Usage      []mcpexclusion.ServerUsage `json:"usage"`
	UsageData  bool                       `json:"usageData"`
	Dirs       []string                   `json:"dirs"`
	Agents     []MCPAgentDTO              `json:"agents"`
	Report     *mcpexclusion.Report       `json:"report,omitempty"`
}

// mcpExclusionsDTO arma la respuesta de la pantalla para unas exclusiones: añade los
// candidatos que se ofrecen, los directorios de aplicación y los agentes soportados.
func (s *Service) mcpExclusionsDTO(exclusions map[string][]string, id, path string) MCPExclusionsDTO {
	if exclusions == nil {
		exclusions = map[string][]string{}
	}
	candidates := candidateMCPNames(exclusions[mcpexclusion.ClaudeAgent])
	usage := mcpexclusion.UsageAnalysis{Servers: []mcpexclusion.ServerUsage{}}
	if home, err := os.UserHomeDir(); err == nil {
		usage = mcpexclusion.Analyze(home, path, candidates, exclusions[mcpexclusion.ClaudeAgent], time.Now())
	}
	return MCPExclusionsDTO{
		ID:         id,
		Path:       path,
		Exclusions: exclusions,
		Candidates: candidates,
		Usage:      usage.Servers,
		UsageData:  usage.HasData,
		Dirs:       mcpexclusion.ProjectDirs(path),
		Agents:     []MCPAgentDTO{{ID: mcpexclusion.ClaudeAgent, Supported: true}},
	}
}

// candidateMCPNames une los servidores MCP de usuario de Claude Code (solo nombres) con
// los ya excluidos, sin repetidos y ordenados: un nombre excluido a mano debe seguir
// visible para poder desmarcarlo aunque ya no exista.
func candidateMCPNames(excluded []string) []string {
	names := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		names = append(names, claude.UserMCPServerNames(home)...)
	}
	for _, name := range excluded {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// GetActiveMCPExclusions devuelve las exclusiones del proyecto activo.
func (s *Service) GetActiveMCPExclusions() (*MCPExclusionsDTO, error) {
	rec, err := s.activeProjectRecord()
	if err != nil {
		return nil, mapNotRegistered(err)
	}
	dto := s.mcpExclusionsDTO(rec.MCPExclusions, rec.ID, rec.Path)
	return &dto, nil
}

// SetActiveMCPExclusions sustituye las exclusiones del proyecto activo en el hub y las
// aplica en la configuración del agente en la raíz y en los worktrees, retirando lo que
// ya no está. Un fallo al aplicar no deshace lo guardado: queda en el informe.
func (s *Service) SetActiveMCPExclusions(exclusions map[string][]string) (*MCPExclusionsDTO, error) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()

	rec, err := s.activeProjectRecord()
	if err != nil {
		return nil, mapNotRegistered(err)
	}
	previous, updated, err := s.hubManager.SetProjectMCPExclusions(rec.ID, exclusions)
	if err != nil {
		return nil, err
	}
	report := mcpexclusion.Apply(updated, previous, updated.MCPExclusions)
	dto := s.mcpExclusionsDTO(updated.MCPExclusions, updated.ID, updated.Path)
	dto.Report = &report
	return &dto, nil
}

func mapNotRegistered(err error) error {
	var notRegistered *projectNotRegisteredError
	if errors.As(err, &notRegistered) {
		return &mcpNotRegisteredError{path: notRegistered.path}
	}
	return err
}
