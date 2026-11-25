//go:build windows

package fileprim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrReparsePoint indica que el objeto abierto es un punto de reanálisis (o que
// no se pudo comprobar), así que no se entrega.
//
// refusal:by-design world-action: the exit is removing the symlink or reparse point (or restoring a plain directory) on the reported path, which only the operator can do; no axiom command follows or rewrites links it refuses to trust
var ErrReparsePoint = errors.New(
	"path is or traverses a symlink or reparse point; remove the symlink/reparse point, then rerun the command",
)

// OpenNoFollow abre un fichero o directorio sin recorrer puntos de reanálisis.
// La ruta debe ser absoluta. Si el objeto abierto resulta ser un punto de
// reanálisis devuelve ErrReparsePoint.
func OpenNoFollow(path string, directory bool) (*os.File, error) {
	handle, err := OpenWindowsObject(NTPath(path), windows.FILE_GENERIC_READ|windows.READ_CONTROL, directory)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if IsReparsePoint(file) {
		_ = file.Close()
		return nil, ErrReparsePoint
	}
	return file, nil
}

// OpenPhysicalPath opens a file or directory without traversing reparse points.
func OpenPhysicalPath(path string, directory bool) (*os.File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return OpenNoFollow(absPath, directory)
}

// OpenWindowsObject abre un objeto del espacio de nombres NT con el acceso
// pedido y sin seguir puntos de reanálisis.
func OpenWindowsObject(objectPath string, access uint32, directory bool) (windows.Handle, error) {
	open := func(path string) (windows.Handle, error) {
		objectName, err := windows.NewNTUnicodeString(path)
		if err != nil {
			return 0, err
		}
		attributes := &windows.OBJECT_ATTRIBUTES{
			Length:     uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
			ObjectName: objectName,
			Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
		}
		options := uint32(windows.FILE_SYNCHRONOUS_IO_NONALERT | windows.FILE_OPEN_REPARSE_POINT)
		if directory {
			options |= windows.FILE_DIRECTORY_FILE
		} else {
			options |= windows.FILE_NON_DIRECTORY_FILE
		}
		var handle windows.Handle
		var status windows.IO_STATUS_BLOCK
		err = windows.NtCreateFile(
			&handle,
			access,
			attributes,
			&status,
			nil,
			windows.FILE_ATTRIBUTE_NORMAL,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			windows.FILE_OPEN,
			options,
			0,
			0,
		)
		return handle, err
	}
	handle, err := open(objectPath)
	if !errors.Is(err, windows.STATUS_REPARSE_POINT_ENCOUNTERED) {
		return handle, err
	}
	directPath, resolveErr := directLocalDriveObjectPath(objectPath, queryWindowsDosDevice)
	if resolveErr != nil {
		return 0, fmt.Errorf("resolve secure Windows RAR path after %w: %v", err, resolveErr)
	}
	return open(directPath)
}

// IsReparsePoint informa de si file es un punto de reanálisis. Un descriptor
// nulo o ilegible se trata como inseguro.
func IsReparsePoint(file *os.File) bool {
	if file == nil {
		return true
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return true
	}
	return info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
