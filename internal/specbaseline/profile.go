package specbaseline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Profile es el perfil de requisito de un documento de la baseline, ya
// compilado (REQ-24.15): cómo se reconoce un requisito, qué lo identifica y
// cómo se emite su encabezado al integrar.
type Profile struct {
	Heading   *regexp.Regexp
	Level     int
	Identity  workspace.IdentityKind
	Render    string
	Scenarios workspace.ScenariosMode
}

// CompileProfile compila el perfil de un documento declarado en `axiom.yaml`.
// La declaración ya se valida al leerla (workspace.Declarations); aquí se
// vuelven a comprobar las condiciones de las que depende la composición, de modo
// que un documento sin perfil o con un patrón que no compila no llegue a
// componerse.
func CompileProfile(doc workspace.BaselineDocument) (Profile, error) {
	if doc.Requirement == nil {
		return Profile{}, fmt.Errorf("el documento del dominio '%s' no declara perfil de requisito ('requirement'): es un documento de prosa y no se compone", doc.Domain)
	}
	requirement := doc.Requirement
	heading, err := requirement.Compile()
	if err != nil {
		return Profile{}, fmt.Errorf("el dominio '%s': %w", doc.Domain, err)
	}
	if requirement.Level < 1 || requirement.Level > 6 {
		return Profile{}, fmt.Errorf("el dominio '%s': 'requirement.level' debe ser un entero entre 1 y 6, no %d", doc.Domain, requirement.Level)
	}
	if !strings.Contains(requirement.Render, "{id}") || !strings.Contains(requirement.Render, "{title}") {
		return Profile{}, fmt.Errorf("el dominio '%s': 'requirement.render' debe contener los marcadores {id} y {title}", doc.Domain)
	}
	return Profile{
		Heading:   heading,
		Level:     requirement.Level,
		Identity:  requirement.EffectiveIdentity(),
		Render:    requirement.Render,
		Scenarios: doc.EffectiveScenarios(),
	}, nil
}

// Match aplica el patrón `heading` a una línea de encabezado completa, sin su
// final de línea, y devuelve los grupos `id` y `title`. Un grupo que no
// participa en la coincidencia queda vacío.
func (p Profile) Match(line string) (id, title string, ok bool) {
	if p.Heading == nil {
		return "", "", false
	}
	indexes := p.Heading.FindStringSubmatchIndex(line)
	if indexes == nil {
		return "", "", false
	}
	return group(line, indexes, p.Heading.SubexpIndex("id")), group(line, indexes, p.Heading.SubexpIndex("title")), true
}

func group(line string, indexes []int, at int) string {
	if at < 0 || 2*at+1 >= len(indexes) || indexes[2*at] < 0 {
		return ""
	}
	return line[indexes[2*at]:indexes[2*at+1]]
}

// Key devuelve el identificador de un requisito según la identidad del perfil:
// su `id` (identidad `id`) o el texto de su `title` (identidad `text`).
func (p Profile) Key(id, title string) string {
	if p.Identity == workspace.IdentityText {
		return title
	}
	return id
}

// RenderHeading emite el encabezado de un requisito sustituyendo `{id}` y
// `{title}` en la plantilla `render`. La sustitución es de una sola pasada: un
// título que contenga el texto «{id}» no se vuelve a expandir.
func (p Profile) RenderHeading(id, title string) string {
	return strings.NewReplacer("{id}", id, "{title}", title).Replace(p.Render)
}

// HeadingLine reconstruye la línea de encabezado que corresponde al nombre de un
// requisito del delta (el texto tras `### Requirement:`): los `#` del nivel, un
// espacio y el nombre (REQ-24.39).
func (p Profile) HeadingLine(name string) string {
	return strings.Repeat("#", p.Level) + " " + name
}
