package knowledge

import (
	"path/filepath"
	"testing"
)

func TestCrawlJobLifecycleAndProgress(t *testing.T) {
	tempDir := t.TempDir()
	jobFile := filepath.Join(tempDir, "crawl-job.json")

	units := []CrawlUnit{
		{
			ID:        "auth/token",
			Domain:    "auth",
			Submodule: "token",
			Path:      "internal/auth/token",
			Status:    UnitStatusPending,
		},
		{
			ID:        "billing/invoice",
			Domain:    "billing",
			Submodule: "invoice",
			Path:      "internal/billing/invoice",
			Status:    UnitStatusPending,
		},
	}

	job := NewCrawlJob("test-job-1", tempDir, filepath.Join(tempDir, "openspec"), units)

	if job.Status != JobStatusPending {
		t.Fatalf("esperado status %s, obtenido %s", JobStatusPending, job.Status)
	}
	if pct := job.ProgressPercentage(); pct != 0.0 {
		t.Fatalf("esperado progreso 0.0%%, obtenido %f", pct)
	}

	// 1. Iniciar unidad 1
	if err := job.UpdateUnitStatus("auth/token", UnitStatusInProgress, ""); err != nil {
		t.Fatalf("error actualizando status: %v", err)
	}
	if job.Status != JobStatusRunning {
		t.Errorf("esperado status %s tras inicio, obtenido %s", JobStatusRunning, job.Status)
	}

	// 2. Completar unidad 1
	if err := job.UpdateUnitStatus("auth/token", UnitStatusCompleted, ""); err != nil {
		t.Fatalf("error completando unidad: %v", err)
	}
	if pct := job.ProgressPercentage(); pct != 50.0 {
		t.Errorf("esperado progreso 50.0%%, obtenido %f", pct)
	}
	if job.Status != JobStatusRunning {
		t.Errorf("esperado status %s con 1 de 2 completada, obtenido %s", JobStatusRunning, job.Status)
	}

	// 3. Fallar unidad 2
	if err := job.UpdateUnitStatus("billing/invoice", UnitStatusFailed, "timeout al analizar AST"); err != nil {
		t.Fatalf("error fallando unidad: %v", err)
	}
	if job.Status != JobStatusFailed {
		t.Errorf("esperado status %s con fallo, obtenido %s", JobStatusFailed, job.Status)
	}
	if job.FailedUnits != 1 {
		t.Errorf("esperado 1 unidad fallida, obtenido %d", job.FailedUnits)
	}

	// 4. Probar persistencia atómica y recarga
	if err := SaveJob(job, jobFile); err != nil {
		t.Fatalf("SaveJob falló: %v", err)
	}

	loaded, err := LoadJob(jobFile)
	if err != nil {
		t.Fatalf("LoadJob falló: %v", err)
	}

	if loaded.ID != job.ID {
		t.Errorf("esperado ID %s, obtenido %s", job.ID, loaded.ID)
	}
	if loaded.CompletedUnits != 1 || loaded.FailedUnits != 1 {
		t.Errorf("contadores inconsistentes tras recargar: %+v", loaded)
	}

	unit, found := loaded.FindUnit("auth/token")
	if !found || unit.Status != UnitStatusCompleted {
		t.Errorf("unidad auth/token no encontrada o estado incorrecto: %+v", unit)
	}
}

func TestCrawlJobAllCompleted(t *testing.T) {
	units := []CrawlUnit{
		{ID: "mod1", Status: UnitStatusPending},
		{ID: "mod2", Status: UnitStatusPending},
	}
	job := NewCrawlJob("test-job-complete", ".", "openspec", units)

	_ = job.UpdateUnitStatus("mod1", UnitStatusCompleted, "")
	_ = job.UpdateUnitStatus("mod2", UnitStatusCompleted, "")

	if job.Status != JobStatusCompleted {
		t.Errorf("esperado status %s, obtenido %s", JobStatusCompleted, job.Status)
	}
	if pct := job.ProgressPercentage(); pct != 100.0 {
		t.Errorf("esperado progreso 100%%, obtenido %f", pct)
	}
}
