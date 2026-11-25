package mcpexclusion

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var usageNow = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestEncodeProjectDir(t *testing.T) {
	cases := map[string]string{
		`C:\repos\axiom`:                   "C--repos-axiom",
		`C:\Users\Iñigo Gutiérrez\proj`:    "C--Users-I-igo-Guti-rrez-proj",
		`C:\repos\axiom-wt\feat-x\axiom`:   "C--repos-axiom-wt-feat-x-axiom",
		`C:\repos\work-copilot\.claude\wt`: "C--repos-work-copilot--claude-wt",
		"/home/ana/mi proyecto":            "-home-ana-mi-proyecto",
		"/tmp/emoji-\U0001F600":            "-tmp-emoji---",
	}
	for dir, want := range cases {
		if got := EncodeProjectDir(dir); got != want {
			t.Errorf("EncodeProjectDir(%q) = %q, quiero %q", dir, got, want)
		}
	}
}

func TestServerSegment(t *testing.T) {
	cases := map[string]string{
		"work-copilot":            "work-copilot",
		"engram":                  "engram",
		"claude.ai Microsoft 365": "claude_ai_Microsoft_365",
		"mi servidor/uno":         "mi_servidor_uno",
		"  con_borde  ":           "con_borde",
	}
	for name, want := range cases {
		if got := ServerSegment(name); got != want {
			t.Errorf("ServerSegment(%q) = %q, quiero %q", name, got, want)
		}
	}
}

func toolUseLine(ts string, tools ...string) string {
	items := make([]string, 0, len(tools))
	for _, tool := range tools {
		items = append(items, `{"type":"tool_use","id":"x","name":"`+tool+`","input":{"q":"secreto"}}`)
	}
	return `{"type":"assistant","timestamp":"` + ts + `","message":{"content":[` + strings.Join(items, ",") + `]}}`
}

func writeTranscript(t *testing.T, home, projectDir, name string, age time.Duration, lines ...string) string {
	t.Helper()
	folder := filepath.Join(home, ".claude", "projects", EncodeProjectDir(projectDir))
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mod := usageNow.Add(-age)
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanUsageCountsMCPToolsAcrossRootAndWorktree(t *testing.T) {
	home := t.TempDir()
	repo, _ := repoWithWorktree(t)
	recent := usageNow.Add(-48 * time.Hour).Format(time.RFC3339)
	older := usageNow.Add(-72 * time.Hour).Format(time.RFC3339)
	dirs := ProjectDirs(repo)
	if len(dirs) != 2 {
		t.Fatalf("dirs = %v, se esperaban raíz y worktree", dirs)
	}
	long := `{"type":"user","message":{"content":"` + strings.Repeat("x", 5<<20) + `"}}`

	writeTranscript(t, home, repo, "a.jsonl", time.Hour,
		toolUseLine(older, "mcp__work-copilot__search_all", "Bash", "Read"),
		`no es json {{{ "tool_use" mcp__`,
		long,
		`{"type":"user","message":{"content":"texto con mcp__engram__x y tool_use"}}`,
		`{"type":"assistant","timestamp":"`+recent+`","message":{"content":"cadena mcp__x__y tool_use"}}`,
		toolUseLine(recent, "mcp__work-copilot__get_item"),
	)
	writeTranscript(t, home, dirs[len(dirs)-1], "b.jsonl", time.Hour, toolUseLine(recent, "mcp__engram__mem_save"))
	writeTranscript(t, home, repo, "vieja.jsonl", 40*24*time.Hour, toolUseLine(recent, "mcp__viejo__tool"))

	usage, hasData := ScanUsage(home, dirs, usageNow)
	if !hasData {
		t.Fatal("hasData debe ser verdadero")
	}
	if got := usage["work-copilot"]; got.Count != 2 || !got.Last.Equal(usageNow.Add(-48*time.Hour)) {
		t.Errorf("work-copilot = %+v", got)
	}
	if got := usage["engram"]; got.Count != 1 {
		t.Errorf("engram = %+v (debe contar el worktree)", got)
	}
	if _, ok := usage["viejo"]; ok {
		t.Error("un fichero de más de 30 días no cuenta")
	}
	if len(usage) != 2 {
		t.Errorf("segmentos = %+v, solo work-copilot y engram", usage)
	}
}

func TestScanUsageIgnoresOldEntriesInsideARecentFile(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	stale := usageNow.Add(-45 * 24 * time.Hour).Format(time.RFC3339)
	writeTranscript(t, home, root, "a.jsonl", time.Hour, toolUseLine(stale, "mcp__engram__x"))
	usage, hasData := ScanUsage(home, []string{root}, usageNow)
	if !hasData || len(usage) != 0 {
		t.Fatalf("usage = %+v hasData = %v", usage, hasData)
	}
}

func TestAnalyzeDistinguishesUnusedFromNoData(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	names := []string{"engram", "work-copilot", "claude.ai Microsoft 365", "engram"}

	empty := Analyze(home, root, names, nil, usageNow)
	if empty.HasData {
		t.Fatal("sin transcripciones debe ser sin datos")
	}
	for _, server := range empty.Servers {
		if server.Unused {
			t.Errorf("sin datos nada es sin uso: %+v", server)
		}
	}

	recent := usageNow.Add(-time.Hour).Format(time.RFC3339)
	writeTranscript(t, home, root, "a.jsonl", time.Hour, toolUseLine(recent, "mcp__engram__mem_save"))
	got := Analyze(home, root, names, []string{"work-copilot"}, usageNow)
	if !got.HasData || len(got.Servers) != 3 {
		t.Fatalf("analysis = %+v", got)
	}
	byName := map[string]ServerUsage{}
	for _, server := range got.Servers {
		byName[server.Name] = server
	}
	if s := byName["engram"]; s.Unused || s.Uses != 1 || s.LastUsed == nil {
		t.Errorf("engram = %+v", s)
	}
	if s := byName["work-copilot"]; s.Unused {
		t.Errorf("un servidor excluido no es sin uso: %+v", s)
	}
	if s := byName["claude.ai Microsoft 365"]; !s.Unused || s.Uses != 0 {
		t.Errorf("claude.ai Microsoft 365 = %+v", s)
	}
}

func TestUnusedCandidatesJoinsUserAndProjectServers(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"engram":{},"ocioso":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(`{"mcpServers":{"del-proyecto":{},"engram":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := UnusedCandidates(home, root, nil, usageNow); got != nil {
		t.Fatalf("sin transcripciones debe ser sin datos, got %+v", got)
	}

	recent := usageNow.Add(-time.Hour).Format(time.RFC3339)
	writeTranscript(t, home, root, "a.jsonl", time.Hour, toolUseLine(recent, "mcp__engram__mem_save"))
	var names []string
	for _, server := range UnusedCandidates(home, root, []string{"ocioso"}, usageNow) {
		names = append(names, server.Name)
	}
	if want := []string{"del-proyecto"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("sin uso = %v, quiero %v", names, want)
	}
}
