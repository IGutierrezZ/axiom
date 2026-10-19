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

// `axiom branch` (INC-25, D-02): la familia de verbos que planifica, crea y consulta las
// ramas y worktrees de un cambio. Trae `status` y `ensure` sobre el motor
// internal/branch/engine (planificador puro, operaciones de git y resultado
// `axiom.branch-result/v1`); los demás verbos llegan con las rebanadas siguientes y la
// ayuda lista solo los verbos que existen (decisión T3). Ningún verbo ejecuta `push` ni
// abre un PR, y ninguno hace operaciones remotas salvo `ensure --fetch`, que ejecuta
// `git fetch origin` (REQ-25.1, REQ-25.10).

const branchHelp = `Uso: axiom branch <subcomando> [argumentos]

Subcomandos:
  ensure   Crea la rama de un cambio y su worktree si faltan (idempotente)
  status   Estado de las ramas y los worktrees de un cambio (solo lectura)

Ejecuta ` + "`axiom branch <subcomando> --help`" + ` para ver las banderas de cada subcomando.
`

const branchStatusHelp = `Uso: axiom branch status [--change <nombre>] [--lane sdd|odd] [--cwd <ruta>] [--json]

Con --change, muestra, solo con lecturas, el estado de las ramas y los worktrees de un
cambio: por repositorio, su papel, la rama actual, la rama de integración que resuelve la
política de ramas de axiom.yaml (bloques git: y repositories:; sin ellos, main) y la
base; por rama, si existe, su worktree, si tiene cambios sin confirmar, los commits de
adelanto y de atraso respecto de su base (solo con referencias locales) y la rama destino
propuesta para su PR. Un cambio sin ramas creadas se informa como «no iniciado», con
código de salida 0. En multirepo, el repositorio de specs se informa in situ, sin ramas
ni worktrees. Si el cambio tiene registro local (en el git common dir), informa también
del modo de trabajo materializado y de los worktrees registrados; sin él, indica que no
lo tiene. Indica además si el cambio está pendiente de archivar (pendingArchive) y, si
lo está, la orden para archivarlo. Rechaza con plan_conflict o path_too_long un plan cuyos
worktrees se solaparían con otro árbol de trabajo o superarían el límite de Windows.

Sin --change, lista los cambios registrados del carril (modo de trabajo y worktrees) y
los cambios pendientes de archivar del proyecto: los que ya están en el árbol de la rama
de integración local (origin/<integración> si existe en local y, si no, la local) y
siguen sin archivar, cada uno con la orden axiom branch ensure --change <c> --archive.
Es información, no una comprobación: nunca produce un código de fallo ni bloquea nada, y
si no existe la rama de integración la lista se omite con un aviso. En multirepo, el
repositorio de specs no tiene pendientes.

No ejecuta fetch, pull, push ni ls-remote ni escribe ningún fichero.

Banderas:
  --change <nombre>   Nombre del cambio (opcional)
  --lane sdd|odd      Carril del cambio y de los registros que se listan (por defecto, sdd)
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json              Imprime el resultado como axiom.branch-result/v1
`

