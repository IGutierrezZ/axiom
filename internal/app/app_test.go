package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/update"
	"github.com/IGutierrezZ/axiom/v3/internal/update/upgrade"
)

// TestListBackupsNewestFirst verifies that ListBackups returns manifests sorted
// newest-first by CreatedAt timestamp, matching the spec "newest first" ordering.
func TestListBackupsNewestFirst(t *testing.T) {
	home := t.TempDir()
	backupRoot := backup.LegacyBackupRootFor(home)

	older := backup.Manifest{
		ID:        "older",
		CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
		RootDir:   filepath.Join(backupRoot, "older"),
		Entries:   []backup.ManifestEntry{},
	}
	newer := backup.Manifest{
		ID:        "newer",
		CreatedAt: time.Date(2026, 3, 22, 15, 4, 5, 0, time.UTC),
		RootDir:   filepath.Join(backupRoot, "newer"),
		Entries:   []backup.ManifestEntry{},
	}

	// Write older backup first.
	for _, m := range []backup.Manifest{older, newer} {
		dir := filepath.Join(backupRoot, m.ID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := backup.WriteManifest(filepath.Join(dir, backup.ManifestFilename), m); err != nil {
			t.Fatalf("WriteManifest: %v", err)
		}
	}

	// Temporarily override home dir resolution for ListBackups.
	setupMockHome(t, home)

	manifests := ListBackups()

	if len(manifests) != 2 {
		t.Fatalf("ListBackups() returned %d manifests, want 2", len(manifests))
	}

	// Newest must be first.
	if manifests[0].ID != "newer" {
		t.Errorf("ListBackups()[0].ID = %q, want %q (newest first)", manifests[0].ID, "newer")
	}
	if manifests[1].ID != "older" {
		t.Errorf("ListBackups()[1].ID = %q, want %q", manifests[1].ID, "older")
	}
}

// TestListBackupsWithSourceMetadata verifies that ListBackups returns manifests
// with Source metadata intact, so display labels can use the source field.
func TestListBackupsWithSourceMetadata(t *testing.T) {
	home := t.TempDir()
	backupRoot := backup.LegacyBackupRootFor(home)

	m := backup.Manifest{
		ID:          "test-with-source",
		CreatedAt:   time.Now().UTC(),
		RootDir:     filepath.Join(backupRoot, "test-with-source"),
		Source:      backup.BackupSourceInstall,
		Description: "pre-install snapshot",
		Entries:     []backup.ManifestEntry{},
	}

	dir := filepath.Join(backupRoot, m.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := backup.WriteManifest(filepath.Join(dir, backup.ManifestFilename), m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	setupMockHome(t, home)

	manifests := ListBackups()

	if len(manifests) != 1 {
		t.Fatalf("ListBackups() returned %d manifests, want 1", len(manifests))
	}

	got := manifests[0]
	if got.Source != backup.BackupSourceInstall {
		t.Errorf("Source = %q, want %q", got.Source, backup.BackupSourceInstall)
	}
	if got.Description != "pre-install snapshot" {
		t.Errorf("Description = %q, want %q", got.Description, "pre-install snapshot")
	}
}

func TestListBackupsKeepsDuplicateIDsRestorableBySelectedRoot(t *testing.T) {
	home := t.TempDir()
	canonicalRoot := backup.BackupRootFor(home)
	legacyRoot := backup.LegacyBackupRootFor(home)
	sourcePath := filepath.Join(home, "config.json")

	createSnapshot := func(root, contents string) backup.Manifest {
		t.Helper()
		if err := os.WriteFile(sourcePath, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", sourcePath, err)
		}
		manifest, err := backup.NewSnapshotter().Create(filepath.Join(root, "shared-id"), []string{sourcePath})
		if err != nil {
			t.Fatalf("Snapshotter.Create(%q): %v", root, err)
		}
		return manifest
	}

	axiomSnapshot := createSnapshot(canonicalRoot, "Axiom snapshot\n")
	legacySnapshot := createSnapshot(legacyRoot, "Gentle AI snapshot\n")
	if axiomSnapshot.ID != legacySnapshot.ID {
		t.Fatalf("test setup IDs differ: Axiom=%q legacy=%q", axiomSnapshot.ID, legacySnapshot.ID)
	}
	if err := os.WriteFile(sourcePath, []byte("current configuration\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current): %v", err)
	}

	setupMockHome(t, home)
	manifests := ListBackups()
	if len(manifests) != 2 {
		t.Fatalf("ListBackups() returned %d manifests, want both same-ID roots", len(manifests))
	}
	byOrigin := make(map[backup.BackupOrigin]backup.Manifest, len(manifests))
	for _, manifest := range manifests {
		if manifest.ID != axiomSnapshot.ID {
			t.Errorf("listed ID = %q, want shared ID %q", manifest.ID, axiomSnapshot.ID)
		}
		byOrigin[manifest.Origin] = manifest
	}
	axiom, hasAxiom := byOrigin[backup.BackupOriginAxiom]
	legacy, hasLegacy := byOrigin[backup.BackupOriginGentleAI]
	if !hasAxiom || !hasLegacy {
		t.Fatalf("ListBackups() origins = %v, want both Axiom and Gentle AI", byOrigin)
	}
	if axiom.RootDir == legacy.RootDir {
		t.Fatalf("same-ID backups share restore path %q; expected root-specific identity", axiom.RootDir)
	}

	restoreSelected := func(selected backup.Manifest, want string) {
		t.Helper()
		if err := (backup.RestoreService{}).Restore(selected); err != nil {
			t.Fatalf("restoring %s backup failed: %v", selected.Origin, err)
		}
		contents, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("ReadFile(restored config): %v", err)
		}
		if string(contents) != want {
			t.Fatalf("restored %s snapshot = %q, want %q", selected.Origin, contents, want)
		}
	}

	restoreSelected(axiom, "Axiom snapshot\n")
	restoreSelected(legacy, "Gentle AI snapshot\n")
}

// TestRunArgsRestoreListIsDispatched verifies that `gentle-ai restore --list`
// is correctly dispatched through RunArgs and produces a meaningful response
// (either a backup list or a "no backups" message — never "unknown command").
func TestRunArgsRestoreListIsDispatched(t *testing.T) {
	home := t.TempDir()
	setupMockHome(t, home)

	var buf bytes.Buffer
	err := RunArgs([]string{"restore", "--list"}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(restore --list) error = %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatalf("restore --list produced no output")
	}

	// Must not produce "unknown command".
	if strings.Contains(out, "unknown command") {
		t.Errorf("restore is not registered in RunArgs; got: %s", out)
	}
}

// TestRunArgsRestoreByIDWithYes verifies end-to-end wiring of `restore <id> --yes`
// through app.RunArgs.
func TestRunArgsRestoreByIDWithYes(t *testing.T) {
	home := t.TempDir()
	backupRoot := backup.LegacyBackupRootFor(home)

	// Create a backup with a real file entry so restore can succeed.
	sourceFile := filepath.Join(home, "config.md")
	if err := os.WriteFile(sourceFile, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	snapshotDir := filepath.Join(backupRoot, "test-backup-001")
	snapshotFile := filepath.Join(snapshotDir, "files", "config.md")
	if err := os.MkdirAll(filepath.Dir(snapshotFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(snapshotFile, []byte("backup-content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile snapshot: %v", err)
	}

	m := backup.Manifest{
		ID:        "test-backup-001",
		CreatedAt: time.Now().UTC(),
		RootDir:   snapshotDir,
		Source:    backup.BackupSourceInstall,
		Entries: []backup.ManifestEntry{
			{OriginalPath: sourceFile, SnapshotPath: snapshotFile, Existed: true, Mode: 0o644},
		},
	}
	if err := backup.WriteManifest(filepath.Join(snapshotDir, backup.ManifestFilename), m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	setupMockHome(t, home)

	// Integration evidence for [D-12]: a legacy-root backup must still be
	// discoverable through ListBackups before this increment's writer
	// migration is exercised anywhere else in this test — the dual-root read
	// contract from INC-12 is unaffected by REQ-20.13/REQ-20.14.
	found := false
	for _, listed := range ListBackups() {
		if listed.ID == m.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ListBackups() did not resolve legacy-root backup %q — dual-root read contract [D-12] broken", m.ID)
	}

	var buf bytes.Buffer
	err := RunArgs([]string{"restore", "test-backup-001", "--yes"}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(restore test-backup-001 --yes) error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(strings.ToLower(out), "restor") {
		t.Errorf("restore output should confirm restoration; got:\n%s", out)
	}

	// Integration evidence for [D-12]: backup.DeleteBackup must still remove
	// a legacy-root backup after this increment's writer migration.
	if err := backup.DeleteBackup(m); err != nil {
		t.Fatalf("DeleteBackup() on legacy-root backup %q error = %v", m.ID, err)
	}
	if _, statErr := os.Stat(snapshotDir); !os.IsNotExist(statErr) {
		t.Fatalf("DeleteBackup() did not remove legacy-root backup directory %q: %v", snapshotDir, statErr)
	}
}

// TestRunArgsRestoreUnknownIDReturnsError verifies that an unknown backup ID
// is surfaced as an error from RunArgs.
func TestRunArgsRestoreUnknownIDReturnsError(t *testing.T) {
	home := t.TempDir()
	setupMockHome(t, home)

	var buf bytes.Buffer
	err := RunArgs([]string{"restore", "no-such-backup", "--yes"}, &buf)
	if err == nil {
		t.Fatalf("RunArgs(restore no-such-backup) expected error")
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Errorf("restore returned 'unknown command' — not dispatched: %v", err)
	}
}

func TestRunArgsUninstallIsDispatched(t *testing.T) {
	var buf bytes.Buffer
	// uninstall without required flags prints usage help — that's enough to
	// confirm the dispatch path works without needing real agents or state.
	_ = RunArgs([]string{"uninstall"}, &buf)
	// If we got here without panic, the dispatch to cli.RunUninstall works.
}

func TestRunArgsUninstallBypassesPlatformValidation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error {
		return fmt.Errorf("unsupported platform")
	}

	var buf bytes.Buffer
	// uninstall should NOT call ensureCurrentOSSupported — it runs before
	// the platform check in the switch.
	_ = RunArgs([]string{"uninstall"}, &buf)
	// If we got here, uninstall bypassed the platform validation.
}

func TestRunArgsUninstallReportsBackupOnFailure(t *testing.T) {
	home := t.TempDir()
	setupMockHome(t, home)
	t.Chdir(t.TempDir())

	statePath := state.Path(home)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err := RunArgs([]string{"uninstall", "--agent", "codex", "--scope", "global", "--yes"}, &output)
	if err == nil || !strings.Contains(err.Error(), "read install state") {
		t.Fatalf("RunArgs(uninstall) error = %v, want install state read failure", err)
	}
	for _, want := range []string{"Backup:", "Backup path:"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("RunArgs(uninstall) output missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunArgsInstallHelpPrintsInstallSpecificHelp(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error {
		return fmt.Errorf("platform validation should not run for install help")
	}

	var buf bytes.Buffer
	err := RunArgs([]string{"install", "--help"}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(install --help) error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"--channel", "beta", "nightly", "GENTLE_AI_CHANNEL"} {
		if !strings.Contains(out, want) {
			t.Fatalf("install help missing %q; output:\n%s", want, out)
		}
	}
}

func TestRunArgsSDDStatusIsDispatchedBeforePlatformValidation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error {
		return fmt.Errorf("unsupported platform")
	}

	root := t.TempDir()
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "proposal.md"), "# Proposal\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "specs", "auth", "spec.md"), "# Spec\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "design.md"), "# Design\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "tasks.md"), "- [ ] 1.1 Work\n")

	var buf bytes.Buffer
	err := RunArgs([]string{"sdd-status", "add-auth", "--cwd", root}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(sdd-status) error = %v", err)
	}
	if !strings.Contains(buf.String(), "## SDD Status: add-auth") {
		t.Fatalf("sdd-status output missing markdown status:\n%s", buf.String())
	}
}

func TestRunArgsSDDAttemptIsDispatchedBeforePlatformValidation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error { return fmt.Errorf("unsupported platform") }

	var output bytes.Buffer
	err := RunArgs([]string{"sdd-attempt", "grant"}, &output)
	if err == nil || !strings.Contains(err.Error(), "sdd-attempt grant requires") || strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("grant was not dispatched before platform validation: %v", err)
	}

}

func TestRunArgsSDDAttemptHelpBypassesPlatformAndRepositoryValidation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error { return fmt.Errorf("platform validation should not run for sdd-attempt help") }

	var output bytes.Buffer
	err := RunArgs([]string{"sdd-attempt", "grant", "--cwd", filepath.Join(t.TempDir(), "missing"), "--change", "missing", "--help"}, &output)
	if err != nil {
		t.Fatalf("RunArgs(sdd-attempt grant --help): %v", err)
	}
	for _, want := range []string{"Usage: axiom sdd-attempt grant [flags]", "-root value", "repeatable"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("sdd-attempt grant help missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunArgsSDDAttemptParentHelpDoesNotSelectChangeValueAsOperation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error { return fmt.Errorf("platform validation should not run for sdd-attempt help") }

	var output bytes.Buffer
	err := RunArgs([]string{"sdd-attempt", "--help", "--cwd", "/definitely/not/a/repository", "--change", "begin"}, &output)
	if err != nil {
		t.Fatalf("RunArgs(sdd-attempt --help --cwd /definitely/not/a/repository --change begin): %v", err)
	}
	if !strings.Contains(output.String(), "Usage: axiom sdd-attempt grant [flags]") || strings.Contains(output.String(), "Usage: axiom sdd-attempt begin [flags]") {
		t.Fatalf("sdd-attempt parent help =\n%s", output.String())
	}
}

func TestRunArgsSDDContinueIsDispatchedBeforePlatformValidation(t *testing.T) {
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() { ensureCurrentOSSupported = origEnsure })
	ensureCurrentOSSupported = func() error {
		return fmt.Errorf("unsupported platform")
	}

	root := t.TempDir()
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "proposal.md"), "# Proposal\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "specs", "auth", "spec.md"), "# Spec\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "design.md"), "# Design\n")
	writeAppSDDStatusFile(t, filepath.Join(root, "openspec", "changes", "add-auth", "tasks.md"), "- [ ] 1.1 Work\n")

	var buf bytes.Buffer
	err := RunArgs([]string{"sdd-continue", "add-auth", "--cwd", root}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(sdd-continue) error = %v", err)
	}
	if !strings.Contains(buf.String(), "## Native SDD Dispatcher: add-auth") {
		t.Fatalf("sdd-continue output missing dispatcher markdown:\n%s", buf.String())
	}
}

// TestListBackupsFallsBackGracefullyForOldManifests verifies that old manifests
// without Source/Description are still returned (not skipped) and can be displayed
// via DisplayLabel without panicking.
func TestListBackupsFallsBackGracefullyForOldManifests(t *testing.T) {
	_ = fmt.Sprintf // Ensure fmt is used.
	home := t.TempDir()
	backupRoot := backup.LegacyBackupRootFor(home)

	// Write a manifest with no Source/Description.
	m := backup.Manifest{
		ID:        "old-backup",
		CreatedAt: time.Now().UTC(),
		RootDir:   filepath.Join(backupRoot, "old-backup"),
		Entries:   []backup.ManifestEntry{},
		// Source and Description intentionally omitted — simulates old manifest.
	}

	dir := filepath.Join(backupRoot, m.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := backup.WriteManifest(filepath.Join(dir, backup.ManifestFilename), m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	setupMockHome(t, home)

	manifests := ListBackups()

	if len(manifests) != 1 {
		t.Fatalf("ListBackups() returned %d manifests, want 1", len(manifests))
	}

	// Must not panic — DisplayLabel should handle empty Source gracefully.
	label := manifests[0].DisplayLabel()
	if label == "" {
		t.Errorf("DisplayLabel() returned empty string, want non-empty fallback label")
	}
}

// ─── BUG 3: SyncOverrides.StrictTDD never read al sincronizar ───────────────────

func boolPtr(b bool) *bool { return &b }

func TestDeferredSyncIncludesCodexPermissionsArgs(t *testing.T) {
	// syncArgsForDiscoveredAgents resolves the effective store from the working
	// directory: a checkout with its own project .axiom/state.json would win
	// over home and the test would read the developer's real state.
	t.Chdir(t.TempDir())
	home := t.TempDir()
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentCodex)}}); err != nil {
		t.Fatalf("state.Write: %v", err)
	}
	if args := syncArgsForDiscoveredAgents(home); len(args) != 1 || args[0] != "--include-permissions" {
		t.Fatalf("legacy sync args = %v, want permissions", args)
	}
	configured := t.TempDir()
	if err := state.Write(configured, state.InstallState{InstalledAgents: []string{"codex"}, SelectionConfigured: true, Components: []model.ComponentID{model.ComponentEngram}}); err != nil {
		t.Fatal(err)
	}
	if args := syncArgsForDiscoveredAgents(configured); len(args) != 0 {
		t.Fatalf("configured Custom sync args = %v, want none", args)
	}
}

