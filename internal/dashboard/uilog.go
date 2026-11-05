package dashboard

import (
	"os"
	"os/exec"
	"path/filepath"
)

// UILogPath devuelve la ruta de ui.log en el directorio de caché del usuario,
// creando el directorio. Si no se puede, devuelve os.DevNull. Lo comparten
// `axiom ui --detach` y el relanzamiento tras actualizar desde el dashboard.
func UILogPath() string {
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

// OpenUILog abre ui.log en modo añadir (o os.DevNull si no se puede) y devuelve el
// fichero y la ruta realmente usada.
func OpenUILog() (*os.File, string, error) {
	path := UILogPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		return f, path, nil
	}
	f, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	return f, os.DevNull, err
}

// startDetachedProcess lanza exe sin consola y desvinculado de este proceso, con la
// salida estándar y de error añadida a ui.log para no perder lo que escriba tras
// actualizarse. El canal recibe el resultado de Wait si el hijo termina; el hijo es
// independiente y sigue vivo cuando este proceso salga.
func startDetachedProcess(exe string, args []string, dir string) (<-chan error, error) {
	logFile, _, err := OpenUILog()
	if err != nil {
		return nil, err
	}
	defer logFile.Close() // el hijo conserva su propia copia del descriptor

	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	ApplyDetachAttrs(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}
