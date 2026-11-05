package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// updateTestService devuelve un servicio cuya comprobación de versión es falsa.
func updateTestService(t *testing.T, info VersionInfo) *Service {
	t.Helper()
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{info})
	return svc
}

var availableVersion = VersionInfo{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true}

// waitUpdateFinished espera a que el trabajo en segundo plano termine.
func waitUpdateFinished(t *testing.T, svc *Service) UpdateJobStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := svc.UpdateStatus(); !st.Running && st.Phase != updatePhaseIdle {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("el trabajo de actualización no terminó: %+v", svc.UpdateStatus())
	return UpdateJobStatus{}
}

func TestUpdateJob_IdleBeforeAnyUpdate(t *testing.T) {
	svc := updateTestService(t, availableVersion)
	st := svc.UpdateStatus()
	if st.Phase != "idle" || st.Running {
		t.Fatalf("estado inicial = %+v, quiero idle sin trabajo en curso", st)
	}
}

func TestUpdateJob_SucceedsAndReportsSync(t *testing.T) {
	syncCalls := stubUpgradeSequence(t,
		app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom"},
		nil,
		&EcosystemActionResponse{Success: true, Action: "sync", Output: []string{"synced"}},
		nil,
	)
	svc := updateTestService(t, availableVersion)

	if _, started := svc.StartUpdate(); !started {
		t.Fatal("StartUpdate debía lanzar un trabajo nuevo")
	}
	st := waitUpdateFinished(t, svc)
	if st.Phase != "done" || st.ManualHint != "" {
		t.Fatalf("estado final = %+v, quiero done sin sugerencia manual", st)
	}
	if st.Sync == nil || !st.Sync.Executed || !st.Sync.Success || len(st.Sync.Output) != 1 {
		t.Fatalf("sync = %+v, quiero ejecutada y correcta", st.Sync)
	}
	if *syncCalls != 1 {
		t.Fatalf("sync invocaciones = %d, quiero 1", *syncCalls)
	}
}

func TestUpdateJob_SelfUpgradeSkipsSyncInTheOldProcess(t *testing.T) {
	syncCalls := stubUpgradeSequence(t,
		app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom", RestartRequired: true},
		nil, nil, nil,
	)
	svc := updateTestService(t, availableVersion)

	svc.StartUpdate()
	st := waitUpdateFinished(t, svc)
	if *syncCalls != 0 {
		t.Fatalf("el proceso viejo no debe sincronizar tras autoactualizarse: %d invocaciones", *syncCalls)
	}
	if st.Sync == nil || st.Sync.Executed || st.Sync.SkippedReason != "restart-required" {
		t.Fatalf("sync = %+v, quiero omitida por restart-required", st.Sync)
	}
}

func TestUpdateJob_UpgradeFailureKeepsServingAndShowsManualHint(t *testing.T) {
	cases := []struct {
		name     string
		report   app.UpgradeRunReport
		err      error
		wantHint string
	}{
		{"error sin pista de la herramienta", app.UpgradeRunReport{Status: app.UpgradeStatusFailed}, errors.New("boom"), manualUpdateHint},
		{"pista propia de la herramienta", app.UpgradeRunReport{Status: app.UpgradeStatusFailed, ManualHint: "ejecuta esto a mano"}, errors.New("boom"), "ejecuta esto a mano"},
		{"relevo manual (no se reemplazó)", app.UpgradeRunReport{Status: app.UpgradeStatusSkipped, ManualHint: "go install otra-cosa@latest"}, nil, "go install otra-cosa@latest"},
		{"relevo manual sin pista", app.UpgradeRunReport{Status: app.UpgradeStatusSkipped}, nil, manualUpdateHint},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubUpgradeSequence(t, tc.report, tc.err, nil, nil)
			svc := updateTestService(t, availableVersion)

			svc.StartUpdate()
			st := waitUpdateFinished(t, svc)
			if st.Phase != "failed" || st.ManualHint != tc.wantHint || st.Message == "" {
				t.Fatalf("estado = %+v, quiero failed con la pista %q", st, tc.wantHint)
			}
			// El dashboard viejo sigue sirviendo y se puede volver a intentar.
			if st.Running {
				t.Fatal("tras fallar no debe quedar un trabajo en curso")
			}
		})
	}
}

