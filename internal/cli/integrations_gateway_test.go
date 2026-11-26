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
