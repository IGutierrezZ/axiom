package dashboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

// Identificadores de los bloques de /api/ecosystem/models y del campo `adapter` del POST.
// Copilot es un solo bloque que cubre VS Code Copilot y Copilot CLI: comparten el mismo
// `.agent.md` por fase, que vive solo en .github/agents del proyecto.
const (
	modelsAdapterOpenCode = "opencode"
	modelsAdapterClaude   = "claude-code"
	modelsAdapterCopilot  = "copilot"
)

// copilotModelSuggestions son nombres de ejemplo para el campo de texto libre de Copilot:
// el catálogo de Copilot cambia a menudo y Axiom no lo valida.
var copilotModelSuggestions = []string{
	"Claude Sonnet 4.6 (copilot)",
	"Claude Haiku 4.5 (copilot)",
	"GPT-5.2 (copilot)",
}

// discoverOpenCodeCatalog consulta el catálogo de modelos de OpenCode. Es una variable
// para que las pruebas no ejecuten el binario `opencode`.
var discoverOpenCodeCatalog = opencode.DiscoverCatalog

// ErrNoProjectOpen indica que no hay un proyecto abierto sobre el que guardar.
var ErrNoProjectOpen = errors.New("no hay un proyecto abierto: abre o registra uno en el dashboard antes de elegir modelos")

// ModelsValidationError reúne todas las violaciones de una petición de modelos, para que
// la pantalla las muestre una a una.
type ModelsValidationError struct {
	Violations []string
}

func (e *ModelsValidationError) Error() string {
	return "las asignaciones de modelo no son válidas: " + strings.Join(e.Violations, "; ")
}

// adapterModelBlocks devuelve un bloque por cada adaptador (OpenCode, Claude Code,
// Copilot) instalado en el proyecto de root, leyendo su estado en .axiom/state.json.
func (s *Service) adapterModelBlocks(ctx context.Context, root string) ([]AdapterModelsDTO, error) {
	project, err := cli.ReadProjectPhaseModels(root)
	if err != nil {
		return nil, err
	}
	blocks := []AdapterModelsDTO{}
	if hasAgent(project.Agents, model.AgentOpenCode) {
		blocks = append(blocks, openCodeBlock(ctx, root, project))
	}
	if hasAgent(project.Agents, model.AgentClaudeCode) {
		blocks = append(blocks, claudeBlock(project))
	}
	if hasAgent(project.Agents, model.AgentVSCodeCopilot) || hasAgent(project.Agents, model.AgentGitHubCopilotCLI) {
		blocks = append(blocks, copilotBlock(project))
	}
	return blocks, nil
}

func hasAgent(agents []string, id model.AgentID) bool {
	return slices.Contains(agents, string(id))
}

// openCodePhases son los sub-agentes de OpenCode con modelo propio: las fases SDD y los
// agentes de Judgment Day.
func openCodePhases() []string {
	return append(opencode.SDDPhases(), opencode.JDPhases()...)
}

// openCodeAssignments son las asignaciones actuales de OpenCode: las del estado del
// proyecto y, si no hay, las que ya tuviera opencode.json.
func openCodeAssignments(root string, project cli.ProjectPhaseModels) map[string]model.ModelAssignment {
	if len(project.OpenCode) > 0 {
		return project.OpenCode
	}
	return readOpenCodeConfigAssignments(root)
}

func openCodeBlock(ctx context.Context, root string, project cli.ProjectPhaseModels) AdapterModelsDTO {
	phases := openCodePhases()
	assignments := map[string]PhaseModelDTO{}
	for phase, a := range openCodeAssignments(root, project) {
		if slices.Contains(phases, phase) && a.ProviderID != "" && a.ModelID != "" {
			assignments[phase] = PhaseModelDTO{Model: a.FullID(), Effort: a.Effort}
		}
	}
	block := AdapterModelsDTO{
		ID:           modelsAdapterOpenCode,
		Name:         "OpenCode",
		Phases:       phases,
		Assignments:  assignments,
		Destinations: []string{"opencode.json (proyecto)"},
	}
	catalog, err := discoverOpenCodeCatalog(ctx, root)
	if err != nil {
		block.Options.CatalogError = err.Error()
	} else {
		block.Options.Providers = providerOptions(catalog)
	}
	block.Notes = openCodeNotes(project)
	return block
}

// openCodeNotes avisa de los límites de OpenCode: el modelo por fase solo aplica en el
// modo SDD multi, y quitar una fase no borra el modelo que ya esté en opencode.json.
func openCodeNotes(project cli.ProjectPhaseModels) []string {
	notes := []string{
		"Quitar el modelo de una fase lo olvida en Axiom, pero no borra el que ya esté escrito en opencode.json: edítalo allí para retirarlo.",
	}
	if project.SDDMode != string(model.SDDModeMulti) {
		notes = append([]string{"OpenCode solo aplica el modelo por fase en el modo SDD «multi»; este proyecto no lo usa, así que la elección queda guardada pero no tendrá efecto hasta cambiar de modo."}, notes...)
	}
	return notes
}

