package azuredevops_test

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const loginPage = "<html><body>Sign in to your account</body></html>"

// readWithScope hace la lectura genérica con un scope y un id de work item dados.
func readWithScope(c *azuredevops.Client, scope string, workItemID int, out any) error {
	return c.Do(context.Background(), azuredevops.NewReadRequest(scope, "leer el recurso", workItemID, "Fabrikam", "_apis", "ping"), out)
}

// Tabla de D-05: cada respuesta produce el error tipado que corresponde, sin reintentos.
func TestResponseClassification(t *testing.T) {
	type check func(t *testing.T, err error)
	isAuth := func(status int, scope string) check {
		return func(t *testing.T, err error) {
			t.Helper()
			var auth *azuredevops.AuthError
			if !errors.As(err, &auth) || !errors.Is(err, azuredevops.ErrAuth) {
				t.Fatalf("esperaba AuthError, obtuvo %T: %v", err, err)
			}
			if auth.Status != status || auth.Scope != scope {
				t.Fatalf("AuthError = %+v, esperaba estado %d y scope %s", auth, status, scope)
			}
			if !strings.Contains(err.Error(), scope) {
				t.Fatalf("el mensaje debe nombrar el scope %s: %v", scope, err)
			}
		}
	}
	tests := []struct {
		name      string
		resp      adotest.Response
		scope     string
		workItem  int
		check     check
		wantCalls int
	}{
		{"401", adotest.Response{Status: 401}, "vso.work_write", 0, isAuth(401, "vso.work_write"), 1},
		{"403", adotest.Response{Status: 403}, "vso.code", 0, isAuth(403, "vso.code"), 1},
		{"403 en iteraciones nombra vso.work", adotest.Response{Status: 403}, "vso.work", 0, isAuth(403, "vso.work"), 1},
		{"203 con página de inicio de sesión", adotest.Response{Status: 203, Body: loginPage, ContentType: "text/html"}, "vso.project", 0, isAuth(203, "vso.project"), 1},
		{"200 con HTML de inicio de sesión", adotest.Response{Body: loginPage, ContentType: "text/html; charset=utf-8"}, "vso.project", 0, isAuth(200, "vso.project"), 1},
		{"302 sin seguir", adotest.Response{Status: 302, Header: http.Header{"Location": {"https://aex.dev.azure.com/signin"}}}, "vso.work_write", 0, isAuth(302, "vso.work_write"), 1},
		{"404 nombra el work item", adotest.Response{Status: 404, JSON: map[string]any{"message": "no existe"}}, "vso.work_write", 999, func(t *testing.T, err error) {
			t.Helper()
			var nf *azuredevops.NotFoundError
			if !errors.As(err, &nf) || !errors.Is(err, azuredevops.ErrNotFound) || nf.WorkItemID != 999 {
				t.Fatalf("esperaba NotFoundError del work item 999: %v", err)
			}
			if !strings.Contains(err.Error(), "999") {
				t.Fatalf("el mensaje debe nombrar el 999: %v", err)
			}
		}, 1},
		{"400", adotest.Response{Status: 400, JSON: map[string]any{"message": "petición mal formada"}}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			var api *azuredevops.APIError
			if !errors.As(err, &api) || !errors.Is(err, azuredevops.ErrBadRequest) || api.Status != 400 || api.Detail != "petición mal formada" {
				t.Fatalf("esperaba APIError 400 con el detalle del cuerpo: %v", err)
			}
		}, 1},
		{"otro 4xx", adotest.Response{Status: 418, JSON: map[string]any{"message": "tetera"}}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			var api *azuredevops.APIError
			if !errors.As(err, &api) || api.Status != 418 || errors.Is(err, azuredevops.ErrBadRequest) {
				t.Fatalf("esperaba APIError 418 que no es ErrBadRequest: %v", err)
			}
		}, 1},
		{"412 fuera de un PATCH no es conflicto", adotest.Response{Status: 412}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			if errors.Is(err, azuredevops.ErrRevConflict) {
				t.Fatalf("un 412 en una lectura no es conflicto de /rev: %v", err)
			}
		}, 1},
		{"500", adotest.Response{Status: 500, JSON: map[string]any{"message": "fallo interno"}}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			var api *azuredevops.APIError
			if !errors.As(err, &api) || !errors.Is(err, azuredevops.ErrServer) || api.Detail != "fallo interno" {
				t.Fatalf("esperaba APIError ErrServer: %v", err)
			}
		}, 1},
		{"204 es éxito sin cuerpo", adotest.Response{Status: 204}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("un 204 es éxito: %v", err)
			}
		}, 1},
		{"201 JSON es éxito", adotest.Response{Status: 201, JSON: map[string]any{"ok": true}}, "vso.work_write", 0, func(t *testing.T, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("un 2xx con JSON es éxito: %v", err)
			}
		}, 1},
		{"200 con cuerpo no JSON", adotest.Response{Body: "esto no es json", ContentType: "text/plain"}, "vso.work_write", 0, invalidResponse, 1},
		{"200 con JSON truncado", adotest.Response{Body: `{"ok":`}, "vso.work_write", 0, invalidResponse, 1},
		{"200 con cuerpo vacío", adotest.Response{ContentType: "application/json"}, "vso.work_write", 0, invalidResponse, 1},
		{"200 con HTML sin Content-Type html es respuesta inválida", adotest.Response{Body: loginPage, ContentType: "text/plain"}, "vso.work_write", 0, invalidResponse, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := adotest.NewServer(t, nil)
			srv.Reply("GET", pingPath, tt.resp)
			client, _ := newTestClient(t, srv, nil)

			var out map[string]any
			err := readWithScope(client, tt.scope, tt.workItem, &out)

			tt.check(t, err)
			if got := srv.Count("GET", pingPath); got != tt.wantCalls {
				t.Fatalf("peticiones = %d, esperaba %d (sin reintentos)", got, tt.wantCalls)
			}
		})
	}
}

