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

// postEcosystemUpgrade lanza POST /api/ecosystem/upgrade y devuelve el estado con
// el que responde.
func postEcosystemUpgrade(t *testing.T, svc *Service, body string) (int, UpdateJobStatus) {
	t.Helper()
	rr := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/ecosystem/upgrade", strings.NewReader(body)))
	var st UpdateJobStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatalf("respuesta no es un estado de trabajo: %v (%s)", err, rr.Body.String())
	}
	return rr.Code, st
}

// TestEcosystemUpgradeJob_RunsTheChainInTheBackground fija el arreglo de
// «Actualizar» de Ecosistema: la petición contesta enseguida (202) y la cadena
// upgrade -> sync corre en el trabajo del proceso, con el canal pedido y el informe
// consolidado en result.
func TestEcosystemUpgradeJob_RunsTheChainInTheBackground(t *testing.T) {
	var channel atomic.Value
	release := make(chan struct{})
	origReport, origSync := upgradeSequenceReportFn, upgradeSequenceSyncFn
	t.Cleanup(func() { upgradeSequenceReportFn, upgradeSequenceSyncFn = origReport, origSync })
	upgradeSequenceReportFn = func(_ context.Context, _ io.Writer, ch ...string) (app.UpgradeRunReport, error) {
		if len(ch) > 0 {
			channel.Store(ch[0])
		}
		<-release // la cadena sigue en curso cuando la petición ya ha contestado
		return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom"}, nil
	}
	upgradeSequenceSyncFn = func(*Service) (*EcosystemActionResponse, error) {
		return &EcosystemActionResponse{Success: true, Action: "sync", Output: []string{"synced"}}, nil
	}
	svc := updateTestService(t, availableVersion)

	code, st := postEcosystemUpgrade(t, svc, `{"channel":"main"}`)
	if code != http.StatusAccepted {
		t.Fatalf("POST /api/ecosystem/upgrade = %d, quiero 202", code)
	}
	if st.Kind != updateJobKindEcosystem || !st.Running || st.Phase != updatePhaseUpgrading {
		t.Fatalf("estado inicial = %+v, quiero ecosystem en curso (upgrading)", st)
	}
	close(release)

	final := waitUpdateFinished(t, svc)
	if final.Phase != updatePhaseDone {
		t.Fatalf("estado final = %+v, quiero done", final)
	}
	if got, _ := channel.Load().(string); got != "main" {
		t.Fatalf("canal = %q, quiero main", got)
	}
	if final.Result == nil || final.Result.Phases == nil || !final.Result.Phases.Sync.Executed {
		t.Fatalf("result = %+v, quiero el informe con la sincronización ejecutada", final.Result)
	}
	if final.Sync == nil || !final.Sync.Success {
		t.Fatalf("sync = %+v, quiero correcta", final.Sync)
	}
}

// TestEcosystemUpgradeJob_RelaunchesWhenAxiomIsReplaced: si la cadena sustituye
// Axiom, el trabajo entra en reinicio y relanza el dashboard como la barra lateral.
func TestEcosystemUpgradeJob_RelaunchesWhenAxiomIsReplaced(t *testing.T) {
	syncCalls := stubUpgradeSequence(t,
		app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom", RestartRequired: true},
		nil, nil, nil,
	)
	svc := updateTestService(t, availableVersion)
	relaunched := make(chan *UpdateSyncSummary, 1)
	svc.restartFn = func(sync *UpdateSyncSummary) { relaunched <- sync }

	if _, started := svc.StartEcosystemUpgrade("stable"); !started {
		t.Fatal("StartEcosystemUpgrade debía lanzar un trabajo nuevo")
	}
	select {
	case sync := <-relaunched:
		if sync == nil || sync.Executed || sync.SkippedReason != "restart-required" {
			t.Fatalf("sync = %+v, quiero omitida por restart-required", sync)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("no se relanzó el dashboard: %+v", svc.UpdateStatus())
	}
	st := svc.UpdateStatus()
	if st.Phase != updatePhaseRestarting || st.Result == nil || !st.Result.Phases.Upgrade.RestartRequired {
		t.Fatalf("estado = %+v, quiero restarting con el informe", st)
	}
	if *syncCalls != 0 {
		t.Fatalf("el proceso viejo no debe sincronizar: %d invocaciones", *syncCalls)
	}
}

// TestEcosystemUpgradeJob_FailureKeepsTheReportAndTheChannelHint: un fallo deja el
// informe por fases y la orden manual del canal pedido.
func TestEcosystemUpgradeJob_FailureKeepsTheReportAndTheChannelHint(t *testing.T) {
	stubUpgradeSequence(t, app.UpgradeRunReport{Status: app.UpgradeStatusFailed}, errors.New("sin red"), nil, nil)
	svc := updateTestService(t, availableVersion)

	svc.StartEcosystemUpgrade("main")
	st := waitUpdateFinished(t, svc)
	if st.Phase != updatePhaseFailed || st.ManualHint != manualBetaUpdateHint {
		t.Fatalf("estado = %+v, quiero failed con la orden de @main", st)
	}
	if st.Result == nil || st.Result.Phases == nil || st.Result.Phases.Upgrade.Error != "sin red" {
		t.Fatalf("result = %+v, quiero el informe con el error de upgrade", st.Result)
	}
}

// TestEcosystemUpgradeJob_ReturnsTheJobInProgress: con otro trabajo en curso no se
// lanza un segundo; se devuelve el que hay.
func TestEcosystemUpgradeJob_ReturnsTheJobInProgress(t *testing.T) {
	release := make(chan struct{})
	origReport, origSync := upgradeSequenceReportFn, upgradeSequenceSyncFn
	t.Cleanup(func() { upgradeSequenceReportFn, upgradeSequenceSyncFn = origReport, origSync })
	upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
		<-release
		return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil
	}
	upgradeSequenceSyncFn = func(*Service) (*EcosystemActionResponse, error) {
		return &EcosystemActionResponse{Success: true}, nil
	}
	svc := updateTestService(t, availableVersion)

	if _, started := svc.StartUpdate(); !started {
		t.Fatal("StartUpdate debía lanzar un trabajo nuevo")
	}
	st, started := svc.StartEcosystemUpgrade("main")
	if started || st.Kind != updateJobKindUpdate || !st.Running {
		t.Fatalf("StartEcosystemUpgrade = %+v, %v; quiero el trabajo de actualización en curso", st, started)
	}
	close(release)
	waitUpdateFinished(t, svc)
}

func TestManualHintForChannel(t *testing.T) {
	t.Parallel()
	for channel, want := range map[string]string{"": manualUpdateHint, "stable": manualUpdateHint, "main": manualBetaUpdateHint, " Beta ": manualBetaUpdateHint} {
		if got := manualHintForChannel(channel); got != want {
			t.Errorf("manualHintForChannel(%q) = %q, quiero %q", channel, got, want)
		}
	}
}
