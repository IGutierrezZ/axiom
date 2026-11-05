//go:build !windows

package dashboard

import (
	"os/exec"
	"syscall"
)

// startDetachedProcess lanza exe sin esperar a que termine y en una sesión propia,
// para que siga vivo cuando el dashboard viejo salga.
func startDetachedProcess(exe string, args []string, dir string) (<-chan error, error) {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// El canal recibe el resultado de Wait si el hijo termina; el hijo es independiente
	// y sigue vivo cuando este proceso salga.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}
