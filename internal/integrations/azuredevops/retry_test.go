package azuredevops_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

func withRetryAfter(status int, value string) adotest.Response {
	return adotest.Response{Status: status, Header: http.Header{"Retry-After": {value}}, JSON: map[string]any{"message": "espera"}}
}

func durations(secs ...int) []time.Duration {
	out := make([]time.Duration, len(secs))
	for i, s := range secs {
		out[i] = time.Duration(s) * time.Second
	}
	return out
}

func TestRetryAfterSecondsOn429WaitsAtLeastThatLong(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Sequence("GET", pingPath, withRetryAfter(429, "2"), okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 2 {
		t.Fatalf("peticiones = %d, esperaba 2", got)
	}
	if got := srv.Clock().Sleeps(); len(got) != 1 || got[0] < 2*time.Second {
		t.Fatalf("esperas = %v, esperaba al menos 2 s", got)
	}
}

func TestRetryAfterHTTPDateOn429(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	date := srv.Clock().Now().Add(5 * time.Second).UTC().Format(http.TimeFormat)
	srv.Sequence("GET", pingPath, withRetryAfter(429, date), okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	if got := srv.Clock().Sleeps(); len(got) != 1 || got[0] < 5*time.Second {
		t.Fatalf("esperas = %v, esperaba al menos 5 s (fecha HTTP relativa al reloj)", got)
	}
}

func TestRetryAfterOn200PacesTheNextRequest(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	first := okJSON()
	first.Header = http.Header{"Retry-After": {"3"}}
	srv.Sequence("GET", pingPath, first, okJSON())
	client, _ := newTestClient(t, srv, nil)

	for i := 0; i < 2; i++ {
		if err := ping(context.Background(), client, nil); err != nil {
			t.Fatalf("lectura %d: %v", i, err)
		}
	}
	reqs := srv.Requests()
	if len(reqs) != 2 {
		t.Fatalf("peticiones = %d", len(reqs))
	}
	if gap := reqs[1].At.Sub(reqs[0].At); gap < 3*time.Second {
		t.Fatalf("la segunda petición salió %v después de la primera, esperaba al menos 3 s", gap)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(3)) {
		t.Fatalf("esperas = %v", got)
	}
}

func TestRetryOn503WithExponentialBackoff(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Sequence("GET", pingPath, adotest.Response{Status: 503}, adotest.Response{Status: 503}, okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 3 {
		t.Fatalf("peticiones = %d, esperaba 3", got)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(1, 2)) {
		t.Fatalf("esperas = %v, esperaba 1 s y 2 s", got)
	}
}

func TestBackoffGrowsAndIsCapped(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 503})
	client, _ := newTestClient(t, srv, func(o *azuredevops.Options) { o.MaxAttempts = 8 })

	err := ping(context.Background(), client, nil)

	var api *azuredevops.APIError
	if !errors.As(err, &api) || api.Status != 503 || api.Attempts != 8 || !errors.Is(err, azuredevops.ErrUnavailable) {
		t.Fatalf("esperaba APIError 503 con 8 intentos, obtuvo %T: %v", err, err)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(1, 2, 4, 8, 16, 16, 16)) {
		t.Fatalf("esperas = %v, esperaba 1, 2, 4, 8 y el tope de 16 s", got)
	}
}

func TestDefaultBudgetOn503IsFourAttempts(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 503})
	client, _ := newTestClient(t, srv, nil)

	err := ping(context.Background(), client, nil)

	var api *azuredevops.APIError
	if !errors.As(err, &api) || api.Attempts != 4 {
		t.Fatalf("esperaba APIError con 4 intentos: %v", err)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(1, 2, 4)) {
		t.Fatalf("esperas = %v, esperaba 1 s, 2 s y 4 s", got)
	}
}

func TestRetryAfterBeyondTheCeilingIsNotRetried(t *testing.T) {
	t.Run("429", func(t *testing.T) {
		srv := adotest.NewServer(t, nil)
		srv.Reply("GET", pingPath, withRetryAfter(429, "61"))
		client, _ := newTestClient(t, srv, nil)

		err := ping(context.Background(), client, nil)

		var limit *azuredevops.RateLimitError
		if !errors.As(err, &limit) || limit.Attempts != 1 || limit.RetryAfter != 61*time.Second {
			t.Fatalf("esperaba RateLimitError con 1 intento y 61 s: %v", err)
		}
		if srv.Count("GET", pingPath) != 1 || len(srv.Clock().Sleeps()) != 0 {
			t.Fatal("no se reintenta ni se espera")
		}
	})
	t.Run("503", func(t *testing.T) {
		srv := adotest.NewServer(t, nil)
		srv.Reply("GET", pingPath, withRetryAfter(503, "120"))
		client, _ := newTestClient(t, srv, nil)

		err := ping(context.Background(), client, nil)

		var api *azuredevops.APIError
		if !errors.As(err, &api) || api.Status != 503 || api.Attempts != 1 {
			t.Fatalf("esperaba APIError 503 con 1 intento: %v", err)
		}
		if srv.Count("GET", pingPath) != 1 {
			t.Fatal("no se reintenta")
		}
	})
}

