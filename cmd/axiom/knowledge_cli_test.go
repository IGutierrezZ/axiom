package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInitKnowledgeProfile(t *testing.T) {
	tempDir := t.TempDir()

	// Probar runInit con --profile=knowledge
	runInit([]string{
		"-path", tempDir,
		"-name", "TestCliKnowledge",
		"-profile", "knowledge",
	})

	// Verificar axiom.yaml
	axiomPath := filepath.Join(tempDir, "axiom.yaml")
	data, err := os.ReadFile(axiomPath)
	if err != nil {
		t.Fatalf("error leyendo axiom.yaml generado por CLI: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `topology: "multirepo"`) {
		t.Errorf("se esperaba topology multirepo en axiom.yaml")
	}
	if !strings.Contains(content, "knowledge:") {
		t.Errorf("se esperaba rol knowledge en axiom.yaml")
	}

	// Verificar .mcp.json
	mcpPath := filepath.Join(tempDir, ".mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Errorf("se esperaba creación de .mcp.json")
	}

	// Verificar openspec/INDEX.md
	indexPath := filepath.Join(tempDir, "openspec", "INDEX.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("se esperaba creación de openspec/INDEX.md")
	}
}

func TestCLIKnowledgeSweepAndQuery(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Inicializar con perfil knowledge
	runInit([]string{
		"-path", tempDir,
		"-name", "TestApp",
		"-profile", "knowledge",
	})

	// 2. Crear un archivo de código
	svcDir := filepath.Join(tempDir, "internal", "payments")
	_ = os.MkdirAll(svcDir, 0755)
	code := `package payments

// ProcessRefund procesa devoluciones en pasarela stripe
func ProcessRefund(id string) error {
	// stripe_gateway_refund activo
	return nil
}
`
	_ = os.WriteFile(filepath.Join(svcDir, "refund.go"), []byte(code), 0644)

	// 3. Ejecutar runKnowledgeSweep
	runKnowledgeSweep([]string{"-cwd", tempDir, "-headless"})

	// Verificar que sweep creó la spec viva para payments
	paymentSpec := filepath.Join(tempDir, "openspec", "specs", "payments", "spec.md")
	if _, err := os.Stat(paymentSpec); os.IsNotExist(err) {
		t.Fatalf("sweep no creó %s", paymentSpec)
	}

	// 4. Ejecutar runKnowledgeQuery en modo solo lectura por defecto (no debe mutar spec)
	runKnowledgeQuery([]string{"-cwd", tempDir, "stripe_gateway_refund"})
	specDataBefore, err := os.ReadFile(paymentSpec)
	if err != nil {
		t.Fatalf("error leyendo spec: %v", err)
	}
	if strings.Contains(string(specDataBefore), "stripe_gateway_refund") {
		t.Errorf("la consulta de solo lectura no debió enriquecer la spec")
	}

	// 5. Ejecutar runKnowledgeQuery con bandera explícita -enrich
	runKnowledgeQuery([]string{"-cwd", tempDir, "-enrich", "stripe_gateway_refund"})

	// Verificar que tras query con -enrich la spec de payments fue enriquecida
	specData, err := os.ReadFile(paymentSpec)
	if err != nil {
		t.Fatalf("error leyendo spec tras query con enrich: %v", err)
	}
	if !strings.Contains(string(specData), "stripe_gateway_refund") {
		t.Errorf("la spec viva no fue enriquecida al pasar bandera -enrich:\n%s", string(specData))
	}
}

func TestCLIHelpIncludesKnowledge(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(axiomHelpText())
	help := buf.String()

	if !strings.Contains(help, "knowledge sweep") {
		t.Errorf("la ayuda no contiene 'knowledge sweep'")
	}
	if !strings.Contains(help, "knowledge crawl") {
		t.Errorf("la ayuda no contiene 'knowledge crawl'")
	}
	if !strings.Contains(help, "knowledge query") {
		t.Errorf("la ayuda no contiene 'knowledge query'")
	}
}

func TestCLIKnowledgeCrawlLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Inicializar con perfil knowledge
	runInit([]string{
		"-path", tempDir,
		"-name", "TestCrawlApp",
		"-profile", "knowledge",
	})

	// 2. Crear módulos de código simulados
	authDir := filepath.Join(tempDir, "internal", "auth")
	_ = os.MkdirAll(authDir, 0755)
	_ = os.WriteFile(filepath.Join(authDir, "jwt.go"), []byte("package auth\nfunc VerifyToken() bool { return true }\n"), 0644)

	jobFile := filepath.Join(tempDir, ".axiom", "knowledge", "crawl-job.json")

	// 3. Generar plan determinista
	runKnowledgeCrawl([]string{
		"-cwd", tempDir,
		"-job", jobFile,
		"-plan",
	})

	if _, err := os.Stat(jobFile); os.IsNotExist(err) {
		t.Fatalf("runKnowledgeCrawl -plan no generó %s", jobFile)
	}

	// 4. Consultar status
	runKnowledgeCrawl([]string{
		"-job", jobFile,
		"-status",
	})

	// 5. Registrar unidad auth
	analysisPayload := `{
		"unit_id": "auth",
		"summary": "Módulo de autenticación con verificación de tokens.",
		"technical_details": "Algoritmo HMAC-SHA256 con claves rotadas.",
		"invariants": ["Firmas inválidas causan rechazo 401"],
		"requirements": [
			{
				"id": "REQ-AUTH-01",
				"title": "Verificación JWT",
				"statement": "El sistema DEBE rechazar tokens manipulados.",
				"scenarios": [
					{
						"name": "Token adulterado",
						"given": "un token con firma corrupta",
						"when": "se valida",
						"then": "retorna false"
					}
				]
			}
		]
	}`
	payloadFile := filepath.Join(tempDir, "auth-analysis.json")
	_ = os.WriteFile(payloadFile, []byte(analysisPayload), 0644)

	runKnowledgeCrawl([]string{
		"-job", jobFile,
		"-record-unit",
		"-unit", "auth",
		"-input", payloadFile,
	})

	// Verificar que la spec viva de auth fue creada y contiene los requisitos
	authSpec := filepath.Join(tempDir, "openspec", "specs", "auth", "spec.md")
	data, err := os.ReadFile(authSpec)
	if err != nil {
		t.Fatalf("no se creó la spec viva en %s: %v", authSpec, err)
	}
	specStr := string(data)
	if !strings.Contains(specStr, "REQ-AUTH-01") || !strings.Contains(specStr, "Token adulterado") {
		t.Errorf("la spec viva no contiene los datos registrados:\n%s", specStr)
	}

	// 6. Finalizar crawl
	runKnowledgeCrawl([]string{
		"-job", jobFile,
		"-finalize",
	})

	// Verificar catálogo openspec/INDEX.md
	indexFile := filepath.Join(tempDir, "openspec", "INDEX.md")
	indexBytes, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatalf("error leyendo INDEX.md tras finalize: %v", err)
	}
	if !strings.Contains(string(indexBytes), "auth") {
		t.Errorf("INDEX.md no incluye el dominio auth finalizado:\n%s", string(indexBytes))
	}
}
