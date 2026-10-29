package dashboard

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestDashboardPreview es un arnés de vista previa, no una prueba: solo se activa
// con AXIOM_DASHBOARD_PREVIEW_PORT. Levanta el servidor real del dashboard
// (NewServer(NewService(raíz))) y se bloquea hasta que el proceso se interrumpe,
// para revisar la interfaz en un navegador sin compilar axiom.exe (el EDR borra
// los binarios sueltos). Uso:
//
//	AXIOM_DASHBOARD_PREVIEW_PORT=4317 go test ./internal/dashboard -run TestDashboardPreview -count=1 -timeout 0 -v
//
// Variables:
//   - AXIOM_DASHBOARD_PREVIEW_PORT: puerto de partida (obligatorio para activarlo).
//     ListenAndServe prueba hasta 10 puertos siguientes si está ocupado; la URL
//     real se imprime en la salida estándar.
//   - AXIOM_DASHBOARD_PREVIEW_ROOT: raíz del proyecto a servir. Por defecto, la
//     raíz del repositorio de este árbol de trabajo.
//
// TestMain aísla HOME en un directorio temporal, así que el Hub del arnés nace
// vacío: se registran en él la raíz servida y un proyecto vacío («sin configurar»)
// para poder revisar el selector de proyecto y la pantalla zero-config. Todo ello
// vive en el HOME temporal y desaparece con el proceso.
func TestDashboardPreview(t *testing.T) {
	portEnv := os.Getenv("AXIOM_DASHBOARD_PREVIEW_PORT")
	if portEnv == "" {
		t.Skip("arnés de vista previa: defina AXIOM_DASHBOARD_PREVIEW_PORT para activarlo")
	}
	port, err := strconv.Atoi(portEnv)
	if err != nil || port <= 0 || port > 65535 {
		t.Fatalf("AXIOM_DASHBOARD_PREVIEW_PORT=%q no es un puerto válido", portEnv)
	}

	root := os.Getenv("AXIOM_DASHBOARD_PREVIEW_ROOT")
	if root == "" {
		root = previewRepoRoot(t)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolver la raíz de la vista previa: %v", err)
	}

	// TestMain acota el descubrimiento de workspace al directorio temporal; la
	// vista previa sirve un proyecto real y no debe heredar ese techo.
	workspace.DiscoveryCeilingOverride = nil

	svc := NewService(root)
	if hubMgr := svc.GetHubManager(); hubMgr != nil {
		if _, err := hubMgr.Register(root, filepath.Base(root), ""); err != nil {
			t.Logf("no se pudo registrar la raíz en el Hub temporal: %v", err)
		}
		empty := filepath.Join(t.TempDir(), "demo-sin-config")
		if err := os.MkdirAll(empty, 0o755); err == nil {
			if _, err := hubMgr.Register(empty, "demo-sin-config", ""); err != nil {
				t.Logf("no se pudo registrar el proyecto vacío: %v", err)
			}
		}
	}

	server := NewServer(svc)
	actual, err := server.ListenAndServe(port)
	if err != nil {
		t.Fatalf("iniciar la vista previa: %v", err)
	}
	fmt.Printf("Vista previa del dashboard: http://127.0.0.1:%d/ (raíz: %s)\n", actual, root)
	fmt.Println("Interrumpe el proceso (Ctrl+C o taskkill) para terminar.")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}

// previewRepoRoot localiza la raíz del repositorio subiendo desde el directorio
// del test (internal/dashboard) hasta encontrar go.mod.
func previewRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("directorio actual: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no se encontró go.mod subiendo desde el directorio del test")
		}
		dir = parent
	}
}
