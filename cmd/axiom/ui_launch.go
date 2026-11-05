package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

// resolveAppBrowser localiza un navegador Chromium capaz de abrir una URL en
// modo aplicación (--app). Las dependencias se inyectan para poder probarlo sin
// tocar el sistema: getenv lee variables de entorno, exists comprueba ficheros y
// lookPath resuelve ejecutables en PATH.
func resolveAppBrowser(goos string, getenv func(string) string, exists func(string) bool, lookPath func(string) (string, error)) (string, bool) {
	if goos == "windows" {
		type candidate struct{ envVar, rel string }
		candidates := []candidate{
			{"ProgramFiles(x86)", `Microsoft\Edge\Application\msedge.exe`},
			{"ProgramFiles", `Microsoft\Edge\Application\msedge.exe`},
			{"LOCALAPPDATA", `Google\Chrome\Application\chrome.exe`},
			{"ProgramFiles", `Google\Chrome\Application\chrome.exe`},
			{"ProgramFiles(x86)", `Google\Chrome\Application\chrome.exe`},
		}
		for _, c := range candidates {
			root := getenv(c.envVar)
			if root == "" {
				continue
			}
			path := filepath.Join(root, c.rel)
			if exists(path) {
				return path, true
			}
		}
		return "", false
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"} {
		if path, err := lookPath(name); err == nil && path != "" {
			return path, true
		}
	}
	return "", false
}

// openAppWindow abre url en una ventana de aplicación sin barra de navegador.
// Si no hay un navegador compatible, cae al navegador por defecto con un aviso.
func openAppWindow(url string) error {
	exists := func(p string) bool {
		info, err := os.Stat(p)
		return err == nil && !info.IsDir()
	}
	if path, ok := resolveAppBrowser(runtime.GOOS, os.Getenv, exists, exec.LookPath); ok {
		return exec.Command(path, "--app="+url).Start()
	}
	fmt.Println("  (Aviso: no se encontró Edge ni Chrome; se abre el navegador por defecto)")
	return openBrowser(url)
}

// openUIWindow abre la interfaz según las banderas: nada con --no-browser,
// ventana de aplicación con --app y navegador por defecto en otro caso.
func openUIWindow(url string, app, noBrowser bool) {
	if noBrowser {
		return
	}
	open := openBrowser
	if app {
		open = openAppWindow
	}
	if err := open(url); err != nil {
		fmt.Printf("  (Aviso: No se pudo abrir el navegador automáticamente: %v)\n", err)
	}
}

// detachedChildArgs construye los argumentos del servidor desacoplado: nunca
// lleva --detach (evita un bucle de relanzamientos) ni --app (el servidor no abre ventanas).
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

	cmd := exec.Command(exe, detachedChildArgs(port, baseDir)...)
	cmd.Dir = baseDir
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
func runUIDetached(port int, baseDir string, app, noBrowser bool) int {
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
	openUIWindow(url, app, noBrowser)
	return 0
}

func errPortBusyMessage(port int) string {
	return fmt.Sprintf("[ERROR] El puerto %d está ocupado por otra aplicación. Usa `axiom ui --port <otro>`.", port)
}