// providerOptions convierte el catálogo de OpenCode en opciones: solo los modelos con
// herramientas (los que sirven para SDD), ordenados por proveedor y por nombre.
func providerOptions(catalog map[string]opencode.Provider) []ProviderOptionDTO {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]ProviderOptionDTO, 0, len(ids))
	for _, id := range ids {
		provider := catalog[id]
		models := opencode.FilterModelsForSDD(provider)
		if len(models) == 0 {
			continue
		}
		option := ProviderOptionDTO{ID: id, Name: provider.Name, Models: make([]CatalogModelOptionDTO, 0, len(models))}
		if option.Name == "" {
			option.Name = id
		}
		for _, m := range models {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			option.Models = append(option.Models, CatalogModelOptionDTO{ID: m.ID, Name: name, Variants: append([]string{}, m.EffortLevels()...)})
		}
		out = append(out, option)
	}
	return out
}

// claudeModelOrder es el orden en que se ofrecen los alias de Claude Code.
var claudeModelOrder = []model.ClaudeModelAlias{
	model.ClaudeModelFable, model.ClaudeModelOpus, model.ClaudeModelSonnet, model.ClaudeModelHaiku,
}

func claudeBlock(project cli.ProjectPhaseModels) AdapterModelsDTO {
	assignments := map[string]PhaseModelDTO{}
	for phase, a := range project.Claude {
		if phase != "orchestrator" && a.Valid() {
			assignments[phase] = PhaseModelDTO{Model: string(a.Model), Effort: string(a.Effort)}
		}
	}
	defaults := map[string]PhaseModelDTO{}
	for phase, alias := range model.ClaudeModelPresetBalanced() {
		if phase != "orchestrator" {
			defaults[phase] = PhaseModelDTO{Model: string(alias)}
		}
	}
	options := ModelOptionsDTO{}
	for _, alias := range claudeModelOrder {
		efforts := []string{}
		for _, effort := range model.ClaudeEffortsForModel(alias) {
			if effort != model.ClaudeEffortDefault {
				efforts = append(efforts, string(effort))
			}
		}
		options.Models = append(options.Models, EffortModelOptionDTO{ID: string(alias), Efforts: efforts})
	}
	return AdapterModelsDTO{
		ID:           modelsAdapterClaude,
		Name:         "Claude Code",
		Phases:       model.ClaudePhaseNames(),
		Assignments:  assignments,
		Defaults:     defaults,
		Options:      options,
		Destinations: []string{".claude/agents"},
		Notes: []string{
			"Claude Code controla el modelo de la sesión principal: aquí solo se eligen los de los sub-agentes. Una fase sin asignación usa el valor por defecto de Axiom.",
		},
	}
}

func copilotBlock(project cli.ProjectPhaseModels) AdapterModelsDTO {
	assignments := map[string]PhaseModelDTO{}
	for phase, a := range project.Copilot {
		assignments[phase] = PhaseModelDTO{Model: a.Model, Effort: a.Effort}
	}
	block := AdapterModelsDTO{
		ID:          modelsAdapterCopilot,
		Name:        "GitHub Copilot",
		Phases:      model.CopilotPhaseNames(),
		Assignments: assignments,
		Options:     ModelOptionsDTO{Suggestions: copilotModelSuggestions, FreeText: true},
		Notes: []string{
			"El nombre del modelo es texto libre con el formato de VS Code, por ejemplo «Claude Sonnet 4.6 (copilot)». Una fase sin asignación hereda el modelo de la conversación.",
		},
	}
	// Los agentes por fase de Copilot viven solo en .github/agents del proyecto: los lee VS
	// Code Copilot y también Copilot CLI, así que basta con que haya uno de los dos.
	block.Destinations = []string{".github/agents"}
	return block
}

