// Package gitcommon reúne lo que Axiom necesita saber de git sin duplicarlo en cada
// paquete. Es una hoja: solo importa la biblioteca estándar.
//
// Las lecturas de este fichero trabajan solo sobre ficheros y no lanzan procesos
// (INC-25, REQ-25.21): saber en qué repositorio git está una ruta no puede
// depender de que haya un binario git ni de su entorno.
package gitcommon

import (
	"os"
	"path/filepath"
)

// gitEntryName es la entrada que marca la raíz de un repositorio: un directorio
// en un checkout normal y un fichero `gitdir:` en un worktree enlazado o un
// submódulo.
const gitEntryName = ".git"

// GitRootFromFiles devuelve la raíz del repositorio git al que pertenece path y
// true, o ("", false) si no pertenece a ninguno. Asciende desde path hasta el
// primer directorio que contiene una entrada `.git`, directorio o fichero, así que
// el repositorio más cercano gana y un worktree enlazado tiene su propia raíz,
// distinta de la del checkout principal. No lanza procesos.
//
// path puede ser un directorio, un fichero o una ruta que aún no existe: la
// ascensión es léxica, sobre la ruta absoluta y limpia, de modo que quien comprueba
// un destino de escritura obtiene la raíz de su repositorio antes de crear nada. Una
// ruta relativa se resuelve contra el directorio actual y la vacía no tiene raíz.
//
// La raíz devuelta conserva la grafía de path (no se resuelven enlaces simbólicos
// ni nombres cortos 8.3 de Windows): quien compare dos raíces debe hacerlo por
// identidad (pathidentity.SameDirectory) y no por texto. Una entrada `.git` que no
// se puede consultar cuenta como ausente.
func GitRootFromFiles(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	current, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Lstat(filepath.Join(current, gitEntryName)); err == nil {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}
