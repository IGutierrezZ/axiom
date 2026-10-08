package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
)

// sddOnlyBody is a managed section as the SDD injector writes it.
const sddOnlyBody = "## SDD orchestrator\nmanaged text\n"

// sddOnlyContent builds a prompt file the way the SDD component creates one
// when it is the first and only component to write it: its own frontmatter
// followed by managed sections (see sdd.injectFileAppend).
func sddOnlyContent(frontmatter string) string {
	return filemerge.InjectMarkdownSection(frontmatter, filemerge.SDDOrchestratorSectionID, sddOnlyBody)
}

// sddFrontmatterFor returns the frontmatter the SDD component writes for the
// adapter case, or "" when it writes none (Cursor).
func sddFrontmatterFor(name string) string {
	switch name {
	case "kiro":
		return filemerge.SDDSteeringFrontmatter
	case "vscode":
		return filemerge.SDDInstructionsFrontmatter
	}
	return ""
}

// writeSDDOnlyLegacyPrompt writes the current prompt file and a legacy copy
// holding legacyContent, and returns the legacy path.
func writeSDDOnlyLegacyPrompt(t *testing.T, home string, c legacyPromptCase, currentContent, legacyContent string) string {
	t.Helper()
	current := c.adapter.SystemPromptFile(home)
	if err := os.MkdirAll(filepath.Dir(current), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current, []byte(currentContent), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(filepath.Dir(current), c.legacy)
	if err := os.WriteFile(legacy, []byte(legacyContent), 0o644); err != nil {
		t.Fatal(err)
	}
	return legacy
}

func assertLegacyPromptKept(t *testing.T, home string, c legacyPromptCase, legacy, wantContent string) {
	t.Helper()
	result := RetireLegacyPromptFiles(home, c.adapter)
	if len(result.Removed) != 0 {
		t.Fatalf("Removed = %q, want none", result.Removed)
	}
	if len(result.Notes) != 1 || !strings.Contains(result.Notes[0], legacy) {
		t.Fatalf("Notes = %q, want one warning naming %s", result.Notes, legacy)
	}
	if got, _ := os.ReadFile(legacy); string(got) != wantContent {
		t.Fatal("kept legacy prompt was modified")
	}
}

// A legacy prompt written by the real SDD injector alone (no persona) carries
// the SDD frontmatter and only managed sections: it is Axiom's and goes.
func TestRetireLegacyPromptFilesRemovesSDDOnlyLegacyCopy(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		t.Run(c.name, func(t *testing.T) {
			if _, err := sdd.Inject(home, c.adapter, ""); err != nil {
				t.Fatalf("sdd.Inject(%s): %v", c.name, err)
			}
			current := c.adapter.SystemPromptFile(home)
			body, err := os.ReadFile(current)
			if err != nil {
				t.Fatal(err)
			}
			if want := sddFrontmatterFor(c.name); !strings.HasPrefix(string(body), want) {
				t.Fatalf("SDD-only prompt does not start with the SDD frontmatter %q:\n%s", want, body)
			}
			legacy := filepath.Join(filepath.Dir(current), c.legacy)
			if err := os.WriteFile(legacy, body, 0o644); err != nil {
				t.Fatal(err)
			}

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 1 || result.Removed[0] != legacy {
				t.Fatalf("Removed = %q, want [%q] (notes: %q)", result.Removed, legacy, result.Notes)
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
		})
	}
}

func TestRetireLegacyPromptFilesRemovesLegacyCopyWithExactSDDFrontmatter(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			content := sddOnlyContent(frontmatter)
			legacy := writeSDDOnlyLegacyPrompt(t, home, c, content, content)

			result := RetireLegacyPromptFiles(home, c.adapter)
			if len(result.Removed) != 1 || result.Removed[0] != legacy || len(result.Notes) != 0 {
				t.Fatalf("result = %+v, want only %s removed", result, legacy)
			}
		})
	}
}

