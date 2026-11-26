package dashboard

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

var errNoInstalledVersion = errors.New("sin versión instalada en las pruebas")

// noInstalledVersion es la sonda que TestMain instala en el paquete: no ejecuta el
// binario de pruebas.
func noInstalledVersion(context.Context) (string, error) { return "", errNoInstalledVersion }

// probeInstalled fija la versión instalada que lee el comprobador.
func probeInstalled(c *versionChecker, version string, err error) {
	c.installed = func(context.Context) (string, error) { return version, err }
}

// recordCheckedVersion devuelve la versión con la que el comprobador consultó la
// última versión.
func recordCheckedVersion(c *versionChecker) *string {
	checked := new(string)
	inner := c.check
	c.check = func(ctx context.Context, v string) VersionInfo {
		*checked = v
		return inner(ctx, v)
	}
	return checked
}

func TestVersionChecker_InstalledBinaryAlreadyLatestAsksForRestart(t *testing.T) {
	t.Parallel()
	c, _, _ := newTestVersionChecker("v3.6.0", []VersionInfo{{Latest: "v3.7.0", Status: versionStatusUpToDate}})
	checked := recordCheckedVersion(c)
	probeInstalled(c, "v3.7.0", nil)

	got := c.Get(context.Background(), false)
	if *checked != "v3.7.0" {
		t.Fatalf("la comprobación debe hacerse con la versión instalada: se hizo con %q", *checked)
	}
	if got.Status != versionStatusRestartRequired || !got.RestartRequired || got.UpdateAvailable {
		t.Fatalf("respuesta = %+v, quiero restart_required sin actualización", got)
	}
	if got.Current != "v3.6.0" || got.Installed != "v3.7.0" {
		t.Fatalf("respuesta = %+v, quiero current v3.6.0 e installed v3.7.0", got)
	}
}

func TestVersionChecker_InstalledBinaryBehindLatestKeepsUpdateAvailable(t *testing.T) {
	t.Parallel()
	c, _, _ := newTestVersionChecker("v3.6.0", []VersionInfo{{Latest: "v3.8.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true}})
	probeInstalled(c, "v3.7.0", nil)

	got := c.Get(context.Background(), false)
	if got.Status != versionStatusUpdateAvailable || !got.UpdateAvailable || !got.RestartRequired || got.Installed != "v3.7.0" {
		t.Fatalf("respuesta = %+v, quiero update_available con restart_required", got)
	}
}

func TestVersionChecker_SameInstalledBinaryChangesNothing(t *testing.T) {
	t.Parallel()
	c, _, _ := newTestVersionChecker("v3.6.0", []VersionInfo{{Latest: "v3.6.0", Status: versionStatusUpToDate}})
	probeInstalled(c, "v3.6.0", nil)

	got := c.Get(context.Background(), false)
	if got.Status != versionStatusUpToDate || got.RestartRequired || got.Installed != "v3.6.0" {
		t.Fatalf("respuesta = %+v, quiero up_to_date sin reinicio", got)
	}
}

func TestVersionChecker_UnreadableInstalledVersionFallsBackToTheProcess(t *testing.T) {
	t.Parallel()
	c, _, _ := newTestVersionChecker("v3.6.0", []VersionInfo{{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true}})
	checked := recordCheckedVersion(c)
	probeInstalled(c, "", errors.New("no arranca"))

	got := c.Get(context.Background(), false)
	if *checked != "v3.6.0" {
		t.Fatalf("sin versión instalada debe comprobarse con la del proceso: se hizo con %q", *checked)
	}
	if got.Status != versionStatusUpdateAvailable || got.RestartRequired || got.Installed != "" {
		t.Fatalf("respuesta = %+v, quiero el comportamiento anterior", got)
	}
}

func TestVersionChecker_RestartRequiredEvenWhenTheCheckFails(t *testing.T) {
	t.Parallel()
	c, _, _ := newTestVersionChecker("v3.6.0", []VersionInfo{{Status: versionStatusCheckFailed}})
	probeInstalled(c, "v3.7.0", nil)

	got := c.Get(context.Background(), false)
	if got.Status != versionStatusRestartRequired || !got.RestartRequired {
		t.Fatalf("respuesta = %+v, quiero restart_required aunque falle la red", got)
	}
}

func TestVersionChecker_CacheFollowsTheInstalledVersion(t *testing.T) {
	t.Parallel()
	c, _, calls := newTestVersionChecker("v3.6.0", []VersionInfo{
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
		{Latest: "v3.7.0", Status: versionStatusUpToDate},
	})
	probeInstalled(c, "v3.6.0", nil)
	c.Get(context.Background(), false)

	// Un `go install` externo cambia el binario instalado: la caché no sirve.
	probeInstalled(c, "v3.7.0", nil)
	got := c.Get(context.Background(), false)
	if *calls != 2 || got.Status != versionStatusRestartRequired {
		t.Fatalf("respuesta = %+v tras %d consultas, quiero una consulta nueva y restart_required", got, *calls)
	}
}

func TestParseVersionOutput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		out, want string
		wantErr   bool
	}{
		{"axiom version v3.9.1-0.20261126150239-56e2de995451+dirty (windows/amd64) commit:dev\n", "v3.9.1-0.20261126150239-56e2de995451+dirty", false},
		{"axiom version 3.9.0 (linux/amd64) commit:abc1234\n", "3.9.0", false},
		{"", "", true},
		{"axiom\n", "", true},
	}
	for _, tc := range cases {
		got, err := parseVersionOutput(tc.out)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Fatalf("parseVersionOutput(%q) = %q, %v; quiero %q (error: %v)", tc.out, got, err, tc.want, tc.wantErr)
		}
	}
}

