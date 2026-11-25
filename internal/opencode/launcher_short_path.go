package opencode

// shortPathName devuelve la forma corta 8.3 de una ruta existente. Es una
// variable para que las pruebas simulen el sistema de ficheros; fuera de
// Windows no hay nombres cortos y devuelve error.
var shortPathName = platformShortPathName

// windowsLauncherTarget devuelve la ruta que el lanzador gestionado de Windows
// escribe para invocar el runtime real.
//
// cmd.exe decodifica un .cmd con la página de códigos OEM de la consola, y
// Windows PowerShell 5.1 lee un .ps1 sin BOM como ANSI: una ruta con
// caracteres no ASCII (un perfil como C:\Users\Iñigo) escrita en UTF-8 llega
// alterada y el lanzador no encuentra el runtime. Si la ruta larga no es ASCII
// y el volumen tiene nombres cortos, se usa la forma 8.3, que siempre es ASCII
// y apunta al mismo fichero. Si no la hay, se conserva la ruta larga.
func windowsLauncherTarget(goos, target string) string {
	if goos != "windows" || isASCII(target) {
		return target
	}
	short, err := shortPathName(target)
	if err != nil || short == "" || !isASCII(short) {
		return target
	}
	return short
}

func isASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= 0x80 {
			return false
		}
	}
	return true
}
