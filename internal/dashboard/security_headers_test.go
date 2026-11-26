package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Sin t.Parallel en estas pruebas: construyen servidores con servicio real y comparten el patrón
// de las demás pruebas del paquete que tocan estado de proceso.

// securityTarget es un servidor con el servicio sobre un directorio temporal. Devuelve el
// envoltorio completo (handler()), el que sirve el proceso real.
func securityTarget(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s := NewServer(NewService(t.TempDir()))
	return s, s.handler()
}

// wantCSPDirectives son las nueve directivas fijas de prov:WEB:1, más style-src 'self' (D-01).
var wantCSPDirectives = []string{
	"default-src 'self'", "script-src 'self'", "style-src 'self'", "img-src 'self' data:",
	"font-src 'self'", "connect-src 'self'", "object-src 'none'", "base-uri 'none'",
	"form-action 'self'", "frame-ancestors 'none'",
}

func assertGeneralHeaders(t *testing.T, label string, rr *httptest.ResponseRecorder) {
	t.Helper()
	values := rr.Header().Values("Content-Security-Policy")
	if len(values) != 1 {
		t.Errorf("%s: se esperaba una sola CSP, hay %d: %v", label, len(values), values)
		return
	}
	got := map[string]bool{}
	for _, directive := range strings.Split(values[0], ";") {
		got[strings.TrimSpace(directive)] = true
	}
	for _, want := range wantCSPDirectives {
		if !got[want] {
			t.Errorf("%s: la CSP no contiene %q: %s", label, want, values[0])
		}
	}
	if strings.Contains(values[0], "unsafe-") {
		t.Errorf("%s: la CSP general no admite unsafe-*: %s", label, values[0])
	}
	if v := rr.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Errorf("%s: X-Content-Type-Options = %q", label, v)
	}
	if v := rr.Header().Get("Referrer-Policy"); v != "no-referrer" {
		t.Errorf("%s: Referrer-Policy = %q", label, v)
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	_, h := securityTarget(t)
	cases := []struct {
		label, method, target, host string
		want                        int
	}{
		{"index", http.MethodGet, "/", "127.0.0.1", http.StatusOK},
		{"asset estático", http.MethodGet, "/style.css", "127.0.0.1", http.StatusOK},
		{"HEAD", http.MethodHead, "/", "127.0.0.1", http.StatusOK},
		{"404 estático", http.MethodGet, "/no-existe.js", "127.0.0.1", http.StatusNotFound},
		{"api ok", http.MethodGet, "/api/health", "127.0.0.1", http.StatusOK},
		{"405", http.MethodPost, "/api/health", "127.0.0.1", http.StatusMethodNotAllowed},
		{"403 Host ajeno", http.MethodGet, "/api/integrations", "evil.example", http.StatusForbidden},
		{"403 mutación", http.MethodPost, "/api/projects/add", "evil.example", http.StatusForbidden},
		{"OPTIONS", http.MethodOptions, "/api/health", "127.0.0.1", http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		rr := requestFrom(h, c.method, c.target, "", c.host, "")
		if rr.Code != c.want {
			t.Errorf("%s: estado %d, se esperaba %d", c.label, rr.Code, c.want)
		}
		assertGeneralHeaders(t, c.label, rr)
	}

	// 304: el servidor de ficheros responde con las cabeceras ya fijadas.
	first := requestFrom(h, http.MethodGet, "/style.css", "", "127.0.0.1", "")
	if etag := first.Header().Get("ETag"); etag != "" {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		req.Host = "127.0.0.1"
		req.Header.Set("If-None-Match", etag)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertGeneralHeaders(t, "304", rr)
	} else if lm := first.Header().Get("Last-Modified"); lm != "" {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		req.Host = "127.0.0.1"
		req.Header.Set("If-Modified-Since", lm)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotModified {
			t.Errorf("304: estado %d", rr.Code)
		}
		assertGeneralHeaders(t, "304", rr)
	}
}

func TestSecurityHeadersNoStoreOnIntegrationsRoutes(t *testing.T) {
	_, h := securityTarget(t)
	// Las rutas /api/integrations* aún no existen (tarea 3.2): con Host local las atendería el
	// servidor de ficheros, que descarta Cache-Control en sus errores. Aquí se prueba la
	// respuesta de la guarda; los handlers de 3.2 llevan su propia prueba con Host local.
	for _, target := range []string{"/api/integrations", "/api/integrations/secret", "/api/integrations/change?name=x"} {
		rr := requestFrom(h, http.MethodGet, target, "", "evil.example", "")
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: Cache-Control = %q, se esperaba no-store", target, got)
		}
	}
	if got := requestFrom(h, http.MethodGet, "/api/health", "", "127.0.0.1", "").Header().Get("Cache-Control"); got == "no-store" {
		t.Errorf("/api/health no debe forzar no-store por esta regla")
	}
}

