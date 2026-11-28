package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCheckMainVersionIsOnDemandAndUncached fija «Buscar en main»: compara con la
// cabeza de main el binario instalado, sin caché y sin tocar la comprobación del
// aviso, que sigue siendo solo de releases.
func TestCheckMainVersionIsOnDemandAndUncached(t *testing.T) {
	t.Parallel()
	svc := NewService(t.TempDir())
	checker, _, releaseCalls := newTestVersionChecker("v3.9.1-0.20261127100000-47e94862e0ab", []VersionInfo{{Latest: "3.9.0", Status: versionStatusUpToDate}})
	var mainBases []string
	checker.checkMain = func(_ context.Context, base string) VersionInfo {
		mainBases = append(mainBases, base)
		return VersionInfo{Latest: "main@0123456789ab", Status: versionStatusUpdateAvailable, UpdateAvailable: true, ReleaseURL: "https://github.com/x/y/compare/a...b"}
	}
	checker.installed = func(context.Context) (string, error) { return "v3.9.1-0.20261128100000-aaaaaaaaaaaa", nil }
	svc.versions = checker
	router := NewServer(svc).Router()

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/version/main", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/system/version/main = %d", rec.Code)
		}
		var info VersionInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
			t.Fatalf("JSON inválido: %v", err)
		}
		if info.Status != versionStatusUpdateAvailable || info.Latest != "main@0123456789ab" || info.Installed != "v3.9.1-0.20261128100000-aaaaaaaaaaaa" || info.Instance == "" {
			t.Fatalf("respuesta = %+v", info)
		}
	}
	if len(mainBases) != 2 {
		t.Fatalf("consultas a main = %d, quiero 2 (sin caché)", len(mainBases))
	}
	if mainBases[0] != "v3.9.1-0.20261128100000-aaaaaaaaaaaa" {
		t.Fatalf("base de la comparación = %q, quiero la versión instalada", mainBases[0])
	}
	if *releaseCalls != 0 {
		t.Fatalf("«Buscar en main» no debe consultar las releases del aviso: %d consultas", *releaseCalls)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/system/version/main", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/system/version/main = %d, quiero 405", rec.Code)
	}
}

func TestCheckMainButtonIsWired(t *testing.T) {
	t.Parallel()
	index := getAsset(t, assetServer(t), "/").Body.String()
	if !strings.Contains(index, `id="btn-eco-check-main"`) || !strings.Contains(index, `id="eco-main-status"`) {
		t.Error("index.html no contiene el botón «Buscar en main» y su estado")
	}
	screen := readAsset(t, "js/screens/ecosystem.js")
	for _, marker := range []string{"api('/api/system/version/main')", "mainCheckView(info)", "checkMain(event.currentTarget)"} {
		if !strings.Contains(screen, marker) {
			t.Errorf("js/screens/ecosystem.js no contiene %q", marker)
		}
	}
	if strings.Contains(readAsset(t, "js/update.js"), "/api/system/version/main") {
		t.Error("el aviso automático no debe consultar main")
	}
}
