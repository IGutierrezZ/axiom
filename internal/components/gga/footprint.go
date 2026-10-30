package gga

import "path/filepath"

// FootprintPaths lista lo que GGA deja fuera del proyecto, en la carpeta de
// usuario, cuando Axiom lo provisiona: su configuración global, su runtime y, en
// Windows, los shims de `~/bin`. Es la huella que se declara al usuario cuando
// elige GGA de forma explícita con el ámbito workspace y la que `axiom doctor`
// busca para listar la huella de máquina. No toca el disco: devuelve rutas, no
// comprueba que existan.
//
// goos se inyecta para poder probar las rutas de cada plataforma sin etiquetas
// de compilación.
func FootprintPaths(homeDir, goos string) []string {
	paths := []string{
		ggaConfigDir(homeDir, goos),
		filepath.Dir(RuntimeLibDir(homeDir)),
	}
	if goos == "windows" {
		paths = append(paths, RuntimeCMDPath(homeDir), RuntimePS1Path(homeDir))
	}
	return paths
}

// FootprintPathEntries lista las carpetas que el provisionado de GGA añade al
// PATH de usuario. Solo Windows persiste el cambio: en el resto de plataformas
// el PATH solo se amplía en el proceso en curso, así que no hay huella.
func FootprintPathEntries(homeDir, goos string) []string {
	if goos != "windows" {
		return nil
	}
	return []string{filepath.Join(homeDir, "bin")}
}
