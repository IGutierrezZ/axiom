package persona

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// LegacyPromptResult reports what RetireLegacyPromptFiles did.
type LegacyPromptResult struct {
	// Removed lists the legacy prompt files that were deleted.
	Removed []string
	// Notes lists warnings for legacy files that were kept because Axiom could
	// not prove it wrote them.
	Notes []string
}

var managedSectionOpenMarker = regexp.MustCompile(`<!-- (axiom|gentle-ai):([A-Za-z0-9_.-]+) -->`)

// RetireLegacyPromptFiles removes the prompt files an earlier release wrote
// under a name the adapter no longer uses (for example Kiro's gentle-ai.md),
// which otherwise stay next to the renamed file and deliver the prompt twice.
//
// A legacy file is deleted only when Axiom provably wrote all of it: the
// current prompt file already exists, and nothing outside the managed marker
// sections and the installer's own frontmatter is anything but the persona text
// the installer generates. A file with any other content, an unreadable file
// and a file that is not a regular file are kept, with a note for the user.
func RetireLegacyPromptFiles(homeDir string, adapter agents.Adapter) LegacyPromptResult {
	var result LegacyPromptResult
	provider, ok := adapter.(agents.LegacyPromptFileProvider)
	if !ok {
		return result
	}
	if _, err := os.Stat(adapter.SystemPromptFile(homeDir)); err != nil {
		// Nothing replaces the legacy file yet, so deleting it would drop the prompt.
		return result
	}
	for _, path := range provider.LegacySystemPromptFiles(homeDir) {
		info, err := os.Lstat(path)
		if err != nil {
			if !os.IsNotExist(err) {
				result.Notes = append(result.Notes, keptLegacyPromptNote(path, adapter.SystemPromptFile(homeDir), err.Error()))
			}
			continue
		}
		if !info.Mode().IsRegular() {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, adapter.SystemPromptFile(homeDir), "it is not a regular file"))
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, adapter.SystemPromptFile(homeDir), err.Error()))
			continue
		}
		if !isInstallerWrittenPrompt(adapter, string(data)) {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, adapter.SystemPromptFile(homeDir), "it has content Axiom did not write"))
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, adapter.SystemPromptFile(homeDir), err.Error()))
			continue
		}
		result.Removed = append(result.Removed, path)
	}
	return result
}

func keptLegacyPromptNote(path, current, reason string) string {
	return fmt.Sprintf("The legacy prompt file %s was kept (%s). Axiom now writes %s, so the prompt may be duplicated; review %s and delete it manually.", path, reason, current, path)
}

// isInstallerWrittenPrompt reports whether content is made only of managed
// marker sections, the installer's frontmatter and installer persona text.
// Content with no managed section at all is never installer-written: the
// markers are what ties a file to the installer.
func isInstallerWrittenPrompt(adapter agents.Adapter, content string) bool {
	rest := strings.ReplaceAll(content, "\r\n", "\n")
	rest = strings.TrimPrefix(rest, "\ufeff")
	for _, frontmatter := range []string{wrapInstructionsFile(""), wrapSteeringFile("")} {
		if strings.HasPrefix(rest, frontmatter) {
			rest = strings.TrimPrefix(rest, frontmatter)
			break
		}
	}
	rest, sections, ok := removeManagedSections(rest)
	if !ok || sections == 0 {
		return false
	}
	return strings.TrimSpace(rest) == "" || isKnownPersonaText(adapter, rest)
}

// removeManagedSections cuts every <!-- axiom|gentle-ai:ID --> ... closing pair
// out of content. ok is false when a section is not closed.
func removeManagedSections(content string) (rest string, sections int, ok bool) {
	for {
		loc := managedSectionOpenMarker.FindStringSubmatchIndex(content)
		if loc == nil {
			return content, sections, true
		}
		prefix, id := content[loc[2]:loc[3]], content[loc[4]:loc[5]]
		closeMarker := "<!-- /" + prefix + ":" + id + " -->"
		end := strings.Index(content[loc[1]:], closeMarker)
		if end < 0 {
			return content, sections, false
		}
		content = content[:loc[0]] + content[loc[1]+end+len(closeMarker):]
		sections++
	}
}

// isKnownPersonaText reports whether text is, ignoring whitespace, one of the
// persona assets the installer embeds for the agent. Text that merely contains
// persona wording is not enough: a user could have added their own rules to it.
func isKnownPersonaText(adapter agents.Adapter, text string) bool {
	for _, persona := range []model.PersonaID{model.PersonaAxiom, model.PersonaNeutral, model.PersonaGentleman} {
		for _, residual := range []bool{false, true} {
			known := normalizeWhitespace(personaContent(adapter.Agent(), persona, residual))
			if known != "" && known == normalizeWhitespace(text) {
				return true
			}
		}
	}
	return false
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
