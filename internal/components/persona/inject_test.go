package persona

import (
	"encoding/json"
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

// claudeOutputStyleLanguageGuardrails son las reglas de idioma de la respuesta
// directa que el output style de Axiom debe conservar.
var claudeOutputStyleLanguageGuardrails = []string{
	"Determine the reply language from the latest actual user request",
	"If the selected reply language is English, every part of the direct reply must be English: greetings, interjections, acknowledgements, transition phrases, and the first sentence.",
	"Do not use Hola, dale, listo, Spanish punctuation, or other Spanish fragments.",
	"Prompts starting with or dominated by hi, hello, hey, or similar English greetings are English prompts unless the user explicitly asks for another language.",
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

func TestInjectClaudeWritesResidualPersonaSection(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter())
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

func TestInjectClaudeWritesOutputStyleFile(t *testing.T) {
	home := t.TempDir()

	_, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Verify output-style file was written.
	stylePath := filepath.Join(home, ".claude", "output-styles", "axiom.md")
	content, err := os.ReadFile(stylePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", stylePath, err)
	}

	text := string(content)
	if !strings.Contains(text, "name: Axiom") {
		t.Fatal("Output style file missing YAML frontmatter 'name: Axiom'")
	}
	if !strings.Contains(text, "keep-coding-instructions: true") {
		t.Fatal("Output style file missing 'keep-coding-instructions: true'")
	}
	if !strings.Contains(text, "Axiom Output Style") {
		t.Fatal("Output style file missing 'Axiom Output Style' heading")
	}
	assertLanguageGuardrails(t, text, claudeOutputStyleLanguageGuardrails, nil)
}

func TestInjectClaudeMergesOutputStyleIntoSettings(t *testing.T) {
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

	_, err := Inject(home, claudeAdapter())
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
	if outputStyle != "Axiom" {
		t.Fatalf("settings.json outputStyle = %q, want %q", outputStyle, "Axiom")
	}

	// Verify existing keys were preserved.
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("settings.json lost 'permissions' key during merge")
	}
	if _, ok := settings["syntaxHighlightingDisabled"]; !ok {
		t.Fatal("settings.json lost 'syntaxHighlightingDisabled' key during merge")
	}
}

func TestInjectClaudeReturnsAllFiles(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Should return 3 files: CLAUDE.md, output-style, settings.json.
	if len(result.Files) != 3 {
		t.Fatalf("Inject() returned %d files, want 3: %v", len(result.Files), result.Files)
	}

	wantSuffixes := []string{"CLAUDE.md", "axiom.md", "settings.json"}
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

func TestInjectPiPersonaWritesSelectedModeAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path := PiPersonaConfigPath(root)

	result, err := InjectPiPersona(root)
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
	if got, want := string(content), "{\n  \"mode\": \"axiom\"\n}\n"; got != want {
		t.Fatalf("Pi persona config = %q, want %q", got, want)
	}

	second, err := InjectPiPersona(root)
	if err != nil {
		t.Fatalf("second InjectPiPersona() error = %v", err)
	}
	if second.Changed {
		t.Fatalf("second Pi persona injection changed = true, want false")
	}
}

func TestInjectOpenCodeWritesAgentsFile(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, opencodeAdapter())
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
	if !strings.Contains(text, "## Personality") {
		t.Fatal("AGENTS.md missing real persona content")
	}
	if !containsPersonaMarker(text) {
		t.Fatal("AGENTS.md missing persona marker")
	}
}

