package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// countingStore cuenta las consultas al almacén de secretos.
type countingStore struct {
	inner secrets.Store
	calls atomic.Int32
}

func (c *countingStore) Get(ctx context.Context, ref secrets.Ref) (secrets.Secret, secrets.Source, error) {
	c.calls.Add(1)
	return c.inner.Get(ctx, ref)
}

func newCountingStore(t *testing.T, value string) *countingStore {
	t.Helper()
	memory := secrets.NewMemory()
	if err := memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(value)); err != nil {
		t.Fatal(err)
	}
	return &countingStore{inner: memory}
}

// portPlugin ofrece ambos puertos con comportamiento programable.
type portPlugin struct {
	*testPlugin
	identity  func(ctx context.Context, req IdentityRequest) (TrackerIdentity, error)
	discovery func(ctx context.Context, req PortRequest) (Discovery, error)
}

func (p *portPlugin) ResolveIdentity(ctx context.Context, req IdentityRequest) (TrackerIdentity, error) {
	return p.identity(ctx, req)
}

func (p *portPlugin) Discover(ctx context.Context, req PortRequest) (Discovery, error) {
	return p.discovery(ctx, req)
}

func newPortPlugin() *portPlugin { return &portPlugin{testPlugin: newTestPlugin()} }

var testPortConfig = workspace.IntegrationPluginConfig{ID: "fake", Enabled: true, Values: map[string]any{"project": "p"}}

func TestPortsAreDetectedByTypeAssertion(t *testing.T) {
	var full Plugin = newPortPlugin()
	if _, ok := full.(IdentityResolver); !ok {
		t.Error("un plugin con ResolveIdentity debe detectarse como IdentityResolver")
	}
	if _, ok := full.(Discoverer); !ok {
		t.Error("un plugin con Discover debe detectarse como Discoverer")
	}
	var bare Plugin = newTestPlugin()
	if _, ok := bare.(IdentityResolver); ok {
		t.Error("un plugin sin puerto no debe detectarse como IdentityResolver")
	}
	if _, ok := bare.(Discoverer); ok {
		t.Error("un plugin sin puerto no debe detectarse como Discoverer")
	}
}

func TestPortsUnsupportedBeforeAnySecretIsResolved(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	plugin := newTestPlugin()

	_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, store, "")
	var unsupported *PortUnsupportedError
	if !errors.Is(err, ErrPortUnsupported) || !errors.As(err, &unsupported) || unsupported.Plugin != "fake" || unsupported.Port != PortIdentity {
		t.Fatalf("ResolveIdentity sin puerto = %v", err)
	}
	if !strings.Contains(err.Error(), "fake") || !strings.Contains(err.Error(), "identidad") {
		t.Errorf("el error debe nombrar el plugin y el puerto: %q", err)
	}

	_, err = Discover(context.Background(), plugin, testPortConfig, store)
	if !errors.Is(err, ErrPortUnsupported) || !errors.As(err, &unsupported) || unsupported.Port != PortDiscovery {
		t.Fatalf("Discover sin puerto = %v", err)
	}
	if !strings.Contains(err.Error(), "descubrimiento") {
		t.Errorf("el error debe nombrar el puerto de descubrimiento: %q", err)
	}
	if store.calls.Load() != 0 {
		t.Errorf("sin puerto no se resuelve ningún secreto (consultas: %d)", store.calls.Load())
	}
}

func TestPortErrorIsSanitizedAndKeepsTheCause(t *testing.T) {
	const secret = "valor-secreto-0123"
	store := newCountingStore(t, secret)
	cause := errors.New("el tracker rechazó la credencial")
	plugin := newPortPlugin()
	plugin.identity = func(ctx context.Context, req IdentityRequest) (TrackerIdentity, error) {
		got, err := req.Secrets.Secret(ctx, "token")
		if err != nil {
			return TrackerIdentity{}, err
		}
		return TrackerIdentity{}, fmt.Errorf("%w: %s", cause, got.Reveal())
	}
	plugin.discovery = func(ctx context.Context, req PortRequest) (Discovery, error) {
		got, _ := req.Secrets.Secret(ctx, "token")
		return Discovery{}, fmt.Errorf("%w: %s", cause, got.Reveal())
	}

	_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, store, "")
	assertSanitizedPortError(t, err, cause, secret, PortIdentity)

	_, err = Discover(context.Background(), plugin, testPortConfig, store)
	assertSanitizedPortError(t, err, cause, secret, PortDiscovery)
}

