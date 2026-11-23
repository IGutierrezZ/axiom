package azuredevops_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

func okJSON() adotest.Response { return adotest.Response{JSON: map[string]any{"ok": true}} }

func TestRequestCarriesAPIVersionAuthAndUserAgent(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, okJSON())
	client, _ := newTestClient(t, srv, nil)

	var out map[string]any
	if err := ping(context.Background(), client, &out); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 {
		t.Fatalf("peticiones = %d", len(reqs))
	}
	r := reqs[0]
	if got := r.Query.Get("api-version"); got != "7.1" {
		t.Errorf("api-version = %q", got)
	}
	if got := r.Header.Get("Authorization"); got != basicFor(testPAT) {
		t.Errorf("Authorization = %q, esperaba %q", got, basicFor(testPAT))
	}
	if got := r.Header.Get("User-Agent"); got != "axiom" {
		t.Errorf("User-Agent = %q", got)
	}
	if out["ok"] != true {
		t.Errorf("respuesta = %v", out)
	}
}

func TestPATIsResolvedOnEveryRequest(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, okJSON())
	client, src := newTestClient(t, srv, nil)

	for i := 0; i < 2; i++ {
		if err := ping(context.Background(), client, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := src.calls.Load(); got != 2 {
		t.Fatalf("la fuente se consultó %d veces, esperaba 2 (una por petición)", got)
	}
}

func TestPATComesOnlyFromTheSecretSource(t *testing.T) {
	t.Setenv("AXIOM_INTEGRATION_AZURE_DEVOPS_PAT", "otro-valor-del-entorno-que-no-debe-usarse")
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, okJSON())
	client, _ := newTestClient(t, srv, nil)

	if err := ping(context.Background(), client, nil); err != nil {
		t.Fatal(err)
	}
	if got := srv.Requests()[0].Header.Get("Authorization"); got != basicFor(testPAT) {
		t.Fatalf("la cabecera se construye con la fuente inyectada y ningún otro origen: %q", got)
	}
}

func TestMissingSecretIsTypedAndSendsNothing(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, okJSON())
	opts := srv.Options()
	src := &patSource{err: &secrets.MissingError{Ref: secrets.Ref{Plugin: "azure-devops", Name: "pat"}}}
	client, err := azuredevops.NewClient(opts, testTarget, src)
	if err != nil {
		t.Fatal(err)
	}

	err = ping(context.Background(), client, nil)

	var missing *secrets.MissingError
	if !errors.As(err, &missing) {
		t.Fatalf("esperaba *secrets.MissingError, obtuvo %T: %v", err, err)
	}
	for _, want := range []string{"pat", "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error debe nombrar %q: %v", want, err)
		}
	}
	if n := len(srv.Requests()); n != 0 {
		t.Fatalf("sin PAT no se envía ninguna petición HTTP, se enviaron %d", n)
	}
}

func TestEmptyPATCountsAsMissing(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, err := azuredevops.NewClient(srv.Options(), testTarget, &patSource{pat: ""})
	if err != nil {
		t.Fatal(err)
	}
	var missing *secrets.MissingError
	if err := ping(context.Background(), client, nil); !errors.As(err, &missing) {
		t.Fatalf("un PAT vacío se trata como ausente: %v", err)
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("no se envía nada")
	}
}

