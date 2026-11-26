package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/codex"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/versions"
)

func mustAdapter(t *testing.T, id string) agents.Adapter {
	t.Helper()
	adapter, err := agents.NewAdapter(model.AgentID(id))
	if err != nil {
		t.Fatalf("NewAdapter(%s) error = %v", id, err)
	}
	return adapter
}

func readJSONObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v\n%s", path, err, raw)
	}
	return root
}

func TestAzureDevOpsArgsPinVersionAndNeverCarryCredentials(t *testing.T) {
	args := AzureDevOpsArgs("contoso")
	want := []string{"-y", "@azure-devops/mcp@" + versions.AzureDevOpsMCP, "contoso", "-d", "core", "work", "work-items", "search"}
	if !slices.Equal(args, want) {
		t.Fatalf("AzureDevOpsArgs() = %v; want %v", args, want)
	}
	for _, arg := range args {
		if strings.Contains(arg, "authentication") || strings.Contains(strings.ToLower(arg), "pat") {
			t.Fatalf("AzureDevOpsArgs() must keep the default interactive (Entra) sign-in; got %q", arg)
		}
	}
}

func TestInjectAzureDevOpsSkipsUserScope(t *testing.T) {
	home := t.TempDir()
	for _, id := range []string{"claude-code", "cursor", "gemini-cli", "opencode", "codex"} {
		result, err := InjectAzureDevOps(home, home, mustAdapter(t, id), "contoso")
		if err != nil {
			t.Fatalf("%s: InjectAzureDevOps(user scope) error = %v", id, err)
		}
		if result.Changed || len(result.Files) != 0 {
			t.Fatalf("%s: user scope must not write; got %+v", id, result)
		}
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("user scope wrote %d entries in the home directory", len(entries))
	}
}

func TestInjectAzureDevOpsRejectsInvalidOrganization(t *testing.T) {
	home, workspace := t.TempDir(), t.TempDir()
	for _, organization := range []string{"", "-a", "contoso --authentication pat", "con/toso", "contoso;rm"} {
		if _, err := InjectAzureDevOps(home, workspace, claudeAdapter(), organization); err == nil {
			t.Fatalf("InjectAzureDevOps(%q) error = nil; want a rejection", organization)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("a rejected organization must not write .mcp.json; stat err = %v", err)
	}
}

func TestInjectAzureDevOpsWritesEachWorkspaceAgentAndIsIdempotent(t *testing.T) {
	cases := []struct {
		name    string
		adapter func(workspace string) agents.Adapter
		path    func(workspace string) string
		check   func(t *testing.T, path string)
	}{
		{
			name:    "claude-code",
			adapter: func(string) agents.Adapter { return claudeAdapter() },
			path:    func(ws string) string { return filepath.Join(ws, ".mcp.json") },
			check:   checkStdioEntry,
		},
		{
			name:    "vscode-copilot",
			adapter: func(ws string) agents.Adapter { return vscode.NewAdapter().WithWorkspace(ws) },
			path:    func(ws string) string { return filepath.Join(ws, ".mcp.json") },
			check:   checkStdioEntry,
		},
		{
			name:    "cursor",
			adapter: func(string) agents.Adapter { return mustAdapter(t, "cursor") },
			path:    func(ws string) string { return mustAdapter(t, "cursor").MCPConfigPath(ws, AzureDevOpsServerID) },
			check:   checkCommandArgsEntry,
		},
		{
			name:    "gemini-cli",
			adapter: func(string) agents.Adapter { return mustAdapter(t, "gemini-cli") },
			path:    func(ws string) string { return mustAdapter(t, "gemini-cli").SettingsPath(ws) },
			check:   checkCommandArgsEntry,
		},
		{
			name:    "opencode",
			adapter: func(string) agents.Adapter { return opencodeAdapter() },
			path:    func(ws string) string { return opencodeAdapter().SettingsPath(ws) },
			check:   checkOpenCodeEntry,
		},
		{
			name:    "codex",
			adapter: func(string) agents.Adapter { return codex.NewAdapter() },
			path:    func(ws string) string { return codex.NewAdapter().MCPConfigPath(ws, AzureDevOpsServerID) },
			check:   checkCodexEntry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, workspace := t.TempDir(), t.TempDir()
			adapter := tc.adapter(workspace)
			first, err := InjectAzureDevOps(home, workspace, adapter, "contoso")
			if err != nil {
				t.Fatalf("InjectAzureDevOps() first error = %v", err)
			}
			path := tc.path(workspace)
			if !first.Changed || !slices.Contains(first.Files, path) {
				t.Fatalf("InjectAzureDevOps() first = %+v; want a change in %s", first, path)
			}
			tc.check(t, path)

			second, err := InjectAzureDevOps(home, workspace, adapter, "contoso")
			if err != nil {
				t.Fatalf("InjectAzureDevOps() second error = %v", err)
			}
			if second.Changed {
				t.Fatalf("InjectAzureDevOps() second changed = true; want idempotent")
			}
		})
	}
}

