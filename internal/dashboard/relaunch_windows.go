package dashboard

import (
	"os/exec"
	"syscall"
)

// detachedProcess es DETACHED_PROCESS (winbase.h): el proceso hijo no hereda ni abre
// consola y sobrevive a la salida de este proceso.
const detachedProcess = 0x00000008

// startDetachedProcess lanza exe sin esperar a que termine y lo desvincula de este
// proceso y de su consola, para que siga vivo cuando el dashboard viejo salga.
func startDetachedProcess(exe string, args []string, dir string) (<-chan error, error) {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// El canal recibe el resultado de Wait si el hijo termina; el hijo es independiente
	// y sigue vivo cuando este proceso salga.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}