func filesUnder(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

// ─── Persist model assignments (TUI path) ───────────────────────────────────

// TestVersionBeforeSystemGuards verifies that `gentle-ai version` returns the
// version string without going through system detection or platform guards.
func TestVersionBeforeSystemGuards(t *testing.T) {
	var buf bytes.Buffer
	err := RunArgs([]string{"version"}, &buf)
	if err != nil {
		t.Fatalf("version should not fail: %v", err)
	}
	if !strings.Contains(buf.String(), "gentle-ai") && !strings.Contains(buf.String(), "axiom") {
		t.Error("version output should contain 'axiom' or 'gentle-ai'")
	}
}

// TestHelpCommand verifies that help, --help, and -h all print USAGE and COMMANDS
// without triggering system detection or platform guards.
func TestHelpCommand(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			var buf bytes.Buffer
			err := RunArgs([]string{arg}, &buf)
			if err != nil {
				t.Fatalf("help should not fail: %v", err)
			}
			if !strings.Contains(buf.String(), "USAGE") {
				t.Errorf("help output for %q should contain USAGE", arg)
			}
			if !strings.Contains(buf.String(), "COMMANDS") {
				t.Errorf("help output for %q should contain COMMANDS", arg)
			}
		})
	}
}

// TestUnknownCommandSuggestsHelp verifies that an unrecognised command returns
// an error whose message suggests running 'axiom help' or 'gentle-ai help'.
func TestUnknownCommandSuggestsHelp(t *testing.T) {
	var buf bytes.Buffer
	err := RunArgs([]string{"notacommand"}, &buf)
	if err == nil {
		t.Fatal("unknown command should return error")
	}
	if !strings.Contains(err.Error(), "axiom help") && !strings.Contains(err.Error(), "gentle-ai help") {
		t.Error("unknown command error should suggest 'axiom help'")
	}
}

