package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLocalSettings(t *testing.T, dir, content string) string {
	t.Helper()
	path := LocalSettingsPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readLocalSettings(t *testing.T, dir string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(LocalSettingsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func deniedNames(t *testing.T, root map[string]any) []string {
	t.Helper()
	var names []string
	list, _ := root["deniedMcpServers"].([]any)
	for _, entry := range list {
		object := entry.(map[string]any)
		if name, ok := object["serverName"].(string); ok {
			names = append(names, name)
		} else {
			names = append(names, "<ajena>")
		}
	}
	return names
}

func TestApplyDeniedMCPServersCreatesAbsentFile(t *testing.T) {
	dir := t.TempDir()
	changed, err := ApplyDeniedMCPServers(dir, []string{"work-copilot", "claude.ai Microsoft 365"}, nil)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got := deniedNames(t, readLocalSettings(t, dir))
	if strings.Join(got, "|") != "work-copilot|claude.ai Microsoft 365" {
		t.Fatalf("deniedMcpServers = %v", got)
	}
}

func TestApplyDeniedMCPServersNothingToAddDoesNotCreateFile(t *testing.T) {
	dir := t.TempDir()
	changed, err := ApplyDeniedMCPServers(dir, nil, []string{"a"})
	if err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(LocalSettingsPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("no debía crearse el fichero: %v", err)
	}
}

func TestApplyDeniedMCPServersPreservesForeignKeysAndEntries(t *testing.T) {
	dir := t.TempDir()
	writeLocalSettings(t, dir, `{
  "enabledMcpjsonServers": ["engram"],
  "deniedMcpServers": [
    {"serverName": "ajeno"},
    {"serverName": "con-url", "serverUrl": "https://x"},
    {"serverCommand": ["node", "s.js"]}
  ]
}`)
	changed, err := ApplyDeniedMCPServers(dir, []string{"nuevo"}, nil)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	root := readLocalSettings(t, dir)
	if got := deniedNames(t, root); strings.Join(got, "|") != "ajeno|con-url|<ajena>|nuevo" {
		t.Fatalf("deniedMcpServers = %v", got)
	}
	enabled, _ := root["enabledMcpjsonServers"].([]any)
	if len(enabled) != 1 || enabled[0] != "engram" {
		t.Fatalf("enabledMcpjsonServers = %v", root["enabledMcpjsonServers"])
	}
}

func TestApplyDeniedMCPServersRemovesOnlyBareRequestedEntries(t *testing.T) {
	dir := t.TempDir()
	writeLocalSettings(t, dir, `{"deniedMcpServers": [
    {"serverName": "viejo"}, {"serverName": "ajeno"},
    {"serverName": "viejo2", "serverUrl": "https://x"}, {"serverName": "queda"}]}`)
	if _, err := ApplyDeniedMCPServers(dir, []string{"queda"}, []string{"viejo", "viejo2", "queda"}); err != nil {
		t.Fatal(err)
	}
	// «queda» está también en add: no se retira. «viejo2» lleva serverUrl: es ajena.
	if got := deniedNames(t, readLocalSettings(t, dir)); strings.Join(got, "|") != "ajeno|viejo2|queda" {
		t.Fatalf("deniedMcpServers = %v", got)
	}
}

func TestApplyDeniedMCPServersIsIdempotentAndAdoptsManualEntries(t *testing.T) {
	dir := t.TempDir()
	path := writeLocalSettings(t, dir, `{"deniedMcpServers":[{"serverName":"a"}]}`)
	changed, err := ApplyDeniedMCPServers(dir, []string{"a"}, nil)
	if err != nil || changed {
		t.Fatalf("adoptar una entrada manual no debe escribir: changed=%v err=%v", changed, err)
	}
	if _, err := ApplyDeniedMCPServers(dir, []string{"b"}, nil); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	changed, err = ApplyDeniedMCPServers(dir, []string{"b"}, nil)
	if err != nil || changed {
		t.Fatalf("segunda ejecución: changed=%v err=%v", changed, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("el fichero cambió en la segunda ejecución")
	}
}

func TestApplyDeniedMCPServersDropsEmptyKey(t *testing.T) {
	dir := t.TempDir()
	writeLocalSettings(t, dir, `{"enabledMcpjsonServers":["engram"],"deniedMcpServers":[{"serverName":"a"}]}`)
	if _, err := ApplyDeniedMCPServers(dir, nil, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	root := readLocalSettings(t, dir)
	if _, present := root["deniedMcpServers"]; present {
		t.Fatalf("la clave debía quitarse: %v", root)
	}
	if _, present := root["enabledMcpjsonServers"]; !present {
		t.Fatalf("enabledMcpjsonServers debía conservarse: %v", root)
	}
}

func TestApplyDeniedMCPServersMalformedJSONLeavesFileUntouched(t *testing.T) {
	dir := t.TempDir()
	const broken = `{"deniedMcpServers": [`
	path := writeLocalSettings(t, dir, broken)
	changed, err := ApplyDeniedMCPServers(dir, []string{"a"}, nil)
	if err == nil || changed {
		t.Fatalf("debía fallar: changed=%v err=%v", changed, err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("el error debe nombrar el fichero: %v", err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != broken {
		t.Fatalf("el fichero cambió: %s", raw)
	}
}
