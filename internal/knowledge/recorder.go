package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UnitScenarioItem define un escenario BDD dentro de un requerimiento.
type UnitScenarioItem struct {
	Name  string `json:"name"`
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

// UnitRequirementItem define un requerimiento formal en formato OpenSpec.
type UnitRequirementItem struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Statement string             `json:"statement"`
	Scenarios []UnitScenarioItem `json:"scenarios,omitempty"`
}

// UnitAnalysisInput representa el payload entregado por un subagente o aplicativo para una unidad.
type UnitAnalysisInput struct {
	UnitID           string                `json:"unit_id"`
	Summary          string                `json:"summary,omitempty"`
	TechnicalDetails string                `json:"technical_details,omitempty"`
	Invariants       []string              `json:"invariants,omitempty"`
	Requirements     []UnitRequirementItem `json:"requirements,omitempty"`
	RawMarkdown      string                `json:"raw_markdown,omitempty"`
}

// RecordUnit inyecta el análisis de una unidad en la especificación viva y actualiza el manifiesto de trabajo.
func RecordUnit(ctx context.Context, jobPath string, input UnitAnalysisInput) (*CrawlJob, error) {
	if jobPath == "" {
		return nil, fmt.Errorf("ruta de job no especificada")
	}

	job, err := LoadJob(jobPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar el manifiesto de job: %w", err)
	}

	unit, found := job.FindUnit(input.UnitID)
	if !found {
		return nil, fmt.Errorf("unidad '%s' no encontrada en el job actual", input.UnitID)
	}

	// Validar payload
	if strings.TrimSpace(input.RawMarkdown) == "" && strings.TrimSpace(input.Summary) == "" {
		return nil, fmt.Errorf("el análisis de la unidad debe contener al menos 'summary' o 'raw_markdown'")
	}

	targetSpecPath := filepath.Join(job.WorkspaceRoot, unit.SpecPath)
	specDir := filepath.Dir(targetSpecPath)
	if err := os.MkdirAll(specDir, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio para spec: %w", err)
	}

	// Renderizar sección de la unidad
	sectionContent := renderUnitSection(unit, input)

	// Inyectar o actualizar en spec.md
	if err := upsertSpecSection(targetSpecPath, unit.Domain, unit.Submodule, sectionContent); err != nil {
		return nil, fmt.Errorf("error escribiendo en spec '%s': %w", targetSpecPath, err)
	}

	// Actualizar estado de la unidad en el job
	if err := job.UpdateUnitStatus(unit.ID, UnitStatusCompleted, ""); err != nil {
		return nil, fmt.Errorf("error actualizando estado de la unidad: %w", err)
	}

	if err := SaveJob(job, jobPath); err != nil {
		return nil, fmt.Errorf("error guardando manifiesto actualizado: %w", err)
	}

	return job, nil
}

// RecordUnitFailure registra el fallo de una unidad en el job.
func RecordUnitFailure(ctx context.Context, jobPath, unitID, reason string) (*CrawlJob, error) {
	job, err := LoadJob(jobPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar el job: %w", err)
	}

	if err := job.UpdateUnitStatus(unitID, UnitStatusFailed, reason); err != nil {
		return nil, fmt.Errorf("error actualizando unidad fallida: %w", err)
	}

	if err := SaveJob(job, jobPath); err != nil {
		return nil, fmt.Errorf("error guardando job: %w", err)
	}

	return job, nil
}

