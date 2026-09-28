package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSweepAutonomousHeadless(t *testing.T) {
	tempWs := t.TempDir()

	// Crear estructura de código simulada
	goMod := filepath.Join(tempWs, "go.mod")
	_ = os.WriteFile(goMod, []byte("module example.com/testapp\ngo 1.25\n"), 0644)

	cmdDir := filepath.Join(tempWs, "cmd", "server")
	_ = os.MkdirAll(cmdDir, 0755)
	_ = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	billingDir := filepath.Join(tempWs, "internal", "billing")
	_ = os.MkdirAll(billingDir, 0755)
	_ = os.WriteFile(filepath.Join(billingDir, "invoice.go"), []byte("package billing\ntype Invoice struct{}\n"), 0644)

	authDir := filepath.Join(tempWs, "internal", "auth")
	_ = os.MkdirAll(authDir, 0755)
	_ = os.WriteFile(filepath.Join(authDir, "token.go"), []byte("package auth\ntype Token struct{}\n"), 0644)

	// Simular conflicto de lockfiles para probar reporte de ambigüedades sin bloqueo
	_ = os.WriteFile(filepath.Join(tempWs, "package-lock.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(tempWs, "yarn.lock"), []byte(""), 0644)

	specsRoot := filepath.Join(tempWs, "openspec")
	res, err := RunSweep(context.Background(), SweepOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		Headless:      true,
	})
	if err != nil {
		t.Fatalf("RunSweep falló: %v", err)
	}

	// 1. Verificación técnica
	if res.PrimaryLanguage != "go" {
		t.Errorf("esperado PrimaryLanguage=go, obtenido: %s", res.PrimaryLanguage)
	}
	if len(res.Entrypoints) == 0 || !strings.Contains(res.Entrypoints[0], "main.go") {
		t.Errorf("se esperaba entrypoint cmd/server/main.go, obtenido: %v", res.Entrypoints)
	}

	// 2. Verificación de módulos funcionales
	foundBilling, foundAuth := false, false
	for _, m := range res.Modules {
		if m.Name == "billing" {
			foundBilling = true
		}
		if m.Name == "auth" {
			foundAuth = true
		}
	}
	if !foundBilling || !foundAuth {
		t.Errorf("se esperaban módulos billing y auth, obtenidos: %v", res.Modules)
	}

	// 3. Verificación de ambigüedades no bloqueantes
	if len(res.Ambiguities) == 0 {
		t.Errorf("se esperaba al menos una ambigüedad detectada (múltiples lockfiles)")
	}

	// 4. Verificación de specs creadas e INDEX.md
	if _, err := os.Stat(filepath.Join(specsRoot, "specs", "billing", "spec.md")); os.IsNotExist(err) {
		t.Errorf("no se creó openspec/specs/billing/spec.md")
	}
	if _, err := os.Stat(filepath.Join(specsRoot, "specs", "auth", "spec.md")); os.IsNotExist(err) {
		t.Errorf("no se creó openspec/specs/auth/spec.md")
	}

	indexPath := filepath.Join(specsRoot, "INDEX.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("error leyendo INDEX.md: %v", err)
	}
	indexStr := string(indexBytes)
	if !strings.Contains(indexStr, "`billing`") || !strings.Contains(indexStr, "`auth`") {
		t.Errorf("INDEX.md no contiene los dominios detectados:\n%s", indexStr)
	}
}

func TestQuerySpecFirstCache(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")
	billingSpecDir := filepath.Join(specsRoot, "specs", "billing")
	_ = os.MkdirAll(billingSpecDir, 0755)

	specContent := `# Especificación Viva: Facturación

### Requirement: Política de Reembolsos Estricta (REQ-01)
Los reembolsos deben procesarse exclusivamente dentro de los primeros 14 días naturales.

#### Scenario: Solicitud a tiempo
- **DADO** un pago válido
- **CUANDO** se pide reembolso antes de 14 días
- **ENTONCES** se aprueba de inmediato
`
	_ = os.WriteFile(filepath.Join(billingSpecDir, "spec.md"), []byte(specContent), 0644)
	_ = SyncIndex("TestProject", specsRoot)

	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		Question:      "¿Cómo funciona la política de reembolsos?",
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if !res.ResolvedFromSpec {
		t.Errorf("se esperaba ResolvedFromSpec=true")
	}
	if res.SpecUpdated {
		t.Errorf("no se debió actualizar la spec cuando la respuesta ya existía")
	}
	if !strings.Contains(res.DirectAnswer, "14 días") {
		t.Errorf("la respuesta directa no contiene el texto de la spec: %s", res.DirectAnswer)
	}
}

func TestQueryDeepInspectionAndSpecEnrichment(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")

	// Crear código con una regla no documentada
	shippingDir := filepath.Join(tempWs, "internal", "shipping")
	_ = os.MkdirAll(shippingDir, 0755)
	codeContent := `package shipping

// CalculateRate aplica la regla tarifa_plana_envio de 5 euros
func CalculateRate(weight float64) float64 {
	// Regla interna: tarifa_plana_envio fija
	return 5.0
}
`
	_ = os.WriteFile(filepath.Join(shippingDir, "calculator.go"), []byte(codeContent), 0644)

	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		Question:      "¿Qué es la tarifa_plana_envio?",
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if res.ResolvedFromSpec {
		t.Errorf("no se debió resolver desde la spec ya que estaba vacía")
	}
	if !res.SpecUpdated {
		t.Errorf("se esperaba SpecUpdated=true tras auto-enriquecimiento")
	}
	if len(res.Evidences) == 0 {
		t.Errorf("se esperaban evidencias de código")
	}

	// Verificar que se creó la spec viva de shipping
	specPath := filepath.Join(specsRoot, "specs", "shipping", "spec.md")
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("error leyendo spec enriquecida %s: %v", specPath, err)
	}
	specStr := string(specBytes)
	if !strings.Contains(specStr, "Requirement:") || !strings.Contains(specStr, "tarifa_plana_envio") {
		t.Errorf("contenido inesperado en la spec enriquecida:\n%s", specStr)
	}

	// Verificar que INDEX.md refleja el nuevo dominio
	indexPath := filepath.Join(specsRoot, "INDEX.md")
	indexBytes, _ := os.ReadFile(indexPath)
	if !strings.Contains(string(indexBytes), "`shipping`") {
		t.Errorf("INDEX.md no incluye el dominio enriquecido 'shipping':\n%s", string(indexBytes))
	}
}
