package persona

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
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

// managedPromptSectionIDs are the only marker sections the installer writes into
// an agent's system prompt file, taken from the constants its injectors use:
// persona (persona), sdd-orchestrator and strict-tdd-mode (sdd), engram-protocol
// (engram), codegraph-guidance (communitytool), agent-routing and the
// remote-authorization boundary nested in it (agentguidance), and the retired
// trigger-rules. Any other ID, even in an axiom or gentle-ai marker, is user
// content: a user can write a block with that syntax and any name.
var managedPromptSectionIDs = map[string]struct{}{
	filemerge.PersonaSectionID:             {},
	filemerge.SDDOrchestratorSectionID:     {},
	filemerge.StrictTDDSectionID:           {},
	filemerge.EngramProtocolSectionID:      {},
	filemerge.CodeGraphGuidanceSectionID:   {},
	filemerge.RemoteAuthorizationSectionID: {},
	filemerge.LegacyTriggerRulesSectionID:  {},
	agentguidance.RoutingSectionID:         {},
}

// legacyInstructionsFrontmatter is the header wrapInstructionsFile wrote under
// the previous product name. Files installed by those releases still start with
// it, so it keeps proving that the installer wrote them.
const legacyInstructionsFrontmatter = "---\n" +
	"name: Gentle AI Persona\n" +
	"description: Teaching-oriented persona with SDD orchestration and Engram protocol\n" +
	"applyTo: \"**\"\n" +
	"---\n\n"

// installerFrontmatters returns the only headers a legacy prompt file of this
// adapter may start with, chosen by the prompt strategy that decides which
// header Axiom writes at that path: steering files (Kiro) take the steering
// headers, instructions files (VS Code) the instructions headers, and any other
// target (Cursor) none. For each target they are the persona installer's
// (wrapSteeringFile, wrapInstructionsFile) and the one the SDD component writes
// when it creates the file on its own (shared with its injector through
// filemerge), plus the instructions headers earlier releases wrote under the
// previous product name. A header that belongs to another adapter is not what Axiom wrote
// at this path. A file is matched byte for byte, so an extra key, a changed
// value or any whitespace difference leaves the header as content Axiom did not
// write.
func installerFrontmatters(adapter agents.Adapter) []string {
	switch adapter.SystemPromptStrategy() {
	case model.StrategySteeringFile:
		return []string{wrapSteeringFile(""), filemerge.SDDSteeringFrontmatter}
	case model.StrategyInstructionsFile:
		return []string{
			wrapInstructionsFile(""),
			filemerge.SDDInstructionsFrontmatter,
			legacyInstructionsFrontmatter,
			filemerge.LegacySDDInstructionsFrontmatter,
		}
	}
	return nil
}

var managedSectionOpenMarker = regexp.MustCompile(`<!-- (axiom|gentle-ai):([A-Za-z0-9_.-]+) -->`)

// RetireLegacyPromptFiles removes the prompt files an earlier release wrote
// under a name the adapter no longer uses (for example Kiro's gentle-ai.md),
// which otherwise stay next to the renamed file and deliver the prompt twice.
//
// A legacy file is deleted only when Axiom provably wrote all of it:
//   - the current prompt file is a regular file that already holds at least one
//     managed section, so deleting the legacy copy cannot drop the prompt;
//   - the legacy file holds at least one managed section, every marker section
//     in it is one of managedPromptSectionIDs, and each is closed;
//   - outside those sections and the frontmatter installerFrontmatters allows for the adapter there is
//     nothing, or exactly the persona text the installer generates.
//
// The content inside a managed section counts as Axiom-owned: sync rewrote it on
// every run, so a user edit there never survived. Anything else (an unknown
// section, text outside the sections, an unreadable or non-regular file) keeps
// the file and adds a note for the user.
func RetireLegacyPromptFiles(homeDir string, adapter agents.Adapter) LegacyPromptResult {
	var result LegacyPromptResult
	provider, ok := adapter.(agents.LegacyPromptFileProvider)
	if !ok {
		return result
	}
	current := adapter.SystemPromptFile(homeDir)
	if _, err := os.Lstat(current); err != nil && os.IsNotExist(err) {
		// Nothing replaces the legacy file yet, so deleting it would drop the prompt.
		// Without a current file there is also no duplicate to warn about.
		return result
	}
	currentReason := currentPromptProblem(current)
	for _, path := range provider.LegacySystemPromptFiles(homeDir) {
		info, err := os.Lstat(path)
		if err != nil {
			if !os.IsNotExist(err) {
				result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, err.Error()))
			}
			continue
		}
		if currentReason != "" {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, currentReason))
			continue
		}
		if !info.Mode().IsRegular() {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, "it is not a regular file"))
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, err.Error()))
			continue
		}
		if reason := legacyPromptProblem(adapter, string(data)); reason != "" {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, reason))
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Notes = append(result.Notes, keptLegacyPromptNote(path, current, err.Error()))
			continue
		}
		result.Removed = append(result.Removed, path)
	}
	return result
}

func keptLegacyPromptNote(path, current, reason string) string {
	return fmt.Sprintf("The legacy prompt file %s was kept (%s). Axiom now writes %s, so the prompt may be duplicated; review %s and delete it manually.", path, reason, current, path)
}

// currentPromptProblem returns why the current prompt file cannot stand in for
// the legacy one, or "" when it is a regular file holding a managed section.
func currentPromptProblem(path string) string {
	info, err := os.Lstat(path)
	if err != nil {
		return err.Error()
	}
	if !info.Mode().IsRegular() {
		return "the current prompt file is not a regular file"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	_, sections, problem := removeManagedSections(normalizePrompt(string(data)))
	if problem != "" || sections == 0 {
		return "the current prompt file holds no section Axiom manages"
	}
	return ""
}

// normalizePrompt drops a byte order mark and normalizes line endings.
func normalizePrompt(content string) string {
	return strings.TrimPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "\ufeff")
}

// legacyPromptProblem returns why content is not provably installer-written, or
// "" when it is made only of managed sections, the installer's frontmatter and
// installer persona text. The markers are what ties a file to the installer, so
// content with no managed section at all is never installer-written.
func legacyPromptProblem(adapter agents.Adapter, content string) string {
	rest := normalizePrompt(content)
	for _, frontmatter := range installerFrontmatters(adapter) {
		if strings.HasPrefix(rest, frontmatter) {
			rest = strings.TrimPrefix(rest, frontmatter)
			break
		}
	}
	rest, sections, problem := removeManagedSections(rest)
	if problem != "" {
		return problem
	}
	if sections == 0 {
		return "it holds no section Axiom manages"
	}
	if strings.TrimSpace(rest) != "" && !isKnownPersonaText(adapter, rest) {
		return "it has content Axiom did not write"
	}
	return ""
}

// removeManagedSections cuts every managed marker section out of content and
// counts them. problem is set when a section is not one the installer writes or
// is not closed.
func removeManagedSections(content string) (rest string, sections int, problem string) {
	for {
		loc := managedSectionOpenMarker.FindStringSubmatchIndex(content)
		if loc == nil {
			return content, sections, ""
		}
		prefix, id := content[loc[2]:loc[3]], content[loc[4]:loc[5]]
		if _, managed := managedPromptSectionIDs[id]; !managed {
			return content, sections, "it has a section Axiom does not manage: " + id
		}
		closeMarker := "<!-- /" + prefix + ":" + id + " -->"
		end := strings.Index(content[loc[1]:], closeMarker)
		if end < 0 {
			return content, sections, "section " + id + " is not closed"
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
