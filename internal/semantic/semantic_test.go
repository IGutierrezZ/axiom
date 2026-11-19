package semantic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectorAgentConfigs(t *testing.T) {
	tempHome := t.TempDir()

	// Crear archivo mcp_config.json simulado para Antigravity con CodeGraph
	antigravityDir := filepath.Join(tempHome, ".gemini", "antigravity")
	if err := os.MkdirAll(antigravityDir, 0755); err != nil {
		t.Fatalf("error creando directorio de prueba: %v", err)
	}

	mcpContent := `{
		"mcpServers": {
			"codegraph": {
				"command": "codegraph",
				"args": ["serve"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(antigravityDir, "mcp_config.json"), []byte(mcpContent), 0644); err != nil {
		t.Fatalf("error escribiendo mcp_config.json: %v", err)
	}

	detector := NewDetectorWithCustomHome(tempHome, func(file string) (string, error) {
		if file == "codegraph" {
			return "/usr/local/bin/codegraph", nil
		}
		return "", os.ErrNotExist
	})

	agents := detector.DetectAgents()
	if len(agents) == 0 {
		t.Fatalf("se esperaban agentes detectados")
	}

	var antigravityAgent *AgentToolStatus
	for i := range agents {
		if agents[i].AgentName == "Google Antigravity" {
			antigravityAgent = &agents[i]
			break
		}
	}

	if antigravityAgent == nil || !antigravityAgent.Configured {
		t.Errorf("se esperaba que Google Antigravity estuviera configurado con codegraph")
	}

	if !detector.CheckCodeGraphAvailability(agents) {
		t.Errorf("se esperaba que CodeGraph estuviera disponible por PATH")
	}
}

func TestEngineExtractSymbolsAndDependencies(t *testing.T) {
	tempDir := t.TempDir()

	sampleGoCode := `// Package sample es un paquete de prueba para el motor semántico.
package sample

import (
	"context"
	"fmt"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RepositoryManager define la interfaz para gestionar repositorios.
type RepositoryManager interface {
	GetRepository(ctx context.Context, id string) (string, error)
	ListRepositories() []string
}

// Service implementa la lógica central.
type Service struct {
	workspaceRoot string
	active        bool
}

// NewService inicializa un nuevo servicio.
func NewService(root string) *Service {
	return &Service{workspaceRoot: root, active: true}
}

// Execute ejecuta una acción sobre el servicio.
func (s *Service) Execute(action string) (bool, error) {
	fmt.Println(action)
	return true, nil
}
`

	sampleFile := filepath.Join(tempDir, "service.go")
	if err := os.WriteFile(sampleFile, []byte(sampleGoCode), 0644); err != nil {
		t.Fatalf("error escribiendo archivo go de prueba: %v", err)
	}

	engine := NewEngine()
	res, err := engine.AnalyzeDirectory(tempDir, "core")
	if err != nil {
		t.Fatalf("error analizando directorio: %v", err)
	}

	if len(res.Packages) != 1 || res.Packages[0] != "sample" {
		t.Errorf("se esperaba paquete 'sample', se obtuvo %v", res.Packages)
	}

	// Comprobar símbolos extraídos
	symbolsByName := make(map[string]SymbolItem)
	for _, s := range res.Symbols {
		symbolsByName[s.Name] = s
	}

	// 1. Interfaz
	if iface, ok := symbolsByName["RepositoryManager"]; !ok {
		t.Errorf("no se extrajo la interfaz RepositoryManager")
	} else {
		if iface.Kind != KindInterface {
			t.Errorf("tipo incorrecto para RepositoryManager: %s", iface.Kind)
		}
		if !strings.Contains(iface.Signature, "interface") {
			t.Errorf("firma incorrecta para interfaz: %s", iface.Signature)
		}
	}

	// 2. Struct
	if str, ok := symbolsByName["Service"]; !ok {
		t.Errorf("no se extrajo el struct Service")
	} else {
		if str.Kind != KindStruct {
			t.Errorf("tipo incorrecto para Service: %s", str.Kind)
		}
	}

	// 3. Función
	if fn, ok := symbolsByName["NewService"]; !ok {
		t.Errorf("no se extrajo la función NewService")
	} else {
		if fn.Kind != KindFunc {
			t.Errorf("tipo incorrecto para NewService: %s", fn.Kind)
		}
		if !strings.Contains(fn.Signature, "func NewService") {
			t.Errorf("firma incorrecta para NewService: %s", fn.Signature)
		}
	}

	// 4. Método
	if meth, ok := symbolsByName["Execute"]; !ok {
		t.Errorf("no se extrajo el método Execute")
	} else {
		if meth.Kind != KindMethod {
			t.Errorf("tipo incorrecto para Execute: %s", meth.Kind)
		}
		if meth.Receiver != "*Service" {
			t.Errorf("receptor incorrecto para Execute: %s", meth.Receiver)
		}
	}

	// Comprobar dependencias de importación
	if len(res.Dependencies) != 3 {
		t.Errorf("se esperaban 3 dependencias, se obtuvieron %d", len(res.Dependencies))
	}

	var hasWorkspaceDep bool
	for _, dep := range res.Dependencies {
		if strings.Contains(dep.TargetPackage, "workspace") {
			hasWorkspaceDep = true
			if !dep.IsInternal {
				t.Errorf("la dependencia workspace debió ser clasificada como interna")
			}
		}
	}
	if !hasWorkspaceDep {
		t.Errorf("no se detectó la dependencia con workspace")
	}
}

// writeSemanticWorkspace crea un workspace mínimo con el conector indicado en axiom.yaml
// (cadena vacía = sin clave `semantic_analysis`).
func writeSemanticWorkspace(t *testing.T, connector string) string {
	t.Helper()
	tempWorkspace := t.TempDir()
	axiomYAML := `workspace:
  name: "TestProject"
  topology: "monorepo-embedded"
  specs_repository: "."
roles:
  core:
    name: "Equipo Core"
    repositories:
      - path: "."
`
	if connector != "" {
		axiomYAML += "governance:\n  semantic_analysis: \"" + connector + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(tempWorkspace, "axiom.yaml"), []byte(axiomYAML), 0644); err != nil {
		t.Fatalf("error escribiendo axiom.yaml: %v", err)
	}
	return tempWorkspace
}

func semanticStatusFor(t *testing.T, connector string, codeGraphInPath bool) *SemanticStatus {
	t.Helper()
	detector := NewDetectorWithCustomHome(t.TempDir(), func(file string) (string, error) {
		if codeGraphInPath && file == "codegraph" {
			return "/usr/local/bin/codegraph", nil
		}
		return "", os.ErrNotExist
	})
	srv := NewService(writeSemanticWorkspace(t, connector), detector, NewEngine())
	status, err := srv.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("error obteniendo status: %v", err)
	}
	return status
}

func TestServiceResolvesConnectorByPriority(t *testing.T) {
	cases := []struct {
		name       string
		connector  string
		codeGraph  bool
		wantActive ConnectorType
		wantWarn   bool
	}{
		{"auto con CodeGraph", "auto", true, ConnectorCodeGraph, false},
		{"auto sin CodeGraph", "auto", false, ConnectorNativeAST, false},
		{"sin valor con CodeGraph", "", true, ConnectorCodeGraph, false},
		{"sin valor sin CodeGraph", "", false, ConnectorNativeAST, false},
		{"codegraph disponible", "codegraph", true, ConnectorCodeGraph, false},
		{"codegraph no disponible", "codegraph", false, ConnectorNativeAST, true},
		{"native-ast fuerza el motor nativo", "native-ast", true, ConnectorNativeAST, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := semanticStatusFor(t, tc.connector, tc.codeGraph)
			if status.ActiveConnector != tc.wantActive {
				t.Errorf("conector activo = %s, se esperaba %s", status.ActiveConnector, tc.wantActive)
			}
			if (len(status.Warnings) > 0) != tc.wantWarn {
				t.Errorf("avisos = %v, se esperaba aviso=%v", status.Warnings, tc.wantWarn)
			}
		})
	}
}

func TestServiceDeprecatedSerenaValueLoadsAsAuto(t *testing.T) {
	for _, tc := range []struct {
		name       string
		value      string
		codeGraph  bool
		wantActive ConnectorType
	}{
		{"serena con CodeGraph", "serena", true, ConnectorCodeGraph},
		{"serena sin CodeGraph", "serena", false, ConnectorNativeAST},
		{"Serena en mayúsculas", "Serena", false, ConnectorNativeAST},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := semanticStatusFor(t, tc.value, tc.codeGraph)
			if string(status.ConfiguredConnector) != strings.ToLower(tc.value) {
				t.Errorf("conector configurado = %q, se esperaba el valor de axiom.yaml", status.ConfiguredConnector)
			}
			if status.ActiveConnector != tc.wantActive {
				t.Errorf("conector activo = %s, se esperaba %s", status.ActiveConnector, tc.wantActive)
			}
			if len(status.Warnings) != 1 || !strings.Contains(status.Warnings[0], "obsoleto") || !strings.Contains(status.Warnings[0], "serena") {
				t.Errorf("se esperaba un aviso de valor obsoleto, se obtuvo %v", status.Warnings)
			}
		})
	}
}

func TestSemanticStatusJSONHasNoSerena(t *testing.T) {
	status := semanticStatusFor(t, "auto", true)
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("error serializando el estado: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("error leyendo el JSON del estado: %v", err)
	}
	for key := range fields {
		if strings.Contains(strings.ToLower(key), "serena") {
			t.Errorf("el JSON del estado no debe exponer la clave %q", key)
		}
	}
}

func TestServiceFindSymbolsAndFallback(t *testing.T) {
	// Conector explícito 'codegraph' pero sin estar instalado
	tempWorkspace := writeSemanticWorkspace(t, "codegraph")

	// Crear código Go
	goCode := `package testpkg
type Handler struct{}
func NewHandler() *Handler { return &Handler{} }
`
	if err := os.WriteFile(filepath.Join(tempWorkspace, "handler.go"), []byte(goCode), 0644); err != nil {
		t.Fatalf("error escribiendo handler.go: %v", err)
	}

	// Detector sin CodeGraph
	detector := NewDetectorWithCustomHome(t.TempDir(), func(file string) (string, error) {
		return "", os.ErrNotExist
	})

	srv := NewService(tempWorkspace, detector, NewEngine())

	status, err := srv.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("error obteniendo status: %v", err)
	}

	if status.ConfiguredConnector != ConnectorCodeGraph {
		t.Errorf("se esperaba conector configurado 'codegraph', se obtuvo %s", status.ConfiguredConnector)
	}

	if status.ActiveConnector != ConnectorNativeAST {
		t.Errorf("se esperaba fallback activo a 'native-ast', se obtuvo %s", status.ActiveConnector)
	}

	if len(status.Warnings) == 0 {
		t.Errorf("se esperaba advertencia por fallback de conector no disponible")
	}

	if status.TotalSymbols == 0 {
		t.Errorf("se esperaba al menos 1 símbolo indexado")
	}

	// Test filtrado de símbolos
	symbols, err := srv.FindSymbols(SemanticQuery{Query: "Handler", Kind: KindStruct})
	if err != nil {
		t.Fatalf("error buscando símbolos: %v", err)
	}
	if len(symbols) != 1 || symbols[0].Name != "Handler" {
		t.Errorf("filtro no retornó el símbolo esperado: %v", symbols)
	}
}

func TestServiceReindexCodeGraph(t *testing.T) {
	tempWorkspace := t.TempDir()
	detector := NewDetectorWithCustomHome(t.TempDir(), func(file string) (string, error) {
		return "", os.ErrNotExist
	})
	srv := NewService(tempWorkspace, detector, NewEngine())

	res, err := srv.ReindexCodeGraph(context.Background())
	if err != nil {
		t.Fatalf("error inesperado en ReindexCodeGraph: %v", err)
	}
	if !res.Success {
		t.Errorf("se esperaba éxito en la respuesta de reindexación informativa")
	}
	if res.Connector != string(ConnectorNativeAST) {
		t.Errorf("connector = %q, se esperaba %q", res.Connector, ConnectorNativeAST)
	}
	if res.Message == "" {
		t.Errorf("se esperaba mensaje en el resultado de reindexación")
	}
	if res.Duration == "" {
		t.Errorf("se esperaba duración en el resultado de reindexación")
	}
}

func TestDetectorProjectWorkspaceConfigsAndDiagnostics(t *testing.T) {
	tempWorkspace := t.TempDir()
	tempHome := t.TempDir()

	// Simular .mcp.json a nivel de proyecto con codegraph
	mcpFile := filepath.Join(tempWorkspace, ".mcp.json")
	mcpContent := `{
		"mcpServers": {
			"codegraph": {
				"command": "codegraph",
				"args": ["serve"]
			}
		}
	}`
	if err := os.WriteFile(mcpFile, []byte(mcpContent), 0644); err != nil {
		t.Fatalf("error escribiendo .mcp.json: %v", err)
	}

	detector := NewDetectorWithCustomHome(tempHome, func(file string) (string, error) {
		return "", os.ErrNotExist
	})

	agents := detector.DetectAgents(tempWorkspace)
	var wsAgent *AgentToolStatus
	for i := range agents {
		if agents[i].Scope == "workspace" && strings.Contains(agents[i].AgentName, "Workspace MCP") {
			wsAgent = &agents[i]
			break
		}
	}

	if wsAgent == nil {
		t.Fatalf("se esperaba detectar agente a nivel de workspace")
	}
	if !wsAgent.Configured || !strings.Contains(wsAgent.Details, "codegraph") {
		t.Errorf("se esperaba que el agente de workspace tuviera codegraph configurado")
	}

	if detector.CheckCodeGraphInPath() {
		t.Errorf("no se esperaba CodeGraph en PATH")
	}
	if !detector.CheckCodeGraphConfigured(agents) {
		t.Errorf("se esperaba CodeGraph configurado en workspace")
	}
}