func TestNoAccessControlHeaders(t *testing.T) {
	_, h := securityTarget(t)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions} {
		for _, host := range []string{"127.0.0.1", "evil.example"} {
			rr := requestFrom(h, method, "/api/health", "", host, "https://sitio-ajeno.invalid")
			for name := range rr.Header() {
				if strings.HasPrefix(strings.ToLower(name), "access-control-") {
					t.Errorf("%s Host %s: la respuesta lleva %s", method, host, name)
				}
			}
		}
	}
}

func TestOptionsIsMethodNotAllowed(t *testing.T) {
	_, h := securityTarget(t)
	for _, target := range []string{"/api/health", "/api/settings", "/", "/api/integrations"} {
		rr := requestFrom(h, http.MethodOptions, target, "", "127.0.0.1", "")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("OPTIONS %s = %d, se esperaba 405", target, rr.Code)
		}
	}
	// Con Host ajeno en una ruta de integraciones manda la guarda (403), no el 405.
	for _, method := range []string{http.MethodOptions, http.MethodHead} {
		rr := requestFrom(h, method, "/api/integrations", "", "evil.example", "")
		if rr.Code != http.StatusForbidden {
			t.Errorf("%s /api/integrations con Host ajeno = %d, se esperaba 403", method, rr.Code)
		}
	}
}

