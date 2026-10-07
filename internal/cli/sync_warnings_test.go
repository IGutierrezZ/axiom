package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/verify"
)

func syncResultWithWarnings() SyncResult {
	return SyncResult{
		Agents:             []model.AgentID{model.AgentClaudeCode},
		FilesChanged:       1,
		ChangedFiles:       []string{"managed.md"},
		SkillRegistryError: "index is read-only",
		Execution:          pipeline.ExecutionResult{ManualActions: []string{"a retired file was kept"}},
		Verify: verify.BuildReport([]verify.CheckResult{
			{ID: "ok", Status: verify.CheckStatusPassed},
			{ID: "retired", Status: verify.CheckStatusWarning, Error: "legacy skill is still installed"},
		}),
	}
}

func TestSyncResultWarningsListEverythingTheReportWarnsAbout(t *testing.T) {
	result := syncResultWithWarnings()

	want := []string{
		"skill index refresh: index is read-only",
		"a retired file was kept",
		"legacy skill is still installed",
	}
	if got := result.Warnings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Warnings() = %q, want %q", got, want)
	}

	report := RenderSyncReport(result)
	lastIndex := -1
	for _, warning := range result.Warnings() {
		line := "WARNING: " + warning
		index := strings.Index(report, line)
		if index < 0 {
			t.Fatalf("RenderSyncReport() lacks %q:\n%s", line, report)
		}
		if index < lastIndex {
			t.Fatalf("RenderSyncReport() prints %q out of order:\n%s", line, report)
		}
		lastIndex = index
	}
}

func TestSyncResultWarningsEmptyWhenSyncIsClean(t *testing.T) {
	result := syncResultWithWarnings()
	result.SkillRegistryError = ""
	result.Execution.ManualActions = nil
	result.Verify = verify.BuildReport([]verify.CheckResult{{ID: "ok", Status: verify.CheckStatusPassed}})

	if got := result.Warnings(); len(got) != 0 {
		t.Fatalf("Warnings() = %q, want none", got)
	}

	// A refreshed index never reports a stale error.
	result.SkillRegistryRefreshed = true
	result.SkillRegistryError = "stale"
	if got := result.Warnings(); len(got) != 0 {
		t.Fatalf("Warnings() with refreshed index = %q, want none", got)
	}
}
