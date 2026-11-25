//go:build unix

package fileprim

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// OpenNoFollow abre un fichero o directorio de solo lectura sin seguir enlaces
// simbólicos en ningún componente de la ruta, con el recorrido anclado en la
// raíz del sistema de ficheros.
func OpenNoFollow(path string, directory bool) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parentFD, err := secureOpenLockParent(string(filepath.Separator), filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	defer unix.Close(parentFD)
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat(parentFD, filepath.Base(absolute), flags, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
