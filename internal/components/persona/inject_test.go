package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/antigravity"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func antigravityAdapter() agents.Adapter { return antigravity.NewAdapter() }
func claudeAdapter() agents.Adapter      { return claude.NewAdapter() }
func opencodeAdapter() agents.Adapter    { return opencode.NewAdapter() }

func containsPersonaMarker(text string) bool {
	return strings.Contains(text, "<!-- axiom:persona -->") || strings.Contains(text, "<!-- gentle-ai:persona -->")
}

func containsPersonaCloseMarker(text string) bool {
	return strings.Contains(text, "<!-- /axiom:persona -->") || strings.Contains(text, "<!-- /gentle-ai:persona -->")
}

func countPersonaMarker(text string) int {
	return strings.Count(text, "<!-- axiom:persona -->") + strings.Count(text, "<!-- gentle-ai:persona -->")
}

func indexPersonaMarker(text string) int {
	idx := strings.Index(text, "<!-- axiom:persona -->")
	if idx >= 0 {
		return idx
	}
	return strings.Index(text, "<!-- gentle-ai:persona -->")
}

func indexPersonaCloseMarker(text string) int {
	idx := strings.Index(text, "<!-- /axiom:persona -->")
	if idx >= 0 {
		return idx
	}
	return strings.Index(text, "<!-- /gentle-ai:persona -->")
}

var claudeOutputStyleLanguageGuardrails = []string{
	"Determine the reply language from the latest actual user request",
	"For mixed-language prompts, use the dominant language of the user's direct request.",
	`phrases like "the Spanish part" do not switch the reply language by themselves.`,
	"If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence.",
	"Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments.",
	"Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language.",
	// Decision 4 union lines — reconciled from the drifted persona copy, now
	// canonical only in the output style.
	"Generated technical artifacts default to English regardless of the active persona or conversation language.",
	"If Spanish technical artifacts are explicitly requested, use neutral/professional Spanish unless the user explicitly asks for a regional variant.",
	"Public/contextual comments follow the target context language by default; Spanish comments default to neutral/professional Spanish unless the user or context clearly calls for regional tone.",
	"Do not switch languages unless the user does, asks you to, or you are quoting/translating content.",
}

func assertLanguageGuardrails(t *testing.T, text string, required []string, banned []string) {
	t.Helper()

	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing language guardrail %q", needle)
		}
	}

	for _, needle := range banned {
		if strings.Contains(text, needle) {
			t.Fatalf("contains drift-prone language instruction %q", needle)
		}
	}
}

func TestInjectClaudeGentlemanWritesSectionWithRealContent(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}

	path := filepath.Join(home, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !containsPersonaMarker(text) {
		t.Fatal("CLAUDE.md missing open marker for persona")
	}
	if !containsPersonaCloseMarker(text) {
		t.Fatal("CLAUDE.md missing close marker for persona")
	}

	// Claude has an active output-style channel — the persona section must be
	// the tooling/action residual, not the full tone/language/philosophy block
	// (that content lives exclusively in the output style now).
	if strings.Contains(text, "Senior Architect") {
		t.Fatal("CLAUDE.md persona section should be a residual — 'Senior Architect' tone content must not appear")
	}
	if strings.Contains(text, "Persona Scope") {
		t.Fatal("CLAUDE.md persona section should be a residual — 'Persona Scope' now lives only in the output style")
	}
	if !strings.Contains(text, "## Rules") {
		t.Fatal("CLAUDE.md residual persona section missing '## Rules'")
	}
	if !strings.Contains(text, "## Expertise") {
		t.Fatal("CLAUDE.md residual persona section missing '## Expertise'")
	}
	if !strings.Contains(text, "Persona Voice") {
		t.Fatal("CLAUDE.md residual persona section missing the 'Persona Voice' pointer to the output style")
	}
}

func TestInjectClaudeGentlemanWritesOutputStyleFile(t *testing.T) {
	home := t.TempDir()

	_, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Verify output-style file was written.
	stylePath := filepath.Join(home, ".claude", "output-styles", "gentleman.md")
	content, err := os.ReadFile(stylePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", stylePath, err)
	}

	text := string(content)
	if !strings.Contains(text, "name: Gentleman") {
		t.Fatal("Output style file missing YAML frontmatter 'name: Gentleman'")
	}
	if !strings.Contains(text, "keep-coding-instructions: true") {
		t.Fatal("Output style file missing 'keep-coding-instructions: true'")
	}
	if !strings.Contains(text, "Gentleman Output Style") {
		t.Fatal("Output style file missing 'Gentleman Output Style' heading")
	}
	assertLanguageGuardrails(t, text, claudeOutputStyleLanguageGuardrails, nil)
}

func TestInjectClaudeGentlemanMergesOutputStyleIntoSettings(t *testing.T) {
	home := t.TempDir()

	// Pre-create a settings.json with some existing content.
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	existingSettings := `{"permissions": {"allow": ["Read"]}, "syntaxHighlightingDisabled": true}`
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(existingSettings), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Verify settings.json has outputStyle merged in.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settingsContent, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", settingsPath, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsContent, &settings); err != nil {
		t.Fatalf("Unmarshal settings.json error = %v", err)
	}

	outputStyle, ok := settings["outputStyle"]
	if !ok {
		t.Fatal("settings.json missing 'outputStyle' key")
	}
	if outputStyle != "Gentleman" {
		t.Fatalf("settings.json outputStyle = %q, want %q", outputStyle, "Gentleman")
	}

	// Verify existing keys were preserved.
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("settings.json lost 'permissions' key during merge")
	}
	if _, ok := settings["syntaxHighlightingDisabled"]; !ok {
		t.Fatal("settings.json lost 'syntaxHighlightingDisabled' key during merge")
	}
}

func TestInjectClaudeGentlemanReturnsAllFiles(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Should return 3 files: CLAUDE.md, output-style, settings.json.
	if len(result.Files) != 3 {
		t.Fatalf("Inject() returned %d files, want 3: %v", len(result.Files), result.Files)
	}

	wantSuffixes := []string{"CLAUDE.md", "gentleman.md", "settings.json"}
	for _, suffix := range wantSuffixes {
		found := false
		for _, f := range result.Files {
			if strings.HasSuffix(f, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Inject() missing file with suffix %q in %v", suffix, result.Files)
		}
	}
}

func TestInjectClaudeNeutralWritesResidualPersonaWithoutRegionalLanguage(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}

	path := filepath.Join(home, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	// Neutral also has an active output-style channel for Claude — the persona
	// section is the residual, not the full teaching-tone block.
	if strings.Contains(text, "Senior Architect") {
		t.Fatal("Neutral persona section should be a residual — 'Senior Architect' must not appear")
	}
	if !strings.Contains(text, "## Rules") {
		t.Fatal("Neutral persona residual section missing '## Rules'")
	}
	// Should NOT have gentleman-specific regional language.
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("Neutral persona should not contain Rioplatense language")
	}
}

func TestInjectClaudeNeutralWritesNeutralOutputStyleAndSettings(t *testing.T) {
	home := t.TempDir()
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(filepath.Join(settingsDir, "output-styles"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	staleGentlemanPath := filepath.Join(settingsDir, "output-styles", "gentleman.md")
	if err := os.WriteFile(staleGentlemanPath, []byte("stale gentleman style"), 0o644); err != nil {
		t.Fatalf("WriteFile(stale gentleman) error = %v", err)
	}
	existingSettings := `{"permissions":{"allow":["Read"]},"outputStyle":"Gentleman"}`
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(existingSettings), 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	result, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	for _, suffix := range []string{"CLAUDE.md", "neutral.md", "settings.json", "gentleman.md"} {
		found := false
		for _, file := range result.Files {
			if strings.HasSuffix(file, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Neutral persona result missing %q in files %v", suffix, result.Files)
		}
	}

	neutralStylePath := filepath.Join(home, ".claude", "output-styles", "neutral.md")
	styleContent, err := os.ReadFile(neutralStylePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", neutralStylePath, err)
	}
	styleText := string(styleContent)
	for _, want := range []string{"name: Neutral", "Neutral Output Style", "minimum useful response", "Generated technical artifacts default to English"} {
		if !strings.Contains(styleText, want) {
			t.Fatalf("neutral output style missing %q; got:\n%s", want, styleText)
		}
	}
	assertLanguageGuardrails(t, styleText, claudeOutputStyleLanguageGuardrails, nil)
	if strings.Contains(styleText, "Rioplatense") || strings.Contains(styleText, "voseo") {
		t.Fatalf("neutral output style contains regional wording:\n%s", styleText)
	}
	if _, err := os.Stat(staleGentlemanPath); !os.IsNotExist(err) {
		t.Fatalf("stale gentleman output style should be removed, stat err=%v", err)
	}

	settingsContent, err := os.ReadFile(filepath.Join(settingsDir, "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsContent, &settings); err != nil {
		t.Fatalf("Unmarshal settings error = %v", err)
	}
	if got, want := settings["outputStyle"], "Neutral"; got != want {
		t.Fatalf("settings outputStyle = %q, want %q", got, want)
	}
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("settings lost existing permissions key")
	}

	second, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("second neutral Claude inject changed = true, want idempotent false")
	}
}

func TestInjectCustomClaudeDoesNothing(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter(), model.PersonaCustom)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.Changed {
		t.Fatal("Custom persona should NOT change anything")
	}
	if len(result.Files) != 0 {
		t.Fatalf("Custom persona should return no files, got %v", result.Files)
	}

	// CLAUDE.md should NOT be created.
	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(claudeMD); !os.IsNotExist(err) {
		t.Fatal("Custom persona should NOT create CLAUDE.md")
	}
}

func TestInjectCustomOpenCodeDoesNothing(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, opencodeAdapter(), model.PersonaCustom)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.Changed {
		t.Fatal("Custom persona (OpenCode) should NOT change anything")
	}
	if len(result.Files) != 0 {
		t.Fatalf("Custom persona (OpenCode) should return no files, got %v", result.Files)
	}

	// AGENTS.md should NOT be created.
	agentsMD := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if _, err := os.Stat(agentsMD); !os.IsNotExist(err) {
		t.Fatal("Custom persona (OpenCode) should NOT create AGENTS.md")
	}
}

