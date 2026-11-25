package assets

import (
	"io/fs"
	"path"
	"regexp"
	"sort"
	"testing"
)

// engramToolReference matches an Engram tool under either Claude Code
// registration prefix, or under the {{ENGRAM_TOOL_PREFIX}} placeholder that
// injection expands to both (expandEngramToolNames in
// internal/components/sdd/inject.go).
var engramToolReference = regexp.MustCompile(`(mcp__plugin_engram_engram__|mcp__engram__|\{\{ENGRAM_TOOL_PREFIX\}\})([A-Za-z0-9_]+)`)

// engramToolsMissingFrom returns, sorted, the tools in have that other lacks.
func engramToolsMissingFrom(have, other map[string]bool) []string {
	var missing []string
	for tool := range have {
		if !other[tool] {
			missing = append(missing, tool)
		}
	}
	sort.Strings(missing)
	return missing
}

// TestEmbeddedAssetsGrantEngramToolsUnderBothPrefixes keeps any asset from
// naming an Engram tool under a single prefix. Claude Code exposes Engram as
// mcp__engram__* when it is registered with `claude mcp add engram` and as
// mcp__plugin_engram_engram__* through the plugin route, so an agent granted
// only one shape runs without memory on the other install. jd-judge-a,
// jd-judge-b and jd-fix-agent hardcoded the plugin shape while the sdd-*
// agents already used the placeholder.
func TestEmbeddedAssetsGrantEngramToolsUnderBothPrefixes(t *testing.T) {
	declaring := map[string]bool{}
	err := fs.WalkDir(FS, ".", func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		content, err := fs.ReadFile(FS, file)
		if err != nil {
			return err
		}
		matches := engramToolReference.FindAllStringSubmatch(string(content), -1)
		if len(matches) == 0 {
			return nil
		}
		declaring[file] = true

		direct, plugin := map[string]bool{}, map[string]bool{}
		for _, match := range matches {
			prefix, tool := match[1], match[2]
			switch prefix {
			case "mcp__engram__":
				direct[tool] = true
			case "mcp__plugin_engram_engram__":
				plugin[tool] = true
			default:
				// Only the sub-agent loop of sdd.Inject expands the placeholder,
				// so it counts as both shapes only inside an agents directory.
				if path.Base(path.Dir(file)) != "agents" {
					t.Errorf("%s uses {{ENGRAM_TOOL_PREFIX}}%s outside an agents directory, where injection never expands it", file, tool)
				}
				direct[tool], plugin[tool] = true, true
			}
		}
		for _, tool := range engramToolsMissingFrom(plugin, direct) {
			t.Errorf("%s grants mcp__plugin_engram_engram__%s without mcp__engram__%s; use {{ENGRAM_TOOL_PREFIX}}%s", file, tool, tool, tool)
		}
		for _, tool := range engramToolsMissingFrom(direct, plugin) {
			t.Errorf("%s grants mcp__engram__%s without mcp__plugin_engram_engram__%s; use {{ENGRAM_TOOL_PREFIX}}%s", file, tool, tool, tool)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded assets: %v", err)
	}

	// The judgment-day agents are the regression subject: if they stop
	// declaring Engram tools, the guard no longer covers them.
	for _, agent := range []string{"claude/agents/jd-judge-a.md", "claude/agents/jd-judge-b.md", "claude/agents/jd-fix-agent.md"} {
		if !declaring[agent] {
			t.Errorf("%s declares no Engram tool; the guard has lost its subject", agent)
		}
	}
}
