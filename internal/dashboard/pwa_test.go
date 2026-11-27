package dashboard

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type manifestIcon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose"`
}

func TestManifestIsInstallable(t *testing.T) {
	t.Parallel()
	h := assetServer(t)
	rr := getAsset(t, h, "/manifest.webmanifest")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /manifest.webmanifest = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/manifest+json") {
		t.Errorf("Content-Type = %q, se esperaba application/manifest+json", got)
	}
	var m struct {
		Display  string         `json:"display"`
		StartURL string         `json:"start_url"`
		Scope    string         `json:"scope"`
		Icons    []manifestIcon `json:"icons"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("manifiesto no es JSON válido: %v", err)
	}
	if m.Display != "standalone" || m.StartURL != "/" || m.Scope != "/" {
		t.Errorf("display/start_url/scope = %q/%q/%q", m.Display, m.StartURL, m.Scope)
	}
	var maskable bool
	for _, ic := range m.Icons {
		if ic.Purpose == "maskable" {
			maskable = true
		}
		res := getAsset(t, h, ic.Src)
		if res.Code != http.StatusOK {
			t.Errorf("icono %s = %d", ic.Src, res.Code)
			continue
		}
		if got := res.Header().Get("Content-Type"); !strings.HasPrefix(got, ic.Type) {
			t.Errorf("icono %s: Content-Type = %q, se esperaba %q", ic.Src, got, ic.Type)
		}
	}
	if !maskable {
		t.Error("el manifiesto debe declarar un icono maskable")
	}
}

func TestPNGIconsHaveDeclaredSize(t *testing.T) {
	t.Parallel()
	h := assetServer(t)
	for target, size := range map[string]int{
		"/icons/icon-192.png":          192,
		"/icons/icon-512.png":          512,
		"/icons/icon-maskable-512.png": 512,
	} {
		rr := getAsset(t, h, target)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s = %d", target, rr.Code)
			continue
		}
		if rr.Header().Get("Cache-Control") == "" {
			t.Errorf("%s sin Cache-Control", target)
		}
		img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
		if err != nil {
			t.Errorf("%s no decodifica como PNG: %v", target, err)
			continue
		}
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Errorf("%s mide %dx%d, se esperaba %dx%d", target, b.Dx(), b.Dy(), size, size)
		}
		// Fondo opaco a sangre (exigido por maskable) y logo pintado en el centro de la base.
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0xffff {
			t.Errorf("%s: la esquina debe ser opaca", target)
		}
	}
}

func TestMaskableIconKeepsLogoInsideSafeZone(t *testing.T) {
	t.Parallel()
	img := renderIcon(512, true)
	bg := img.RGBAAt(0, 0)
	// Fuera del círculo seguro (80 % central) solo hay fondo degradado, nunca el trazo.
	safe := 512.0 * 0.4
	for y := 0; y < 512; y += 4 {
		for x := 0; x < 512; x += 4 {
			dx, dy := float64(x)-256, float64(y)-256
			if dx*dx+dy*dy <= safe*safe {
				continue
			}
			c := img.RGBAAt(x, y)
			if int(c.B) > int(bg.B)+40 || int(c.R) > int(bg.R)+40 {
				t.Fatalf("pixel (%d,%d) = %v: el logo sale de la zona segura", x, y, c)
			}
		}
	}
}

func TestServiceWorkerIsServedAtRoot(t *testing.T) {
	t.Parallel()
	rr := getAsset(t, assetServer(t), "/sw.js")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /sw.js = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") {
		t.Errorf("Content-Type = %q", got)
	}
	var code []string
	for _, line := range strings.Split(rr.Body.String(), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			code = append(code, line)
		}
	}
	src := strings.Join(code, "\n")
	// Red primero: la caché solo responde cuando la red falla, y nunca guarda respuestas
	// en tiempo de ejecución (una copia obsoleta del dashboard sería peor que un fallo).
	if !strings.Contains(src, "fetch(request).catch(") {
		t.Error("sw.js debe pedir primero a la red y usar la caché solo si la red falla")
	}
	if strings.Contains(src, ".put(") {
		t.Error("sw.js no debe guardar respuestas en tiempo de ejecución")
	}
	if !strings.Contains(src, "request.mode === 'navigate'") {
		t.Error("sw.js debe limitar la página sin conexión a las navegaciones")
	}
}

var offlineAssetsPattern = regexp.MustCompile(`(?s)const OFFLINE_ASSETS = \[(.*?)\];`)
var quotedPathPattern = regexp.MustCompile(`'(/[^']*)'`)