func TestPacingBeyondTheCeilingSendsNothing(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	slow := okJSON()
	slow.Header = http.Header{"Retry-After": {"90"}}
	srv.Sequence("GET", pingPath, slow, okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatal(err)
	}
	err := ping(context.Background(), client, nil)

	var limit *azuredevops.RateLimitError
	if !errors.As(err, &limit) || limit.Attempts != 0 {
		t.Fatalf("esperaba RateLimitError con 0 intentos: %v", err)
	}
	if srv.Count("GET", pingPath) != 1 {
		t.Fatal("la segunda lectura no debe enviarse")
	}
}

func TestExhaustedBudgetOn429ReportsAttempts(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 429})
	client, _ := newTestClient(t, srv, nil)

	err := ping(context.Background(), client, nil)

	var limit *azuredevops.RateLimitError
	if !errors.As(err, &limit) || !errors.Is(err, azuredevops.ErrRateLimited) || limit.Attempts != 4 {
		t.Fatalf("esperaba RateLimitError con 4 intentos: %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 4 {
		t.Fatalf("peticiones = %d, esperaba el máximo de intentos y no más", got)
	}
}

func TestRateLimitHeadersAreExposedAndResourceIsIgnored(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	resp := okJSON()
	resp.Header = http.Header{
		"X-Ratelimit-Delay":     {"1.5"},
		"X-Ratelimit-Remaining": {"42"},
		"X-Ratelimit-Limit":     {"200"},
		"X-Ratelimit-Reset":     {"1767323045"},
		"X-Ratelimit-Resource":  {"git"},
	}
	srv.Reply("GET", pingPath, resp)
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatal(err)
	}
	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatal(err)
	}

	got := client.RateLimit()
	want := azuredevops.RateLimitInfo{Delay: 1500 * time.Millisecond, Remaining: 42, Limit: 200, Reset: time.Unix(1767323045, 0)}
	if got != want {
		t.Fatalf("RateLimit() = %+v, esperaba %+v", got, want)
	}
	if sleeps := srv.Clock().Sleeps(); len(sleeps) != 0 {
		t.Fatalf("ninguna cabecera X-RateLimit-* provoca esperas: %v", sleeps)
	}
}

func TestRateLimitHeadersUnreadableOrMissingKeepWhatWasRead(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	good := okJSON()
	good.Header = http.Header{"X-Ratelimit-Remaining": {"7"}}
	bad := okJSON()
	bad.Header = http.Header{"X-Ratelimit-Remaining": {"muchos"}, "X-Ratelimit-Limit": {"abc"}}
	srv.Sequence("GET", pingPath, good, bad)
	client, _ := newTestClient(t, srv, nil)

	for i := 0; i < 2; i++ {
		if err := ping(context.Background(), client, nil); err != nil {
			t.Fatalf("un valor ilegible no es un fallo: %v", err)
		}
	}
	if got := client.RateLimit(); got.Remaining != 7 || got.Limit != 0 {
		t.Fatalf("RateLimit() = %+v", got)
	}
}

func TestDroppedConnectionIsRetried(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Sequence("GET", pingPath, adotest.Response{Drop: true}, okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 2 {
		t.Fatalf("peticiones = %d, esperaba 2", got)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(1)) {
		t.Fatalf("esperas = %v", got)
	}
}

func TestPersistentNetworkErrorIsATypedTransportError(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Drop: true})
	client, _ := newTestClient(t, srv, func(o *azuredevops.Options) { o.MaxAttempts = 3 })

	err := ping(context.Background(), client, nil)

	var transport *azuredevops.TransportError
	if !errors.As(err, &transport) || transport.Attempts != 3 {
		t.Fatalf("esperaba TransportError con 3 intentos: %v", err)
	}
	if got := srv.Clock().Sleeps(); !reflect.DeepEqual(got, durations(1, 2)) {
		t.Fatalf("esperas = %v", got)
	}
}

func TestServerErrorsOtherThan503AreNotRetried(t *testing.T) {
	for _, status := range []int{500, 502, 504} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := adotest.NewServer(t, nil)
			srv.Reply("GET", pingPath, adotest.Response{Status: status, JSON: map[string]any{"message": "fallo"}})
			client, _ := newTestClient(t, srv, nil)

			err := ping(context.Background(), client, nil)

			var api *azuredevops.APIError
			if !errors.As(err, &api) || api.Status != status || !errors.Is(err, azuredevops.ErrServer) {
				t.Fatalf("esperaba APIError %d: %v", status, err)
			}
			if got := srv.Count("GET", pingPath); got != 1 {
				t.Fatalf("peticiones = %d, esperaba 1 (sin reintento)", got)
			}
			if len(srv.Clock().Sleeps()) != 0 {
				t.Fatal("sin reintento no hay esperas")
			}
		})
	}
}
