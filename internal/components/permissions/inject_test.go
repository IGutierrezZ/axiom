package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/antigravity"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/codex"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/cursor"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/gemini"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/hermes"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/qwen"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// unmatchedAction is what remoteAction reports when no rule of the bash object
// matches the command: Axiom declares no catch-all, so the user's own default
// decides and the fixture cannot say what it is.
const unmatchedAction = "unmatched"

// Fixture port of OpenCode v1.2.27 util/wildcard.ts and permission/service.ts:
// anchored, case-sensitive POSIX matching, optional trailing " *", last match wins.
// Inputs are strings only; this does not emulate bash.ts's tree-sitter extraction.
func remoteAction(t *testing.T, raw []byte, command string) string {
	t.Helper()
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	var scalar string
	if json.Unmarshal(root["permission"], &scalar) == nil {
		return scalar
	}
	var permission map[string]json.RawMessage
	if err := json.Unmarshal(root["permission"], &permission); err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(permission["bash"], &scalar) == nil {
		return scalar
	}
	decoder := json.NewDecoder(strings.NewReader(string(permission["bash"])))
	_, _ = decoder.Token()
	action := unmatchedAction
	for decoder.More() {
		key, _ := decoder.Token()
		var value string
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		pattern := regexp.QuoteMeta(strings.ReplaceAll(key.(string), `\`, "/"))
		pattern = strings.ReplaceAll(strings.ReplaceAll(pattern, `\*`, ".*"), `\?`, ".")
		if strings.HasSuffix(pattern, " .*") {
			pattern = strings.TrimSuffix(pattern, " .*") + "( .*)?"
		}
		if regexp.MustCompile("(?s)^" + pattern + "$").MatchString(strings.ReplaceAll(command, `\`, "/")) {
			action = value
		}
	}
	return action
}

func TestRemoteMatcherBoundaryFixtures(t *testing.T) {
	// These are matcher inputs, not shell programs. bash.ts extracts command
	// nodes separately; no claim is made about parsing arbitrary shell syntax.
	for _, input := range []string{"/usr/bin/ssh example.invalid", "env ssh example.invalid", "true && ssh example.invalid", `python -c 'import subprocess'`} {
		if got := remoteAction(t, openCodeOverlayJSON, input); got != unmatchedAction {
			t.Errorf("unsupported matcher input %q unexpectedly intercepted: %s", input, got)
		}
	}
}

func TestRemoteCommandApprovalDefaults(t *testing.T) {
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		for _, seed := range []string{`{}`, `{"permission":{"bash":{"*":"allow"}}}`, `{"permission":{"bash":{"*":"allow","ssh*":"deny"}}}`, `{"permission":"deny"}`, `{"permission":{"bash":"ask"}}`, `{"permission":{"bash":"deny"}}`, `{"permission":{"bash":{"*":"deny"}}}`, `{"permission":{"bash":{"*":"allow","ssh *":"deny"}}}`, `{"permission":{"bash":{"*":"allow","ssh*":"allow"}}}`} {
			t.Run(string(id)+seed, func(t *testing.T) {
				home := t.TempDir()
				adapter, _ := agents.NewAdapter(id)
				path := adapter.SettingsPath(home)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
					t.Fatal(err)
				}
				if _, err := Inject(home, adapter); err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				for _, command := range []string{"ssh", "ssh example.invalid", "scp", "scp file example.invalid:file", "sftp", "sftp example.invalid", "rsync", "rsync source destination"} {
					want := "ask"
					if seed == `{"permission":"deny"}` || seed == `{"permission":{"bash":"deny"}}` || strings.Contains(seed, `"*":"deny"`) || (strings.Contains(seed, `"deny"`) && strings.HasPrefix(command, "ssh")) {
						want = "deny"
					}
					if strings.Contains(seed, `"ssh*":"allow"`) && strings.HasPrefix(command, "ssh") {
						want = "allow" // Explicit personal allow is not silently rewritten.
					}
					if got := remoteAction(t, raw, command); got != want {
						t.Errorf("%q: got %s, want %s", command, got, want)
					}
				}
				if result, err := Inject(home, adapter); err != nil || result.Changed {
					t.Fatalf("repeat injection = %+v, %v", result, err)
				}
			})
		}
	}
}

func claudeAdapter() agents.Adapter      { return claude.NewAdapter() }
func opencodeAdapter() agents.Adapter    { return opencode.NewAdapter() }
func geminiAdapter() agents.Adapter      { return gemini.NewAdapter() }
func cursorAdapter() agents.Adapter      { return cursor.NewAdapter() }
func qwenAdapter() agents.Adapter        { return qwen.NewAdapter() }
func vscodeAdapter() agents.Adapter      { return vscode.NewAdapter() }
func codexAdapter() agents.Adapter       { return codex.NewAdapter() }
func antigravityAdapter() agents.Adapter { return antigravity.NewAdapter() }
func hermesAdapter() agents.Adapter      { return hermes.NewAdapter() }

// codexInjectedLegacyConfig mirrors a config.toml produced by the retired
// gentle-dev permission profile injection, wrapped in user-authored content.
const codexInjectedLegacyConfig = `model = "gpt-5.5"
approval_policy = "on-request"
default_permissions = "gentle-dev"

[permissions.gentle-dev]
description = "Comfortable local development profile with workspace writes, network access, and read-only access to Git and Nix/Home Manager metadata."

[permissions.gentle-dev.network]
enabled = true

[permissions.gentle-dev.network.domains]
"*" = "allow"

[permissions.gentle-dev.filesystem]
glob_scan_max_depth = 6
":minimal" = "read"
"~/.config/git" = "read"
"~/.gitconfig" = "read"
"~/.local/state/nix/profiles/home-manager/home-path" = "read"
"~/.nix-profile" = "read"
"/nix/store" = "read"
":tmpdir" = "write"
":slash_tmp" = "write"

[permissions.gentle-dev.filesystem.":root"]
"." = "read"

[permissions.gentle-dev.filesystem.":workspace_roots"]
"." = "write"
".git/**" = "write"
"**/.env" = "deny"
"**/*.pem" = "deny"
"**/*.key" = "deny"

[permissions.gentle-dev.workspace_roots]
"~" = true

[mcp_servers.engram]
command = "engram"
args = ["mcp", "--tools=agent"]
`

func writeCodexConfig(t *testing.T, home, content string) string {
	t.Helper()
	configPath := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return configPath
}

// TestInjectHermesSkipsPermissions verifies that Hermes returns nil (no file written)
// because Hermes permission format is undocumented — §14 of spec.
func TestInjectHermesSkipsPermissions(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, hermesAdapter())
	if err != nil {
		t.Fatalf("Inject(hermes) error = %v", err)
	}
	if result.Changed {
		t.Fatal("Inject(hermes) changed = true, want false (no file should be written)")
	}
	if len(result.Files) != 0 {
		t.Fatalf("Inject(hermes) files = %v, want [] (no file should be written)", result.Files)
	}

	// Confirm no config.yaml or settings file was created.
	hermesDir := filepath.Join(home, ".hermes")
	if _, err := os.Stat(hermesDir); err == nil {
		t.Fatal("Inject(hermes) created ~/.hermes directory, want no files written")
	}
}

func TestInjectOpenCodeIsIdempotent(t *testing.T) {
	home := t.TempDir()

	first, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}

	second, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true")
	}

	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file %q: %v", path, err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, `"permission"`) {
		t.Fatal("opencode.json missing permission key")
	}
	if strings.Contains(text, `"permissions"`) {
		t.Fatal("opencode.json should use 'permission' (singular), not 'permissions'")
	}
	if !strings.Contains(text, `"bash"`) {
		t.Fatal("opencode.json permission missing bash section")
	}
	if !strings.Contains(text, `"read"`) {
		t.Fatal("opencode.json permission missing read section")
	}
}

func TestInjectAddsEnvToDenyList(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file %q: %v", settingsPath, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal settings json: %v", err)
	}

	permissionsNode, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions node missing or invalid: %#v", settings["permissions"])
	}

	denyList, ok := permissionsNode["deny"].([]any)
	if !ok {
		t.Fatalf("deny list missing or invalid: %#v", permissionsNode["deny"])
	}

	for _, entry := range denyList {
		if value, ok := entry.(string); ok && value == "Read(.env)" {
			return
		}
	}

	t.Fatalf("deny list missing explicit .env rule: %#v", denyList)
}

// settingsOf reads and decodes the JSON settings file at path.
func settingsOf(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings file %q: %v", path, err)
	}
	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal %q: %v", path, err)
	}
	return settings
}

// isolateEditorUserDirs keeps the VS Code adapter's user directory under home so
// a test never reaches the real editor settings.
func isolateEditorUserDirs(t *testing.T, home string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
}

// autoApprovalCase is an agent whose overlay used to impose an auto-approval
// mode, with a seed that already carries the user's own value.
type autoApprovalCase struct {
	name    string
	adapter agents.Adapter
	seed    string
	// none is true when the agent has no overlay left, so Inject must not
	// create or write its file at all.
	none bool
}

func autoApprovalCases() []autoApprovalCase {
	return []autoApprovalCase{
		{"gemini-cli", geminiAdapter(), `{"general":{"defaultApprovalMode":"auto_edit"}}`, true},
		{"qwen-code", qwenAdapter(), `{"permissions":{"defaultMode":"auto_edit"}}`, true},
		{"vscode-copilot", vscodeAdapter(), `{"chat.tools.autoApprove": true}`, true},
		{"claude-code", claudeAdapter(), `{"permissions":{"defaultMode":"bypassPermissions"}}`, false},
	}
}

func TestInjectNeverWritesAutoApprovalModes(t *testing.T) {
	forbidden := []string{"bypassPermissions", "auto_edit", "chat.tools.autoApprove", "defaultMode", "defaultApprovalMode"}

	for _, id := range []model.AgentID{
		model.AgentClaudeCode, model.AgentOpenCode, model.AgentKilocode,
		model.AgentGeminiCLI, model.AgentQwenCode, model.AgentVSCodeCopilot,
	} {
		t.Run(string(id), func(t *testing.T) {
			home := t.TempDir()
			isolateEditorUserDirs(t, home)
			adapter, err := agents.NewAdapter(id)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := Inject(home, adapter); err != nil {
				t.Fatalf("Inject() error = %v", err)
			}

			path := adapter.SettingsPath(home)
			content, err := os.ReadFile(path)
			if agentOverlay(id) == nil {
				if !os.IsNotExist(err) {
					t.Fatalf("%s has no overlay but %q exists (read err = %v): no file may be created", id, path, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("read %q: %v", path, err)
			}
			for _, word := range forbidden {
				if strings.Contains(string(content), word) {
					t.Errorf("%s settings contain %q:\n%s", id, word, content)
				}
			}
			if id == model.AgentOpenCode || id == model.AgentKilocode {
				var settings struct {
					Permission struct {
						Bash map[string]string `json:"bash"`
						Read map[string]string `json:"read"`
					} `json:"permission"`
				}
				if err := json.Unmarshal(content, &settings); err != nil {
					t.Fatal(err)
				}
				for tool, rules := range map[string]map[string]string{"bash": settings.Permission.Bash, "read": settings.Permission.Read} {
					for pattern, action := range rules {
						if action == "allow" {
							t.Errorf("%s permission.%s[%q] = allow, want no catch-all allow", id, tool, pattern)
						}
					}
				}
			}
		})
	}
}

func TestInjectLeavesUserAutoApprovalValuesUntouched(t *testing.T) {
	home := t.TempDir()
	isolateEditorUserDirs(t, home)

	for _, tt := range autoApprovalCases() {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.adapter.SettingsPath(home)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tt.seed), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := Inject(home, tt.adapter)
			if err != nil {
				t.Fatalf("Inject() error = %v", err)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tt.none {
				if result.Changed || len(result.Files) != 0 {
					t.Fatalf("Inject() = %+v, want no change for an agent without overlay", result)
				}
				if string(after) != tt.seed {
					t.Fatalf("settings = %q, want byte-identical %q", after, tt.seed)
				}
				return
			}
			// Claude keeps the user's mode and only gains the deny rules.
			perms, _ := settingsOf(t, path)["permissions"].(map[string]any)
			if perms["defaultMode"] != "bypassPermissions" {
				t.Fatalf("user defaultMode = %v, want bypassPermissions preserved", perms["defaultMode"])
			}
			if _, ok := perms["deny"].([]any); !ok {
				t.Fatalf("deny list missing after injection: %#v", perms)
			}
		})
	}
}

func TestInjectOpenCodeKeepsUserCatchAllAllow(t *testing.T) {
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		t.Run(string(id), func(t *testing.T) {
			home := t.TempDir()
			adapter, _ := agents.NewAdapter(id)
			path := adapter.SettingsPath(home)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(`{"permission":{"bash":{"*":"allow"},"read":{"*":"allow"}}}`), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := Inject(home, adapter); err != nil {
				t.Fatalf("Inject() error = %v", err)
			}

			permission, _ := settingsOf(t, path)["permission"].(map[string]any)
			for _, tool := range []string{"bash", "read"} {
				rules, _ := permission[tool].(map[string]any)
				if rules["*"] != "allow" {
					t.Errorf("user's %s catch-all = %v, want allow preserved", tool, rules["*"])
				}
			}
		})
	}
}

func TestInjectVSCodeCopilotLeavesJSONCSettingsUntouched(t *testing.T) {
	home := t.TempDir()
	isolateEditorUserDirs(t, home)

	adapter := vscodeAdapter()
	settingsPath := adapter.SettingsPath(home)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	baseSettings := `{
	  // User has comments and trailing commas in VS Code settings
	  "editor.formatOnSave": true,
	  "files.exclude": {
	    "**/.git": true,
	  },
	}
`
	if err := os.WriteFile(settingsPath, []byte(baseSettings), 0o644); err != nil {
		t.Fatalf("WriteFile(settings.json) error = %v", err)
	}

	result, err := Inject(home, adapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.Changed {
		t.Fatal("Inject() changed = true, want false: VS Code has no protective overlay")
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}
	if string(content) != baseSettings {
		t.Fatalf("settings = %q, want byte-identical %q", content, baseSettings)
	}
}

// claudeDenyFromFile returns the permissions.deny list of a Claude settings file.
func claudeDenyFromFile(t *testing.T, path string) []string {
	t.Helper()
	perms, _ := settingsOf(t, path)["permissions"].(map[string]any)
	raw, _ := perms["deny"].([]any)
	deny := make([]string, 0, len(raw))
	for _, entry := range raw {
		deny = append(deny, entry.(string))
	}
	return deny
}

func TestInjectClaudeCodeUnionsUserDenyList(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// "Read(.env)" is also an Axiom entry: it must not be duplicated, and the
	// user's entries keep their order ahead of everything Axiom adds.
	seed := `{"permissions":{"deny":["Bash(curl *)","Read(.env)","WebFetch"],"allow":["Bash(ls)"]}}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	deny := claudeDenyFromFile(t, settingsPath)
	wantHead := []string{"Bash(curl *)", "Read(.env)", "WebFetch"}
	if len(deny) < len(wantHead) || strings.Join(deny[:len(wantHead)], "|") != strings.Join(wantHead, "|") {
		t.Fatalf("deny list must start with the user's entries in order %v, got %v", wantHead, deny)
	}
	seen := map[string]int{}
	for _, entry := range deny {
		seen[entry]++
		if seen[entry] > 1 {
			t.Errorf("deny entry %q duplicated in %v", entry, deny)
		}
	}
	for _, axiomEntry := range []string{"Bash(rm -rf /)", "Read(~/.ssh/**)", "Edit(**/secrets/*)"} {
		if seen[axiomEntry] != 1 {
			t.Errorf("Axiom deny entry %q missing from %v", axiomEntry, deny)
		}
	}
	perms, _ := settingsOf(t, settingsPath)["permissions"].(map[string]any)
	if allow, _ := perms["allow"].([]any); len(allow) != 1 || allow[0] != "Bash(ls)" {
		t.Errorf("user's allow list = %v, want it untouched", perms["allow"])
	}
}

func TestInjectClaudeCodeDenyUnionIsIdempotent(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"permissions":{"deny":["Bash(curl *)"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Inject(home, claudeAdapter())
	if err != nil || !first.Changed {
		t.Fatalf("first Inject() = %+v, %v; want changed", first, err)
	}
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Inject(home, claudeAdapter())
	if err != nil || second.Changed {
		t.Fatalf("second Inject() = %+v, %v; want unchanged", second, err)
	}
	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("second injection rewrote the file:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestInjectClaudeCodeWithoutDenyListWritesAxiomRules(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	deny := claudeDenyFromFile(t, filepath.Join(home, ".claude", "settings.json"))
	if len(deny) == 0 || deny[0] != "Bash(rm -rf /)" {
		t.Fatalf("deny list = %v, want Axiom's declared list in its own order", deny)
	}
}

// TestTargetPathFollowsRoot pins that the destination is rooted where the
// caller says (home for global, workspace for workspace), for every agent that
// has one, so no path is baked to the user's home.
func TestTargetPathFollowsRoot(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	for _, tt := range []struct {
		id   model.AgentID
		rel  []string
		name string
	}{
		{model.AgentClaudeCode, []string{".claude", "settings.json"}, "claude"},
		{model.AgentOpenCode, []string{".config", "opencode", "opencode.json"}, "opencode"},
		{model.AgentKilocode, []string{".config", "kilo", "opencode.json"}, "kilocode"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := agents.NewAdapter(tt.id)
			if err != nil {
				t.Fatal(err)
			}
			for _, root := range []string{home, workspace} {
				want := filepath.Join(append([]string{root}, tt.rel...)...)
				if got := TargetPath(root, adapter); got != want {
					t.Errorf("TargetPath(%q) = %q, want %q", root, got, want)
				}
			}
		})
	}
}

func TestTargetPathEmptyForAgentsWithoutOverlay(t *testing.T) {
	home := t.TempDir()
	for _, id := range []model.AgentID{model.AgentGeminiCLI, model.AgentQwenCode, model.AgentVSCodeCopilot} {
		adapter, err := agents.NewAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		if got := TargetPath(home, adapter); got != "" {
			t.Errorf("TargetPath(%s) = %q, want empty: no overlay left", id, got)
		}
	}
}

func TestInjectCursorSkipsPermissions(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, cursorAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.Changed {
		t.Fatal("Inject() for Cursor should not change anything (permissions via cli-config.json)")
	}
	if len(result.Files) != 0 {
		t.Fatalf("Inject() for Cursor should return no files, got %v", result.Files)
	}
}

func TestInjectAntigravitySkipsPermissions(t *testing.T) {
	overlay := agentOverlay(model.AgentAntigravity)
	if overlay != nil {
		t.Errorf("expected nil overlay for Antigravity, got %s", overlay)
	}
}

// TestInjectCodexNeverWritesConfig pins the decision that gentle-ai writes
// nothing to Codex's permissions configuration — neither a profile nor the
// legacy migration that used to strip one. Codex refuses to load a config that
// defines a [permissions.*] profile without default_permissions, so a cleanup
// that removed the pointer while a profile survived stopped Codex from starting
// at all (#1794). Whoever still carries an old gentle-dev profile keeps it.
//
// Byte equality alone is not the assertion: an atomic rewrite that happens to
// produce identical bytes is still a write. Every fixture is backdated so the
// modification time proves the file was left alone.
func TestInjectCodexNeverWritesConfig(t *testing.T) {
	backdated := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		content string
	}{
		{name: "user reported blocker", content: "approval_policy = \"on-request\"\ndefault_permissions = \"gentle-dev\"\n\n[permissions.gentle-dev]\nnetwork.enabled = true\n"},
		{name: "fully injected legacy profile", content: codexInjectedLegacyConfig},
		{name: "residual generated profile", content: "model = \"gpt-5.5\"\n\n[permissions.gentle-dev]\n\n[permissions.gentle-dev.workspace_roots]\n"},
		{name: "user owned gentle-dev content", content: "approval_policy = \"never\"\ndefault_permissions = \"gentle-dev\"\n\n[permissions.custom]\ndescription = \"user profile\"\n\n[permissions.gentle-dev] # user-owned\nuser_note = \"keep\"\n"},
		{name: "quoted gentle-dev forms", content: "permissions.\"gentle-dev\".workspace_roots.\"~\" = true\n"},
		{name: "dotted user value", content: "default_permissions = \"gentle-dev\"\npermissions.gentle-dev.custom_flag = true\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			configPath := writeCodexConfig(t, home, tt.content)
			if err := os.Chtimes(configPath, backdated, backdated); err != nil {
				t.Fatalf("Chtimes() error = %v", err)
			}

			result, err := Inject(home, codexAdapter())
			if err != nil {
				t.Fatalf("Inject() error = %v", err)
			}
			if result.Changed {
				t.Error("Inject(codex) changed = true, want false")
			}
			if len(result.Files) != 0 {
				t.Errorf("Inject(codex) files = %v, want none", result.Files)
			}

			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config.toml: %v", err)
			}
			if string(content) != tt.content {
				t.Errorf("config.toml = %q, want byte-identical %q", content, tt.content)
			}

			info, err := os.Stat(configPath)
			if err != nil {
				t.Fatalf("Stat(config.toml) error = %v", err)
			}
			if !info.ModTime().UTC().Equal(backdated) {
				t.Errorf("config.toml mtime = %v, want %v — the file was written, not left alone", info.ModTime().UTC(), backdated)
			}
		})
	}
}

func TestInjectCodexMissingConfigDoesNothing(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, codexAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.Changed {
		t.Fatal("Inject() changed = true, want false for missing config")
	}
	if len(result.Files) != 0 {
		t.Fatalf("Inject() files = %v, want none for missing config", result.Files)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("Inject() created ~/.codex (stat err = %v), want no file creation", err)
	}
}

func TestTargetPathCodexHasNoInjectionTarget(t *testing.T) {
	home := t.TempDir()
	if got := TargetPath(home, codexAdapter()); got != "" {
		t.Fatalf("TargetPath(codex) = %q, want empty (Codex relies on built-in defaults)", got)
	}
}

// TestInjectClaudeCodeSensitivePathsDenied verifies that the default sensitive-path
// deny list is present in the Claude Code permissions block.
func TestInjectClaudeCodeSensitivePathsDenied(t *testing.T) {
	sensitivePatterns := []string{
		"Read(~/.ssh/**)",
		"Edit(~/.ssh/**)",
		"Read(~/.credentials/**)",
		"Edit(~/.credentials/**)",
		"Read(~/Library/Keychains/**)",
		"Edit(~/Library/Keychains/**)",
		"Read(~/.aws/credentials)",
		"Edit(~/.aws/credentials)",
		"Read(~/.config/gh/hosts.yml)",
		"Edit(~/.config/gh/hosts.yml)",
		"Read(**/*.pem)",
		"Edit(**/*.pem)",
		"Read(**/*.key)",
		"Edit(**/*.key)",
		"Read(**/secrets/*)",
		"Edit(**/secrets/*)",
	}

	home := t.TempDir()
	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file %q: %v", settingsPath, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal settings json: %v", err)
	}

	permissionsNode, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions node missing or invalid: %#v", settings["permissions"])
	}

	denyList, ok := permissionsNode["deny"].([]any)
	if !ok {
		t.Fatalf("deny list missing or invalid: %#v", permissionsNode["deny"])
	}

	denySet := make(map[string]bool, len(denyList))
	for _, entry := range denyList {
		if v, ok := entry.(string); ok {
			denySet[v] = true
		}
	}

	for _, pattern := range sensitivePatterns {
		t.Run(pattern, func(t *testing.T) {
			if !denySet[pattern] {
				t.Errorf("deny list missing pattern %q; got: %v", pattern, denyList)
			}
		})
	}
}

