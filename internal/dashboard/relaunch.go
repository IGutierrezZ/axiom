package dashboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Costuras del relanzamiento. Las pruebas las sustituyen para que nunca se lance un
// proceso real, se resuelva el ejecutable del que corre `go test` ni se llame a os.Exit.
var (
	executablePathFn = os.Executable
	lookPathFn       = exec.LookPath
	fileExistsFn     = func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && !info.IsDir()
	}
	startProcessFn = startDetachedProcess
	exitProcessFn  = os.Exit

	// relaunchDelay deja que la interfaz vea la fase «restarting» antes de que el
	// servidor viejo deje de responder.
	relaunchDelay = 800 * time.Millisecond
	// relaunchLiveCheck es lo que se espera tras lanzar el proceso nuevo para comprobar
	// que no ha muerto al instante (ejecutable borrado o dañado, banderas rechazadas).
	// Se mira antes de cerrar el servidor viejo: si el hijo ya murió, el dashboard
	// viejo sigue sirviendo y la UI recibe el comando manual. El hijo reintenta enlazar
	// el puerto durante 10 s, así que esta ventana no lo mata por el puerto.
	relaunchLiveCheck = 1500 * time.Millisecond
	// shutdownTimeout acota el cierre ordenado del servidor viejo.
	shutdownTimeout = 3 * time.Second
)

// selfBinaryName es el nombre con el que se busca el ejecutable en el PATH cuando la
// ruta de os.Executable ya no sirve.
const selfBinaryName = "axiom"

// resolveRelaunchExecutable devuelve el ejecutable que debe lanzarse tras
// actualizar. Normalmente es el propio (os.Executable). En Windows, `go install`
// aparta el binario en uso con otro nombre (`axiom.exe~`) antes de escribir el
// nuevo: si la ruta devuelta ya no existe o apunta al apartado, se usa el `axiom`
// del PATH, que la comprobación previa de la actualización garantiza que es el
// mismo destino donde se ha escrito la versión nueva.
func resolveRelaunchExecutable() (string, error) {
	exe, err := executablePathFn()
	if err == nil && exe != "" && !strings.HasSuffix(exe, "~") && !strings.HasSuffix(strings.ToLower(exe), ".old") && fileExistsFn(exe) {
		return exe, nil
	}
	if path, lookErr := lookPathFn(selfBinaryName); lookErr == nil && path != "" {
		return path, nil
	}
	if err == nil {
		err = errors.New("el ejecutable actual ya no existe en disco")
	}
	return "", fmt.Errorf("no se encontró el ejecutable de Axiom para reiniciar: %w", err)
}

// relaunchArgs son los argumentos con los que el binario nuevo vuelve a servir el
// dashboard: mismo workspace y mismo puerto, sin abrir otra pestaña, y con la
// sincronización posterior a la actualización.
func relaunchArgs(root string, port int) []string {
	return []string{"ui", "--path", root, "--port", strconv.Itoa(port), "--no-browser", "--post-update-sync"}
}

// manualRelaunchHint es el comando para volver a abrir el dashboard a mano.
func manualRelaunchHint(root string) string {
	// Comillas dobles simples (no %q): en Windows %q duplicaría las barras invertidas
	// y la orden dejaría de ser copiable.
	return "axiom ui --path \"" + root + "\""
}

// relaunchAfterUpdate se ejecuta cuando la autoactualización terminó bien: lanza el
// binario nuevo sobre el mismo puerto, cierra este servidor de forma ordenada y sale.
// Si no puede lanzarlo, el dashboard viejo sigue sirviendo y la UI recibe el motivo
// y el comando manual.
//
// Orden: primero se lanza el proceso nuevo y después se libera el puerto; el nuevo
// reintenta enlazarlo unos segundos (ListenAndServeExact), de modo que no hace falta
// que el viejo haya salido ya.
func (s *Server) relaunchAfterUpdate(sync *UpdateSyncSummary) {
	j := &s.service.update
	root := s.service.getRootPath()
	hint := manualRelaunchHint(root)

	exe, err := resolveRelaunchExecutable()
	if err != nil {
		j.finish(updatePhaseFailed, "Axiom se ha actualizado, pero no se pudo reiniciar el dashboard: "+err.Error(), hint, sync)
		return
	}

	time.Sleep(relaunchDelay)
	exited, err := startProcessFn(exe, relaunchArgs(root, s.port), root)
	if err != nil {
		j.finish(updatePhaseFailed, "Axiom se ha actualizado, pero no se pudo lanzar la versión nueva del dashboard: "+err.Error(), hint, sync)
		return
	}
	select {
	case werr := <-exited:
		reason := "terminó nada más arrancar"
		if werr != nil {
			reason = "terminó nada más arrancar: " + werr.Error()
		}
		j.finish(updatePhaseFailed, "Axiom se ha actualizado, pero la versión nueva del dashboard "+reason+". Este dashboard sigue activo.", hint, sync)
		return
	case <-time.After(relaunchLiveCheck):
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = s.Shutdown(ctx)
	exitProcessFn(0)
}
