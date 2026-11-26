package assets

import (
	"io/fs"
	"sort"
	"strings"
	"testing"
)

// orchestratorTemplateCount es el número de plantillas de orquestador SDD por agente
// (más la genérica) que se embeben.
const orchestratorTemplateCount = 8

// El ciclo de vida de la investigación lo resuelve el marcador canónico, una sola
// vez por orquestador.
func TestSDDOrchestratorAssetsUseCanonicalResearchGate(t *testing.T) {
	assetPaths := allSDDOrchestratorAssetPaths(t)
	if len(assetPaths) != orchestratorTemplateCount {
		t.Fatalf("SDD orchestrator family count = %d, want %d", len(assetPaths), orchestratorTemplateCount)
	}

	for _, path := range assetPaths {
		t.Run(path, func(t *testing.T) {
			content := resolveSharedOrchestratorSections(MustRead(path))
			if path == "claude/sdd-orchestrator.md" {
				content = MustRead("claude/sdd-orchestrator-workflow.md")
			}
			if strings.Count(content, "{{AXIOM_RESEARCH_LIFECYCLE}}") != 1 {
				t.Fatalf("%s must reference the canonical research lifecycle exactly once", path)
			}
		})
	}
}

func allSDDOrchestratorAssetPaths(t *testing.T) []string {
	t.Helper()
	var paths []string
	if err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "/sdd-orchestrator.md") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkDir embedded assets: %v", err)
	}
	sort.Strings(paths)
	return paths
}