func TestURLEscapesSegmentsAndKeepsTheHost(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	// Rutas decodificadas: la «/» y la «?» del nombre llegan dentro de un único segmento.
	srv.Reply("GET", "/con/barra?x/Mi Proyecto/_apis/ping", okJSON())
	target := azuredevops.Target{Organization: "con/barra?x", Project: "Mi Proyecto", Team: "t"}
	client, err := azuredevops.NewClient(srv.Options(), target, &patSource{pat: testPAT})
	if err != nil {
		t.Fatal(err)
	}

	err = client.Do(context.Background(), azuredevops.NewReadRequest("vso.work_write", "leer", 0, "Mi Proyecto", "_apis", "ping"), nil)

	if err != nil {
		t.Fatalf("lectura: %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 {
		t.Fatalf("la petición debe llegar al servidor local: %d", len(reqs))
	}
	if want := "/con%2Fbarra%3Fx/Mi%20Proyecto/_apis/ping"; reqs[0].EscapedPath != want {
		t.Fatalf("ruta escapada = %q, esperaba %q", reqs[0].EscapedPath, want)
	}
	if reqs[0].Query.Get("api-version") != "7.1" {
		t.Fatalf("la «?» del nombre no debe romper la consulta: %q", reqs[0].RawQuery)
	}
}

func TestDangerousSegmentsAreConfigurationErrors(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	cases := []azuredevops.Target{
		{Organization: "", Project: "p", Team: "t"},
		{Organization: ".", Project: "p", Team: "t"},
		{Organization: "..", Project: "p", Team: "t"},
		{Organization: "o", Project: "", Team: "t"},
		{Organization: "o", Project: "..", Team: "t"},
		{Organization: "o", Project: "p", Team: "."},
		{Organization: "o", Project: "p", Team: "  "},
	}
	for _, target := range cases {
		t.Run(target.Organization+"|"+target.Project+"|"+target.Team, func(t *testing.T) {
			_, err := azuredevops.NewClient(srv.Options(), target, &patSource{pat: testPAT})
			if !errors.Is(err, azuredevops.ErrConfig) {
				t.Fatalf("esperaba ErrConfig, obtuvo %v", err)
			}
		})
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("ningún segmento peligroso llega a enviarse")
	}
}

func TestDangerousRequestSegmentsNeverLeaveTheOrganization(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, _ := newTestClient(t, srv, nil)
	for _, segment := range []string{"", ".", ".."} {
		err := client.Do(context.Background(), azuredevops.NewReadRequest("vso.work_write", "leer", 0, "Fabrikam", segment, "x"), nil)
		if !errors.Is(err, azuredevops.ErrConfig) {
			t.Errorf("segmento %q: esperaba ErrConfig, obtuvo %v", segment, err)
		}
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("ninguna petición con un segmento peligroso llega a enviarse")
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: http.StatusFound, Header: http.Header{"Location": {"/login"}}})
	srv.Reply("GET", "/login", adotest.Response{Body: "<html>login</html>", ContentType: "text/html"})
	client, _ := newTestClient(t, srv, nil)

	err := ping(context.Background(), client, nil)

	var auth *azuredevops.AuthError
	if !errors.As(err, &auth) || auth.Status != http.StatusFound {
		t.Fatalf("una redirección es un error de autenticación: %v", err)
	}
	if srv.Count("GET", "/login") != 0 {
		t.Fatal("la redirección no se sigue")
	}
	if srv.Count("GET", pingPath) != 1 {
		t.Fatal("una sola petición, sin reintento")
	}
}

func TestInjectedHTTPClientIsNotModified(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	hc := srv.HTTPClient()
	opts := srv.Options()
	opts.HTTPClient = hc
	if _, err := azuredevops.NewClient(opts, testTarget, &patSource{pat: testPAT}); err != nil {
		t.Fatal(err)
	}
	if hc.CheckRedirect != nil {
		t.Fatal("el cliente inyectado no se modifica: el cliente trabaja sobre una copia")
	}
}

func TestTimeoutPerAttemptIsATypedNetworkError(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Hang: true})
	client, _ := newTestClient(t, srv, func(o *azuredevops.Options) {
		o.Timeout = 20 * time.Millisecond
		o.MaxAttempts = 1
	})

	began := time.Now()
	err := ping(context.Background(), client, nil)

	var transport *azuredevops.TransportError
	if !errors.As(err, &transport) || !errors.Is(err, azuredevops.ErrTransport) {
		t.Fatalf("esperaba TransportError, obtuvo %T: %v", err, err)
	}
	if transport.Attempts != 1 {
		t.Errorf("Attempts = %d", transport.Attempts)
	}
	if strings.Contains(err.Error(), testPAT) {
		t.Fatalf("el error no puede contener el PAT: %v", err)
	}
	if time.Since(began) > 5*time.Second {
		t.Fatal("el timeout por intento no se respetó")
	}
}

