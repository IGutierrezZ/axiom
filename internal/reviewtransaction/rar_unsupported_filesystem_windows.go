//go:build windows

package reviewtransaction

import "strings"

// Windows-only: the only callers are the Windows authority checks in
// rar_path_safety_windows.go. Kept out of the portable file so the dead-code
// ratchet does not see them as unreachable on other platforms.

// asUnsupportedAuthorityFilesystem projects a platform refusal onto the
// exported type for errors.As, so a caller outside this package can name the
// continuation without matching platform internals.
func asUnsupportedAuthorityFilesystem(target any, refusal *UnsupportedAuthorityFilesystemError) bool {
	typed, ok := target.(**UnsupportedAuthorityFilesystemError)
	if !ok {
		return false
	}
	*typed = refusal
	return true
}

// windowsNetworkPath reports whether a Win32 path names a server share: the
// `\\server\share` form or its `\\?\UNC\` and `\\.\UNC\` namespace spellings.
//
// #5533: the Windows authority checks refuse every path in the `\\` family,
// and that refusal stays exactly as it is. Only its wording depends on this:
// the long-path and device forms of a drive letter or volume, such as
// `\\?\C:\repo\.git`, are on this machine and must not be reported as an
// SMB/UNC share. It is pure so tests execute it on every platform.
func windowsNetworkPath(path string) bool {
	for _, prefix := range []string{`\\?\UNC\`, `\\.\UNC\`} {
		if len(path) >= len(prefix) && strings.EqualFold(path[:len(prefix)], prefix) {
			return true
		}
	}
	return strings.HasPrefix(path, `\\`) &&
		!strings.HasPrefix(path, `\\?\`) && !strings.HasPrefix(path, `\\.\`)
}