func TestExplainerKeepsItsOwnCSPOnce(t *testing.T) {
	_, h := securityTarget(t)
	// Cambio inexistente: el handler fija su CSP antes de validar, así que también sale en el error.
	rr := requestFrom(h, http.MethodGet, "/api/explainer?change=x&kind=spec", "", "127.0.0.1", "")
	values := rr.Header().Values("Content-Security-Policy")
	if len(values) != 1 || values[0] != explainerCSP {
		t.Fatalf("CSP del explicador = %v, se esperaba solo %q", values, explainerCSP)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
}

func TestLocalGuardRejectsForeignHostAndOrigin(t *testing.T) {
	_, h := securityTarget(t)
	hosts := []string{"evil.example", "mi-equipo.local", "", "127.0.0.1.evil.example", "localhost.evil.example:80"}
	for _, host := range hosts {
		for _, target := range []string{"/api/integrations", "/api/integrations/secret", "/api/settings", "/api/health"} {
			rr := requestFrom(h, http.MethodGet, target, "", host, "")
			if rr.Code != http.StatusForbidden {
				t.Errorf("GET %s con Host %q = %d, se esperaba 403", target, host, rr.Code)
			}
		}
		rr := requestFrom(h, http.MethodPut, "/api/settings", `{}`, host, "")
		if rr.Code != http.StatusForbidden {
			t.Errorf("PUT /api/settings con Host %q = %d, se esperaba 403", host, rr.Code)
		}
	}
	origins := []string{
		"null", "https://127.0.0.1", "https://localhost", "http://evil.example",
		"http://localhost.evil.example", "http://mi-equipo.local", "http://127.0.0.1@evil.example",
		"http://127.0.0.1/ruta", "http://user:pw@127.0.0.1",
	}
	for _, origin := range origins {
		for _, route := range []struct{ method, target string }{
			{http.MethodGet, "/api/integrations"},
			{http.MethodPut, "/api/settings"},
			{http.MethodPost, "/api/projects/add"},
		} {
			rr := requestFrom(h, route.method, route.target, `{}`, "127.0.0.1", origin)
			if rr.Code != http.StatusForbidden {
				t.Errorf("%s %s con Origin %q = %d, se esperaba 403", route.method, route.target, origin, rr.Code)
			}
			body := rr.Body.String()
			if !strings.Contains(body, "forbidden_origin") && route.target == "/api/integrations" {
				t.Errorf("el 403 de integraciones debe llevar code forbidden_origin: %s", body)
			}
		}
	}
}

func TestLocalGuardAcceptsLocalWithoutOrigin(t *testing.T) {
	s, h := securityTarget(t)
	for _, host := range []string{"127.0.0.1", "127.0.0.1:4242", "localhost", "LocalHost:80", "localhost.", "localhost.:80", "[::1]", "[::1]:4242"} {
		rr := requestFrom(h, http.MethodGet, "/api/health", "", host, "")
		if rr.Code != http.StatusOK {
			t.Errorf("Host %q sin Origin = %d, se esperaba 200", host, rr.Code)
		}
	}
	for _, origin := range []string{"http://127.0.0.1:4242", "http://LOCALHOST:4242", "http://localhost.:4242", "http://[::1]:4242"} {
		rr := requestFrom(h, http.MethodGet, "/api/health", "", "127.0.0.1:4242", origin)
		if rr.Code != http.StatusOK {
			t.Errorf("Origin %q local = %d, se esperaba 200", origin, rr.Code)
		}
	}

	// Con servidor escuchando el puerto enlazado es el único válido.
	s.port = 4242
	cases := []struct {
		host, origin string
		want         int
	}{
		{"127.0.0.1:4242", "", http.StatusOK},
		{"127.0.0.1:4242", "http://127.0.0.1:4242", http.StatusOK},
		{"[::1]:4242", "", http.StatusOK},
		{"127.0.0.1:9999", "", http.StatusForbidden},
		{"127.0.0.1", "", http.StatusForbidden},
		{"127.0.0.1:4242", "http://127.0.0.1:9999", http.StatusForbidden},
		{"127.0.0.1:4242", "http://127.0.0.1", http.StatusForbidden},
	}
	for _, c := range cases {
		rr := requestFrom(h, http.MethodGet, "/api/health", "", c.host, c.origin)
		if rr.Code != c.want {
			t.Errorf("puerto 4242: Host %q Origin %q = %d, se esperaba %d", c.host, c.origin, rr.Code, c.want)
		}
	}
}

func TestLocalGuardCoversExistingMutatingRoutes(t *testing.T) {
	_, h := securityTarget(t)
	routes := []struct{ method, target string }{
		{http.MethodPost, "/api/projects/add"},
		{http.MethodPost, "/api/agents/install"},
		{http.MethodPost, "/api/skills/approve"},
		{http.MethodPost, "/api/increments/marks"},
		{http.MethodPut, "/api/projects/active/worktree-prefs"},
		{http.MethodPut, "/api/settings/personal"},
		{http.MethodPost, "/api/increments"},
	}
	for _, route := range routes {
		foreignHost := requestFrom(h, route.method, route.target, `{}`, "evil.example", "")
		if foreignHost.Code != http.StatusForbidden {
			t.Errorf("%s %s con Host ajeno = %d, se esperaba 403", route.method, route.target, foreignHost.Code)
		}
		foreignOrigin := requestFrom(h, route.method, route.target, `{}`, "127.0.0.1", "https://sitio-ajeno.invalid")
		if foreignOrigin.Code != http.StatusForbidden {
			t.Errorf("%s %s con Origin ajeno = %d, se esperaba 403", route.method, route.target, foreignOrigin.Code)
		}
	}
	// Una petición local con cuerpo inválido llega al handler: no la corta la guarda.
	rr := requestFrom(h, http.MethodPost, "/api/projects/add", `no es json`, "127.0.0.1", "")
	if rr.Code == http.StatusForbidden {
		t.Errorf("POST local a /api/projects/add rechazada por la guarda: %s", rr.Body.String())
	}
}