// TestInjectOpenCodeSensitivePathsDenied verifies that the default sensitive-path
// deny list is present in the OpenCode/Kilocode read permissions block.
func TestInjectOpenCodeSensitivePathsDenied(t *testing.T) {
	sensitivePatterns := []string{
		"**/.ssh/**",
		"**/.credentials/**",
		"**/Library/Keychains/**",
		"**/.aws/credentials",
		"**/.config/gh/hosts.yml",
		"**/*.pem",
		"**/*.key",
	}

	tests := []struct {
		name    string
		adapter agents.Adapter
	}{
		{"opencode", opencodeAdapter()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			if _, err := Inject(home, tt.adapter); err != nil {
				t.Fatalf("Inject() error = %v", err)
			}

			settingsPath := tt.adapter.SettingsPath(home)
			content, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("read settings file %q: %v", settingsPath, err)
			}

			var settings map[string]any
			if err := json.Unmarshal(content, &settings); err != nil {
				t.Fatalf("unmarshal settings json: %v", err)
			}

			permNode, ok := settings["permission"].(map[string]any)
			if !ok {
				t.Fatalf("permission node missing or invalid: %#v", settings["permission"])
			}

			readNode, ok := permNode["read"].(map[string]any)
			if !ok {
				t.Fatalf("read node missing or invalid: %#v", permNode["read"])
			}

			for _, pattern := range sensitivePatterns {
				t.Run(pattern, func(t *testing.T) {
					val, exists := readNode[pattern]
					if !exists {
						t.Errorf("read deny list missing pattern %q", pattern)
						return
					}
					if val != "deny" {
						t.Errorf("pattern %q has value %q, want %q", pattern, val, "deny")
					}
				})
			}
		})
	}
}