// SaveModelAssignments valida y guarda las asignaciones por fase de un adaptador en el
// estado del proyecto abierto y regenera sus ficheros en el proyecto, sin cambiar el
// directorio actual del proceso. La petición es un parche: una fase con modelo vacío se
// quita y las que no aparecen se conservan.
func (s *Service) SaveModelAssignments(ctx context.Context, req ModelsSaveRequest) (*ModelsSaveResponse, error) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()

	root := s.getRootPath()
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, ErrNoProjectOpen
	}
	project, err := cli.ReadProjectPhaseModels(root)
	if err != nil {
		return nil, err
	}

	var (
		violations []string
		overrides  cli.PhaseModelOverrides
		agents     []model.AgentID
		result     map[string]PhaseModelDTO
		notes      []string
	)
	switch req.Adapter {
	case modelsAdapterOpenCode:
		violations = installedViolation(violations, project, "OpenCode", model.AgentOpenCode)
		overrides.OpenCode, result, violations = s.mergeOpenCode(ctx, root, project, req.Assignments, violations)
		agents = []model.AgentID{model.AgentOpenCode}
		notes = openCodeNotes(project)
	case modelsAdapterClaude:
		violations = installedViolation(violations, project, "Claude Code", model.AgentClaudeCode)
		overrides.Claude, result, violations = mergeClaude(project, req.Assignments, violations)
		agents = []model.AgentID{model.AgentClaudeCode}
	case modelsAdapterCopilot:
		if !hasAgent(project.Agents, model.AgentVSCodeCopilot) && !hasAgent(project.Agents, model.AgentGitHubCopilotCLI) {
			violations = append(violations, "GitHub Copilot no está instalado en este proyecto")
		}
		overrides.Copilot, result, violations = mergeCopilot(project, req.Assignments, violations)
		// Con VS Code Copilot en el proyecto, su adaptador escribe .github/agents. Con solo
		// Copilot CLI (que no tiene modo workspace) los escribe el sync del proyecto.
		if hasAgent(project.Agents, model.AgentVSCodeCopilot) {
			agents = []model.AgentID{model.AgentVSCodeCopilot}
		} else if hasAgent(project.Agents, model.AgentGitHubCopilotCLI) {
			agents = []model.AgentID{model.AgentGitHubCopilotCLI}
		}
	default:
		violations = append(violations, fmt.Sprintf("el adaptador %q no existe: usa %s, %s o %s", req.Adapter, modelsAdapterOpenCode, modelsAdapterClaude, modelsAdapterCopilot))
	}
	if len(violations) > 0 {
		return nil, &ModelsValidationError{Violations: violations}
	}

	sync, err := cli.SaveProjectPhaseModels(root, agents, overrides)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron guardar las asignaciones: %w", err)
	}
	return &ModelsSaveResponse{
		Adapter:     req.Adapter,
		Assignments: result,
		Regenerated: relativePaths(root, sync.ChangedFiles),
		Notes:       notes,
	}, nil
}

func installedViolation(violations []string, project cli.ProjectPhaseModels, name string, id model.AgentID) []string {
	if !hasAgent(project.Agents, id) {
		violations = append(violations, name+" no está instalado en este proyecto")
	}
	return violations
}

// relativePaths devuelve las rutas relativas a root con barras normales; las que caen
// fuera de root se dejan absolutas. Siempre un slice, para que el JSON lleve `[]`.
func relativePaths(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			p = rel
		}
		out = append(out, filepath.ToSlash(p))
	}
	slices.Sort(out)
	return out
}

// mergeClaude aplica el parche sobre las asignaciones guardadas y valida lo que llega
// con el mismo validador que usa el modelo (claves, alias y esfuerzo admitido).
func mergeClaude(project cli.ProjectPhaseModels, patch map[string]PhaseModelDTO, violations []string) (map[string]model.ClaudePhaseAssignment, map[string]PhaseModelDTO, []string) {
	incoming := map[string]model.ClaudePhaseAssignment{}
	merged := map[string]model.ClaudePhaseAssignment{}
	for phase, a := range project.Claude {
		if phase != "orchestrator" && a.Valid() {
			merged[phase] = a
		}
	}
	for phase, p := range patch {
		if strings.TrimSpace(p.Model) == "" {
			delete(merged, phase)
			continue
		}
		a := model.ClaudePhaseAssignment{Model: model.ClaudeModelAlias(strings.TrimSpace(p.Model)), Effort: model.ClaudeEffort(strings.TrimSpace(p.Effort))}
		incoming[phase] = a
		merged[phase] = a
	}
	violations = append(violations, unknownPhaseOnClear(patch, model.ClaudePhaseNames())...)
	violations = append(violations, model.ClaudePhaseAssignmentProblems(incoming)...)
	result := map[string]PhaseModelDTO{}
	for phase, a := range merged {
		result[phase] = PhaseModelDTO{Model: string(a.Model), Effort: string(a.Effort)}
	}
	return merged, result, violations
}

