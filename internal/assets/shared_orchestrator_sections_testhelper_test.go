package assets

import "strings"

// resolveSharedOrchestratorSections resolves the shared-section placeholders a
// runtime orchestrator carries. #3817 moved the subsections every runtime
// stated identically into one shared asset, so a "contains" assertion over the
// raw runtime file would now miss text that reaches every rendered prompt.
// This mirrors what composeOrchestratorPrompt does; the assets package cannot
// import the renderer without a cycle.
func resolveSharedOrchestratorSections(content string) string {
	const open = "{{GENTLE_AI_SDD_SECTION:"
	shared := MustRead("skills/_shared/sdd-orchestrator-sections.md")
	for {
		start := strings.Index(content, open)
		if start < 0 {
			return content
		}
		rest := content[start+len(open):]
		stop := strings.Index(rest, "}}")
		if stop < 0 {
			return content
		}
		name := rest[:stop]
		openMarker := "<!-- sdd-orchestrator-section:" + name + ":start -->"
		closeMarker := "<!-- sdd-orchestrator-section:" + name + ":end -->"
		body := ""
		if from := strings.Index(shared, openMarker); from >= 0 {
			if to := strings.Index(shared, closeMarker); to > from {
				body = strings.TrimSpace(shared[from+len(openMarker) : to])
			}
		}
		content = content[:start] + body + rest[stop+len("}}"):]
	}
}

// mustReadResolved reads an embedded asset the way a runtime receives it: a
// runtime orchestrator template comes back with its shared sections expanded,
// because most of the text those tests assert over now lives once in the
// shared asset. Every other asset is returned untouched.
func mustReadResolved(assetPath string) string {
	content := MustRead(assetPath)
	if strings.HasSuffix(assetPath, "/sdd-orchestrator.md") {
		return resolveSharedOrchestratorSections(content)
	}
	return content
}
