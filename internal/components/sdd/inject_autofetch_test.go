package sdd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func runtimeGOOSForTest() string { return runtime.GOOS }

func TestAutofetchHookCommandEndsWithZeroPerRuntimeAndOS(t *testing.T) {
	cases := []struct {
		agent model.AgentID
		goos  string
		want  string
	}{
		{model.AgentClaudeCode, "linux", "axiom autofetch run --agent claude-code || true"},
		{model.AgentClaudeCode, "darwin", "axiom autofetch run --agent claude-code || true"},
		{model.AgentClaudeCode, "windows", "powershell -NoProfile -Command 'axiom autofetch run --agent claude-code; exit 0'"},
		{model.AgentCodex, "linux", "axiom autofetch run --agent codex || true"},
		{model.AgentCodex, "windows", "axiom autofetch run --agent codex || true"},
	}
	for _, tc := range cases {
		if got := AutofetchHookCommand(tc.agent, tc.goos); got != tc.want {
			t.Errorf("AutofetchHookCommand(%s, %s) = %q, se esperaba %q", tc.agent, tc.goos, got, tc.want)
		}
	}
}

func readAutofetchHooks(t *testing.T, path string) (map[string]any, []byte) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	hooks, _ := root["hooks"].(map[string]any)
	return hooks, raw
}

// assertAutofetchEntry exige exactamente una orden canónica, bajo SessionStart, con el matcher y el timeout fijados.
func assertAutofetchEntry(t *testing.T, path, command string) {
	t.Helper()
	hooks, raw := readAutofetchHooks(t, path)
	if n := strings.Count(string(raw), autofetchHookMarker); n != 1 {
		t.Fatalf("%s: %d órdenes de autofetch, se esperaba 1\n%s", path, n, raw)
	}
	entries, _ := hooks["SessionStart"].([]any)
	for _, item := range entries {
		entry := item.(map[string]any)
		for _, h := range entry["hooks"].([]any) {
			hook := h.(map[string]any)
			if hook["command"] != command {
				continue
			}
			if entry["matcher"] != "startup|resume" || hook["type"] != "command" || hook["timeout"] != float64(10) {
				t.Fatalf("entrada con matcher %v type %v timeout %v\n%s", entry["matcher"], hook["type"], hook["timeout"], raw)
			}
			return
		}
	}
	t.Fatalf("%s: la orden %q no está bajo SessionStart\n%s", path, command, raw)
}

func TestEnsureAutofetchHookInstallsOnceAndKeepsForeignAndGuard(t *testing.T) {
	cases := []struct {
		name    string
		agent   model.AgentID
		path    string
		ensure  func(string, model.AgentID) (bool, error)
		guardFn func(string, model.AgentID) (bool, error)
	}{
		{"claude", model.AgentClaudeCode, filepath.Join(t.TempDir(), ".claude", "settings.json"), ensureClaudeAutofetchHook, ensureClaudeBranchGuardHook},
		{"codex", model.AgentCodex, filepath.Join(t.TempDir(), ".codex", "hooks.json"), ensureCodexAutofetchHook, ensureCodexBranchGuardHook},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.MkdirAll(filepath.Dir(tc.path), 0o755); err != nil {
				t.Fatal(err)
			}
			seed := `{"model":"opus","hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"echo foreign start"}]}]}}`
			if err := os.WriteFile(tc.path, []byte(seed), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := tc.guardFn(tc.path, tc.agent); err != nil {
				t.Fatal(err)
			}
			command := AutofetchHookCommand(tc.agent, runtimeGOOSForTest())
			if changed, err := tc.ensure(tc.path, tc.agent); err != nil || !changed {
				t.Fatalf("la primera instalación debe cambiar: changed=%v err=%v", changed, err)
			}
			assertAutofetchEntry(t, tc.path, command)
			first, _ := os.ReadFile(tc.path)
			if changed, err := tc.ensure(tc.path, tc.agent); err != nil || changed {
				t.Fatalf("la segunda instalación debe ser un no-op: changed=%v err=%v", changed, err)
			}
			second, _ := os.ReadFile(tc.path)
			if !bytes.Equal(first, second) {
				t.Fatalf("la segunda instalación cambió los bytes:\n%s\n%s", first, second)
			}
			for _, keep := range []string{"echo foreign start", "axiom branch guard --agent " + string(tc.agent), `"model": "opus"`} {
				if !strings.Contains(string(first), keep) {
					t.Fatalf("%q no se conservó:\n%s", keep, first)
				}
			}
		})
	}
}

