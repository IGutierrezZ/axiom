package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// permissionsTreeSnapshot maps every file under root (relative path) to its
// content, so a test can prove a root was left byte-identical.
func permissionsTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		snapshot[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %q: %v", root, err)
	}
	return snapshot
}

func assertSameTree(t *testing.T, label string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("%s changed: files before=%d after=%d\nbefore=%v\nafter=%v", label, len(before), len(after), keysOf(before), keysOf(after))
	}
	for path, content := range before {
		if got, ok := after[path]; !ok || got != content {
			t.Fatalf("%s: %q is no longer byte-identical", label, path)
		}
	}
}

func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// seedAgentConfigs writes a settings file for every permission agent under
// root, each holding the user's own values, and returns the paths.
func seedAgentConfigs(t *testing.T, root string) []string {
	t.Helper()
	seeds := map[string]string{
		filepath.Join(root, ".claude", "settings.json"):             `{"permissions":{"defaultMode":"default","deny":["WebFetch"]}}`,
		filepath.Join(root, ".config", "opencode", "opencode.json"): `{"permission":{"bash":{"*":"ask"}}}`,
		filepath.Join(root, ".config", "kilo", "opencode.json"):     `{"permission":{"bash":{"*":"ask"}}}`,
		filepath.Join(root, ".gemini", "settings.json"):             `{"general":{"defaultApprovalMode":"default"}}`,
	}
	paths := make([]string, 0, len(seeds))
	for path, content := range seeds {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

var permissionScopeAgents = []model.AgentID{
	model.AgentClaudeCode, model.AgentOpenCode, model.AgentKilocode,
	model.AgentGeminiCLI, model.AgentQwenCode, model.AgentVSCodeCopilot, model.AgentCodex,
}

// permissionTargets are the files a permission run is expected to write under
// its root for permissionScopeAgents.
func permissionTargets(root string) []string {
	return []string{
		filepath.Join(root, ".claude", "settings.json"),
		filepath.Join(root, ".config", "opencode", "opencode.json"),
		filepath.Join(root, ".config", "kilo", "opencode.json"),
	}
}

// workspacePermissionTargets son los ficheros que escribe una ejecución de permisos con
// el ámbito workspace: OpenCode lee opencode.json de la raíz y Kilo su kilo.json de .kilo.
func workspacePermissionTargets(root string) []string {
	return []string{
		filepath.Join(root, ".claude", "settings.json"),
		filepath.Join(root, "opencode.json"),
		filepath.Join(root, ".kilo", "kilo.json"),
	}
}

func isolatePermissionEnv(t *testing.T, home string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
}

func TestPermissionsRoot(t *testing.T) {
	home, workspace := "/home/u", "/work/p"
	for _, tt := range []struct {
		name      string
		scope     InstallScope
		workspace string
		agent     model.AgentID
		wantRoot  string
		wantOK    bool
	}{
		{"global claude", ScopeGlobal, workspace, model.AgentClaudeCode, home, true},
		{"global unsupported agent keeps home", ScopeGlobal, workspace, model.AgentVSCodeCopilot, home, true},
		{"workspace claude", ScopeWorkspace, workspace, model.AgentClaudeCode, workspace, true},
		{"workspace opencode", ScopeWorkspace, workspace, model.AgentOpenCode, workspace, true},
		{"workspace kilocode", ScopeWorkspace, workspace, model.AgentKilocode, workspace, true},
		{"workspace vscode skipped", ScopeWorkspace, workspace, model.AgentVSCodeCopilot, "", false},
		{"workspace trae skipped", ScopeWorkspace, workspace, model.AgentTrae, "", false},
		{"workspace windsurf skipped", ScopeWorkspace, workspace, model.AgentWindsurf, "", false},
		{"workspace antigravity skipped", ScopeWorkspace, workspace, model.AgentAntigravity, "", false},
		// Codex ya tiene modo workspace; sigue sin destino de permisos (TargetPath
		// vacío), así que el componente no escribe nada para él en ningún ámbito.
		{"workspace codex", ScopeWorkspace, workspace, model.AgentCodex, workspace, true},
		{"workspace without root never falls back to home", ScopeWorkspace, " ", model.AgentClaudeCode, "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := agents.NewAdapter(tt.agent)
			if err != nil {
				t.Fatal(err)
			}
			root, ok := permissionsRoot(home, tt.workspace, tt.scope, adapter)
			if root != tt.wantRoot || ok != tt.wantOK {
				t.Fatalf("permissionsRoot() = (%q, %v), want (%q, %v)", root, ok, tt.wantRoot, tt.wantOK)
			}
		})
	}
}

func TestComponentPathsPermissionsFollowScope(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	isolatePermissionEnv(t, home)
	for _, tt := range []struct {
		name  string
		scope InstallScope
		root  string
		want  []string
	}{
		{"global", ScopeGlobal, home, permissionTargets(home)},
		{"workspace", ScopeWorkspace, workspace, workspacePermissionTargets(workspace)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adapters := resolveAdaptersScoped(tt.scope, workspace, permissionScopeAgents)
			paths := componentPathsWithWorkspaceScoped(home, workspace, tt.scope, model.Selection{}, adapters, model.ComponentPermission)

			want := tt.want
			if len(paths) != len(want) {
				t.Fatalf("paths = %v, want exactly %v (agents without overlay contribute none)", paths, want)
			}
			for _, path := range want {
				if !containsPath(paths, path) {
					t.Errorf("missing %q in %v", path, paths)
				}
			}
		})
	}
}

