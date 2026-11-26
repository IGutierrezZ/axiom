package assets

import (
	"strings"
	"testing"
)

// Los comandos meta de Claude son puntos de entrada: llevan el frontmatter
// `description:` y el marcador `$ARGUMENTS` que sustituye Claude Code.
func TestClaudeMetaCommandsKeepTheirEntrypointMarkers(t *testing.T) {
	for _, name := range []string{"new", "continue", "ff"} {
		t.Run(name, func(t *testing.T) {
			content := MustRead("claude/commands/sdd-" + name + ".md")
			for _, required := range []string{"description:", "$ARGUMENTS"} {
				if !strings.Contains(content, required) {
					t.Errorf("entrypoint missing %q", required)
				}
			}
		})
	}
}
