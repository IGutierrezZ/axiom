package gga

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// RuntimeLibDir returns the runtime lib path used by gga.
func RuntimeLibDir(homeDir string) string {
	return filepath.Join(homeDir, ".local", "share", "gga", "lib")
}

// RuntimeBinDir returns ~/.local/share/gga/bin — where GGA's bash script lives on Linux/Windows.
func RuntimeBinDir(homeDir string) string {
	return filepath.Join(homeDir, ".local", "share", "gga", "bin")
}

// RuntimePRModePath returns the expected pr_mode.sh runtime path.
func RuntimePRModePath(homeDir string) string {
	return filepath.Join(RuntimeLibDir(homeDir), "pr_mode.sh")
}

// RuntimePS1Path returns the expected gga.ps1 path.
// On Windows, the shim goes to ~/bin/ so PowerShell finds it as a native command.
func RuntimePS1Path(homeDir string) string {
	return filepath.Join(homeDir, "bin", "gga.ps1")
}

// RuntimeCMDPath returns the expected gga.cmd path.
// On Windows, cmd.exe and exec.LookPath resolve .cmd files through PATHEXT.
func RuntimeCMDPath(homeDir string) string {
	return filepath.Join(homeDir, "bin", "gga.cmd")
}

// EnsureRuntimeAssets ensures critical gga runtime files are current.
//
// Behavior change from "only-if-missing" to "always-write":
// WriteFileAtomic performs a content-equality check — it is a no-op when the
// embedded asset matches the file on disk, and an atomic replace when it differs.
// This guarantees pr_mode.sh stays current after gentle-ai updates without
// touching the file on every sync when nothing has changed.
func EnsureRuntimeAssets(homeDir string) error {
	prModePath := RuntimePRModePath(homeDir)

	content, err := assets.Read("gga/pr_mode.sh")
	if err != nil {
		return fmt.Errorf("read embedded gga runtime asset pr_mode.sh: %w", err)
	}

	if _, err := filemerge.WriteFileAtomicMode(prModePath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write gga runtime file %q: %w", prModePath, err)
	}

	return nil
}

// EnsurePowerShellShim writes gga.ps1 to the GGA bin directory.
// Uses WriteFileAtomicMode: atomic replace when content differs, and a
// mode-only repair to 0755 when the content already matches.
// Must only be called on Windows (caller is responsible for the OS guard).
func EnsurePowerShellShim(homeDir string) error {
	ps1Path := RuntimePS1Path(homeDir)

	content, err := assets.Read("gga/gga.ps1")
	if err != nil {
		return fmt.Errorf("read embedded gga runtime asset gga.ps1: %w", err)
	}

	if _, err := filemerge.WriteFileAtomicMode(ps1Path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write gga runtime file %q: %w", ps1Path, err)
	}

	return nil
}

// EnsureCommandShim writes gga.cmd to the GGA bin directory.
// Must only be called on Windows (caller is responsible for the OS guard).
func EnsureCommandShim(homeDir string) error {
	cmdPath := RuntimeCMDPath(homeDir)

	content, err := assets.Read("gga/gga.cmd")
	if err != nil {
		return fmt.Errorf("read embedded gga runtime asset gga.cmd: %w", err)
	}

	if _, err := filemerge.WriteFileAtomicMode(cmdPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write gga runtime file %q: %w", cmdPath, err)
	}

	return nil
}

// Managed informa de si Axiom ya aprovisionó GGA en este HOME: existe la
// configuración global que escribe Inject o el recurso de ejecución que deja
// EnsureRuntimeAssets. Sync lo usa para no dejar de gestionar GGA a quien ya lo
// tenía cuando GGA dejó de formar parte de los presets por defecto. No consulta
// el PATH ni lanza procesos.
func Managed(homeDir string) bool {
	for _, path := range []string{ConfigPath(homeDir), RuntimePRModePath(homeDir)} {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}
