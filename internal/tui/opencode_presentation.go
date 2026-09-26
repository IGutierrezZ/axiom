package tui

import (
	"context"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
	tea "github.com/charmbracelet/bubbletea"
)

// Presentation evidence never changes runtime selection or installation policy.
type openCodePresentationMsg struct{ major opencode.RuntimeMajor }

func openCodePresentationCommand() tea.Cmd {
	return func() tea.Msg {
		major, _ := opencode.DetectRuntimeMajor(context.Background())
		return openCodePresentationMsg{major: major}
	}
}
