package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

// Estas pruebas usan el servidor doble de Azure DevOps (adoEnv); nunca el servicio real.
// No se mezclan con integrationsWorkspace (Fake): el catálogo de pruebas es global.

func TestIntegrationsGatewayConnectStoresTheIdentityAndReportsReplacement(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	gw := NewIntegrationsGateway()

	first, err := gw.Connect(context.Background(), e.root, "azure-devops", "")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if first.Identity.ID != "guid-1" || first.Identity.DisplayName != "Ana Pérez" || first.ResolvedAt == "" || first.Replaced != nil {
		t.Errorf("primera conexión = %+v", first)
	}
	if file := readIdentityFile(t, e.identityPath()); file.Plugins["azure-devops"]["contoso"].ID != "guid-1" {
		t.Errorf("fichero de identidad = %+v", file)
	}

	second, err := gw.Connect(context.Background(), e.root, "azure-devops", "")
	if err != nil {
		t.Fatalf("segunda conexión: %v", err)
	}
	if second.Replaced == nil || second.Replaced.ID != "guid-1" {
		t.Errorf("la segunda conexión debe informar de la identidad sustituida: %+v", second)
	}
}

func TestIntegrationsGatewayConnectWithoutUsableIdentityReturnsTheSelectionAndSavesNothing(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 401})

	_, err := NewIntegrationsGateway().Connect(context.Background(), e.root, "azure-devops", "")

	var selection *integrations.IdentitySelectionError
	if !errors.As(err, &selection) {
		t.Fatalf("debe devolver *IdentitySelectionError, no %T: %v", err, err)
	}
	if len(selection.Candidates) != 3 || selection.Candidates[0].ID != "abc-123" {
		t.Errorf("candidatos = %+v", selection.Candidates)
	}
	if _, statErr := os.Stat(e.identityPath()); statErr == nil {
		t.Error("no se guarda ninguna identidad")
	}
	if strings.Contains(err.Error(), adoPAT) {
		t.Errorf("el error no puede contener el PAT: %v", err)
	}
}

func TestIntegrationsGatewayConnectRejectsUnknownUnconfiguredAndDisabledPlugins(t *testing.T) {
	tests := []struct {
		name, plugin, block, wants string
	}{
		{"desconocido", "jira", adoBlock, "plugin desconocido"},
		{"sin bloque", "azure-devops", "", "no está configurado"},
		{"desactivado", "azure-devops", "integrations:\n  azure-devops:\n    enabled: false\n", "desactivado"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newADOEnv(t, tt.block)
			e.serveProcess(adoConnectionOK())

			_, err := NewIntegrationsGateway().Connect(context.Background(), e.root, tt.plugin, "")

			if err == nil || !strings.Contains(err.Error(), tt.wants) {
				t.Fatalf("error = %v, debe contener %q", err, tt.wants)
			}
			if len(e.srv.Requests()) != 0 {
				t.Errorf("sin plugin activo no hay peticiones (%d)", len(e.srv.Requests()))
			}
		})
	}
}

func TestIntegrationsGatewayConnectFailureNeverLeaksThePAT(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 500})

	_, err := NewIntegrationsGateway().Connect(context.Background(), e.root, "azure-devops", "")

	if err == nil {
		t.Fatal("un fallo del servidor debe devolver error")
	}
	if strings.Contains(err.Error(), adoPAT) {
		t.Errorf("el error no puede contener el PAT: %v", err)
	}
}

func TestIntegrationsGatewayConnectHonoursACancelledContext(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewIntegrationsGateway().Connect(ctx, e.root, "azure-devops", "")

	if err == nil {
		t.Fatal("con el contexto cancelado la conexión debe fallar")
	}
	if _, statErr := os.Stat(e.identityPath()); statErr == nil {
		t.Error("una conexión cancelada no guarda identidad")
	}
}

func TestIntegrationsGatewayDiscoverReturnsTheProcessWithoutWritingFiles(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	before := snapshotFiles(t, e.stateDir)

	got, err := NewIntegrationsGateway().Discover(context.Background(), e.root, "azure-devops")

	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if got.Plugin != "azure-devops" || len(got.ItemTypes) != 2 || len(got.Iterations) != 2 || len(got.Members) != 3 {
		t.Errorf("descubrimiento = %+v", got)
	}
	assertSameSnapshot(t, "discover", before, snapshotFiles(t, e.stateDir))
}