func TestManualUpdateHintFollowsTheChannel(t *testing.T) {
	t.Parallel()
	if got := manualUpdateHintFor("main@56e2de9"); got != manualBetaUpdateHint || !strings.HasSuffix(got, "@main") {
		t.Fatalf("canal beta: %q, quiero la orden @main", got)
	}
	if got := manualUpdateHintFor("v3.9.0"); got != manualUpdateHint {
		t.Fatalf("canal estable: %q, quiero %q", got, manualUpdateHint)
	}
}

// stubCountingUpgrade sustituye la actualización y cuenta cuántas veces se lanza.
func stubCountingUpgrade(t *testing.T, report app.UpgradeRunReport, err error) *int32 {
	t.Helper()
	calls := new(int32)
	orig := upgradeSequenceReportFn
	t.Cleanup(func() { upgradeSequenceReportFn = orig })
	upgradeSequenceReportFn = func(context.Context, io.Writer, ...string) (app.UpgradeRunReport, error) {
		atomic.AddInt32(calls, 1)
		return report, err
	}
	return calls
}

// awaitRestart registra restartFn y devuelve el canal por el que avisa.
func awaitRestart(svc *Service) chan *UpdateSyncSummary {
	restarted := make(chan *UpdateSyncSummary, 1)
	svc.restartFn = func(sync *UpdateSyncSummary) { restarted <- sync }
	return restarted
}

func TestUpdateJob_RestartRequiredRelaunchesWithoutInstalling(t *testing.T) {
	upgrades := stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil)
	svc := updateTestService(t, VersionInfo{Latest: "v3.7.0", Status: versionStatusUpToDate})
	probeInstalled(svc.versions, "v3.7.0", nil)
	restarted := awaitRestart(svc)

	svc.StartUpdate()
	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatalf("el dashboard no se relanzó: %+v", svc.UpdateStatus())
	}
	st := svc.UpdateStatus()
	if st.Phase != updatePhaseRestarting || !st.Running || !strings.Contains(st.Message, "v3.7.0") {
		t.Fatalf("estado = %+v, quiero restarting citando el binario instalado", st)
	}
	if st.ManualHint == "" {
		t.Fatal("el reinicio debe dejar a la vista el comando para relanzar a mano")
	}
	if n := atomic.LoadInt32(upgrades); n != 0 {
		t.Fatalf("un reinicio pendiente no debe instalar nada: %d actualizaciones", n)
	}
}

func TestUpdateJob_BetaFailureOffersTheMainCommand(t *testing.T) {
	stubCountingUpgrade(t, app.UpgradeRunReport{Status: app.UpgradeStatusFailed}, errors.New("boom"))
	svc := updateTestService(t, VersionInfo{Latest: "main@56e2de9", Status: versionStatusUpdateAvailable, UpdateAvailable: true})

	svc.StartUpdate()
	st := waitUpdateFinished(t, svc)
	if st.Phase != updatePhaseFailed || st.ManualHint != manualBetaUpdateHint {
		t.Fatalf("estado = %+v, quiero failed con la orden %q", st, manualBetaUpdateHint)
	}
}
