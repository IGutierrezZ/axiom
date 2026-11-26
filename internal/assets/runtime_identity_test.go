package assets

import (
	"io/fs"
	"regexp"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// runtimeIdentityBindingRegexp captures the runtime identity an embedded asset
// binds, in both shapes the shipped review ledger contract uses: the
// `--agent <id>` CLI flag and the `agent: <id>` consent-envelope field.
var runtimeIdentityBindingRegexp = regexp.MustCompile(`(?:--agent|agent:) +([A-Za-z0-9._{}-]+)`)

// TestEmbeddedAssetsBindNoLiteralRuntimeIdentity is the asset-layer guard for
// issue #2440, and it is deliberately stricter than a golden comparison: a
// golden diff can be re-blessed with `-update`, which would silently reinstate
// the bug it was supposed to catch. This test cannot be re-blessed. Every
// embedded asset renders verbatim into some runtime's generated instructions,
// so a literal runtime identity baked into shared prose is a claim about who
// is executing that only the renderer can truthfully make. The only admissible
// binding in an asset is the substitution placeholder
// internal/components/sdd/asset_render.go fills in with the real runtime.
func TestEmbeddedAssetsBindNoLiteralRuntimeIdentity(t *testing.T) {
	t.Parallel()

	identities := map[string]bool{}
	for _, agent := range catalog.AllAgents() {
		identities[string(agent.ID)] = true
	}
	if len(identities) == 0 {
		t.Fatal("catalog.AllAgents() enumerated no runtime identities; this guard would pass vacuously")
	}

	scanned := 0
	err := fs.WalkDir(FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}
		scanned++
		for _, match := range runtimeIdentityBindingRegexp.FindAllStringSubmatch(string(content), -1) {
			named := match[1]
			if !identities[named] {
				// Sub-agent names and other non-runtime bindings are fine:
				// only a compiled runtime identity can be handed to the
				// review transport gate.
				continue
			}
			t.Errorf("%s binds the literal runtime identity %q; assets render verbatim into every runtime's generated instructions, so a runtime that is not %q would follow this text and pass a false identity to the review transport gate. Use the %s placeholder instead.",
				path, named, named, runtimeAgentIDPlaceholderForAssets)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Fatal("walked the embedded asset tree and read no files; this guard would pass vacuously")
	}
}

// runtimeAgentIDPlaceholderForAssets mirrors
// internal/components/sdd.runtimeAgentIDPlaceholder, which is unexported and in
// another package. TestAssetPlaceholderMatchesTheRenderer keeps the two from
// drifting apart.
const runtimeAgentIDPlaceholderForAssets = "{{AXIOM_RUNTIME_AGENT_ID}}"