// TestRunArgsRetiredWorkCommandsNoLongerDispatch proves the remote control
// plane's command surface is gone: every retired work-* command falls through
// to the unknown-command error instead of dispatching provider machinery.
func TestRunArgsRetiredWorkCommandsNoLongerDispatch(t *testing.T) {
	origSelfUpdate := selfUpdateFn
	origDetect := detectSystem
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() {
		selfUpdateFn = origSelfUpdate
		detectSystem = origDetect
		ensureCurrentOSSupported = origEnsure
	})
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true}}, nil
	}
	selfUpdateFn = func(context.Context, string, system.PlatformProfile, io.Writer) error { return nil }

	retired := []string{
		"work-capabilities", "work-start", "work-route", "work-advance",
		"work-verification-decide", "work-reconcile", "work-status", "work-transition",
	}
	for _, command := range retired {
		t.Run(command, func(t *testing.T) {
			var buf bytes.Buffer
			err := RunArgs([]string{command}, &buf)
			if err == nil {
				t.Fatalf("RunArgs(%q) error = nil, want unknown command", command)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("unknown command %q", command)) {
				t.Fatalf("RunArgs(%q) error = %v, want unknown command", command, err)
			}
		})
	}
}

func TestRunArgs_UpdateSkipsSelfUpdate(t *testing.T) {
	origSelfUpdate := selfUpdateFn
	origCheckAll := updateCheckAll
	origDetect := detectSystem
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() {
		selfUpdateFn = origSelfUpdate
		updateCheckAll = origCheckAll
		detectSystem = origDetect
		ensureCurrentOSSupported = origEnsure
	})

	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true}}, nil
	}

	selfUpdateCalled := 0
	selfUpdateFn = func(context.Context, string, system.PlatformProfile, io.Writer) error {
		selfUpdateCalled++
		return nil
	}

	updateCheckAll = func(context.Context, string, system.PlatformProfile) []update.UpdateResult {
		return []update.UpdateResult{
			{
				Tool:             update.ToolInfo{Name: "gentle-ai"},
				InstalledVersion: "1.0.0",
				LatestVersion:    "1.0.0",
				Status:           update.UpToDate,
			},
		}
	}

	var buf bytes.Buffer
	err := RunArgs([]string{"update"}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(update) error = %v", err)
	}
	if selfUpdateCalled != 0 {
		t.Fatalf("selfUpdate should be skipped for explicit update flow; got %d call(s)", selfUpdateCalled)
	}
}

