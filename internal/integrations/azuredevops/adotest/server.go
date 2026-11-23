// Package adotest es el doble de Azure DevOps de las pruebas del adaptador: un servidor httptest
// programable por ruta, con registro de peticiones y tiempo virtual, que simula la revisión (rev)
// de los work items; un reloj manual; y la instalación en las costuras del marco de integraciones.
//
// Garantía de REQ-45.16: ninguna prueba que use este paquete puede llamar a Azure DevOps real. El
// cliente HTTP que entrega Server.HTTPClient rechaza (y hace fallar la prueba) cualquier petición a
// un host que no sea local.
//
// Estado de paquete: las pruebas que llaman a Install NO deben llamar a t.Parallel().
package adotest

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
)

// Request es una petición recibida por el servidor.
type Request struct {
	Index       int
	Method      string
	Path        string // decodificada
	EscapedPath string // tal como viajó
	RawQuery    string
	Query       url.Values
	Header      http.Header
	Body        []byte
	At          time.Time // instante virtual del reloj manual al recibirla
}

// Response es la respuesta programada para una ruta.
type Response struct {
	Status      int         // por defecto 200
	Header      http.Header // cabeceras adicionales
	ContentType string      // por defecto application/json; charset=utf-8
	JSON        any         // cuerpo serializado a JSON (si Body está vacío)
	Body        string      // cuerpo literal; gana a JSON
	Hang        bool        // no responde hasta que el cliente cancele
	Drop        bool        // cierra la conexión sin responder
}

// Handler responde a una petición.
type Handler func(*Request) Response

type route struct {
	method, path string
	handler      Handler
}

// Server es el servidor de Azure DevOps simulado.
type Server struct {
	t     testing.TB
	clock *Clock
	hs    *httptest.Server
	done  chan struct{}

	mu       sync.Mutex
	routes   []route
	requests []Request
	items    *workItemStore
}

// NewServer arranca el servidor y lo cierra con Cleanup. El tiempo virtual de las peticiones sale de
// clock; si es nil se crea un reloj manual nuevo.
func NewServer(t testing.TB, clock *Clock) *Server {
	t.Helper()
	if clock == nil {
		clock = NewClock()
	}
	s := &Server{t: t, clock: clock, done: make(chan struct{})}
	s.hs = httptest.NewServer(http.HandlerFunc(s.serve))
	// Sin conexiones persistentes: cada petición abre la suya, y un corte de conexión no provoca el
	// reintento silencioso del transporte (que alteraría el recuento de peticiones).
	s.hs.Config.SetKeepAlivesEnabled(false)
	t.Cleanup(func() {
		close(s.done)
		s.hs.Close()
	})
	return s
}

// URL es la URL base del servidor (el «dev.azure.com» de las pruebas).
func (s *Server) URL() string { return s.hs.URL }

// Clock devuelve el reloj manual del servidor.
func (s *Server) Clock() *Clock { return s.clock }

// HTTPClient devuelve un cliente HTTP que solo puede hablar con hosts locales.
func (s *Server) HTTPClient() *http.Client {
	return &http.Client{Transport: &localOnlyTransport{t: s.t, base: &http.Transport{DisableKeepAlives: true, Proxy: nil}}}
}

// Options devuelve las opciones del adaptador apuntadas a este servidor: URL local, cliente
// local-only y reloj manual.
func (s *Server) Options() azuredevops.Options {
	return azuredevops.Options{Endpoint: s.URL(), HTTPClient: s.HTTPClient(), Clock: s.clock}
}

// localOnlyTransport rechaza cualquier petición a un host que no sea de bucle local y hace fallar la
// prueba: ninguna prueba puede llamar a Azure DevOps real (REQ-45.16).
type localOnlyTransport struct {
	t    testing.TB
	base http.RoundTripper
}

func (l *localOnlyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !isLocalHost(r.URL.Hostname()) {
		l.t.Errorf("adotest: petición a un host no local (%q): las pruebas no pueden llamar a Azure DevOps real", r.URL.Host)
		return nil, fmt.Errorf("adotest: host no local %q", r.URL.Host)
	}
	return l.base.RoundTrip(r)
}

func isLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Handle registra el manejador de un método y una ruta decodificada exacta. Las rutas se prueban en
// orden de registro; la primera que coincide responde.
func (s *Server) Handle(method, path string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = append(s.routes, route{method: method, path: path, handler: h})
}

// Reply responde siempre lo mismo a un método y una ruta.
func (s *Server) Reply(method, path string, r Response) {
	s.Handle(method, path, func(*Request) Response { return r })
}

// Sequence responde con cada respuesta por orden; la última se repite.
func (s *Server) Sequence(method, path string, responses ...Response) {
	if len(responses) == 0 {
		s.t.Fatalf("adotest: Sequence sin respuestas")
	}
	var mu sync.Mutex
	n := 0
	s.Handle(method, path, func(*Request) Response {
		mu.Lock()
		defer mu.Unlock()
		r := responses[min(n, len(responses)-1)]
		n++
		return r
	})
}

// Requests devuelve las peticiones recibidas, en orden.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Count devuelve cuántas peticiones recibió un método y una ruta decodificada.
func (s *Server) Count(method, path string) int {
	n := 0
	for _, r := range s.Requests() {
		if r.Method == method && r.Path == path {
			n++
		}
	}
	return n
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	req := &Request{
		Method:      r.Method,
		Path:        r.URL.Path,
		EscapedPath: r.URL.EscapedPath(),
		RawQuery:    r.URL.RawQuery,
		Query:       r.URL.Query(),
		Header:      r.Header.Clone(),
		Body:        body,
		At:          s.clock.Now(),
	}
	s.mu.Lock()
	req.Index = len(s.requests)
	s.requests = append(s.requests, *req)
	handler := s.find(req)
	s.mu.Unlock()

	s.write(w, r, handler(req))
}

// find devuelve el manejador de una petición. Se llama con s.mu tomado.
func (s *Server) find(req *Request) Handler {
	for _, rt := range s.routes {
		if rt.method == req.Method && rt.path == req.Path {
			return rt.handler
		}
	}
	if s.items != nil {
		if h := s.items.match(s, req); h != nil {
			return h
		}
	}
	return func(*Request) Response {
		return Response{Status: http.StatusNotFound, JSON: map[string]any{"message": "adotest: sin manejador para " + req.Method + " " + req.Path}}
	}
}

func (s *Server) write(w http.ResponseWriter, r *http.Request, resp Response) {
	switch {
	case resp.Drop:
		if hj, ok := w.(http.Hijacker); ok {
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
		return
	case resp.Hang:
		select {
		case <-r.Context().Done():
		case <-s.done:
		}
		return
	}
	body := []byte(resp.Body)
	if resp.Body == "" && resp.JSON != nil {
		data, err := json.Marshal(resp.JSON)
		if err != nil {
			s.t.Errorf("adotest: JSON no serializable: %v", err)
		}
		body = data
	}
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if len(body) > 0 || resp.ContentType != "" {
		ctype := resp.ContentType
		if ctype == "" {
			ctype = "application/json; charset=utf-8"
		}
		w.Header().Set("Content-Type", ctype)
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_, _ = w.Write(body)
	}
}