func TestComponentPathsPermissionsWorkspaceSkipsAgentsWithoutWorkspaceMode(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	adapters := resolveAdapters([]model.AgentID{model.AgentVSCodeCopilot, model.AgentCodex, model.AgentTrae, model.AgentWindsurf, model.AgentAntigravity})

	paths := componentPathsWithWorkspaceScoped(home, workspace, ScopeWorkspace, model.Selection{}, adapters, model.ComponentPermission)

	if len(paths) != 0 {
		t.Fatalf("paths = %v, want none for agents without workspace mode", paths)
	}
}

func TestBackupTargetsPermissionsFollowScope(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	isolatePermissionEnv(t, home)
	adapters := resolveAdapters([]model.AgentID{model.AgentClaudeCode, model.AgentOpenCode, model.AgentKilocode})

	// backupTargets and the post-apply verification both read this one list.
	for _, tt := range []struct {
		scope InstallScope
		root  string
		other string
	}{
		{ScopeWorkspace, workspace, home},
		{ScopeGlobal, home, workspace},
	} {
		paths := componentPathsWithWorkspaceScoped(home, workspace, tt.scope, model.Selection{}, adapters, model.ComponentPermission)
		for _, path := range paths {
			if !strings.HasPrefix(path, tt.root) || strings.HasPrefix(path, tt.other) {
				t.Errorf("scope %s: path %q is not under %q", tt.scope, path, tt.root)
			}
		}
	}
}

func TestPermissionApplyStepFollowsScope(t *testing.T) {
	for _, tt := range []struct {
		name  string
		scope InstallScope
	}{
		{"workspace", ScopeWorkspace},
		{"global", ScopeGlobal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			workspace := t.TempDir()
			isolatePermissionEnv(t, home)
			root, other := workspace, home
			if tt.scope == ScopeGlobal {
				root, other = home, workspace
			}
			// Seed the user's own configs under both roots. Whatever sits under
			// the root the scope does not select must come out byte-identical.
			seedAgentConfigs(t, home)
			seedAgentConfigs(t, workspace)
			otherBefore := permissionsTreeSnapshot(t, other)

			step := componentApplyStep{
				component:    model.ComponentPermission,
				homeDir:      home,
				workspaceDir: workspace,
				scope:        tt.scope,
				agents:       permissionScopeAgents,
			}
			if err := step.Run(); err != nil {
				t.Fatalf("componentApplyStep.Run() error = %v", err)
			}

			assertSameTree(t, "the root outside the selected scope", otherBefore, permissionsTreeSnapshot(t, other))
			claude, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
			if err != nil || !strings.Contains(string(claude), "Read(.env)") || !strings.Contains(string(claude), "WebFetch") {
				t.Fatalf("claude settings under %q = %q (%v), want Axiom's deny rules unioned with the user's", root, claude, err)
			}
			gemini, err := os.ReadFile(filepath.Join(root, ".gemini", "settings.json"))
			if err != nil || string(gemini) != `{"general":{"defaultApprovalMode":"default"}}` {
				t.Fatalf("gemini settings = %q (%v), want untouched", gemini, err)
			}
		})
	}
}