func TestRunArgs_UpgradeSkipsSelfUpdate(t *testing.T) {
	origSelfUpdate := selfUpdateFn
	origCheckFiltered := updateCheckFiltered
	origUpgradeExecute := upgradeExecute
	origDetect := detectSystem
	origEnsure := ensureCurrentOSSupported
	t.Cleanup(func() {
		selfUpdateFn = origSelfUpdate
		updateCheckFiltered = origCheckFiltered
		upgradeExecute = origUpgradeExecute
		detectSystem = origDetect
		ensureCurrentOSSupported = origEnsure
	})

	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true}}, nil
	}

	selfUpdateCalled := 0
	selfUpdateFn = func(context.Context, string, system.PlatformProfile, io.Writer) error {
		selfUpdateCalled++
		return nil
	}

	updateCheckFiltered = func(context.Context, string, system.PlatformProfile, []string) []update.UpdateResult {
		return []update.UpdateResult{
			{
				Tool:             update.ToolInfo{Name: "gentle-ai", InstallMethod: update.InstallBinary},
				InstalledVersion: "1.0.0",
				LatestVersion:    "1.0.0",
				Status:           update.UpToDate,
			},
		}
	}

	upgradeExecute = func(context.Context, []update.UpdateResult, system.PlatformProfile, string, bool, ...io.Writer) upgrade.UpgradeReport {
		return upgrade.UpgradeReport{}
	}

	var buf bytes.Buffer
	err := RunArgs([]string{"upgrade", "--dry-run"}, &buf)
	if err != nil {
		t.Fatalf("RunArgs(upgrade --dry-run) error = %v", err)
	}
	if selfUpdateCalled != 0 {
		t.Fatalf("selfUpdate should be skipped for explicit upgrade flow; got %d call(s)", selfUpdateCalled)
	}
}

