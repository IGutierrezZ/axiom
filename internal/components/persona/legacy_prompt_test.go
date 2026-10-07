package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/cursor"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kiro"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// legacyPromptCase describes one adapter whose prompt file was renamed.
type legacyPromptCase struct {
	name    string
	adapter agents.Adapter
	legacy  string
}

func legacyPromptCases(t *testing.T, home string) []legacyPromptCase {
	t.Helper()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	cases := []legacyPromptCase{
		{name: "kiro", adapter: kiro.NewAdapter(), legacy: "gentle-ai.md"},
		{name: "vscode", adapter: vscode.NewAdapter(), legacy: "gentle-ai.instructions.md"},
		{name: "cursor", adapter: cursor.NewAdapter(), legacy: "axiom-legacy-placeholder"},
	}
	cases[2].legacy = "gentle-ai.mdc"
	return cases
}

// writeManagedLegacyPrompt installs the persona at the adapter's current prompt
// file, adds a managed SDD section, and copies the result to the legacy name,
// which is what an earlier release left behind.
func writeManagedLegacyPrompt(t *testing.T, home string, c legacyPromptCase) (current, legacy string) {
	t.Helper()
	if _, err := Inject(home, c.adapter, model.PersonaGentleman); err != nil {
		t.Fatalf("Inject(%s): %v", c.name, err)
	}
	current = c.adapter.SystemPromptFile(home)
	body, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", current, err)
	}
	managed := filemerge.InjectMarkdownSection(string(body), "sdd-orchestrator", "## SDD orchestrator\nmanaged text\n")
	if err := os.WriteFile(current, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy = filepath.Join(filepath.Dir(current), c.legacy)
	if err := os.WriteFile(legacy, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}
	return current, legacy
}

func TestRetireLegacyPromptFilesRemovesManagedLegacyCopy(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			current, legacy := writeManagedLegacyPrompt(t, home, c)

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 1 || result.Removed[0] != legacy {
				t.Fatalf("Removed = %q, want [%q]", result.Removed, legacy)
			}
			if len(result.Notes) != 0 {
				t.Fatalf("Notes = %q, want none", result.Notes)
			}
			if _, err := os.Stat(legacy); !os.IsNotExist(err) {
				t.Fatalf("legacy prompt still exists: %v", err)
			}
			if _, err := os.Stat(current); err != nil {
				t.Fatalf("current prompt file was removed: %v", err)
			}
			if again := RetireLegacyPromptFiles(home, c.adapter); len(again.Removed) != 0 || len(again.Notes) != 0 {
				t.Fatalf("second run = %+v, want a no-op", again)
			}
		})
	}
}

func TestRetireLegacyPromptFilesKeepsEditedLegacyCopyWithWarning(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			_, legacy := writeManagedLegacyPrompt(t, home, c)
			body, err := os.ReadFile(legacy)
			if err != nil {
				t.Fatal(err)
			}
			edited := string(body) + "\nMy own team rule: always answer in haiku.\n"
			if err := os.WriteFile(legacy, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 0 {
				t.Fatalf("Removed = %q, want none", result.Removed)
			}
			if len(result.Notes) != 1 || !strings.Contains(result.Notes[0], legacy) {
				t.Fatalf("Notes = %q, want one warning naming %s", result.Notes, legacy)
			}
			if got, _ := os.ReadFile(legacy); string(got) != edited {
				t.Fatal("edited legacy prompt was modified")
			}
		})
	}
}

func TestRetireLegacyPromptFilesKeepsFilesWithoutManagedMarkers(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			current, _ := writeManagedLegacyPrompt(t, home, c)
			legacy := filepath.Join(filepath.Dir(current), c.legacy)
			if err := os.WriteFile(legacy, []byte("# My hand-written rules\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 0 || len(result.Notes) != 1 {
				t.Fatalf("result = %+v, want the file kept with one warning", result)
			}
			if _, err := os.Stat(legacy); err != nil {
				t.Fatalf("hand-written legacy file was removed: %v", err)
			}
		})
	}
}

func TestRetireLegacyPromptFilesWaitsForTheCurrentPromptFile(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			current, legacy := writeManagedLegacyPrompt(t, home, c)
			if err := os.Remove(current); err != nil {
				t.Fatal(err)
			}

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 0 || len(result.Notes) != 0 {
				t.Fatalf("result = %+v, want a silent no-op while nothing replaces the legacy file", result)
			}
			if _, err := os.Stat(legacy); err != nil {
				t.Fatalf("legacy prompt was removed without a replacement: %v", err)
			}
		})
	}
}

