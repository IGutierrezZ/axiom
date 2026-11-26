package app

import (
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/statecoord"
)

// concurrentCheck is the fixed LastUpdateCheck a concurrent writer leaves in the
// state; it must survive every write site under test.
var concurrentCheck = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// holdCanonicalInstallStateLock acquires the real canonical install-state lock
// for home so a subsequent write site fails fast on contention, mirroring the
// contention shape used by internal/cli/install_state_lock_test.go.
func holdCanonicalInstallStateLock(t *testing.T, home string) *fileprim.Lock {
	t.Helper()
	lockPath, err := statecoord.LockPath(home)
	if err != nil {
		t.Fatalf("install state lock path: %v", err)
	}
	held, err := fileprim.AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquire install state lock for contention: %v", err)
	}
	return held
}

// TestClearPendingSyncAfterDeferredSyncReReadsLatestStateUnderLock proves the
// deferred-sync PendingSync clear (site 1, #1809) no longer clobbers a
// concurrent writer: the read happens inside the canonical lock, so a field
// written by another writer after the deferred sync survives the clear.
func TestClearPendingSyncAfterDeferredSyncReReadsLatestStateUnderLock(t *testing.T) {
	home := t.TempDir()
	held := holdCanonicalInstallStateLock(t, home)
	if err := clearPendingSyncAfterDeferredSync(home, state.InstallState{PendingSync: true}); err == nil || !strings.Contains(err.Error(), "acquire install state lock") {
		t.Fatalf("contended deferred-sync clear error = %v, want lock acquisition failure", err)
	}
	if err := state.Write(home, state.InstallState{PendingSync: true, LastUpdateCheck: &concurrentCheck, InstalledBinaryVersion: "9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	if err := clearPendingSyncAfterDeferredSync(home, state.InstallState{PendingSync: true}); err != nil {
		t.Fatalf("clearPendingSyncAfterDeferredSync() error = %v", err)
	}
	got, err := state.Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if got.PendingSync {
		t.Errorf("PendingSync = true, want false after deferred sync")
	}
	if got.LastUpdateCheck == nil || !got.LastUpdateCheck.Equal(concurrentCheck) || got.InstalledBinaryVersion != "9.9.9" {
		t.Fatalf("concurrent writer state was clobbered: %#v", got)
	}
}

// TestMarkPendingSyncAfterSelfUpdateReReadsLatestStateUnderLock proves the
// self-update PendingSync set (site 3, #1809) re-reads the latest state inside
// the canonical lock instead of writing a stale pre-lock read, so a
// concurrent writer's changes survive.
func TestMarkPendingSyncAfterSelfUpdateReReadsLatestStateUnderLock(t *testing.T) {
	home := t.TempDir()
	held := holdCanonicalInstallStateLock(t, home)
	if err := markPendingSyncAfterSelfUpdate(home); err == nil || !strings.Contains(err.Error(), "acquire install state lock") {
		t.Fatalf("contended self-update PendingSync set error = %v, want lock acquisition failure", err)
	}
	if err := state.Write(home, state.InstallState{LastUpdateCheck: &concurrentCheck, InstalledBinaryVersion: "9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	if err := markPendingSyncAfterSelfUpdate(home); err != nil {
		t.Fatalf("markPendingSyncAfterSelfUpdate() error = %v", err)
	}
	got, err := state.Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PendingSync {
		t.Errorf("PendingSync = false, want true after self-update")
	}
	if got.LastUpdateCheck == nil || !got.LastUpdateCheck.Equal(concurrentCheck) || got.InstalledBinaryVersion != "9.9.9" {
		t.Fatalf("concurrent writer state was clobbered: %#v", got)
	}
}
