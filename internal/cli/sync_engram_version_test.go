package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sync sondea la versión de Engram con la misma orden que install resolvió. Si
// sondeara el nombre a secas, una máquina en la que el binario resuelto responde
// (aquí, 1.18.0, por encima del mínimo slim) pero el nombre a secas no, daría una
// sección slim en install y una full en el primer sync: CLAUDE.md cambiaría sin
// que nada de la instalación hubiera cambiado. Es lo que rompía en CI de Linux
// `TestSyncWithoutChangesLeavesNoNewBackup` cuando había un engram en
// /usr/local/bin.
func TestSyncProbesTheSameEngramCommandAsInstall(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	root, _, _ := workspaceRootFixture(t, false, true)
	t.Chdir(root)

	// verifyEngramVersion (la orden a secas) sigue sin versión por TestMain: solo
	// responde la orden resuelta, como un binario real en la ruta que da PATH.
	restoreCommand := verifyEngramVersionCommand
	t.Cleanup(func() { verifyEngramVersionCommand = restoreCommand })
	var probed []string
	verifyEngramVersionCommand = func(command string) (string, error) {
		probed = append(probed, command)
		return "engram 1.18.0", nil
	}

	installInProject(t, root, "--component", "engram", "--persona", "neutral")
	claudeMD := filepath.Join(root, ".claude", "CLAUDE.md")
	installed, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(installed), "needs_review") || !strings.Contains(string(installed), "SessionStart hook") {
		t.Fatalf("precondición: install debía escribir la sección slim de Engram:\n%s", installed)
	}

	probed = nil
	result, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if !result.NoOp {
		t.Fatalf("el sync no debía cambiar nada: %v", result.ChangedFiles)
	}
	if len(probed) == 0 || probed[0] != "/usr/local/bin/engram" {
		t.Fatalf("sync debía sondear la orden que resuelve PATH, sondeó %v", probed)
	}
	after, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(installed) {
		t.Fatal("CLAUDE.md cambió tras un sync sin cambios")
	}
}
