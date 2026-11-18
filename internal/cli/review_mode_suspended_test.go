package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// suspendRDD apaga el punto de inyección para el test: RDD suspendido (C-01-c).
func suspendRDD(t *testing.T) {
	t.Helper()
	restore := rddharness.SetForTesting(false)
	t.Cleanup(restore)
	if rddharness.Enabled() {
		t.Skip("binario compilado con rdd_harness: el apagado no aplica")
	}
}

func TestReviewModeEnableFailsAtEveryScopeWhileSuspended(t *testing.T) {
	suspendRDD(t)
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	for _, scope := range []string{"global", "project", "clone"} {
		var output bytes.Buffer
		err := RunReviewMode([]string{"enable", "--scope", scope, "--cwd", repo, "--json"}, &output)
		if err == nil || !strings.Contains(err.Error(), "RDD no está disponible") {
			t.Fatalf("enable --scope %s: error = %v; quiere «RDD no está disponible»", scope, err)
		}
	}
	if _, err := os.Lstat(state.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("enable escribió el estado de usuario: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "gentle-ai")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("enable creó estado del repositorio: %v", err)
	}
}

func TestReviewModeSettersRejectOnWhileSuspended(t *testing.T) {
	suspendRDD(t)
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	if _, err := SetGlobalReviewMode(t.Context(), repo, true); err == nil {
		t.Fatal("SetGlobalReviewMode(true) debe fallar")
	}
	if _, err := SetProjectReviewMode(t.Context(), repo, "on", nil); err == nil {
		t.Fatal("SetProjectReviewMode(on) debe fallar")
	}
	if _, err := os.Lstat(state.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("los setters escribieron el estado de usuario: %v", err)
	}
}

func TestReviewModeDisableAndStatusStillWorkWhileSuspended(t *testing.T) {
	suspendRDD(t)
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"disable", "--scope", "global", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("disable global: %v", err)
	}
	output.Reset()
	if err := RunReviewMode([]string{"status", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("status: %v", err)
	}
	result := decodeReviewModeResult(t, output.Bytes())
	if result.Status.Effective != reviewtransaction.RDDModeOff {
		t.Fatalf("status = %#v; quiere off", result.Status)
	}
}

func TestReviewModeStatusIsOffWithStoredOnWhileSuspended(t *testing.T) {
	suspendRDD(t)
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	recordedAt := time.Now().UTC()
	if err := state.Write(home, state.InstallState{
		RDDMode:           string(reviewtransaction.RDDModeOn),
		RDDModeRecordedAt: &recordedAt,
	}); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := RunReviewMode([]string{"status", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("status: %v", err)
	}
	result := decodeReviewModeResult(t, output.Bytes())
	if result.Status.Effective != reviewtransaction.RDDModeOff || result.Status.Source != reviewtransaction.RDDModeSourceDefault {
		t.Fatalf("status = %#v; quiere off/default con «on» guardado", result.Status)
	}
}

func TestReviewStopHookIsANoOpWhileSuspended(t *testing.T) {
	suspendRDD(t)
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	for _, payload := range []string{
		reviewStopHookTestPayload(t, "sess-suspended", repo, false, nil),
		"esto no es JSON",
		"",
	} {
		var stdout, stderr bytes.Buffer
		if err := runReviewStopHook([]string{"--agent", "claude-code"}, strings.NewReader(payload), &stdout, &stderr); err != nil {
			t.Fatalf("stop-hook debe terminar con 0: %v", err)
		}
		if stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("stop-hook debe ser silencioso: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	}
}
