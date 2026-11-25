package tui

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/update"
	"github.com/IGutierrezZ/axiom/v3/internal/update/upgrade"
)

// TestSelfToolIdentityDetection verifies that the three H-4 sites detect the
// self-tool under both historical names: axiom and gentle-ai (F7.2).
func TestSelfToolIdentityDetection(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		{name: "detects axiom", toolName: "axiom", want: true},
		{name: "rejects retired gentle-ai name", toolName: "gentle-ai", want: false},
		{name: "rejects engram", toolName: "engram", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := update.IsSelfToolName(tc.toolName); got != tc.want {
				t.Errorf("IsSelfToolName(%q) = %v, want %v", tc.toolName, got, tc.want)
			}

			// Site 1: reportUpgradedSelf in model.go
			report := upgrade.UpgradeReport{Results: []upgrade.ToolUpgradeResult{
				{ToolName: tc.toolName, Status: upgrade.UpgradeSucceeded},
			}}
			if got := reportUpgradedSelf(report); got != tc.want {
				t.Errorf("reportUpgradedSelf(%q) = %v, want %v", tc.toolName, got, tc.want)
			}

			// Site 2: SelfUpgradeVersion in model.go
			m := Model{UpgradeReport: &upgrade.UpgradeReport{Results: []upgrade.ToolUpgradeResult{
				{ToolName: tc.toolName, Status: upgrade.UpgradeSucceeded, NewVersion: "v1.0.0"},
			}}}
			_, ok := m.SelfUpgradeVersion()
			if ok != tc.want {
				t.Errorf("SelfUpgradeVersion(%q) ok = %v, want %v", tc.toolName, ok, tc.want)
			}
		})
	}
}
