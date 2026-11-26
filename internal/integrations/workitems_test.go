package integrations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// workItemPlugin ofrece el puerto de work items con comportamiento programable.
type workItemPlugin struct {
	*testPlugin
	parse    func(cfg map[string]any, raw string) (string, error)
	read     func(ctx context.Context, req ItemRequest) (ExternalItem, error)
	category func(ctx context.Context, req StateRequest) (StateCategory, error)
	download func(ctx context.Context, req AttachmentRequest) (Attachment, error)
	backlink func(ctx context.Context, req BackLinkRequest) (BackLink, error)
}

func (p *workItemPlugin) ParseItemRef(cfg map[string]any, raw string) (string, error) {
	return p.parse(cfg, raw)
}

func (p *workItemPlugin) ReadItem(ctx context.Context, req ItemRequest) (ExternalItem, error) {
	return p.read(ctx, req)
}

func (p *workItemPlugin) StateCategory(ctx context.Context, req StateRequest) (StateCategory, error) {
	return p.category(ctx, req)
}

func (p *workItemPlugin) DownloadAttachment(ctx context.Context, req AttachmentRequest) (Attachment, error) {
	return p.download(ctx, req)
}

func (p *workItemPlugin) LinkBack(ctx context.Context, req BackLinkRequest) (BackLink, error) {
	return p.backlink(ctx, req)
}

func newWorkItemPlugin() *workItemPlugin { return &workItemPlugin{testPlugin: newTestPlugin()} }

func TestWorkItemPortDispatch(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	p := newWorkItemPlugin()
	p.parse = func(cfg map[string]any, raw string) (string, error) {
		if cfg["project"] != "p" {
			t.Errorf("ParseItemRef sin la configuración: %v", cfg)
		}
		return "42", nil
	}
	p.read = func(_ context.Context, req ItemRequest) (ExternalItem, error) {
		if req.ID != "42" || req.Config["project"] != "p" || req.Secrets == nil {
			t.Errorf("ReadItem recibió %+v", req)
		}
		return ExternalItem{ID: req.ID, Type: "Bug", Title: "t"}, nil
	}
	p.category = func(_ context.Context, req StateRequest) (StateCategory, error) {
		if req.Type != "Bug" || req.State != "Active" {
			t.Errorf("StateCategory recibió %+v", req)
		}
		return StateCategory{Category: "InProgress", Known: true}, nil
	}
	p.download = func(_ context.Context, req AttachmentRequest) (Attachment, error) {
		if req.URL != "https://x/a" || req.MaxBytes != 10 {
			t.Errorf("DownloadAttachment recibió %+v", req)
		}
		return Attachment{Data: []byte("abc"), ContentType: "application/octet-stream"}, nil
	}
	p.backlink = func(_ context.Context, req BackLinkRequest) (BackLink, error) {
		if req.ID != "42" || req.Tag != "axiom:c" {
			t.Errorf("LinkBack recibió %+v", req)
		}
		if err := req.Allow([]string{"a"}); err != nil {
			t.Errorf("Allow = %v", err)
		}
		return BackLink{Outcome: BackLinkAdded}, nil
	}
	ctx := context.Background()

	id, err := ParseItemRef(p, testPortConfig, "x")
	if err != nil || id != "42" {
		t.Fatalf("ParseItemRef = %q, %v", id, err)
	}
	item, err := ReadItem(ctx, p, testPortConfig, store, "42")
	if err != nil || item.Type != "Bug" {
		t.Fatalf("ReadItem = %+v, %v", item, err)
	}
	cat, err := ReadStateCategory(ctx, p, testPortConfig, store, "Bug", "Active")
	if err != nil || cat.Category != "InProgress" || !cat.Known {
		t.Fatalf("ReadStateCategory = %+v, %v", cat, err)
	}
	att, err := DownloadAttachment(ctx, p, testPortConfig, store, "https://x/a", 10)
	if err != nil || string(att.Data) != "abc" {
		t.Fatalf("DownloadAttachment = %+v, %v", att, err)
	}
	bl, err := LinkBack(ctx, p, testPortConfig, store, "42", "axiom:c", func([]string) error { return nil })
	if err != nil || bl.Outcome != BackLinkAdded {
		t.Fatalf("LinkBack = %+v, %v", bl, err)
	}
}

