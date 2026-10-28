package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
)

// `axiom branch guard` (INC-25, S11, REQ-25.21, design D-18): el hook de los runtimes con hooks
// que `axiom sync` instala. La decisión vive en el motor (internal/branch/engine); aquí solo están
// las banderas, la ayuda y la entrega al motor de la carga que el runtime envía por la entrada
// estándar. Es la única orden de la familia pensada para que la ejecute un runtime y no una persona.

const branchGuardHelp = `Uso: axiom branch guard --agent <runtime>

Hook que instala axiom sync en los runtimes con hooks. Lee de la entrada estándar la carga que el
runtime envía antes (evento pre) o después (evento post) de cada escritura del agente y bloquea las
que van a un documento de Axiom (la carpeta de un cambio, openspec/changes/<cambio>/, o
odd/tasks/<cambio>.md) cuando el repositorio no está en la rama que corresponde al cambio: la rama
del cambio, una de rol o la de archivado. El mensaje del bloqueo incluye la orden axiom branch ensure
que pone el repositorio en su rama. Rige en cualquier proyecto Axiom, haya o no un cambio registrado.

Solo lee ficheros (el HEAD lo lee sin lanzar git ni ningún otro proceso), no escribe nada y falla en
abierto: ante una carga ilegible o de más de 256 KiB, una configuración o un registro ilegibles, un
proyecto que no es de Axiom o 1 s de presupuesto agotado, no bloquea ni avisa. Solo afecta a las
escrituras del agente a través del runtime, nunca a un editor humano ni a las órdenes de la CLI.

Protocolo de salida por runtime:
  claude-code, codex   Código 2 con el motivo por la salida de error si bloquea; 0 en otro caso
  opencode             Siempre código 0; por la salida estándar, {"action":"none|block|warn","message":"..."}

Banderas:
  --agent <runtime>   Runtime que envía la carga (obligatorio): claude-code, codex u opencode
`

// runBranchGuard ejecuta `axiom branch guard`: --agent es obligatorio y solo admite los runtimes
// con hook. Los errores de uso salen con el código 2 de la familia y no llegan a leer la carga.
func runBranchGuard(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch guard", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	agent := flags.String("agent", "", "runtime que envía la carga: claude-code, codex u opencode")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchGuardHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch guard --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch guard. Ejecuta `axiom branch guard --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	if !slices.Contains(engine.GuardAgents, *agent) {
		fmt.Fprintf(stderr, "Error: --agent '%s' no es válido: usa %s. Ejecuta `axiom branch guard --help`.\n", *agent, strings.Join(engine.GuardAgents, ", "))
		return branch.ExitUsage
	}
	return engine.RunGuard(*agent, stdin, stdout, stderr)
}
