package cli

import (
	"fmt"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// projectIDPattern es el patrón de identificadores de requisito del proyecto que contiene
// loc, leído en vivo de su axiom.yaml (RF-RID-01). Sin declaración devuelve el defecto. Si
// axiom.yaml no se puede leer o su bloque `sdd:` es inválido, el patrón es inaccesible y el
// verbo `command` se niega con la continuación de `axiom workspace validate` (RF-RID-08):
// ninguno cae en silencio al defecto.
func projectIDPattern(command string, loc workspace.Location) (reqid.Pattern, error) {
	pattern, err := loc.Declarations.IDPattern()
	if err != nil {
		return reqid.Pattern{}, idPatternRefusal(command, loc, err)
	}
	return pattern, nil
}

// composeIDPattern es el patrón de `axiom sdd-archive-compose`: el del workspace ya resuelto
// de `--cwd` (haveLocation) o, sin él, el del workspace que contiene la delta, como
// refuseISOCompose. Una delta fuera de cualquier workspace usa el patrón por defecto.
func composeIDPattern(loc workspace.Location, haveLocation bool, deltaPath string) (reqid.Pattern, error) {
	if !haveLocation {
		absolute, err := filepath.Abs(deltaPath)
		if err != nil {
			return reqid.Default(), nil
		}
		located, err := workspace.Locate(filepath.Dir(absolute))
		if err != nil {
			return reqid.Default(), nil
		}
		loc = located
	}
	return projectIDPattern("sdd-archive-compose", loc)
}

// idPatternRefusal es el error de un verbo que no puede conocer el patrón: nombra la causa y
// la continuación `axiom workspace validate --path <proyecto>`.
func idPatternRefusal(command string, loc workspace.Location, err error) error {
	master := loc.Master
	if master == "" {
		master = loc.WorkspaceRoot
	}
	return fmt.Errorf("%s: el patrón de identificadores de requisito (sdd.id_pattern de axiom.yaml) no es accesible: %w; ejecuta `axiom workspace validate --path %s` y corrígelo", command, err, pathquote.Quote(master))
}
