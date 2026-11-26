package main

import (
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

// C-08-l retiró `axiom knowledge` (sweep, crawl, query) y `axiom semantic`: la
// ayuda ya no los anuncia y main los trata como un comando desconocido. El perfil
// `knowledge` de `axiom init` se queda.
func TestCLIHelpOmitsRetiredKnowledgeAndSemantic(t *testing.T) {
	help := axiomHelpText()
	for _, retired := range []string{"knowledge sweep", "knowledge crawl", "knowledge query", "semantic status", "semantic symbols", "semantic inspect", "semantic reindex"} {
		if strings.Contains(help, retired) {
			t.Errorf("la ayuda todavía anuncia %q", retired)
		}
	}
}
