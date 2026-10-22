package cli

// ConsoleInteractive dice si la consola de la orden tiene una terminal real en la entrada y en
// la salida de error y no hay marcador de CI: es el mismo criterio con el que las revisiones
// preguntan por el consentimiento (reviewConsoleInteractive). Lo exportan las órdenes de
// `cmd/axiom` que preguntan solo ante una persona (por ejemplo, las preferencias de worktree
// de `init` y `project add`): una pregunta que nadie puede contestar nunca debe convertirse en
// una lectura bloqueante.
func ConsoleInteractive() bool {
	return reviewConsoleInteractive()
}