func checkArgs(t *testing.T, got any) {
	t.Helper()
	raw, _ := json.Marshal(got)
	want, _ := json.Marshal(AzureDevOpsArgs("contoso"))
	if string(raw) != string(want) {
		t.Fatalf("args = %s; want %s", raw, want)
	}
}

func azureDevOpsServer(t *testing.T, path, root string) map[string]any {
	t.Helper()
	servers, _ := readJSONObject(t, path)[root].(map[string]any)
	entry, ok := servers[AzureDevOpsServerID].(map[string]any)
	if !ok {
		t.Fatalf("%s: %s.%s missing; got %#v", path, root, AzureDevOpsServerID, servers)
	}
	return entry
}

func checkStdioEntry(t *testing.T, path string) {
	entry := azureDevOpsServer(t, path, "mcpServers")
	if entry["type"] != "stdio" || entry["command"] != "npx" {
		t.Fatalf("entry = %#v; want type stdio and command npx", entry)
	}
	checkArgs(t, entry["args"])
}

func checkCommandArgsEntry(t *testing.T, path string) {
	entry := azureDevOpsServer(t, path, "mcpServers")
	if entry["command"] != "npx" {
		t.Fatalf("entry = %#v; want command npx", entry)
	}
	checkArgs(t, entry["args"])
}

func checkOpenCodeEntry(t *testing.T, path string) {
	entry := azureDevOpsServer(t, path, "mcp")
	if entry["type"] != "local" || entry["enabled"] != true {
		t.Fatalf("entry = %#v; want a local, enabled server", entry)
	}
	command, _ := entry["command"].([]any)
	if len(command) == 0 || command[0] != "npx" {
		t.Fatalf("command = %#v; want npx first", entry["command"])
	}
	checkArgs(t, command[1:])
}

func checkCodexEntry(t *testing.T, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{"[mcp_servers.azure-devops]", `command = "npx"`, `"@azure-devops/mcp@` + versions.AzureDevOpsMCP + `"`, `"contoso"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("config.toml lacks %q:\n%s", want, text)
		}
	}
}

func TestInjectAzureDevOpsCoexistsWithContext7AndSharedMCPJSONIsStable(t *testing.T) {
	home, workspace := t.TempDir(), t.TempDir()
	path := filepath.Join(workspace, ".mcp.json")
	claudeCode := claudeAdapter()
	copilot := vscode.NewAdapter().WithWorkspace(workspace)

	for _, step := range []func() (InjectionResult, error){
		func() (InjectionResult, error) { return Inject(home, workspace, claudeCode) },
		func() (InjectionResult, error) { return InjectAzureDevOps(home, workspace, claudeCode, "contoso") },
		func() (InjectionResult, error) { return InjectAzureDevOps(home, workspace, copilot, "contoso") },
	} {
		if _, err := step(); err != nil {
			t.Fatalf("injection error = %v", err)
		}
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	servers, _ := readJSONObject(t, path)["mcpServers"].(map[string]any)
	if _, ok := servers["context7"]; !ok {
		t.Fatalf(".mcp.json lost context7: %#v", servers)
	}

	again, err := InjectAzureDevOps(home, workspace, claudeCode, "contoso")
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || string(before) != string(after) {
		t.Fatalf("Claude Code and VS Code Copilot must write the same azure-devops entry; rewrite detected:\n%s\n---\n%s", before, after)
	}
}

func TestInjectAzureDevOpsReplacesTheOrganization(t *testing.T) {
	home, workspace := t.TempDir(), t.TempDir()
	if _, err := InjectAzureDevOps(home, workspace, claudeAdapter(), "contoso"); err != nil {
		t.Fatal(err)
	}
	if _, err := InjectAzureDevOps(home, workspace, claudeAdapter(), "fabrikam"); err != nil {
		t.Fatal(err)
	}
	entry := azureDevOpsServer(t, filepath.Join(workspace, ".mcp.json"), "mcpServers")
	raw, _ := json.Marshal(entry["args"])
	want, _ := json.Marshal(AzureDevOpsArgs("fabrikam"))
	if string(raw) != string(want) {
		t.Fatalf("args = %s; want %s (no stale organization)", raw, want)
	}
}