func invalidResponse(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, azuredevops.ErrInvalidResponse) {
		t.Fatalf("esperaba ErrInvalidResponse, obtuvo %v", err)
	}
}

func TestLoginPageIsNeverParsedAsJSON(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 203, Body: loginPage, ContentType: "text/html"})
	client, _ := newTestClient(t, srv, nil)

	err := readWithScope(client, "vso.project", 0, new(map[string]any))

	if errors.Is(err, azuredevops.ErrInvalidResponse) || !errors.Is(err, azuredevops.ErrAuth) {
		t.Fatalf("la página de inicio de sesión es un AuthError, no JSON inválido: %v", err)
	}
}

func TestOversizedSuccessBodyIsAnInvalidResponse(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Body: strings.Repeat("a", 17<<20)})
	client, _ := newTestClient(t, srv, nil)

	err := readWithScope(client, "vso.work_write", 0, new(map[string]any))

	if !errors.Is(err, azuredevops.ErrInvalidResponse) {
		t.Fatalf("un cuerpo de más de 16 MiB es ErrInvalidResponse, nunca un valor vacío: %v", err)
	}
}

func TestAuthErrorMessageNamesScopeAndNoRetirementDates(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 401})
	client, _ := newTestClient(t, srv, nil)

	err := readWithScope(client, "vso.work_write", 0, nil)

	msg := err.Error()
	for _, want := range []string{"HTTP 401", "vso.work_write", "una sola organización", "PAT globales se retiran"} {
		if !strings.Contains(msg, want) {
			t.Errorf("el mensaje debe contener %q: %s", want, msg)
		}
	}
	if regexp.MustCompile(`20\d\d`).MatchString(msg) {
		t.Errorf("el mensaje no debe citar fechas de retirada: %s", msg)
	}
}

