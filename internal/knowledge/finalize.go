package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FinalizeResult contiene el balance final tras consolidar el crawl.
type FinalizeResult struct {
	JobID        string   `json:"job_id"`
	TotalSpecs   int      `json:"total_specs"`
	IndexUpdated bool     `json:"index_updated"`
	Domains      []string `json:"domains"`
}

// FinalizeCrawl reconcilia el índice maestro OpenSpec y valida la integridad de los artefactos.
func FinalizeCrawl(ctx context.Context, jobPath string) (*FinalizeResult, error) {
	job, err := LoadJob(jobPath)
	if err != nil {
		return nil, fmt.Errorf("error cargando job para finalizar: %w", err)
	}

	projectName := filepath.Base(job.WorkspaceRoot)

	// 1. Reconciliar openspec/INDEX.md
	if err := SyncIndex(projectName, job.SpecsRoot); err != nil {
		return nil, fmt.Errorf("error sincronizando catálogo openspec/INDEX.md: %w", err)
	}

	// 2. Comprobar dominios generados con éxito
	specsDir := filepath.Join(job.SpecsRoot, "specs")
	domains := make([]string, 0)
	if entries, err := os.ReadDir(specsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				specFile := filepath.Join(specsDir, e.Name(), "spec.md")
				if info, err := os.Stat(specFile); err == nil && info.Size() > 0 {
					domains = append(domains, e.Name())
				}
			}
		}
	}

	return &FinalizeResult{
		JobID:        job.ID,
		TotalSpecs:   len(domains),
		IndexUpdated: true,
		Domains:      domains,
	}, nil
}
