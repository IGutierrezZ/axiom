package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// Preferencias personales de worktree al registrar un proyecto (INC-25, REQ-25.30): `init` y
// `project add` las toman de `--worktree-mode` y `--worktree-root` o, con una terminal
// interactiva y sin opciones, las preguntan ofreciendo los valores por defecto.

// La consola de la pregunta. Son variables para sustituirlas en las pruebas. La pregunta va a
// la salida de error: la salida estándar es la del resumen del registro.
var (
	registrationInteractive           = cli.ConsoleInteractive
	registrationInput       io.Reader = os.Stdin
	registrationOutput      io.Writer = os.Stderr
)

// registrationWorktreePrefs resuelve las preferencias del proyecto cuya carpeta maestra es
// master con la consola real del proceso.
func registrationWorktreePrefs(mode, root, master string) (hub.WorktreePrefs, error) {
	return resolveRegistrationWorktreePrefs(mode, root, registrationInteractive(), registrationInput, registrationOutput, master)
}

// resolveRegistrationWorktreePrefs decide las preferencias que se guardan al registrar:
//
//   - Con cualquiera de las dos opciones, o sin terminal interactiva, no pregunta ni lee la
//     entrada: usa las opciones (validadas) o, sin ellas, ninguna preferencia.
//   - Con terminal y sin opciones pregunta el modo (con `worktree` preseleccionado) y la raíz
//     (mostrando branch.DefaultWorktreeRoot(master)). Enter, el valor por defecto escrito o el
//     fin de la entrada dejan ese campo sin preferencia, de modo que aceptar los valores por
//     defecto no guarda nada. Un valor inválido tecleado se explica y se vuelve a preguntar.
//
// Una opción explícita se guarda tal cual aunque coincida con el valor por defecto: quien la
// escribe la está fijando.
func resolveRegistrationWorktreePrefs(mode, root string, interactive bool, in io.Reader, out io.Writer, master string) (hub.WorktreePrefs, error) {
	if mode != "" || root != "" || !interactive {
		prefs := hub.WorktreePrefs{Mode: mode, Root: root}
		if err := hub.ValidateWorktreePrefs(prefs); err != nil {
			return hub.WorktreePrefs{}, err
		}
		return prefs, nil
	}

	reader := bufio.NewReader(in)
	chosenMode, err := askWorktreeValue(reader, out,
		fmt.Sprintf("Modo de trabajo [%s] (%s, %s, %s): ", hub.WorktreeModeWorktree, hub.WorktreeModeWorktree, hub.WorktreeModeInPlace, hub.WorktreeModeAsk),
		func(answer string) error { return hub.ValidateWorktreePrefs(hub.WorktreePrefs{Mode: answer}) },
		func(answer string) bool { return answer == hub.WorktreeModeWorktree })
	if err != nil {
		return hub.WorktreePrefs{}, err
	}

	defaultRoot := branch.DefaultWorktreeRoot(master)
	chosenRoot, err := askWorktreeValue(reader, out,
		fmt.Sprintf("Raíz de worktrees [%s]: ", defaultRoot),
		func(answer string) error { return hub.ValidateWorktreePrefs(hub.WorktreePrefs{Root: answer}) },
		func(answer string) bool { return sameRoot(answer, defaultRoot) })
	if err != nil {
		return hub.WorktreePrefs{}, err
	}
	return hub.WorktreePrefs{Mode: chosenMode, Root: chosenRoot}, nil
}

// askWorktreeValue hace una pregunta y devuelve la respuesta, o "" si la persona acepta el
// valor por defecto (Enter, el propio valor por defecto o el fin de la entrada). Una respuesta
// que validate rechaza se explica en out y se vuelve a preguntar; el fin de la entrada acaba
// siempre la pregunta.
func askWorktreeValue(reader *bufio.Reader, out io.Writer, question string, validate func(string) error, isDefault func(string) bool) (string, error) {
	for {
		_, _ = fmt.Fprint(out, question)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("no se pudo leer la respuesta: %w", err)
		}
		answer := strings.TrimSpace(line)
		if answer == "" || isDefault(answer) {
			if err != nil {
				_, _ = fmt.Fprintln(out)
			}
			return "", nil
		}
		if invalid := validate(answer); invalid != nil {
			_, _ = fmt.Fprintf(out, "  %v\n", invalid)
			continue
		}
		return answer, nil
	}
}

// sameRoot dice si dos rutas de raíz de worktrees son la misma ruta limpia, sin distinguir
// mayúsculas en Windows.
func sameRoot(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// printWorktreePrefs añade al resumen del registro las preferencias que guarda la entrada; sin
// ninguna no escribe nada.
func printWorktreePrefs(w io.Writer, rec hub.WorkspaceRecord) {
	if rec.WorktreeMode != "" {
		_, _ = fmt.Fprintf(w, "  - Modo de worktree:    %s\n", rec.WorktreeMode)
	}
	if rec.WorktreeRoot != "" {
		_, _ = fmt.Fprintf(w, "  - Raíz de worktrees:   %s\n", rec.WorktreeRoot)
	}
}