func TestRetireLegacyPromptFilesKeepsSDDFrontmatterWithAnExtraKey(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		edits := map[string]string{
			"extra key":      strings.TrimSuffix(frontmatter, "---\n") + "owner: me\n---\n",
			"trailing space": strings.Replace(frontmatter, "\n", " \n", 2),
			"changed value":  strings.Replace(frontmatter, "always", "manual", 1),
			"no closing":     strings.TrimSuffix(frontmatter, "---\n"),
		}
		if c.name == "vscode" {
			edits["changed value"] = strings.Replace(frontmatter, "Gentle AI Persona", "My Persona", 1)
		}
		for name, edited := range edits {
			t.Run(c.name+"/"+name, func(t *testing.T) {
				current := sddOnlyContent(frontmatter)
				legacyContent := sddOnlyContent(edited)
				legacy := writeSDDOnlyLegacyPrompt(t, home, c, current, legacyContent)
				assertLegacyPromptKept(t, home, c, legacy, legacyContent)
			})
		}
	}
}

func TestRetireLegacyPromptFilesKeepsSDDFrontmatterWithUserText(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		for name, userText := range map[string]string{
			"after sections":  "\nMy own team rule: always answer in haiku.\n",
			"between":         "\nMy own team rule.\n\n",
			"second metadata": "---\nowner: me\n---\n",
		} {
			t.Run(c.name+"/"+name, func(t *testing.T) {
				legacyContent := sddOnlyContent(frontmatter) + userText
				if name == "between" {
					legacyContent = frontmatter + userText + strings.TrimPrefix(sddOnlyContent(frontmatter), frontmatter)
				}
				legacy := writeSDDOnlyLegacyPrompt(t, home, c, sddOnlyContent(frontmatter), legacyContent)
				assertLegacyPromptKept(t, home, c, legacy, legacyContent)
			})
		}
	}
}

func TestRetireLegacyPromptFilesKeepsSDDFrontmatterWithUnmanagedSection(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		for _, block := range []string{
			"<!-- axiom:my-team-rules -->\nALWAYS ANSWER IN HAIKU\n<!-- /axiom:my-team-rules -->\n",
			"<!-- gentle-ai:foo -->\nkeep me\n<!-- /gentle-ai:foo -->\n",
			"<!-- axiom:sdd-orchestrator-extra -->\nuser text\n<!-- /axiom:sdd-orchestrator-extra -->\n",
		} {
			t.Run(c.name+"/"+strings.SplitN(block, " ", 3)[1], func(t *testing.T) {
				legacyContent := sddOnlyContent(frontmatter) + "\n" + block
				legacy := writeSDDOnlyLegacyPrompt(t, home, c, sddOnlyContent(frontmatter), legacyContent)
				result := RetireLegacyPromptFiles(home, c.adapter)
				if len(result.Removed) != 0 || len(result.Notes) != 1 || !strings.Contains(result.Notes[0], "does not manage") {
					t.Fatalf("result = %+v, want the file kept with a warning about the unknown section", result)
				}
				if got, _ := os.ReadFile(legacy); string(got) != legacyContent {
					t.Fatal("legacy prompt with a user section was modified")
				}
			})
		}
	}
}

// The SDD frontmatter alone proves nothing: with no managed section the file is
// still kept, as is a file whose managed sections are gone.
func TestRetireLegacyPromptFilesKeepsSDDFrontmatterWithoutManagedSection(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			legacyContent := frontmatter + "\nSome notes of mine.\n"
			legacy := writeSDDOnlyLegacyPrompt(t, home, c, sddOnlyContent(frontmatter), legacyContent)
			assertLegacyPromptKept(t, home, c, legacy, legacyContent)
		})
	}
}

// The SDD frontmatter is recognized only when the current prompt can stand in
// for the legacy file, like every other proof.
func TestRetireLegacyPromptFilesKeepsSDDOnlyCopyWhenCurrentPromptHasNoSection(t *testing.T) {
	home := t.TempDir()
	for _, c := range legacyPromptCases(t, home) {
		frontmatter := sddFrontmatterFor(c.name)
		if frontmatter == "" {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			legacyContent := sddOnlyContent(frontmatter)
			legacy := writeSDDOnlyLegacyPrompt(t, home, c, "just my notes\n", legacyContent)
			assertLegacyPromptKept(t, home, c, legacy, legacyContent)
		})
	}
}