func TestInjectPiPersonaWritesSelectedModeAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path := PiPersonaConfigPath(root)

	result, err := InjectPiPersona(root, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("InjectPiPersona() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("first Pi persona injection changed = false, want true")
	}
	if len(result.Files) != 1 || result.Files[0] != path {
		t.Fatalf("first Pi persona injection files = %v, want [%q]", result.Files, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if got, want := string(content), "{\n  \"mode\": \"neutral\"\n}\n"; got != want {
		t.Fatalf("Pi persona config = %q, want %q", got, want)
	}

	second, err := InjectPiPersona(root, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("second InjectPiPersona() error = %v", err)
	}
	if second.Changed {
		t.Fatalf("second Pi persona injection changed = true, want false")
	}
}

func TestInjectPiPersonaCustomPreservesExistingConfig(t *testing.T) {
	root := t.TempDir()
	path := PiPersonaConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	const existing = "{\n  \"mode\": \"user-defined\"\n}\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := InjectPiPersona(root, model.PersonaCustom)
	if err != nil {
		t.Fatalf("InjectPiPersona(custom) error = %v", err)
	}
	if result.Changed || len(result.Files) != 0 {
		t.Fatalf("InjectPiPersona(custom) result = %#v, want no-op", result)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(content) != existing {
		t.Fatalf("custom Pi persona config = %q, want existing content %q", content, existing)
	}
}

func TestInjectOpenCodeGentlemanWritesAgentsFile(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}

	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("AGENTS.md missing real persona content")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing persona marker")
	}
}

func TestInjectAntigravityGentlemanWritesMarkedPersonaSection(t *testing.T) {
	home := t.TempDir()
	promptPath := filepath.Join(home, ".gemini", "GEMINI.md")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("# User Gemini rules\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := Inject(home, antigravityAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !containsPersonaMarker(text) || !containsPersonaCloseMarker(text) || !strings.Contains(text, "# User Gemini rules") || !strings.Contains(text, "Senior Architect") {
		t.Fatalf("GEMINI.md missing required content; got:\n%s", text)
	}

	second, err := Inject(home, antigravityAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true; want false")
	}

	content, err = os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() after second inject error = %v", err)
	}
	if got := countPersonaMarker(string(content)); got != 1 {
		t.Fatalf("persona marker count = %d, want 1", got)
	}
}

func TestInjectOpenCodeGentlemanDoesNotCreateSDDConductor(t *testing.T) {
	home := t.TempDir()

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}
	text := string(content)
	if strings.Contains(text, `"sdd-orchestrator"`) {
		t.Fatal("persona injection must not create legacy sdd-orchestrator conductor")
	}
	if strings.Contains(text, `"gentle-orchestrator"`) {
		t.Fatal("persona injection must not create SDD conductor; SDD component owns gentle-orchestrator")
	}
	if !strings.Contains(text, `"gentleman"`) {
		t.Fatal("persona injection should still create the gentleman persona agent")
	}
	if strings.Contains(text, `"tools"`) {
		t.Fatal("persona injection must not emit deprecated gentleman tools")
	}
}

func TestInjectForSyncOpenCodeGentlemanRemovesOnlyStaleTools(t *testing.T) {
	home := t.TempDir()
	settingsPath := opencodeAdapter().SettingsPath(home)
	before := `{
  "user-setting": {"keep": true},
  "agent": {
    "gentleman": {
      "mode": "primary",
      "description": "keep this persona",
      "tools": {"write": true, "edit": true}
    },
    "user-owned": {"tools": {"custom": true}}
  }
}
`
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := InjectForSync(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("InjectForSync() error = %v", err)
	}
	if !first.Changed {
		t.Fatal("Gentleman sync did not remove stale tools")
	}
	payload, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	root := map[string]any{}
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatal(err)
	}
	if got := root["user-setting"]; fmt.Sprint(got) != "map[keep:true]" {
		t.Fatalf("user-owned setting changed: %#v", got)
	}
	agents := root["agent"].(map[string]any)
	gentleman := agents["gentleman"].(map[string]any)
	if _, exists := gentleman["tools"]; exists {
		t.Fatalf("Gentleman sync retained stale tools: %#v", gentleman["tools"])
	}
	if got := gentleman["description"]; got != "keep this persona" {
		t.Fatalf("Gentleman sync overwrote persona data: %#v", got)
	}
	if got := agents["user-owned"].(map[string]any)["tools"]; fmt.Sprint(got) != "map[custom:true]" {
		t.Fatalf("Gentleman sync changed user-owned agent tools: %#v", got)
	}

	second, err := InjectForSync(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("second InjectForSync() error = %v", err)
	}
	if second.Changed {
		t.Fatal("second Gentleman sync changed already-clean settings")
	}
}

func TestInjectForSyncOpenCodeGentlemanSucceedsWithoutSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := opencodeAdapter().SettingsPath(home)

	result, err := InjectForSync(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("InjectForSync() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("first Gentleman sync should write the persona file")
	}
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Fatalf("settings file was created or stat failed: %v", err)
	}
}

func TestInjectOpenCodePreservesUserContentInsteadOfOverwriting(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	userContent := "# My custom rules\n\nDo not overwrite this file.\n"
	if err := os.WriteFile(path, []byte(userContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "Do not overwrite this file.") {
		t.Fatal("AGENTS.md user content was overwritten")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing managed persona section after inject")
	}
}

func TestInjectOpenCodeDoesNotStripLookalikeUserContent(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookalike := "## Rules\n\n- Team rules.\n\n## Personality\n\nSenior Architect for my org.\n\nDo not delete this custom preface.\n"
	if err := os.WriteFile(path, []byte(lookalike), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "Do not delete this custom preface.") {
		t.Fatal("OpenCode AGENTS.md lookalike user content was stripped")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing managed persona section after inject")
	}
}

func TestInjectOpenCodePreservesUserPrefaceAboveATLBlock(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// User has custom content with fingerprint-like headings ABOVE an old ATL block.
	// ATL markers must NOT trigger persona legacy stripping.
	existing := "## Rules\n\n- My team's custom rules.\n\n## Personality\n\nSenior Architect in my org.\n\n" +
		"<!-- BEGIN:agent-teams-lite -->\nOld ATL content.\n<!-- END:agent-teams-lite -->\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "My team's custom rules.") {
		t.Fatal("user preface above ATL block was stripped — ATL should not enable persona stripping")
	}
	if strings.Contains(text, "BEGIN:agent-teams-lite") {
		t.Fatal("ATL block should have been stripped by StripLegacyATLBlock")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing managed persona section")
	}
}

func TestInjectOpenCodeReplacesExactLegacyAssetWithoutDuplication(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Write the exact legacy asset (no markers) — simulates old installer output.
	legacyContent := assets.MustRead("opencode/persona-gentleman.md")
	if err := os.WriteFile(path, []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	// Must have exactly ONE persona marker — no duplication.
	if countPersonaMarker(text) != 1 {
		t.Fatalf("expected exactly 1 persona marker, got %d — legacy asset was not replaced cleanly",
			countPersonaMarker(text))
	}
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("persona content missing after replacing legacy asset")
	}
}

func TestInjectOpenCodePreservesUserPrefaceAboveManagedMarkers(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Simulate: user has custom content with fingerprint-like headings ABOVE
	// existing managed markers. This is the exact scenario where aggressive
	// legacy stripping would destroy user content.
	existing := "## Rules\n\n- My team's custom rules.\n\n## Personality\n\nSenior Architect in my org.\n\n" +
		"<!-- gentle-ai:engram-protocol -->\nEngram protocol here.\n<!-- /gentle-ai:engram-protocol -->\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "My team's custom rules.") {
		t.Fatal("user preface above managed markers was stripped — should be preserved")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing managed persona section after inject")
	}
	if !strings.Contains(text, "<!-- gentle-ai:engram-protocol -->") {
		t.Fatal("existing engram section was lost")
	}
}

func TestInjectOpenCodeNeutralPreservesManagedSections(t *testing.T) {
	home := t.TempDir()

	// First install gentleman persona + simulate SDD/engram sections
	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")

	// Simulate SDD and engram sections appended by sdd.Inject and engram.Inject
	existing, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	withSections := string(existing) + "\n\n<!-- gentle-ai:sdd-orchestrator -->\nSDD orchestrator content here\n<!-- /gentle-ai:sdd-orchestrator -->\n\n<!-- gentle-ai:engram-protocol -->\nEngram protocol content here\n<!-- /gentle-ai:engram-protocol -->\n"
	if err := os.WriteFile(path, []byte(withSections), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Now switch to neutral persona
	result, err := Inject(home, opencodeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(neutral) should report changed")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after neutral error = %v", err)
	}
	text := string(content)

	// Neutral content should be present
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("AGENTS.md missing neutral persona content")
	}
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("AGENTS.md has Rioplatense language in neutral persona — should be neutral tone")
	}

	// Managed sections MUST be preserved
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("AGENTS.md lost SDD orchestrator section after switching to neutral persona")
	}
	if !strings.Contains(text, "<!-- gentle-ai:engram-protocol -->") {
		t.Fatal("AGENTS.md lost engram protocol section after switching to neutral persona")
	}

	// Gentleman-specific language should be gone — neutral has the same personality but no regional language
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("AGENTS.md still has Rioplatense language after switching to neutral")
	}
}

