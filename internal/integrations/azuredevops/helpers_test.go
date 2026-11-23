package azuredevops_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// testPAT es un valor ficticio y reconocible: ninguna prueba usa credenciales reales.
const testPAT = "ficticio-pat-0123456789abcdef"

var testTarget = azuredevops.Target{Organization: "contoso", Project: "Fabrikam", Team: "Fabrikam Team"}

// patSource es una SecretSource sustituible que cuenta las consultas.
type patSource struct {
	pat   string
	err   error
	calls atomic.Int32
}

func (p *patSource) Secret(_ context.Context, name string) (secrets.Secret, error) {
	p.calls.Add(1)
	if name != "pat" {
		return secrets.Secret{}, fmt.Errorf("secreto inesperado %q", name)
	}
	if p.err != nil {
		return secrets.Secret{}, p.err
	}
	return secrets.NewSecret(p.pat), nil
}

// basicFor devuelve el valor exacto de la cabecera Authorization para el PAT.
func basicFor(pat string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+pat))
}

// newTestClient construye un cliente contra el servidor local con el PAT ficticio. mutate ajusta las
// opciones (timeout, intentos...) antes de construirlo.
func newTestClient(t *testing.T, srv *adotest.Server, mutate func(*azuredevops.Options)) (*azuredevops.Client, *patSource) {
	t.Helper()
	opts := srv.Options()
	if mutate != nil {
		mutate(&opts)
	}
	src := &patSource{pat: testPAT}
	client, err := azuredevops.NewClient(opts, testTarget, src)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, src
}

// pingPath es la ruta decodificada de la petición de lectura genérica de las pruebas del cliente.
const pingPath = "/contoso/Fabrikam/_apis/ping"

// ping hace una lectura genérica GET {org}/Fabrikam/_apis/ping.
func ping(ctx context.Context, c *azuredevops.Client, out any) error {
	return c.Do(ctx, azuredevops.NewReadRequest("vso.work_write", "leer el ping", 0, "Fabrikam", "_apis", "ping"), out)
}