func TestCancellationDuringAWaitSendsNoMoreRequests(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Status: 429, Header: http.Header{"Retry-After": {"30"}}})
	client, _ := newTestClient(t, srv, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.Clock().OnSleep(func(time.Duration) { cancel() })

	err := ping(ctx, client, nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperaba el error de cancelación, obtuvo %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 1 {
		t.Fatalf("tras cancelar no se envían más peticiones: %d", got)
	}
}

func TestCancellationInFlightReturnsTheContextError(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", pingPath, adotest.Response{Hang: true})
	client, _ := newTestClient(t, srv, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for srv.Count("GET", pingPath) == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	err := ping(ctx, client, nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperaba el error de cancelación, obtuvo %v", err)
	}
	if got := srv.Count("GET", pingPath); got != 1 {
		t.Fatalf("peticiones = %d", got)
	}
}

func TestNewClientValidatesEndpointAndSource(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	for _, endpoint := range []string{"ftp://example.test", "http://", "://x", "http://u:p@host", "http://host?x=1"} {
		opts := srv.Options()
		opts.Endpoint = endpoint
		if _, err := azuredevops.NewClient(opts, testTarget, &patSource{pat: testPAT}); !errors.Is(err, azuredevops.ErrConfig) {
			t.Errorf("endpoint %q: esperaba ErrConfig, obtuvo %v", endpoint, err)
		}
	}
	if _, err := azuredevops.NewClient(srv.Options(), testTarget, nil); !errors.Is(err, azuredevops.ErrConfig) {
		t.Errorf("sin fuente de secretos: esperaba ErrConfig, obtuvo %v", err)
	}
}

func TestDefaultEndpointIsAzureDevOpsAndMakesNoRequest(t *testing.T) {
	client, err := azuredevops.NewClient(azuredevops.Options{}, testTarget, &patSource{pat: testPAT})
	if err != nil {
		t.Fatal(err)
	}
	if client.Endpoint() != azuredevops.DefaultEndpoint || azuredevops.DefaultEndpoint != "https://dev.azure.com" {
		t.Fatalf("endpoint por defecto = %q", client.Endpoint())
	}
	if client.Target() != testTarget {
		t.Fatalf("Target = %+v", client.Target())
	}
}

func TestPluginNewClientSharesStateByOrganizationIgnoringCase(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", "/Contoso/Fabrikam/_apis/ping", adotest.Response{JSON: map[string]any{}, Header: http.Header{"Retry-After": {"3"}}})
	srv.Reply("GET", "/contoso/Fabrikam/_apis/ping", okJSON())
	plugin := azuredevops.New(srv.Options())
	src := &patSource{pat: testPAT}

	first, err := plugin.NewClient(map[string]any{"organization": "Contoso", "project": "Fabrikam", "team": "t"}, src)
	if err != nil {
		t.Fatal(err)
	}
	second, err := plugin.NewClient(map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "t"}, src)
	if err != nil {
		t.Fatal(err)
	}
	req := func() azuredevops.DoRequest {
		return azuredevops.NewReadRequest("vso.work_write", "leer", 0, "Fabrikam", "_apis", "ping")
	}
	if err := first.Do(context.Background(), req(), nil); err != nil {
		t.Fatal(err)
	}
	if err := second.Do(context.Background(), req(), nil); err != nil {
		t.Fatal(err)
	}

	if got := srv.Clock().Sleeps(); len(got) != 1 || got[0] != 3*time.Second {
		t.Fatalf("el ritmo de Retry-After se comparte entre clientes de la misma organización: %v", got)
	}

	if _, err := plugin.NewClient(map[string]any{"organization": "o"}, src); !errors.Is(err, azuredevops.ErrConfig) {
		t.Fatalf("una configuración incompleta es ErrConfig: %v", err)
	}
	if _, err := plugin.NewClient(map[string]any{"organization": "..", "project": "p", "team": "t"}, src); !errors.Is(err, azuredevops.ErrConfig) {
		t.Fatalf("un segmento peligroso es ErrConfig: %v", err)
	}
}

// REQ-45.16: el endpoint efectivo de las pruebas es el servidor local, nunca dev.azure.com.
func TestTestOptionsPointAtTheLocalServer(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, _ := newTestClient(t, srv, nil)

	u, err := url.Parse(client.Endpoint())
	if err != nil {
		t.Fatal(err)
	}
	if ip := net.ParseIP(u.Hostname()); ip == nil || !ip.IsLoopback() {
		t.Fatalf("el endpoint efectivo de las pruebas debe ser de bucle local: %s", client.Endpoint())
	}
	if client.Endpoint() == azuredevops.DefaultEndpoint || strings.Contains(client.Endpoint(), "dev.azure.com") {
		t.Fatalf("las pruebas no pueden apuntar a Azure DevOps real: %s", client.Endpoint())
	}
}
