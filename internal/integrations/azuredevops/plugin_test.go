package azuredevops_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// failingTransport falla la prueba ante cualquier petición HTTP.
type failingTransport struct {
	t     *testing.T
	count atomic.Int32
}

func (f *failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.count.Add(1)
	f.t.Errorf("petición HTTP inesperada a %s", r.URL)
	return nil, errors.New("sin red en esta prueba")
}

func TestManifestMatchesCatalog(t *testing.T) {
	p := azuredevops.New(azuredevops.Options{})
	cataloged, ok := manifest.Lookup(manifest.AzureDevOpsID)
	if !ok {
		t.Fatal("azure-devops debe estar en el catálogo")
	}
	if !reflect.DeepEqual(p.Manifest(), cataloged) {
		t.Fatal("el manifiesto del plugin debe ser idéntico al del catálogo")
	}
	if _, err := integrations.NewRegistry(p); err != nil {
		t.Fatalf("el registro con el plugin debe construirse: %v", err)
	}
}

func TestDeliverIsPermanentAndMakesNoRequest(t *testing.T) {
	transport := &failingTransport{t: t}
	p := azuredevops.New(azuredevops.Options{
		Endpoint:   "http://127.0.0.1:1",
		HTTPClient: &http.Client{Transport: transport},
	})
	_, err := p.Deliver(context.Background(), integrations.Delivery{
		Event:  integrations.Event{Type: manifest.EventChangeStart},
		Config: map[string]any{"organization": "contoso", "project": "p", "team": "t"},
	})
	if err == nil || !integrations.IsPermanent(err) {
		t.Fatalf("Deliver debe devolver un error permanente, obtuvo %v", err)
	}
	if !errors.Is(err, azuredevops.ErrLifecycleUnsupported) {
		t.Fatalf("el error debe envolver ErrLifecycleUnsupported: %v", err)
	}
	if n := transport.count.Load(); n != 0 {
		t.Fatalf("Deliver no debe hacer peticiones, hizo %d", n)
	}
}

func TestTargetFromConfig(t *testing.T) {
	good := map[string]any{"organization": "Contoso", "project": "Fabrikam", "team": "Fabrikam Team"}
	got, err := azuredevops.TargetFromConfig(good)
	if err != nil || got != (azuredevops.Target{Organization: "Contoso", Project: "Fabrikam", Team: "Fabrikam Team"}) {
		t.Fatalf("TargetFromConfig = %+v, %v", got, err)
	}
	cases := map[string]map[string]any{
		"sin organization": {"project": "p", "team": "t"},
		"sin project":      {"organization": "o", "team": "t"},
		"sin team":         {"organization": "o", "project": "p"},
		"vacío":            {"organization": "", "project": "p", "team": "t"},
		"solo espacios":    {"organization": "o", "project": "  ", "team": "t"},
		"no es cadena":     {"organization": "o", "project": 3, "team": "t"},
		"nil":              nil,
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := azuredevops.TargetFromConfig(cfg); !errors.Is(err, azuredevops.ErrConfig) {
				t.Fatalf("esperaba ErrConfig, obtuvo %v", err)
			}
		})
	}
}
