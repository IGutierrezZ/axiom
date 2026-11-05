package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// postUpdateService devuelve un servicio sobre un workspace con axiom.yaml.
func postUpdateService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "axiom.yaml"), []byte("topology: monorepo-embedded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root)
	svc.versions, _, _ = newTestVersionChecker("v3.7.0", []VersionInfo{{Status: versionStatusUpToDate}})
	return svc
}

type syncInvocation struct {
	args []string
}

// stubRunAppArgs sustituye la ejecución en proceso de `axiom ...` y registra cada
// invocación; run decide la salida y el error.
func stubRunAppArgs(t *testing.T, run func(args []string, out io.Writer) error) *[]syncInvocation {
	t.Helper()
	var calls []syncInvocation
	orig := runAppArgsFn
	t.Cleanup(func() { runAppArgsFn = orig })
	runAppArgsFn = func(args []string, out io.Writer) error {
		calls = append(calls, syncInvocation{args: append([]string(nil), args...)})
		return run(args, out)
	}
	return &calls
}

func TestPostUpdateSync_RunsTheWorkspaceSyncAndReportsIt(t *testing.T) {
	calls := stubRunAppArgs(t, func(_ []string, out io.Writer) error {
		_, _ = io.WriteString(out, "claude-code: sincronizado\nopencode: sincronizado")
		return nil
	})
	svc := postUpdateService(t)

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)

	if len(*calls) != 1 || !reflect.DeepEqual((*calls)[0].args, []string{"sync", "--scope", "workspace"}) {
		t.Fatalf("invocaciones = %+v, quiero una sola de `sync --scope workspace`", *calls)
	}
	if st.Phase != "done" || st.ManualHint != "" {
		t.Fatalf("estado = %+v, quiero done sin comando manual", st)
	}
	if st.Sync == nil || !st.Sync.Executed || !st.Sync.Success || len(st.Sync.Output) != 2 {
		t.Fatalf("sync = %+v, quiero ejecutada, correcta y con su salida", st.Sync)
	}
	if want := "Axiom se ha actualizado a v3.7.0 y el workspace está sincronizado."; st.Message != want {
		t.Fatalf("mensaje = %q, quiero %q", st.Message, want)
	}
}

// El estado queda en «syncing» desde el primer instante, también por HTTP: la UI que
// recarga tras el reinicio lo ve en curso y lo sigue consultando.
func TestPostUpdateSync_IsSyncingWhileItRunsAndVisibleOverHTTP(t *testing.T) {
	release := make(chan struct{})
	stubRunAppArgs(t, func([]string, io.Writer) error {
		<-release
		return nil
	})
	svc := postUpdateService(t)
	router := NewServer(svc).Router()

	svc.StartPostUpdateSync()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/update/status", nil))
	var st UpdateJobStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if st.Phase != "syncing" || !st.Running {
		t.Fatalf("estado = %+v, quiero syncing en curso", st)
	}

	// Mientras sincroniza, pedir otra actualización no lanza otro trabajo.
	var upgrades int32
	origReport := upgradeSequenceReportFn
	t.Cleanup(func() { upgradeSequenceReportFn = origReport })
	upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
		atomic.AddInt32(&upgrades, 1)
		return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil
	}
	again, started := svc.StartUpdate()
	if started || again.Phase != "syncing" {
		t.Fatalf("StartUpdate durante la sincronización = %+v, started=%v; quiero el trabajo existente", again, started)
	}

	close(release)
	if final := waitUpdateFinished(t, svc); final.Phase != "done" {
		t.Fatalf("estado final = %+v", final)
	}
	if atomic.LoadInt32(&upgrades) != 0 {
		t.Fatal("no debía ejecutarse otra actualización")
	}
}

func TestPostUpdateSync_FailureShowsTheErrorAndTheManualCommand(t *testing.T) {
	stubRunAppArgs(t, func(_ []string, out io.Writer) error {
		_, _ = io.WriteString(out, "pi: fallo parcial")
		return errors.New("sync abortado")
	})
	svc := postUpdateService(t)

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)

	if st.Phase != "failed" || st.ManualHint != "axiom sync --scope workspace" {
		t.Fatalf("estado = %+v, quiero failed con el comando manual", st)
	}
	if st.Sync == nil || !st.Sync.Executed || st.Sync.Success || st.Sync.Error != "sync abortado" || len(st.Sync.Output) != 1 {
		t.Fatalf("sync = %+v, quiero ejecutada, fallida, con su error y su salida", st.Sync)
	}
}

func TestPostUpdateSync_PanicBecomesFailed(t *testing.T) {
	stubRunAppArgs(t, func([]string, io.Writer) error { panic("kaboom") })
	svc := postUpdateService(t)

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)
	if st.Phase != "failed" || st.ManualHint != "axiom sync --scope workspace" {
		t.Fatalf("estado = %+v, quiero failed con el comando manual", st)
	}
}

// Una carpeta sin axiom.yaml no es un workspace de Axiom: no se escribe nada en ella.
func TestPostUpdateSync_SkipsAFolderThatIsNotAnAxiomWorkspace(t *testing.T) {
	calls := stubRunAppArgs(t, func([]string, io.Writer) error { return nil })
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.7.0", []VersionInfo{{Status: versionStatusUpToDate}})

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)
	if len(*calls) != 0 {
		t.Fatalf("no debía sincronizar: %+v", *calls)
	}
	if st.Phase != "done" || st.Sync == nil || st.Sync.Executed || st.Sync.SkippedReason != "no-workspace" {
		t.Fatalf("estado = %+v, quiero done con la sincronización omitida", st)
	}
}

func TestPostUpdateSync_RunsOnlyOnce(t *testing.T) {
	release := make(chan struct{})
	calls := stubRunAppArgs(t, func([]string, io.Writer) error {
		<-release
		return nil
	})
	svc := postUpdateService(t)

	svc.StartPostUpdateSync()
	svc.StartPostUpdateSync() // ya hay un trabajo en curso: no hace nada
	close(release)
	waitUpdateFinished(t, svc)
	time.Sleep(20 * time.Millisecond)
	if len(*calls) != 1 {
		t.Fatalf("la sincronización se ejecutó %d veces, quiero 1", len(*calls))
	}
}
