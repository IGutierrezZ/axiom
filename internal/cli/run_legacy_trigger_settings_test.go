package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	opencodeagent "github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/components/opencodedefault"
)

func seededLegacyPrompt(t *testing.T) string {
	t.Helper()
	seeded := filemerge.InjectMarkdownSection("# Política existente\n", legacyTriggerRulesSection, "Ceremonia retirada\n")
	encoded, err := json.Marshal(seeded)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// La limpieza de reglas heredadas debe actuar sobre el fichero de settings
// efectivo (opencode.jsonc del proyecto) y no sobre opencode.json global, y
// debe conservar los comentarios y las comas finales del JSONC.
func TestLegacyTriggerCleanupTargetsEffectiveSettingsAndPreservesJSONC(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "proyecto")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	globalPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	global := `{"agent":{"` + opencodedefault.ManagedAgent + `":{"prompt":` + seededLegacyPrompt(t) + `}}}`
	if err := os.WriteFile(globalPath, []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(project, "opencode.jsonc")
	before := "{\n  // nota del equipo\n  \"theme\": \"default\",\n  \"agent\": {\n    \"" + opencodedefault.ManagedAgent + "\": {\"prompt\": " + seededLegacyPrompt(t) + "},\n  },\n}\n"
	if err := os.WriteFile(projectPath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := opencodeagent.NewAdapter()
	effective := effectiveOpenCodeSettingsPath(home, project, ScopeGlobal, adapter)
	if effective != projectPath {
		t.Fatalf("effective settings = %q, want the project's opencode.jsonc %q", effective, projectPath)
	}

	result, err := stripLegacyTriggerRules(home, adapter, effective)
	if err != nil {
		t.Fatalf("stripLegacyTriggerRules error = %v", err)
	}
	if !result.Changed || len(result.Files) != 1 || result.Files[0] != projectPath {
		t.Fatalf("result = %+v, want a change confined to %q", result, projectPath)
	}
	after := readTextFile(t, projectPath)
	for _, want := range []string{"// nota del equipo", `"theme": "default",`, ",\n}"} {
		if !strings.Contains(after, want) {
			t.Fatalf("se perdió %q del JSONC:\n%s", want, after)
		}
	}
	if strings.Contains(after, "Ceremonia retirada") || !strings.Contains(after, "Política existente") {
		t.Fatalf("la sección heredada no se retiró limpiamente:\n%s", after)
	}
	if got := readTextFile(t, globalPath); got != global {
		t.Fatalf("el opencode.json global no debe tocarse:\n%s", got)
	}
}

// Un JSONC que la fusión conservadora no puede tocar con seguridad queda
// intacto y la limpieza no falla: el inyector de guía falla cerrado después.
func TestLegacyTriggerCleanupLeavesUnsafeJSONCUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.jsonc")
	content := `{"agent": {}, "agent": {"` + opencodedefault.ManagedAgent + `": {"prompt": ` + seededLegacyPrompt(t) + `}}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := stripLegacyTriggerRulesFromOrchestrator(path)
	if err != nil {
		t.Fatalf("la limpieza de mejor esfuerzo devolvió error: %v", err)
	}
	if result.Changed || len(result.Files) != 0 {
		t.Fatalf("la limpieza informó trabajo sobre un JSONC no seguro: %+v", result)
	}
	if got := readTextFile(t, path); got != content {
		t.Fatalf("se reescribió un JSONC no seguro:\n%s", got)
	}
}
