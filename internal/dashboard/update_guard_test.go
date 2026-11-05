package dashboard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// POST /api/system/update cambia estado: solo lo acepta el propio dashboard (Host y
// Origin de 127.0.0.1/localhost en el puerto realmente enlazado). CORS abierto y DNS
// rebinding no deben poder lanzar una actualización desde otra página.
func TestUpdateEndpoint_RejectsForeignOriginAndHost(t *testing.T) {
	stubUpgradeSequence(t, app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded}, nil,
		&EcosystemActionResponse{Success: true}, nil)
	cases := []struct {
		name, host, origin string
		want               int
	}{
		{"mismo origen 127.0.0.1", "127.0.0.1:8123", "http://127.0.0.1:8123", http.StatusAccepted},
		{"mismo origen localhost", "localhost:8123", "http://localhost:8123", http.StatusAccepted},
		{"sin Origin", "127.0.0.1:8123", "", http.StatusAccepted},
		{"Origin ajeno", "127.0.0.1:8123", "https://evil.example", http.StatusForbidden},
		{"Origin local de otro puerto", "127.0.0.1:8123", "http://127.0.0.1:9999", http.StatusForbidden},
		{"Origin null", "127.0.0.1:8123", "null", http.StatusForbidden},
		{"Host ajeno (DNS rebinding)", "evil.example:8123", "http://evil.example:8123", http.StatusForbidden},
		{"Host ajeno sin Origin", "evil.example:8123", "", http.StatusForbidden},
		{"Host local de otro puerto", "127.0.0.1:9999", "", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := updateTestService(t, availableVersion)
			srv := NewServer(svc)
			srv.port = 8123 // puerto realmente enlazado
			req := httptest.NewRequest(http.MethodPost, "/api/system/update", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, quiero %d (%s)", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusForbidden && svc.UpdateStatus().Running {
				t.Fatal("una petición rechazada no debe lanzar el trabajo")
			}
			if tc.want == http.StatusAccepted {
				waitUpdateFinished(t, svc)
			}
		})
	}
}

// Anunciada (pending) no ejecuta nada: la sincronización real solo arranca al llamar
// a StartPostUpdateSync, ya enlazado el puerto.
func TestPostUpdateSync_PendingDoesNotRunUntilStarted(t *testing.T) {
	calls := stubRunAppArgs(t, func([]string, io.Writer) error { return nil })
	svc := postUpdateService(t)

	svc.MarkPostUpdateSyncPending()
	if st := svc.UpdateStatus(); st.Phase != "syncing" || !st.Running {
		t.Fatalf("estado pendiente = %+v, quiero syncing en curso", st)
	}
	if len(*calls) != 0 {
		t.Fatalf("no debía sincronizar antes de StartPostUpdateSync: %+v", *calls)
	}

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)
	if len(*calls) != 1 || st.Phase != "done" {
		t.Fatalf("calls=%d estado=%+v, quiero una sincronización terminada", len(*calls), st)
	}
}
