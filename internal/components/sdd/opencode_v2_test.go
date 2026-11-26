package sdd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestOpenCodeV2GeneratedPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.json")
	if err := os.WriteFile(path, []byte(`{"agents":{"worker":{"system":"Native","permissions":[{"action":"edit","resource":"*","effect":"allow"}]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := mergeOpenCodeJSONFile(path, []byte(`{"agent":{"worker":{"permission":{"task":{"__replace__":{"*":"deny","explore":"allow"}},"bash":"ask"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	root, _ := filemerge.UnmarshalJSONObject(data)
	entry := root["agents"].(map[string]any)["worker"].(map[string]any)
	rules, ok := entry["permissions"].([]any)
	if !ok || len(rules) != 4 {
		t.Fatalf("permissions=%#v", entry["permissions"])
	}
	if rules[0].(map[string]any)["action"] != "edit" || rules[1].(map[string]any)["action"] != "shell" || rules[2].(map[string]any)["action"] != "subagent" || rules[2].(map[string]any)["resource"] != "*" {
		t.Fatalf("rule order/actions=%#v", rules)
	}
}

func TestOpenCodeV2ShippedPermissionPreservation(t *testing.T) {
	content, err := assets.Read(overlayAssetPath(model.SDDModeMulti))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := filemerge.UnmarshalJSONObject([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	agents := overlay["agent"].(map[string]any)
	for _, tt := range []struct {
		name, agent, action, resource string
		rules                         []any
		want                          string
	}{
		{"wildcard allow becomes generated default", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "allow"}}, "allow"},
		{"converted wildcard still denies unrelated agents", "axiom-orchestrator", "subagent", "untrusted-agent", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "allow"}}, "deny"},
		{"converted wildcard preserves specific user deny", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "allow"}, map[string]any{"action": "subagent", "resource": "explore", "effect": "deny"}}, "deny"},
		{"multiple user task denies", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "deny"}, map[string]any{"action": "subagent", "resource": "explore", "effect": "deny"}}, "deny"},
		{"specific deny with generated exception", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "deny"}, map[string]any{"action": "subagent", "resource": "sdd-*", "effect": "allow"}, map[string]any{"action": "subagent", "resource": "explore", "effect": "deny"}}, "deny"},
		{"existing generated exception survives", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "deny"}, map[string]any{"action": "subagent", "resource": "explore", "effect": "allow"}}, "allow"},
		{"specific ask survives replacement", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "deny"}, map[string]any{"action": "subagent", "resource": "explore", "effect": "ask"}}, "ask"},
		{"existing question deny", "axiom-orchestrator", "question", "any", []any{map[string]any{"action": "question", "resource": "*", "effect": "deny"}}, "deny"},
		{"managed allowed scope", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "untrusted-agent", "effect": "allow"}}, "allow"},
		{"universal user task deny", "axiom-orchestrator", "subagent", "explore", []any{map[string]any{"action": "subagent", "resource": "*", "effect": "deny"}}, "deny"},
		{"managed task replacement", "axiom-orchestrator", "subagent", "untrusted-agent", []any{map[string]any{"action": "subagent", "resource": "untrusted-agent", "effect": "allow"}}, "deny"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "opencode.json")
			base, _ := json.Marshal(map[string]any{"agents": map[string]any{tt.agent: map[string]any{"system": "Existing", "permissions": tt.rules}}})
			if err := os.WriteFile(path, base, 0600); err != nil {
				t.Fatal(err)
			}
			patch, _ := json.Marshal(map[string]any{"agent": map[string]any{tt.agent: agents[tt.agent]}})
			if _, err := mergeOpenCodeJSONFile(path, patch); err != nil {
				t.Fatal(err)
			}
			first, _ := os.ReadFile(path)
			if _, err := mergeOpenCodeJSONFile(path, patch); err != nil {
				t.Fatal(err)
			}
			second, _ := os.ReadFile(path)
			if string(first) != string(second) {
				t.Fatalf("native permission merge is not idempotent\nfirst=%s\nsecond=%s", first, second)
			}
			data, _ := os.ReadFile(path)
			root, _ := filemerge.UnmarshalJSONObject(data)
			rules := root["agents"].(map[string]any)[tt.agent].(map[string]any)["permissions"].([]any)
			effect := "ask"
			matches := func(pattern, value string) bool {
				return regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$").MatchString(value)
			}
			for _, raw := range rules {
				rule := raw.(map[string]any)
				if matches(rule["action"].(string), tt.action) && matches(rule["resource"].(string), tt.resource) {
					effect = rule["effect"].(string)
				}
			}
			if effect != tt.want {
				t.Fatalf("effective %s permission = %s, want %s; rules=%#v", tt.action, effect, tt.want, rules)
			}
		})
	}
}
