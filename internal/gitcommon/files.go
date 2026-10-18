// Package gitcommon reúne lo que Axiom necesita saber de git sin duplicarlo en cada
// paquete. Es una hoja: solo importa la biblioteca estándar. Tiene dos partes:
//
//   - exec.go es el único punto que ejecuta git (INC-25, REQ-25.1, G-7): Run, con
//     el observador Observe, y las consultas TopLevel y CommonDir. Una prueba de
//     arquitectura impide que `internal/branch/**` o este paquete lancen procesos o
//     abran conexiones de red por otra vía.
//   - files.go reúne las lecturas que trabajan solo sobre ficheros y no lanzan
//     procesos (REQ-25.21): saber en qué repositorio git está una ruta, o en qué
//     rama está su HEAD, no puede depender de que haya un binario git ni de su
//     entorno.
package gitcommon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnknownHead lo devuelve ReadHead cuando no puede decir en qué rama está un
// repositorio ni si su HEAD está desacoplado: `HEAD` ilegible o ausente, con una
// forma que no se reconoce (el backend reftable escribe `refs/heads/.invalid`) o sin
// repositorio. Quien lo recibe decide si falla en abierto (`guard`) o lo informa.
var ErrUnknownHead = errors.New("no se pudo determinar el HEAD del repositorio")

// maxSmallFile acota la lectura de `.git` y de `HEAD`: son ficheros de una línea y
// un fichero mucho mayor no es de git.
const maxSmallFile = 64 << 10

// reftableStubBranch es la rama que el backend reftable escribe en `HEAD` como
// marcador: el HEAD real no está en un fichero y no se puede leer sin git.
const reftableStubBranch = ".invalid"

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

// GitDirFromFiles devuelve el directorio git del repositorio cuya raíz es gitRoot:
// la propia entrada `.git` si es un directorio (checkout normal) y el destino de su
// línea `gitdir:` si es un fichero (worktree enlazado o submódulo), que puede ser
// relativo a gitRoot. La ruta devuelta es absoluta y está limpia. No lanza procesos.
//
// En un worktree enlazado el directorio git es `<common dir>/worktrees/<nombre>`,
// no el common dir, que es el que devuelve CommonDir.
func GitDirFromFiles(gitRoot string) (string, error) {
	if gitRoot == "" {
		return "", errors.New("la raíz git está vacía: indica la raíz de un repositorio")
	}
	root, err := filepath.Abs(gitRoot)
	if err != nil {
		return "", fmt.Errorf("resolver la raíz git %q: %w", gitRoot, err)
	}
	entry := filepath.Join(root, gitEntryName)
	info, err := os.Stat(entry)
	if err != nil {
		return "", fmt.Errorf("%s no tiene una entrada %s legible: %w", root, gitEntryName, err)
	}
	if info.IsDir() {
		return entry, nil
	}
	content, err := readSmallFile(entry)
	if err != nil {
		return "", fmt.Errorf("leer %s: %w", entry, err)
	}
	firstLine, _, _ := strings.Cut(content, "\n")
	target, isGitdir := strings.CutPrefix(strings.TrimRight(firstLine, "\r"), "gitdir:")
	target = strings.TrimSpace(target)
	if !isGitdir || target == "" {
		return "", fmt.Errorf("%s no es un fichero gitdir válido: se esperaba una línea `gitdir: <ruta>`", entry)
	}
	target = filepath.FromSlash(target)
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	return filepath.Clean(target), nil
}

// ReadHead dice en qué está el HEAD del repositorio cuya raíz es gitRoot, leyendo
// `<gitdir>/HEAD` sin lanzar procesos (REQ-25.21):
//
//   - `ref: refs/heads/<x>` es la rama x (branch = x, detached = false);
//   - un identificador de objeto (40 o 64 hexadecimales) es un HEAD desacoplado
//     (branch vacío, detached = true);
//   - cualquier otra forma, o un HEAD ilegible o ausente, devuelve un error que
//     envuelve ErrUnknownHead. El backend reftable, que escribe `ref:
//     refs/heads/.invalid`, es una de ellas.
//
// Tolera CRLF y espacios al final de la línea y solo lee la primera. En una rama sin
// commits (recién iniciada) devuelve la rama aunque aún no exista.
func ReadHead(gitRoot string) (branch string, detached bool, err error) {
	gitDir, err := GitDirFromFiles(gitRoot)
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUnknownHead, err)
	}
	content, err := readSmallFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUnknownHead, err)
	}
	firstLine, _, _ := strings.Cut(content, "\n")
	line := strings.TrimSpace(firstLine)

	if ref, isRef := strings.CutPrefix(line, "ref:"); isRef {
		name, isBranch := strings.CutPrefix(strings.TrimSpace(ref), "refs/heads/")
		if !isBranch || name == "" || name == reftableStubBranch {
			return "", false, fmt.Errorf("%w: el HEAD de %s apunta a %q, que no es una rama que se pueda leer sin git", ErrUnknownHead, gitDir, strings.TrimSpace(ref))
		}
		return name, false, nil
	}
	if isObjectID(line) {
		return "", true, nil
	}
	return "", false, fmt.Errorf("%w: el HEAD de %s tiene una forma que no se reconoce (%q)", ErrUnknownHead, gitDir, line)
}

// isObjectID dice si s es un identificador de objeto de git completo: 40
// hexadecimales (SHA-1) o 64 (SHA-256).
func isObjectID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, char := range s {
		switch {
		case char >= '0' && char <= '9', char >= 'a' && char <= 'f', char >= 'A' && char <= 'F':
		default:
			return false
		}
	}
	return true
}

// readSmallFile lee un fichero de git de una línea (`.git`, `HEAD`) con un tope de
// tamaño: uno mayor no es de git y no se lee entero.
func readSmallFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxSmallFile+1))
	if err != nil {
		return "", err
	}
	if len(content) > maxSmallFile {
		return "", fmt.Errorf("%s supera los %d bytes: no es un fichero de git", path, maxSmallFile)
	}
	return string(content), nil
}
