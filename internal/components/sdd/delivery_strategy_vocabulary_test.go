package sdd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// A preserved OpenCode prompt with a legacy owned block must replace that block,
// so retired delivery mappings cannot survive a sync.
func TestInjectOpenCodeMigratesRetiredDeliveryStrategyMapping(t *testing.T) {
	home := t.TempDir()
	mockNoPackageManager(t)

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}

	stalePrompt := legacySDDSessionPreflightMarker + "\n" +
		"3. PRs: Ask me, Single PR, Auto.\n" +
		"Ask me -> `ask-always`; Single PR -> `single-pr-default`; Auto -> `auto-forecast`\n" +
		legacySDDSessionPreflightEnd
	if !strings.Contains(stalePrompt, "Ask me -> `ask-always`") {
		t.Fatal("test seed did not reproduce the retired mapping; the literal shape changed")
	}

	seed := `{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "prompt": ` + strconv.Quote("# Custom prompt\n"+stalePrompt) + `
    }
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(opencode.json) error = %v", err)
	}

	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	text := preservedOrchestratorPrompt(t, settingsPath)

	for _, retired := range []string{"ask-always", "single-pr-default", "force-chained", "auto-forecast"} {
		if strings.Contains(text, retired) {
			t.Errorf("opencode.json kept retired delivery strategy value %q after sync", retired)
		}
	}
	if strings.Contains(text, "Ask me ->") {
		t.Error("opencode.json kept the retired delivery strategy mapping")
	}
	if !strings.Contains(text, "# Custom prompt") {
		t.Error("migration discarded the user's own prompt content")
	}
}

// The four-option preflight shipped alongside the already-corrected
// `ask-on-risk` mapping, so a prompt that still offers `Chained` satisfies every
// freshness clause that existed before this change — including the one added
// when the retired mapping was fixed. Without a clause that fails on the retired
// option, an installed config would keep the four-option menu forever and the
// asset-only removal would be reverted on the operator's next sync.
func TestInjectOpenCodeMigratesRetiredChainedPRPreflightOption(t *testing.T) {
	home := t.TempDir()
	mockNoPackageManager(t)

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}

	// Rebuild the exact prompt the previous release injected by reversing this
	// change on the current literal, so the seed tracks the literal instead of
	// freezing a copy of it that could drift.
	stalePrompt := legacySDDSessionPreflightMarker + "\n" +
		"3. PRs: Ask me, Single PR, Chained, Auto.\n" +
		"Chained -> `auto-chain`\n" +
		"Chained and Auto both resolve to `auto-chain` because `delivery_strategy` is only consulted once the tasks forecast flags review-budget risk.\n" +
		legacySDDSessionPreflightEnd

	for _, seeded := range []string{
		"3. PRs: Ask me, Single PR, Chained, Auto.",
		"Chained -> `auto-chain`",
		"Chained and Auto both resolve to `auto-chain`",
	} {
		if !strings.Contains(stalePrompt, seeded) {
			t.Fatalf("test seed did not reproduce the retired four-option preflight fragment %q; the literal shape changed", seeded)
		}
	}

	seed := `{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "prompt": ` + strconv.Quote("# Custom prompt\n"+stalePrompt) + `
    }
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(opencode.json) error = %v", err)
	}

	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	text := preservedOrchestratorPrompt(t, settingsPath)

	for _, residue := range []string{
		"3. PRs: Ask me, Single PR, Chained, Auto.",
		"Chained -> `auto-chain`",
		"Chained and Auto both resolve to `auto-chain`",
	} {
		if strings.Contains(text, residue) {
			t.Errorf("opencode.json kept retired preflight fragment %q after sync", residue)
		}
	}
	if strings.Contains(text, "PR strategy") || strings.Contains(text, "auto-chain") || strings.Contains(text, "Auto ->") {
		t.Error("opencode.json kept the PR strategy group after the migration; the preflight no longer offers it")
	}
	if !strings.Contains(text, "# Custom prompt") {
		t.Error("migration discarded the user's own prompt content")
	}
	if count := strings.Count(text, sddSessionPreflightMarker); count != 1 {
		t.Errorf("migrated prompt carries %d canonical preflight blocks; the retired menu must be replaced, not appended to", count)
	}

	// The freshness clause that fires this migration must also stop firing once
	// it has run, or every later sync would rewrite the operator's prompt again.
	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("second Inject() error = %v", err)
	}
	if resynced := preservedOrchestratorPrompt(t, settingsPath); resynced != text {
		t.Error("re-syncing the migrated prompt changed it again; the migration is not idempotent")
	}
}

// preservedOrchestratorPrompt reads the preserved prompt back through JSON
// rather than matching raw bytes: encoding/json escapes the ">" in "->", so a
// substring match on the file would silently never fire.
func preservedOrchestratorPrompt(t *testing.T, settingsPath string) string {
	t.Helper()

	settingsBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}
	var settings struct {
		Agent map[string]struct {
			Prompt string `json:"prompt"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		t.Fatalf("Unmarshal(opencode.json) error = %v", err)
	}
	prompt := settings.Agent["axiom-orchestrator"].Prompt
	if prompt == "" {
		prompt = settings.Agent["gentle-orchestrator"].Prompt
	}
	if prompt == "" {
		t.Fatal("opencode.json lost the preserved orchestrator prompt")
	}
	return prompt
}