func renderUnitSection(unit *CrawlUnit, input UnitAnalysisInput) string {
	if strings.TrimSpace(input.RawMarkdown) != "" {
		return strings.TrimSpace(input.RawMarkdown) + "\n\n"
	}

	var sb strings.Builder
	title := strings.Title(unit.Submodule)
	if unit.Submodule == "core" || unit.Submodule == "root" {
		title = strings.Title(unit.Domain)
	}

	sb.WriteString(fmt.Sprintf("## Componente: %s (`%s`)\n\n", title, unit.ID))
	sb.WriteString(fmt.Sprintf("> **Ruta:** `%s`  \n", unit.Path))
	sb.WriteString(fmt.Sprintf("> **Archivos clave:** %d archivos analizados\n\n", len(unit.Files)))

	sb.WriteString("### Descripción y Propósito\n\n")
	sb.WriteString(strings.TrimSpace(input.Summary) + "\n\n")

	if strings.TrimSpace(input.TechnicalDetails) != "" {
		sb.WriteString("### Arquitectura Técnica\n\n")
		sb.WriteString(strings.TrimSpace(input.TechnicalDetails) + "\n\n")
	}

	if len(unit.ExposedInterfaces) > 0 {
		sb.WriteString("#### Interfaces y Símbolos Exportados\n\n")
		for _, iface := range unit.ExposedInterfaces {
			sb.WriteString(fmt.Sprintf("- `%s`\n", iface))
		}
		sb.WriteString("\n")
	}

	if len(unit.Dependencies) > 0 {
		sb.WriteString("#### Dependencias Clave\n\n")
		for _, dep := range unit.Dependencies {
			sb.WriteString(fmt.Sprintf("- `%s`\n", dep))
		}
		sb.WriteString("\n")
	}

	if len(input.Invariants) > 0 {
		sb.WriteString("### Invariantes del Sistema\n\n")
		for _, inv := range input.Invariants {
			sb.WriteString(fmt.Sprintf("- **INVARIANTE:** %s\n", inv))
		}
		sb.WriteString("\n")
	}

	if len(input.Requirements) > 0 {
		sb.WriteString("### Requerimientos Formales\n\n")
		for _, req := range input.Requirements {
			sb.WriteString(fmt.Sprintf("#### Requirement: %s (%s)\n\n", req.Title, req.ID))
			sb.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(req.Statement)))

			for _, sc := range req.Scenarios {
				sb.WriteString(fmt.Sprintf("##### Scenario: %s\n", sc.Name))
				if sc.Given != "" {
					sb.WriteString(fmt.Sprintf("- **DADO** %s\n", sc.Given))
				}
				if sc.When != "" {
					sb.WriteString(fmt.Sprintf("- **CUANDO** %s\n", sc.When))
				}
				if sc.Then != "" {
					sb.WriteString(fmt.Sprintf("- **ENTONCES** %s\n", sc.Then))
				}
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("---\n\n")
	return sb.String()
}

func upsertSpecSection(specPath, domain, submodule, newSection string) error {
	var currentContent string
	if data, err := os.ReadFile(specPath); err == nil {
		currentContent = string(data)
	}

	sectionHeaderPrefix := fmt.Sprintf("## Componente: %s", strings.Title(submodule))
	if submodule == "core" || submodule == "root" {
		sectionHeaderPrefix = fmt.Sprintf("## Componente: %s", strings.Title(domain))
	}

	if currentContent == "" {
		// Crear nuevo archivo con cabecera estándar
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Especificación Viva: %s\n\n", strings.Title(domain)))
		sb.WriteString(fmt.Sprintf("> **Dominio:** `%s`  \n", slugify(domain)))
		sb.WriteString(fmt.Sprintf("> **Generación:** Documentación viva exhaustiva (knowledge crawl)  \n"))
		sb.WriteString(fmt.Sprintf("> **Última Actualización:** %s\n\n---\n\n", time.Now().Format("2006-01-02 15:04:05")))
		sb.WriteString(newSection)
		return atomicWriteFile(specPath, []byte(sb.String()))
	}

	// Si ya contiene la sección del componente, reemplazarla
	idx := strings.Index(currentContent, sectionHeaderPrefix)
	if idx != -1 {
		// Buscar final de la sección (siguiente '## Componente:' o fin de archivo)
		rest := currentContent[idx+len(sectionHeaderPrefix):]
		nextIdx := strings.Index(rest, "\n## Componente:")
		if nextIdx != -1 {
			endIdx := idx + len(sectionHeaderPrefix) + nextIdx + 1
			updated := currentContent[:idx] + newSection + currentContent[endIdx:]
			return atomicWriteFile(specPath, []byte(updated))
		}
		// Es la última sección del archivo
		updated := currentContent[:idx] + newSection
		return atomicWriteFile(specPath, []byte(updated))
	}

	// Añadir al final del archivo existente
	updated := strings.TrimRight(currentContent, "\n") + "\n\n" + newSection
	return atomicWriteFile(specPath, []byte(updated))
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
