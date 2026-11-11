//go:build unix || windows

package reviewtransaction

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// #4670: the clone-local override can only disable. When the deciding source
// computed without it (project, then global, then the default) is not on,
// review is off whatever the override holds, so an override the common
// directory refuses to yield must not fail resolution. When that source is on
// the override can still decide, and its failure keeps failing closed.
func TestResolveRDDModeOverrideReadFailureOnlyFailsClosedWhenTheRemainingSourceIsOn(t *testing.T) {
	tests := []struct {
		project, global string
		wantFailClosed  bool
		wantSource      RDDModeSource
	}{
		{"", "", false, RDDModeSourceDefault},
		{"", "off", false, RDDModeSourceGlobal},
		{"", "on", true, ""},
		{"off", "", false, RDDModeSourceProject},
		{"off", "off", false, RDDModeSourceProject},
		{"off", "on", false, RDDModeSourceProject},
		{"on", "", true, ""},
		{"on", "off", true, ""},
		{"on", "on", true, ""},
	}
	for _, tt := range tests {
		t.Run("project="+tt.project+"/global="+tt.global, func(t *testing.T) {
			repo := initSnapshotRepo(t)
			ctx := context.Background()
			if _, err := SetCloneLocalRDDMode(ctx, repo, RDDModeOff, "", RDDGlobalMode{}); err != nil {
				t.Fatalf("disable clone-local mode: %v", err)
			}
			unvalidatableGitCommonDir(t)

			status, err := ResolveRDDMode(ctx, repo, RDDGlobalMode{Project: tt.project, Value: tt.global})
			if tt.wantFailClosed {
				if !errors.Is(err, errUnsafeRARAuthorityPath) {
					t.Fatalf("err = %v, want the unsafe-path refusal", err)
				}
				if status.Enabled() {
					t.Fatalf("unvalidatable override did not fail closed: %#v", status)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolution failed although the remaining source is not on: %v", err)
			}
			if status.Effective != RDDModeOff || status.Source != tt.wantSource {
				t.Fatalf("status = %#v, want off decided by %s", status, tt.wantSource)
			}
			if status.CloneLocal != RDDModeUnset || status.Revision != "" {
				t.Fatalf("an unread override leaked into the status: %#v", status)
			}
		})
	}
}

// The tolerance covers the override alone: an unintelligible project or global
// value is still refused, and the strict resolver never tolerates anything.
func TestResolveRDDModeOverrideToleranceDoesNotHideOtherFailures(t *testing.T) {
	repo := initSnapshotRepo(t)
	ctx := context.Background()
	if _, err := SetCloneLocalRDDMode(ctx, repo, RDDModeOff, "", RDDGlobalMode{}); err != nil {
		t.Fatalf("disable clone-local mode: %v", err)
	}
	unvalidatableGitCommonDir(t)

	for _, global := range []RDDGlobalMode{{Project: "bogus"}, {Value: "bogus"}} {
		if _, err := ResolveRDDMode(ctx, repo, global); !errors.Is(err, ErrRDDModeUnknown) {
			t.Fatalf("ResolveRDDMode(%#v) err = %v, want the unknown-mode refusal", global, err)
		}
	}
	for _, global := range []RDDGlobalMode{{}, {Value: "off"}, {Project: "off"}} {
		if _, err := ResolveRDDModeStrict(ctx, repo, global); !errors.Is(err, errUnsafeRARAuthorityPath) {
			t.Fatalf("ResolveRDDModeStrict(%#v) err = %v, want the unsafe-path refusal", global, err)
		}
	}
}

// A clone that never recorded an override has nothing under its common
// directory to validate, so reading "no override" must not depend on the
// directory passing validation.
func TestResolveRDDModeWithoutAnOverrideNeverValidatesTheCommonDir(t *testing.T) {
	tests := []struct {
		project, global string
		wantEff         RDDMode
		wantSource      RDDModeSource
	}{
		{"", "", RDDModeOff, RDDModeSourceDefault},
		{"", "on", RDDModeOn, RDDModeSourceGlobal},
		{"", "off", RDDModeOff, RDDModeSourceGlobal},
		{"on", "", RDDModeOn, RDDModeSourceProject},
		{"off", "on", RDDModeOff, RDDModeSourceProject},
	}
	for _, tt := range tests {
		t.Run("project="+tt.project+"/global="+tt.global, func(t *testing.T) {
			repo := initSnapshotRepo(t)
			unvalidatableGitCommonDir(t)

			status, err := ResolveRDDMode(context.Background(), repo, RDDGlobalMode{Project: tt.project, Value: tt.global})
			if err != nil {
				t.Fatalf("resolution without an override validated the common dir: %v", err)
			}
			if status.Effective != tt.wantEff || status.Source != tt.wantSource || status.Revision != "" {
				t.Fatalf("status = %#v, want %s decided by %s", status, tt.wantEff, tt.wantSource)
			}
			if path, err := CloneLocalRDDModeRecordPath(context.Background(), repo); err != nil || path != "" {
				t.Fatalf("record path = %q, %v; want no record and no error", path, err)
			}
		})
	}
}

// Writes never take the read-path shortcut: recording an override still
// validates the common directory, whatever the other sources say.
func TestSetCloneLocalRDDModeStillValidatesAnUnvalidatableCommonDir(t *testing.T) {
	for _, global := range []RDDGlobalMode{{}, {Value: "off"}, {Project: "off"}} {
		repo := initSnapshotRepo(t)
		unvalidatableGitCommonDir(t)

		_, err := SetCloneLocalRDDMode(context.Background(), repo, RDDModeOff, "", global)
		if !errors.Is(err, errUnsafeRARAuthorityPath) {
			t.Fatalf("global %#v: err = %v, want the unsafe-path refusal", global, err)
		}
		switchRoot := filepath.Join(repo, ".git", "gentle-ai", rddModeSwitchDirectory, rarAuthorityDirectory)
		if _, statErr := os.Lstat(switchRoot); !os.IsNotExist(statErr) {
			t.Fatalf("a refused write created %s: %v", switchRoot, statErr)
		}
	}
}

// An override that predates the switch relocation is still an override: its
// presence alone sends the read through full validation.
func TestResolveRDDModeLegacyOverrideStillValidatesTheCommonDir(t *testing.T) {
	repo := initSnapshotRepo(t)
	legacy := filepath.Join(repo, ".git", "gentle-ai", rddModeLegacySwitchDirectory, rarAuthorityDirectory, rarAuthorityVersion, rddModeDirectory)
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	unvalidatableGitCommonDir(t)

	if _, err := ResolveRDDMode(context.Background(), repo, RDDGlobalMode{Project: "on"}); !errors.Is(err, errUnsafeRARAuthorityPath) {
		t.Fatalf("err = %v, want the unsafe-path refusal", err)
	}
}

// The clone-scope compare-and-set token is read with the strict resolver: a
// global OFF must not hide the head a disable --scope clone has to replace.
func TestSetCloneLocalRDDModeKeepsItsRevisionWhileTheOtherSourcesAreOff(t *testing.T) {
	repo := initSnapshotRepo(t)
	ctx := context.Background()
	first, err := SetCloneLocalRDDMode(ctx, repo, RDDModeOff, "", RDDGlobalMode{})
	if err != nil {
		t.Fatalf("disable clone-local mode: %v", err)
	}
	off := RDDGlobalMode{Value: "off"}
	status, err := ResolveRDDMode(ctx, repo, off)
	if err != nil || status.Source != RDDModeSourceCloneLocal || status.Revision != first.Revision {
		t.Fatalf("readable override under global off = %#v, %v; want clone_local at %q", status, err, first.Revision)
	}
	again, err := SetCloneLocalRDDMode(ctx, repo, RDDModeOff, first.Revision, off)
	if err != nil || again.Revision != first.Revision {
		t.Fatalf("idempotent disable under global off = %#v, %v", again, err)
	}
}
