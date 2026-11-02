package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

const claudeWorkspaceLegacyEngramJSON = "{\n  \"command\": \"/usr/local/bin/engram\",\n  \"args\": [\"mcp\", \"--tools=agent\"]\n}\n"

// TestRunInstallEngramClaudeWorkspaceScopeWritesProjectMCPJSON comprueba que la
// instalación de Engram con --scope workspace declara el servidor en
// <proyecto>/.mcp.json (lo que Claude Code lee), conserva los servidores del
// usuario y no escribe el registro de usuario ni el fichero .claude/mcp.
func TestRunInstallEngramClaudeWorkspaceScopeWritesProjectMCPJSON(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()

	restoreHome := osUserHomeDir
	restoreCommand := runCommand
	restoreLookPath := cmdLookPath
	restoreBackupHome := backup.UserHomeDirFn
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("no se pudo leer el directorio actual: %v", err)
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

	mcpPath := filepath.Join(workspace, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{"propio":{"command":"propio-mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("no se pudo cambiar al workspace temporal: %v", err)
	}

	result, err := RunInstall([]string{
		"--scope", "workspace",
		"--agent", "claude-code",
		"--component", "engram",
		"--preset", "custom",
	}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if !result.Verify.Ready {
		t.Fatalf("la verificación posterior falló: %#v", result.Verify)
	}

	raw, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"engram"`, `"propio"`, `"mcpServers"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf(".mcp.json debe contener %s; contenido:\n%s", want, raw)
		}
	}
	for _, absent := range []string{
		filepath.Join(workspace, ".claude", "mcp", "engram.json"),
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".claude", "mcp", "engram.json"),
		filepath.Join(home, ".mcp.json"),
	} {
		if _, statErr := os.Stat(absent); !os.IsNotExist(statErr) {
			t.Errorf("no debe existir %q tras instalar en workspace; stat = %v", absent, statErr)
		}
	}
}

// TestSyncBackupTargetsWorkspaceIncludeClaudeEngramLegacyMigrationSource asegura
// que la copia de seguridad de sync en workspace cubre el fichero heredado que la
// inyección migra y retira, además del .mcp.json de destino.
func TestSyncBackupTargetsWorkspaceIncludeClaudeEngramLegacyMigrationSource(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	selection := model.Selection{Agents: []model.AgentID{model.AgentClaudeCode}, Components: []model.ComponentID{model.ComponentEngram}}

	targets, err := syncBackupTargetsScoped(home, workspace, ScopeWorkspace, selection, resolveAdapters(selection.Agents))
	if err != nil {
		t.Fatalf("syncBackupTargetsScoped() error = %v", err)
	}
	for _, want := range []string{
		filepath.Join(workspace, ".mcp.json"),
		filepath.Join(workspace, ".claude", "mcp", "engram.json"),
	} {
		if !containsPath(targets, want) {
			t.Errorf("los objetivos de copia de sync en workspace no incluyen %q: %v", want, targets)
		}
	}
	for _, absent := range []string{
		filepath.Join(home, ".claude", "mcp", "engram.json"),
		filepath.Join(home, ".claude.json"),
	} {
		if containsPath(targets, absent) {
			t.Errorf("sync en workspace no debe declarar %q bajo el HOME: %v", absent, targets)
		}
	}
}

// TestRunSyncEngramClaudeWorkspaceMigratesLegacyFileToProjectMCPJSON cubre el
// camino de actualización: una instalación previa dejó .claude/mcp/engram.json,
// que Claude Code no lee. Sync lo migra a .mcp.json sin tocar el HOME.
func TestRunSyncEngramClaudeWorkspaceMigratesLegacyFileToProjectMCPJSON(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	legacy := filepath.Join(workspace, ".claude", "mcp", "engram.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(claudeWorkspaceLegacyEngramJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(workspace, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{"propio":{"command":"propio-mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	restoreBackupHome := backup.UserHomeDirFn
	backup.UserHomeDirFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { backup.UserHomeDirFn = restoreBackupHome })

	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode},
		Components: []model.ComponentID{model.ComponentEngram},
		Persona:    model.PersonaNeutral,
	}
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("RunSyncWithSelectionScoped() error = %v", err)
	}

	raw, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"engram"`, `"propio"`, "/usr/local/bin/engram"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf(".mcp.json debe contener %s tras sync; contenido:\n%s", want, raw)
		}
	}
	if _, statErr := os.Stat(legacy); !os.IsNotExist(statErr) {
		t.Errorf("el fichero heredado debe retirarse tras sync; stat = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(statErr) {
		t.Errorf("sync en workspace no debe escribir el registro de usuario; stat = %v", statErr)
	}
}