func TestRunArgs_BareStartFailsClosedOnUnreadableState(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)
	if err := os.MkdirAll(filepath.Dir(state.Path(home)), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(state.Path(home), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	origDetect, origEnsure := detectSystem, ensureCurrentOSSupported
	t.Cleanup(func() {
		detectSystem = origDetect
		ensureCurrentOSSupported = origEnsure
	})
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true}}, nil
	}

	if err := RunArgs(nil, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "read install state") {
		t.Fatalf("RunArgs(bare) error = %v, want unreadable state error", err)
	}
}

func TestIsExplicitUpdateFlow(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "empty args", args: nil, want: false},
		{name: "no command", args: []string{}, want: false},
		{name: "update", args: []string{"update"}, want: true},
		{name: "upgrade", args: []string{"upgrade"}, want: true},
		{name: "version", args: []string{"version"}, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "install", args: []string{"install"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExplicitUpdateFlow(tt.args)
			if got != tt.want {
				t.Fatalf("isExplicitUpdateFlow(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func setupMockHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv(cli.ScopeAxiomEnvVar, string(cli.ScopeGlobal))
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
}

// ─── Carril model assignment tests (W-3 fix) ─────────────────────────────────

// ─── BUG-1: Custom→preset via sync path does not clear CodexPhaseModelAssignments ───

func writeAppSDDStatusFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// ─── Slice 4 RED: deferred sync on launch via pending_sync flag ───────────────

// TestRunArgs_PendingSync_RunsSyncAndClearsFlag verifies that when
// state.json has PendingSync=true, RunArgs (TUI path / no args) calls
// the deferred sync runner and writes PendingSync=false on success.
func TestRunArgs_PendingSync_RunsSyncAndClearsFlag(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	// Write initial state with PendingSync=true.
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     true,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	var syncCalled int
	deferredSyncFn = func() error {
		syncCalled++
		return nil
	}

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v", err)
	}

	if syncCalled != 1 {
		t.Errorf("deferredSyncFn called %d times, want 1", syncCalled)
	}

	// PendingSync must be cleared after successful sync.
	s, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read() error = %v", err)
	}
	if s.PendingSync {
		t.Errorf("PendingSync = true after successful deferred sync, want false")
	}
}

// TestRunArgs_PendingSync_LeavesSetOnFailure verifies that when the deferred
// sync fails, PendingSync remains true so the next launch retries idempotently.
func TestRunArgs_PendingSync_LeavesSetOnFailure(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     true,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	deferredSyncFn = func() error {
		return fmt.Errorf("sync: network error")
	}

	var buf bytes.Buffer
	// RunArgs must NOT return an error — deferred sync failure is non-fatal.
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v (deferred sync failure must be non-fatal)", err)
	}

	// The warning must be printed to stdout so the user knows sync was skipped.
	out := buf.String()
	if !strings.Contains(out, "Warning: deferred sync failed:") {
		t.Errorf("stdout = %q, want warning message for deferred sync failure", out)
	}

	// PendingSync must remain set so the next launch retries.
	s, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read() error = %v", err)
	}
	if !s.PendingSync {
		t.Errorf("PendingSync = false after failed deferred sync, want true (idempotent retry)")
	}
}

