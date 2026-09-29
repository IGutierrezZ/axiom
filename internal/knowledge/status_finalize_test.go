package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrawlStatusAndFinalize(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")
	jobFile := filepath.Join(tempWs, "crawl-job.json")

	units := []CrawlUnit{
		{
			ID:        "auth/token",
			Domain:    "auth",
			Submodule: "token",
			Status:    UnitStatusCompleted,
			SpecPath:  "openspec/specs/auth/spec.md",
		},
		{
			ID:        "billing/invoice",
			Domain:    "billing",
			Submodule: "invoice",
			Status:    UnitStatusPending,
			SpecPath:  "openspec/specs/billing/spec.md",
		},
		{
			ID:        "catalog/items",
			Domain:    "catalog",
			Submodule: "items",
			Status:    UnitStatusFailed,
			Error:     "error de parseo sintáctico",
			SpecPath:  "openspec/specs/catalog/spec.md",
		},
	}

	job := NewCrawlJob("test-status-job", tempWs, specsRoot, units)
	if err := SaveJob(job, jobFile); err != nil {
		t.Fatalf("SaveJob falló: %v", err)
	}

	// 1. Probar GetCrawlStatus
	status, err := GetCrawlStatus(jobFile)
	if err != nil {
		t.Fatalf("GetCrawlStatus falló: %v", err)
	}

	if status.TotalUnits != 3 || status.CompletedUnits != 1 || status.FailedUnits != 1 || status.PendingUnits != 1 {
		t.Errorf("conteo incorrecto en status: %+v", status)
	}

	report := FormatStatusReport(status)
	if !strings.Contains(report, "auth/token") || !strings.Contains(report, "[✓]") || !strings.Contains(report, "[✗]") {
		t.Errorf("reporte formateado incompleto: %s", report)
	}

	// 2. Probar FinalizeCrawl
	// Crear una spec simulada para auth
	authSpecDir := filepath.Join(specsRoot, "specs", "auth")
	_ = os.MkdirAll(authSpecDir, 0755)
	_ = os.WriteFile(filepath.Join(authSpecDir, "spec.md"), []byte("# Especificación Viva: Auth\n\n### Requirement: Auth (REQ-01)\n"), 0644)

	res, err := FinalizeCrawl(context.Background(), jobFile)
	if err != nil {
		t.Fatalf("FinalizeCrawl falló: %v", err)
	}

	if !res.IndexUpdated || res.TotalSpecs != 1 || len(res.Domains) != 1 || res.Domains[0] != "auth" {
		t.Errorf("resultado de finalización inesperado: %+v", res)
	}

	// Verificar que openspec/INDEX.md fue generado
	indexPath := filepath.Join(specsRoot, "INDEX.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("no se creó INDEX.md: %v", err)
	}
	if !strings.Contains(string(indexBytes), "Catálogo Maestro") || !strings.Contains(string(indexBytes), "auth") {
		t.Errorf("contenido de INDEX.md no contiene la spec: %s", string(indexBytes))
	}
}
