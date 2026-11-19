package agentbuilder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

// validSkillName es un único segmento de ruta seguro: Install lo usa como
// nombre de directorio y como `name:` del frontmatter.
var validSkillName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// composeSkillMD devuelve el SKILL.md que se instala. El LLM genera el cuerpo
// sin frontmatter (`# {Title}`...), así que el código compone el bloque a
// partir de Name, Description y Trigger, con la descripción en una línea y
// entrecomillada. Si el contenido ya trae frontmatter se respeta tal cual y
// la validación decide.
func composeSkillMD(agent *GeneratedAgent) string {
	if hasFrontmatter(agent.Content) {
		return agent.Content
	}
	desc := strings.TrimRight(oneLine(agent.Description), ". ")
	trigger := oneLine(agent.Trigger)
	text := desc
	if trigger != "" {
		if text != "" {
			text += ". "
		}
		text += "Trigger: " + trigger
	}
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", agent.Name, strconv.Quote(text), agent.Content)
}

func hasFrontmatter(content string) bool {
	return strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n")
}

// oneLine colapsa cualquier espacio en blanco (saltos de línea incluidos) a un
// único espacio.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// validateSkill comprueba el nombre y el SKILL.md compuesto con el perfil
// portable de skillcheck (el frontmatter lo compone el código, no un tercero).
// Solo los errores bloquean.
func validateSkill(agent *GeneratedAgent, content string) error {
	if !validSkillName.MatchString(agent.Name) {
		return fmt.Errorf("install: nombre de skill no válido %q: debe ser un único segmento de ruta (letras, dígitos, '.', '_' o '-')", agent.Name)
	}
	findings := skillcheck.CheckContent(content, agent.Name, skillcheck.Options{Profile: skillcheck.Portable})
	if msg := skillcheck.FormatErrors(findings); msg != "" {
		return fmt.Errorf("install: la skill %q no supera la validación y no se escribió nada:\n%s", agent.Name, msg)
	}
	return nil
}
