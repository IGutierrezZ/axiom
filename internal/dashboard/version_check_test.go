package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/update"
)

// fakeVersionClock permite avanzar el tiempo sin dormir.
type fakeVersionClock struct{ t time.Time }

func (c *fakeVersionClock) now() time.Time { return c.t }

// newTestVersionChecker devuelve un comprobador con reloj y comprobador falsos, y
// el contador de consultas reales que ha hecho.
func newTestVersionChecker(current string, results []VersionInfo) (*versionChecker, *fakeVersionClock, *int) {
	clock := &fakeVersionClock{t: time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)}
	calls := new(int)
	c := &versionChecker{
		ttl:     versionCacheTTL,
		now:     clock.now,
		current: func() string { return current },
		check: func(context.Context, string) VersionInfo {
			i := *calls
			*calls++
			if i >= len(results) {
				i = len(results) - 1
			}
			return results[i]
		},
	}
	return c, clock, calls
}

func TestVersionChecker_CachesWithinTTLAndRefreshesAfter(t *testing.T) {
	c, clock, calls := newTestVersionChecker("v3.6.0", []VersionInfo{
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true, ReleaseURL: "https://example.test/r"},
		{Latest: "v3.8.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
	})

	first := c.Get(context.Background(), false)
	if first.Current != "v3.6.0" || first.Latest != "v3.7.0" || !first.UpdateAvailable {
		t.Fatalf("primera consulta = %+v", first)
	}

	clock.t = clock.t.Add(versionCacheTTL - time.Minute)
	if got := c.Get(context.Background(), false); got.Latest != "v3.7.0" || *calls != 1 {
		t.Fatalf("dentro del TTL debe servir la caché: %+v, consultas=%d", got, *calls)
	}

	clock.t = clock.t.Add(2 * time.Minute)
	if got := c.Get(context.Background(), false); got.Latest != "v3.8.0" || *calls != 2 {
		t.Fatalf("tras el TTL debe volver a consultar: %+v, consultas=%d", got, *calls)
	}
}

func TestVersionChecker_RefreshBypassesCache(t *testing.T) {
	c, _, calls := newTestVersionChecker("v3.6.0", []VersionInfo{
		{Latest: "v3.6.0", Status: versionStatusUpToDate},
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
	})

	_ = c.Get(context.Background(), false)
	got := c.Get(context.Background(), true)
	if *calls != 2 || got.Latest != "v3.7.0" {
		t.Fatalf("refresh debe omitir la caché: %+v, consultas=%d", got, *calls)
	}
}

func TestVersionChecker_FailureIsNotCached(t *testing.T) {
	c, _, calls := newTestVersionChecker("v3.6.0", []VersionInfo{
		{Status: versionStatusCheckFailed},
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
	})

	if got := c.Get(context.Background(), false); got.Status != versionStatusCheckFailed || got.UpdateAvailable {
		t.Fatalf("fallo = %+v, quiero check_failed sin aviso", got)
	}
	if got := c.Get(context.Background(), false); got.Status != versionStatusUpdateAvailable || *calls != 2 {
		t.Fatalf("un fallo no debe cachearse: %+v, consultas=%d", got, *calls)
	}
}

func TestVersionChecker_CacheDoesNotSurviveVersionChange(t *testing.T) {
	current := "v3.6.0"
	c, _, calls := newTestVersionChecker(current, []VersionInfo{
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
		{Latest: "v3.7.0", Status: versionStatusUpToDate},
	})
	c.current = func() string { return current }

	_ = c.Get(context.Background(), false)
	current = "v3.7.0"
	got := c.Get(context.Background(), false)
	if *calls != 2 || got.Status != versionStatusUpToDate || got.Current != "v3.7.0" {
		t.Fatalf("con otra versión actual debe reconsultar: %+v, consultas=%d", got, *calls)
	}
}

func TestVersionInfoFromResult(t *testing.T) {
	cases := []struct {
		name       string
		result     update.UpdateResult
		wantStatus string
		wantAvail  bool
	}{
		{"hay versión nueva", update.UpdateResult{Status: update.UpdateAvailable, LatestVersion: "v3.7.0", ReleaseURL: "u"}, versionStatusUpdateAvailable, true},
		{"al día", update.UpdateResult{Status: update.UpToDate, LatestVersion: "v3.6.0"}, versionStatusUpToDate, false},
		{"compilación de desarrollo", update.UpdateResult{Status: update.DevBuild}, versionStatusDevBuild, false},
		{"fallo de red", update.UpdateResult{Status: update.CheckFailed}, versionStatusCheckFailed, false},
		{"versión desconocida", update.UpdateResult{Status: update.VersionUnknown}, versionStatusCheckFailed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := versionInfoFromResult("v3.6.0", tc.result)
			if got.Status != tc.wantStatus || got.UpdateAvailable != tc.wantAvail || got.Current != "v3.6.0" {
				t.Fatalf("versionInfoFromResult = %+v", got)
			}
		})
	}
}

func getVersion(t *testing.T, svc *Service, query string) (int, VersionInfo, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/version"+query, nil))
	var info VersionInfo
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	return rec.Code, info, raw
}

func TestSystemVersionEndpoint_Contract(t *testing.T) {
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true, ReleaseURL: "https://example.test/r"},
	})

	code, info, raw := getVersion(t, svc, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if info.Current != "v3.6.0" || info.Latest != "v3.7.0" || !info.UpdateAvailable || info.Status != "update_available" || info.ReleaseURL != "https://example.test/r" {
		t.Fatalf("respuesta = %+v", info)
	}
	for _, key := range []string{"current", "latest", "update_available", "status", "release_url"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("falta el campo %q en %v", key, raw)
		}
	}
}

func TestSystemVersionEndpoint_NetworkFailureIs200WithoutBanner(t *testing.T) {
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{{Status: versionStatusCheckFailed}})

	code, info, _ := getVersion(t, svc, "")
	if code != http.StatusOK || info.Status != "check_failed" || info.UpdateAvailable {
		t.Fatalf("code=%d info=%+v", code, info)
	}
}

func TestSystemVersionEndpoint_DevBuildAndRefresh(t *testing.T) {
	svc := NewService(t.TempDir())
	var calls *int
	svc.versions, _, calls = newTestVersionChecker("dev", []VersionInfo{
		{Status: versionStatusDevBuild},
		{Latest: "v3.7.0", Status: versionStatusUpdateAvailable, UpdateAvailable: true},
	})

	if _, info, _ := getVersion(t, svc, ""); info.Status != "dev_build" || info.UpdateAvailable {
		t.Fatalf("dev build = %+v", info)
	}
	getVersion(t, svc, "")
	if *calls != 1 {
		t.Fatalf("la segunda consulta debía salir de la caché, consultas=%d", *calls)
	}
	if _, info, _ := getVersion(t, svc, "?refresh=1"); *calls != 2 || info.Status != "update_available" {
		t.Fatalf("?refresh=1 debía reconsultar: %+v, consultas=%d", info, *calls)
	}
}

func TestSystemVersionEndpoint_RejectsNonGET(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer(NewService(t.TempDir())).Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/system/version", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, quiero 405", rec.Code)
	}
}
