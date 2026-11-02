package workspace

import (
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// RootSource dice de dónde sale la raíz que devuelve ResolveRoot.
type RootSource string

const (
	// RootSourceAxiomProject es el directorio del axiom.yaml (propio o alcanzado
	// por el puntero .axiom-workspace) del proyecto Axiom que contiene start.
	RootSourceAxiomProject RootSource = "proyecto Axiom"
	// RootSourceGitRepository es la raíz del repositorio git, o del worktree
	// enlazado, que contiene start cuando no hay proyecto Axiom.
	RootSourceGitRepository RootSource = "repositorio git"
	// RootSourceCurrentDir es el propio start: no hay proyecto Axiom ni
	// repositorio git que lo contengan.
	RootSourceCurrentDir RootSource = "carpeta actual"
)

// ResolveRoot devuelve la raíz de proyecto en la que deben instalarse los
// ficheros de ámbito workspace cuando el usuario lanza el comando desde start, que
// puede ser una subcarpeta del proyecto. Es de solo lectura y no lanza procesos.
//
// La regla, por orden:
//
//  1. La raíz del proyecto Axiom que contiene start (IsAxiomProject): el
//     directorio del axiom.yaml o del puntero .axiom-workspace más cercano.
//  2. La raíz del repositorio git que contiene start (el `.git` más cercano,
//     directorio o fichero).
//  3. start, tal cual: el comportamiento de siempre fuera de todo proyecto.
//
// En un worktree enlazado la raíz es siempre la del propio worktree, nunca la del
// checkout principal: si el worktree no tiene marcador propio, IsAxiomProject
// resuelve desde el checkout principal (REQ-25.28) y aquí se descarta ese
// resultado, porque escribir allí ensuciaría otro árbol de trabajo.
//
// homeDir, si no está vacío, nunca se toma como raíz git: quien versiona su
// directorio personal (un repositorio de dotfiles) no debe recibir los ficheros de
// un proyecto por lanzar el comando desde cualquier carpeta de su home. El
// axiom.yaml sí vale aunque esté en homeDir, porque es una declaración explícita.
//
// Un start inexistente, que no es un directorio o es la raíz del sistema de
// ficheros se devuelve sin cambios con RootSourceCurrentDir.
func ResolveRoot(start, homeDir string) (string, RootSource) {
	abs, err := normalizeStart(start)
	if err != nil {
		return start, RootSourceCurrentDir
	}
	linked := gitcommon.LinkedWorktreeRoot(abs)
	if root, ok := IsAxiomProject(abs); ok && (linked == "" || pathidentity.Contains(linked, root)) {
		return root, RootSourceAxiomProject
	}
	if root, ok := gitcommon.GitRootFromFiles(abs); ok {
		if homeDir == "" || !pathidentity.SameDirectory(root, homeDir) {
			return root, RootSourceGitRepository
		}
	}
	return abs, RootSourceCurrentDir
}
