package engine

import (
	"context"
	"fmt"
	"path"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// intermediateMerge dice si la carpeta de un cambio presente en el árbol de la rama de
// integración es una fusión intermedia de planificación y no un cambio terminado (REQ-26.11,
// design DD-09): una PR de la salida 2 de una parada publicó parte del cambio, y fusionarla no
// significa que el cambio esté integrado ni listo para archivar.
//
// Solo lee lo que la propia salida 2 deja integrado, `gates.yaml` y `kickoff.yaml`, del árbol
// del commit sha ya resuelto de root, sin cambiar de rama y sin red (REQ-25.10). Es información:
// ante cualquier cosa que no se pueda leer o entender devuelve false, es decir, el cambio se
// trata como hasta ahora (REQ-26.11, supuesto A-8). Solo un fallo de git, que no es un problema
// del contenido, añade un aviso (frase en castellano) para que no pase inadvertido.
//
//  1. `gates.yaml` ausente o ilegible: false.
//  2. Ningún registro con la marca de cesión: false, sin leer nada más.
//  3. `kickoff.yaml` ausente o ilegible: false.
//  4. kickoff.IntermediateMerge decide con el roster del kickoff integrado.
func (e *Engine) intermediateMerge(ctx context.Context, root, sha, changesRel, change string) (bool, string) {
	changePath := path.Join(changesRel, change)
	read := func(file string) ([]byte, bool, string) {
		filePath := path.Join(changePath, file)
		content, exists, err := e.git.FileAt(ctx, root, sha, filePath)
		if err != nil {
			return nil, false, fmt.Sprintf("no se pudo leer %s en la rama de integración, así que %s se trata como un cambio sin fusión intermedia: %v", filePath, change, err)
		}
		return content, exists, ""
	}

	gatesContent, exists, warning := read(kickoff.GatesFileName)
	if warning != "" || !exists {
		return false, warning
	}
	ledger, err := kickoff.ParseGateLedger(gatesContent)
	if err != nil || !handedOffAnywhere(ledger) {
		return false, ""
	}

	kickoffContent, exists, warning := read(kickoff.KickoffFileName)
	if warning != "" || !exists {
		return false, warning
	}
	sealed, err := kickoff.ParseKickoff(kickoffContent)
	if err != nil {
		return false, ""
	}
	return kickoff.IntermediateMerge(sealed, ledger), ""
}

// handedOffAnywhere dice si algún registro aprobado lleva la marca de cesión: sin ella el
// cambio no pasó por la salida 2 y no hace falta leer el kickoff.
func handedOffAnywhere(ledger kickoff.GateLedger) bool {
	for _, record := range ledger.Records {
		if record.Decision == kickoff.DecisionApproved && record.HandedOff {
			return true
		}
	}
	return false
}