func TestInjectForSyncNeutralCleansOnlyGentlemanAgent(t *testing.T) {
	for _, tc := range []struct {
		name        string
		adapter     agents.Adapter
		settingsRel string
	}{
		{name: "opencode", adapter: opencodeAdapter(), settingsRel: filepath.Join(".config", "opencode", "opencode.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			settingsPath := filepath.Join(home, tc.settingsRel)
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			existing := `{"agent":{"gentleman":{"mode":"primary"},"custom":{"mode":"primary"}},"theme":"dark"}`
			if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
				t.Fatalf("WriteFile(settings) error = %v", err)
			}

			result, err := InjectForSync(home, tc.adapter, model.PersonaNeutral)
			if err != nil {
				t.Fatalf("InjectForSync() error = %v", err)
			}
			if !result.Changed {
				t.Fatal("InjectForSync() changed = false, want cleanup change")
			}

			content, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("ReadFile(settings) error = %v", err)
			}
			var root map[string]any
			if err := json.Unmarshal(content, &root); err != nil {
				t.Fatalf("Unmarshal(settings) error = %v", err)
			}
			agentMap, ok := root["agent"].(map[string]any)
			if !ok {
				t.Fatalf("settings lost agent object: %s", string(content))
			}
			if _, exists := agentMap["gentleman"]; exists {
				t.Fatalf("settings still has agent.gentleman: %s", string(content))
			}
			if _, exists := agentMap["custom"]; !exists {
				t.Fatalf("settings lost agent.custom sibling: %s", string(content))
			}
			if got, want := root["theme"], "dark"; got != want {
				t.Fatalf("settings theme = %q, want %q", got, want)
			}
		})
	}
}

func TestInjectForSyncNeutralToleratesMalformedOpenCodeSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	malformed := []byte(`{"agent":`)
	if err := os.WriteFile(settingsPath, malformed, 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	if _, err := InjectForSync(home, opencodeAdapter(), model.PersonaNeutral); err != nil {
		t.Fatalf("InjectForSync() should tolerate malformed settings, got error: %v", err)
	}
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	if string(content) != string(malformed) {
		t.Fatalf("malformed settings should be preserved untouched; got %q", string(content))
	}
}

func TestInjectVSCodeNeutralPreservesManagedSections(t *testing.T) {
	home := t.TempDir()

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	_, err = Inject(home, vscodeAdapter, model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	path := vscodeAdapter.SystemPromptFile(home)

	existing, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	withSections := string(existing) + "\n\n<!-- gentle-ai:sdd-orchestrator -->\nSDD content\n<!-- /gentle-ai:sdd-orchestrator -->\n"
	if err := os.WriteFile(path, []byte(withSections), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = Inject(home, vscodeAdapter, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after neutral error = %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("instructions file missing neutral persona content")
	}
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("instructions file has Rioplatense language in neutral persona")
	}
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("instructions file lost SDD section after switching to neutral persona")
	}
	if !strings.Contains(text, "---\nname:") {
		t.Fatal("instructions file lost YAML frontmatter")
	}
}

func TestInjectNeutralPreservesWhenMarkerAtByteZero(t *testing.T) {
	home := t.TempDir()

	opencodeAdapter, err := agents.NewAdapter("opencode")
	if err != nil {
		t.Fatalf("NewAdapter(opencode) error = %v", err)
	}

	promptPath := opencodeAdapter.SystemPromptFile(home)
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// File starts DIRECTLY with a managed marker at byte 0 — no persona preamble.
	markerOnly := "<!-- gentle-ai:sdd-orchestrator -->\nSDD content\n<!-- /gentle-ai:sdd-orchestrator -->\n"
	if err := os.WriteFile(promptPath, []byte(markerOnly), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = Inject(home, opencodeAdapter, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("missing neutral persona content")
	}
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("SDD section destroyed when marker was at byte 0")
	}
}

func TestInjectNeutralIdempotentWithManagedSections(t *testing.T) {
	home := t.TempDir()

	opencodeAdapter, err := agents.NewAdapter("opencode")
	if err != nil {
		t.Fatalf("NewAdapter(opencode) error = %v", err)
	}

	promptPath := opencodeAdapter.SystemPromptFile(home)
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Set up: neutral + managed sections
	// Simulate a file with neutral persona + managed sections.
	// Use a fingerprint from the real neutral asset so the test is realistic.
	neutralContent := assets.MustRead("generic/persona-neutral.md")
	initial := neutralContent + "\n\n<!-- gentle-ai:sdd-orchestrator -->\nSDD content\n<!-- /gentle-ai:sdd-orchestrator -->\n\n<!-- gentle-ai:engram-protocol -->\nEngram content\n<!-- /gentle-ai:engram-protocol -->\n"
	if err := os.WriteFile(promptPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// First neutral inject
	result1, err := Inject(home, opencodeAdapter, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) first error = %v", err)
	}

	// Second neutral inject — should be idempotent
	result2, err := Inject(home, opencodeAdapter, model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) second error = %v", err)
	}

	if result2.Changed && !result1.Changed {
		t.Fatal("second neutral inject should not report changed when first didn't")
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	// Verify no duplication
	if strings.Count(text, "<!-- gentle-ai:sdd-orchestrator -->") != 1 {
		t.Fatal("SDD section duplicated after idempotent neutral inject")
	}
	if strings.Count(text, "## Rules") != 1 {
		t.Fatal("neutral persona duplicated after idempotent inject")
	}
	if strings.Count(text, "<!-- gentle-ai:engram-protocol -->") != 1 {
		t.Fatal("engram section duplicated after idempotent neutral inject")
	}
}

func TestInjectClaudeIsIdempotent(t *testing.T) {
	home := t.TempDir()

	first, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}

	second, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true")
	}
}