func assertSanitizedPortError(t *testing.T, err, cause error, secret, port string) {
	t.Helper()
	var portErr *PortError
	if !errors.As(err, &portErr) || portErr.Port != port || portErr.Plugin != "fake" {
		t.Fatalf("se esperaba *PortError de %s, obtuvo %v", port, err)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(portErr.Message, secret) {
		t.Errorf("el error contiene el secreto: %q", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Errorf("el error debe llevar el marcador de redacción: %q", err)
	}
	if !errors.Is(err, cause) {
		t.Error("la causa debe conservarse para errors.Is")
	}
}

func TestPortSelectionErrorIsPreservedAndItsReasonSanitized(t *testing.T) {
	const secret = "valor-secreto-0123"
	store := newCountingStore(t, secret)
	plugin := newPortPlugin()
	plugin.identity = func(ctx context.Context, req IdentityRequest) (TrackerIdentity, error) {
		got, _ := req.Secrets.Secret(ctx, "token")
		return TrackerIdentity{}, &IdentitySelectionError{
			Plugin: "fake", Account: "contoso", Reason: "falló con " + got.Reveal(),
			Candidates: []Member{{ID: "abc-123", DisplayName: "Ana"}},
		}
	}

	_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, store, "")

	var selection *IdentitySelectionError
	if !errors.Is(err, ErrIdentitySelectionRequired) || !errors.As(err, &selection) {
		t.Fatalf("se esperaba un IdentitySelectionError, obtuvo %v", err)
	}
	if len(selection.Candidates) != 1 || selection.Candidates[0].ID != "abc-123" {
		t.Errorf("los candidatos deben conservarse: %+v", selection.Candidates)
	}
	if strings.Contains(selection.Reason, secret) || strings.Contains(err.Error(), secret) {
		t.Errorf("el motivo o el texto contienen el secreto: %q / %q", selection.Reason, err)
	}
}

func TestPortPanicIsRecoveredAsAnError(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	plugin := newPortPlugin()
	plugin.identity = func(context.Context, IdentityRequest) (TrackerIdentity, error) { panic("roto") }
	plugin.discovery = func(context.Context, PortRequest) (Discovery, error) { panic("roto") }

	_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, store, "")
	var portErr *PortError
	if !errors.As(err, &portErr) || !strings.Contains(err.Error(), "pánico") {
		t.Fatalf("un pánico se recupera como error: %v", err)
	}
	if _, err := Discover(context.Background(), plugin, testPortConfig, store); !errors.As(err, &portErr) || !strings.Contains(err.Error(), "pánico") {
		t.Fatalf("un pánico se recupera como error: %v", err)
	}
}

func TestDiscoverSetsSchemaAndPluginAndNeverEmitsNullLists(t *testing.T) {
	plugin := newPortPlugin()
	plugin.discovery = func(context.Context, PortRequest) (Discovery, error) {
		// Un plugin que devuelve todo nulo y un esquema y un plugin propios, que se sobrescriben.
		return Discovery{Schema: "otro", Plugin: "otro", ItemTypes: []ItemType{{Name: "Bug", ReferenceName: "X.Bug"}}}, nil
	}

	got, err := Discover(context.Background(), plugin, testPortConfig, newCountingStore(t, "valor-de-prueba"))

	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != DiscoverySchemaV1 || got.Plugin != "fake" {
		t.Errorf("esquema y plugin los fija Discover: %q %q", got.Schema, got.Plugin)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "null") {
		t.Errorf("el JSON no debe llevar null: %s", payload)
	}
}

func TestResolveIdentityRejectsAnIncompleteIdentity(t *testing.T) {
	for name, identity := range map[string]TrackerIdentity{
		"sin id":     {Account: "contoso", DisplayName: "Ana", Source: "x"},
		"sin cuenta": {ID: "abc", DisplayName: "Ana", Source: "x"},
	} {
		t.Run(name, func(t *testing.T) {
			plugin := newPortPlugin()
			plugin.identity = func(context.Context, IdentityRequest) (TrackerIdentity, error) { return identity, nil }

			_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, newCountingStore(t, "valor-de-prueba"), "")

			if !errors.Is(err, ErrInvalidPortResult) {
				t.Fatalf("se esperaba ErrInvalidPortResult, obtuvo %v", err)
			}
		})
	}
}

func TestResolveIdentityPassesTheUserAndACopyOfTheConfig(t *testing.T) {
	plugin := newPortPlugin()
	var got IdentityRequest
	plugin.identity = func(_ context.Context, req IdentityRequest) (TrackerIdentity, error) {
		got = req
		req.Config["project"] = "mutado"
		return TrackerIdentity{Account: "contoso", ID: "abc", DisplayName: "Ana", Source: "x"}, nil
	}
	cfg := workspace.IntegrationPluginConfig{ID: "fake", Enabled: true, Values: map[string]any{"project": "p"}}

	identity, err := ResolveIdentity(context.Background(), plugin, cfg, newCountingStore(t, "valor-de-prueba"), "abc")

	if err != nil || identity.ID != "abc" {
		t.Fatalf("identidad = %+v, %v", identity, err)
	}
	if got.User != "abc" {
		t.Errorf("User = %q", got.User)
	}
	if cfg.Values["project"] != "p" {
		t.Error("el plugin recibe una copia de la configuración")
	}
}

func TestPortSecretsAreScopedToTheManifest(t *testing.T) {
	plugin := newPortPlugin()
	plugin.identity = func(ctx context.Context, req IdentityRequest) (TrackerIdentity, error) {
		_, err := req.Secrets.Secret(ctx, "otro")
		return TrackerIdentity{}, err
	}

	_, err := ResolveIdentity(context.Background(), plugin, testPortConfig, newCountingStore(t, "valor-de-prueba"), "")

	if err == nil || !strings.Contains(err.Error(), "no declara el secreto") {
		t.Fatalf("un secreto no declarado es un error: %v", err)
	}
}
