package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// RF-BRA-40: la desinstalación retira las órdenes de `axiom autofetch run` y de skill-registry (en
// cualquier forma y evento, también las que solo aparecen en `commandWindows`, el campo de Codex
// para `cmd.exe`) y conserva las ajenas y la guarda de rama.
func TestComponentOperationsSDD_AutofetchRemovesHookAndKeepsForeign(t *testing.T) {
	cases := []struct {
		name  string
		agent model.AgentID
		path  func(homeDir string, adapter agents.Adapter) string
	}{
		{"Claude Code", model.AgentClaudeCode, func(h string, a agents.Adapter) string { return a.SettingsPath(h) }},
		{"Codex", model.AgentCodex, func(h string, a agents.Adapter) string { return filepath.Join(a.GlobalConfigDir(h), "hooks.json") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			homeDir := t.TempDir()
			svc, err := NewService(homeDir, t.TempDir(), "dev")
			if err != nil {
				t.Fatal(err)
			}
			adapter, ok := svc.registry.Get(tc.agent)
			if !ok {
				t.Fatalf("adapter %s not found", tc.agent)
			}
			path := tc.path(homeDir, adapter)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			initial := `{"hooks":{
  "SessionStart":[
    {"matcher":"startup|resume","hooks":[{"type":"command","command":"axiom autofetch run --agent codex || true","timeout":10},{"type":"command","command":"echo keep start"}]},
    {"matcher":"startup|resume","hooks":[{"type":"command","command":"powershell -NoProfile -Command 'axiom autofetch run --agent claude-code; exit 0'"}]},
    {"matcher":"startup|resume","hooks":[{"type":"command","command":"axiom autofetch run --agent codex || true","commandWindows":"axiom autofetch run --agent codex 2>nul & exit /b 0","timeout":10}]},
    {"matcher":"startup|resume|clear|compact","hooks":[{"type":"command","command":"axiom skill-registry refresh --quiet --no-gitignore --cwd \"$PWD\" || true","commandWindows":"axiom skill-registry refresh --quiet --no-gitignore 2>nul & exit /b 0","timeout":30}]},
    {"matcher":"startup","hooks":[{"type":"command","command":"echo posix only","commandWindows":"axiom autofetch run --agent codex 2>nul & exit /b 0"}]}
  ],
  "UserPromptSubmit":[{"hooks":[{"type":"command","command":"axiom autofetch run --agent codex || true"}]}],
  "PreToolUse":[{"matcher":"x","hooks":[{"type":"command","command":"echo keep pre"}]}]
}}`
			if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
				t.Fatal(err)
			}
			ops, _, err := svc.componentOperations(adapter, model.ComponentSDD)
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range ops {
				if op.typeID == opRewriteFile && op.path == path {
					if _, _, err := op.apply(op.path); err != nil {
						t.Fatal(err)
					}
				}
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(raw)
			for _, gone := range []string{"axiom autofetch run", "axiom skill-registry refresh", "commandWindows", "UserPromptSubmit"} {
				if strings.Contains(text, gone) {
					t.Fatalf("%q: los hooks de Axiom y los eventos vaciados deben retirarse:\n%s", gone, text)
				}
			}
			if !strings.Contains(text, "echo keep start") || !strings.Contains(text, "echo keep pre") {
				t.Fatalf("lo ajeno debe conservarse:\n%s", text)
			}
		})
	}
}
