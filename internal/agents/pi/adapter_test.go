package pi

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

func TestAdapterIdentityAndCapabilities(t *testing.T) {
	a := NewAdapter()

	if got := a.Agent(); got != model.AgentPi {
		t.Fatalf("Agent() = %q, want %q", got, model.AgentPi)
	}
	if got := a.Tier(); got != model.TierFull {
		t.Fatalf("Tier() = %q, want %q", got, model.TierFull)
	}

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"SupportsSkills", a.SupportsSkills(), false},
		{"SupportsMCP", a.SupportsMCP(), true},
		{"SupportsSystemPrompt", a.SupportsSystemPrompt(), false},
		{"SupportsSlashCommands", a.SupportsSlashCommands(), false},
		{"SupportsOutputStyles", a.SupportsOutputStyles(), false},
		{"SupportsSubAgents", a.SupportsSubAgents(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestAdapterPaths(t *testing.T) {
	a := NewAdapter()
	homeDir := t.TempDir()
	piDir := filepath.Join(homeDir, ".pi")
	piAgentDir := filepath.Join(piDir, "agent")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"GlobalConfigDir", a.GlobalConfigDir(homeDir), piDir},
		{"SystemPromptDir", a.SystemPromptDir(homeDir), piAgentDir},
		{"SystemPromptFile", a.SystemPromptFile(homeDir), filepath.Join(piAgentDir, "APPEND_SYSTEM.md")},
		{"SkillsDir", a.SkillsDir(homeDir), ""},
		{"SettingsPath", a.SettingsPath(homeDir), filepath.Join(piAgentDir, "settings.json")},
		{"CommandsDir", a.CommandsDir(homeDir), ""},
		{"MCPConfigPath", a.MCPConfigPath(homeDir, "context7"), filepath.Join(piAgentDir, "mcp.json")},
		{"OutputStyleDir", a.OutputStyleDir(homeDir), ""},
		{"SubAgentsDir", a.SubAgentsDir(homeDir), ""},
		{"EmbeddedSubAgentsDir", a.EmbeddedSubAgentsDir(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestCodeGraphPathsResolveConfiguredAgentDirectory(t *testing.T) {
	home := t.TempDir()
	configured := filepath.Join(home, "custom-pi")
	t.Setenv("PI_CODING_AGENT_DIR", configured)

	paths := CodeGraphPaths(home)
	if paths.AgentDir != configured {
		t.Fatalf("AgentDir = %q, want %q", paths.AgentDir, configured)
	}
	if paths.MCPConfig != filepath.Join(configured, "mcp.json") {
		t.Fatalf("MCPConfig = %q", paths.MCPConfig)
	}
	if paths.Manifest != filepath.Join(home, ".gentle-ai", "pi-codegraph.json") {
		t.Fatalf("Manifest = %q", paths.Manifest)
	}
}

func TestCodeGraphPathsKeepsAgentDirectoryWhenProjectMCPOverrides(t *testing.T) {
	home := t.TempDir()
	configured := filepath.Join(home, "custom-pi")
	workspace := filepath.Join(home, "project")
	t.Setenv("PI_CODING_AGENT_DIR", configured)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".mcp.json"), []byte(`{"mcpServers":{"codegraph":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := CodeGraphPaths(home)
	effective, err := EffectiveCodeGraphMCPPath(home, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if paths.AgentDir != configured || effective != filepath.Join(workspace, ".mcp.json") {
		t.Fatalf("agent=%q effective=%q, want configured agent and project config", paths.AgentDir, effective)
	}
}

func TestDiscoverCodeGraphChildrenUsesProjectOverrideAndPreservesPackageSource(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(home, "project")
	mustWrite := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(home, ".pi", "agent", "subagents", "worker.md"), "---\ntools: bash\n---\npackage worker\n")
	mustWrite(filepath.Join(workspace, ".pi", "subagents", "worker.md"), "---\ntools: bash, mcp\n---\nproject worker\n")
	mustWrite(filepath.Join(home, ".pi", "agent", "agents", "reader.md"), "---\ntools: read\n---\nreader\n")
	mustWrite(filepath.Join(home, ".pi", "agent", "node_modules", "gentle-pi", "subagents", "package-worker.md"), "---\ntools: bash\n---\npackage worker\n")

	children, err := DiscoverCodeGraphChildren(home, workspace)
	if err != nil {
		t.Fatalf("DiscoverCodeGraphChildren() error = %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("children = %#v, want three effective children", children)
	}
	if children[0].Name != "package-worker" || children[1].Name != "reader" || children[2].Name != "worker" {
		t.Fatalf("children = %#v, want sorted package-worker, reader, and worker", children)
	}
	if !children[0].PackageOwned || children[0].Target == children[0].Source {
		t.Fatalf("package worker = %#v, want owned overlay", children[0])
	}
	if children[2].Source != filepath.Join(workspace, ".pi", "subagents", "worker.md") || children[2].PackageOwned {
		t.Fatalf("worker = %#v, want project effective child", children[2])
	}
}

func TestDiscoverCodeGraphChildrenReturnsUnreadableDirectoryError(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent", "subagents"), 0o755); err != nil {
		t.Fatal(err)
	}
	previous := piWalkDir
	piWalkDir = func(path string, walkFn fs.WalkDirFunc) error {
		return &fs.PathError{Op: "readdir", Path: path, Err: fs.ErrPermission}
	}
	t.Cleanup(func() { piWalkDir = previous })

	_, err := DiscoverCodeGraphChildren(home, "")
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("DiscoverCodeGraphChildren() error = %v, want unreadable-directory error", err)
	}
}

func TestDiscoverCodeGraphChildrenUsesNormalizedRuntimeIdentity(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(home, "project")
	for path, body := range map[string]string{
		filepath.Join(home, ".pi", "agent", "subagents", "Worker.md"): "---\ntools: bash\n---\nuser\n",
		filepath.Join(workspace, ".pi", "subagents", "worker.md"):     "---\ntools: bash\n---\nproject\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	children, err := DiscoverCodeGraphChildren(home, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].Source != filepath.Join(workspace, ".pi", "subagents", "worker.md") {
		t.Fatalf("children = %#v, want one project runtime identity", children)
	}
}

func TestAdapterDetectUsesPiBinaryAndConfigPath(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".pi", "agent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{
		lookPath: func(file string) (string, error) {
			if file != "pi" {
				t.Fatalf("lookPath called with %q, want pi", file)
			}
			return "/usr/local/bin/pi", nil
		},
		statPath: defaultStat,
	}

	installed, binaryPath, configPath, configFound, err := a.Detect(context.Background(), homeDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !installed {
		t.Fatalf("Detect() installed = false, want true")
	}
	if binaryPath != "/usr/local/bin/pi" {
		t.Fatalf("Detect() binaryPath = %q, want /usr/local/bin/pi", binaryPath)
	}
	if configPath != configDir {
		t.Fatalf("Detect() configPath = %q, want %q", configPath, configDir)
	}
	if !configFound {
		t.Fatalf("Detect() configFound = false, want true")
	}
}

func TestAdapterDetectMissingPiBinary(t *testing.T) {
	homeDir := t.TempDir()
	a := &Adapter{
		lookPath: func(file string) (string, error) {
			return "", os.ErrNotExist
		},
		statPath: defaultStat,
	}

	installed, binaryPath, configPath, configFound, err := a.Detect(context.Background(), homeDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if installed {
		t.Fatalf("Detect() installed = true, want false")
	}
	if binaryPath != "" {
		t.Fatalf("Detect() binaryPath = %q, want empty", binaryPath)
	}
	if configPath != filepath.Join(homeDir, ".pi", "agent") {
		t.Fatalf("Detect() configPath = %q, want ~/.pi/agent under home", configPath)
	}
	if configFound {
		t.Fatalf("Detect() configFound = true, want false")
	}
}

func TestManagedPackageSourcesReturnsCanonicalCopy(t *testing.T) {
	want := []string{
		"npm:gentle-pi",
		"npm:gentle-engram",
		"npm:pi-mcp-adapter",
		"npm:@juicesharp/rpiv-ask-user-question",
		"npm:pi-web-access",
		"npm:pi-btw",
	}
	got := ManagedPackageSources()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ManagedPackageSources() = %v, want %v", got, want)
	}
	got[0] = "changed"
	if sources := ManagedPackageSources(); sources[0] != want[0] {
		t.Fatalf("ManagedPackageSources() exposed mutable adapter state: %v", sources)
	}
}

func TestAdapterInstallCommandSequenceUsesNpmWhenPnpmIsUnavailable(t *testing.T) {
	a := &Adapter{
		lookPath: func(file string) (string, error) {
			if file == "pnpm" {
				return "", os.ErrNotExist
			}
			return "/usr/local/bin/" + file, nil
		},
		statPath: defaultStat,
	}
	commands, err := a.InstallCommand(system.PlatformProfile{})
	if err != nil {
		t.Fatalf("InstallCommand() error = %v", err)
	}

	want := [][]string{
		{"pi", "install", "npm:gentle-pi"},
		{"pi", "install", "npm:gentle-engram"},
		{"pi", "install", "npm:pi-mcp-adapter"},
		{"npm", "exec", "--yes", "--package", "gentle-engram@latest", "--", "pi-engram", "init"},
		{"pi", "install", "npm:@juicesharp/rpiv-ask-user-question"},
		{"pi", "install", "npm:pi-web-access"},
		{"pi", "install", "npm:pi-btw"},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("InstallCommand() = %#v, want %#v", commands, want)
	}
}

func TestAdapterInstallCommandSequenceUsesNpmForEngramInitWhenPnpmIsAvailable(t *testing.T) {
	a := &Adapter{
		lookPath: func(file string) (string, error) {
			if file == "pnpm" {
				return "/usr/local/bin/pnpm", nil
			}
			return "", os.ErrNotExist
		},
		statPath: defaultStat,
	}
	commands, err := a.InstallCommand(system.PlatformProfile{})
	if err != nil {
		t.Fatalf("InstallCommand() error = %v", err)
	}

	want := []string{"npm", "exec", "--yes", "--package", "gentle-engram@latest", "--", "pi-engram", "init"}
	if !reflect.DeepEqual(commands[3], want) {
		t.Fatalf("InstallCommand()[3] = %#v, want %#v", commands[3], want)
	}
}

func TestAppendPiPackageKeepsSubagentsPackageWhileGentlePiIsPinnedBelowGentleAgents(t *testing.T) {
	kept := appendPiPackage([]any{"npm:gentle-pi@2.4.0", "npm:pi-subagents-j0k3r@1.5.13"}, "npm:pi-mcp-adapter")
	if !reflect.DeepEqual(kept, []any{"npm:gentle-pi@2.4.0", "npm:pi-subagents-j0k3r@1.5.13", "npm:pi-mcp-adapter"}) {
		t.Fatalf("appendPiPackage() with an old gentle-pi pin = %v, want the subagents package kept", kept)
	}
	dropped := appendPiPackage([]any{"npm:gentle-pi@2.5.0", "npm:pi-subagents-j0k3r"}, "npm:pi-mcp-adapter")
	if !reflect.DeepEqual(dropped, []any{"npm:gentle-pi@2.5.0", "npm:pi-mcp-adapter"}) {
		t.Fatalf("appendPiPackage() with gentle-pi 2.5.0 = %v, want the subagents package dropped", dropped)
	}
}

func TestMergePiSettingsFileRemovesRetiredCompanionPackages(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}
	initial := `{
  "packages": [
    "npm:@juicesharp/rpiv-todo",
    "npm:@juicesharp/rpiv-todo@2.9.0",
    "npm:pi-subagents-j0k3r",
    "npm:pi-subagents-j0k3r@1.5.13",
    "npm:@juicesharp/rpiv-ask-user-question",
    "npm:other@1.0.0"
  ]
}`
	if err := os.WriteFile(settingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	if _, err := mergePiSettingsFile(settingsPath); err != nil {
		t.Fatalf("mergePiSettingsFile() error = %v", err)
	}

	var settings struct {
		Packages []string `json:"packages"`
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal(settings) error = %v", err)
	}
	if !reflect.DeepEqual(settings.Packages, []string{"npm:@juicesharp/rpiv-ask-user-question", "npm:other@1.0.0", "npm:pi-mcp-adapter"}) {
		t.Fatalf("packages = %#v, want the retired todo and subagents-j0k3r packages gone and the rest untouched", settings.Packages)
	}
}

func TestMergePiSettingsFileRemovesLegacySubagentPackages(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}
	initial := `{
  "theme": "kanagawa",
  "packages": [
    "npm:pi-subagents",
    "npm:pi-subagents@1.0.0",
    "vendor/pi-subagents",
    "vendor/pi-subagents-fixed@0.0.1",
    "npm:pi-web-access",
    "npm:other@1.0.0"
  ]
}`
	if err := os.WriteFile(settingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	if _, err := mergePiSettingsFile(settingsPath); err != nil {
		t.Fatalf("mergePiSettingsFile() error = %v", err)
	}

	var settings struct {
		Packages []string `json:"packages"`
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal(settings) error = %v", err)
	}

	for _, forbidden := range []string{"npm:pi-subagents", "npm:pi-subagents@1.0.0", "vendor/pi-subagents", "vendor/pi-subagents-fixed@0.0.1"} {
		for _, pkg := range settings.Packages {
			if pkg == forbidden {
				t.Fatalf("packages still contains legacy subagent package %q: %#v", forbidden, settings.Packages)
			}
		}
	}
	if !reflect.DeepEqual(settings.Packages, []string{"npm:pi-web-access", "npm:other@1.0.0", "npm:pi-mcp-adapter"}) {
		t.Fatalf("packages = %#v", settings.Packages)
	}
}

// TestProvisionEngramMCPPreservesExactNumberTokens keeps unrelated number
// tokens verbatim in every Pi file it rewrites.
func TestProvisionEngramMCPPreservesExactNumberTokens(t *testing.T) {
	a := NewAdapter()
	home := t.TempDir()
	numbers := `"big":9007199254740993,"neg":-9007199254740995,"max":9223372036854775807,` +
		`"nested":{"list":[1.10,6.02214076e23,1E+2,{"min":-9223372036854775808}]},"huge":1e999`
	files := map[string]string{
		a.SettingsPath(home): `{"packages":["npm:pi-mcp-adapter"],` + numbers + `}`,
		filepath.Join(ConfigPath(home), "npm", "package.json"): `{"dependencies":{"other":"^1.0.0"},` + numbers + `}`,
	}
	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if _, _, err := a.ProvisionEngramMCP(home); err != nil {
		t.Fatalf("ProvisionEngramMCP() error = %v", err)
	}
	for path, original := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if string(body) == original {
			t.Fatalf("%s was not rewritten; the test must exercise a re-encode", path)
		}
		var root map[string]any
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.UseNumber()
		if err := decoder.Decode(&root); err != nil {
			t.Fatalf("decode %s error = %v", path, err)
		}
		assertExactPiNumbers(t, path, root)
	}

	// Strict single-document JSON stays required: no trailing documents.
	settingsPath := a.SettingsPath(home)
	if err := os.WriteFile(settingsPath, []byte(`{"packages":[]}{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ProvisionEngramMCP(home); err == nil || !strings.Contains(err.Error(), "unmarshal pi json file") {
		t.Fatalf("ProvisionEngramMCP() error = %v, want trailing document rejected", err)
	}
}

// assertExactPiNumbers checks each number of the fixture at its own path.
func assertExactPiNumbers(t *testing.T, name string, object map[string]any) {
	t.Helper()
	nested, _ := object["nested"].(map[string]any)
	list, _ := nested["list"].([]any)
	if len(list) != 4 {
		t.Fatalf("%s nested.list = %#v, want 4 entries", name, nested["list"])
	}
	last, _ := list[3].(map[string]any)
	got := []any{object["big"], object["neg"], object["max"], list[0], list[1], list[2], last["min"], object["huge"]}
	want := []string{"9007199254740993", "-9007199254740995", "9223372036854775807",
		"1.10", "6.02214076e23", "1E+2", "-9223372036854775808", "1e999"}
	for i := range want {
		if number, ok := got[i].(json.Number); !ok || number.String() != want[i] {
			t.Fatalf("%s number %d = %#v, want exact %s", name, i, got[i], want[i])
		}
	}
}