func TestRetireLegacyPromptFilesNoLegacyFileIsANoOp(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Inject(home, c.adapter, model.PersonaGentleman); err != nil {
				t.Fatal(err)
			}
			if result := RetireLegacyPromptFiles(home, c.adapter); len(result.Removed) != 0 || len(result.Notes) != 0 {
				t.Fatalf("result = %+v, want a no-op", result)
			}
		})
	}
}

func TestRetireLegacyPromptFilesIgnoresAdaptersWithoutLegacyNames(t *testing.T) {
	home := t.TempDir()
	if result := RetireLegacyPromptFiles(home, nonLegacyAdapter{kiro.NewAdapter()}); len(result.Removed) != 0 || len(result.Notes) != 0 {
		t.Fatalf("result = %+v, want a no-op", result)
	}
}

// nonLegacyAdapter hides the optional LegacyPromptFileProvider capability.
type nonLegacyAdapter struct{ agents.Adapter }

func TestRetireLegacyPromptFilesKeepsUnknownSections(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		for _, block := range []string{
			"<!-- axiom:my-team-rules -->\nALWAYS ANSWER IN HAIKU\n<!-- /axiom:my-team-rules -->\n",
			"<!-- gentle-ai:foo -->\nkeep me\n<!-- /gentle-ai:foo -->\n",
			// Syntax of a managed section around a name Axiom never writes.
			"<!-- axiom:persona-extra -->\nuser text\n<!-- /axiom:persona-extra -->\n",
		} {
			t.Run(c.name+"/"+strings.SplitN(block, " ", 3)[1], func(t *testing.T) {
				_, legacy := writeManagedLegacyPrompt(t, home, c)
				body, err := os.ReadFile(legacy)
				if err != nil {
					t.Fatal(err)
				}
				edited := string(body) + "\n" + block
				if err := os.WriteFile(legacy, []byte(edited), 0o644); err != nil {
					t.Fatal(err)
				}

				result := RetireLegacyPromptFiles(home, c.adapter)
				if len(result.Removed) != 0 || len(result.Notes) != 1 || !strings.Contains(result.Notes[0], "does not manage") {
					t.Fatalf("result = %+v, want the file kept with a warning about the unknown section", result)
				}
				if got, _ := os.ReadFile(legacy); string(got) != edited {
					t.Fatal("legacy prompt with a user section was modified")
				}
			})
		}
	}
}

func TestRetireLegacyPromptFilesKeepsUnclosedManagedSection(t *testing.T) {
	home := t.TempDir()
	c := legacyPromptCases(t, home)[0]
	_, legacy := writeManagedLegacyPrompt(t, home, c)
	if err := os.WriteFile(legacy, []byte("<!-- axiom:sdd-orchestrator -->\nno closing marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := RetireLegacyPromptFiles(home, c.adapter); len(result.Removed) != 0 || len(result.Notes) != 1 {
		t.Fatalf("result = %+v, want the file kept with a warning", result)
	}
}

func TestRetireLegacyPromptFilesRequiresAUsableCurrentPrompt(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name+"/directory", func(t *testing.T) {
			current, legacy := writeManagedLegacyPrompt(t, home, c)
			if err := os.Remove(current); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(current, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Remove(current) })

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 0 || len(result.Notes) != 1 {
				t.Fatalf("result = %+v, want the legacy file kept with a warning", result)
			}
			if _, err := os.Stat(legacy); err != nil {
				t.Fatalf("legacy prompt was removed: %v", err)
			}
		})
		t.Run(c.name+"/no managed section", func(t *testing.T) {
			current, legacy := writeManagedLegacyPrompt(t, home, c)
			if err := os.WriteFile(current, []byte("# only my own rules\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 0 || len(result.Notes) != 1 {
				t.Fatalf("result = %+v, want the legacy file kept with a warning", result)
			}
			if _, err := os.Stat(legacy); err != nil {
				t.Fatalf("legacy prompt was removed: %v", err)
			}
		})
	}
}

func TestRetireLegacyPromptFilesAcceptsCRLFAndByteOrderMark(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		for name, transform := range map[string]func(string) string{
			"crlf":     func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") },
			"bom":      func(s string) string { return "\ufeff" + s },
			"bom+crlf": func(s string) string { return "\ufeff" + strings.ReplaceAll(s, "\n", "\r\n") },
		} {
			t.Run(c.name+"/"+name, func(t *testing.T) {
				_, legacy := writeManagedLegacyPrompt(t, home, c)
				body, err := os.ReadFile(legacy)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(legacy, []byte(transform(string(body))), 0o644); err != nil {
					t.Fatal(err)
				}

				result := RetireLegacyPromptFiles(home, c.adapter)
				if len(result.Removed) != 1 || len(result.Notes) != 0 {
					t.Fatalf("result = %+v, want the managed legacy file removed", result)
				}
			})
		}
	}
}