// TestRunArgs_PendingSync_ClearWriteFailureIsLogged verifies that when the
// deferred sync succeeds but state.Write (to clear PendingSync) fails, the
// error is printed to stdout and RunArgs does not return an error.
// This guards against silently swallowed write failures (Issue 2).
func TestRunArgs_PendingSync_ClearWriteFailureIsLogged(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	// Write initial state with PendingSync=true.
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     true,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	// Keep state readable through a symlink while making atomic replacement refuse it.
	stateFilePath := state.Path(home)
	stateTargetPath := filepath.Join(filepath.Dir(stateFilePath), "persisted-state.json")
	if err := os.Rename(stateFilePath, stateTargetPath); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := os.Symlink(stateTargetPath, stateFilePath); err != nil {
		t.Skipf("state symlink unavailable: %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	// Deferred sync succeeds — the write-clear is what we're testing.
	deferredSyncFn = func() error { return nil }

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v (clear-write failure must be non-fatal)", err)
	}

	// The warning must appear in stdout when the clear-write fails.
	out := buf.String()
	if !strings.Contains(out, "Warning:") {
		t.Errorf("stdout = %q; want a warning message when PendingSync clear-write fails", out)
	}
}

// TestRunArgs_NoPendingSync_NoSyncCall verifies that when PendingSync=false,
// the deferred sync runner is NOT called (no extra sync on a normal launch).
func TestRunArgs_NoPendingSync_NoSyncCall(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	// Write state without PendingSync.
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     false,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	var syncCalled int
	deferredSyncFn = func() error {
		syncCalled++
		return nil
	}

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v", err)
	}

	if syncCalled != 0 {
		t.Errorf("deferredSyncFn called %d times, want 0 (no pending sync)", syncCalled)
	}
}

