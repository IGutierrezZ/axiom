package integrations

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// stubPlugin es un plugin mínimo para probar el registro sin depender de integrationstest.
type stubPlugin struct{ m manifest.Manifest }

func (s stubPlugin) Manifest() manifest.Manifest { return s.m }
func (s stubPlugin) Deliver(context.Context, Delivery) (Outcome, error) {
	return Outcome{}, nil
}

func stubManifest(id string) manifest.Manifest {
	return manifest.Manifest{
		Schema:       manifest.SchemaV1,
		ID:           id,
		Title:        "Plugin " + id,
		Capabilities: []manifest.Capability{manifest.CapItemsRead},
		Events:       []manifest.EventType{manifest.EventArchive},
		Config:       []manifest.ConfigField{{Key: "project", Type: manifest.FieldString, Required: true}},
		Secrets:      []manifest.SecretSpec{{Name: "token", Required: true}},
		Identity:     manifest.IdentitySpec{Source: manifest.IdentityNone},
	}
}

func TestNewRegistryAcceptsCatalogedPlugins(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"), stubManifest("beta"))
	reg, err := NewRegistry(stubPlugin{stubManifest("alpha")}, stubPlugin{stubManifest("beta")})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if got := idsOf(reg); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("IDs() = %v", got)
	}
	if p, ok := reg.Get("beta"); !ok || p.Manifest().ID != "beta" {
		t.Fatalf("Get(beta) = %v, %v", p, ok)
	}
	if _, ok := reg.Get("gamma"); ok {
		t.Fatal("Get(gamma) debe fallar")
	}
}

func TestNewRegistryRejectsDuplicates(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	reg, err := NewRegistry(stubPlugin{stubManifest("alpha")}, stubPlugin{stubManifest("alpha")})
	if !errors.Is(err, ErrDuplicatePlugin) {
		t.Fatalf("esperaba ErrDuplicatePlugin, obtuvo %v", err)
	}
	if reg != nil {
		t.Fatal("sin registro cuando hay duplicados")
	}
}

func TestNewRegistryRejectsInvalidManifest(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	bad := stubManifest("alpha")
	bad.Capabilities = nil
	reg, err := NewRegistry(stubPlugin{bad})
	if err == nil || reg != nil {
		t.Fatalf("un manifiesto inválido se rechaza: %v", err)
	}
}

func TestNewRegistryRejectsPluginMissingFromCatalog(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	reg, err := NewRegistry(stubPlugin{stubManifest("fuera")})
	if err == nil || reg != nil || !strings.Contains(err.Error(), "fuera") || !strings.Contains(err.Error(), "catálogo") {
		t.Fatalf("un id ausente del catálogo se rechaza nombrándolo: %v", err)
	}
}

func TestNewRegistryRejectsManifestDifferentFromCatalog(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	different := stubManifest("alpha")
	different.Title = "Otro título"
	reg, err := NewRegistry(stubPlugin{different})
	if err == nil || reg != nil || !strings.Contains(err.Error(), "alpha") {
		t.Fatalf("un manifiesto distinto del catálogo se rechaza: %v", err)
	}
}

func TestDefaultRegistryIsEmptyAndMatchesCatalog(t *testing.T) {
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	if len(idsOf(reg)) != 0 {
		t.Fatalf("la lista cerrada de esta versión está vacía, obtuvo %v", idsOf(reg))
	}
	var catalogIDs []string
	for _, m := range manifest.Catalog() {
		catalogIDs = append(catalogIDs, m.ID)
	}
	sort.Strings(catalogIDs)
	got := idsOf(reg)
	sort.Strings(got)
	if len(catalogIDs) != len(got) {
		t.Fatalf("los ids de DefaultRegistry %v deben coincidir con los de Catalog %v", got, catalogIDs)
	}
}

func TestSetDefaultRegistryForTestRestores(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	inner := &cleanupRecorder{}
	SetDefaultRegistryForTest(inner, stubPlugin{stubManifest("alpha")})

	reg, err := DefaultRegistry()
	if err != nil || !reflect.DeepEqual(idsOf(reg), []string{"alpha"}) {
		t.Fatalf("DefaultRegistry instalado: %v %v", reg, err)
	}
	inner.run()
	reg, err = DefaultRegistry()
	if err != nil || len(idsOf(reg)) != 0 {
		t.Fatalf("Cleanup restaura el registro vacío: %v %v", reg, err)
	}
}

func TestSetDefaultRegistryForTestPanicsOnInvalidPlugins(t *testing.T) {
	manifest.SetCatalogForTest(t, stubManifest("alpha"))
	defer func() {
		if recover() == nil {
			t.Fatal("un registro inválido debe entrar en pánico para que la prueba falle")
		}
	}()
	SetDefaultRegistryForTest(&cleanupRecorder{}, stubPlugin{stubManifest("fuera")})
}

type cleanupRecorder struct{ fns []func() }

func (c *cleanupRecorder) Helper()           {}
func (c *cleanupRecorder) Cleanup(fn func()) { c.fns = append(c.fns, fn) }
func (c *cleanupRecorder) run() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		c.fns[i]()
	}
}

// idsOf devuelve los identificadores del registro en orden de registro (solo lo usan las pruebas).
func idsOf(r *Registry) []string { return append([]string(nil), r.ids...) }
