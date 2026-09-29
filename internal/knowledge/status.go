package knowledge

import (
	"fmt"
	"strings"
	"time"
)

// UnitStatusBrief resume el estado de una unidad de crawl.
type UnitStatusBrief struct {
	ID        string          `json:"id"`
	Domain    string          `json:"domain"`
	Submodule string          `json:"submodule"`
	Status    CrawlUnitStatus `json:"status"`
	Error     string          `json:"error,omitempty"`
}

// CrawlStatusSummary expone métricas y estados de ejecución para herramientas CLI y aplicativas.
type CrawlStatusSummary struct {
	JobID          string            `json:"job_id"`
	Status         CrawlJobStatus    `json:"status"`
	TotalUnits     int               `json:"total_units"`
	CompletedUnits int               `json:"completed_units"`
	FailedUnits    int               `json:"failed_units"`
	PendingUnits   int               `json:"pending_units"`
	ProgressPct    float64           `json:"progress_pct"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Units          []UnitStatusBrief `json:"units"`
}

// GetCrawlStatus calcula y devuelve el resumen del estado del job.
func GetCrawlStatus(jobPath string) (*CrawlStatusSummary, error) {
	job, err := LoadJob(jobPath)
	if err != nil {
		return nil, fmt.Errorf("error cargando job: %w", err)
	}

	job.RecalculateProgress()

	briefs := make([]UnitStatusBrief, 0, len(job.Units))
	for _, u := range job.Units {
		briefs = append(briefs, UnitStatusBrief{
			ID:        u.ID,
			Domain:    u.Domain,
			Submodule: u.Submodule,
			Status:    u.Status,
			Error:     u.Error,
		})
	}

	pending := job.TotalUnits - (job.CompletedUnits + job.FailedUnits)
	if pending < 0 {
		pending = 0
	}

	return &CrawlStatusSummary{
		JobID:          job.ID,
		Status:         job.Status,
		TotalUnits:     job.TotalUnits,
		CompletedUnits: job.CompletedUnits,
		FailedUnits:    job.FailedUnits,
		PendingUnits:   pending,
		ProgressPct:    job.ProgressPercentage(),
		UpdatedAt:      job.UpdatedAt,
		Units:          briefs,
	}, nil
}

// FormatStatusReport formatea el estado en una salida visual legible para terminal.
func FormatStatusReport(s *CrawlStatusSummary) string {
	var sb strings.Builder
	sb.WriteString("=== Estado de Crawling Exhaustivo (Knowledge Crawl) ===\n\n")
	sb.WriteString(fmt.Sprintf("Job ID:       %s\n", s.JobID))
	sb.WriteString(fmt.Sprintf("Estado:       %s\n", s.Status))
	sb.WriteString(fmt.Sprintf("Progreso:     %.1f%% (%d/%d completadas)\n", s.ProgressPct, s.CompletedUnits, s.TotalUnits))
	sb.WriteString(fmt.Sprintf("Pendientes:   %d\n", s.PendingUnits))
	sb.WriteString(fmt.Sprintf("Fallidas:     %d\n", s.FailedUnits))
	sb.WriteString(fmt.Sprintf("Actualizado:  %s\n\n", s.UpdatedAt.Format("2006-01-02 15:04:05")))

	sb.WriteString("Desglose de Unidades:\n")
	for _, u := range s.Units {
		icon := "[ ]"
		switch u.Status {
		case UnitStatusCompleted:
			icon = "[✓]"
		case UnitStatusFailed:
			icon = "[✗]"
		case UnitStatusInProgress:
			icon = "[→]"
		}

		line := fmt.Sprintf("  %s %-30s (%s)", icon, u.ID, u.Status)
		if u.Error != "" {
			line += fmt.Sprintf(" - ERROR: %s", u.Error)
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
