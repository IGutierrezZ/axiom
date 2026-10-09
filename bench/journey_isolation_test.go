package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestJourneyUnsupportedReasonOnlyOnWindows(t *testing.T) {
	marked := Journey{ID: "jShim", ExecutesPOSIXShims: true}
	plain := Journey{ID: "jPlain"}
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		if reason := journeyUnsupportedReason(marked, goos); reason != "" {
			t.Fatalf("%s: a shim journey must keep running, got %q", goos, reason)
		}
	}
	reason := journeyUnsupportedReason(marked, "windows")
	if !strings.Contains(reason, "POSIX") || !strings.Contains(reason, "Windows") {
		t.Fatalf("windows reason = %q, want a legible explanation naming POSIX shims and Windows", reason)
	}
	if reason := journeyUnsupportedReason(plain, "windows"); reason != "" {
		t.Fatalf("a journey with no shim must run on Windows, got %q", reason)
	}
}

func TestUnsupportedJourneyResultIsExplicitAndExcludedFromTotals(t *testing.T) {
	journey := Journey{ID: "jShim", Title: "needs a shim", Source: "src", ExecutesPOSIXShims: true}
	result := unsupportedJourneyResult(journey, "because")
	if result.Status != StatusUnsupported || result.ID != "jShim" || result.Title != "needs a shim" || result.Source != "src" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.UnsupportedSteps) != 1 || !strings.Contains(result.UnsupportedSteps[0], "because") {
		t.Fatalf("unsupported steps = %v, want the reason to reach the report", result.UnsupportedSteps)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatal(err)
	}
	_, counted, unsupported, failed := aggregate([]JourneyResult{result})
	if counted != 0 || unsupported != 1 || failed != 0 {
		t.Fatalf("counted/unsupported/failed = %d/%d/%d, want 0/1/0", counted, unsupported, failed)
	}
}

// The three journeys that EXECUTE a `#!/bin/sh` shim, found by their
// PathOverride fixtures. Journeys that merely write a shebang into a candidate
// file (wave1, wave3, edge, intended-untracked) never run it and stay portable.
func TestShimJourneysDeclareTheirPlatformLimit(t *testing.T) {
	want := map[string]bool{
		"j105-compiled-provider-capture-retries-same-binding":               true,
		"j116-codex-committed-correction-runs-returned-status-continuation": true,
		"j3043-opencode-managed-background-activation":                      true,
	}
	seen := map[string]bool{}
	for _, journey := range Journeys() {
		if journey.ExecutesPOSIXShims != want[journey.ID] {
			t.Errorf("%s: ExecutesPOSIXShims = %v, want %v", journey.ID, journey.ExecutesPOSIXShims, want[journey.ID])
		}
		seen[journey.ID] = true
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("journey %s is not registered", id)
		}
	}
}

func TestRunJourneyOnWindowsReportsShimJourneysUnsupportedWithoutRunningThem(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the unsupported declaration only applies on Windows")
	}
	ran := false
	journey := Journey{ID: "jShim", Review: reviewUntouched, ExecutesPOSIXShims: true, Steps: []Step{{
		Name:    "fixture would write a shim",
		Fixture: func(*Sandbox) error { ran = true; return nil },
	}}}
	result := runJourney("unused-binary", journey)
	if result.Status != StatusUnsupported || ran {
		t.Fatalf("status = %s, fixture ran = %v, want unsupported without running anything", result.Status, ran)
	}
}

// A journey that adds a directory to the sandbox PATH is one that ships shims,
// so it has to say so: otherwise it would run on Windows and silently fall
// back to whatever the host resolves.
func TestRunJourneyRejectsAPathOverrideTheJourneyDidNotDeclare(t *testing.T) {
	journey := Journey{ID: "jUndeclared", Review: reviewUntouched, Steps: []Step{{
		Name: "fixture adds a sandbox directory", Fixture: func(sandbox *Sandbox) error {
			sandbox.PathOverride = filepath.Join(sandbox.Root, "shims")
			return os.MkdirAll(sandbox.PathOverride, 0o755)
		},
	}}}
	result := runJourney("unused-binary", journey)
	if result.Status != StatusFailed || !strings.Contains(result.FailureReason, "ExecutesPOSIXShims") {
		t.Fatalf("status = %s, reason = %q, want a failure naming ExecutesPOSIXShims", result.Status, result.FailureReason)
	}
}
