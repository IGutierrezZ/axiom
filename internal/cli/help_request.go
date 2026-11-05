package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// helpRequestError transporta el bloque de uso que el paquete flag deriva de
// los flags registrados por un comando. Existe porque algunas funciones de
// análisis (ParseUninstallFlags) no poseen un writer y no pueden imprimir el
// uso por sí mismas: devuelven este error y quien sí tiene un writer lo
// responde con writeHelpRequest.
type helpRequestError struct {
	usage string
}

func (e *helpRequestError) Error() string {
	return e.usage
}

// parseCommandFlags analiza args conservando el uso que el paquete flag genera
// a partir de los flags registrados en fs, de modo que ningún comando tenga que
// repetir su propia documentación de flags ni pueda desviarse de ella.
//
// Distingue lo que la biblioteca estándar deja al llamador: flag.ErrHelp es un
// centinela que significa "el operador pidió ayuda", no un fallo. Devolverlo tal
// cual convierte una petición explícita en un error opaco, con salida distinta
// de cero y sin nombres de flags. Un error de análisis real sigue siendo error y
// se envuelve junto con el mismo uso derivado, para que el rechazo indique cómo
// continuar.
func parseCommandFlags(fs *flag.FlagSet, args []string) error {
	var usage bytes.Buffer
	fs.SetOutput(&usage)
	err := fs.Parse(args)
	if err == nil {
		return nil
	}
	text := derivedUsageText(&usage)
	if errors.Is(err, flag.ErrHelp) {
		return &helpRequestError{usage: text}
	}
	if text == "" {
		return err
	}
	return fmt.Errorf("%w — ejecuta `axiom %s --help` para ver los flags admitidos:\n%s", err, fs.Name(), text)
}

// writeHelpRequest responde a una petición de ayuda imprimiendo el uso derivado
// y devolviendo éxito. Cualquier otro error pasa intacto, de modo que los
// llamadores pueden envolver con esta función un resultado de análisis sin
// cambiar el tratamiento de los fallos.
func writeHelpRequest(err error, stdout io.Writer) error {
	var request *helpRequestError
	if errors.As(err, &request) {
		_, _ = fmt.Fprintln(stdout, request.usage)
		return nil
	}
	return err
}

// derivedUsageText recorta la línea de error duplicada que emite el paquete
// flag, para que el bloque de uso no vaya precedido del mismo mensaje que el
// llamador está a punto de reportar.
func derivedUsageText(usage *bytes.Buffer) string {
	text := usage.String()
	if index := strings.Index(text, "Usage of "); index >= 0 {
		text = text[index:]
	}
	return strings.TrimRight(text, "\n")
}
