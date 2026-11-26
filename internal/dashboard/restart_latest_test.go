package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// Reinicio forzado con la última versión (RF-UPD-10).

// waitRestarted espera a que el trabajo llame a restartFn.
func waitRestarted(t *testing.T, svc *Service, restarted chan *UpdateSyncSummary) {
	t.Helper()
	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatalf("el dashboard no se relanzó: %+v", svc.UpdateStatus())
	}
}

func TestRestartJob_RelaunchesWithoutInstallingWhenUpToDate(t *testing.T) {
	cases := []struct {
		name string
		info VersionInfo
	}{
		{"al día", VersionInfo{Latest: "v3.6.0", Status: versionStatusUpToDate}},
		{"compilación de desarrollo", VersionInfo{Status: versionStatusDevBuild}},
		{"comprobación fallida", VersionInfo{Status: versionStatusCheckFailed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upgrades := stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil)
			svc := updateTestService(t, tc.info)
			restarted := awaitRestart(svc)

			if _, started := svc.StartRestart(); !started {
				t.Fatal("StartRestart debía lanzar un trabajo nuevo")
			}
			waitRestarted(t, svc, restarted)
			if st := svc.UpdateStatus(); st.Phase != updatePhaseRestarting || st.ManualHint == "" {
				t.Fatalf("estado = %+v, quiero restarting con el comando para relanzar a mano", st)
			}
			if n := atomic.LoadInt32(upgrades); n != 0 {
				t.Fatalf("sin versión nueva no se instala nada: %d actualizaciones", n)
			}
		})
	}
}

func TestRestartJob_InstallsTheLatestVersionBeforeRelaunching(t *testing.T) {
	upgrades := stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom", RestartRequired: true}, nil)
	svc := updateTestService(t, availableVersion)
	restarted := awaitRestart(svc)

	svc.StartRestart()
	waitRestarted(t, svc, restarted)
	if n := atomic.LoadInt32(upgrades); n != 1 {
		t.Fatalf("con una versión nueva se instala una vez antes de reiniciar: %d actualizaciones", n)
	}
}

func TestRestartJob_RelaunchesEvenWhenTheOwnBinaryWasNotReplaced(t *testing.T) {
	stubUpgradeSequence(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom"}, nil,
		&EcosystemActionResponse{Success: true, Action: "sync"}, nil)
	svc := updateTestService(t, availableVersion)
	restarted := awaitRestart(svc)

	svc.StartRestart()
	waitRestarted(t, svc, restarted)
}

func TestRestartJob_InstallFailureKeepsServing(t *testing.T) {
	stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusFailed}, errors.New("boom"))
	svc := updateTestService(t, availableVersion)
	restarted := awaitRestart(svc)

	svc.StartRestart()
	st := waitUpdateFinished(t, svc)
	if st.Phase != updatePhaseFailed || st.ManualHint != manualUpdateHint {
		t.Fatalf("estado = %+v, quiero failed con la orden manual", st)
	}
	select {
	case <-restarted:
		t.Fatal("si la instalación falla, el dashboard viejo sigue sirviendo: no se relanza")
	default:
	}
}

func TestRestartEndpoint_StartsTheForcedJobAndRejectsForeignHosts(t *testing.T) {
	stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil)

	svc := updateTestService(t, VersionInfo{Latest: "v3.6.0", Status: versionStatusUpToDate})
	srv := NewServer(svc)
	srv.port = 8123
	restarted := awaitRestart(svc) // después de NewServer, que registra el suyo

	foreign := httptest.NewRequest(http.MethodPost, "/api/system/restart", nil)
	foreign.Host = "evil.example:8123"
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, foreign)
	if rec.Code != http.StatusForbidden || svc.UpdateStatus().Running {
		t.Fatalf("Host ajeno: status %d y trabajo en curso %v; quiero 403 sin trabajo", rec.Code, svc.UpdateStatus().Running)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/system/restart", nil)
	req.Host = "127.0.0.1:8123"
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, quiero 202 (%s)", rec.Code, rec.Body.String())
	}
	var st UpdateJobStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil || !st.Running {
		t.Fatalf("respuesta = %s (%v), quiero el trabajo en curso", rec.Body.String(), err)
	}
	waitRestarted(t, svc, restarted)

	get := httptest.NewRecorder()
	srv.Router().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/system/restart", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /api/system/restart = %d, quiero 405", get.Code)
	}
}

func TestVersionEndpoint_ReportsTheServerInstance(t *testing.T) {
	t.Parallel()
	first := updateTestService(t, VersionInfo{Latest: "v3.6.0", Status: versionStatusUpToDate})
	second := updateTestService(t, VersionInfo{Latest: "v3.6.0", Status: versionStatusUpToDate})

	rec := httptest.NewRecorder()
	NewServer(first).Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/version", nil))
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, rec.Body.String())
	}
	instance, _ := body["instance"].(string)
	if instance == "" || instance != first.instance {
		t.Fatalf("instance = %q, quiero la del servicio %q", instance, first.instance)
	}
	if first.instance == second.instance {
		t.Fatalf("dos servicios comparten la instancia %q: la UI no distinguiría un reinicio", first.instance)
	}
}
