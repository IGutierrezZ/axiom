package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEffectiveConfigStopsProjectSearchAtGitRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	parent := t.TempDir()
	project := filepath.Join(parent, "repo")
	nested := filepath.Join(project, "packages", "app")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o700); err != nil {
		t.Fatalf("mkdir git root: %v", err)
	}
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatalf("mkdir nested project: %v", err)
	}
	parentConfig := filepath.Join(parent, "opencode.jsonc")
	if err := os.WriteFile(parentConfig, []byte(`{"provider":{"outside":{"models":{"m":{}}}}}`), 0o600); err != nil {
		t.Fatalf("write parent config: %v", err)
	}
	globalConfig := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(globalConfig), 0o700); err != nil {
		t.Fatalf("mkdir global config: %v", err)
	}
	if err := os.WriteFile(globalConfig, []byte(`{"provider":{"global":{"models":{"m":{}}}}}`), 0o600); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	snapshot, err := ResolveEffectiveConfigForHome(home, nested)
	if err != nil {
		t.Fatalf("ResolveEffectiveConfigForHome() error = %v", err)
	}
	if snapshot.Path != globalConfig {
		t.Fatalf("effective path = %q, want global config %q without crossing git root to %q", snapshot.Path, globalConfig, parentConfig)
	}
}

