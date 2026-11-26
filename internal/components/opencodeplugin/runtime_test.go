package opencodeplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

func init() {
	opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
		return opencode.CommandOutput{Stdout: []byte("1.18.30")}, nil
	}
}
func TestV2TUIPreservesPlacement(t *testing.T) {
	old := opencode.VersionRunnerOverride
	t.Cleanup(func() { opencode.VersionRunnerOverride = old })
	opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
		return opencode.CommandOutput{Stdout: []byte("2.0.4")}, nil
	}
	for _, id := range []model.OpenCodeCommunityPluginID{model.OpenCodePluginSubAgentStatusline, model.OpenCodePluginSDDEngramManage} {
		t.Run(string(id), func(t *testing.T) {
			home := t.TempDir()
			if _, err := Install(home, id); err == nil {
				t.Fatal("unproven TUI silently installed")
			}
			if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
				t.Fatal("TUI refusal wrote files")
			}
		})
	}
}
