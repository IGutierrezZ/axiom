package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

func TestAzureDevOpsMCPOrganizationFollowsThePluginAndTheScope(t *testing.T) {
	tests := []struct {
		name  string
		block string
		scope InstallScope
		want  string
	}{
		{"plugin activo en workspace", adoBlock, ScopeWorkspace, "contoso"},
		{"plugin activo en ámbito de usuario", adoBlock, ScopeGlobal, ""},
		{"sin bloque", "", ScopeWorkspace, ""},
		{"plugin desactivado", "integrations:\n  azure-devops:\n    enabled: false\n    organization: contoso\n", ScopeWorkspace, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), integrationsBase+tt.block)
			if got := azureDevOpsMCPOrganization(tt.scope, root); got != tt.want {
				t.Fatalf("azureDevOpsMCPOrganization() = %q; want %q", got, tt.want)
			}
		})
	}
	if got := azureDevOpsMCPOrganization(ScopeWorkspace, t.TempDir()); got != "" {
		t.Fatalf("sin axiom.yaml = %q; want \"\"", got)
	}
}

// TestRunInstallWorkspaceInjectsAzureDevOpsMCPOnlyWithThePlugin instala el
// componente MCP en ámbito workspace con y sin el plugin azure-devops activo: con
// él, <proyecto>/.mcp.json declara también el servidor de Azure DevOps; sin él,
// solo Context7, como antes.
func TestRunInstallWorkspaceInjectsAzureDevOpsMCPOnlyWithThePlugin(t *testing.T) {
	for _, tt := range []struct {
		name  string
		block string
		want  bool
	}{
		{"con plugin activo", adoBlock, true},
		{"sin plugin", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			workspace := t.TempDir()
			writeCLIFixture(t, filepath.Join(workspace, "axiom.yaml"), integrationsBase+tt.block)

			restoreHome := osUserHomeDir
			restoreCommand := runCommand
			restoreLookPath := cmdLookPath
			restoreBackupHome := backup.UserHomeDirFn
			originalCwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				osUserHomeDir = restoreHome
				runCommand = restoreCommand
				cmdLookPath = restoreLookPath
				backup.UserHomeDirFn = restoreBackupHome
				if err := os.Chdir(originalCwd); err != nil {
					t.Errorf("no se pudo restaurar el directorio de trabajo: %v", err)
				}
			})
			osUserHomeDir = func() (string, error) { return home, nil }
			backup.UserHomeDirFn = func() (string, error) { return home, nil }
			runCommand = func(string, ...string) error { return nil }
			cmdLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
			if err := os.Chdir(workspace); err != nil {
				t.Fatal(err)
			}

			if _, err := RunInstall([]string{
				"--scope", "workspace",
				"--agent", "claude-code",
				"--component", "context7",
			}, system.DetectionResult{}); err != nil {
				t.Fatalf("RunInstall() error = %v", err)
			}

			raw, err := os.ReadFile(filepath.Join(workspace, ".mcp.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), `"context7"`) {
				t.Fatalf(".mcp.json debe declarar context7:\n%s", raw)
			}
			if got := strings.Contains(string(raw), `"azure-devops"`); got != tt.want {
				t.Fatalf(".mcp.json declara azure-devops = %v; want %v:\n%s", got, tt.want, raw)
			}
			if _, statErr := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(statErr) {
				t.Fatalf("el ámbito workspace no debe escribir ~/.claude.json; stat = %v", statErr)
			}
		})
	}
}
