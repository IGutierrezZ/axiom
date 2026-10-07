package cli

import (
	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
)

// legacyPromptRetirementStep retires the prompt files that earlier releases
// wrote under a name an adapter no longer uses (Kiro's gentle-ai.md, VS Code's
// gentle-ai.instructions.md, Cursor's gentle-ai.mdc), which otherwise sit next to
// the renamed file and deliver the prompt twice. It never fails the run and has
// no rollback: only a file Axiom provably wrote is removed, and anything else is
// kept and reported as a warning for the user to act on.
type legacyPromptRetirementStep struct {
	state    *runtimeState
	id       string
	adapters []agents.Adapter
	// targetDir resolves the directory an adapter writes its prompt file under.
	targetDir func(agents.Adapter) string
}

func (s legacyPromptRetirementStep) ID() string { return s.id }

func (s legacyPromptRetirementStep) Run() error {
	for _, adapter := range s.adapters {
		result := persona.RetireLegacyPromptFiles(s.targetDir(adapter), adapter)
		if s.state == nil {
			continue
		}
		s.state.retiredFiles = append(s.state.retiredFiles, result.Removed...)
		s.state.retirementNotes = append(s.state.retirementNotes, result.Notes...)
	}
	return nil
}

// legacyPromptAdapters returns the selected adapters that declare legacy prompt
// file names.
func legacyPromptAdapters(adapters []agents.Adapter) []agents.Adapter {
	var selected []agents.Adapter
	for _, adapter := range adapters {
		if _, ok := adapter.(agents.LegacyPromptFileProvider); ok {
			selected = append(selected, adapter)
		}
	}
	return selected
}
