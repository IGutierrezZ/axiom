package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// El colector de investigación solo puede usar herramientas externas de lectura:
// el modo de solo lectura de Cursor y la lista de herramientas de Claude son
// concesiones de permisos, no redacción.
func TestResearchCollectorsKeepReadOnlyToolGrants(t *testing.T) {
	if !strings.Contains(assets.MustRead("cursor/agents/sdd-research.md"), "readonly: true") {
		t.Error("cursor/agents/sdd-research.md lost read-only execution")
	}
	if !strings.Contains(assets.MustRead("claude/agents/sdd-research.md"), "tools: WebFetch, WebSearch") {
		t.Error("claude/agents/sdd-research.md research tools changed")
	}
}
