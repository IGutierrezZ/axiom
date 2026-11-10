package sdd

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// sharedSectionMarkerLine matches one `<!-- sdd-orchestrator-section:NAME:start -->`
// line of the shared sections asset.
var sharedSectionMarkerLine = regexp.MustCompile(`(?m)^<!-- sdd-orchestrator-section:(.+):start -->$`)

// anyOrchestratorSectionPlaceholder matches every `{{GENTLE_AI_..._SECTION:` token,
// not only the SDD one, so a future family of shared sections cannot ship
// unresolved either.
var anyOrchestratorSectionPlaceholder = regexp.MustCompile(`\{\{GENTLE_AI_[A-Z_]*SECTION:`)

// resolvedOrchestratorSource returns an orchestrator template with its shared
// sections expanded and nothing else applied. Tests that assert over the text a
// runtime states, rather than over the template, read it through here: most of
// that text now lives once in the shared asset.
func resolvedOrchestratorSource(assetPath string) string {
	return substituteSharedOrchestratorSections(assets.MustRead(assetPath))
}

// orchestratorTemplatePaths lists every runtime orchestrator template.
func orchestratorTemplatePaths(t *testing.T) []string {
	t.Helper()
	paths, err := fs.Glob(assets.FS, "*/sdd-orchestrator.md")
	if err != nil {
		t.Fatalf("glob orchestrator templates: %v", err)
	}
	if len(paths) != 12 {
		t.Fatalf("orchestrator templates = %d (%v), want 12", len(paths), paths)
	}
	return paths
}

// sharedSectionNamesInAsset lists every section the shared asset defines, so
// the no-drift checks cover a section the moment it is added instead of
// waiting for someone to register its name.
func sharedSectionNamesInAsset(t *testing.T) []string {
	t.Helper()
	matches := sharedSectionMarkerLine.FindAllStringSubmatch(assets.MustRead(sharedOrchestratorSectionsAsset), -1)
	if len(matches) == 0 {
		t.Fatal("shared orchestrator sections asset defines no section")
	}
	names := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		name := match[1]
		if seen[name] {
			t.Fatalf("shared orchestrator section %q is defined twice", name)
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// TestOrchestratorTemplatesRenderWithoutUnresolvedSectionPlaceholders pins that
// no template reaches a runtime carrying a literal section token, for all
// twelve templates.
func TestOrchestratorTemplatesRenderWithoutUnresolvedSectionPlaceholders(t *testing.T) {
	for _, templatePath := range orchestratorTemplatePaths(t) {
		t.Run(path.Dir(templatePath), func(t *testing.T) {
			rendered := resolvedOrchestratorSource(templatePath)
			if loc := anyOrchestratorSectionPlaceholder.FindStringIndex(rendered); loc != nil {
				t.Fatalf("%s kept an unresolved section placeholder near %q", templatePath, rendered[loc[0]:min(len(rendered), loc[0]+80)])
			}
		})
	}
}

// TestSharedOrchestratorSectionsRenderExactlyOncePerReference pins that a
// section a template references is expanded once, and that a template that does
// not reference it does not carry it by accident.
func TestSharedOrchestratorSectionsRenderExactlyOncePerReference(t *testing.T) {
	names := sharedSectionNamesInAsset(t)
	for _, templatePath := range orchestratorTemplatePaths(t) {
		raw := assets.MustRead(templatePath)
		rendered := substituteSharedOrchestratorSections(raw)
		for _, name := range names {
			if strings.TrimSpace(sharedOrchestratorSection(name)) == "" {
				t.Fatalf("shared section %q has an empty body", name)
			}
			// The expanded text, not the canonical one: a body may carry tokens
			// (the stop-exits marker) that expansion resolves.
			body := substituteSharedOrchestratorSections(sharedOrchestratorSectionOpen + name + "}}")
			references := strings.Count(raw, sharedOrchestratorSectionOpen+name+"}}")
			if references > 1 {
				t.Errorf("%s references shared section %q %d times, want at most 1", templatePath, name, references)
			}
			if got := strings.Count(rendered, body); got != references {
				t.Errorf("%s renders shared section %q %d times for %d reference(s)", templatePath, name, got, references)
			}
		}
	}
}

// TestSharedOrchestratorSectionsAreNotCopiedBackIntoTemplates is the ratchet
// against duplication coming back: no raw template may carry the literal body
// of a shared section, because the next edit would reach only some runtimes.
func TestSharedOrchestratorSectionsAreNotCopiedBackIntoTemplates(t *testing.T) {
	names := sharedSectionNamesInAsset(t)
	for _, templatePath := range orchestratorTemplatePaths(t) {
		raw := assets.MustRead(templatePath)
		for _, name := range names {
			if body := sharedOrchestratorSection(name); strings.Contains(raw, body) {
				t.Errorf("%s carries the body of shared section %q inline; it must carry {{GENTLE_AI_SDD_SECTION:%s}} instead", templatePath, name, name)
			}
		}
	}
}

// TestEverySharedOrchestratorSectionHasAUser keeps the shared asset from
// accumulating bodies no template references.
func TestEverySharedOrchestratorSectionHasAUser(t *testing.T) {
	used := map[string]bool{}
	for _, templatePath := range orchestratorTemplatePaths(t) {
		for _, match := range sharedOrchestratorSectionPlaceholder.FindAllStringSubmatch(assets.MustRead(templatePath), -1) {
			used[match[1]] = true
		}
	}
	for _, name := range sharedSectionNamesInAsset(t) {
		if !used[name] {
			t.Errorf("shared section %q is defined but no orchestrator template references it", name)
		}
	}
}