func TestIntegrationsGatewayDiscoverRejectsAPluginThatIsNotConfigured(t *testing.T) {
	e := newADOEnv(t, "")

	_, err := NewIntegrationsGateway().Discover(context.Background(), e.root, "azure-devops")

	if err == nil || !strings.Contains(err.Error(), "no está configurado") {
		t.Fatalf("error = %v", err)
	}
}

func TestIntegrationsGatewayWorkItemApplicability(t *testing.T) {
	t.Run("aplica con el plugin activo", func(t *testing.T) {
		e := newStartEnv(t)

		got := NewIntegrationsGateway().WorkItemApplicability(e.root)

		if !got.Applicable || got.Plugin != "azure-devops" || got.RefAlias == "" || got.Reason != "" {
			t.Errorf("aplicabilidad = %+v", got)
		}
	})
	for name, prepare := range map[string]func(t *testing.T, e *adoEnv){
		"sin integraciones": func(t *testing.T, e *adoEnv) {
			writeCLIFixture(t, filepath.Join(e.root, "axiom.yaml"), integrationsBase)
		},
		"plugin desactivado": func(t *testing.T, e *adoEnv) {
			writeCLIFixture(t, filepath.Join(e.root, "axiom.yaml"), integrationsBase+"integrations:\n  azure-devops:\n    enabled: false\n")
		},
		"sin axiom.yaml": func(t *testing.T, e *adoEnv) {
			if err := os.Remove(filepath.Join(e.root, "axiom.yaml")); err != nil {
				t.Fatal(err)
			}
		},
		"yaml inválido": func(t *testing.T, e *adoEnv) {
			writeCLIFixture(t, filepath.Join(e.root, "axiom.yaml"), "workspace: [esto no es un mapa\n")
		},
		"plugin no compilado": func(t *testing.T, e *adoEnv) {
			integrations.SetDefaultRegistryForTest(t)
		},
	} {
		t.Run("no aplica: "+name, func(t *testing.T) {
			e := newStartEnv(t)
			prepare(t, e)

			got := NewIntegrationsGateway().WorkItemApplicability(e.root)

			if got.Applicable || got.Plugin != "" || strings.TrimSpace(got.Reason) == "" {
				t.Errorf("no debe aplicar y debe explicar por qué: %+v", got)
			}
		})
	}
}

// wantWorkItemRefusal exige un *WorkItemRefusal con el código dado, un mensaje y una continuación
// y lo devuelve.
func wantWorkItemRefusal(t *testing.T, err error, code string) *WorkItemRefusal {
	t.Helper()
	var refusal *WorkItemRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, se esperaba un *WorkItemRefusal %s", err, code)
	}
	if refusal.Code != code || strings.TrimSpace(refusal.Message) == "" || refusal.Error() != refusal.Message {
		t.Fatalf("rechazo = %+v, se esperaba el código %s con mensaje", refusal, code)
	}
	return refusal
}

func TestIntegrationsGatewayWorkItemStartStagesAndResumes(t *testing.T) {
	e := newStartEnv(t)
	e.seedStory(2525, "Alta de usuarios", "New")
	gw := NewIntegrationsGateway()

	first, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	if err != nil {
		t.Fatalf("fase 1: %v", err)
	}
	if first.Status != "staged" || first.Change != "US-2525-alta-de-usuarios" || first.ExternalRef != "ado:2525" {
		t.Errorf("primer arranque = %+v", first)
	}
	// Volver a dar el mismo work item reanuda: la fase 1 es idempotente.
	again, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	if err != nil || again.Status != "already_staged" || again.Change != first.Change {
		t.Fatalf("reanudación = %+v, %v", again, err)
	}
	staged, err := gw.WorkItemStaged(e.root, first.Change)
	if err != nil || staged.Change != first.Change || staged.ExternalRef != "ado:2525" {
		t.Fatalf("staging = %+v, %v", staged, err)
	}
}