// TestRunArgs_PendingSync_PrintsDoctorAdvisory verifies the TUI self-update
// deferred path: when the new binary starts and observes PendingSync=true, it
// prints the doctor advisory after the deferred sync attempt (success or
// failure). The advisory lets the user verify ecosystem health against the
// post-upgrade state. Per #1901, this reuses the existing PendingSync signal
// rather than introducing a new persisted flag.
func TestRunArgs_PendingSync_PrintsDoctorAdvisory(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     true,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	deferredSyncFn = func() error {
		return nil // successful sync
	}

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Run 'axiom doctor' to verify ecosystem health after upgrade") {
		t.Errorf("stdout = %q, want doctor advisory when PendingSync=true on launch", out)
	}
}

// TestRunArgs_PendingSync_PrintsDoctorAdvisoryEvenOnSyncFailure verifies that
// the doctor advisory is printed regardless of deferred sync outcome. The
// advisory is informational and complements the sync outcome (not a replacement).
func TestRunArgs_PendingSync_PrintsDoctorAdvisoryEvenOnSyncFailure(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     true,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	deferredSyncFn = func() error {
		return fmt.Errorf("simulated network error")
	}

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v (deferred sync failure must be non-fatal)", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Run 'axiom doctor' to verify ecosystem health after upgrade") {
		t.Errorf("stdout = %q, want doctor advisory even when deferred sync fails", out)
	}
}

// TestRunArgs_NoPendingSync_DoesNotPrintDoctorAdvisory verifies that the
// doctor advisory is NOT printed on a normal launch where PendingSync=false.
// The advisory is gated strictly on the post-upgrade signal.
func TestRunArgs_NoPendingSync_DoesNotPrintDoctorAdvisory(t *testing.T) {
	assumeInteractiveTTY(t)
	home := t.TempDir()
	setupMockHome(t, home)

	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"claude-code"},
		PendingSync:     false,
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}

	origSelf := selfUpdateFn
	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origDeferredSync := deferredSyncFn
	t.Cleanup(func() {
		selfUpdateFn = origSelf
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		deferredSyncFn = origDeferredSync
	})

	selfUpdateFn = func(_ context.Context, _ string, _ system.PlatformProfile, _ io.Writer) error {
		return nil
	}
	ensureCurrentOSSupported = func() error { return nil }
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{System: system.SystemInfo{Supported: true, Profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true}}}, nil
	}

	var buf bytes.Buffer
	if err := RunArgs(nil, &buf); err != nil {
		t.Fatalf("RunArgs(nil) error = %v", err)
	}

	if strings.Contains(buf.String(), "ecosystem health after upgrade") {
		t.Errorf("stdout must NOT contain doctor advisory on a normal launch (PendingSync=false):\n%s", buf.String())
	}
}