const branchEnsureHelp = `Uso: axiom branch ensure --change <nombre> [--lane sdd|odd] [--mode worktree|in_place] [--fetch] [--cwd <ruta>] [--json]

Crea lo que falta de las ramas de un cambio y de sus worktrees. En cada repositorio del
plan crea la rama del cambio (<prefijo>/<cambio>; con dos o más roles sobre el mismo
repositorio es la rama de seguimiento y cada rol saca <prefijo>/<cambio>-<rol> desde ella)
y, en modo worktree, un worktree por rama en la raíz de worktrees:
<carpeta del proyecto>-wt/<cambio>/<repositorio>, fuera de cualquier repositorio del plan
(el sufijo del rol va en la carpeta del cambio: <cambio>-<rol>). El prefijo y la rama de
integración salen de los bloques git: y repositories: de axiom.yaml (sin ellos, feat y
main). Antes del sellado del kickoff solo se crea la rama del cambio del repositorio del
proyecto; una ejecución posterior completa las ramas de rol y los repositorios de código.

La base es origin/<integración> si esa referencia ya existe en local y, si no, la rama
local de integración. Antes de crear nada comprueba todos los repositorios del plan: sin
base (base_not_found), con una rama desplegada en un worktree desconocido
(branch_checked_out_elsewhere) o con una ruta planificada ocupada (plan_conflict) rechaza
sin dejar efectos.

Crea solo lo que falta: una rama que ya existe se reutiliza sin moverla ni reiniciarla y
un worktree que ya está en su ruta se devuelve tal cual. El checkout principal no se toca
(ni su HEAD, ni su índice, ni su árbol de trabajo). El resultado describe el estado final
y la orden es idempotente: repetirla con el mismo estado da el mismo resultado byte a
byte y no cambia ni ramas ni worktrees. Si git falla a medias (git_failed), repetir la
orden completa lo que falte.

El modo de trabajo sale de tus preferencias personales del hub (~/.axiom/workspaces.json,
nunca de axiom.yaml) y, sin ellas, es worktree; la raíz de worktrees también puede venir de
ahí. --mode worktree|in_place lo fija para el cambio y se registra con su origen. El registro
del cambio manda sobre --mode y sobre el hub, y un --mode distinto del ya materializado es
plan_conflict. Con el modo ask y sin respuesta registrada, ensure devuelve decision_required
(código de salida 1) con la pregunta y las dos órdenes posibles, sin crear nada (tampoco el
registro) y sin leer la entrada estándar: ensure no espera entrada, quien invoca presenta la
pregunta y repite la orden con --mode. La respuesta se registra por cambio y no modifica tus
preferencias.

En modo in_place no hay worktrees: ensure crea las ramas igual y despliega la del cambio
(la de seguimiento, si hay ramas de rol) con git switch en el propio repositorio. Antes de
cambiar de rama exige el árbol limpio, sin cambios con stage, sin stage ni ficheros sin
seguimiento (los ignorados cuentan como limpios): si no, rechaza con dirty_worktree sin
modificar nada y nombra los ficheros. Si el repositorio ya está en la rama del cambio no
cambia de rama y no mira la suciedad.

Guarda el plan en un registro local por cambio, dentro del git common dir del repositorio
del proyecto (<git common dir>/axiom/branch/v1/<carril>/<cambio>.json), que solo se
reescribe si cambia su contenido: es estado local derivado y no se versiona. Si lo borras,
el repositorio no sufre y la siguiente ejecución lo reconstruye desde las ramas y los
worktrees existentes. Un bloqueo por cambio impide dos ejecuciones a la vez (plan_conflict).

Ninguna orden hace red sin --fetch: con él, ensure ejecuta git fetch origin una vez en cada
repositorio del plan que tiene ramas, antes de resolver la base, y el resultado marca
base.fetched; si el fetch falla (fetch_failed) no crea ninguna rama, worktree ni registro.
Sin --fetch trabaja solo con las referencias que ya hay en local. Nunca ejecuta push, pull
ni ls-remote ni abre ningún PR.

Banderas:
  --change <nombre>   Nombre del cambio (obligatorio)
  --lane sdd|odd      Carril del cambio (por defecto, sdd)
  --mode worktree|in_place
                      Modo de trabajo del cambio; es la respuesta a decision_required (por defecto, el de tus preferencias)
  --fetch             Ejecuta git fetch origin en cada repositorio del plan antes de resolver la base
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json              Imprime el resultado como axiom.branch-result/v1

Código de salida: 0 si el cambio queda preparado, 1 ante un rechazo o un error de git,
2 ante un error de uso.
`

// RunBranch ejecuta `axiom branch <subcomando>` y devuelve el código de salida.
func RunBranch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, branchHelp)
		return branch.ExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, branchHelp)
		return branch.ExitReady
	case "ensure":
		return runBranchEnsure(rest, stdout, stderr)
	case "status":
		return runBranchStatus(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para branch; los disponibles son ensure y status. Ejecuta `axiom branch --help`.\n", subcommand)
		return branch.ExitUsage
	}
}

