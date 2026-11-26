package agentbuilder

import (
	"fmt"
	"os"
	"strings"
)

// markerFormat is the HTML comment marker used to identify a custom-agent block.
// Example: <!-- axiom:custom-agent:my-skill -->
const markerFormat = "<!-- axiom:custom-agent:%s -->"

// El bloque se cierra con el mismo identificador precedido de "/".
const endMarkerFormat = "<!-- /axiom:custom-agent:%s -->"

// Marcadores que escribieron las versiones con el nombre anterior: solo se
// reconocen para sustituir el bloque ya escrito por el actual, nunca se escriben.
const (
	legacyMarkerFormat    = "<!-- gentle-ai:custom-agent:%s -->"
	legacyEndMarkerFormat = "<!-- /gentle-ai:custom-agent:%s -->"
)

// InjectSDDReference appends (or replaces) a custom-agent reference block in the
// system prompt file at systemPromptPath.
//
// For SDDPhaseSupport mode the block declares that the skill supports an existing
// phase. For SDDNewPhase mode the block integrates it as a first-class new phase.
//
// The function is a no-op when agent.SDDConfig is nil or the mode is SDDStandalone.
func InjectSDDReference(agent *GeneratedAgent, systemPromptPath string) error {
	if agent == nil || agent.SDDConfig == nil || agent.SDDConfig.Mode == SDDStandalone {
		return nil
	}

	data, err := os.ReadFile(systemPromptPath)
	if err != nil {
		return fmt.Errorf("sdd inject: read %s: %w", systemPromptPath, err)
	}

	content := string(data)
	marker := fmt.Sprintf(markerFormat, agent.Name)
	block := buildSDDBlock(agent, marker)

	legacyMarker := fmt.Sprintf(legacyMarkerFormat, agent.Name)
	switch {
	case strings.Contains(content, marker):
		// Replace the existing block.
		content = replaceBlock(content, marker, fmt.Sprintf(endMarkerFormat, agent.Name), block)
	case strings.Contains(content, legacyMarker):
		// Migrate a block written under the previous name: it is replaced, not duplicated.
		content = replaceBlock(content, legacyMarker, fmt.Sprintf(legacyEndMarkerFormat, agent.Name), block)
	default:
		// Append the block.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + block + "\n"
	}

	if err := os.WriteFile(systemPromptPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("sdd inject: write %s: %w", systemPromptPath, err)
	}

	return nil
}

// buildSDDBlock returns the full marker+content block for the agent.
func buildSDDBlock(agent *GeneratedAgent, marker string) string {
	cfg := agent.SDDConfig
	var body string

	switch cfg.Mode {
	case SDDPhaseSupport:
		body = fmt.Sprintf(
			"## Custom Agent: %s (Phase Support)\n\n"+
				"This skill provides additional support for the `sdd-%s` phase.\n"+
				"When working on tasks related to `%s`, load the `%s` skill for enhanced guidance.\n\n"+
				"Trigger phrases: %s\n",
			agent.Title,
			cfg.TargetPhase,
			cfg.TargetPhase,
			agent.Name,
			agent.Trigger,
		)
	case SDDNewPhase:
		phaseName := cfg.PhaseName
		if phaseName == "" {
			phaseName = agent.Name
		}
		body = fmt.Sprintf(
			"## Custom Agent: %s (New SDD Phase)\n\n"+
				"This skill adds a new phase `%s` to the SDD dependency graph.\n"+
				"Load the `%s` skill when the orchestrator launches you for the `%s` phase.\n\n"+
				"Trigger phrases: %s\n",
			agent.Title,
			phaseName,
			agent.Name,
			phaseName,
			agent.Trigger,
		)
	default:
		body = fmt.Sprintf("## Custom Agent: %s\n\nTrigger: %s\n", agent.Title, agent.Trigger)
	}

	endMarker := fmt.Sprintf(endMarkerFormat, agent.Name)
	return marker + "\n" + body + endMarker
}

// replaceBlock replaces the content between the opening marker and its matching
// closing marker with the new block string.
func replaceBlock(content, marker, endMarker, newBlock string) string {
	start := strings.Index(content, marker)
	if start == -1 {
		return content + "\n" + newBlock
	}

	end := strings.Index(content[start:], endMarker)
	if end == -1 {
		// No closing marker: replace from start marker to end of line.
		lineEnd := strings.Index(content[start:], "\n")
		if lineEnd == -1 {
			return content[:start] + newBlock
		}
		return content[:start] + newBlock + content[start+lineEnd:]
	}

	replaceEnd := start + end + len(endMarker)
	return content[:start] + newBlock + content[replaceEnd:]
}