func TestInjectOpenCodeIsIdempotent(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(opencodeAdapter().SettingsPath(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opencodeAdapter().SettingsPath(home), []byte(`{"theme":"dark","agent":{"gentleman":{"tools":{"write":true}},"user":{"tools":{"read":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}
	settings, err := os.ReadFile(opencodeAdapter().SettingsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	root := map[string]any{}
	if err := json.Unmarshal(settings, &root); err != nil {
		t.Fatal(err)
	}
	agents := root["agent"].(map[string]any)
	if _, exists := agents["gentleman"].(map[string]any)["tools"]; exists || root["theme"] != "dark" || agents["user"].(map[string]any)["tools"].(map[string]any)["read"] != true {
		t.Fatalf("stale managed tools cleanup changed user settings: %#v", root)
	}

	second, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true")
	}
}

func TestInjectWindsurfIsIdempotent(t *testing.T) {
	home := t.TempDir()

	windsurfAdapter, err := agents.NewAdapter("windsurf")
	if err != nil {
		t.Fatalf("NewAdapter(windsurf) error = %v", err)
	}

	first, err := Inject(home, windsurfAdapter, model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}

	promptPath := windsurfAdapter.SystemPromptFile(home)
	contentAfterFirst, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() after first inject error = %v", err)
	}

	second, err := Inject(home, windsurfAdapter, model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true — persona was duplicated in global_rules.md")
	}

	contentAfterSecond, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() after second inject error = %v", err)
	}

	if string(contentAfterFirst) != string(contentAfterSecond) {
		t.Fatal("global_rules.md content changed on second inject — persona was duplicated")
	}
}

func TestInjectCursorGentlemanWritesRulesFileWithRealContent(t *testing.T) {
	home := t.TempDir()

	cursorAdapter, err := agents.NewAdapter("cursor")
	if err != nil {
		t.Fatalf("NewAdapter(cursor) error = %v", err)
	}

	result, injectErr := Inject(home, cursorAdapter, model.PersonaGentleman)
	if injectErr != nil {
		t.Fatalf("Inject(cursor) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatalf("Inject(cursor, gentleman) changed = false")
	}

	// Verify the generic persona content was used — not just neutral one-liner.
	path := filepath.Join(home, ".cursor", "rules", "axiom.mdc")
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, readErr)
	}

	text := string(content)
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("Cursor persona missing 'Senior Architect' — got neutral fallback instead of generic persona")
	}
	if !strings.Contains(text, "Contextual Skill Loading") {
		t.Fatal("Cursor persona missing contextual skill loading directive")
	}
}

func TestInjectGeminiGentlemanWritesSystemPromptWithRealContent(t *testing.T) {
	home := t.TempDir()

	geminiAdapter, err := agents.NewAdapter("gemini-cli")
	if err != nil {
		t.Fatalf("NewAdapter(gemini-cli) error = %v", err)
	}

	result, injectErr := Inject(home, geminiAdapter, model.PersonaGentleman)
	if injectErr != nil {
		t.Fatalf("Inject(gemini) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatal("Inject(gemini, gentleman) changed = false")
	}

	path := filepath.Join(home, ".gemini", "GEMINI.md")
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, readErr)
	}

	text := string(content)
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("Gemini persona missing 'Senior Architect'")
	}
	assertLanguageGuardrails(t, text,
		[]string{
			"Match the user's current language in your REPLY ONLY",
			"Do not switch languages unless the user does, asks you to, or you are quoting/translating content.",
			"When replying to the user in English, keep the full reply in natural English with the same warm energy.",
		},
		[]string{
			`Say "déjame verificar"`,
			"Spanish input → Rioplatense Spanish",
			"English input → same warm energy",
		},
	)
}

func TestInjectVSCodeGentlemanWritesInstructionsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	result, injectErr := Inject(home, vscodeAdapter, model.PersonaGentleman)
	if injectErr != nil {
		t.Fatalf("Inject(vscode) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatal("Inject(vscode, gentleman) changed = false")
	}

	path := vscodeAdapter.SystemPromptFile(home)
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, readErr)
	}

	text := string(content)
	if !strings.Contains(text, "applyTo: \"**\"") {
		t.Fatal("VS Code instructions file missing YAML frontmatter applyTo pattern")
	}
	if !strings.Contains(text, "Senior Architect") {
		t.Fatal("VS Code persona missing 'Senior Architect'")
	}
}

// --- Auto-heal tests: Claude Code stale free-text persona ---

// legacyClaudePersonaBlock simulates a Gentleman persona block that was written
// directly (without markers) by an old installer or manually by the user.
const legacyClaudePersonaBlock = `## Rules

- NEVER add "Co-Authored-By" or any AI attribution to commits. Use conventional commits format only.

## Personality

Senior Architect, 15+ years experience, GDE & MVP.

## Language

- Spanish input → Rioplatense Spanish.

## Behavior

- Push back when user asks for code without context.

`

func TestInjectClaudeAutoHealsStaleFreeTextPersona(t *testing.T) {
	home := t.TempDir()

	// Pre-populate CLAUDE.md with legacy persona content (no markers) followed
	// by a properly-marked section from a previous installer run.
	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(claudeMD), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}

	// Simulate a stale install: free-text persona block at top, then a different
	// marked section below (e.g., from a previous SDD install).
	stalePreamble := legacyClaudePersonaBlock + "\n<!-- gentle-ai:sdd -->\nOld SDD content.\n<!-- /gentle-ai:sdd -->\n"
	if err := os.WriteFile(claudeMD, []byte(stalePreamble), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	result, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject() should have changed the file to remove the legacy block")
	}

	content, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	// The file should now have the persona inside markers, not as free text.
	if !containsPersonaMarker(text) {
		t.Fatal("CLAUDE.md missing persona marker after heal")
	}
	if !containsPersonaCloseMarker(text) {
		t.Fatal("CLAUDE.md missing persona close marker after heal")
	}

	// The existing SDD section must be preserved.
	if !strings.Contains(text, "<!-- gentle-ai:sdd -->") {
		t.Fatal("CLAUDE.md lost the sdd section during heal")
	}
	if !strings.Contains(text, "Old SDD content.") {
		t.Fatal("CLAUDE.md lost the sdd section content during heal")
	}

	// The persona content is now the residual block — the legacy fixture's
	// "Senior Architect" tone text must be fully stripped, not preserved or
	// duplicated anywhere in the healed file.
	if strings.Contains(text, "Senior Architect") {
		t.Fatal("CLAUDE.md still contains legacy 'Senior Architect' text — legacy block not fully stripped")
	}

	openMarkerIdx := indexPersonaMarker(text)
	closeMarkerIdx := indexPersonaCloseMarker(text)
	if openMarkerIdx < 0 || closeMarkerIdx < 0 || closeMarkerIdx < openMarkerIdx {
		t.Fatal("CLAUDE.md missing a valid persona marker section after heal")
	}
	markerSection := text[openMarkerIdx:closeMarkerIdx]
	if !strings.Contains(markerSection, "## Rules") {
		t.Fatal("healed persona marker section missing residual '## Rules'")
	}
	if !strings.Contains(markerSection, "Persona Voice") {
		t.Fatal("healed persona marker section missing residual 'Persona Voice' pointer")
	}
}

func TestInjectClaudeAutoHealStalePersonaOnlyFile(t *testing.T) {
	home := t.TempDir()

	// CLAUDE.md contains ONLY the legacy persona block (no markers at all).
	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(claudeMD), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(claudeMD, []byte(legacyClaudePersonaBlock), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	result, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject() should have changed the file")
	}

	content, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	// Must have markers now.
	if !containsPersonaMarker(text) {
		t.Fatal("CLAUDE.md missing persona marker")
	}

	// Must NOT have the legacy free-text block before markers.
	openMarkerIdx := indexPersonaMarker(text)
	if openMarkerIdx >= 0 {
		before := text[:openMarkerIdx]
		if strings.Contains(before, "## Rules") {
			t.Fatal("legacy '## Rules' block still present before persona marker")
		}
	}

	// The legacy fixture's "Senior Architect" tone content must be fully
	// replaced by the residual, not preserved anywhere in the healed file.
	if strings.Contains(text, "Senior Architect") {
		t.Fatal("CLAUDE.md still contains legacy 'Senior Architect' text after heal")
	}
	if !strings.Contains(text, "Persona Voice") {
		t.Fatal("CLAUDE.md missing residual 'Persona Voice' pointer after heal")
	}
}

func TestInjectClaudeHealDoesNotTouchNonPersonaContent(t *testing.T) {
	home := t.TempDir()

	// CLAUDE.md has user content that does NOT match persona fingerprints.
	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(claudeMD), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	userContent := "# My custom config\n\nI like turtles.\n"
	if err := os.WriteFile(claudeMD, []byte(userContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	result, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject() should write persona section")
	}

	content, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	// User content must be preserved.
	if !strings.Contains(text, "I like turtles.") {
		t.Fatal("user content was erased — heal was too aggressive")
	}
	// Persona section must be appended.
	if !containsPersonaMarker(text) {
		t.Fatal("persona section not appended")
	}
}

// --- Auto-heal tests: VSCode stale legacy path cleanup ---

func TestInjectVSCodeCleansLegacyGitHubPersonaFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Plant an old-style Gentleman persona file at the legacy path.
	legacyPath := filepath.Join(home, ".github", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	// Old installer wrote raw persona content without YAML frontmatter.
	oldContent := "## Personality\n\nSenior Architect, 15+ years experience.\n"
	if err := os.WriteFile(legacyPath, []byte(oldContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	result, injectErr := Inject(home, vscodeAdapter, model.PersonaGentleman)
	if injectErr != nil {
		t.Fatalf("Inject(vscode) error = %v", injectErr)
	}
	if !result.Changed {
		t.Fatal("Inject(vscode) should report changed (legacy cleanup + new file write)")
	}

	// Legacy file must be gone.
	if _, statErr := os.Stat(legacyPath); !os.IsNotExist(statErr) {
		t.Fatal("legacy ~/.github/copilot-instructions.md was NOT removed by auto-heal")
	}

	// New file must exist at the current path.
	newPath := vscodeAdapter.SystemPromptFile(home)
	content, readErr := os.ReadFile(newPath)
	if readErr != nil {
		t.Fatalf("ReadFile new path %q error = %v", newPath, readErr)
	}
	if !strings.Contains(string(content), "applyTo: \"**\"") {
		t.Fatal("new VSCode instructions file missing YAML frontmatter")
	}
}

func TestInjectVSCodePreservesNonPersonaGitHubFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Plant a .github/copilot-instructions.md that has user content (not a
	// Gentleman persona) — it must NOT be deleted.
	legacyPath := filepath.Join(home, ".github", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	userContent := "# My custom Copilot instructions\n\nAlways be concise.\n"
	if err := os.WriteFile(legacyPath, []byte(userContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	_, injectErr := Inject(home, vscodeAdapter, model.PersonaGentleman)
	if injectErr != nil {
		t.Fatalf("Inject(vscode) error = %v", injectErr)
	}

	// User's file must still exist.
	remaining, readErr := os.ReadFile(legacyPath)
	if readErr != nil {
		t.Fatalf("legacy user file was deleted: ReadFile error = %v", readErr)
	}
	if string(remaining) != userContent {
		t.Fatalf("user file content was modified: got %q", string(remaining))
	}
}

func TestNeutralAndGentlemanToneSectionsMatch(t *testing.T) {
	neutral := assets.MustRead("generic/persona-neutral.md")
	gentleman := assets.MustRead("generic/persona-gentleman.md")

	extractSection := func(content, section string) string {
		idx := strings.Index(content, "## "+section)
		if idx < 0 {
			return ""
		}
		rest := content[idx:]
		nextIdx := strings.Index(rest[1:], "\n## ")
		if nextIdx < 0 {
			return rest
		}
		return rest[:nextIdx+1]
	}

	neutralTone := extractSection(neutral, "Tone")
	gentlemanTone := extractSection(gentleman, "Tone")

	if neutralTone != gentlemanTone {
		t.Fatalf("## Tone sections diverged:\nneutral:\n%s\ngentleman:\n%s", neutralTone, gentlemanTone)
	}
}

func TestInjectVSCodeIdempotentAfterHeal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Plant legacy file and run inject twice — second run should be idempotent.
	legacyPath := filepath.Join(home, ".github", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("## Personality\n\nSenior Architect.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	first, err := Inject(home, vscodeAdapter, model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatal("first inject should have changed")
	}

	second, err := Inject(home, vscodeAdapter, model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("second inject should be idempotent (changed = false), but changed = true")
	}
}

func TestInjectClaude_SwitchGentlemanToNeutral_CleansOutputStyle(t *testing.T) {
	home := t.TempDir()

	// Step 1: install gentleman — creates output-styles/gentleman.md and sets outputStyle in settings.json.
	_, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	stylePath := filepath.Join(home, ".claude", "output-styles", "gentleman.md")
	if _, statErr := os.Stat(stylePath); os.IsNotExist(statErr) {
		t.Fatal("precondition: gentleman.md must exist after gentleman install")
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settingsRaw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("precondition: settings.json must exist after gentleman install: %v", err)
	}
	var settingsBefore map[string]any
	if err := json.Unmarshal(settingsRaw, &settingsBefore); err != nil {
		t.Fatalf("precondition: unmarshal settings.json: %v", err)
	}
	if settingsBefore["outputStyle"] != "Gentleman" {
		t.Fatalf("precondition: outputStyle must be 'Gentleman', got %v", settingsBefore["outputStyle"])
	}

	// Step 2: switch to neutral — should clean both residuals.
	result, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(neutral) should report changed when cleaning gentleman residuals")
	}

	// output-styles/gentleman.md must be gone.
	if _, statErr := os.Stat(stylePath); !os.IsNotExist(statErr) {
		t.Fatal("gentleman.md must be removed when switching to neutral")
	}

	// outputStyle must now point at the managed Neutral style.
	settingsRaw, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings.json) after neutral: %v", err)
	}
	var settingsAfter map[string]any
	if err := json.Unmarshal(settingsRaw, &settingsAfter); err != nil {
		t.Fatalf("Unmarshal settings.json after neutral: %v", err)
	}
	if got, want := settingsAfter["outputStyle"], "Neutral"; got != want {
		t.Fatalf("outputStyle = %v, want %q after switching to neutral", got, want)
	}
}

func TestInjectClaude_NeutralSelectsManagedOutputStyleAndPreservesOtherSettings(t *testing.T) {
	home := t.TempDir()

	// Pre-create settings.json with a user-defined outputStyle that is NOT "Gentleman".
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	userSettings := `{"outputStyle": "MyCustom", "syntaxHighlightingDisabled": true}`
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(userSettings), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}

	settingsRaw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		t.Fatalf("Unmarshal settings.json error = %v", err)
	}

	if got, want := settings["outputStyle"], "Neutral"; got != want {
		t.Fatalf("outputStyle = %v, want %q", got, want)
	}
	// Other user keys must also survive.
	if settings["syntaxHighlightingDisabled"] != true {
		t.Fatal("syntaxHighlightingDisabled was lost")
	}
}

func TestInjectClaude_SwitchGentlemanToNeutral_IsIdempotent(t *testing.T) {
	home := t.TempDir()

	// Install gentleman, then switch to neutral twice — second switch must be a no-op.
	_, err := Inject(home, claudeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	first, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) first error = %v", err)
	}
	if !first.Changed {
		t.Fatal("first neutral inject after gentleman should report changed")
	}

	second, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) second error = %v", err)
	}
	if second.Changed {
		t.Fatal("second neutral inject should be idempotent (no residuals to clean)")
	}
}

func TestInjectOpenCode_SwitchGentlemanToNeutral_CleansAgentOverlay(t *testing.T) {
	home := t.TempDir()

	// Step 1: install gentleman — agent.gentleman key must appear in opencode.json.
	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	settingsRaw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("precondition: opencode.json must exist after gentleman install: %v", err)
	}
	var before map[string]any
	if err := json.Unmarshal(settingsRaw, &before); err != nil {
		t.Fatalf("precondition: unmarshal opencode.json: %v", err)
	}
	agentBefore, ok := before["agent"].(map[string]any)
	if !ok {
		t.Fatal("precondition: 'agent' key must be present after gentleman install")
	}
	if _, ok := agentBefore["gentleman"]; !ok {
		t.Fatal("precondition: agent.gentleman must be present after gentleman install")
	}

	// Pre-populate a user-defined agent to verify it survives the cleanup.
	agentBefore["my-custom-agent"] = map[string]any{"mode": "secondary"}
	before["agent"] = agentBefore
	before["someUserKey"] = "preserved"
	encoded, _ := json.MarshalIndent(before, "", "  ")
	if err := os.WriteFile(settingsPath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile() setup error = %v", err)
	}

	// Step 2: switch to neutral — agent.gentleman must be removed.
	result, err := Inject(home, opencodeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(neutral) should report changed when cleaning agent.gentleman residual")
	}

	settingsRaw, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) after neutral: %v", err)
	}
	var after map[string]any
	if err := json.Unmarshal(settingsRaw, &after); err != nil {
		t.Fatalf("Unmarshal opencode.json after neutral: %v", err)
	}

	// agent.gentleman must be gone.
	if agentAfter, ok := after["agent"].(map[string]any); ok {
		if _, stillPresent := agentAfter["gentleman"]; stillPresent {
			t.Fatal("agent.gentleman must be removed from opencode.json after switching to neutral")
		}
		// User-defined agent must survive.
		if _, ok := agentAfter["my-custom-agent"]; !ok {
			t.Fatal("user-defined agent 'my-custom-agent' was removed — only agent.gentleman should be cleaned")
		}
	}

	// Other top-level user keys must survive.
	if after["someUserKey"] != "preserved" {
		t.Fatalf("user key 'someUserKey' was lost: got %v", after["someUserKey"])
	}
}

func TestInjectOpenCode_NeutralFresh_IsNoOp(t *testing.T) {
	home := t.TempDir()

	_, err := Inject(home, opencodeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) on fresh install error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if _, statErr := os.Stat(settingsPath); !os.IsNotExist(statErr) {
		data, _ := os.ReadFile(settingsPath)
		if strings.Contains(string(data), `"gentleman"`) {
			t.Fatal("Neutral fresh install must not create gentleman agent key")
		}
	}
}

func TestInjectOpenCode_GentlemanOnly_WritesAgentOverlay(t *testing.T) {
	home := t.TempDir()

	_, err := Inject(home, opencodeAdapter(), model.PersonaGentleman)
	if err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}
	if !strings.Contains(string(data), `"gentleman"`) {
		t.Fatal("Gentleman install must write gentleman agent overlay in opencode.json")
	}
}

func TestInjectOpenCode_MalformedJSON_DoesNotPanic(t *testing.T) {
	home := t.TempDir()

	settingsDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	malformed := `{ "agent": { "gentleman": {invalid json`
	if err := os.WriteFile(filepath.Join(settingsDir, "opencode.json"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, opencodeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) with malformed JSON must not error, got: %v", err)
	}
}

func TestInjectClaude_MalformedJSON_DoesNotPanic(t *testing.T) {
	home := t.TempDir()

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	malformed := `{ "outputStyle": "Gentleman", invalid`
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, claudeAdapter(), model.PersonaNeutral)
	if err != nil {
		t.Fatalf("Inject(neutral) with malformed settings.json must not error, got: %v", err)
	}
}

func TestInjectForSync_OpenCodeNeutral_CleansAgentGentleman(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, opencodeAdapter(), model.PersonaGentleman); err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) after install error = %v", err)
	}
	if !strings.Contains(string(before), `"gentleman"`) {
		t.Fatalf("opencode.json missing gentleman agent after install; got:\n%s", string(before))
	}

	if _, err := InjectForSync(home, opencodeAdapter(), model.PersonaNeutral); err != nil {
		t.Fatalf("InjectForSync(neutral) error = %v", err)
	}

	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) after sync error = %v", err)
	}
	if strings.Contains(string(after), `"gentleman"`) {
		t.Fatalf("opencode.json still has gentleman agent after InjectForSync(neutral); got:\n%s", string(after))
	}
}

func TestInjectForSync_ClaudeGentlemanToNeutral_CleansOutputStyle(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, claudeAdapter(), model.PersonaGentleman); err != nil {
		t.Fatalf("Inject(gentleman) error = %v", err)
	}

	stylePath := filepath.Join(home, ".claude", "output-styles", "gentleman.md")
	if _, err := os.Stat(stylePath); os.IsNotExist(err) {
		t.Fatal("gentleman.md not written by Inject(gentleman) — precondition failed")
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", err)
	}
	if !strings.Contains(string(raw), `"outputStyle"`) {
		t.Fatal("settings.json missing outputStyle after install — precondition failed")
	}

	if _, err := InjectForSync(home, claudeAdapter(), model.PersonaNeutral); err != nil {
		t.Fatalf("InjectForSync(neutral) error = %v", err)
	}

	if _, err := os.Stat(stylePath); !os.IsNotExist(err) {
		t.Fatal("gentleman.md still present after InjectForSync(neutral) — residue not cleaned")
	}

	afterRaw, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadFile(settings.json) after sync error = %v", err)
	}
	if !strings.Contains(string(afterRaw), `"outputStyle": "Neutral"`) {
		t.Fatalf("settings.json should select Neutral outputStyle after InjectForSync(neutral); got:\n%s", string(afterRaw))
	}
}

// TestPersonaContentAliasRoutesToNeutral verifies that the gentleman-neutral-artifacts
// alias is routed to neutral content, not gentleman content.
func TestPersonaContentAliasRoutesToNeutral(t *testing.T) {
	alias := personaContent(model.AgentClaudeCode, model.PersonaGentlemanNeutralArtifacts, false)
	neutral := personaContent(model.AgentClaudeCode, model.PersonaNeutral, false)
	if alias != neutral {
		t.Fatal("gentleman-neutral-artifacts must produce identical content to neutral")
	}
	if alias == "" {
		t.Fatal("alias persona content must not be empty")
	}
}

// TestPersonaContentGentlemanResidualIgnoredForClaude pins the JD-020
// trap documented on personaContent: unlike PersonaNeutral, PersonaGentleman
// dispatch for Claude is agent-hardcoded, NOT driven by the residual
// argument, because claude/persona-gentleman.md was slimmed to the residual block IN PLACE (Decision 3) instead of split
// into separate full/residual files. Calling with residual=false still
// returns the SAME slim asset — there is no full-Gentleman fallback to serve.
// This is currently safe because residualChannel() evaluates true
// unconditionally for that adapter (claude/adapter.go), but if that ever became conditional, this call
// pattern would silently serve tone-free content to a caller expecting the
// full persona. This test locks TODAY's behavior so a future change to the
// dispatch logic is a deliberate, reviewed decision rather than an accident.
func TestPersonaContentGentlemanResidualIgnoredForClaude(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentClaudeCode} {
		t.Run(string(agent), func(t *testing.T) {
			withResidualFalse := personaContent(agent, model.PersonaGentleman, false)
			withResidualTrue := personaContent(agent, model.PersonaGentleman, true)

			if withResidualFalse != withResidualTrue {
				t.Fatalf("personaContent(%q, gentleman, residual=false) != residual=true — Gentleman dispatch for Claude is expected to be agent-hardcoded and ignore the residual flag entirely", agent)
			}
			if strings.Contains(withResidualFalse, "Senior Architect") {
				t.Fatalf("personaContent(%q, gentleman, residual=false) still contains tone content 'Senior Architect' — expected the slim residual asset even with residual=false (JD-020 trap)", agent)
			}
			if !strings.Contains(withResidualFalse, "## Rules") {
				t.Fatalf("personaContent(%q, gentleman, residual=false) missing residual '## Rules'", agent)
			}
		})
	}
}

// TestPersonaContentNonResidualNeutralUnchanged is a regression test verifying that
// non-residual agents still receive the byte-identical
// generic/persona-neutral.md when PersonaNeutral is selected. This ensures the
// refactor is additive-only. Claude Code is excluded: under this design its
// neutral personaContent becomes the residual asset, not the generic one.
func TestPersonaContentNonResidualNeutralUnchanged(t *testing.T) {
	genericNeutral := assets.MustRead("generic/persona-neutral.md")
	if genericNeutral == "" {
		t.Fatal("generic/persona-neutral.md asset is empty")
	}

	agentIDs := []model.AgentID{
		model.AgentOpenCode,
		model.AgentGeminiCLI,
		model.AgentCursor,
		model.AgentCodex,
	}
	for _, agent := range agentIDs {
		t.Run(string(agent), func(t *testing.T) {
			got := personaContent(agent, model.PersonaNeutral, false)
			if got != genericNeutral {
				t.Fatalf("personaContent(%q, neutral) is no longer byte-identical to generic/persona-neutral.md — regression", agent)
			}
		})
	}
}

// TestPersonaContentResidualDispatchAllAgents is a table-driven regression test
// covering every model.AgentID constant: Claude Code receives the residual
// (tone-free) persona asset when residual=true; all other agents receive the full persona section unaffected by the residual flag.
// Neutral's generic/persona-neutral.md remains the byte-identical fallback for
// every non-Claude agent.
func TestPersonaContentResidualDispatchAllAgents(t *testing.T) {
	allAgents := []model.AgentID{
		model.AgentClaudeCode,
		model.AgentOpenCode,
		model.AgentGeminiCLI,
		model.AgentCursor,
		model.AgentVSCodeCopilot,
		model.AgentCodex,
		model.AgentAntigravity,
		model.AgentWindsurf,
		model.AgentQwenCode,
		model.AgentKiroIDE,
		model.AgentPi,
		model.AgentTrae,
	}

	isResidualCapable := func(agent model.AgentID) bool {
		return agent == model.AgentClaudeCode
	}

	genericNeutral := assets.MustRead("generic/persona-neutral.md")

	for _, agent := range allAgents {
		t.Run(string(agent)+"/gentleman", func(t *testing.T) {
			residual := isResidualCapable(agent)
			content := personaContent(agent, model.PersonaGentleman, residual)
			if content == "" {
				t.Fatalf("personaContent(%q, gentleman, residual=%v) returned empty string", agent, residual)
			}
			if residual {
				if strings.Contains(content, "Senior Architect") {
					t.Fatalf("personaContent(%q, gentleman, residual=true) still contains tone content 'Senior Architect'", agent)
				}
				if !strings.Contains(content, "## Rules") {
					t.Fatalf("personaContent(%q, gentleman, residual=true) missing residual '## Rules'", agent)
				}
			} else if !strings.Contains(content, "Senior Architect") {
				t.Fatalf("personaContent(%q, gentleman, residual=false) missing full persona tone content 'Senior Architect'", agent)
			}
		})

		t.Run(string(agent)+"/neutral", func(t *testing.T) {
			residual := isResidualCapable(agent)
			content := personaContent(agent, model.PersonaNeutral, residual)
			if content == "" {
				t.Fatalf("personaContent(%q, neutral, residual=%v) returned empty string", agent, residual)
			}
			switch {
			case residual:
				if strings.Contains(content, "Senior Architect") {
					t.Fatalf("personaContent(%q, neutral, residual=true) still contains tone content 'Senior Architect'", agent)
				}
				if !strings.Contains(content, "## Rules") {
					t.Fatalf("personaContent(%q, neutral, residual=true) missing residual '## Rules'", agent)
				}
			default:
				if content != genericNeutral {
					t.Fatalf("personaContent(%q, neutral, residual=false) should be byte-identical to generic/persona-neutral.md", agent)
				}
			}
		})
	}
}

// TestResidualChannelAllAgents is a direct table test over
// residualChannel(adapter) using real adapter instances for all 16
// model.AgentID constants (JD-021): TestPersonaContentResidualDispatchAllAgents
// hand-derives the residual predicate via isResidualCapable() instead of
// exercising residualChannel() itself, so a bug in residualChannel would not
// be caught there. This test calls residualChannel() directly.
func TestResidualChannelAllAgents(t *testing.T) {
	allAgentIDs := []model.AgentID{
		model.AgentClaudeCode,
		model.AgentOpenCode,
		model.AgentGeminiCLI,
		model.AgentCursor,
		model.AgentVSCodeCopilot,
		model.AgentCodex,
		model.AgentAntigravity,
		model.AgentWindsurf,
		model.AgentQwenCode,
		model.AgentKiroIDE,
		model.AgentPi,
		model.AgentTrae,
	}

	for _, agentID := range allAgentIDs {
		t.Run(string(agentID), func(t *testing.T) {
			adapter, err := agents.NewAdapter(agentID)
			if err != nil {
				t.Fatalf("agents.NewAdapter(%q) error = %v", agentID, err)
			}

			want := agentID == model.AgentClaudeCode
			got := residualChannel(adapter)
			if got != want {
				t.Fatalf("residualChannel(%q) = %v, want %v", agentID, got, want)
			}
		})
	}
}

// movedPersonaRule pairs a frozen (verbatim, HEAD-captured) line from a
// MOVE-tagged persona section with the substring actually asserted against
// the reconciled output style. checkAgainst equals frozen for the common
// case (the line survived untouched). When the output style legitimately
// pre-existed with its own equivalent wording for that rule (not a JD-016/017
// content-loss bug), checkAgainst is a documented merged/reworded substring
// instead, with mergedNote explaining the mapping — the pattern
// established for the JD-013 merged English-reply bullet.
type movedPersonaRule struct {
	frozen       string
	checkAgainst string
	mergedNote   string
}

func assertMovedPersonaRules(t *testing.T, styleAsset string, rules []movedPersonaRule) {
	t.Helper()
	reconciled := assets.MustRead(styleAsset)
	for _, r := range rules {
		if !strings.Contains(reconciled, r.checkAgainst) {
			if r.mergedNote != "" {
				t.Fatalf("%s: merged form %q (for frozen HEAD rule %q; %s) not found in reconciled style", styleAsset, r.checkAgainst, r.frozen, r.mergedNote)
			}
			t.Fatalf("%s: lost MOVE-tagged persona rule %q (frozen verbatim from HEAD)", styleAsset, r.frozen)
		}
	}
}

// claudeGentlemanMovedRules freezes every normative line from HEAD
// claude/persona-gentleman.md's MOVE-tagged sections (Personality, Persona
// Scope, Language, Tone, Philosophy, Behavior — captured 2026-07-08, before
// the Decision 3 slim-in-place). Expertise/Rules/Contextual Skill Loading are
// NOT MOVE-tagged (they remain in the residual persona file itself) and are
// intentionally excluded.
var claudeGentlemanMovedRules = []movedPersonaRule{
	{frozen: "Senior Architect, 15+ years experience, GDE & MVP. Passionate teacher who genuinely wants people to learn and grow. Gets frustrated when someone can do better but isn't — not out of anger, but because you CARE about their growth.",
		checkAgainst: "Passionate teacher who genuinely wants people to learn and grow.",
		mergedNote:   "the output style's own pre-existing Personality section reworded the bio around this shared sentence"},
	{frozen: "The persona's Language, Tone, Speech Patterns, and Personality rules govern ONLY your reply text addressed to the user — what you SAY in chat.", checkAgainst: "The persona's Language, Tone, Speech Patterns, and Personality rules govern ONLY your reply text addressed to the user — what you SAY in chat."},
	{frozen: "They do NOT govern artifacts you produce for the task:", checkAgainst: "They do NOT govern artifacts you produce for the task:"},
	{frozen: "- Code, identifiers, function/variable names, comments", checkAgainst: "- Code, identifiers, function/variable names, comments"},
	{frozen: "- UI copy, labels, button text, error messages, accessibility strings", checkAgainst: "- UI copy, labels, button text, error messages, accessibility strings"},
	{frozen: "- Documentation, README files, commit messages, PR descriptions", checkAgainst: "- Documentation, README files, commit messages, PR descriptions"},
	{frozen: "- Any string literal inside source code", checkAgainst: "- Any string literal inside source code"},
	{frozen: "For those artifacts:", checkAgainst: "For those artifacts:"},
	{frozen: "- Default to English. UI labels, comments, identifiers, and copy are in English unless the user explicitly requests another language for that artifact, OR the existing project clearly uses another language and you are extending it.", checkAgainst: "- Default to English. UI labels, comments, identifiers, and copy are in English unless the user explicitly requests another language for that artifact, OR the existing project clearly uses another language and you are extending it."},
	{frozen: "- Never inject Rioplatense slang, voseo, or persona stylistic emphasis (CAPS, exclamations, rhetorical questions) into generated code, UI strings, or any task artifact.", checkAgainst: "- Never inject Rioplatense slang, voseo, or persona stylistic emphasis (CAPS, exclamations, rhetorical questions) into generated code, UI strings, or any task artifact."},
	{frozen: "- The persona styles HOW YOU TALK, not WHAT YOU BUILD.", checkAgainst: "- The persona styles HOW YOU TALK, not WHAT YOU BUILD."},
	{frozen: "- Generated technical artifacts default to English regardless of the active persona or conversation language.", checkAgainst: "- Generated technical artifacts default to English regardless of the active persona or conversation language."},
	{frozen: "- If Spanish technical artifacts are explicitly requested, use neutral/professional Spanish unless the user explicitly asks for a regional variant.", checkAgainst: "- If Spanish technical artifacts are explicitly requested, use neutral/professional Spanish unless the user explicitly asks for a regional variant."},
	{frozen: "- Public/contextual comments follow the target context language by default; Spanish comments default to neutral/professional Spanish unless the user or context clearly calls for regional tone.", checkAgainst: "- Public/contextual comments follow the target context language by default; Spanish comments default to neutral/professional Spanish unless the user or context clearly calls for regional tone."},
	{frozen: "- Match the user's current language in your REPLY ONLY (see Persona Scope above).",
		checkAgainst: "Always match the user's current language in your reply.",
		mergedNote:   "JD-019: the persona's 'REPLY ONLY' phrasing was folded into the style's own Language Rules opener; this is the exact combined-channel phrase"},
	{frozen: "- Determine the reply language from the latest actual user request, not from Engram or memory context, repository/project language, tool output, previous assistant turns, examples, or persona momentum.",
		checkAgainst: "Determine the reply language from the latest actual user request, not from Engram or memory context, repository/project language, tool output, previous assistant turns, persona wording, examples, or stylistic momentum.",
		mergedNote:   "style expanded the exclusion list ('persona wording'/'stylistic momentum' vs 'persona momentum') without dropping the priority-ordering rule"},
	{frozen: "- For mixed-language prompts, use the dominant language of the user's direct request. Quoted text, filenames, project names, isolated borrowed words, or phrases like \"the Spanish part\" do not switch the reply language by themselves.", checkAgainst: "- For mixed-language prompts, use the dominant language of the user's direct request. Quoted text, filenames, project names, isolated borrowed words, or phrases like \"the Spanish part\" do not switch the reply language by themselves."},
	{frozen: "- Do not switch languages unless the user does, asks you to, or you are quoting/translating content.", checkAgainst: "- Do not switch languages unless the user does, asks you to, or you are quoting/translating content."},
	{frozen: "- When replying to the user in Spanish, use warm natural Rioplatense Spanish (voseo) without overloading the reply with slang.", checkAgainst: "- When replying to the user in Spanish, use warm natural Rioplatense Spanish (voseo) without overloading the reply with slang."},
	{frozen: "- When replying to the user in English, keep the full reply in natural English with the same warm energy.",
		checkAgainst: "keep the full reply in natural English with the same warm energy",
		mergedNote:   "JD-013/Decision 4: merged with the style's own near-duplicate English-reply bullet into one canonical sentence"},
	{frozen: "- If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence. Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments.", checkAgainst: "- If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence. Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments."},
	{frozen: "- Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language.", checkAgainst: "- Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language."},
	{frozen: "Passionate and direct, but from a place of CARING. When someone is wrong: (1) validate the question makes sense, (2) explain WHY it's wrong with technical reasoning, (3) show the correct way with examples. Frustration comes from caring they can do better. Use CAPS for emphasis.",
		checkAgainst: "Passionate and direct, but from a place of CARING.",
		mergedNote:   "style's own pre-existing ## Tone section shares this opening sentence but reworks the rest for the Gentleman voice; the CARING framing survives verbatim"},
	{frozen: "- CONCEPTS > CODE: call out people who code without understanding fundamentals",
		checkAgainst: "CONCEPTS > CODE:",
		mergedNote:   "style's own ## Philosophy restates this pillar as a first-person quote; the pillar label is the shared anchor"},
	{frozen: "- AI IS A TOOL: we direct, AI executes; the human always leads",
		checkAgainst: "AI IS A TOOL:",
		mergedNote:   "style's own ## Philosophy restates this pillar as a first-person quote; the pillar label is the shared anchor"},
	{frozen: "- SOLID FOUNDATIONS: design patterns, architecture, bundlers before frameworks",
		checkAgainst: "FOUNDATIONS FIRST:",
		mergedNote:   "style renamed this pillar from 'SOLID FOUNDATIONS' to 'FOUNDATIONS FIRST' with a concrete DOM/React example; the fundamentals-before-frameworks theme is preserved under a different label"},
	{frozen: "- AGAINST IMMEDIACY: no shortcuts; real learning takes effort and time",
		checkAgainst: "AGAINST IMMEDIACY:",
		mergedNote:   "style's own ## Philosophy restates this pillar as a first-person quote; the pillar label is the shared anchor"},
	{frozen: "- Push back when user asks for code without context or understanding",
		checkAgainst: "code without context",
		mergedNote:   "style's own numbered ## Behavior item 2 already covered this rule under different phrasing before reconciliation"},
	// JD-017: this exact bullet was dropped entirely from ALL current claude
	// assets (rg -i analog = zero hits) — verbatim check, no merged form.
	{frozen: "- Use construction/architecture analogies when they clarify the point, not by default", checkAgainst: "Use construction/architecture analogies when they clarify the point, not by default"},
	{frozen: "- Correct errors ruthlessly but explain WHY technically",
		checkAgainst: "Correct errors",
		mergedNote:   "style's own ## Behavior item 4 ('Correct errors but always explain the technical WHY') already covers this rule; 'ruthlessly' softened but the correct-with-WHY core survives"},
	{frozen: "- For concepts: (1) explain problem, (2) propose solution, (3) mention examples or tools only when they materially help",
		checkAgainst: "examples or tools only when they materially help",
		mergedNote:   "style's own ## Behavior item 5 restates the same 3-step method almost verbatim ('add' vs 'mention'); the materiality qualifier is the exact shared tail"},
}

// neutralMovedRules freezes every normative line from HEAD
// generic/persona-neutral.md's MOVE-tagged sections. Persona Scope's four
// itemized artifact-type bullets ("Code, identifiers...", "UI copy...",
// "Documentation...", "Any string literal...") and the
// "Never inject regional slang...into generated code" bullet are
// intentionally NOT included here: they were already condensed into two
// prose paragraphs in output-style-neutral.md before this change (not part
// of the JD-016 Behavior/Tone gap this test targets), and forcing a strict
// per-bullet mapping for a pre-existing, unflagged restructuring would
// falsely fail this test on content outside JD-016/017's scope.
var neutralMovedRules = []movedPersonaRule{
	{frozen: "- The persona styles HOW YOU TALK, not WHAT YOU BUILD.", checkAgainst: "- The persona styles HOW YOU TALK, not WHAT YOU BUILD."},
	{frozen: "Senior Architect, 15+ years experience, GDE & MVP. Passionate teacher who genuinely wants people to learn and grow. Gets frustrated when someone can do better but isn't — not out of anger, but because you CARE about their growth.",
		checkAgainst: "senior mentor",
		mergedNote:   "Neutral's output style deliberately does not restate Gentleman-flavor bio details ('Senior Architect', 'GDE & MVP'); the mentor identity itself survives via the Core Principle's 'You are a senior mentor' framing"},
	{frozen: "The persona's Language, Tone, Speech Patterns, and Personality rules govern ONLY your reply text addressed to the user — what you SAY in chat.",
		checkAgainst: "This output style governs direct replies to the user only.",
		mergedNote:   "style's own Persona Scope section restates the scope-limiting rule as one declarative sentence"},
	{frozen: "They do NOT govern artifacts you produce for the task:",
		checkAgainst: "It does not define the language, tone, or style of generated artifacts.",
		mergedNote:   "same paragraph, condensed opener establishing the same replies-only/not-artifacts boundary"},
	{frozen: "- Default to English. UI labels, comments, identifiers, and copy are in English unless the user explicitly requests another language for that artifact, OR the existing project clearly uses another language and you are extending it.",
		checkAgainst: "Generated technical artifacts default to English and neutral professional wording unless the user explicitly requests another artifact language or the existing project convention requires it.",
		mergedNote:   "condensed restatement of the same default-to-English artifact rule"},
	{frozen: "- Generated technical artifacts default to English regardless of the active persona or conversation language.", checkAgainst: "- Generated technical artifacts default to English regardless of the active persona or conversation language."},
	{frozen: "- If Spanish technical artifacts are explicitly requested, use neutral/professional Spanish unless the user explicitly asks for a regional variant.", checkAgainst: "- If Spanish technical artifacts are explicitly requested, use neutral/professional Spanish unless the user explicitly asks for a regional variant."},
	{frozen: "- Public/contextual comments follow the target context language by default; Spanish comments default to neutral/professional Spanish unless the user or context clearly calls for regional tone.", checkAgainst: "- Public/contextual comments follow the target context language by default; Spanish comments default to neutral/professional Spanish unless the user or context clearly calls for regional tone."},
	{frozen: "- Match the user's current language in your REPLY ONLY (see Persona Scope above).",
		checkAgainst: "Match the user's current language in direct replies.",
		mergedNote:   "the neutral style's own Language and Tone opener carries the same rule without the Gentleman-specific 'REPLY ONLY' cross-reference"},
	{frozen: "- Do not switch languages unless the user does, asks you to, or you are quoting/translating content.", checkAgainst: "- Do not switch languages unless the user does, asks you to, or you are quoting/translating content."},
	{frozen: "- Use warm, natural, professional language without regional slang or dialect-specific grammar.",
		checkAgainst: "Use warm, natural, professional wording without regional slang or dialect-specific grammar.",
		mergedNote:   "'language' -> 'wording' rewording, otherwise verbatim"},
	{frozen: "- When replying to the user in English, keep the full reply in natural English with the same warm energy.",
		checkAgainst: "When replying to the user in English, keep the full",
		mergedNote:   "style's own bullet continues differently ('...response in English unless...') but shares this opening normative clause"},
	{frozen: "- If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence. Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments.", checkAgainst: "- If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence. Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments."},
	{frozen: "- Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language.", checkAgainst: "- Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language."},
	// JD-016: this entire paragraph was marked MOVE in Table C but never
	// added to output-style-neutral.md — verbatim check, no merged form.
	{frozen: "Passionate and direct, but from a place of CARING. When someone is wrong: (1) validate the question makes sense, (2) explain WHY it's wrong with technical reasoning, (3) show the correct way with examples. Frustration comes from caring they can do better. Use CAPS for emphasis.",
		checkAgainst: "Passionate and direct, but from a place of CARING. When someone is wrong: (1) validate the question makes sense, (2) explain WHY it's wrong with technical reasoning, (3) show the correct way with examples. Frustration comes from caring they can do better. Use CAPS for emphasis."},
	{frozen: "- CONCEPTS > CODE: call out people who code without understanding fundamentals",
		checkAgainst: "CONCEPTS > CODE:",
		mergedNote:   "style's own ## Teaching Behavior restates this pillar with different verbs under the same label"},
	{frozen: "- AI IS A TOOL: we direct, AI executes; the human always leads",
		checkAgainst: "AI IS A TOOL:",
		mergedNote:   "style's own ## Teaching Behavior restates this pillar with different verbs under the same label"},
	{frozen: "- SOLID FOUNDATIONS: design patterns, architecture, bundlers before frameworks",
		checkAgainst: "SOLID FOUNDATIONS:",
		mergedNote:   "style's own ## Teaching Behavior restates this pillar with different verbs under the same label"},
	{frozen: "- AGAINST IMMEDIACY: no shortcuts; real learning takes effort and time",
		checkAgainst: "AGAINST IMMEDIACY:",
		mergedNote:   "style's own ## Teaching Behavior restates this pillar with different verbs under the same label"},
	// JD-016: the entire ## Behavior section (4 rules) was marked MOVE in
	// Table C but never added to output-style-neutral.md — verbatim checks,
	// no merged form.
	{frozen: "- Push back when user asks for code without context or understanding", checkAgainst: "- Push back when user asks for code without context or understanding"},
	{frozen: "- Use construction/architecture analogies when they clarify the point, not by default", checkAgainst: "- Use construction/architecture analogies when they clarify the point, not by default"},
	{frozen: "- Correct errors ruthlessly but explain WHY technically", checkAgainst: "- Correct errors ruthlessly but explain WHY technically"},
	{frozen: "- For concepts: (1) explain problem, (2) propose solution, (3) mention examples or tools only when they materially help", checkAgainst: "- For concepts: (1) explain problem, (2) propose solution, (3) mention examples or tools only when they materially help"},
}

// TestReconciledStylesCarryAllMovedPersonaRules is the JD-018 guard against
// content that was supposed to MOVE from a persona.md section into the output style but never arrived — exactly the
// JD-016/017 class of bug, where the union completion silently dropped
// rules. This test freezes every normative rule from the HEAD (pre-apply)
// MOVE-tagged persona sections and asserts each is still discoverable
// (verbatim or via a documented merged form) in the corresponding
// reconciled output style.
func TestReconciledStylesCarryAllMovedPersonaRules(t *testing.T) {
	t.Run("claude-gentleman", func(t *testing.T) {
		assertMovedPersonaRules(t, "claude/output-style-gentleman.md", claudeGentlemanMovedRules)
	})
	t.Run("claude-neutral", func(t *testing.T) {
		assertMovedPersonaRules(t, "claude/output-style-neutral.md", neutralMovedRules)
	})
}

func TestWrapSteeringFileAddsKiroFrontmatter(t *testing.T) {
	got := wrapSteeringFile("## Persona\n\nBody")

	for _, want := range []string{
		"---\n",
		"inclusion: always",
		"---\n\n## Persona",
		"Body",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrapSteeringFile() missing %q; got:\n%s", want, got)
		}
	}
}

func TestMergeJSONFileToleratingMalformed(t *testing.T) {
	home := t.TempDir()

	t.Run("merges valid json", func(t *testing.T) {
		path := filepath.Join(home, "valid.json")
		if err := os.WriteFile(path, []byte(`{"permissions":{"allow":["Read"]}}`), 0o644); err != nil {
			t.Fatalf("WriteFile(valid): %v", err)
		}

		result, err := mergeJSONFileToleratingMalformed(path, []byte(`{"outputStyle":"Neutral"}`))
		if err != nil {
			t.Fatalf("mergeJSONFileToleratingMalformed(valid) error = %v", err)
		}
		if !result.Changed {
			t.Fatal("mergeJSONFileToleratingMalformed(valid) changed = false")
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(valid): %v", err)
		}
		text := string(raw)
		if !strings.Contains(text, `"outputStyle": "Neutral"`) {
			t.Fatalf("merged JSON missing outputStyle; got:\n%s", text)
		}
		if !strings.Contains(text, `"permissions"`) {
			t.Fatalf("merged JSON lost existing permissions; got:\n%s", text)
		}
	})

	t.Run("ignores malformed overlay to avoid data loss", func(t *testing.T) {
		path := filepath.Join(home, "malformed-overlay.json")
		original := `{"outputStyle":"Gentleman"}`
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatalf("WriteFile(malformed overlay): %v", err)
		}

		result, err := mergeJSONFileToleratingMalformed(path, []byte(`{"outputStyle":"Neutral"`))
		if err != nil {
			t.Fatalf("mergeJSONFileToleratingMalformed(malformed overlay) error = %v", err)
		}
		if result.Changed {
			t.Fatal("mergeJSONFileToleratingMalformed(malformed overlay) changed = true")
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(malformed overlay): %v", err)
		}
		if string(raw) != original {
			t.Fatalf("JSON was modified after malformed overlay; got %q, want %q", string(raw), original)
		}
	})

	t.Run("returns non-json read errors", func(t *testing.T) {
		originalReadFile := osReadFile
		t.Cleanup(func() { osReadFile = originalReadFile })
		osReadFile = func(string) ([]byte, error) {
			return nil, fmt.Errorf("permission denied")
		}

		if _, err := mergeJSONFileToleratingMalformed(filepath.Join(home, "denied.json"), []byte(`{}`)); err == nil {
			t.Fatal("mergeJSONFileToleratingMalformed(non-json error) error = nil")
		}
	})
}

func TestRemoveJSONKeyIfValueScenarios(t *testing.T) {
	home := t.TempDir()

	t.Run("removes matching managed value and preserves siblings", func(t *testing.T) {
		path := filepath.Join(home, "matching.json")
		if err := os.WriteFile(path, []byte(`{"outputStyle":"Gentleman","theme":"dark"}`), 0o644); err != nil {
			t.Fatalf("WriteFile(matching): %v", err)
		}

		removed, err := removeJSONKeyIfValue(path, "outputStyle", "Gentleman")
		if err != nil {
			t.Fatalf("removeJSONKeyIfValue(matching) error = %v", err)
		}
		if !removed {
			t.Fatal("removeJSONKeyIfValue(matching) removed = false")
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(matching): %v", err)
		}
		text := string(raw)
		if strings.Contains(text, "outputStyle") {
			t.Fatalf("outputStyle was not removed; got:\n%s", text)
		}
		if !strings.Contains(text, `"theme": "dark"`) {
			t.Fatalf("sibling key was not preserved; got:\n%s", text)
		}
	})

	t.Run("preserves user value", func(t *testing.T) {
		path := filepath.Join(home, "custom.json")
		original := `{"outputStyle":"MyCustom","theme":"dark"}`
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatalf("WriteFile(custom): %v", err)
		}

		removed, err := removeJSONKeyIfValue(path, "outputStyle", "Gentleman")
		if err != nil {
			t.Fatalf("removeJSONKeyIfValue(custom) error = %v", err)
		}
		if removed {
			t.Fatal("removeJSONKeyIfValue(custom) removed = true")
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(custom): %v", err)
		}
		if string(raw) != original {
			t.Fatalf("custom JSON was modified; got %q, want %q", string(raw), original)
		}
	})

	t.Run("ignores malformed json", func(t *testing.T) {
		path := filepath.Join(home, "malformed-cleanup.json")
		original := `{"outputStyle":"Gentleman", invalid`
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatalf("WriteFile(malformed): %v", err)
		}

		removed, err := removeJSONKeyIfValue(path, "outputStyle", "Gentleman")
		if err != nil {
			t.Fatalf("removeJSONKeyIfValue(malformed) error = %v", err)
		}
		if removed {
			t.Fatal("removeJSONKeyIfValue(malformed) removed = true")
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(malformed): %v", err)
		}
		if string(raw) != original {
			t.Fatalf("malformed JSON was modified; got %q, want %q", string(raw), original)
		}
	})

	t.Run("propagates read errors", func(t *testing.T) {
		originalReadFile := osReadFile
		t.Cleanup(func() { osReadFile = originalReadFile })
		osReadFile = func(string) ([]byte, error) {
			return nil, fmt.Errorf("read failed")
		}

		if _, err := removeJSONKeyIfValue(filepath.Join(home, "denied-cleanup.json"), "outputStyle", "Gentleman"); err == nil {
			t.Fatal("removeJSONKeyIfValue(read error) error = nil")
		}
	})
}

// En un proyecto de VS Code (adaptador ligado a la raíz), .github/copilot-instructions.md
// es del equipo: no se retira como resto de una instalación antigua ni siquiera si su
// texto se parece a la persona.
func TestInjectVSCodeWorkspaceKeepsTeamCopilotInstructions(t *testing.T) {
	root := t.TempDir()
	teamPath := filepath.Join(root, ".github", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(teamPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	teamContent := "## Personality\n\nSenior Architect, texto del equipo.\n"
	if err := os.WriteFile(teamPath, []byte(teamContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	global, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}
	adapter := agents.BindWorkspace(global, root)

	if _, err := Inject(root, adapter, model.PersonaGentleman); err != nil {
		t.Fatalf("Inject(vscode, workspace) error = %v", err)
	}

	if got, err := os.ReadFile(teamPath); err != nil || string(got) != teamContent {
		t.Fatalf("copilot-instructions.md del equipo cambió (err=%v):\n%s", err, got)
	}
	instructions, err := os.ReadFile(filepath.Join(root, ".github", "instructions", "axiom.instructions.md"))
	if err != nil || !strings.Contains(string(instructions), "applyTo: \"**\"") {
		t.Fatalf("falta .github/instructions/axiom.instructions.md con applyTo (err=%v)", err)
	}
}