func TestInjectAntigravityWritesMarkedPersonaSection(t *testing.T) {
	home := t.TempDir()
	promptPath := filepath.Join(home, ".gemini", "GEMINI.md")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("# User Gemini rules\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := Inject(home, antigravityAdapter())
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
	if !containsPersonaMarker(text) || !containsPersonaCloseMarker(text) || !strings.Contains(text, "# User Gemini rules") || !strings.Contains(text, "## Personality") {
		t.Fatalf("GEMINI.md missing required content; got:\n%s", text)
	}

	second, err := Inject(home, antigravityAdapter())
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

func TestInjectOpenCodeDoesNotCreateSDDConductorNorPersonaAgent(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, opencodeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// La persona de Axiom no necesita opencode.json: sin él no hay nada que limpiar.
			return
		}
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}
	text := string(content)
	for _, forbidden := range []string{`"sdd-orchestrator"`, `"gentle-orchestrator"`, `"gentleman"`, `"tools"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("persona injection must not write %s into opencode.json: %s", forbidden, text)
		}
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

	_, err := Inject(home, opencodeAdapter())
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

	_, err := Inject(home, opencodeAdapter())
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

	_, err := Inject(home, opencodeAdapter())
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

	_, err := Inject(home, opencodeAdapter())
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

func TestInjectOpenCodePreservesManagedSectionsOnReinject(t *testing.T) {
	home := t.TempDir()

	// First install the persona + simulate SDD/engram sections
	_, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
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

	// Now re-inject the persona
	result, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject() should report changed")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after re-inject error = %v", err)
	}
	text := string(content)

	// Persona content should be present
	if !strings.Contains(text, "## Personality") {
		t.Fatal("AGENTS.md missing persona content")
	}
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("AGENTS.md has Rioplatense language in the Axiom persona")
	}

	// Managed sections MUST be preserved
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("AGENTS.md lost SDD orchestrator section after re-injecting the persona")
	}
	if !strings.Contains(text, "<!-- gentle-ai:engram-protocol -->") {
		t.Fatal("AGENTS.md lost engram protocol section after re-injecting the persona")
	}

	// Regional language must not appear
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("AGENTS.md still has Rioplatense language after re-inject")
	}
}

func TestInjectForSyncCleansOnlyGentlemanAgent(t *testing.T) {
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

			result, err := InjectForSync(home, tc.adapter)
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

func TestInjectForSyncToleratesMalformedOpenCodeSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	malformed := []byte(`{"agent":`)
	if err := os.WriteFile(settingsPath, malformed, 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	if _, err := InjectForSync(home, opencodeAdapter()); err != nil {
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

func TestInjectVSCodePreservesManagedSectionsOnReinject(t *testing.T) {
	home := t.TempDir()

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	_, err = Inject(home, vscodeAdapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
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

	_, err = Inject(home, vscodeAdapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after re-inject error = %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "## Personality") {
		t.Fatal("instructions file missing persona content")
	}
	if strings.Contains(text, "Rioplatense") {
		t.Fatal("instructions file has Rioplatense language in the Axiom persona")
	}
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("instructions file lost SDD section after re-injecting the persona")
	}
	if !strings.Contains(text, "---\nname:") {
		t.Fatal("instructions file lost YAML frontmatter")
	}
}

func TestInjectPreservesWhenMarkerAtByteZero(t *testing.T) {
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

	_, err = Inject(home, opencodeAdapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "## Personality") {
		t.Fatal("missing persona content")
	}
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->") {
		t.Fatal("SDD section destroyed when marker was at byte 0")
	}
}

func TestInjectClaudeIsIdempotent(t *testing.T) {
	home := t.TempDir()

	first, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}

	second, err := Inject(home, claudeAdapter())
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
	if err := os.WriteFile(opencodeAdapter().SettingsPath(home), []byte(`{"theme":"dark","agent":{"gentleman":{"mode":"primary"},"user":{"tools":{"read":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := Inject(home, opencodeAdapter())
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
	if _, exists := agents["gentleman"]; exists || root["theme"] != "dark" || agents["user"].(map[string]any)["tools"].(map[string]any)["read"] != true {
		t.Fatalf("retiring agent.gentleman changed user settings: %#v", root)
	}

	second, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true")
	}
}

func TestInjectCursorWritesRulesFileWithRealContent(t *testing.T) {
	home := t.TempDir()

	cursorAdapter, err := agents.NewAdapter("cursor")
	if err != nil {
		t.Fatalf("NewAdapter(cursor) error = %v", err)
	}

	result, injectErr := Inject(home, cursorAdapter)
	if injectErr != nil {
		t.Fatalf("Inject(cursor) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatalf("Inject(cursor) changed = false")
	}

	// Verify the generic persona content was used.
	path := filepath.Join(home, ".cursor", "rules", "axiom.mdc")
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, readErr)
	}

	text := string(content)
	if !strings.Contains(text, "## Personality") {
		t.Fatal("Cursor persona missing '## Personality' — not the generic persona")
	}
	if !strings.Contains(text, "Contextual Skill Loading") {
		t.Fatal("Cursor persona missing contextual skill loading directive")
	}
}

func TestInjectGeminiWritesSystemPromptWithRealContent(t *testing.T) {
	home := t.TempDir()

	geminiAdapter, err := agents.NewAdapter("gemini-cli")
	if err != nil {
		t.Fatalf("NewAdapter(gemini-cli) error = %v", err)
	}

	result, injectErr := Inject(home, geminiAdapter)
	if injectErr != nil {
		t.Fatalf("Inject(gemini) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatal("Inject(gemini) changed = false")
	}

	path := filepath.Join(home, ".gemini", "GEMINI.md")
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, readErr)
	}

	text := string(content)
	if !strings.Contains(text, "## Personality") {
		t.Fatal("Gemini persona missing '## Personality'")
	}
	assertLanguageGuardrails(t, text,
		[]string{
			"Match the user's current language in your conversational replies.",
			"Do not switch languages unless the user does, asks you to, or you are quoting/translating content.",
			"When replying to the user in English, maintain a direct, professional, natural English with technical precision.",
		},
		[]string{
			`Say "déjame verificar"`,
			"Spanish input → Rioplatense Spanish",
			"English input → same warm energy",
		},
	)
}

func TestInjectVSCodeWritesInstructionsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}

	result, injectErr := Inject(home, vscodeAdapter)
	if injectErr != nil {
		t.Fatalf("Inject(vscode) error = %v", injectErr)
	}

	if !result.Changed {
		t.Fatal("Inject(vscode) changed = false")
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
	if !strings.Contains(text, "## Personality") {
		t.Fatal("VS Code persona missing '## Personality'")
	}
}

func TestInjectClaudePreservesNonPersonaContent(t *testing.T) {
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

	result, err := Inject(home, claudeAdapter())
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

func TestInjectVSCodeLeavesGitHubCopilotFileAlone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Plant a .github/copilot-instructions.md that has user content (not a
	// persona) — it must NOT be deleted.
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

	_, injectErr := Inject(home, vscodeAdapter)
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

func TestInjectClaude_SelectsManagedOutputStyleAndPreservesOtherSettings(t *testing.T) {
	home := t.TempDir()

	// Pre-create settings.json with a user-defined outputStyle that is NOT "Axiom".
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	userSettings := `{"outputStyle": "MyCustom", "syntaxHighlightingDisabled": true}`
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(userSettings), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsRaw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsRaw, &settings); err != nil {
		t.Fatalf("Unmarshal settings.json error = %v", err)
	}

	if got, want := settings["outputStyle"], "Axiom"; got != want {
		t.Fatalf("outputStyle = %v, want %q", got, want)
	}
	// Other user keys must also survive.
	if settings["syntaxHighlightingDisabled"] != true {
		t.Fatal("syntaxHighlightingDisabled was lost")
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

	_, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() with malformed JSON must not error, got: %v", err)
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

	_, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() with malformed settings.json must not error, got: %v", err)
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
		model.AgentQwenCode,
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

	if _, err := Inject(root, adapter); err != nil {
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

// TestPersonaContentResidualDispatchAllAgents recorre todos los agentes: Claude
// Code recibe el residual de la persona de Axiom cuando hay canal de output
// style; OpenCode, su asset propio; el resto, el genérico.
func TestPersonaContentResidualDispatchAllAgents(t *testing.T) {
	allAgents := []model.AgentID{
		model.AgentClaudeCode,
		model.AgentOpenCode,
		model.AgentGeminiCLI,
		model.AgentCursor,
		model.AgentVSCodeCopilot,
		model.AgentCodex,
		model.AgentAntigravity,
		model.AgentQwenCode,
		model.AgentPi,
		model.AgentTrae,
	}

	for _, agent := range allAgents {
		t.Run(string(agent), func(t *testing.T) {
			residual := agent == model.AgentClaudeCode
			want := assets.MustRead("generic/persona-axiom.md")
			switch agent {
			case model.AgentClaudeCode:
				want = assets.MustRead("claude/persona-axiom.md")
			case model.AgentOpenCode:
				want = assets.MustRead("opencode/persona-axiom.md")
			}
			if got := personaContent(agent, residual); got != want || got == "" {
				t.Fatalf("personaContent(%q, residual=%v) no coincide con el asset de Axiom esperado", agent, residual)
			}
		})
	}
}

func TestInjectClaudeOutputStylePreservesSettingsAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	existingSettings := `{"permissions":{"allow":["Read"]}}`
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(existingSettings), 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	first, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !first.Changed {
		t.Fatal("first Inject() changed = false, want true")
	}

	settingsContent, err := os.ReadFile(filepath.Join(settingsDir, "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsContent, &settings); err != nil {
		t.Fatalf("Unmarshal settings error = %v", err)
	}
	if got, want := settings["outputStyle"], "Axiom"; got != want {
		t.Fatalf("settings outputStyle = %q, want %q", got, want)
	}
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("settings lost existing permissions key")
	}

	second, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatal("second Claude inject changed = true, want idempotent false")
	}
}
