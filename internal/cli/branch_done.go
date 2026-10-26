package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
)

// `axiom branch done` (INC-25, S6, REQ-25.16, REQ-25.17, design D-09): elimina el worktree y la
// rama local de cada rama de un cambio ya fusionado. El comportamiento vive en el motor
// (internal/branch/engine); aquí solo están las banderas, la ayuda y la salida.

const branchDoneHelp = `Uso: axiom branch done --change <nombre> [--lane sdd|odd] [--force] [--fetch] [--cwd <ruta>] [--json]

Elimina, de cada rama del plan de un cambio ya fusionado, su worktree y su rama local: primero
las ramas de rol, después la del cambio y por último la de archivado, cada una por separado.
Toda escritura de git se lanza desde el checkout principal del repositorio, nunca desde el
worktree que se retira.

Antes de eliminar una rama confirma que está fusionada en su destino (la rama de rol, en la de
seguimiento local; la del cambio y la de archivado, en cada referencia de integración que exista
entre origin/<integración> y <integración>). En esta versión la fusión se confirma por
ascendencia (ancestry): la rama es ancestro del destino. El resultado indica, por rama, la capa
que la confirmó (confirmedBy). Una rama sin fusión confirmada se conserva con
merge_not_confirmed: ni su worktree ni su rama se tocan.

Rechaza con dirty_worktree la rama cuyo worktree tiene cambios sin confirmar (con stage, sin
stage o sin seguimiento; los ficheros ignorados cuentan como limpios): no elimina nada. Con
--force salta la exigencia de fusión confirmada (la confirmación es entonces forced y el
resultado lleva la punta, tipSha, para recuperar la rama) y la de árbol limpio (el resultado lista
los cambios descartados, discarded).

Con el modo de trabajo in_place no hay worktree que retirar: si el repositorio está desplegado en
la rama y el árbol está limpio, hace git switch a la rama de integración y elimina la rama local.
Con el árbol sucio rechaza con dirty_worktree también con --force, que en in_place solo salta la
exigencia de fusión confirmada.

Si la retirada del worktree falla (en Windows, un worktree que es el directorio actual de algún
proceso no se puede borrar), la rama se conserva y el mensaje pide repetir la orden con --cwd
sobre el checkout principal, desde otro directorio. Actualiza el registro local del cambio: borra
de él las ramas eliminadas y el fichero cuando no queda ninguna.

El resultado es ready (código de salida 0) si todas las ramas se procesaron, partial (1) si algunas
se eliminaron y otras se conservaron, y el rechazo de la primera rama conservada (1) si ninguna se
pudo procesar. Una rama que ya no existe se informa como tal: repetir la orden es idempotente.

Solo elimina ramas locales y worktrees del plan del cambio: nunca toca una rama remota ni ejecuta
push, pull ni ls-remote. Sin --fetch no hace ninguna operación remota; con él ejecuta git fetch
origin una vez en cada repositorio del plan antes de evaluar, y si falla (fetch_failed) no
elimina nada.

Banderas:
  --change <nombre>   Nombre del cambio (obligatorio)
  --lane sdd|odd      Carril del cambio (por defecto, sdd)
  --force             Elimina también con fusión no confirmada o con cambios sin confirmar en un worktree
  --fetch             Ejecuta git fetch origin en cada repositorio del plan antes de evaluar
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json              Imprime el resultado como axiom.branch-result/v1

Código de salida: 0 si todas las ramas se procesan, 1 ante un rechazo, un resultado parcial o un
error de git, 2 ante un error de uso.
`

// runBranchDone ejecuta `axiom branch done`: --change es obligatorio y no puede estar en blanco;
// --lane solo admite sdd u odd.
func runBranchDone(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch done", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	change := flags.String("change", "", "nombre del cambio")
	lane := flags.String("lane", string(branch.LaneSDD), "carril del cambio: sdd u odd")
	force := flags.Bool("force", false, "elimina también con fusión no confirmada o con cambios sin confirmar")
	fetch := flags.Bool("fetch", false, "ejecuta git fetch origin en cada repositorio del plan antes de evaluar")
	cwd := flags.String("cwd", ".", "directorio de partida")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.branch-result/v1")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchDoneHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch done --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch done. Ejecuta `axiom branch done --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	if strings.TrimSpace(*change) == "" {
		fmt.Fprintln(stderr, "Error: falta --change: indica el nombre del cambio, por ejemplo `axiom branch done --change <nombre>`. Ejecuta `axiom branch done --help`.")
		return branch.ExitUsage
	}
	if *lane != string(branch.LaneSDD) && *lane != string(branch.LaneODD) {
		fmt.Fprintf(stderr, "Error: --lane '%s' no es válido: usa sdd u odd. Ejecuta `axiom branch done --help`.\n", *lane)
		return branch.ExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch done --change <nombre> --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	result := engine.Default().Done(context.Background(), engine.DoneRequest{
		CWD: dir, Change: *change, Lane: branch.Lane(*lane), Force: *force, Fetch: *fetch,
	})
	return writeBranchResult(result, *asJSON, stdout, stderr)
}
