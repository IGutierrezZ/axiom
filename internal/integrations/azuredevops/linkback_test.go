package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const itemPathFor42 = "/contoso/Fabrikam/_apis/wit/workitems/42"

func linkBackRequest(req integrations.PortRequest, tag string, allow func([]string) error) integrations.BackLinkRequest {
	return integrations.BackLinkRequest{PortRequest: req, ID: "42", Tag: tag, Allow: allow}
}

func tagsOf(t *testing.T, srv *adotest.Server) any {
	t.Helper()
	item, ok := srv.WorkItem(42)
	if !ok {
		t.Fatal("el WI 42 no existe")
	}
	return item.Fields["System.Tags"]
}

func TestLinkBackAddsKeepingOthers(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Tags": "a; b"})
	p, req, _ := newPort(srv)

	got, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", func([]string) error { return nil }))
	if err != nil || got.Outcome != integrations.BackLinkAdded {
		t.Fatalf("LinkBack = %+v, %v", got, err)
	}
	if tags := tagsOf(t, srv); tags != "a; b; axiom:cambio" {
		t.Fatalf("etiquetas = %v", tags)
	}
	if srv.AppliedPatches(42) != 1 {
		t.Fatalf("parches = %d", srv.AppliedPatches(42))
	}
}

func TestLinkBackAddsToAnItemWithoutTags(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.Title": "t"})
	p, req, _ := newPort(srv)

	if got, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", nil)); err != nil || got.Outcome != integrations.BackLinkAdded {
		t.Fatalf("LinkBack = %+v, %v", got, err)
	}
	if tags := tagsOf(t, srv); tags != "axiom:cambio" {
		t.Fatalf("etiquetas = %v", tags)
	}
}

func TestLinkBackPresentNoWrite(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Tags": "a; AXIOM:Cambio ;b"})
	p, req, _ := newPort(srv)

	got, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", func([]string) error { return nil }))
	if err != nil || got.Outcome != integrations.BackLinkPresent {
		t.Fatalf("LinkBack = %+v, %v", got, err)
	}
	if srv.AppliedPatches(42) != 0 || srv.Count("PATCH", itemPathFor42) != 0 {
		t.Fatal("si la etiqueta ya está (sin distinguir mayúsculas) no se escribe")
	}
}

func TestLinkBackRevConflictRetries(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Tags": "a"})
	// Otra persona añade una etiqueta justo después de la primera lectura: el test /rev falla.
	srv.EditAfterNextRead(42, map[string]any{"System.Tags": "a; ajena"})
	p, req, _ := newPort(srv)

	got, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", func([]string) error { return nil }))
	if err != nil || got.Outcome != integrations.BackLinkAdded {
		t.Fatalf("LinkBack = %+v, %v", got, err)
	}
	if tags := tagsOf(t, srv); tags != "a; ajena; axiom:cambio" {
		t.Fatalf("no se pierden las etiquetas ajenas: %v", tags)
	}
	if srv.Count("GET", itemPathFor42) != 2 || srv.Count("PATCH", itemPathFor42) != 2 || srv.AppliedPatches(42) != 1 {
		t.Fatalf("lecturas=%d, parches enviados=%d, aplicados=%d", srv.Count("GET", itemPathFor42), srv.Count("PATCH", itemPathFor42), srv.AppliedPatches(42))
	}
}

func TestLinkBackPersistentConflict(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Tags": "a"})
	for _, tags := range []string{"a; x1", "a; x1; x2", "a; x1; x2; x3", "a; x1; x2; x3; x4"} {
		srv.EditAfterNextRead(42, map[string]any{"System.Tags": tags})
	}
	p, req, _ := newPort(srv)

	_, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", func([]string) error { return nil }))
	if !errors.Is(err, integrations.ErrItemConflict) {
		t.Fatalf("tras 3 intentos es ErrItemConflict: %v", err)
	}
	if srv.Count("GET", itemPathFor42) != 3 || srv.AppliedPatches(42) != 0 {
		t.Fatalf("exactamente 3 intentos y ninguna escritura aplicada: lecturas=%d, aplicados=%d", srv.Count("GET", itemPathFor42), srv.AppliedPatches(42))
	}
}

func TestLinkBackOnlyTagsField(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 4, map[string]any{"System.Tags": "a", "System.Title": "no se toca"})
	p, req, _ := newPort(srv)

	if _, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", nil)); err != nil {
		t.Fatal(err)
	}
	var patches []struct {
		Op, Path string
		Value    any
	}
	for _, r := range srv.Requests() {
		if r.Method == "PATCH" {
			if err := json.Unmarshal(r.Body, &patches); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(patches) != 2 || patches[0].Op != "test" || patches[0].Path != "/rev" || patches[0].Value != float64(4) ||
		patches[1].Op != "add" || patches[1].Path != "/fields/System.Tags" {
		t.Fatalf("el PATCH solo lleva test /rev y System.Tags: %+v", patches)
	}
	if item, _ := srv.WorkItem(42); item.Fields["System.Title"] != "no se toca" {
		t.Fatalf("otros campos intactos: %v", item.Fields)
	}
}

func TestLinkBackAllowRefuses(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Tags": "a; axiom:otro; AXIOM:Viejo"})
	p, req, _ := newPort(srv)

	var seen []string
	got, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", func(tags []string) error {
		seen = tags
		return errors.New("conflicto")
	}))
	if err != nil || got.Outcome != integrations.BackLinkConflict {
		t.Fatalf("LinkBack = %+v, %v", got, err)
	}
	if len(seen) != 3 || seen[1] != "axiom:otro" {
		t.Fatalf("Allow recibe las etiquetas actuales: %v", seen)
	}
	if len(got.Conflicting) != 2 || got.Conflicting[0] != "axiom:otro" || got.Conflicting[1] != "AXIOM:Viejo" {
		t.Fatalf("conflictivas = %v", got.Conflicting)
	}
	if srv.Count("PATCH", itemPathFor42) != 0 {
		t.Fatal("con conflicto no se escribe")
	}
}

func TestLinkBackNotFound(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	p, req, _ := newPort(srv)
	_, err := p.LinkBack(context.Background(), linkBackRequest(req, "axiom:cambio", nil))
	if !errors.Is(err, integrations.ErrItemNotFound) {
		t.Fatalf("un WI inexistente es ErrItemNotFound: %v", err)
	}
}