// mergeCopilot aplica el parche sobre las asignaciones guardadas y valida lo que llega
// con el mismo validador que usa `axiom sync` (fases, modelo no vacío, sin caracteres de
// control).
func mergeCopilot(project cli.ProjectPhaseModels, patch map[string]PhaseModelDTO, violations []string) (map[string]model.CopilotPhaseAssignment, map[string]PhaseModelDTO, []string) {
	incoming := map[string]model.CopilotPhaseAssignment{}
	merged := map[string]model.CopilotPhaseAssignment{}
	for phase, a := range project.Copilot {
		merged[phase] = a
	}
	for phase, p := range patch {
		if strings.TrimSpace(p.Model) == "" {
			delete(merged, phase)
			continue
		}
		a := model.CopilotPhaseAssignment{Model: strings.TrimSpace(p.Model), Effort: strings.TrimSpace(p.Effort)}
		incoming[phase] = a
		merged[phase] = a
	}
	violations = append(violations, unknownPhaseOnClear(patch, model.CopilotPhaseNames())...)
	violations = append(violations, model.CopilotPhaseAssignmentProblems(incoming)...)
	result := map[string]PhaseModelDTO{}
	for phase, a := range merged {
		result[phase] = PhaseModelDTO{Model: a.Model, Effort: a.Effort}
	}
	return merged, result, violations
}

// unknownPhaseOnClear avisa de las fases desconocidas que el parche quita (las que
// llegan con modelo ya las revisa el validador del modelo).
func unknownPhaseOnClear(patch map[string]PhaseModelDTO, known []string) []string {
	var unknown []string
	for phase, p := range patch {
		if strings.TrimSpace(p.Model) == "" && !slices.Contains(known, phase) {
			unknown = append(unknown, phase)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return []string{fmt.Sprintf("fases desconocidas: %s", strings.Join(unknown, ", "))}
}

// mergeOpenCode aplica el parche sobre las asignaciones actuales de OpenCode. Valida las
// fases, el formato «proveedor/modelo» y, cuando el catálogo está disponible, que el
// modelo exista y que la variante la admita.
func (s *Service) mergeOpenCode(ctx context.Context, root string, project cli.ProjectPhaseModels, patch map[string]PhaseModelDTO, violations []string) (map[string]model.ModelAssignment, map[string]PhaseModelDTO, []string) {
	phases := openCodePhases()
	catalog, catalogErr := discoverOpenCodeCatalog(ctx, root)
	merged := map[string]model.ModelAssignment{}
	for phase, a := range openCodeAssignments(root, project) {
		if a.ProviderID != "" && a.ModelID != "" {
			merged[phase] = a
		}
	}
	var unknown []string
	for phase, p := range patch {
		if !slices.Contains(phases, phase) {
			unknown = append(unknown, phase)
			continue
		}
		spec := strings.TrimSpace(p.Model)
		if spec == "" {
			delete(merged, phase)
			continue
		}
		providerID, modelID, ok := model.SplitModelSpec(spec)
		if !ok {
			violations = append(violations, fmt.Sprintf("%s: el modelo %q debe tener el formato proveedor/modelo", phase, spec))
			continue
		}
		effort := strings.TrimSpace(p.Effort)
		if strings.ContainsFunc(spec+effort, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
			violations = append(violations, fmt.Sprintf("%s: el modelo y la variante no pueden llevar caracteres de control", phase))
			continue
		}
		if catalogErr == nil {
			violations = append(violations, openCodeCatalogProblems(catalog, phase, providerID, modelID, effort)...)
		}
		merged[phase] = model.ModelAssignment{ProviderID: providerID, ModelID: modelID, Effort: effort}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		violations = append(violations, fmt.Sprintf("fases desconocidas: %s", strings.Join(unknown, ", ")))
	}
	result := map[string]PhaseModelDTO{}
	for phase, a := range merged {
		result[phase] = PhaseModelDTO{Model: a.FullID(), Effort: a.Effort}
	}
	return merged, result, violations
}

// openCodeCatalogProblems comprueba una asignación contra el catálogo de OpenCode.
func openCodeCatalogProblems(catalog map[string]opencode.Provider, phase, providerID, modelID, effort string) []string {
	provider, ok := catalog[providerID]
	if !ok {
		return []string{fmt.Sprintf("%s: el proveedor %q no está en el catálogo de OpenCode", phase, providerID)}
	}
	m, ok := provider.Models[modelID]
	if !ok {
		return []string{fmt.Sprintf("%s: el modelo %q no está en el catálogo del proveedor %q", phase, modelID, providerID)}
	}
	if effort != "" && !slices.Contains(m.Variants, effort) {
		if len(m.Variants) == 0 {
			return []string{fmt.Sprintf("%s: el modelo %s/%s no admite variantes (%q)", phase, providerID, modelID, effort)}
		}
		return []string{fmt.Sprintf("%s: la variante %q no es válida para %s/%s (opciones: %s)", phase, effort, providerID, modelID, strings.Join(m.Variants, ", "))}
	}
	return nil
}