func TestIntegrationsGatewayWorkItemStartRefusalsAreTypedAndLeaveNoTrace(t *testing.T) {
	e := newStartEnv(t)
	e.seedStory(2525, "Alta de usuarios", "Hecho")
	gw := NewIntegrationsGateway()
	before := snapshotFiles(t, e.root)

	_, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	closed := wantWorkItemRefusal(t, err, "work_item_closed")
	if strings.TrimSpace(closed.Continuation) == "" {
		t.Errorf("el rechazo debe llevar continuación: %+v", closed)
	}
	_, err = gw.StartWorkItem(context.Background(), e.root, "no-es-un-work-item")
	wantWorkItemRefusal(t, err, "work_item_reference_invalid")
	_, err = gw.StartWorkItem(context.Background(), e.root, "9999")
	wantWorkItemRefusal(t, err, "work_item_not_found")

	for path := range snapshotFiles(t, e.root) {
		if _, existed := before[path]; !existed && !strings.Contains(filepath.ToSlash(path), "change-start/v1/.locks/") {
			t.Errorf("un rechazo de la fase 1 no deja rastro: %s", path)
		}
	}
}

func TestIntegrationsGatewayWorkItemWithoutApplicablePluginIsTyped(t *testing.T) {
	e := newStartEnv(t)
	writeCLIFixture(t, filepath.Join(e.root, "axiom.yaml"), integrationsBase)
	gw := NewIntegrationsGateway()

	_, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	wantWorkItemRefusal(t, err, "work_items_not_applicable")
	_, err = gw.AdoptWorkItem(context.Background(), e.root, "x")
	wantWorkItemRefusal(t, err, "work_items_not_applicable")
	_, err = gw.DiscardWorkItem(e.root, "x")
	wantWorkItemRefusal(t, err, "work_items_not_applicable")
}

func TestIntegrationsGatewayWorkItemStagedAndDiscard(t *testing.T) {
	e := newStartEnv(t)
	e.seedStory(2525, "Alta de usuarios", "New")
	gw := NewIntegrationsGateway()

	_, err := gw.WorkItemStaged(e.root, "US-2525-alta-de-usuarios")
	wantWorkItemRefusal(t, err, "staging_not_found")
	for _, bad := range []string{"", "../x", "a/b", "archive", filepath.Join(e.root, "x")} {
		_, err = gw.WorkItemStaged(e.root, bad)
		wantWorkItemRefusal(t, err, "usage_invalid")
		_, err = gw.DiscardWorkItem(e.root, bad)
		wantWorkItemRefusal(t, err, "usage_invalid")
	}

	start, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	if err != nil {
		t.Fatal(err)
	}
	out, err := gw.DiscardWorkItem(e.root, start.Change)
	if err != nil || out.Status != "discarded" || out.Change != start.Change {
		t.Fatalf("descarte = %+v, %v", out, err)
	}
	if _, err := gw.WorkItemStaged(e.root, start.Change); err == nil {
		t.Error("tras descartar no debe quedar staging")
	}
}

// El motivo de change_not_prepared llega en Fields["reason"] y el mensaje propio del motivo (hueco 8).
func TestIntegrationsGatewayWorkItemAdoptRefusalKeepsItsReason(t *testing.T) {
	e := newStartEnv(t)
	e.seedStory(2525, "Alta de usuarios", "New")
	gw := NewIntegrationsGateway()
	start, err := gw.StartWorkItem(context.Background(), e.root, "2525")
	if err != nil {
		t.Fatal(err)
	}

	_, err = gw.AdoptWorkItem(context.Background(), e.root, start.Change)

	refusal := wantWorkItemRefusal(t, err, "change_not_prepared")
	if refusal.Fields["reason"] != "no_record" {
		t.Errorf("reason = %q, se esperaba no_record: %+v", refusal.Fields["reason"], refusal)
	}
	if !strings.Contains(refusal.Continuation, "axiom branch ensure") {
		t.Errorf("la continuación debe llevar el ensure del cambio: %q", refusal.Continuation)
	}
	if _, ok := refusal.Fields["detail"]; ok {
		t.Errorf("Fields no debe exponer detail: %+v", refusal.Fields)
	}
}

func TestIntegrationsGatewayWorkItemAdoptWithoutStagingIsTyped(t *testing.T) {
	e := newStartEnv(t)
	gw := NewIntegrationsGateway()

	_, err := gw.AdoptWorkItem(context.Background(), e.root, "US-1-sin-staging")

	wantWorkItemRefusal(t, err, "change_not_prepared")
}