// offlineAssets devuelve las rutas que sw.js guarda al instalarse.
func offlineAssets(t *testing.T) []string {
	t.Helper()
	m := offlineAssetsPattern.FindStringSubmatch(readAsset(t, "sw.js"))
	if m == nil {
		t.Fatal("sw.js no declara OFFLINE_ASSETS")
	}
	var paths []string
	for _, p := range quotedPathPattern.FindAllStringSubmatch(m[1], -1) {
		paths = append(paths, p[1])
	}
	return paths
}

// El service worker solo guarda la página sin conexión y sus recursos: nada de la API
// ni del dashboard, y todo lo que guarda existe (addAll falla entero ante un 404).
func TestServiceWorkerCachesOnlyTheOfflinePage(t *testing.T) {
	t.Parallel()
	h := assetServer(t)
	paths := offlineAssets(t)
	if !slices.Contains(paths, "/offline.html") {
		t.Fatalf("OFFLINE_ASSETS = %v, debe incluir /offline.html", paths)
	}
	for _, p := range paths {
		if strings.HasPrefix(p, "/api/") || p == "/" || p == "/index.html" || p == "/js/main.js" {
			t.Errorf("OFFLINE_ASSETS incluye %s: solo se guarda la página sin conexión", p)
		}
		if rr := getAsset(t, h, p); rr.Code != http.StatusOK {
			t.Errorf("GET %s = %d: el service worker no podría guardarlo", p, rr.Code)
		}
	}
}

var localRefPattern = regexp.MustCompile(`(?:href|src)="(/[^"#]*)"`)

// La página sin conexión explica cómo arrancar el servidor y no depende de nada que el
// service worker no haya guardado.
func TestOfflinePageExplainsHowToStartTheServer(t *testing.T) {
	t.Parallel()
	rr := getAsset(t, assetServer(t), "/offline.html")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /offline.html = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q", got)
	}
	body := rr.Body.String()
	for _, marker := range []string{"axiom ui --detach", "Start-ScheduledTask -TaskName Axiom", `id="offline-retry"`} {
		if !strings.Contains(body, marker) {
			t.Errorf("offline.html no contiene %q", marker)
		}
	}
	cached := offlineAssets(t)
	refs := localRefPattern.FindAllStringSubmatch(body, -1)
	if len(refs) == 0 {
		t.Fatal("offline.html no referencia ningún recurso")
	}
	for _, ref := range refs {
		if !slices.Contains(cached, ref[1]) {
			t.Errorf("offline.html usa %s, que el service worker no guarda", ref[1])
		}
	}
	script := readAsset(t, "offline.js")
	for _, marker := range []string{"/api/health", "location.reload()"} {
		if !strings.Contains(script, marker) {
			t.Errorf("offline.js no contiene %q: debe recargar sola cuando el servidor responda", marker)
		}
	}
}

func TestIndexHTMLDeclaresPWA(t *testing.T) {
	t.Parallel()
	body := getAsset(t, assetServer(t), "/").Body.String()
	for _, marker := range []string{
		`<link rel="manifest" href="/manifest.webmanifest">`,
		`<meta name="theme-color"`,
		`<link rel="icon" href="/icons/icon.svg" type="image/svg+xml">`,
		`<link rel="apple-touch-icon" href="/icons/icon-192.png">`,
		`id="btn-install-app"`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
}
