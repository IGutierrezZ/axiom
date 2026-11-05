//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const (
	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
)

// applyDetachAttrs desacopla el hijo de la consola y del grupo de procesos del
// lanzador. Se usa CREATE_NO_WINDOW en lugar de DETACHED_PROCESS porque crea una
// consola oculta que los procesos descendientes heredan, de modo que ninguno
// abre una ventana de consola; con DETACHED_PROCESS cualquier hijo con consola
// crearía la suya visible. CREATE_NEW_PROCESS_GROUP evita que un Ctrl+C del
// lanzador llegue al servidor.
func applyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow | createNewProcessGroup,
		HideWindow:    true,
	}
}