func TestResolveEffectiveConfigMalformedModelIsPresentButNotCleared(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, "opencode.jsonc")
	if err := os.WriteFile(configPath, []byte(`{
		"agent": {
			"sdd-apply": {"model": "typo-without-provider"},
			"sdd-spec": {"model": ""}
		}
	}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	snapshot, err := ResolveEffectiveConfigForHome(home, projectDir)
	if err != nil {
		t.Fatalf("ResolveEffectiveConfigForHome() error = %v", err)
	}
	apply := snapshot.Assignments["sdd-apply"]
	if !apply.Present || apply.Cleared {
		t.Fatalf("malformed model assignment = %+v, want present but not cleared", apply)
	}
	spec := snapshot.Assignments["sdd-spec"]
	if !spec.Present || !spec.Cleared {
		t.Fatalf("empty model assignment = %+v, want explicit clear", spec)
	}
}

func TestEffectiveSettingsPathPreservesSelectedWritePathOnReadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, "opencode.jsonc")
	if err := os.WriteFile(configPath, []byte(`{"provider":`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if got := EffectiveSettingsPath(home, projectDir); got != configPath {
		t.Fatalf("EffectiveSettingsPath() = %q, want selected malformed config path %q", got, configPath)
	}
}

func TestResolveEffectiveConfigSelectsOpenCodeConfigFile(t *testing.T) {
	for _, tt := range []struct {
		name         string
		writeJSON    bool
		jsonManaged  bool
		writeJSONC   bool
		jsoncManaged bool
		wantName     string
	}{
		{name: "only json", writeJSON: true, wantName: "opencode.json"},
		{name: "only jsonc", writeJSONC: true, wantName: "opencode.jsonc"},
		{name: "both managed only in json", writeJSON: true, jsonManaged: true, writeJSONC: true, wantName: "opencode.json"},
		{name: "both managed only in jsonc", writeJSON: true, writeJSONC: true, jsoncManaged: true, wantName: "opencode.jsonc"},
		{name: "both managed in neither", writeJSON: true, writeJSONC: true, wantName: "opencode.json"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			projectDir := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv("OPENCODE_CONFIG_DIR", "")

			if tt.writeJSON {
				writeOpenCodeConfigFixture(t, filepath.Join(projectDir, "opencode.json"), tt.jsonManaged)
			}
			if tt.writeJSONC {
				writeOpenCodeConfigFixture(t, filepath.Join(projectDir, "opencode.jsonc"), tt.jsoncManaged)
			}

			snapshot, err := ResolveEffectiveConfigForHome(home, projectDir)
			if err != nil {
				t.Fatalf("ResolveEffectiveConfigForHome() error = %v", err)
			}
			wantPath := filepath.Join(projectDir, tt.wantName)
			if snapshot.Path != wantPath || snapshot.WritePath != wantPath {
				t.Fatalf("paths = (%q, %q), want %q", snapshot.Path, snapshot.WritePath, wantPath)
			}
		})
	}

	// Regression for CodeRabbit r3952255572: when JSON has a user-owned agent
	// with the managed shape (hidden + prompt + permission) but no Axiom
	// ownership marker, and JSONC has the real Axiom-managed config with
	// the marker, the resolver must select JSONC.
	t.Run("json user-owned managed shape without marker selects jsonc with marker", func(t *testing.T) {
		home := t.TempDir()
		projectDir := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("OPENCODE_CONFIG_DIR", "")

		jsonPath := filepath.Join(projectDir, "opencode.json")
		// User-owned agent that matches hidden+prompt+permission shape
		// but lacks the __managed_by marker.
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir project dir: %v", err)
		}
		if err := os.WriteFile(jsonPath, []byte(`{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "hidden": true,
      "prompt": "my custom orchestrator",
      "permission": {"task": "allow"}
    }
  }
}`), 0o600); err != nil {
			t.Fatalf("write json fixture: %v", err)
		}

		jsoncPath := filepath.Join(projectDir, "opencode.jsonc")
		// Real Axiom managed config with the ownership marker.
		if err := os.WriteFile(jsoncPath, []byte(`{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "hidden": true,
      "prompt": "managed by Gentle AI",
      "permission": {},
      "__managed_by": "gentle-ai/sdd"
    }
  }
}`), 0o600); err != nil {
			t.Fatalf("write jsonc fixture: %v", err)
		}

		snapshot, err := ResolveEffectiveConfigForHome(home, projectDir)
		if err != nil {
			t.Fatalf("ResolveEffectiveConfigForHome() error = %v", err)
		}
		wantPath := filepath.Join(projectDir, "opencode.jsonc")
		if snapshot.Path != wantPath || snapshot.WritePath != wantPath {
			t.Fatalf("paths = (%q, %q), want %q", snapshot.Path, snapshot.WritePath, wantPath)
		}
	})
}

func TestResolveEffectiveConfigUsesOpenCodeConfigDir(t *testing.T) {
	home := t.TempDir()
	projectDir := t.TempDir()
	defaultDir := filepath.Join(home, ".config", "opencode")
	overrideDir := filepath.Join(t.TempDir(), "opencode")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", overrideDir)
	writeOpenCodeConfigFixture(t, filepath.Join(defaultDir, "opencode.json"), true)
	writeOpenCodeConfigFixture(t, filepath.Join(overrideDir, "opencode.jsonc"), false)

	snapshot, err := ResolveEffectiveConfigForHome(home, projectDir)
	if err != nil {
		t.Fatalf("ResolveEffectiveConfigForHome() error = %v", err)
	}
	wantPath := filepath.Join(overrideDir, "opencode.jsonc")
	if snapshot.Path != wantPath || snapshot.WritePath != wantPath {
		t.Fatalf("paths = (%q, %q), want OPENCODE_CONFIG_DIR config %q", snapshot.Path, snapshot.WritePath, wantPath)
	}
}

func writeOpenCodeConfigFixture(t *testing.T, path string, managed bool) {
	t.Helper()
	content := `{"provider":{"fixture":{"models":{"m":{}}}}}`
	if managed {
		content = `{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "hidden": true,
      "prompt": "managed by Gentle AI",
      "permission": {}
    }
  }
}`
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
}

// TestManagedConfigPriorityRecognizesBothManagedByMarkers pins that the writer's
// current ownership marker (axiom/sdd) and the one older installs carry
// (gentle-ai/sdd) both rank as explicit ownership proof, so a config written
// before the marker rename is still recognized and migrated by sync.
func TestManagedConfigPriorityRecognizesBothManagedByMarkers(t *testing.T) {
	tests := []struct {
		name   string
		marker string
		want   int
	}{
		{name: "current marker", marker: "axiom/sdd", want: 2},
		{name: "legacy marker from older installs", marker: "gentle-ai/sdd", want: 2},
		{name: "foreign marker is not proof of ownership", marker: "someone-else/sdd", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "opencode.json")
			content := `{"agent":{"custom":{"mode":"primary","__managed_by":"` + tt.marker + `"}}}`
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			if got := managedConfigPriority(path); got != tt.want {
				t.Fatalf("managedConfigPriority() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Sin marcador, la detección por nombre y forma sigue puntuando: el overlay actual no
// escribe `__managed_by`, y el orquestador (que no es hidden) también cuenta.
func TestManagedConfigPriorityDetectsManagedAgentsWithoutMarker(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{name: "orquestador gestionado", content: `{"agent":{"axiom-orchestrator":{"mode":"primary","prompt":"p","permission":{}}}}`, want: 1},
		{name: "perfil gestionado", content: `{"agent":{"sdd-orchestrator-fast":{"mode":"primary","prompt":"p","permission":{}}}}`, want: 1},
		{name: "fase gestionada", content: `{"agent":{"sdd-apply":{"mode":"subagent","hidden":true,"prompt":"p","permission":{}}}}`, want: 1},
		{name: "nombre de usuario con la misma forma", content: `{"agent":{"mio":{"mode":"primary","hidden":true,"prompt":"p","permission":{}}}}`, want: 0},
		{name: "general del usuario sin hidden", content: `{"agent":{"general":{"prompt":"p","permission":{}}}}`, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "opencode.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			if got := managedConfigPriority(path); got != tt.want {
				t.Fatalf("managedConfigPriority() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsManagedAgent(t *testing.T) {
	shaped := map[string]any{"prompt": "p", "permission": map[string]any{}}
	tests := []struct {
		name string
		def  map[string]any
		want bool
	}{
		{"axiom-orchestrator", shaped, true},
		{"sdd-orchestrator-fast", shaped, true},
		{"jd-judge-a-fast", shaped, true},
		{"review-validator", shaped, false},
		{"sdd-apply", map[string]any{"mode": "subagent"}, false},
		{"sdd-orchestrator-", shaped, false},
		{"mio", shaped, false},
		{"general", shaped, false},
		{"general", map[string]any{"hidden": true, "prompt": "p", "permission": map[string]any{}}, true},
		{"mio", map[string]any{"__managed_by": "axiom/sdd"}, true},
		{"mio", map[string]any{"__managed_by": "gentle-ai/sdd"}, true},
		{"sdd-apply", map[string]any{"__managed_by": "otra"}, false},
	}
	for _, tt := range tests {
		if got := IsManagedAgent(tt.name, tt.def); got != tt.want {
			t.Errorf("IsManagedAgent(%q, %v) = %v, want %v", tt.name, tt.def, got, tt.want)
		}
	}
}
