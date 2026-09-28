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

	// 4. Ejecutar runKnowledgeQuery sobre la regla no documentada en spec pero presente en código
	runKnowledgeQuery([]string{"-cwd", tempDir, "stripe_gateway_refund"})

	// Verificar que tras query la spec de payments fue enriquecida
	specData, err := os.ReadFile(paymentSpec)
	if err != nil {
		t.Fatalf("error leyendo spec tras query: %v", err)
	}
	if !strings.Contains(string(specData), "stripe_gateway_refund") {
		t.Errorf("la spec viva no fue enriquecida con la regla encontrada en código:\n%s", string(specData))
	}
}

func TestCLIHelpIncludesKnowledge(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(axiomHelpText())
	help := buf.String()

	if !strings.Contains(help, "knowledge sweep") {
		t.Errorf("la ayuda no contiene 'knowledge sweep'")
	}
	if !strings.Contains(help, "knowledge query") {
		t.Errorf("la ayuda no contiene 'knowledge query'")
	}
}