// ─── Issue #535: upgrade argument validation pre-effect gate ───────────────

// installUpgradeSentinels replaces every effect that the upgrade preflight
// MUST NOT reach, with stubs that fail the test if invoked. It restores the
// originals on cleanup. HOME is isolated to a temp dir so the parser cannot
// accidentally trigger real home-directory effects.
func installUpgradeSentinels(t *testing.T, home string) {
	t.Helper()
	setupMockHome(t, home)

	origEnsure := ensureCurrentOSSupported
	origDetect := detectSystem
	origSelfUpdate := selfUpdateFn
	origCheckFiltered := updateCheckFiltered
	origCheckAll := updateCheckAll
	origUpgradeExecute := upgradeExecute
	origUpgradeExecuteWithOptions := upgradeExecuteWithOptions
	t.Cleanup(func() {
		ensureCurrentOSSupported = origEnsure
		detectSystem = origDetect
		selfUpdateFn = origSelfUpdate
		updateCheckFiltered = origCheckFiltered
		updateCheckAll = origCheckAll
		upgradeExecute = origUpgradeExecute
		upgradeExecuteWithOptions = origUpgradeExecuteWithOptions
	})

	ensureCurrentOSSupported = func() error {
		return fmt.Errorf("ensureCurrentOSSupported must not run for this upgrade invocation")
	}
	detectSystem = func(context.Context) (system.DetectionResult, error) {
		return system.DetectionResult{}, fmt.Errorf("detectSystem must not run for this upgrade invocation")
	}
	selfUpdateFn = func(context.Context, string, system.PlatformProfile, io.Writer) error {
		return fmt.Errorf("selfUpdate must not run for this upgrade invocation")
	}
	updateCheckFiltered = func(context.Context, string, system.PlatformProfile, []string) []update.UpdateResult {
		t.Fatalf("updateCheckFiltered must not run for this upgrade invocation")
		return nil
	}
	updateCheckAll = func(context.Context, string, system.PlatformProfile) []update.UpdateResult {
		t.Fatalf("updateCheckAll must not run for this upgrade invocation")
		return nil
	}
	upgradeExecute = func(context.Context, []update.UpdateResult, system.PlatformProfile, string, bool, ...io.Writer) upgrade.UpgradeReport {
		t.Fatalf("upgradeExecute must not run for this upgrade invocation")
		return upgrade.UpgradeReport{}
	}
	upgradeExecuteWithOptions = func(context.Context, []update.UpdateResult, system.PlatformProfile, string, bool, upgrade.ExecuteOptions) upgrade.UpgradeReport {
		t.Fatalf("upgradeExecuteWithOptions must not run for this upgrade invocation")
		return upgrade.UpgradeReport{}
	}
}

// TestRunArgs_UpgradeUnsupportedOptionStopsBeforeAnyEffect proves unsupported
// dash args (--verbose) and the #535 remediation (--help, -h) are rejected
// before every effect: identifiable token, errors.Is, zero effects.
func TestRunArgs_UpgradeUnsupportedOptionStopsBeforeAnyEffect(t *testing.T) {
	for _, token := range []string{"--verbose", "--help", "-h"} {
		t.Run(token, func(t *testing.T) {
			installUpgradeSentinels(t, t.TempDir())
			var buf bytes.Buffer
			err := RunArgs([]string{"upgrade", token}, &buf)
			if err == nil {
				t.Fatalf("RunArgs(upgrade %s) error = nil, want error", token)
			}
			want := fmt.Sprintf(`unsupported upgrade argument: %q`, token)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("RunArgs(upgrade %s) error = %v, want %q", token, err, want)
			}
			if !errors.Is(err, errUnsupportedUpgradeArgument) {
				t.Fatalf("RunArgs(upgrade %s) error not identifiable: %v", token, err)
			}
		})
	}
}

func TestRunArgsRefusesTheRetiredReviewCommands(t *testing.T) {
	for _, args := range [][]string{
		{"review", "--help"},
		{"review", "mode", "status"},
		{"review-start"},
		{"review-resume"},
		{"review-step"},
		{"review-bundle-export"},
		{"review-bundle-import"},
		{"review-validate"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var output bytes.Buffer
			err := RunArgs(args, &output)
			if err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("RunArgs(%v) error = %v, want an unknown command refusal", args, err)
			}
		})
	}
}
