package uninstall

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/permissions"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// applyPermissionOperations ejecuta las operaciones del componente de permisos
// de un agente sobre homeDir.
func applyPermissionOperations(t *testing.T, homeDir string, agentID model.AgentID) {
	t.Helper()
	svc, err := NewService(homeDir, t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	adapter, ok := svc.registry.Get(agentID)
	if !ok {
		t.Fatalf("adapter %q not found", agentID)
	}
	ops, _, err := svc.componentOperations(adapter, model.ComponentPermission)
	if err != nil {
		t.Fatalf("componentOperations() error = %v", err)
	}
	for _, op := range ops {
		if _, _, err := op.apply(op.path); err != nil {
			t.Fatalf("operation %v on %q error = %v", op.typeID, op.path, err)
		}
	}
}

func stringsAt(t *testing.T, root map[string]any, path ...string) []string {
	t.Helper()
	var current any = root
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("%v: %v is not an object in %#v", path, key, root)
		}
		current = object[key]
	}
	list, _ := current.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.(string))
	}
	return out
}

func axiomClaudeDenyRules(t *testing.T) []string {
	t.Helper()
	overlay := readOverlayForTest(t, model.AgentClaudeCode)
	return stringsAt(t, overlay, "permissions", "deny")
}

func readOverlayForTest(t *testing.T, agentID model.AgentID) map[string]any {
	t.Helper()
	path := filepath.Join(t.TempDir(), "overlay.json")
	if err := os.WriteFile(path, permissions.Overlay(agentID), 0o644); err != nil {
		t.Fatalf("WriteFile(overlay) error = %v", err)
	}
	return readJSONFileForTest(t, path)
}

func TestPermissionUninstallClaudeKeepsUserDenyAndMode(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	axiomDeny := axiomClaudeDenyRules(t)
	if len(axiomDeny) == 0 {
		t.Fatal("el overlay de Claude no declara reglas deny")
	}

	deny := `"Bash(curl *)", "Read(./secreto.txt)"`
	for _, rule := range axiomDeny {
		deny += `, "` + rule + `"`
	}
	seed := `{"permissions":{"defaultMode":"acceptEdits","allow":["Bash(git status)"],"deny":[` + deny + `]},"theme":"dark"}`
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	applyPermissionOperations(t, homeDir, model.AgentClaudeCode)

	got := readJSONFileForTest(t, settingsPath)
	if gotDeny := stringsAt(t, got, "permissions", "deny"); !slices.Equal(gotDeny, []string{"Bash(curl *)", "Read(./secreto.txt)"}) {
		t.Fatalf("permissions.deny = %v, want solo las entradas del usuario y en su orden", gotDeny)
	}
	permissionsObject := got["permissions"].(map[string]any)
	if permissionsObject["defaultMode"] != "acceptEdits" {
		t.Fatalf("defaultMode = %v, want el del usuario intacto", permissionsObject["defaultMode"])
	}
	if allow := stringsAt(t, got, "permissions", "allow"); !slices.Equal(allow, []string{"Bash(git status)"}) {
		t.Fatalf("permissions.allow = %v, want intacto", allow)
	}
	if got["theme"] != "dark" {
		t.Fatalf("theme = %v, want intacto", got["theme"])
	}
}

func TestPermissionUninstallClaudeRemovesPermissionsThatAxiomEmptied(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	deny := ""
	for i, rule := range axiomClaudeDenyRules(t) {
		if i > 0 {
			deny += ", "
		}
		deny += `"` + rule + `"`
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"permissions":{"deny":[`+deny+`]},"theme":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	applyPermissionOperations(t, homeDir, model.AgentClaudeCode)

	got := readJSONFileForTest(t, settingsPath)
	if _, exists := got["permissions"]; exists {
		t.Fatalf("permissions debería desaparecer cuando solo tenía lo de Axiom: %#v", got)
	}
	if got["theme"] != "dark" {
		t.Fatalf("theme = %v, want intacto", got["theme"])
	}
}

func TestPermissionUninstallClaudeWithOnlyAxiomRulesDeletesTheFile(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	deny := ""
	for i, rule := range axiomClaudeDenyRules(t) {
		if i > 0 {
			deny += ", "
		}
		deny += `"` + rule + `"`
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"permissions":{"deny":[`+deny+`]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	applyPermissionOperations(t, homeDir, model.AgentClaudeCode)

	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Fatalf("un settings.json que solo tenía lo de Axiom debería borrarse; stat err = %v", err)
	}
}

func TestPermissionUninstallOpenCodeKeepsUserRules(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	seed := `{"permission":{"bash":{"*":"allow","git push *":"ask","npm publish":"deny"},"read":{"**/.env":"deny","**/mis-notas/**":"deny"},"edit":"ask"}}`
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	applyPermissionOperations(t, homeDir, model.AgentOpenCode)

	got := readJSONFileForTest(t, settingsPath)
	permission := got["permission"].(map[string]any)
	bash := permission["bash"].(map[string]any)
	if bash["*"] != "allow" || bash["npm publish"] != "deny" {
		t.Fatalf("permission.bash = %#v, want las reglas del usuario intactas", bash)
	}
	if _, exists := bash["git push *"]; exists {
		t.Fatalf("permission.bash conserva la regla de Axiom: %#v", bash)
	}
	read := permission["read"].(map[string]any)
	if read["**/mis-notas/**"] != "deny" {
		t.Fatalf("permission.read = %#v, want la regla del usuario", read)
	}
	if _, exists := read["**/.env"]; exists {
		t.Fatalf("permission.read conserva la regla de Axiom: %#v", read)
	}
	if permission["edit"] != "ask" {
		t.Fatalf("permission.edit = %v, want intacto", permission["edit"])
	}
}

func TestPermissionUninstallDoesNotTouchApprovalModesAxiomNoLongerWrites(t *testing.T) {
	cases := []struct {
		agent model.AgentID
		path  []string
		seed  string
	}{
		{model.AgentGeminiCLI, []string{".gemini", "settings.json"}, `{"general":{"defaultApprovalMode":"auto_edit"}}`},
		{model.AgentVSCodeCopilot, []string{"AppData", "Roaming", "Code", "User", "settings.json"}, `{"chat.tools.autoApprove":true}`},
	}
	for _, tc := range cases {
		t.Run(string(tc.agent), func(t *testing.T) {
			homeDir := t.TempDir()
			svc, err := NewService(homeDir, t.TempDir(), "dev")
			if err != nil {
				t.Fatal(err)
			}
			adapter, _ := svc.registry.Get(tc.agent)
			path := adapter.SettingsPath(homeDir)
			if path == "" {
				t.Skip("sin ruta de ajustes")
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.seed), 0o644); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(path)

			applyPermissionOperations(t, homeDir, tc.agent)

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("el fichero del usuario debe seguir existiendo: %v", err)
			}
			if string(after) != string(before) {
				t.Fatalf("el fichero cambió:\nantes %s\ndespués %s", before, after)
			}
		})
	}
}
