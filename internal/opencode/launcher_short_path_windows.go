//go:build windows

package opencode

import "golang.org/x/sys/windows"

func platformShortPathName(path string) (string, error) {
	long, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetShortPathName(long, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", err
	}
	if n > uint32(len(buffer)) {
		buffer = make([]uint16, n)
		if n, err = windows.GetShortPathName(long, &buffer[0], n); err != nil {
			return "", err
		}
	}
	return windows.UTF16ToString(buffer[:n]), nil
}