func TestPermissionSyncStepFollowsScope(t *testing.T) {
	for _, tt := range []struct {
		name  string
		scope InstallScope
	}{
		{"workspace", ScopeWorkspace},
		{"global", ScopeGlobal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			workspace := t.TempDir()
			isolatePermissionEnv(t, home)
			root, other := workspace, home
			if tt.scope == ScopeGlobal {
				root, other = home, workspace
			}
			seedAgentConfigs(t, other)
			otherBefore := permissionsTreeSnapshot(t, other)

			var changed []string
			step := componentSyncStep{
				component:    model.ComponentPermission,
				homeDir:      home,
				workspaceDir: workspace,
				scope:        tt.scope,
				agents:       permissionScopeAgents,
				changedFiles: &changed,
			}
			if err := step.Run(); err != nil {
				t.Fatalf("componentSyncStep.Run() error = %v", err)
			}
			assertSameTree(t, "the root outside the selected scope", otherBefore, permissionsTreeSnapshot(t, other))

			targets := permissionTargets(root)
			if tt.scope == ScopeWorkspace {
				targets = workspacePermissionTargets(root)
			}
			for _, path := range targets {
				if _, err := os.Stat(path); err != nil {
					t.Errorf("sync did not write %q: %v", path, err)
				}
			}
			for _, path := range changed {
				if !strings.HasPrefix(path, root) {
					t.Errorf("sync reported %q outside the selected root %q", path, root)
				}
			}

			// A second sync is a no-op.
			changed = nil
			if err := step.Run(); err != nil {
				t.Fatalf("second componentSyncStep.Run() error = %v", err)
			}
			if len(changed) != 0 {
				t.Fatalf("second sync changed %v, want none", changed)
			}
		})
	}
}

// TestPermissionStepsWorkspaceNeverWriteHome is the headline guarantee: a
// workspace run leaves a fake home byte-identical for every agent, including
// those without a workspace mode.
func TestPermissionStepsWorkspaceNeverWriteHome(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	isolatePermissionEnv(t, home)
	seedAgentConfigs(t, home)
	homeBefore := permissionsTreeSnapshot(t, home)

	var out bytes.Buffer
	restore := permissionsNoticeWriter
	permissionsNoticeWriter = &out
	t.Cleanup(func() { permissionsNoticeWriter = restore })

	allAgents := append([]model.AgentID{model.AgentTrae, model.AgentWindsurf, model.AgentAntigravity}, permissionScopeAgents...)
	apply := componentApplyStep{component: model.ComponentPermission, homeDir: home, workspaceDir: workspace, scope: ScopeWorkspace, agents: allAgents}
	if err := apply.Run(); err != nil {
		t.Fatalf("apply: %v", err)
	}
	sync := componentSyncStep{component: model.ComponentPermission, homeDir: home, workspaceDir: workspace, scope: ScopeWorkspace, agents: allAgents}
	if err := sync.Run(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	assertSameTree(t, "home", homeBefore, permissionsTreeSnapshot(t, home))
	if out.Len() != 0 {
		t.Fatalf("agents with no permission target are skipped silently, got notice %q", out.String())
	}
}

func TestNoteSkippedPermissions(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	restore := permissionsNoticeWriter
	permissionsNoticeWriter = &out
	t.Cleanup(func() { permissionsNoticeWriter = restore })

	// An agent with a permission target is reported in Spanish.
	claude, _ := agents.NewAdapter(model.AgentClaudeCode)
	noteSkippedPermissions(home, claude)
	got := out.String()
	for _, want := range []string{"Permisos omitidos", "claude-code", "--scope global"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice %q missing %q", got, want)
		}
	}
	if strings.Count(got, "\n") != 1 {
		t.Errorf("notice must be a single line, got %q", got)
	}

	// An agent with nothing to inject has nothing to skip: no noise.
	out.Reset()
	for _, id := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentCodex, model.AgentTrae} {
		adapter, _ := agents.NewAdapter(id)
		noteSkippedPermissions(home, adapter)
	}
	if out.Len() != 0 {
		t.Errorf("agents without a permission target must stay silent, got %q", out.String())
	}
}
