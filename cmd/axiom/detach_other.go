//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// applyDetachAttrs inicia el hijo en una sesión nueva para que sobreviva al
// terminal y al proceso lanzador.
func applyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
