//go:build !windows

package opencode

import "errors"

func platformShortPathName(string) (string, error) {
	return "", errors.New("short path names exist only on Windows")
}