func TestUpdateJob_NothingToUpdate(t *testing.T) {
	cases := []struct {
		name      string
		info      VersionInfo
		wantPhase string
	}{
		{"al día", VersionInfo{Status: versionStatusUpToDate}, "done"},
		{"compilación de desarrollo", VersionInfo{Status: versionStatusDevBuild}, "done"},
		{"comprobación fallida", VersionInfo{Status: versionStatusCheckFailed}, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var upgrades int32
			orig := upgradeSequenceReportFn
			t.Cleanup(func() { upgradeSequenceReportFn = orig })
			upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
				atomic.AddInt32(&upgrades, 1)
				return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil
			}
			svc := updateTestService(t, tc.info)

			svc.StartUpdate()
			st := waitUpdateFinished(t, svc)
			if st.Phase != tc.wantPhase {
				t.Fatalf("fase = %q, quiero %q (%+v)", st.Phase, tc.wantPhase, st)
			}
			if atomic.LoadInt32(&upgrades) != 0 {
				t.Fatal("sin versión nueva no se debe ejecutar la actualización")
			}
			if tc.wantPhase == "failed" && st.ManualHint != manualUpdateHint {
				t.Fatalf("la comprobación fallida debe ofrecer el comando manual, tiene %q", st.ManualHint)
			}
		})
	}
}

func TestUpdateJob_PanicBecomesFailed(t *testing.T) {
	orig := upgradeSequenceReportFn
	t.Cleanup(func() { upgradeSequenceReportFn = orig })
	upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
		panic("kaboom")
	}
	svc := updateTestService(t, availableVersion)

	svc.StartUpdate()
	st := waitUpdateFinished(t, svc)
	if st.Phase != "failed" || !strings.Contains(st.Message, "kaboom") || st.ManualHint != manualUpdateHint {
		t.Fatalf("estado = %+v, quiero failed con el motivo y el comando manual", st)
	}
}

// El POST contesta enseguida aunque la actualización siga en marcha, y una segunda
// petición no lanza otro trabajo: devuelve el que ya existe.
func TestUpdateJob_SingleJobAndPostReturnsImmediately(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{}, 4)
	var upgrades int32
	orig := upgradeSequenceReportFn
	t.Cleanup(func() { upgradeSequenceReportFn = orig })
	upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
		atomic.AddInt32(&upgrades, 1)
		entered <- struct{}{}
		<-release
		return app.UpgradeRunReport{Status: app.UpgradeStatusFailed, ManualHint: "x"}, errors.New("fin")
	}

	svc := updateTestService(t, availableVersion)
	router := NewServer(svc).Router()

	post := func() (int, UpdateJobStatus) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/system/update", nil))
		var st UpdateJobStatus
		if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
			t.Fatalf("JSON inválido: %v (%s)", err, rec.Body.String())
		}
		return rec.Code, st
	}

	code, first := post()
	if code != http.StatusAccepted || !first.Running {
		t.Fatalf("primer POST = %d %+v, quiero 202 con el trabajo en curso", code, first)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("el trabajo no llegó a la fase de actualización")
	}

	// Segundo POST con el trabajo bloqueado en upgrading: devuelve el existente.
	code, second := post()
	if code != http.StatusAccepted || !second.Running || second.Phase != "upgrading" {
		t.Fatalf("segundo POST = %d %+v, quiero el trabajo existente en upgrading", code, second)
	}

	// El estado se consulta aparte y refleja la fase.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/update/status", nil))
	var polled UpdateJobStatus
	_ = json.Unmarshal(rec.Body.Bytes(), &polled)
	if rec.Code != http.StatusOK || polled.Phase != "upgrading" {
		t.Fatalf("GET status = %d %+v", rec.Code, polled)
	}

	close(release)
	st := waitUpdateFinished(t, svc)
	if st.Phase != "failed" || st.ManualHint != "x" {
		t.Fatalf("estado final = %+v", st)
	}
	if n := atomic.LoadInt32(&upgrades); n != 1 {
		t.Fatalf("la actualización se ejecutó %d veces, quiero 1", n)
	}

	// Terminado el trabajo, se puede lanzar otro.
	if _, started := svc.StartUpdate(); !started {
		t.Fatal("tras terminar debe poder lanzarse un trabajo nuevo")
	}
	waitUpdateFinished(t, svc)
}

func TestUpdateEndpoints_RejectWrongMethods(t *testing.T) {
	router := NewServer(NewService(t.TempDir())).Router()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/system/update"},
		{http.MethodPost, "/api/system/update/status"},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s = %d, quiero 405", tc.method, tc.path, rec.Code)
		}
	}
}

func TestUpdateStatusJSONContract(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer(NewService(t.TempDir())).Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/update/status", nil))
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	for _, key := range []string{"phase", "running", "message", "manual_hint", "sync"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("falta el campo %q en %v", key, raw)
		}
	}
}
