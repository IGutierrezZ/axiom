// Package testenv reúne las comprobaciones de entorno que comparten las
// pruebas del repositorio: detectar una capacidad del sistema antes de
// depender de ella y escalar los presupuestos de tiempo. Es un paquete hoja
// solo de pruebas: ningún binario de producción lo importa y no importa ningún
// otro paquete interno.
package testenv

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
)

// errPrivilegeNotHeld es ERROR_PRIVILEGE_NOT_HELD (1314): lo que devuelve
// os.Symlink en Windows cuando el proceso no tiene
// SeCreateSymbolicLinkPrivilege (sin modo desarrollador ni administrador).
// errors.Is no lo traduce a os.ErrPermission, así que se compara el valor.
const errPrivilegeNotHeld = syscall.Errno(1314)

// symlinkSkipMessage nombra la capacidad que falta y cómo habilitarla.
const symlinkSkipMessage = "symlink creation requires privilege on Windows (error 1314); " +
	"enable Windows Developer Mode or run as administrator to execute this test"

type outcome int

const (
	outcomeRun outcome = iota
	outcomeSkip
	outcomeFail
)

// isSymlinkPrivilegeError informa de si err es el error 1314 de os.Symlink.
func isSymlinkPrivilegeError(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}
	var errno syscall.Errno
	return errors.As(linkErr.Err, &errno) && errno == errPrivilegeNotHeld
}

// decideSymlink traduce el resultado de la sonda en una decisión. Solo omite
// cuando la sonda prueba que falta el privilegio de Windows; cualquier otro
// error es un fallo real y nunca se convierte en omisión.
func decideSymlink(probeErr error, goos string) (outcome, string) {
	if probeErr == nil {
		return outcomeRun, ""
	}
	if goos == "windows" && isSymlinkPrivilegeError(probeErr) {
		return outcomeSkip, symlinkSkipMessage
	}
	return outcomeFail, "symlink capability probe failed unexpectedly: " + probeErr.Error()
}

var (
	symlinkProbeOnce sync.Once
	symlinkProbeErr  error
)

// probeSymlink intenta crear un enlace simbólico a un fichero en un directorio
// temporal propio, que limpia siempre.
func probeSymlink() error {
	dir, err := os.MkdirTemp("", "axiom-testenv-symlink-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("probe"), 0o600); err != nil {
		return err
	}
	return os.Symlink(target, filepath.Join(dir, "link"))
}

// RequireSymlink omite la prueba solo cuando la máquina no puede crear enlaces
// simbólicos por falta del privilegio de Windows (error 1314). Donde la
// capacidad existe la prueba se ejecuta; cualquier otro fallo de la sonda
// detiene la prueba con un error, nunca la omite. La sonda se ejecuta una vez
// por proceso.
func RequireSymlink(t testing.TB) {
	t.Helper()
	symlinkProbeOnce.Do(func() { symlinkProbeErr = probeSymlink() })
	switch decision, msg := decideSymlink(symlinkProbeErr, runtime.GOOS); decision {
	case outcomeSkip:
		t.Skip(msg)
	case outcomeFail:
		t.Fatal(msg)
	}
}
