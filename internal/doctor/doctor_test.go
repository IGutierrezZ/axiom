package doctor

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestToolCheckID(t *testing.T) {
	tests := []struct {
		tool string
		want CheckID
	}{
		{tool: "gentle-ai", want: "tool:gentle-ai"},
		{tool: "engram", want: "tool:engram"},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			if got := ToolCheckID(tt.tool); got != tt.want {
				t.Fatalf("ToolCheckID(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestRunnerPreservesDeclarationOrderAndStableIDs(t *testing.T) {
	var order []CheckID
	checks := []Check{
		{ID: CheckStateJSON, Run: recordingCheck(&order, CheckStateJSON, Result{Name: "unstable", Status: StatusPass})},
		{ID: CheckEngramReachable, Run: func(context.Context) Result { order = append(order, CheckEngramReachable); panic("secret panic value") }},
		{ID: CheckDiskSpace, Run: recordingCheck(&order, CheckDiskSpace, Result{Status: StatusWarn})},
	}
	want := []Result{{Name: CheckStateJSON, Scope: ScopeGlobal, Status: StatusPass}, {Name: CheckEngramReachable, Scope: ScopeGlobal, Status: StatusFail, Detail: "check failed unexpectedly"}, {Name: CheckDiskSpace, Scope: ScopeGlobal, Status: StatusWarn}}
	got := (Runner{Checks: checks}).Run(context.Background())
	if !reflect.DeepEqual(got.Checks, want) || !reflect.DeepEqual(order, []CheckID{CheckStateJSON, CheckEngramReachable, CheckDiskSpace}) {
		t.Fatalf("checks/order = %#v/%v, want %#v", got.Checks, order, want)
	}
	if strings.Contains(got.Checks[1].Detail, "secret") {
		t.Fatalf("panic value leaked in evidence: %q", got.Checks[1].Detail)
	}
	if got := (Runner{}).Run(context.Background()).Checks; got == nil || len(got) != 0 {
		t.Fatalf("empty checks = %#v, want non-nil empty slice", got)
	}
}

func TestNewRemedyClassifiesCurrentRemedies(t *testing.T) {
	tests := []struct {
		id       RemedyID
		category RemedyCategory
		mode     ActionMode
		eligible bool
	}{
		{RemedyInstallTool, RemedyCategoryInstall, ActionManualOnly, false},
		{RemedyRemoveDuplicates, RemedyCategoryEnvironment, ActionManualOnly, false},
		{RemedyReorderPath, RemedyCategoryEnvironment, ActionManualOnly, false},
		{RemedyInstall, RemedyCategoryInstall, ActionManualOnly, false},
		{RemedyRepairState, RemedyCategoryConfiguration, ActionManualOnly, false},
		{RemedySync, RemedyCategoryConfiguration, ActionConfirmation, true},
		{RemedyStartEngram, RemedyCategoryService, ActionManualOnly, false},
		{RemedyInspectEngram, RemedyCategoryService, ActionManualOnly, false},
		{RemedyFreeDiskSpace, RemedyCategoryStorage, ActionManualOnly, false},
	}
	for _, tt := range tests {
		r := NewRemedy(tt.id, "guidance")
		if r.Description != "guidance" || r.Category != tt.category || r.ActionMode != tt.mode || r.Eligible != tt.eligible || r.EligibilityReason == "" || len(r.SupportedPlatforms) != 4 {
			t.Errorf("NewRemedy(%q) = %#v", tt.id, r)
		}
	}
}

func recordingCheck(order *[]CheckID, id CheckID, result Result) func(context.Context) Result {
	return func(context.Context) Result { *order = append(*order, id); return result }
}

func TestRunnerStampsScopeAndReportFailed(t *testing.T) {
	checks := []Check{
		{ID: "a", Scope: ScopeProject, Run: func(context.Context) Result { return Result{Status: StatusWarn} }},
		{ID: "b", Run: func(context.Context) Result { return Result{Status: StatusPass} }},
	}
	got := (Runner{Checks: checks}).Run(context.Background())
	if got.Checks[0].Scope != ScopeProject || got.Checks[1].Scope != ScopeGlobal {
		t.Fatalf("scopes = %q/%q, want project/global", got.Checks[0].Scope, got.Checks[1].Scope)
	}
	if got.Failed() {
		t.Fatalf("Failed() = true with only pass/warn results")
	}
	got.Checks = append(got.Checks, Result{Status: StatusFail})
	if !got.Failed() {
		t.Fatalf("Failed() = false with a failing result")
	}
}

func TestReportMarshalsStableJSON(t *testing.T) {
	report := Report{Checks: []Result{{
		Name: "x", Scope: ScopeProject, Status: StatusWarn, Detail: "d",
		Remedy: NewRemedy(RemedySync, "Run sync"),
	}}}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"name":"x"`, `"scope":"project"`, `"status":"warn"`, `"detail":"d"`, `"id":"sync"`, `"action_mode":"confirmation"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("JSON %s lacks %s", raw, want)
		}
	}
}
