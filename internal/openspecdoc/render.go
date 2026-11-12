package openspecdoc

import "strings"

// RenderNewSpec escribe la especificación de una capacidad que aún no existe a partir
// de los requisitos ADDED de una delta: el título, el propósito con el documento ODD
// de origen y los bloques, en su orden, separados por una línea en blanco. El
// resultado se puede componer después con Compose.
func RenderNewSpec(capability, origin string, added []Requirement) string {
	var b strings.Builder
	b.WriteString("# Especificación: " + capability + "\n\n")
	b.WriteString("## Propósito\n\n")
	b.WriteString("Integrada desde el documento ODD `" + origin + "`.\n\n")
	b.WriteString("## Requirements\n")
	for _, r := range added {
		b.WriteString("\n")
		b.WriteString(strings.TrimRight(strings.ReplaceAll(r.Body, "\r\n", "\n"), " \t\n"))
		b.WriteString("\n")
	}
	return b.String()
}
