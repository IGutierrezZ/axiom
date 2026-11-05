package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/dashboard"
)

const (
	// detachWaitTimeout es el tiempo máximo que espera el proceso lanzador a que
	// el servidor desacoplado responda en /api/health.
	detachWaitTimeout = 10 * time.Second
	// detachPollInterval es el intervalo de sondeo mientras se espera al servidor.
	detachPollInterval = 200 * time.Millisecond
)

// openUIWindow abre el navegador por defecto en url, salvo con --no-browser.
func openUIWindow(url string, noBrowser bool) {
	if noBrowser {
		return
	}
	if err := openBrowser(url); err != nil {
		fmt.Printf("  (Aviso: No se pudo abrir el navegador automáticamente: %v)\n", err)
	}
}

// detachedChildArgs construye los argumentos del servidor desacoplado: nunca
// lleva --detach (evita un bucle de relanzamientos) y nunca abre el navegador.
func detachedChildArgs(port int, baseDir string) []string {
	return []string{"ui", "--no-browser", "--port", strconv.Itoa(port), "--path", baseDir}
}

// waitForAxiom sondea hasta que probe devuelva true, el contexto termine o se
// agote el timeout. Devuelve true si el servidor llegó a responder.
func waitForAxiom(ctx context.Context, probe func(context.Context) bool, interval, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if probe(ctx) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

// portIsFree comprueba rápidamente si 127.0.0.1:port admite una escucha.
func portIsFree(port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// uiLogPath devuelve la ruta de ui.log en el directorio de caché del usuario,
// creando el directorio. Si no se puede, devuelve os.DevNull.
func uiLogPath() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return os.DevNull
	}
	dir := filepath.Join(cache, "axiom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return os.DevNull
	}
	return filepath.Join(dir, "ui.log")
}

// serverWorkDir devuelve el directorio de trabajo del servidor de `axiom ui`:
// el home del usuario o, en su defecto, el temporal. Nunca el workspace: en
// Windows un proceso cuyo cwd está dentro de un directorio impide borrarlo, y un
// servidor lanzado desde un worktree haría fallar `git worktree remove`.
func serverWorkDir(baseDir string) string {
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home)
	}
	candidates = append(candidates, os.TempDir())
	return neutralWorkDir(baseDir, candidates...)
}

// neutralWorkDir elige el primer candidato no vacío que no esté dentro de
// baseDir (ni sea baseDir). Si ninguno sirve, devuelve la raíz del volumen de
// baseDir, que solo coincide con el workspace si este es la propia raíz.
func neutralWorkDir(baseDir string, candidates ...string) string {
	for _, c := range candidates {
		if c != "" && !isWithinDir(baseDir, c) {
			return c
		}
	}
	return filepath.VolumeName(baseDir) + string(filepath.Separator)
}

// isWithinDir indica si path es dir o está dentro de dir.
func isWithinDir(dir, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// newDetachedServerCommand construye el comando del servidor desacoplado. El
// workspace llega por --path; el cwd del hijo queda fuera de él.
func newDetachedServerCommand(exe string, port int, baseDir string) *exec.Cmd {
	cmd := exec.Command(exe, detachedChildArgs(port, baseDir)...)
	cmd.Dir = serverWorkDir(baseDir)
	return cmd
}

// startDetachedServer relanza este ejecutable como servidor sin consola y sin
// vínculo con el proceso actual. Devuelve la ruta del log donde escribe.
func startDetachedServer(port int, baseDir string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("no se pudo localizar el ejecutable: %w", err)
	}
	logPath := uiLogPath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logPath = os.DevNull
		if logFile, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0); err != nil {
			return "", err
		}
	}
	defer logFile.Close()

	cmd := newDetachedServerCommand(exe, port, baseDir)
	cmd.Stdin = nil // os/exec lo conecta a os.DevNull
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	applyDetachAttrs(cmd)
	if err := cmd.Start(); err != nil {
		return logPath, fmt.Errorf("no se pudo lanzar el servidor en segundo plano: %w", err)
	}
	// No se espera al hijo: se libera para que sobreviva a este proceso.
	_ = cmd.Process.Release()
	return logPath, nil
}

// runUIDetached lanza el servidor en segundo plano, espera a que responda y
// abre la ventana. Devuelve el código de salida del proceso lanzador.
func runUIDetached(port int, baseDir string, noBrowser bool) int {
	if !portIsFree(port) {
		fmt.Println(errPortBusyMessage(port))
		return 1
	}
	logPath, err := startDetachedServer(port, baseDir)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return 1
	}
	probe := func(ctx context.Context) bool {
		ok, _ := dashboard.ProbeAxiom(ctx, port)
		return ok
	}
	if !waitForAxiom(context.Background(), probe, detachPollInterval, detachWaitTimeout) {
		fmt.Printf("[ERROR] El servidor no respondió en %s. Revisa el registro: %s\n", detachWaitTimeout, logPath)
		return 1
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("Axiom activo en %s\n", url)
	openUIWindow(url, noBrowser)
	return 0
}

func errPortBusyMessage(port int) string {
	return fmt.Sprintf("[ERROR] El puerto %d está ocupado por otra aplicación. Usa `axiom ui --port <otro>`.", port)
}
