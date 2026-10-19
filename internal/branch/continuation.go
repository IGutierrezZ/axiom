package branch

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Continuaciones: toda negativa del dominio nombra una orden ejecutable (`axiom ...`
// o `git ...`) con las rutas citadas por pathquote.Quote, de modo que la orden
// impresa se puede copiar tal cual en Windows y en POSIX (tasks.md, regla 11).
// Cada continuación nace con su primer llamador: las de `ensure`, `done` y el
// archivado llegan con ellos.

// cwdContinuation indica cómo apuntar la orden a un repositorio cuando el directorio
// de partida no sirve. `<ruta>` es la plantilla de lo que el usuario debe elegir.
func cwdContinuation(operation string) string {
	return fmt.Sprintf("indica el directorio de un repositorio git con `axiom branch %s --cwd <ruta>`", operation)
}

// initContinuation añade, a la anterior, iniciar un repositorio en el directorio de
// partida.
func initContinuation(operation, dir string) string {
	return fmt.Sprintf("%s o inicia uno aquí con `git -C %s init`", cwdContinuation(operation), pathquote.Quote(dir))
}

// retryContinuation indica qué comprobar y cómo repetir la orden tras un fallo de
// git que ningún otro código cubre.
func retryContinuation(operation, dir string) string {
	return fmt.Sprintf("comprueba que git está instalado y que puedes leer %s, y repite `axiom branch %s --cwd %s`", dir, operation, pathquote.Quote(dir))
}

// validateConfigContinuation indica cómo comprobar que la configuración queda sin
// errores. Sin carpeta maestra conocida, deja la plantilla `<ruta>`.
func validateConfigContinuation(master string) string {
	target := "<ruta>"
	if master != "" {
		target = pathquote.Quote(master)
	}
	return fmt.Sprintf("corrige la configuración de axiom.yaml y comprueba con `axiom workspace validate --path %s` que queda sin errores", target)
}
