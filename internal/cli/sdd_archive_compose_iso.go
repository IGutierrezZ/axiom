package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// refuseISOCompose es la tercera barrera del archivado ISO (design D13, REQ-29.10
// y REQ-29.11): tras `dependencies.archive` del status y `axiom sdd archive-check`,
// `archive-compose` se niega, antes de leer la canónica y sin escribir, a componer
// la delta de un cambio ISO que no cumple la regla del archivado (escenarios de
// todas las deltas y matriz de verify válida y de éxito).
//
// Determina el cambio con changeRootOfDelta, la misma función que usa la guarda A
// (guardComposeDelta). En el modo documento recibe el espacio ya resuelto y el
// dominio; en el modo --canonical (documentMode falso) localiza el espacio desde
// la carpeta de la delta y toma el dominio de esa carpeta, como
// guardComposeLooseDelta. Sin cambio determinable (una delta suelta, fuera de
// `changes/<cambio>/specs/<dominio>/`) no aplica nada, y un cambio no ISO nunca
// tiene hallazgos: su salida no cambia ni un byte (REQ-24.38). El modo
// `--change --in-place` no pasa por aquí.
func refuseISOCompose(loc workspace.Location, documentMode bool, deltaPath, domain string) error {
	if !documentMode {
		absolute, err := filepath.Abs(deltaPath)
		if err != nil {
			return nil
		}
		dir := filepath.Dir(absolute)
		located, err := workspace.Locate(dir)
		if err != nil {
			return nil
		}
		loc, deltaPath, domain = located, absolute, filepath.Base(dir)
	}
	changeRoot, ok := changeRootOfDelta(loc, deltaPath, domain)
	if !ok {
		return nil
	}
	findings := sddstatus.ISOArchiveRefusal(loc.WorkspaceRoot, changeRoot)
	if len(findings) == 0 {
		return nil
	}
	change := filepath.Base(changeRoot)
	reasons := make([]string, 0, len(findings))
	for _, finding := range findings {
		reasons = append(reasons, finding.Reason())
	}
	return fmt.Errorf("sdd-archive-compose: el cambio ISO %q no puede archivarse y no se compone ni se escribe nada: %s; corrígelo y comprueba con `axiom sdd archive-check --cwd %s --change %s`",
		change, strings.Join(reasons, " | "), pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(change))
}
