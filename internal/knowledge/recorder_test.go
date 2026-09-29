package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordUnitAndUpsertSpec(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")
	jobFile := filepath.Join(tempWs, "crawl-job.json")

	units := []CrawlUnit{
		{
			ID:                "auth/tokens",
			Domain:            "auth",
			Submodule:         "tokens",
			Path:              "internal/auth/tokens",
			Files:             []string{"internal/auth/tokens/jwt.go"},
			ExposedInterfaces: []string{"func GenerateToken() string"},
			Dependencies:      []string{"time", "crypto/rsa"},
			Status:            UnitStatusPending,
			SpecPath:          filepath.ToSlash(filepath.Join("openspec", "specs", "auth", "spec.md")),
		},
		{
			ID:        "auth/session",
			Domain:    "auth",
			Submodule: "session",
			Path:      "internal/auth/session",
			Files:     []string{"internal/auth/session/store.go"},
			Status:    UnitStatusPending,
			SpecPath:  filepath.ToSlash(filepath.Join("openspec", "specs", "auth", "spec.md")),
		},
	}

	job := NewCrawlJob("test-record-job", tempWs, specsRoot, units)
	if err := SaveJob(job, jobFile); err != nil {
		t.Fatalf("SaveJob falló: %v", err)
	}

	// 1. Inyectar unidad 1 (auth/tokens)
	payload1 := UnitAnalysisInput{
		UnitID:           "auth/tokens",
		Summary:          "Gestión de tokens criptográficos y validación de firma.",
		TechnicalDetails: "Implementa RS256 con claves rotadas en memoria.",
		Invariants: []string{
			"Todo token expirado debe ser rechazado inmediatamente.",
		},
		Requirements: []UnitRequirementItem{
			{
				ID:        "REQ-AUTH-01",
				Title:     "Verificación de Firma",
				Statement: "El módulo DEBE validar la firma antes de procesar cualquier claim.",
				Scenarios: []UnitScenarioItem{
					{
						Name:  "Firma inválida",
						Given: "un token con firma manipulada",
						When:  "se ejecuta Validate()",
						Then:  "retorna ErrInvalidSignature",
					},
				},
			},
		},
	}

	updatedJob, err := RecordUnit(context.Background(), jobFile, payload1)
	if err != nil {
		t.Fatalf("RecordUnit falló: %v", err)
	}

	if updatedJob.CompletedUnits != 1 {
		t.Errorf("esperado 1 unidad completada, obtenido %d", updatedJob.CompletedUnits)
	}

	targetSpec := filepath.Join(tempWs, "openspec", "specs", "auth", "spec.md")
	content, err := os.ReadFile(targetSpec)
	if err != nil {
		t.Fatalf("no se pudo leer la spec generada: %v", err)
	}
	specText := string(content)

	if !strings.Contains(specText, "## Componente: Tokens (`auth/tokens`)") {
		t.Errorf("la spec no contiene el encabezado del componente: %s", specText)
	}
	if !strings.Contains(specText, "REQ-AUTH-01") || !strings.Contains(specText, "ErrInvalidSignature") {
		t.Errorf("la spec no contiene los requisitos inyectados")
	}

	// 2. Inyectar unidad 2 (auth/session) en el mismo archivo spec.md
	payload2 := UnitAnalysisInput{
		UnitID:  "auth/session",
		Summary: "Gestión de sesiones distribuidas en Redis.",
		Invariants: []string{
			"Una sesión revocada no puede volver a activarse.",
		},
	}

	updatedJob2, err := RecordUnit(context.Background(), jobFile, payload2)
	if err != nil {
		t.Fatalf("RecordUnit 2 falló: %v", err)
	}

	if updatedJob2.CompletedUnits != 2 || updatedJob2.Status != JobStatusCompleted {
		t.Errorf("esperadas 2 unidades y status completed, obtenido: %s (%d/%d)", updatedJob2.Status, updatedJob2.CompletedUnits, updatedJob2.TotalUnits)
	}

	content2, _ := os.ReadFile(targetSpec)
	specText2 := string(content2)

	// Ambos componentes deben cohabitar en la misma especificación viva del dominio auth
	if !strings.Contains(specText2, "## Componente: Tokens (`auth/tokens`)") {
		t.Errorf("se perdió el componente Tokens tras agregar Session")
	}
	if !strings.Contains(specText2, "## Componente: Session (`auth/session`)") {
		t.Errorf("no se agregó el componente Session")
	}

	// 3. Probar reemplazo idempotente (actualizar auth/tokens)
	payload1Updated := payload1
	payload1Updated.Summary = "Versión actualizada de tokens con soporte Ed25519."
	_, err = RecordUnit(context.Background(), jobFile, payload1Updated)
	if err != nil {
		t.Fatalf("error actualizando unidad existentemente: %v", err)
	}

	content3, _ := os.ReadFile(targetSpec)
	specText3 := string(content3)
	if !strings.Contains(specText3, "Ed25519") {
		t.Errorf("la actualización idempotente no surtió efecto")
	}
	if strings.Count(specText3, "## Componente: Tokens") != 1 {
		t.Errorf("se duplicó la sección del componente en vez de actualizarse")
	}
}

func TestRecordUnitFailure(t *testing.T) {
	tempWs := t.TempDir()
	jobFile := filepath.Join(tempWs, "crawl-job.json")

	units := []CrawlUnit{
		{ID: "mod-fail", Status: UnitStatusPending},
	}
	job := NewCrawlJob("test-fail-job", tempWs, filepath.Join(tempWs, "openspec"), units)
	_ = SaveJob(job, jobFile)

	updated, err := RecordUnitFailure(context.Background(), jobFile, "mod-fail", "error de timeout en LLM")
	if err != nil {
		t.Fatalf("RecordUnitFailure falló: %v", err)
	}

	if updated.FailedUnits != 1 || updated.Status != JobStatusFailed {
		t.Errorf("esperado status %s con 1 fallo, obtenido: %s", JobStatusFailed, updated.Status)
	}

	u, _ := updated.FindUnit("mod-fail")
	if u.Error != "error de timeout en LLM" {
		t.Errorf("mensaje de error no guardado: %s", u.Error)
	}
}