func TestHugeErrorBodyIsBounded(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 500, Body: strings.Repeat("x", 5<<20), ContentType: "text/plain"})
	client, _ := newTestClient(t, srv, nil)

	err := readWithScope(client, "vso.work_write", 0, nil)

	var api *azuredevops.APIError
	if !errors.As(err, &api) {
		t.Fatalf("esperaba APIError: %v", err)
	}
	if n := utf8.RuneCountInString(api.Detail); n > 513 {
		t.Fatalf("el detalle incluye %d runas, el máximo es 512 (más la elipsis)", n)
	}
	if len(err.Error()) > 1024 {
		t.Fatalf("el error completo mide %d bytes: no está acotado", len(err.Error()))
	}
}

// El error no incluye cabeceras, el cuerpo de la petición ni la consulta de la URL.
func TestErrorsNeverCarryHeadersRequestOrQuery(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 400, JSON: map[string]any{"message": "inválido"}})
	client, _ := newTestClient(t, srv, nil)

	err := readWithScope(client, "vso.work_write", 0, nil)

	for _, forbidden := range []string{"api-version", "Authorization", "Basic", "User-Agent", testPAT} {
		if strings.Contains(err.Error(), forbidden) {
			t.Errorf("el error no debe contener %q: %v", forbidden, err)
		}
	}
}

// Eco del servidor: el cuerpo de error repite la cabecera Authorization recibida.
func TestServerEchoOfCredentialsIsRedacted(t *testing.T) {
	b64 := strings.TrimPrefix(basicFor(testPAT), "Basic ")
	for _, shape := range []struct {
		name string
		body func(auth string) adotest.Response
	}{
		{"JSON con el eco en el mensaje", func(auth string) adotest.Response {
			return adotest.Response{Status: 400, JSON: map[string]any{"message": "cabecera recibida: " + auth + " y el PAT " + testPAT}}
		}},
		{"texto plano con el eco", func(auth string) adotest.Response {
			return adotest.Response{Status: 500, Body: "Authorization: " + auth + " pat=" + testPAT, ContentType: "text/plain"}
		}},
	} {
		t.Run(shape.name, func(t *testing.T) {
			srv := adotest.NewServer(t, nil)
			srv.Handle("GET", pingPath, func(r *adotest.Request) adotest.Response {
				return shape.body(r.Header.Get("Authorization"))
			})
			client, _ := newTestClient(t, srv, nil)

			err := readWithScope(client, "vso.work_write", 0, nil)

			msg := err.Error()
			for _, leaked := range []string{testPAT, b64, "Basic " + b64} {
				if strings.Contains(msg, leaked) {
					t.Errorf("el error no debe contener %q: %s", leaked, msg)
				}
			}
			if !strings.Contains(msg, "[oculto]") {
				t.Errorf("el error debe marcar lo redactado: %s", msg)
			}
		})
	}
}

// echoingTransport devuelve siempre un error de transporte cuyo texto repite el PAT, para comprobar
// que la causa de red también se redacta.
type echoingTransport struct{ text string }

func (e echoingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New(e.text)
}

func TestTransportCauseEchoingThePATIsRedacted(t *testing.T) {
	b64 := strings.TrimPrefix(basicFor(testPAT), "Basic ")
	srv := adotest.NewServer(t, nil)
	opts := srv.Options()
	opts.HTTPClient = &http.Client{Transport: echoingTransport{text: "fallo con " + testPAT + " y Basic " + b64}}
	opts.MaxAttempts = 1
	client, err := azuredevops.NewClient(opts, testTarget, &patSource{pat: testPAT})
	if err != nil {
		t.Fatal(err)
	}

	err = readWithScope(client, "vso.work_write", 0, nil)

	var transport *azuredevops.TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("esperaba TransportError: %v", err)
	}
	for _, leaked := range []string{testPAT, b64} {
		if strings.Contains(err.Error(), leaked) {
			t.Errorf("el error de transporte no debe contener %q: %v", leaked, err)
		}
	}
}
