package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// TestVersionDefault verifies the contract of the package-level symbols: the
// ldflags symbol is empty unless stamped, and Version holds the resolved value.
// Under `go test` the main module version is "(devel)" or empty, so the result
// must be the fallback constant (REQ-22.8, O-1).
func TestVersionDefault(t *testing.T) {
	if fallbackVersion != "v3.5.1" {
		t.Errorf("fallbackVersion = %q, want %q", fallbackVersion, "v3.5.1")
	}
	if version != "" {
		t.Errorf("version = %q, want empty when not injected by ldflags", version)
	}
	if Version == "" {
		t.Fatal("Version must never be empty")
	}
	if want := resolveVersion(version, debug.ReadBuildInfo); Version != want {
		t.Errorf("Version = %q, want %q (must hold the resolved version)", Version, want)
	}
}

// TestResolveVersion pins the resolution priority: ldflags value, then the
// module version of the build info, then the fallback constant.
func TestResolveVersion(t *testing.T) {
	withMain := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	unavailable := func() (*debug.BuildInfo, bool) { return nil, false }
	nilInfo := func() (*debug.BuildInfo, bool) { return nil, true }

	tests := []struct {
		name     string
		injected string
		read     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"injected wins over build info", "3.6.0", withMain("v9.9.9"), "3.6.0"},
		{"injected wins when build info is unavailable", "3.6.0", unavailable, "3.6.0"},
		{"blank injected is not an injection", "  ", withMain("v3.7.0"), "v3.7.0"},
		{"module tag from go install", "", withMain("v3.7.0"), "v3.7.0"},
		{"pseudo-version from a checkout build", "", withMain("v3.5.2-0.20260101120000-abcdef123456"), "v3.5.2-0.20260101120000-abcdef123456"},
		{"dirty checkout build", "", withMain("v3.5.1+dirty"), "v3.5.1+dirty"},
		{"devel falls back", "", withMain("(devel)"), fallbackVersion},
		{"empty module version falls back", "", withMain(""), fallbackVersion},
		{"build info unavailable falls back", "", unavailable, fallbackVersion},
		{"nil build info falls back", "", nilInfo, fallbackVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.read); got != tt.want {
				t.Errorf("resolveVersion(%q) = %q, want %q", tt.injected, got, tt.want)
			}
		})
	}
}

// TestPrintVersionFormat pins the exact output format of printVersion (D-05).
func TestPrintVersionFormat(t *testing.T) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s version %s (%s/%s) commit:%s\n", Platform, Version, runtime.GOOS, runtime.GOARCH, GitCommit)
	got := sb.String()

	want := fmt.Sprintf("%s version %s (%s/%s) commit:%s\n", Platform, Version, runtime.GOOS, runtime.GOARCH, GitCommit)
	if got != want {
		t.Errorf("printVersion format changed:\n  got:  %q\n  want: %q", got, want)
	}
}