func TestWorkItemPortUnsupported(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	p := newTestPlugin()
	ctx := context.Background()
	check := func(name string, err error) {
		t.Helper()
		var unsupported *PortUnsupportedError
		if !errors.Is(err, ErrPortUnsupported) || !errors.As(err, &unsupported) || unsupported.Port != PortWorkItems || unsupported.Plugin != "fake" {
			t.Errorf("%s sin puerto = %v", name, err)
		}
		if err != nil && !strings.Contains(err.Error(), "work items") {
			t.Errorf("%s: el error debe nombrar el puerto: %q", name, err)
		}
	}
	_, err := ParseItemRef(p, testPortConfig, "1")
	check("ParseItemRef", err)
	_, err = ReadItem(ctx, p, testPortConfig, store, "1")
	check("ReadItem", err)
	_, err = ReadStateCategory(ctx, p, testPortConfig, store, "Bug", "New")
	check("ReadStateCategory", err)
	_, err = DownloadAttachment(ctx, p, testPortConfig, store, "https://x", 1)
	check("DownloadAttachment", err)
	_, err = LinkBack(ctx, p, testPortConfig, store, "1", "t", nil)
	check("LinkBack", err)
	if store.calls.Load() != 0 {
		t.Errorf("sin puerto no se resuelve ningún secreto (consultas: %d)", store.calls.Load())
	}
}

func TestWorkItemPortPanicRecovered(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	p := newWorkItemPlugin()
	p.parse = func(map[string]any, string) (string, error) { panic("boom") }
	p.read = func(context.Context, ItemRequest) (ExternalItem, error) { panic("boom") }
	p.backlink = func(_ context.Context, req BackLinkRequest) (BackLink, error) {
		_ = req.Allow(nil) // un pánico del callback también lo recupera invokePort
		return BackLink{}, nil
	}
	_, err := ParseItemRef(p, testPortConfig, "1")
	var portErr *PortError
	if !errors.As(err, &portErr) || !strings.Contains(err.Error(), "pánico") {
		t.Fatalf("ParseItemRef con pánico = %v", err)
	}
	_, err = ReadItem(context.Background(), p, testPortConfig, store, "1")
	if !errors.As(err, &portErr) || !strings.Contains(err.Error(), "pánico") {
		t.Fatalf("ReadItem con pánico = %v", err)
	}
	_, err = LinkBack(context.Background(), p, testPortConfig, store, "1", "t", func([]string) error { panic("callback") })
	if !errors.As(err, &portErr) || !strings.Contains(err.Error(), "pánico") {
		t.Fatalf("LinkBack con pánico en Allow = %v", err)
	}
}

func TestWorkItemPortSanitizesSecret(t *testing.T) {
	const secret = "valor-secreto-0123"
	store := newCountingStore(t, secret)
	cause := errors.New("el tracker rechazó la credencial")
	p := newWorkItemPlugin()
	leak := func(ctx context.Context, req PortRequest) error {
		got, err := req.Secrets.Secret(ctx, "token")
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: %s", cause, got.Reveal())
	}
	p.read = func(ctx context.Context, req ItemRequest) (ExternalItem, error) {
		return ExternalItem{}, leak(ctx, req.PortRequest)
	}
	p.download = func(ctx context.Context, req AttachmentRequest) (Attachment, error) {
		return Attachment{}, leak(ctx, req.PortRequest)
	}
	_, err := ReadItem(context.Background(), p, testPortConfig, store, "1")
	assertSanitizedPortError(t, err, cause, secret, PortWorkItems)
	_, err = DownloadAttachment(context.Background(), p, testPortConfig, store, "https://x", 1)
	assertSanitizedPortError(t, err, cause, secret, PortWorkItems)
}

// Un plugin anterior, sin el puerto, sigue registrándose y entregando como siempre (RF-INT-23).
func TestLegacyPluginStillRegisters(t *testing.T) {
	var legacy Plugin = newTestPlugin()
	if _, ok := legacy.(WorkItemSource); ok {
		t.Fatal("un plugin sin los cinco métodos no debe detectarse como WorkItemSource")
	}
	manifest.SetCatalogForTest(t, testManifest())
	reg, err := NewRegistry(legacy)
	if err != nil {
		t.Fatalf("NewRegistry con el plugin heredado: %v", err)
	}
	if _, ok := reg.Get("fake"); !ok {
		t.Fatal("el plugin heredado debe resolverse en el registro")
	}
	var full Plugin = newWorkItemPlugin()
	if _, ok := full.(WorkItemSource); !ok {
		t.Fatal("un plugin con los cinco métodos debe detectarse como WorkItemSource")
	}
}