// TestInjectClaudeCodeDefaultDenyRulesApplied ensures that the default deny
// rules (including sensitive paths) are written into settings.json even when
// a pre-existing permissions block is already present with other top-level keys.
func TestInjectClaudeCodeDefaultDenyRulesApplied(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Pre-existing settings with a sibling key under permissions (not deny).
	existing := `{
  "permissions": {
    "defaultMode": "default"
  }
}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions node missing")
	}

	denyList, ok := perms["deny"].([]any)
	if !ok {
		t.Fatalf("deny list missing")
	}

	denySet := make(map[string]bool, len(denyList))
	for _, entry := range denyList {
		if v, ok := entry.(string); ok {
			denySet[v] = true
		}
	}

	// Sensitive-path rules must be present after overlay application.
	for _, rule := range []string{"Read(~/.ssh/**)", "Read(**/*.pem)", "Read(**/*.key)"} {
		if !denySet[rule] {
			t.Errorf("default deny rule %q was not present; got: %v", rule, denyList)
		}
	}

	// Axiom never imposes a mode: the user's own defaultMode survives.
	mode, _ := perms["defaultMode"].(string)
	if mode != "default" {
		t.Errorf("expected the user's defaultMode=default to be preserved, got %q", mode)
	}
}

// TestInjectOpenCodePreservesExistingDenyRules ensures that user-managed read deny
// entries already present in settings.json are not removed when the overlay is applied.
func TestInjectOpenCodePreservesExistingDenyRules(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	existing := `{
  "permission": {
    "read": {
      "**/my-secret/**": "deny"
    }
  }
}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Inject(home, opencodeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	permNode, ok := settings["permission"].(map[string]any)
	if !ok {
		t.Fatalf("permission node missing")
	}

	readNode, ok := permNode["read"].(map[string]any)
	if !ok {
		t.Fatalf("read node missing")
	}

	// Original user rule must still be present
	if readNode["**/my-secret/**"] != "deny" {
		t.Errorf("user-managed read deny rule '**/my-secret/**' was removed; got: %v", readNode)
	}

	// New sensitive-path rules must also be present
	if readNode["**/.ssh/**"] != "deny" {
		t.Errorf("default read deny rule '**/.ssh/**' was not added; got: %v", readNode)
	}
}

// injectSeeded siembra el fichero de ajustes del agente con seed, inyecta el
// overlay y devuelve el contenido resultante.
func injectSeeded(t *testing.T, id model.AgentID, seed string) (string, string, agents.Adapter, []byte) {
	t.Helper()
	home := t.TempDir()
	adapter, _ := agents.NewAdapter(id)
	path := adapter.SettingsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inject(home, adapter); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return home, path, adapter, raw
}

// Un usuario con deny por defecto lo conserva: ninguna regla del overlay más
// laxa que el "*" raíz entra en un mapa de herramienta que el usuario acotó,
// donde seguiría a la regla raíz y ganaría como última coincidencia.
func TestInjectKeepsUserDenyByDefault(t *testing.T) {
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		t.Run(string(id), func(t *testing.T) {
			home, path, adapter, raw := injectSeeded(t, id, `{"permission":{"*":"deny","bash":{"git status":"allow"}}}`)
			var root struct {
				Permission map[string]json.RawMessage `json:"permission"`
			}
			if err := json.Unmarshal(raw, &root); err != nil {
				t.Fatal(err)
			}
			if string(root.Permission["*"]) != `"deny"` {
				t.Errorf("root deny changed: %s", root.Permission["*"])
			}
			for tool, value := range root.Permission {
				var patterns map[string]string
				if json.Unmarshal(value, &patterns) != nil {
					continue
				}
				if action, ok := patterns["*"]; ok {
					t.Errorf("%s gained \"*\": %q over the user's deny-by-default", tool, action)
				}
				for pattern, action := range patterns {
					if action != "deny" && !(tool == "bash" && pattern == "git status") {
						t.Errorf("%s %q: %s loosens the user's deny-by-default", tool, pattern, action)
					}
				}
			}
			if got := remoteAction(t, raw, "git push"); got == "ask" {
				t.Errorf("overlay ask turned a denied command into a prompt: %s", got)
			}
			if got := remoteAction(t, raw, "git status"); got != "allow" {
				t.Errorf("user's own allow: got %s, want allow", got)
			}
			if result, err := Inject(home, adapter); err != nil || result.Changed {
				t.Fatalf("repeat injection = %+v, %v (%s)", result, err, path)
			}
		})
	}
}

