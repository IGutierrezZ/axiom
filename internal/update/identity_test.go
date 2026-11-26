package update

import "testing"

// TestIsSelfToolName verifies the closed-set self-tool identity predicate
// (D-01, REQ-22.3). Matching is case- and edge-whitespace-insensitive so the
// lookup can never be bypassed by a spelling variant.
func TestIsSelfToolName(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		{name: "accepts lowercase axiom", toolName: "axiom", want: true},
		{name: "rejects the retired gentle-ai name", toolName: "gentle-ai", want: false},
		{name: "accepts uppercase AXIOM", toolName: "AXIOM", want: true},
		{name: "accepts mixed case with edge spaces", toolName: " Axiom ", want: true},
		{name: "rejects engram", toolName: "engram", want: false},
		{name: "rejects empty", toolName: "", want: false},
		{name: "rejects lookalike Ax1om", toolName: "Ax1om", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSelfToolName(tt.toolName); got != tt.want {
				t.Errorf("IsSelfToolName(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

// TestIsSelfTool verifies the ToolInfo-level identity predicate (D-01)
// delegates to IsSelfToolName on the tool name.
func TestIsSelfTool(t *testing.T) {
	tests := []struct {
		name string
		tool ToolInfo
		want bool
	}{
		{name: "accepts ToolInfo named axiom", tool: ToolInfo{Name: "axiom"}, want: true},
		{name: "rejects ToolInfo named gentle-ai", tool: ToolInfo{Name: "gentle-ai"}, want: false},
		{name: "rejects ToolInfo named engram", tool: ToolInfo{Name: "engram"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSelfTool(tt.tool); got != tt.want {
				t.Errorf("IsSelfTool(%+v) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

// TestSelectSelfToolResultFindsTheRegistryEntry is the regression for the
// start-up self-update that filtered by a stale literal and never found its own
// tool: the result produced for the shipped registry entry must be selected.
func TestSelectSelfToolResultFindsTheRegistryEntry(t *testing.T) {
	var selfEntry ToolInfo
	for _, tool := range Tools {
		if IsSelfTool(tool) {
			selfEntry = tool
			break
		}
	}
	if selfEntry.Name == "" {
		t.Fatal("no self-tool entry in the registry")
	}

	results := []UpdateResult{
		{Tool: ToolInfo{Name: "engram"}, Status: UpToDate},
		{Tool: selfEntry, Status: UpdateAvailable, LatestVersion: "9.9.9"},
	}
	got := SelectSelfToolResult(results)
	if got == nil {
		t.Fatalf("SelectSelfToolResult() = nil for the registry entry %q", selfEntry.Name)
	}
	if got.Tool.Name != SelfToolName || got.LatestVersion != "9.9.9" {
		t.Errorf("SelectSelfToolResult() = %+v, want the self-tool result", got)
	}

	if SelectSelfToolResult([]UpdateResult{{Tool: ToolInfo{Name: "gentle-ai"}}, {Tool: ToolInfo{Name: "gga"}}}) != nil {
		t.Error("SelectSelfToolResult() must ignore the retired gentle-ai name and non-self tools")
	}
	if SelectSelfToolResult(nil) != nil {
		t.Error("SelectSelfToolResult(nil) must be nil")
	}
}

// TestSelfToolRegistryNameMatchesTheIdentity pins that the shipped registry
// entry is named after the identity constant, so no rewrite is needed at start-up.
func TestSelfToolRegistryNameMatchesTheIdentity(t *testing.T) {
	if Tools[0].Name != SelfToolName || !IsSelfTool(Tools[0]) {
		t.Fatalf("Tools[0].Name = %q, want %q", Tools[0].Name, SelfToolName)
	}
}
