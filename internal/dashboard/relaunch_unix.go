//go:build !windows

package dashboard

import (
	"os/exec"
	"syscall"
)

// startDetachedProcess lanza exe sin esperar a que termine y en una sesión propia,
// para que siga vivo cuando el dashboard viejo salga.
func startDetachedProcess(exe string, args []string, dir string) error {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
