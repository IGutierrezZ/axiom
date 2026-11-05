//go:build !windows

package dashboard

import (
	"os/exec"
	"syscall"
)

// ApplyDetachAttrs inicia el hijo en una sesión nueva para que sobreviva al
// terminal y al proceso lanzador.
func ApplyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
