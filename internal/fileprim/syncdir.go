package fileprim

import (
	"errors"
	"os"
	"runtime"
	"syscall"
)

// SyncDirectory persiste una entrada de directorio cuando la plataforma lo
// admite. Los sistemas de ficheros de Windows pueden rechazar los descriptores
// de directorio; en ese caso el renombrado del fichero sigue siendo atómico,
// pero no se reclama durabilidad de la entrada ante un corte de corriente.
func SyncDirectory(path string) error {
	return TolerateUnsupportedDirectorySync(SyncDirectoryStrict(path), runtime.GOOS)
}

// TolerateUnsupportedDirectorySync descarta el error de una sincronización de
// directorio cuando solo indica que la plataforma no la admite (EINVAL,
// ErrUnsupported, o ErrPermission en Windows) y devuelve cualquier otro error.
func TolerateUnsupportedDirectorySync(err error, goos string) error {
	if err == nil {
		return nil
	}
	unsupported := errors.Is(err, syscall.EINVAL) || errors.Is(err, errors.ErrUnsupported) ||
		goos == "windows" && errors.Is(err, os.ErrPermission)
	if !unsupported {
		return err
	}
	return nil
}

// SyncDirectoryStrict sincroniza el directorio sin tolerar ningún error.
func SyncDirectoryStrict(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
