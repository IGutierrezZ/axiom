//go:build unix && !darwin

package fileprim

import "golang.org/x/sys/unix"

func secureOpenLocalStoreLockLeaf(parentFD int, name string, flags int, mode uint32) (int, error) {
	return unix.Openat(parentFD, name, flags, mode)
}
