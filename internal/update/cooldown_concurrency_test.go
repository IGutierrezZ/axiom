package update

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/statecoord"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// cooldownScenarioBudget bounds a whole two-actor scenario. It is a ceiling,
// not an expectation: a genuine deadlock still fails the test, only later.
const cooldownScenarioBudget = time.Minute

// markPendingSyncAsActorB is the concurrent writer of these scenarios: another
// actor that takes the canonical install-state lock, re-reads the latest state
// and sets PendingSync, a machine field the cooldown timestamp write must never
// revert. It fails fast, without writing, when the lock is held.
func markPendingSyncAsActorB(home string) error {
	return statecoord.WithLock(home, func() error {
		current, err := state.Read(home)
		if err != nil {
			return err
		}
		current.PendingSync = true
		return state.WriteReconciled(home, current)
	})
}

func awaitSignal(t *testing.T, ctx context.Context, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-ctx.Done():
		t.Fatalf("%s: %v", message, ctx.Err())
	}
}

func TestCheckAllWithCooldown_ConcurrentWriterPreservesPendingSync(t *testing.T) {
	home := t.TempDir()
	t.Setenv(system.EnvStateDirAxiom, filepath.Join(home, ".axiom"))
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-7 * time.Hour)
	if err := state.Write(home, state.InstallState{LastUpdateCheck: &stale}); err != nil {
		t.Fatalf("state.Write initial state: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cooldownScenarioBudget)
	defer cancel()
	persistReached := make(chan struct{}, 1)
	releasePersist := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releasePersist) }) }
	t.Cleanup(release)
	actorADone := make(chan struct{})

	go func() {
		defer close(actorADone)
		checkAllWithCooldown(
			ctx,
			"1.0.0",
			system.PlatformProfile{OS: "darwin", PackageManager: "brew"},
			home,
			6*time.Hour,
			func() time.Time { return now },
			func(context.Context, string, system.PlatformProfile) []UpdateResult {
				return []UpdateResult{{Status: UpToDate}}
			},
			func(homeDir string, timestamp time.Time) error {
				persistReached <- struct{}{}
				select {
				case <-releasePersist:
					return persistLastUpdateCheck(homeDir, timestamp)
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		)
	}()

	awaitSignal(t, ctx, persistReached, "actor A did not reach timestamp persistence before actor B sets PendingSync")

	if err := markPendingSyncAsActorB(home); err != nil {
		t.Fatalf("actor B PendingSync write failed: %v", err)
	}

	intermediate, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read after actor B: %v", err)
	}
	if !intermediate.PendingSync {
		t.Fatal("actor B did not persist PendingSync while actor A was blocked")
	}
	if intermediate.LastUpdateCheck == nil || !intermediate.LastUpdateCheck.Equal(stale) {
		t.Fatalf("actor B changed the timestamp while setting PendingSync: got %v, want %v", intermediate.LastUpdateCheck, stale)
	}

	release()
	awaitSignal(t, ctx, actorADone, "actor A did not complete after its write boundary was released")

	final, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read after actor A: %v", err)
	}
	if final.LastUpdateCheck == nil || !final.LastUpdateCheck.Equal(now) {
		t.Fatalf("last_update_check = %v, want %v after actor A writes", final.LastUpdateCheck, now)
	}
	if !final.PendingSync {
		t.Fatal("stale cooldown write reverted actor B's persisted PendingSync")
	}
}

func TestCheckAllWithCooldown_ContendedPersistenceRejectsConcurrentWriter(t *testing.T) {
	home := t.TempDir()
	t.Setenv(system.EnvStateDirAxiom, filepath.Join(home, ".axiom"))
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-7 * time.Hour)
	if err := state.Write(home, state.InstallState{LastUpdateCheck: &stale}); err != nil {
		t.Fatalf("state.Write initial state: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cooldownScenarioBudget)
	defer cancel()
	writerEntered := make(chan struct{}, 1)
	releaseWriter := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseWriter) }) }
	t.Cleanup(release)
	actorADone := make(chan struct{})

	go func() {
		defer close(actorADone)
		checkAllWithCooldown(
			ctx,
			"1.0.0",
			system.PlatformProfile{OS: "darwin", PackageManager: "brew"},
			home,
			6*time.Hour,
			func() time.Time { return now },
			func(context.Context, string, system.PlatformProfile) []UpdateResult {
				return []UpdateResult{{Status: UpToDate}}
			},
			func(homeDir string, timestamp time.Time) error {
				return persistLastUpdateCheckWithWriter(homeDir, timestamp, func(homeDir string, updated state.InstallState) error {
					writerEntered <- struct{}{}
					select {
					case <-releaseWriter:
						return state.WriteReconciled(homeDir, updated)
					case <-ctx.Done():
						return ctx.Err()
					}
				})
			},
		)
	}()

	awaitSignal(t, ctx, writerEntered, "actor A did not hold the install-state lock after updating the timestamp")
	before, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read before contended actor B: %v", err)
	}

	if err := markPendingSyncAsActorB(home); err == nil || !strings.Contains(err.Error(), "acquire install state lock") {
		t.Fatalf("actor B error while cooldown held the lock = %v, want lock acquisition failure", err)
	}

	afterAttempt, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read after contended actor B: %v", err)
	}
	if !reflect.DeepEqual(before, afterAttempt) {
		t.Fatalf("contended actor B changed persisted state: before %#v, after %#v", before, afterAttempt)
	}

	release()
	awaitSignal(t, ctx, actorADone, "actor A did not complete after its write boundary was released")

	afterActorA, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read after actor A: %v", err)
	}
	if afterActorA.PendingSync || afterActorA.LastUpdateCheck == nil || !afterActorA.LastUpdateCheck.Equal(now) {
		t.Fatalf("actor A state = %#v, want PendingSync unset with timestamp %v", afterActorA, now)
	}

	if err := markPendingSyncAsActorB(home); err != nil {
		t.Fatalf("actor B retry PendingSync write failed: %v", err)
	}

	final, err := state.Read(home)
	if err != nil {
		t.Fatalf("state.Read after actor B retry: %v", err)
	}
	if !final.PendingSync || final.LastUpdateCheck == nil || !final.LastUpdateCheck.Equal(now) {
		t.Fatalf("final state = %#v, want PendingSync set with timestamp %v", final, now)
	}
}
