package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCrawlPlan(t *testing.T) {
	tempWs := t.TempDir()

	// 1. Crear estructura simulada con submódulos
	cmdServer := filepath.Join(tempWs, "cmd", "server")
	_ = os.MkdirAll(cmdServer, 0755)
	_ = os.WriteFile(filepath.Join(cmdServer, "main.go"), []byte("package main\nimport \"fmt\"\nfunc Main() { fmt.Println(\"ok\") }\n"), 0644)

	cmdWorker := filepath.Join(tempWs, "cmd", "worker")
	_ = os.MkdirAll(cmdWorker, 0755)
	_ = os.WriteFile(filepath.Join(cmdWorker, "worker.go"), []byte("package main\nfunc RunWorker() {}\n"), 0644)

	// 2. Módulo simple sin submódulos
	internalAuth := filepath.Join(tempWs, "internal", "auth")
	_ = os.MkdirAll(internalAuth, 0755)
	_ = os.WriteFile(filepath.Join(internalAuth, "token.go"), []byte(`package auth

import "time"

type TokenManager interface {
	Generate() string
}

type AuthConfig struct {
	TTL time.Duration
}

func NewAuthService() *AuthConfig {
	return &AuthConfig{}
}
`), 0644)

	specsRoot := filepath.Join(tempWs, "openspec")
	jobFile := filepath.Join(tempWs, ".axiom", "knowledge", "crawl-job.json")

	job, err := GenerateCrawlPlan(context.Background(), CrawlPlanOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		JobPath:       jobFile,
		Force:         true,
	})
	if err != nil {
		t.Fatalf("GenerateCrawlPlan falló: %v", err)
	}

	if job.TotalUnits < 3 {
		t.Fatalf("se esperaban al menos 3 unidades (cmd/server, cmd/worker, auth), obtenido: %d", job.TotalUnits)
	}

	// Verificar unidad auth
	authUnit, found := job.FindUnit("auth")
	if !found {
		t.Fatalf("unidad 'auth' no encontrada en el plan: %+v", job.Units)
	}
	if len(authUnit.Files) == 0 {
		t.Errorf("unidad auth no tiene archivos asociados")
	}

	// Verificar interfaces expuestas extraídas mediante AST
	hasTokenManager := false
	hasAuthConfig := false
	hasNewAuthService := false
	for _, iface := range authUnit.ExposedInterfaces {
		if iface == "interface TokenManager (interface (1 métodos))" {
			hasTokenManager = true
		}
		if iface == "struct AuthConfig (struct (1 campos))" {
			hasAuthConfig = true
		}
		if iface == "func NewAuthService (func NewAuthService() *AuthConfig)" {
			hasNewAuthService = true
		}
	}

	if !hasTokenManager || !hasAuthConfig || !hasNewAuthService {
		t.Errorf("no se extrajeron correctamente los símbolos AST públicos: %v", authUnit.ExposedInterfaces)
	}

	// Verificar persistencia y reanudación
	reloaded, err := GenerateCrawlPlan(context.Background(), CrawlPlanOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		JobPath:       jobFile,
		Force:         false,
	})
	if err != nil {
		t.Fatalf("error recargando plan existente: %v", err)
	}
	if reloaded.ID != job.ID {
		t.Errorf("esperado mismo job ID al reanudar, obtenido %s vs %s", job.ID, reloaded.ID)
	}
}