// runBranchEnsure ejecuta `axiom branch ensure`: --change es obligatorio y no puede
// estar en blanco; --lane solo admite sdd u odd.
func runBranchEnsure(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch ensure", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	change := flags.String("change", "", "nombre del cambio")
	lane := flags.String("lane", string(branch.LaneSDD), "carril del cambio: sdd u odd")
	mode := flags.String("mode", "", "modo de trabajo del cambio: worktree o in_place")
	fetch := flags.Bool("fetch", false, "ejecuta git fetch origin en cada repositorio del plan antes de resolver la base")
	cwd := flags.String("cwd", ".", "directorio de partida")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.branch-result/v1")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchEnsureHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch ensure --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch ensure. Ejecuta `axiom branch ensure --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	if strings.TrimSpace(*change) == "" {
		fmt.Fprintln(stderr, "Error: falta --change: indica el nombre del cambio, por ejemplo `axiom branch ensure --change <nombre>`. Ejecuta `axiom branch ensure --help`.")
		return branch.ExitUsage
	}
	if *lane != string(branch.LaneSDD) && *lane != string(branch.LaneODD) {
		fmt.Fprintf(stderr, "Error: --lane '%s' no es válido: usa sdd u odd. Ejecuta `axiom branch ensure --help`.\n", *lane)
		return branch.ExitUsage
	}
	// Una bandera presente pero en blanco (casi siempre una variable de entorno vacía) no se
	// confunde con su ausencia: ask no es una respuesta y vacío tampoco.
	modeGiven := false
	flags.Visit(func(given *flag.Flag) {
		if given.Name == "mode" {
			modeGiven = true
		}
	})
	if modeGiven && *mode != string(branch.ModeWorktree) && *mode != string(branch.ModeInPlace) {
		fmt.Fprintf(stderr, "Error: --mode '%s' no es válido: usa worktree o in_place (ask es una preferencia del hub, no una respuesta). Ejecuta `axiom branch ensure --help`.\n", *mode)
		return branch.ExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch ensure --change <nombre> --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	result := engine.Default().Ensure(context.Background(), engine.EnsureRequest{CWD: dir, Change: *change, Lane: branch.Lane(*lane), Fetch: *fetch, Mode: branch.Mode(*mode)})
	return writeBranchResult(result, *asJSON, stdout, stderr)
}

func runBranchStatus(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	change := flags.String("change", "", "nombre del cambio")
	lane := flags.String("lane", string(branch.LaneSDD), "carril del cambio: sdd u odd")
	cwd := flags.String("cwd", ".", "directorio de partida")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.branch-result/v1")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchStatusHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch status --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch status. Ejecuta `axiom branch status --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	// Sin la bandera, `status` lista los registrados y los pendientes; una bandera presente
	// pero en blanco (casi siempre una variable de entorno vacía) no se confunde con su
	// ausencia, para no listar lo que el usuario no pidió.
	changeGiven := false
	flags.Visit(func(given *flag.Flag) {
		if given.Name == "change" {
			changeGiven = true
		}
	})
	if changeGiven && strings.TrimSpace(*change) == "" {
		fmt.Fprintln(stderr, "Error: --change no puede estar vacío: indica el nombre del cambio o quita la bandera para listar los registrados y los pendientes de archivar. Ejecuta `axiom branch status --help`.")
		return branch.ExitUsage
	}
	if *lane != string(branch.LaneSDD) && *lane != string(branch.LaneODD) {
		fmt.Fprintf(stderr, "Error: --lane '%s' no es válido: usa sdd u odd. Ejecuta `axiom branch status --help`.\n", *lane)
		return branch.ExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch status --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	return writeBranchResult(engine.Default().Status(context.Background(), dir, *change, branch.Lane(*lane)), *asJSON, stdout, stderr)
}

// writeBranchResult imprime el resultado, como JSON o como texto en castellano, y
// devuelve el código de salida: 0 solo cuando el estado pedido se alcanzó.
func writeBranchResult(result branch.Result, asJSON bool, stdout, stderr io.Writer) int {
	if asJSON {
		if err := result.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom branch %s --json`.\n", err, result.Operation)
			return branch.ExitRefused
		}
		return result.ExitCode()
	}
	result.WriteText(stdout, stderr)
	return result.ExitCode()
}