// Con un "*" raíz de ask, el overlay conserva sus ask y deny pero no añade
// ningún allow que siguiera a la regla raíz.
func TestInjectKeepsUserAskByDefault(t *testing.T) {
	_, _, _, raw := injectSeeded(t, model.AgentOpenCode, `{"permission":{"*":"ask","bash":{"ls":"allow"}}}`)
	var root struct {
		Permission map[string]map[string]string `json:"permission"`
	}
	_ = json.Unmarshal(raw, &root)
	for pattern, action := range root.Permission["bash"] {
		if action == "allow" && pattern != "ls" {
			t.Errorf("bash %q: allow loosens the user's ask-by-default", pattern)
		}
	}
	if got := remoteAction(t, raw, "git push"); got != "ask" {
		t.Errorf("overlay ask lost: got %s", got)
	}
}

// Un "*" raíz en forma de mapa de patrones con deny también es deny por
// defecto: el overlay no añade reglas más laxas y el filtro tolera esa forma.
func TestInjectToleratesPatternMapRootDefault(t *testing.T) {
	_, _, _, raw := injectSeeded(t, model.AgentKilocode, `{"permission":{"*":{"*":"deny"}}}`)
	var root struct {
		Permission map[string]json.RawMessage `json:"permission"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	for tool, value := range root.Permission {
		var patterns map[string]string
		if tool == "*" || json.Unmarshal(value, &patterns) != nil {
			continue
		}
		for pattern, action := range patterns {
			if action != "deny" {
				t.Errorf("%s %q: %s loosens the user's deny-by-default", tool, pattern, action)
			}
		}
	}
}

// Un "*": "allow" del usuario no es un valor por defecto estricto: el overlay
// se aplica intacto, con sus ask de bash.
func TestInjectKeepsOverlayIntactUnderUserAllowByDefault(t *testing.T) {
	_, _, _, raw := injectSeeded(t, model.AgentOpenCode, `{"permission":{"*":"allow"}}`)
	if got := remoteAction(t, raw, "git push"); got != "ask" {
		t.Errorf("overlay ask dropped under allow default: got %s", got)
	}
	if got := remoteAction(t, raw, "git reset --hard HEAD"); got != "ask" {
		t.Errorf("overlay ask dropped under allow default: got %s", got)
	}
}

// TestInjectClaudeCodeAnchorsStaleHomeDenyRules verifica que las reglas antiguas
// de Axiom, relativas al proyecto, se reescriben en su sitio como reglas ancladas
// al HOME, sin duplicados y sin tocar las reglas del usuario.
func TestInjectClaudeCodeAnchorsStaleHomeDenyRules(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Read(~/.aws/credentials) ya está anclada: la antigua debe descartarse.
	seed := `{"permissions":{"deny":["Bash(curl *)","Read(.ssh/*)","Read(custom/**)","Edit(.config/gh/hosts.yml)","Read(~/.aws/credentials)","Read(.aws/credentials)"]}}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Inject(home, claudeAdapter())
	if err != nil || !first.Changed {
		t.Fatalf("first Inject() = %+v, %v; want changed", first, err)
	}

	deny := claudeDenyFromFile(t, settingsPath)
	wantHead := []string{"Bash(curl *)", "Read(~/.ssh/**)", "Read(custom/**)", "Edit(~/.config/gh/hosts.yml)", "Read(~/.aws/credentials)"}
	if len(deny) < len(wantHead) || strings.Join(deny[:len(wantHead)], "|") != strings.Join(wantHead, "|") {
		t.Fatalf("deny list must start with %v, got %v", wantHead, deny)
	}
	seen := map[string]int{}
	for _, entry := range deny {
		seen[entry]++
		if seen[entry] > 1 {
			t.Errorf("deny entry %q duplicated in %v", entry, deny)
		}
	}
	for stale := range staleClaudeCodeDenyRules {
		if seen[stale] != 0 {
			t.Errorf("stale deny rule %q still present in %v", stale, deny)
		}
	}
	if seen["Read(custom/**)"] != 1 {
		t.Errorf("user rule Read(custom/**) lost from %v", deny)
	}

	second, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("second Inject() error = %v", err)
	}
	if second.Changed {
		t.Fatalf("second Inject() Changed = true, want false (idempotent)")
	}
}

func TestLegacyOverlayOnlyForClaudeCode(t *testing.T) {
	if LegacyOverlay(model.AgentOpenCode) != nil {
		t.Fatal("LegacyOverlay(opencode) must be nil")
	}
	var legacy struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(LegacyOverlay(model.AgentClaudeCode), &legacy); err != nil {
		t.Fatal(err)
	}
	if len(legacy.Permissions.Deny) != len(staleClaudeCodeDenyRules) {
		t.Fatalf("LegacyOverlay deny = %v, want %d stale rules", legacy.Permissions.Deny, len(staleClaudeCodeDenyRules))
	}
}
