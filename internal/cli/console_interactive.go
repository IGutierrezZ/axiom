package cli

import (
	"os"
	"strings"
)

// ConsoleInteractive dice si la consola de la orden tiene una terminal real en la entrada y en
// la salida de error y no hay marcador de CI. Lo exportan las órdenes de `cmd/axiom` que
// preguntan solo ante una persona (por ejemplo, las preferencias de worktree de `init` y
// `project add`): una pregunta que nadie puede contestar nunca debe convertirse en una lectura
// bloqueante.
func ConsoleInteractive() bool {
	return consoleInteractive()
}

// consoleInteractive requires a real terminal on both ends and no CI
// marker. A question nobody can answer must never become a blocking read.
func consoleInteractive() bool {
	if strings.TrimSpace(os.Getenv("CI")) != "" {
		return false
	}
	return consoleTerminal(os.Stdin) && consoleTerminal(os.Stderr)
}

func consoleTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
