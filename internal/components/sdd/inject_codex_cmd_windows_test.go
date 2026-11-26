//go:build windows

package sdd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// runCodexHookInCmd ejecuta command como lo hace Codex en Windows (`cmd.exe /C "<orden>"`) con un
// PATH de Windows limpio, sin Git Bash ni ningún `true`, y devuelve el código de salida.
func runCodexHookInCmd(t *testing.T, command string) int {
	t.Helper()
	system32 := filepath.Join(os.Getenv("SystemRoot"), "System32")
	comspec := filepath.Join(system32, "cmd.exe")
	cmd := exec.Command(comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /C "` + command + `"`}
	cmd.Env = []string{"PATH=" + system32, "SystemRoot=" + os.Getenv("SystemRoot"), "ComSpec=" + comspec}
	cmd.Dir = t.TempDir()
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr):
		return exitErr.ExitCode()
	default:
		t.Fatalf("no se pudo lanzar cmd.exe: %v", err)
		return -1
	}
}

// Con `axiom` ausente del PATH, las formas `commandWindows` de los hooks de Codex terminan con 0
// en `cmd.exe` (RF-BRA-40), mientras que la forma POSIX `|| true` falla.
func TestCodexHookCommandWindowsExitsZeroInCmdWithoutAxiom(t *testing.T) {
	missing := func(command string) string {
		if !strings.HasPrefix(command, "axiom ") {
			t.Fatalf("la orden debe empezar por `axiom `: %q", command)
		}
		return "axiom-no-existe-fix-hook " + strings.TrimPrefix(command, "axiom ")
	}
	for name, command := range map[string]string{
		"autofetch":      AutofetchHookCommandWindows(model.AgentCodex),
		"skill-registry": codexSkillRegistryCommandWindows,
	} {
		if code := runCodexHookInCmd(t, missing(command)); code != 0 {
			t.Errorf("%s: %q salió con %d en cmd.exe, se esperaba 0", name, command, code)
		}
	}
	if code := runCodexHookInCmd(t, missing(AutofetchHookCommand(model.AgentCodex, "windows"))); code == 0 {
		t.Errorf("la forma POSIX `|| true` no debería salir con 0 en un cmd.exe sin `true`")
	}
}