func TestPruneStaleAutofetchHooksLeavesOnlyCanonicalUnderSessionStart(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCodex} {
		t.Run(string(agent), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cfg", "hooks.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			command := AutofetchHookCommand(agent, runtimeGOOSForTest())
			old := "axiom autofetch run --agent " + string(agent) + " 2>&1"
			seed := `{"hooks":{
  "UserPromptSubmit":[{"matcher":"","hooks":[{"type":"command","command":"axiom autofetch run --agent ` + string(agent) + ` || true"},{"type":"command","command":"echo keep prompt"}]}],
  "SessionStart":[
    {"matcher":"startup","hooks":[{"type":"command","command":"` + old + `"}]},
    {"matcher":"startup|resume","hooks":[{"type":"command","command":"echo foreign start"}]}
  ],
  "Stop":[{"hooks":[{"type":"command","command":"` + old + `"}]}],
  "PreToolUse":[{"matcher":"x","hooks":[{"type":"command","command":"axiom branch guard --agent ` + string(agent) + `"}]}]
}}`
			if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
				t.Fatal(err)
			}
			if changed, err := ensureAutofetchHookEntries(path, "test", autofetchHookMatcher, command); err != nil || !changed {
				t.Fatalf("changed=%v err=%v", changed, err)
			}
			assertAutofetchEntry(t, path, command)
			hooks, raw := readAutofetchHooks(t, path)
			if _, ok := hooks["Stop"]; ok {
				t.Fatalf("el evento Stop vaciado debe borrarse:\n%s", raw)
			}
			for _, keep := range []string{"echo keep prompt", "echo foreign start", "axiom branch guard --agent " + string(agent)} {
				if !strings.Contains(string(raw), keep) {
					t.Fatalf("%q no se conservó:\n%s", keep, raw)
				}
			}
			first := raw
			if changed, err := ensureAutofetchHookEntries(path, "test", autofetchHookMatcher, command); err != nil || changed {
				t.Fatalf("segunda pasada: changed=%v err=%v", changed, err)
			}
			if second, _ := os.ReadFile(path); !bytes.Equal(first, second) {
				t.Fatalf("el segundo sync cambió los bytes:\n%s\n%s", first, second)
			}
		})
	}
}

func TestPruneStaleAutofetchHooksRemovesDuplicatesAndOtherOSForm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	command := AutofetchHookCommand(model.AgentClaudeCode, "linux")
	other := AutofetchHookCommand(model.AgentClaudeCode, "windows")
	seed, _ := json.Marshal(map[string]any{"hooks": map[string]any{"SessionStart": []any{
		map[string]any{"matcher": autofetchHookMatcher, "hooks": []any{map[string]any{"type": "command", "command": command, "timeout": 10}}},
		map[string]any{"matcher": autofetchHookMatcher, "hooks": []any{map[string]any{"type": "command", "command": command, "timeout": 10}}},
		map[string]any{"matcher": autofetchHookMatcher, "hooks": []any{map[string]any{"type": "command", "command": other}}},
	}}})
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, err := ensureAutofetchHookEntries(path, "test", autofetchHookMatcher, command); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	assertAutofetchEntry(t, path, command)
}

func TestPruneStaleAutofetchHooksWithEmptyCanonicalRemovesAll(t *testing.T) {
	hooks := map[string]any{
		"SessionStart": []any{map[string]any{"matcher": autofetchHookMatcher, "hooks": []any{map[string]any{"command": "axiom autofetch run --agent codex || true"}}}},
		"Stop":         []any{map[string]any{"hooks": []any{map[string]any{"command": "echo keep"}}}},
	}
	if !pruneStaleAutofetchHooks(hooks, autofetchHookMatcher, "") {
		t.Fatal("debe informar del cambio")
	}
	if _, ok := hooks["SessionStart"]; ok || hooks["Stop"] == nil {
		t.Fatalf("solo debe quedar lo ajeno: %v", hooks)
	}
}

func TestEnsureAutofetchHookRejectsUnexpectedShapeWithoutTouchingFile(t *testing.T) {
	for _, seed := range []string{`{"hooks":"nope"}`, `{"hooks":{"SessionStart":{"bad":true}}}`} {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
			t.Fatal(err)
		}
		changed, err := ensureAutofetchHookEntries(path, "test", autofetchHookMatcher, "axiom autofetch run --agent codex || true")
		if err == nil || changed {
			t.Fatalf("%s: changed=%v err=%v", seed, changed, err)
		}
		if after, _ := os.ReadFile(path); string(after) != seed {
			t.Fatalf("el fichero no debe tocarse: %s", after)
		}
	}
}

// Solo Claude Code y Codex reciben el hook; Pi y los demás no escriben nada.
func TestInstallSkillRegistryAutomationAutofetchReachesClaudeAndCodexOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	claude := claudeAdapter()
	codex := codexInjectAdapter()
	if _, err := installSkillRegistryAutomation(home, claude); err != nil {
		t.Fatal(err)
	}
	if _, err := installSkillRegistryAutomation(home, codex); err != nil {
		t.Fatal(err)
	}
	assertAutofetchEntry(t, claude.SettingsPath(home), AutofetchHookCommand(model.AgentClaudeCode, runtimeGOOSForTest()))
	assertAutofetchEntry(t, filepath.Join(codex.GlobalConfigDir(home), "hooks.json"), AutofetchHookCommand(model.AgentCodex, runtimeGOOSForTest()))
}
