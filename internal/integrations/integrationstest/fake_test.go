package integrationstest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

func event(t manifest.EventType, key string) integrations.Event {
	return integrations.Event{Type: t, Change: "x", Actor: "cli", IdempotencyKey: key}
}

func TestFakeManifestIsValid(t *testing.T) {
	m := FakeManifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("el manifiesto del doble debe ser válido: %v", err)
	}
	if m.ID != "fake" || m.Identity.Source != manifest.IdentityGitActor || len(m.Secrets) != 1 || m.Secrets[0].Name != "token" {
		t.Fatalf("manifiesto del doble = %#v", m)
	}
}

func TestFakeRecordsEventsWithKey(t *testing.T) {
	f := NewFake()
	_, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventGateSpecApproved, "k1")})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	got := f.Delivered()
	if len(got) != 1 || got[0].IdempotencyKey != "k1" || got[0].Type != manifest.EventGateSpecApproved {
		t.Fatalf("Delivered = %#v", got)
	}
	if keys := f.DeliveredKeys(); len(keys) != 1 || keys[0] != "k1" {
		t.Fatalf("DeliveredKeys = %v", keys)
	}
}

func TestFakeIsIdempotentByKey(t *testing.T) {
	link := integrations.Link{Plugin: "fake", Kind: "work_item", ID: "1"}
	f := NewFake().ReturnLinks(link)
	for i := 0; i < 3; i++ {
		out, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "same")})
		if err != nil || len(out.Links) != 1 {
			t.Fatalf("entrega %d: %v %#v", i, err, out)
		}
	}
	if n := len(f.Delivered()); n != 1 {
		t.Fatalf("la misma clave solo se registra una vez, hay %d", n)
	}
	if f.Calls() != 3 {
		t.Fatalf("Calls() cuenta todas las llamadas, obtuvo %d", f.Calls())
	}
	_, _ = f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "otra")})
	if n := len(f.Delivered()); n != 2 {
		t.Fatalf("otra clave sí se registra, hay %d", n)
	}
}

func TestFakeProgrammableFailures(t *testing.T) {
	boom := errors.New("boom")

	t.Run("siempre", func(t *testing.T) {
		f := NewFake().FailAlways(boom)
		if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k")}); !errors.Is(err, boom) {
			t.Fatalf("FailAlways: %v", err)
		}
		if len(f.Delivered()) != 0 {
			t.Fatal("una entrega fallida no se registra como entregada")
		}
		f.Recover()
		if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k")}); err != nil {
			t.Fatalf("tras Recover: %v", err)
		}
	})

	t.Run("N veces", func(t *testing.T) {
		f := NewFake().FailTimes(2, boom)
		for i := 0; i < 2; i++ {
			if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k")}); !errors.Is(err, boom) {
				t.Fatalf("intento %d debía fallar: %v", i, err)
			}
		}
		if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k")}); err != nil {
			t.Fatalf("el tercer intento debe tener éxito: %v", err)
		}
	})

	t.Run("por evento", func(t *testing.T) {
		f := NewFake().FailOnEvent(manifest.EventRoleStart, boom)
		if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventRoleStart, "a")}); !errors.Is(err, boom) {
			t.Fatalf("role.start debía fallar: %v", err)
		}
		if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "b")}); err != nil {
			t.Fatalf("archive no debía fallar: %v", err)
		}
	})
}

func TestFakeHonorsContextAndCanIgnoreIt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewFake().Deliver(ctx, integrations.Delivery{Event: event(manifest.EventArchive, "k")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("debe respetar el contexto cancelado: %v", err)
	}

	release := make(chan struct{})
	f := NewFake().BlockUntil(release)
	done := make(chan struct{})
	go func() {
		_, _ = f.Deliver(ctx, integrations.Delivery{Event: event(manifest.EventArchive, "k")})
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("BlockUntil debe bloquear aunque el contexto esté cancelado")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Deliver debe volver al cerrar el canal")
	}
}

func TestFakePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("PanicWith debe entrar en pánico")
		}
	}()
	_, _ = NewFake().PanicWith("kaboom").Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k")})
}

func TestFakeReadsAndCanLeakToken(t *testing.T) {
	mem := Install(t, NewFake())
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret("valor-secreto-123"))

	src := stubSource{value: "valor-secreto-123"}
	f := NewFake().ReadToken()
	if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k"), Secrets: src}); err != nil {
		t.Fatalf("ReadToken con secreto disponible: %v", err)
	}
	if _, err := f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k2")}); err == nil {
		t.Fatal("ReadToken sin SecretSource debe fallar")
	}

	leaky := NewFake().LeakTokenInError()
	_, err := leaky.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k"), Secrets: src})
	if err == nil || !strings.Contains(err.Error(), "valor-secreto-123") {
		t.Fatalf("LeakTokenInError debe incluir el valor en el error: %v", err)
	}
}

func TestFakeRecordsReceivedDeliveriesWithCopies(t *testing.T) {
	f := NewFake()
	cfg := map[string]any{"project": "p"}
	_, _ = f.Deliver(context.Background(), integrations.Delivery{Event: event(manifest.EventArchive, "k"), Config: cfg, Links: []integrations.Link{{ID: "1"}}})
	cfg["project"] = "mutado"
	got := f.Received()
	if len(got) != 1 || got[0].Config["project"] != "p" || len(got[0].Links) != 1 {
		t.Fatalf("Received = %#v", got)
	}
}

func TestInstallInstallsAndRestoresAfterCleanup(t *testing.T) {
	inner := &cleanups{}
	fake := NewFake()
	mem := Install(inner, fake)

	if _, ok := manifest.Lookup("fake"); !ok {
		t.Fatal("Install debe instalar el manifiesto en el catálogo")
	}
	reg, err := integrations.DefaultRegistry()
	if err != nil || len(reg.IDs()) != 1 || reg.IDs()[0] != "fake" {
		t.Fatalf("Install debe instalar el registro: %v %v", reg, err)
	}
	ref := secrets.Ref{Plugin: "fake", Name: "token"}
	_ = mem.Set(context.Background(), ref, secrets.NewSecret("v"))
	if s, src, err := secrets.Default().Get(context.Background(), ref); err != nil || s.Reveal() != "v" || src != secrets.SourceKeyring {
		t.Fatalf("Install debe instalar el almacén en memoria: %v %q %q", err, s.Reveal(), src)
	}

	inner.run()
	if _, ok := manifest.Lookup("fake"); ok {
		t.Fatal("Cleanup debe restaurar el catálogo")
	}
	if reg, _ := integrations.DefaultRegistry(); len(reg.IDs()) != 0 {
		t.Fatal("Cleanup debe restaurar el registro")
	}
	if _, _, err := secrets.Default().Get(context.Background(), ref); err == nil {
		t.Fatal("Cleanup debe restaurar el almacén de producción")
	}
}

type stubSource struct{ value string }

func (s stubSource) Secret(context.Context, string) (secrets.Secret, error) {
	return secrets.NewSecret(s.value), nil
}

type cleanups struct{ fns []func() }

func (c *cleanups) Helper()           {}
func (c *cleanups) Cleanup(fn func()) { c.fns = append(c.fns, fn) }
func (c *cleanups) run() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		c.fns[i]()
	}
}
