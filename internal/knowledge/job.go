package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CrawlUnitStatus define el ciclo de vida de una unidad discreta de análisis.
type CrawlUnitStatus string

const (
	UnitStatusPending    CrawlUnitStatus = "pending"
	UnitStatusInProgress CrawlUnitStatus = "in_progress"
	UnitStatusCompleted  CrawlUnitStatus = "completed"
	UnitStatusFailed     CrawlUnitStatus = "failed"
)

// CrawlJobStatus define el estado general del trabajo de crawling exhaustivo.
type CrawlJobStatus string

const (
	JobStatusPending   CrawlJobStatus = "pending"
	JobStatusRunning   CrawlJobStatus = "running"
	JobStatusCompleted CrawlJobStatus = "completed"
	JobStatusFailed    CrawlJobStatus = "failed"
)

// CrawlUnit representa una unidad de trabajo acotada (módulo o submódulo) para análisis semántico.
type CrawlUnit struct {
	ID                string          `json:"id"`
	Domain            string          `json:"domain"`
	Submodule         string          `json:"submodule"`
	Path              string          `json:"path"`
	Files             []string        `json:"files"`
	Dependencies      []string        `json:"dependencies"`
	ExposedInterfaces []string        `json:"exposed_interfaces"`
	Status            CrawlUnitStatus `json:"status"`
	Error             string          `json:"error,omitempty"`
	SpecPath          string          `json:"spec_path"`
	StartedAt         *time.Time      `json:"started_at,omitempty"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
}

// CrawlJob representa el manifiesto completo del proceso de crawling con control de estado y reanudación.
type CrawlJob struct {
	ID             string         `json:"id"`
	WorkspaceRoot  string         `json:"workspace_root"`
	SpecsRoot      string         `json:"specs_root"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Status         CrawlJobStatus `json:"status"`
	Units          []CrawlUnit    `json:"units"`
	TotalUnits     int            `json:"total_units"`
	CompletedUnits int            `json:"completed_units"`
	FailedUnits    int            `json:"failed_units"`
}

// DefaultJobRelativePath define la ubicación estándar del manifiesto de crawling dentro del workspace.
const DefaultJobRelativePath = ".axiom/knowledge/crawl-job.json"

// DefaultJobPath devuelve la ruta canónica absoluta al fichero de estado del job.
func DefaultJobPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".axiom", "knowledge", "crawl-job.json")
}

// NewCrawlJob inicializa un nuevo CrawlJob con las unidades proporcionadas.
func NewCrawlJob(id, workspaceRoot, specsRoot string, units []CrawlUnit) *CrawlJob {
	now := time.Now()
	job := &CrawlJob{
		ID:            id,
		WorkspaceRoot: workspaceRoot,
		SpecsRoot:     specsRoot,
		CreatedAt:     now,
		UpdatedAt:     now,
		Status:        JobStatusPending,
		Units:         units,
		TotalUnits:    len(units),
	}
	job.RecalculateProgress()
	return job
}

// RecalculateProgress recalcula los contadores y el estado global del trabajo.
func (j *CrawlJob) RecalculateProgress() {
	j.TotalUnits = len(j.Units)
	completed := 0
	failed := 0
	inProgress := 0

	for _, u := range j.Units {
		switch u.Status {
		case UnitStatusCompleted:
			completed++
		case UnitStatusFailed:
			failed++
		case UnitStatusInProgress:
			inProgress++
		}
	}

	j.CompletedUnits = completed
	j.FailedUnits = failed

	if j.TotalUnits == 0 {
		j.Status = JobStatusPending
		return
	}

	if completed == j.TotalUnits {
		j.Status = JobStatusCompleted
	} else if failed > 0 && (completed+failed == j.TotalUnits) {
		j.Status = JobStatusFailed
	} else if inProgress > 0 || completed > 0 {
		j.Status = JobStatusRunning
	} else {
		j.Status = JobStatusPending
	}
}

// ProgressPercentage calcula el porcentaje de unidades completadas (0.0 a 100.0).
func (j *CrawlJob) ProgressPercentage() float64 {
	if j.TotalUnits == 0 {
		return 0.0
	}
	return (float64(j.CompletedUnits) / float64(j.TotalUnits)) * 100.0
}

// UpdateUnitStatus actualiza el estado de una unidad concreta por su ID y recalcula el progreso.
func (j *CrawlJob) UpdateUnitStatus(unitID string, status CrawlUnitStatus, errMsg string) error {
	found := false
	now := time.Now()
	for i := range j.Units {
		if j.Units[i].ID == unitID {
			j.Units[i].Status = status
			j.Units[i].Error = errMsg
			if status == UnitStatusInProgress && j.Units[i].StartedAt == nil {
				j.Units[i].StartedAt = &now
			}
			if status == UnitStatusCompleted || status == UnitStatusFailed {
				j.Units[i].CompletedAt = &now
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unidad con ID '%s' no encontrada en el job", unitID)
	}
	j.UpdatedAt = now
	j.RecalculateProgress()
	return nil
}

// FindUnit busca una unidad por su ID.
func (j *CrawlJob) FindUnit(unitID string) (*CrawlUnit, bool) {
	for i := range j.Units {
		if j.Units[i].ID == unitID {
			return &j.Units[i], true
		}
	}
	return nil, false
}

// SaveJob persiste atómicamente el estado del job en disco.
func SaveJob(job *CrawlJob, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio para el job: %w", err)
	}

	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando el job a JSON: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("error escribiendo fichero temporal de job: %w", err)
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("error al renombrar fichero de job atómico: %w", err)
	}
	return nil
}

// LoadJob carga y deserializa el manifiesto de trabajo desde disco.
func LoadJob(path string) (*CrawlJob, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el fichero de job '%s': %w", path, err)
	}

	var job CrawlJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("error deserializando job JSON: %w", err)
	}

	return &job, nil
}
